# Laputa — Markdown Authority Module

**Current contract:** [`../docs/architecture/0012-laputa-markdown-clean-break.md`](../docs/architecture/0012-laputa-markdown-clean-break.md)

Laputa is Garden MemoryOS's personality authority and Persona history subsystem. It will be rebuilt around one profile-level directory containing exactly these Markdown authority files:

```text
IDENTITY.MD      RELATIONSHIP.MD  REDLINE.MD  USER.MD
DREAM.MD         DARK.MD          WORLD.MD
```

`ACTMEM.MD` is a separate profile-level cross-session activity file. It is not a Persona authority and is tool-only.

## Target Behavior

- Authority files use uppercase names, Markdown bodies, fixed body limits, atomic writes, and complete revision history with textual diffs.
- Initial setup atomically creates `IDENTITY.MD`, `RELATIONSHIP.MD`, `REDLINE.MD`, `USER.MD`, and `WORLD.MD` only when all five are absent. It does not create `DREAM.MD`.
- Frozen Core is a bounded projection of the first six authority files, captured once per session.
- `WORLD.MD` and `ACTMEM.MD` never enter automatic context assembly. They are explicit tool reads.
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
