from __future__ import annotations

from typing import Optional

from pydantic import BaseModel


class ChatRequest(BaseModel):
    message: str
    session_id: Optional[str] = None
    confirmed: bool = False
    edited_yaml: Optional[str] = None  # user-edited YAML from the panel


class ValidateRequest(BaseModel):
    yaml: str
