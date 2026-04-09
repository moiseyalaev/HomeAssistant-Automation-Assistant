"""HTTP client for the Go HA bridge."""
import httpx
from .config import settings


async def get_entities() -> list[dict]:
    async with httpx.AsyncClient() as client:
        r = await client.get(f"{settings.bridge_url}/entities")
        r.raise_for_status()
        return r.json()


async def get_entity(entity_id: str) -> dict:
    async with httpx.AsyncClient() as client:
        r = await client.get(f"{settings.bridge_url}/entities/{entity_id}")
        r.raise_for_status()
        return r.json()


async def write_automation(automation_id: str, yaml_content: str) -> None:
    async with httpx.AsyncClient() as client:
        r = await client.post(
            f"{settings.bridge_url}/automations",
            json={"id": automation_id, "yaml": yaml_content},
        )
        r.raise_for_status()
