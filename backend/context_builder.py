"""Builds the Claude system prompt from HA entity data fetched from the bridge."""


def build_system_prompt(entities: list[dict]) -> str:
    entity_lines = "\n".join(
        f"  - {e['entity_id']} — state: {e['state']}"
        for e in entities
    )
    return f"""You are a Home Assistant automation assistant.

== Current HA Context ==
Entities:
{entity_lines}

== Instructions ==
1. Ask clarifying questions if the user's intent is ambiguous.
2. Use propose_automation when ready — never call create_automation without user confirmation.
3. Always explain the automation in plain English before showing YAML.
"""
