# Laputa

This module is being replaced under [`../docs/architecture/0012-laputa-markdown-clean-break.md`](../docs/architecture/0012-laputa-markdown-clean-break.md).

## Target

Laputa will be a profile-level Markdown authority store with exactly:

```text
IDENTITY.MD      RELATIONSHIP.MD  REDLINE.MD  USER.MD
DREAM.MD         DARK.MD          WORLD.MD
```

It also defines `ACTMEM.MD` as a separate cross-session activity-memory file.

The first six authority files provide bounded, session-frozen Frozen Core projections. `WORLD.MD` and `ACTMEM.MD` are explicit tool reads and must never be automatic context.

## Retired

The current JSON sections implementation is not a supported target model. Do not use or extend `.laputa/sections/*.json`, section numbering, `Commitment`, `Preferences`, `memory_md`, `MEMORY.MD`, JSON Patch, or generic governance mutation to build new behavior.

There is no migration, compatibility mapping, dual read/write, or fallback route from those files to the Markdown authority model.

## Boundaries

Laputa owns personality authority, Persona review/history, and ACTMEM semantics. Mentle owns evidence and retrieval. Garden owns runtime orchestration. EvoMap owns every capability artifact lifecycle.

## Verification

```bash
GOSUMDB=off go test ./...
```

The current tests validate the Markdown-first clean-break implementation and its deletion boundaries. Archived JSON-section tests are historical evidence only.
