import json

from fastapi import FastAPI
from fastapi.responses import JSONResponse, StreamingResponse
from fastapi.staticfiles import StaticFiles

from .bridge_client import get_entities
from .claude_client import CONTEXT_DOMAINS, stream_chat
from .context_builder import build_system_prompt
from .models import ChatRequest
from .session import get_or_create

app = FastAPI(title="HA Automation Assistant")


@app.get("/health")
async def health():
    return JSONResponse({"status": "ok"})


@app.post("/api/chat")
async def chat(req: ChatRequest):
    session_id, session = get_or_create(req.session_id)

    # Add user message; if confirming without text, inject a standard trigger
    if req.message:
        session["history"].append({"role": "user", "content": req.message})
    elif req.confirmed:
        session["history"].append(
            {"role": "user", "content": "Yes, please create the automation."}
        )

    entities = await get_entities(domains=CONTEXT_DOMAINS)
    system_prompt = build_system_prompt(entities)

    async def event_stream():
        yield f"data: {json.dumps({'session_id': session_id})}\n\n"
        async for chunk in stream_chat(session, system_prompt, confirmed=req.confirmed):
            yield chunk
        yield "data: {\"done\": true}\n\n"

    return StreamingResponse(event_stream(), media_type="text/event-stream")


# Serve frontend — must come last so API routes take priority
app.mount("/", StaticFiles(directory="frontend", html=True), name="frontend")
