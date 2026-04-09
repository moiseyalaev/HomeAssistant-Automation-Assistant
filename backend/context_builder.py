"""Builds the Claude system prompt from HA entity data fetched from the bridge."""


def build_system_prompt(entities: list[dict]) -> str:
    entity_lines = "\n".join(
        f"  - {e['entity_id']} ({e['attributes'].get('friendly_name', '')})"
        f" — state: {e['state']}"
        for e in entities
    )
    return f"""You are a Home Assistant automation assistant. \
Help the user create, modify, and understand their smart home automations.

== Current HA Context ==
Entities ({len(entities)} shown — filtered to actionable domains):
{entity_lines}

== Instructions ==
1. Ask clarifying questions if the user's intent is ambiguous.
2. When proposing an automation, always explain it in plain English first, \
then show the YAML.
3. Never write to Home Assistant without explicit user confirmation.
"""
