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

    # Append user message to history
    if req.message:
        session["history"].append({"role": "user", "content": req.message})

    # Fetch filtered entity context from Go bridge
    entities = await get_entities(domains=CONTEXT_DOMAINS)
    system_prompt = build_system_prompt(entities)

    async def event_stream():
        # Send session_id first so the frontend can track the session
        yield f"data: {json.dumps({'session_id': session_id})}\n\n"

        full_response = ""
        async for chunk in stream_chat(session["history"], system_prompt):
            yield chunk
            # Accumulate text from SSE chunks to store in history
            try:
                data = json.loads(chunk.removeprefix("data: ").strip())
                full_response += data.get("text", "")
            except Exception:
                pass

        # Store assistant response in history for next turn
        if full_response:
            session["history"].append(
                {"role": "assistant", "content": full_response}
            )

        yield "data: {\"done\": true}\n\n"

    return StreamingResponse(event_stream(), media_type="text/event-stream")


# Serve frontend static files — must come last so API routes take priority
app.mount("/", StaticFiles(directory="frontend", html=True), name="frontend")
