"""HTTP client for the Go HA bridge."""
from __future__ import annotations

from typing import Optional

import httpx

from .config import settings


async def get_entities(domains: Optional[str] = None) -> list[dict]:
    params = {"domains": domains} if domains else {}
    async with httpx.AsyncClient() as client:
        r = await client.get(f"{settings.bridge_url}/entities", params=params)
        r.raise_for_status()
        return r.json()


async def get_entity(entity_id: str) -> dict:
    async with httpx.AsyncClient() as client:
        r = await client.get(f"{settings.bridge_url}/entities/{entity_id}")
        r.raise_for_status()
        return r.json()


async def write_automation(automation_id: str, yaml_content: str) -> None:
    import yaml  # pyyaml — already in requirements.txt

    config = yaml.safe_load(yaml_content)
    if not isinstance(config, dict):
        raise ValueError("Automation YAML must be a mapping")

    # Strip 'id' from the config body — we pass it separately as the URL key
    config.pop("id", None)

    async with httpx.AsyncClient() as client:
        r = await client.post(
            f"{settings.bridge_url}/automations",
            json={"id": automation_id, "config": config},
        )
        r.raise_for_status()
