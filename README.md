# Home Assistant Automation Assistant

Describe a Home Assistant automation in plain English, talk it through with an AI agent that knows your actual devices, review and edit the YAML it drafts, and deploy it to your house only after you confirm.

Writing automations by hand means remembering entity IDs, trigger syntax and YAML indentation. This project replaces that with a conversation: "turn the porch light on at sunset and off at 11" becomes a validated automation, written to Home Assistant and editable in its UI afterwards.

## How it works

Three processes cooperate. The browser only ever talks to the FastAPI backend.

```
Browser  (React + Vite)
   │  POST /api/chat      → streamed reply (Server-Sent Events)
   │  POST /api/validate  → YAML and entity checks
   ▼
Backend  (Python, FastAPI)          :8000
   ├─ agent loop with tool use (Anthropic SDK)
   ├─ system prompt built from your live entity list
   ├─ per-session history and the pending automation
   └─ the human-confirm gate
   ▼
HA bridge  (Go)                     :8080
   ├─ polls entity state over REST into a mutex-guarded cache
   ├─ reads area, device and entity registries over WebSocket, with reconnect
   └─ writes automations and triggers a reload
   ▼
Home Assistant                      :8123
```

A request goes like this:

1. You type (or say) what you want.
2. The backend builds a system prompt from the bridge's cached entities, grouped by area, and streams the model's reply.
3. The model can call three tools:

   | Tool | Purpose |
   |---|---|
   | `get_entity_details` | Look up the state and attributes of one device |
   | `propose_automation` | Show a named YAML draft in the side panel |
   | `create_automation` | Write the confirmed draft to Home Assistant |

4. The draft appears in a split pane next to the chat. You can edit the YAML directly; your edits are sent back with every message, so the agent always works from what you see.
5. Nothing is written until you press Confirm.

## The confirm gate

The model is told not to deploy without confirmation, but the system does not rely on that. `create_automation` is rejected in `backend/tools.py` unless the request carries `confirmed: true` from the UI, and the automation ID must match the pending draft. A model that tries to skip ahead gets an error back, and your house is unchanged.

## Features

- Streaming chat with an agentic tool-use loop
- Live device context: entity states plus area and device names
- Side-by-side YAML panel, editable while the conversation continues
- Validation before deploy: YAML parsing, required fields, and a check that every referenced entity exists
- Voice input and spoken replies through the browser's Web Speech API
- Connection status banner when the bridge or the registry is unavailable
- Automations are written as Home Assistant config entries, so they stay editable in the HA UI

## Requirements

- A running Home Assistant instance and a [long-lived access token](https://www.home-assistant.io/docs/authentication/#your-account-profile)
- An Anthropic API key
- Go 1.22+, Python 3.12+, Node 18+
- [`just`](https://github.com/casey/just)

## Quick start

```bash
just setup     # creates .env from .env.example
# fill in ANTHROPIC_API_KEY, HA_URL and HA_TOKEN in .env
just install   # Go modules, Python virtualenv, npm packages
just run       # builds everything and serves the app at http://localhost:8000
```

For development with hot reload (frontend on `:5173`, backend on `:8000`, bridge on `:8080`):

```bash
just dev
```

## Configuration

| Variable | Used by | Description |
|---|---|---|
| `ANTHROPIC_API_KEY` | backend | Claude API key |
| `HA_URL` | bridge | For example `http://homeassistant.local:8123` |
| `HA_TOKEN` | bridge | Long-lived Home Assistant access token |
| `BRIDGE_URL` | backend | Bridge address, default `http://localhost:8080` |

The model (`claude-sonnet-4-6`) and the entity domains included in context (lights, switches, climate, covers, media players, scenes, scripts and a few more) are set at the top of `backend/claude_client.py`.

## Project layout

```
backend/      FastAPI app: chat endpoint, agent loop, tools, validation, sessions
ha-bridge/    Go service: Home Assistant REST and WebSocket clients, caches, HTTP API
frontend/     React app: chat, split-pane YAML editor, voice, status and deploy banners
justfile      setup, install, build, dev and run recipes
plan.md       original architecture plan
```

## Limitations

- Built for personal use on a home network. There is no authentication on the app itself, so do not expose it to the internet.
- Sessions live in memory and are lost when the backend restarts.
- The Docker Compose files are a starting point and do not yet build the frontend into the image; use `just run`.
- Device names and states from your Home Assistant instance are sent to the Anthropic API as part of each request.
