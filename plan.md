# HA Automation Builder — High-Level Architecture Plan

## Context

Creating automations in Home Assistant via YAML or the UI is tedious. This project builds a conversational LLM interface — with voice — that lets the user describe what they want in natural language, have a back-and-forth with an AI assistant to clarify intent, preview the generated automation, confirm it, and have it written directly to HA. Personal use only, runs on the home network.

---

## System Components

### 1. Frontend (React/Vite or plain HTML/JS)
- Chat thread UI (user messages, assistant responses, automation previews)
- Web Speech API for voice input (STT — browser-native, uses Chrome's engine)
- Web Speech SpeechSynthesis for voice output (TTS — browser-native)
- Push-to-talk button + voice mode toggle
- YAML preview panel with Confirm / Modify / Cancel flow
- All traffic goes to the backend — no direct HA or Claude API calls

### 2. Backend Middleware (Python FastAPI)
- `POST /api/chat` — receives messages, returns SSE stream of Claude response
- Manages session state: conversation history + pending automation proposal
- Builds Claude system prompt using HA context fetched from the Go bridge
- Handles Claude tool calls: dispatches to Go bridge, returns results back into conversation
- Enforces human-confirm gate: `create_automation` tool only executes when `confirmed: true` in request
- Serves built frontend static files (same port, single deploy unit)

### 3. HA Bridge (Go sidecar)
- Owns all communication with Home Assistant — Python backend never calls HA directly
- Maintains a persistent WebSocket connection to HA for entity/area/device registry data
- Automatic reconnect with backoff if the HA WebSocket drops
- In-memory entity cache with 30–60s TTL, refreshed in the background via a ticker goroutine
- Exposes a small HTTP API consumed by the FastAPI backend:
  - `GET /entities` — full cached entity list (used to build Claude system prompt)
  - `GET /entities/{id}` — single entity state (for `get_entity_details` tool)
  - `POST /automations` — write automation YAML to HA + trigger reload
  - `GET /health` — HA connection status
- Go patterns in use: goroutines for WebSocket keepalive + cache refresh, channels for reconnect coordination, `sync.RWMutex` for safe concurrent cache reads

### 4. Home Assistant (local network)
- Source of truth for all entity, area, and automation data
- Read via REST API (`GET /api/states`) and WebSocket (`config/*_registry/list`)
- Write automations via `POST /api/config/automation/config` (config entry approach — editable in HA UI)
- Reload after write: `POST /api/services/automation/reload`
- Auth: long-lived access token stored in backend `.env` only

### 4. Claude API (Anthropic)
- Model: `claude-sonnet-4-6` (fast, interactive)
- Used with tool use / function calling
- Three tools (see below)
- Full entity list in system prompt context (~5k–15k tokens for 100–200 entities — fine)

---

## Communication Flow

```
Browser
  │  POST /api/chat {session_id, message}
  ▼
FastAPI Backend
  ├── GET /entities → Go Bridge (cached entity list)
  ├── build system prompt (areas + entities + existing automation names)
  ├── call Claude API with history + tools
  │     Claude → text (clarifying question) or tool_use
  │       tool: get_entity_details → GET /entities/{id} → Go Bridge → HA
  │       tool: propose_automation  → store proposal in session, surface to UI
  │       tool: create_automation   → only if user confirmed → POST /automations → Go Bridge → HA
  └── stream response back via SSE
  ▼
Browser
  ├── render text response (+ speak it if voice mode on)
  └── if automation proposed: show YAML + Confirm/Cancel buttons
        → Confirm → POST /api/chat {confirmed: true}

Go Bridge (always running)
  ├── persistent WebSocket → HA (entity/area/device registry)
  ├── background ticker → refresh entity cache every 30–60s
  └── HTTP server on :8080 → FastAPI calls this, never HA directly
```

---

## Claude Tools

| Tool | Input | Backend Action | Notes |
|------|-------|----------------|-------|
| `get_entity_details` | `entity_id` or search term | `GET /api/states/{entity_id}` | Claude fetches details for a specific device |
| `propose_automation` | `{name, description, yaml, entities_used[]}` | Store in session, return to frontend | Soft tool — no HA write, surfaces proposal only |
| `create_automation` | `{automation_id}` | Write YAML to HA, reload automations | Only executes if `confirmed: true` flag in request — backend enforces this |

### System Prompt Structure
```
You are a Home Assistant automation assistant...

== Current HA Context ==
Areas: Living Room, Kitchen, Bedroom...
Entities:
  - light.living_room_ceiling (Living Room) - state: on
  - switch.garage_door (Garage) - state: closed
  ...
Existing Automations: [names only]

== Instructions ==
1. Ask clarifying questions if intent is ambiguous
2. Use propose_automation when ready — never create without user confirmation
3. Always explain in plain English before showing YAML
```

---

## Voice Layer

- **STT:** Web Speech API (`webkitSpeechRecognition`) — browser-native, no backend change needed
- **TTS:** `SpeechSynthesisUtterance` — reads Claude responses aloud when voice mode active
- Voice is a pure frontend concern — backend is unchanged
- Note: Chrome's Web Speech API sends audio to Google for transcription. Acceptable for personal home use. Whisper.cpp local STT is an option if privacy matters more later.

---

## Session State (Backend, In-Memory)

```python
sessions[session_id] = {
  "history": [...],                        # conversation turns
  "pending_automation": {                  # last proposed automation
    "id": "uuid", "name": "...", "yaml": "..."
  },
  "ha_context_cache": {                    # entity list cache
    "entities": [...], "cached_at": timestamp
  }
}
```

Ephemeral by default. Swap dict → SQLite for persistence across restarts if desired later.

---

## Deployment

**Development:** FastAPI on `localhost:8000`, Go bridge on `localhost:8080`, HA on `homeassistant.local:8123`

**Production:** Docker Compose on Pi/NUC alongside HA
```yaml
services:
  ha-bridge:
    build: ./ha-bridge
    ports: ["8080:8080"]
    env_file: .env
    restart: unless-stopped

  automation-builder:
    build: .
    ports: ["8000:8000"]
    env_file: .env
    depends_on: [ha-bridge]
    restart: unless-stopped
```

FastAPI serves both the API and the React static build from port 8000. Go bridge is internal — not exposed to the browser.

---

## Project Structure

```
automation-builder/
  backend/                        # Python FastAPI
    main.py                       # FastAPI app, routes, SSE, confirm flow
    session.py                    # Session store
    bridge_client.py              # HTTP client to Go bridge (replaces ha_client.py)
    claude_client.py              # Anthropic SDK wrapper + tool dispatch loop (critical)
    context_builder.py            # Assembles HA context from bridge response (critical)
    tools.py                      # Tool handler implementations
    models.py                     # Pydantic request/response models
    config.py                     # pydantic-settings, loads .env
  ha-bridge/                      # Go sidecar
    main.go                       # HTTP server setup, config
    ha/
      websocket.go                # Persistent WS connection + reconnect logic
      cache.go                    # Entity cache with TTL + RWMutex
      client.go                   # HA REST calls (automation write/reload)
    api/
      handlers.go                 # HTTP handlers: /entities, /automations, /health
  frontend/
    src/
      App.jsx                     # Chat UI, SSE consumer, voice handler (critical)
      AutomationPreview.jsx
    index.html
  docker-compose.yml
  .env.example
```

---

## Security

- API keys (`ANTHROPIC_API_KEY`, `HA_TOKEN`) in `.env` only — never in frontend
- No HTTPS needed (local network only)
- No auth needed (single user, home network)
- Block port 8000 from WAN at the router

---

## Implementation Sequence

1. Go bridge skeleton — HTTP server, `/health` endpoint, `.env` config
2. Go bridge HA layer — WebSocket connection, entity cache, REST automation write. Test in isolation against real HA
3. Backend skeleton — FastAPI, `.env`, `bridge_client.py` calling Go bridge, health check
4. Claude integration — basic chat without tools. Confirm end-to-end conversation works
5. Add tool use — wire `get_entity_details` and `propose_automation`. Test Claude calls them correctly
6. Frontend — minimal chat UI over SSE. Text only first
7. Voice layer — add Web Speech API on top of working text UI
8. Automation write — implement `create_automation` + confirm flow. Test real write to HA via bridge
9. Polish — YAML preview, reset button, error handling for HA offline / bridge unreachable

---

## Key Design Decisions

- **Backend proxy for Claude API:** Keeps API key server-side; never exposed in browser devtools
- **SSE over WebSocket for streaming:** SSE is simpler for one-way server→client stream; WebSocket reserved for HA real-time state if needed later
- **`propose` vs `create` tool split:** Most important safety call — Claude proposes, human confirms, backend enforces. Claude cannot write to HA unilaterally
- **Config entry approach for automation write:** `POST /api/config/automation/config` creates automations editable in the HA UI. Avoids mixing file-based and UI automations
- **Full entity list in context:** 100–200 entities ≈ 10k tokens, well within budget. Only needs optimization at 500+ entities
