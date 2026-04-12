"""Claude tool definitions and dispatch logic for HA automation."""
from __future__ import annotations

import json
import logging
import uuid

from . import bridge_client as bridge

log = logging.getLogger(__name__)

TOOLS = [
    {
        "name": "get_entity_details",
        "description": (
            "Get the current state and all attributes for a specific Home Assistant entity. "
            "Use this when you need more detail about a device before writing an automation."
        ),
        "input_schema": {
            "type": "object",
            "properties": {
                "entity_id": {
                    "type": "string",
                    "description": "The entity_id to look up, e.g. light.living_room_ceiling",
                }
            },
            "required": ["entity_id"],
        },
    },
    {
        "name": "propose_automation",
        "description": (
            "Propose an automation to the user. Surfaces a YAML preview with Confirm/Cancel "
            "buttons in the UI. Do NOT call create_automation — wait for the user to confirm."
        ),
        "input_schema": {
            "type": "object",
            "properties": {
                "name": {
                    "type": "string",
                    "description": "Short human-readable name for the automation",
                },
                "description": {
                    "type": "string",
                    "description": "Plain English explanation of what this automation does",
                },
                "yaml": {
                    "type": "string",
                    "description": "Complete Home Assistant automation YAML, ready to be written",
                },
            },
            "required": ["name", "description", "yaml"],
        },
    },
    {
        "name": "create_automation",
        "description": (
            "Write a confirmed automation to Home Assistant. Only call this after the user "
            "has explicitly confirmed the proposed automation. The backend enforces this — "
            "the call will fail if the user has not confirmed."
        ),
        "input_schema": {
            "type": "object",
            "properties": {
                "automation_id": {
                    "type": "string",
                    "description": "The id returned when propose_automation was called",
                }
            },
            "required": ["automation_id"],
        },
    },
]


async def dispatch(block, session: dict, confirmed: bool) -> str:
    name = block.name
    inp = block.input

    if name == "get_entity_details":
        try:
            entity = await bridge.get_entity(inp["entity_id"])
            return json.dumps(entity)
        except Exception as e:
            return f"Error fetching entity: {e}"

    if name == "propose_automation":
        automation_id = str(uuid.uuid4())[:8]
        session["pending_automation"] = {
            "id": automation_id,
            "name": inp["name"],
            "description": inp["description"],
            "yaml": inp["yaml"],
        }
        log.info("propose_automation: stored id=%s name=%r", automation_id, inp["name"])
        return (
            f"Automation proposed (id={automation_id}). "
            "The user now sees a preview with Confirm/Cancel. "
            "Tell them to confirm if it looks correct."
        )

    if name == "create_automation":
        log.info("create_automation: called — confirmed=%s input=%s", confirmed, inp)
        if not confirmed:
            log.warning("create_automation: rejected — not confirmed")
            return "Cannot write automation: user has not confirmed yet."
        pending = session.get("pending_automation")
        if not pending:
            log.warning("create_automation: no pending automation in session")
            return "No pending automation found in this session."
        if pending["id"] != inp.get("automation_id"):
            log.warning(
                "create_automation: ID mismatch — pending=%s claude_sent=%s",
                pending["id"], inp.get("automation_id"),
            )
            return f"Automation ID mismatch. Expected {pending['id']}."
        log.info("create_automation: writing id=%s name=%r", pending["id"], pending["name"])
        try:
            await bridge.write_automation(pending["id"], pending["yaml"])
            session["pending_automation"] = None
            log.info("create_automation: success")
            return f"Automation '{pending['name']}' successfully written to Home Assistant."
        except Exception as e:
            log.error("create_automation: bridge error: %s", e)
            return f"Error writing automation to HA: {e}"

    return f"Unknown tool: {name}"
