# Laputa

This module implements the Markdown authority clean break described by [ADR-0012](../docs/architecture/0012-laputa-markdown-clean-break.md). It remains independently importable; Garden assembles it with Mentle and applies agent-facing policy.

## Target

Laputa is a profile-level Markdown authority store with exactly:

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

External Go hosts should use Garden's [public in-process API](../garden/README.md#in-process-go-host) for governed context and capture. Direct Laputa Persona/ACTMEM calls remain appropriate for trusted operator setup, but `WORLD.MD`/`ACTMEM.MD` are never silently added to automatic context; agent principal/review policy lives at Garden's entrypoint. See the [Vivy handoff](../docs/bmad/laputa-modular-monolith-2026-09/vivy-handoff.md); Vivy is not yet integrated.

## Verification

```bash
CGO_ENABLED=0 GOSUMDB=off go test ./... -count=1
```

The current tests validate the Markdown-first clean-break implementation and its deletion boundaries. Archived JSON-section tests are historical evidence only.
