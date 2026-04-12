set dotenv-load

bridge_bin := "ha-bridge/bin/ha-bridge"

# List available recipes
default:
    @just --list

# Create .env from template — skips if already exists
setup:
    @test -f .env \
        && echo ".env already exists — skipping" \
        || (cp .env.example .env && echo "Created .env — fill in HA_URL, HA_TOKEN, and ANTHROPIC_API_KEY")

# Install all dependencies
install:
    @echo "→ Go dependencies"
    cd ha-bridge && go mod tidy
    @echo "→ Python virtual environment"
    python3 -m venv .venv
    @echo "→ Python packages"
    .venv/bin/pip install -q -r backend/requirements.txt
    @echo "→ Frontend dependencies"
    cd frontend && npm install
    @echo "✓ Done — run 'just run' to start"

# Build the Go bridge binary + frontend
build:
    mkdir -p ha-bridge/bin
    cd ha-bridge && go build -o bin/ha-bridge .
    cd frontend && npm run build

# Start all services + Vite dev server — Ctrl+C shuts everything down
dev: build
    #!/usr/bin/env bash
    set -euo pipefail

    echo "▶  Starting HA bridge   → http://localhost:8080"
    HA_URL="$HA_URL" HA_TOKEN="$HA_TOKEN" ./{{bridge_bin}} &
    BRIDGE_PID=$!

    echo "▶  Starting backend     → http://localhost:8000"
    ANTHROPIC_API_KEY="$ANTHROPIC_API_KEY" BRIDGE_URL="http://localhost:8080" \
        .venv/bin/uvicorn backend.main:app --host 0.0.0.0 --port 8000 &
    BACKEND_PID=$!

    echo "▶  Starting Vite dev    → http://localhost:5173"
    cd frontend && npm run dev &
    VITE_PID=$!

    cleanup() {
        echo ""
        echo "■  Stopping services..."
        kill "$VITE_PID" "$BACKEND_PID" "$BRIDGE_PID" 2>/dev/null || true
        wait "$VITE_PID" "$BACKEND_PID" "$BRIDGE_PID" 2>/dev/null || true
        echo "■  Stopped."
    }
    trap cleanup INT TERM EXIT

    wait

# Start both services (production) — Ctrl+C shuts both down cleanly
run: build
    #!/usr/bin/env bash
    set -euo pipefail

    echo "▶  Starting HA bridge   → http://localhost:8080"
    HA_URL="$HA_URL" HA_TOKEN="$HA_TOKEN" ./{{bridge_bin}} &
    BRIDGE_PID=$!

    cleanup() {
        echo ""
        echo "■  Stopping services..."
        kill "$BRIDGE_PID" 2>/dev/null || true
        wait "$BRIDGE_PID" 2>/dev/null || true
        echo "■  Stopped."
    }
    trap cleanup INT TERM EXIT

    echo "▶  Starting backend     → http://localhost:8000"
    ANTHROPIC_API_KEY="$ANTHROPIC_API_KEY" BRIDGE_URL="http://localhost:8080" \
        .venv/bin/uvicorn backend.main:app --host 0.0.0.0 --port 8000
