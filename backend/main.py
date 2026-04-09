from fastapi import FastAPI
from fastapi.responses import JSONResponse

app = FastAPI(title="HA Automation Assistant")


@app.get("/health")
async def health():
    return JSONResponse({"status": "ok"})


# Phase 2: POST /api/chat with SSE streaming
# Phase 3: tool dispatch loop
