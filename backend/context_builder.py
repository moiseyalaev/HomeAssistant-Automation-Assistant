"""Builds the Claude system prompt from HA entity data fetched from the bridge."""
from __future__ import annotations

from collections import defaultdict
from typing import Optional


def _group_entities(entities: list[dict]) -> dict[str, dict[str | None, list[dict]]]:
    """Group entities by area_name, then by device_name.

    Returns {area_name: {device_name_or_None: [entity, ...]}}
    where area_name defaults to "Unassigned" for empty strings and
    device_name is None when empty (entities listed directly under area).
    """
    grouped: dict[str, dict[str | None, list[dict]]] = defaultdict(lambda: defaultdict(list))
    for e in entities:
        area = e.get("area_name") or "Unassigned"
        device = e.get("device_name") or None
        grouped[area][device].append(e)
    return grouped


def _render_grouped_entities(entities: list[dict]) -> str:
    grouped = _group_entities(entities)
    lines: list[str] = []
    for area in sorted(grouped.keys()):
        lines.append(f"{area}:")
        devices = grouped[area]
        # Entities with no device first, then named devices sorted
        for device_key in [None] + sorted(k for k in devices if k is not None):
            device_entities = devices[device_key]
            if device_key is None:
                for e in device_entities:
                    lines.append(
                        f"  - {e['entity_id']} — state: {e['state']}"
                    )
            else:
                first = device_entities[0]
                manufacturer = first.get("manufacturer", "")
                model = first.get("model", "")
                if manufacturer and model:
                    subheading = f"  {device_key} ({manufacturer} {model}):"
                elif manufacturer:
                    subheading = f"  {device_key} ({manufacturer}):"
                else:
                    subheading = f"  {device_key}:"
                lines.append(subheading)
                for e in device_entities:
                    lines.append(
                        f"    - {e['entity_id']} — state: {e['state']}"
                    )
        lines.append("")  # blank line between areas
    # Remove trailing blank line
    while lines and lines[-1] == "":
        lines.pop()
    return "\n".join(lines)


def _render_flat_entities(entities: list[dict]) -> str:
    return "\n".join(
        f"  - {e['entity_id']} ({e['attributes'].get('friendly_name', '')})"
        f" — state: {e['state']}"
        for e in entities
    )


def build_system_prompt(
    entities: list[dict],
    ha_version: str = "unknown",
    pending_automation: Optional[dict] = None,
    registry_available: bool = True,
) -> str:
    version_note = (
        f"This instance is running Home Assistant {ha_version}."
        if ha_version != "unknown"
        else "The Home Assistant version could not be determined."
    )
    pending_section = _pending_section(pending_automation)

    if registry_available:
        entity_header = f"Entities ({len(entities)} total, grouped by area and device):"
        entity_body = _render_grouped_entities(entities)
        registry_note = ""
    else:
        entity_header = f"Entities ({len(entities)} shown — filtered to actionable domains):"
        entity_body = _render_flat_entities(entities)
        registry_note = (
            "\nNote: device and area context is currently unavailable (registry service unreachable).\n"
            "You only have entity IDs and friendly names. If the user's request is ambiguous about\n"
            "which specific device they mean, ask them to clarify rather than guessing.\n"
        )

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
{entity_header}
{entity_body}
{registry_note}{pending_section}
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
