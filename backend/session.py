"""In-memory session store. Swap dict → SQLite later if persistence needed."""
from __future__ import annotations

import uuid
from typing import Any, Optional, Tuple


sessions: dict[str, dict[str, Any]] = {}


def get_or_create(session_id: Optional[str]) -> Tuple[str, dict]:
    if session_id is None or session_id not in sessions:
        session_id = str(uuid.uuid4())
        sessions[session_id] = {
            "history": [],
            "pending_automation": None,
        }
    return session_id, sessions[session_id]


def clear(session_id: str) -> None:
    sessions.pop(session_id, None)
