# Laputa — Protocol Direction and Existing Markdown Module

**Current direction:** [ADR-0016](../docs/architecture/0016-laputa-protocol-and-native-agent-governance.md)

New Laputa is a persona/memory-governance specification/protocol. Reuse native
persona files/APIs, long-term backend and agent-owned Recap. Garden may provide
a closed-loop governance workflow. Do not impose a Go library/server, seven-file
layout, ACTMEM engine or review policy on every host. This is a design decision,
not an implemented module replacement.

**Existing Go contract:** [`../docs/architecture/0012-laputa-markdown-clean-break.md`](../docs/architecture/0012-laputa-markdown-clean-break.md)

The existing Go module is Garden's personality authority and Persona history
subsystem, with one profile-level directory containing these Markdown files:

```text
IDENTITY.MD      RELATIONSHIP.MD  REDLINE.MD  USER.MD
DREAM.MD         DARK.MD          WORLD.MD
```

`ACTMEM.MD` is a separate profile-level cross-session activity file. It is not a Persona authority and is tool-only.

## Existing Go Profile Behavior

- Authority files use uppercase names, Markdown bodies, fixed body limits, atomic writes, and complete revision history with textual diffs.
- Initial setup atomically creates `IDENTITY.MD`, `RELATIONSHIP.MD`, `REDLINE.MD`, `USER.MD`, and `WORLD.MD` only when all five are absent. It does not create `DREAM.MD`.
- Frozen Core is a bounded projection of the first six authority files, captured once per session.
- `WORLD.MD` and `ACTMEM.MD` never enter automatic context assembly. They are explicit tool reads.
- Persona content review is distinct from dangerous runtime approval. `DREAM.MD`, `DARK.MD`, and user observations support the direct-write exceptions defined in ADR-0012.

## Retired Implementation

The `.laputa/sections/*.json` registry, section names, JSON Patch, `map[string]any` bodies, `memory_md`, `Commitment`, `Preferences`, and generic `GovernedService` Persona mutation are retired and are not supported authority paths in the existing Go contract.

Do not add compatibility code, migration code, or fallback reads for them.

## Relation to Garden and EvoMap

Laputa does not own Mentle materials, report persistence, EvoMap mailbox state, or Skill lifecycle. EvoMap owns all capability artifacts. Garden owns runtime orchestration and ContextView assembly.

## Test Baseline

```bash
cd laputa
GOSUMDB=off go test ./...
```

During cutover, run focused `./persona/...` and `./actmem/...` tests first, then the full module. New tests must be Markdown-first and prove the old JSON personality path is unreachable.
