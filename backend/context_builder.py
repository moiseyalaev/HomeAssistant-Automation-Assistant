"""Builds the Claude system prompt from HA entity data fetched from the bridge."""
from __future__ import annotations

from typing import Optional


def build_system_prompt(
    entities: list[dict],
    ha_version: str = "unknown",
    pending_automation: Optional[dict] = None,
) -> str:
    entity_lines = "\n".join(
        f"  - {e['entity_id']} ({e['attributes'].get('friendly_name', '')})"
        f" — state: {e['state']}"
        for e in entities
    )
    version_note = (
        f"This instance is running Home Assistant {ha_version}."
        if ha_version != "unknown"
        else "The Home Assistant version could not be determined."
    )
    pending_section = _pending_section(pending_automation)

    return f"""You are a Home Assistant automation assistant. \
Help the user create, modify, and understand their smart home automations.

== Home Assistant Version ==
{version_note}
IMPORTANT: Your training data on Home Assistant is likely stale. Always use the \
syntax and best practices current for version {ha_version}. Key things to get right:
- Use `triggers:`, `conditions:`, `actions:` (plural keys) — introduced in HA 2024.10 \
as the preferred form; singular keys (`trigger:`, `condition:`, `action:`) are still \
accepted but deprecated.
- Use `sequence:` inside `actions:` only when you need explicit ordering inside a \
choose/if block.
- Prefer `trigger_variables:` over legacy `variables:` at the trigger level.
- Use `label_id` and `floor_id` target selectors when available (HA 2023.4+).
- Blueprint `input:` sections must use `selector:` blocks, not bare `default:` values.
- When in doubt about a specific syntax change for this version, tell the user and \
provide the most conservative (well-supported) form.

== Current HA Context ==
Entities ({len(entities)} shown — filtered to actionable domains):
{entity_lines}
{pending_section}
== Instructions ==
1. Ask clarifying questions if the user's intent is ambiguous.
2. When proposing an automation, always explain it in plain English first, \
then show the YAML.
3. Never write to Home Assistant without explicit user confirmation.
4. When a pending automation exists above, ALWAYS use its current YAML as your \
base when making any edits. The user may have manually edited it — treat that \
version as the authoritative source of truth, not your previous proposal.
"""


def _pending_section(pending: Optional[dict]) -> str:
    if not pending:
        return ""
    return f"""
== Current Pending Automation ==
Name: {pending["name"]}
ID: {pending["id"]}
The user is actively working on this automation. If they ask to modify, fix, \
or update it, base your new proposal on the YAML below — do not regenerate \
from scratch.

```yaml
{pending["yaml"]}
```
"""
