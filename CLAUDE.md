# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

All commands assume a `.env` file exists (copy from `.env.example`).

```bash
# First-time setup
just setup    # create .env from template
just install  # Go deps + Python venv + npm install

# Development (hot-reload frontend at localhost:5173, backend at :8000)
just dev

# Production (serves built frontend from FastAPI at localhost:8000)
just run

# Build only (Go binary + npm run build → frontend-dist/)
just build
```

Running services individually:
```bash
# Go bridge
HA_URL=... HA_TOKEN=... ./ha-bridge/bin/ha-bridge

# Python backend
ANTHROPIC_API_KEY=... BRIDGE_URL=http://localhost:8080 \
  .venv/bin/uvicorn backend.main:app --host 0.0.0.0 --port 8000 --reload

# Frontend dev server (proxies /api/* to :8000)
cd frontend && npm run dev

# Frontend production build (outputs to ../frontend-dist/)
cd frontend && npm run build
```

## Architecture

Three processes collaborate; the browser only ever talks to the FastAPI backend.

```
Browser (React/Vite :5173 dev / :8000 prod)
  │  POST /api/chat  →  SSE stream
  │  POST /api/validate
  ▼
FastAPI backend (:8000)          Python, .venv/
  ├── session.py                 in-memory {history, pending_automation} per session
  ├── claude_client.py           Anthropic SDK agentic loop — streams text, dispatches tools
  ├── context_builder.py         builds system prompt (entity list + pending automation YAML)
  ├── tools.py                   3 tools: get_entity_details, propose_automation, create_automation
  ├── bridge_client.py           httpx calls to Go bridge
  └── main.py                    FastAPI routes + CORS + static file mount (frontend-dist/)
  ▼
Go bridge (:8080)                ha-bridge/
  ├── ha/client.go               polls HA REST /api/states every 10s
  ├── ha/cache.go                RWMutex entity cache (TTL 10s)
  └── api/handlers.go            GET /entities, GET /entities/{id}, POST /automations, GET /health
  ▼
Home Assistant (:8123)
  └── REST API — read states, write automations via /api/config/automation/config/{id}
```

### Key data flows

**Chat request:** Browser `POST /api/chat` → backend syncs `edited_yaml` into session → builds system prompt including current pending automation YAML → streams Claude response as SSE → tool calls dispatched inline → `propose_automation` tool stores automation in `session["pending_automation"]` and emits `data: {"automation": ...}` SSE event.

**Confirm/deploy:** Browser sends `confirmed: true` → backend enforces gate in `tools.py` → `create_automation` tool calls `bridge_client.write_automation` → Go bridge parses YAML, POSTs config to HA, triggers reload.

**YAML validation:** Browser `POST /api/validate` → pyyaml parse → field checks → entity ID walk (only under `entity_id`/`entity_ids` keys in parsed structure, not raw regex, to avoid matching service names).

### Claude tool contract

| Tool | When called | Backend action |
|------|-------------|----------------|
| `get_entity_details` | needs device info | `GET /entities/{id}` |
| `propose_automation` | ready to show YAML | stores in session, SSE push to frontend |
| `create_automation` | only when `confirmed=true` in request | writes YAML to HA via bridge |

`create_automation` silently fails if `confirmed` is not set — this is the safety gate and is enforced in `tools.py::dispatch`, not in Claude.

### Frontend split-pane design

When `pendingAutomation` state is non-null, `App.jsx` renders a draggable split pane (left/right or top/bottom). The automation YAML panel stays open during further chat — Claude always receives the current edited YAML via `edited_yaml` in every request body, which the backend syncs into `session["pending_automation"]` before building the system prompt.

### Session state

Sessions are in-memory dicts in `backend/session.py`. Restarting the backend loses all sessions. The structure is:
```python
{"history": [...], "pending_automation": {"id", "name", "description", "yaml"} | None}
```

### Environment variables

| Variable | Used by | Description |
|----------|---------|-------------|
| `ANTHROPIC_API_KEY` | backend | Claude API key |
| `HA_URL` | Go bridge | e.g. `http://homeassistant.local:8123` |
| `HA_TOKEN` | Go bridge | Long-lived HA access token |
| `BRIDGE_URL` | backend | Go bridge address, default `http://localhost:8080` |

### Model

`claude-sonnet-4-6` — set in `backend/claude_client.py::MODEL`. Domains included in entity context: `light,switch,climate,cover,automation,media_player,scene,script,input_boolean` — set in `CONTEXT_DOMAINS`.
