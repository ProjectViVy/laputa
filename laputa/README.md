# Laputa

> **2026-10-11 policy supersession:** WORLD remains tool-only. ACTMEM context/schema/tool assumptions below are the baseline; selected VIVY subject-Recap policy overrides only those assumptions. Preserve existing Frozen Core immutability and Mission/Persona authorization. See [ADR-0018](../docs/architecture/0018-vivy-unified-context-and-recap-policy.md). Original design, interface and verification records below remain evidence of their recorded snapshots; this notice does not change their pass/fail results or claim the new policy is implemented.

This module implements the Markdown authority clean break described by [ADR-0012](../docs/architecture/0012-laputa-markdown-clean-break.md). It remains independently importable; Garden assembles it with Mentle and applies agent-facing policy.

## Target

Laputa is a profile-level Markdown authority store with the original seven files:

```text
IDENTITY.MD      RELATIONSHIP.MD  REDLINE.MD  USER.MD
DREAM.MD         DARK.MD          WORLD.MD
```

The existing cognitive implementation additionally supports human-only `MISSION.MD` and its versioned Mission slot (ADR-0017). This policy adds no new Persona kind or Frozen Core slot.

It also defines `ACTMEM.MD` as a separate cross-session activity-memory file.

The first six authority files provide bounded, session-frozen Frozen Core projections. `WORLD.MD` remains an explicit tool read. ACTMEM follows ADR-0018 only under the selected VIVY context policy; baseline APIs retain their implemented behavior until revised.

## Retired

The current JSON sections implementation is not a supported target model. Do not use or extend `.laputa/sections/*.json`, section numbering, `Commitment`, `Preferences`, `memory_md`, `MEMORY.MD`, JSON Patch, or generic governance mutation to build new behavior.

There is no migration, compatibility mapping, dual read/write, or fallback route from those files to the Markdown authority model.

## Boundaries

Laputa owns personality authority, Persona review/history, and ACTMEM semantics. Mentle owns evidence and retrieval. Garden owns runtime orchestration. EvoMap owns every capability artifact lifecycle.

External Go hosts should use Garden's [public in-process API](../garden/README.md#in-process-go-host) for governed context and capture. Direct Laputa Persona/ACTMEM calls remain appropriate for trusted operator setup, while `WORLD.MD` remains tool-only and selected ACTMEM projections require VIVY scope/budget admission; agent principal/review policy lives at Garden's entrypoint. See the [Vivy handoff](../docs/bmad/laputa-modular-monolith-2026-09/vivy-handoff.md); the original handoff is historical; current merged DIVA memory integration and remaining gates are indexed in the [alignment inventory](../docs/dev/memory-policy-alignment-2026-10-11.md).

## Verification

```bash
CGO_ENABLED=0 GOSUMDB=off go test ./... -count=1
```

The current tests validate the Markdown-first clean-break implementation and its deletion boundaries. Archived JSON-section tests are historical evidence only.
