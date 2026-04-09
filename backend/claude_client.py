"""Anthropic SDK wrapper — agentic loop with tool use and SSE streaming."""
from __future__ import annotations

import json
from collections.abc import AsyncIterator

import anthropic

from .config import settings
from . import tools as tool_module

client = anthropic.AsyncAnthropic(api_key=settings.anthropic_api_key)


def _block_to_dict(block) -> dict:
    """Serialize a content block to only the fields the API accepts on replay."""
    if block.type == "text":
        return {"type": "text", "text": block.text}
    if block.type == "tool_use":
        return {"type": "tool_use", "id": block.id, "name": block.name, "input": block.input}
    return {"type": block.type}

MODEL = "claude-sonnet-4-6"
CONTEXT_DOMAINS = "light,switch,climate,cover,automation,media_player,scene,script,input_boolean"


async def stream_chat(
    session: dict,
    system_prompt: str,
    confirmed: bool = False,
) -> AsyncIterator[str]:
    """
    Run the agentic loop: stream text, dispatch tool calls, loop until end_turn.
    Yields SSE data lines. Appends all new messages to session['history'] when done.
    """
    messages = list(session["history"])  # snapshot — don't mutate during loop
    new_messages: list[dict] = []

    while True:
        async with client.messages.stream(
            model=MODEL,
            max_tokens=2048,
            system=system_prompt,
            messages=messages,
            tools=tool_module.TOOLS,
        ) as stream:
            async for text in stream.text_stream:
                yield f"data: {json.dumps({'text': text})}\n\n"

            final = await stream.get_final_message()

        # Serialize content to dicts for storage and replay.
        # Only include fields the API accepts — model_dump() adds internal fields that cause 400s.
        content_dicts = [_block_to_dict(b) for b in final.content]
        assistant_msg = {"role": "assistant", "content": content_dicts}
        messages.append(assistant_msg)
        new_messages.append(assistant_msg)

        if final.stop_reason != "tool_use":
            break

        # Dispatch all tool calls in this turn
        tool_results = []
        for block in final.content:
            if block.type != "tool_use":
                continue

            result_str = await tool_module.dispatch(block, session, confirmed)

            # Notify frontend when an automation is proposed
            if block.name == "propose_automation" and session.get("pending_automation"):
                yield f"data: {json.dumps({'automation': session['pending_automation']})}\n\n"

            tool_results.append({
                "type": "tool_result",
                "tool_use_id": block.id,
                "content": result_str,
            })

        tool_msg = {"role": "user", "content": tool_results}
        messages.append(tool_msg)
        new_messages.append(tool_msg)

    # Persist everything added this turn to session history
    session["history"].extend(new_messages)
