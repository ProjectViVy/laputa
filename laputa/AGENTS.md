# Laputa — Markdown Authority Module

> **2026-10-11 policy supersession:** WORLD remains tool-only. ACTMEM context/schema/tool assumptions below are the baseline; selected VIVY subject-Recap policy overrides only those assumptions. Preserve existing Frozen Core immutability and Mission/Persona authorization. See [ADR-0018](../docs/architecture/0018-vivy-unified-context-and-recap-policy.md). Original design, interface and verification records below remain evidence of their recorded snapshots; this notice does not change their pass/fail results or claim the new policy is implemented.

**Current contract:** [`../docs/architecture/0012-laputa-markdown-clean-break.md`](../docs/architecture/0012-laputa-markdown-clean-break.md)

Laputa is Garden MemoryOS's personality authority and Persona history subsystem. Its implemented authority model uses one profile-level directory containing the original seven Markdown authority files:

```text
IDENTITY.MD      RELATIONSHIP.MD  REDLINE.MD  USER.MD
DREAM.MD         DARK.MD          WORLD.MD
```

The existing cognitive implementation additionally supports human-only `MISSION.MD` and its versioned Mission slot (ADR-0017). This policy adds no new Persona kind or Frozen Core slot.

`ACTMEM.MD` is a separate profile-level cross-session activity file. It is not a Persona authority. The selected VIVY policy replaces its active content with one subject-level free-form Recap; baseline tools/schema remain unchanged until an implementation plan lands.

## Target Behavior

- Authority files use uppercase names, Markdown bodies, fixed body limits, atomic writes, and complete revision history with textual diffs.
- Initial setup atomically creates `IDENTITY.MD`, `RELATIONSHIP.MD`, `REDLINE.MD`, `USER.MD`, and `WORLD.MD` only when all five are absent. It does not create `DREAM.MD`.
- Frozen Core preserves the six bounded Persona projections and existing human-controlled Mission v2 slot; it is captured once per session. No Recap edit creates permanent pinning or a new authority.
- `WORLD.MD` never enters automatic context assembly. Selected ACTMEM projections require the VIVY admission policy and never enter Frozen Core.
- Persona content review is distinct from dangerous runtime approval. `DREAM.MD`, `DARK.MD`, and user observations support the direct-write exceptions defined in ADR-0012.

## Retired Implementation

The existing `.laputa/sections/*.json` registry, section names, JSON Patch, `map[string]any` bodies, `memory_md`, `Commitment`, `Preferences`, and generic `GovernedService` Persona mutation are retired. They remain in source only as deletion targets until the clean-break implementation starts.

Do not add compatibility code, migration code, or fallback reads for them.

## Relation to Garden and EvoMap

Laputa does not own Mentle materials, report persistence, EvoMap mailbox state, or Skill lifecycle. EvoMap owns all capability artifacts. Garden owns runtime orchestration and ContextView assembly.

## Test Baseline

```bash
cd laputa
GOSUMDB=off go test ./...
```

During cutover, run focused `./persona/...` and `./actmem/...` tests first, then the full module. New tests must be Markdown-first and prove the old JSON personality path is unreachable.
