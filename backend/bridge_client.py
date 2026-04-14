"""HTTP client for the Go HA bridge."""
from __future__ import annotations

from typing import Optional

import httpx
import yaml

from .config import settings

# Module-level persistent client — reuses the connection pool across all calls.
# Closed in main.py's lifespan handler via close_client().
_client: httpx.AsyncClient | None = None


def get_client() -> httpx.AsyncClient:
    global _client
    if _client is None or _client.is_closed:
        _client = httpx.AsyncClient()
    return _client


async def close_client() -> None:
    global _client
    if _client is not None and not _client.is_closed:
        await _client.aclose()
    _client = None


async def get_ha_version() -> str:
    """Return the running HA version string, e.g. '2024.4.1'. Returns 'unknown' on failure."""
    try:
        r = await get_client().get(f"{settings.bridge_url}/version", timeout=5)
        r.raise_for_status()
        return r.json().get("version", "unknown")
    except Exception:
        return "unknown"


async def get_entities(domains: Optional[str] = None) -> list[dict]:
    params = {"domains": domains} if domains else {}
    r = await get_client().get(f"{settings.bridge_url}/entities", params=params)
    r.raise_for_status()
    return r.json()


async def get_entity(entity_id: str) -> dict:
    r = await get_client().get(f"{settings.bridge_url}/entities/{entity_id}")
    r.raise_for_status()
    return r.json()


async def write_automation(automation_id: str, yaml_content: str) -> None:
    config = yaml.safe_load(yaml_content)
    if not isinstance(config, dict):
        raise ValueError("Automation YAML must be a mapping")

    # Strip 'id' from the config body — we pass it separately as the URL key
    config.pop("id", None)

    r = await get_client().post(
        f"{settings.bridge_url}/automations",
        json={"id": automation_id, "config": config},
    )
    r.raise_for_status()
