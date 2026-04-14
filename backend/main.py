import json
import logging
from pathlib import Path

from contextlib import asynccontextmanager

import yaml as pyyaml
from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware
from fastapi.responses import JSONResponse, StreamingResponse
from fastapi.staticfiles import StaticFiles

from .bridge_client import close_client, get_entities, get_ha_version, get_registry_status
from .claude_client import CONTEXT_DOMAINS, stream_chat
from .context_builder import build_system_prompt
from .models import ChatRequest, ValidateRequest
from .session import get_or_create

log = logging.getLogger(__name__)

# HA entity domains that can appear as entity_ids in automations
_HA_DOMAINS = {
    "alarm_control_panel", "automation", "binary_sensor", "button", "calendar",
    "camera", "climate", "counter", "cover", "device_tracker", "event", "fan",
    "humidifier", "image", "input_boolean", "input_button", "input_datetime",
    "input_number", "input_select", "input_text", "lawn_mower", "light", "lock",
    "media_player", "notify", "number", "person", "remote", "scene", "script",
    "select", "sensor", "siren", "sun", "switch", "text", "timer", "todo",
    "vacuum", "valve", "weather", "zone",
}
_ENTITY_KEYS = {"entity_id", "entity_ids", "entities"}


@asynccontextmanager
async def lifespan(app: FastAPI):
    version = await get_ha_version()
    app.state.ha_version = version
    log.info("Home Assistant version detected: %s", version)
    yield
    await close_client()


app = FastAPI(title="HA Automation Assistant", lifespan=lifespan)

# CORS — allow Vite dev server during development
app.add_middleware(
    CORSMiddleware,
    allow_origins=["http://localhost:5173"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)


@app.get("/health")
async def health():
    return JSONResponse({"status": "ok", "ha_version": app.state.ha_version})


@app.get("/api/status")
async def status():
    """Proxy bridge registry availability for the frontend."""
    registry_available = await get_registry_status()
    return JSONResponse({"registry_available": registry_available})


@app.post("/api/chat")
async def chat(req: ChatRequest):
    session_id, session = get_or_create(req.session_id)

    # Sync user-edited YAML back into the session so Claude sees the latest version
    if req.edited_yaml and session.get("pending_automation"):
        session["pending_automation"]["yaml"] = req.edited_yaml
        log.info("chat: synced edited YAML for automation id=%s", session["pending_automation"]["id"])

    # Add user message; if confirming without text, inject a standard trigger
    if req.message:
        session["history"].append({"role": "user", "content": req.message})
    elif req.confirmed:
        session["history"].append(
            {"role": "user", "content": "Yes, please create the automation."}
        )

    entities, registry_available = await get_entities(domains=CONTEXT_DOMAINS)
    system_prompt = build_system_prompt(
        entities,
        ha_version=app.state.ha_version,
        pending_automation=session.get("pending_automation"),
        registry_available=registry_available,
    )

    async def event_stream():
        yield f"data: {json.dumps({'session_id': session_id})}\n\n"
        async for chunk in stream_chat(session, system_prompt, confirmed=req.confirmed):
            yield chunk
        yield "data: {\"done\": true}\n\n"

    return StreamingResponse(event_stream(), media_type="text/event-stream")


@app.post("/api/validate")
async def validate_yaml(req: ValidateRequest):
    """
    Validate automation YAML: syntax, required fields, and entity existence.
    Mirrors what HA's developer tools config check does.
    """
    errors: list[str] = []
    warnings: list[str] = []
    missing_entities: list[str] = []

    # 1. YAML syntax
    try:
        config = pyyaml.safe_load(req.yaml)
    except pyyaml.YAMLError as exc:
        return JSONResponse({
            "valid": False,
            "errors": [f"YAML syntax error: {exc}"],
            "warnings": [],
            "missing_entities": [],
        })

    if not isinstance(config, dict):
        return JSONResponse({
            "valid": False,
            "errors": ["Automation must be a YAML mapping (got a list or scalar)."],
            "warnings": [],
            "missing_entities": [],
        })

    # 2. Required fields
    has_trigger = "triggers" in config or "trigger" in config
    has_action = "actions" in config or "action" in config
    if not has_trigger:
        errors.append("Missing required field: 'triggers'")
    if not has_action:
        errors.append("Missing required field: 'actions'")
    if not config.get("alias"):
        warnings.append("No 'alias' — automation will get an auto-generated name in HA")

    # 3. Deprecated singular keys
    for old, new in [("trigger", "triggers"), ("condition", "conditions"), ("action", "actions")]:
        if old in config and new not in config:
            warnings.append(f"'{old}:' is deprecated since HA 2024.10 — prefer '{new}:'")

    # 4. Entity existence check against the live bridge cache.
    # Walk the parsed YAML and only collect values under known entity-reference keys,
    # which avoids false positives from service names like "light.turn_on".
    def collect_entity_refs(node) -> list[str]:
        refs = []
        if isinstance(node, dict):
            for k, v in node.items():
                if k in _ENTITY_KEYS:
                    if isinstance(v, str):
                        refs.append(v)
                    elif isinstance(v, list):
                        refs.extend(x for x in v if isinstance(x, str))
                else:
                    refs.extend(collect_entity_refs(v))
        elif isinstance(node, list):
            for item in node:
                refs.extend(collect_entity_refs(item))
        return refs

    try:
        all_entities, _ = await get_entities(domains=CONTEXT_DOMAINS)
        known_ids = {e["entity_id"] for e in all_entities}
        candidates = collect_entity_refs(config)
        for ref in sorted(set(candidates)):
            domain = ref.split(".")[0] if "." in ref else ""
            if domain in _HA_DOMAINS and ref not in known_ids:
                missing_entities.append(ref)
        if missing_entities:
            warnings.append(
                "These entity IDs were not found in your Home Assistant: "
                + ", ".join(missing_entities)
            )
    except Exception as exc:
        warnings.append(f"Could not verify entity IDs (bridge unreachable): {exc}")

    return JSONResponse({
        "valid": len(errors) == 0,
        "errors": errors,
        "warnings": warnings,
        "missing_entities": missing_entities,
    })


# Serve built frontend — must come last so API routes take priority
_frontend_dir = Path(__file__).resolve().parent.parent / "frontend-dist"
if _frontend_dir.is_dir():
    app.mount("/", StaticFiles(directory=str(_frontend_dir), html=True), name="frontend")
