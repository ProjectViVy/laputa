# Laputa

The owner's [2026-10-31 direction](../docs/architecture/0016-laputa-protocol-and-native-agent-governance.md)
repositions Laputa as a persona/memory-governance specification/protocol with
host-native implementations. Reuse the host's persona files/APIs, long-term
backend and Recap engine; Garden may supply a collection-to-effective-update
workflow. This does not require every agent to import this Go module or run a
Garden service. The new Vivy prefers a small Rust persona implementation.

The module below is the existing Go implementation governed by
[ADR-0012](../docs/architecture/0012-laputa-markdown-clean-break.md). ADR-0016's
protocol, native adapters and reusable workflow are not delivered by this record.

## Existing Go Profile

The Go profile has a Markdown authority store with these files:

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

Within the existing Go profile, Laputa owns personality authority, Persona
review/history and ACTMEM semantics; Mentle owns evidence/retrieval, Garden owns
application orchestration and EvoMap owns capability artifacts. These do not
impose a second persona or ACTMEM store on a native host adapter.

## Verification

```bash
GOSUMDB=off go test ./...
```

The current tests validate the Markdown-first clean-break implementation and its deletion boundaries. Archived JSON-section tests are historical evidence only.
