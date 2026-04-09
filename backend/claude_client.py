"""Anthropic SDK wrapper — Phase 2: basic streaming chat, no tools yet."""
import json
from collections.abc import AsyncIterator

import anthropic

from .config import settings

client = anthropic.AsyncAnthropic(api_key=settings.anthropic_api_key)

MODEL = "claude-sonnet-4-6"

# Actionable domains sent to Claude — keeps context lean on a large HA setup
CONTEXT_DOMAINS = "light,switch,climate,cover,automation,media_player,scene,script,input_boolean"


async def stream_chat(
    history: list[dict],
    system_prompt: str,
) -> AsyncIterator[str]:
    """
    Stream a Claude response as SSE-ready JSON strings.
    Yields lines of the form: data: {...}\n\n
    """
    async with client.messages.stream(
        model=MODEL,
        max_tokens=1024,
        system=system_prompt,
        messages=history,
    ) as stream:
        async for text in stream.text_stream:
            yield f"data: {json.dumps({'text': text})}\n\n"
