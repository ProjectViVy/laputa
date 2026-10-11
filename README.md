# LAPUTA

[中文文档](README_CN.md)

Garden MemoryOS is a governed operating system for continuous agents. It connects material evidence, bounded working context, personality authority, and reusable capability without treating any one of them as another.

## Go modules

The repository owns three Go modules:

| Directory | Import path |
| --- | --- |
| `garden/` | `github.com/ProjectViVy/laputa/garden` |
| `mentle/` | `github.com/ProjectViVy/laputa/mentle` |
| `laputa/` | `github.com/ProjectViVy/laputa/laputa` |

Clone this repository once to obtain all three modules. Garden's local
replacements resolve inside this checkout. Laputa pins INOFY to a published
revision; no sibling `INOFY/` directory is needed. Build `garden/console` before
building or testing the standalone Garden application because it embeds the
console's `dist/` output.

## Current Architecture

[ADR-0012: Laputa Markdown Clean Break](docs/architecture/0012-laputa-markdown-clean-break.md) is the preserved authority contract. [ADR-0018](docs/architecture/0018-vivy-unified-context-and-recap-policy.md) records the current selected VIVY context direction; implementation is pending.

[ADR-0013](docs/architecture/0013-laputa-clean-break-implementation-architecture.md) defines the clean-break implementation order. [ADR-0014](docs/architecture/0014-mentle-canonical-authority-and-derived-index-recovery.md) makes canonical SQLite the sole Mentle memory authority and treats vector/BM25 indexes as rebuildable projections.

```text
Laputa  = governed Markdown personality authorities + Persona review/history + ACTMEM semantics
Mentle  = raw material, evidence, retrieval, index, provenance
Garden  = activity/runtime orchestration, bounded ContextView, host integration, console
EvoMap  = capability candidates, proposal review, artifact lifecycle, mailbox, Hub policy
```

Laputa authority is one profile, one directory, and the original seven uppercase Markdown files:

```text
IDENTITY.MD      RELATIONSHIP.MD  REDLINE.MD  USER.MD
DREAM.MD         DARK.MD          WORLD.MD
```

The existing cognitive implementation additionally supports human-only `MISSION.MD` and its versioned Mission slot (ADR-0017). This policy adds no new Persona kind or Frozen Core slot.

`ACTMEM.MD` remains separate from personality authority and backend evidence. The selected VIVY policy uses one persistent, free-form AI-organized Recap per subject, shared across its sessions. LIFE is the first default-enabled pilot; coding is excluded from phase 1.

## Context Discipline

| Lane | Content |
| --- | --- |
| Frozen Core | Existing Mission v2 slot and bounded session-frozen projections of `IDENTITY`, `RELATIONSHIP`, `REDLINE`, `USER`, `DREAM`, and `DARK` |
| Dynamic loading | Baseline empty; selected VIVY policy may admit a bounded Recap through authorized scope/budget policy |
| Tool access | Full `WORLD.MD`, full `ACTMEM.MD`, full authority documents, Mentle evidence, reports, and history |

`WORLD.MD` remains tool-only. ACTMEM stays outside Frozen Core and is admitted only by the selected policy. Commit revision-aware Recap consolidation before retiring raw CONTEXT; retained raw evidence is preserved. A dedicated UI must allow full human inspection/edit/delete; clearing active Recap invalidates projections without cascade deletion. Agent inspect/update/delete tools require deferred discovery and explicit activation; the exact mechanism remains unresolved.

## EvoMap

EvoMap is retained and owns all capability-artifact lifecycle: evolution candidates, proposals, evaluation, versioning, installation permission, mailbox, privacy gate, and Hub publication policy. Laputa does not produce, install, or publish Skills.

## Clean Break

The following are retired and must not appear in new runtime design: JSON Persona descriptors, `.laputa/sections/*.json` authority state, `Commitment`, `Preferences`, `MemoryMD`, `MEMORY.MD`, `memory_md`, JSON Patch, generic Persona governance mutation, automatic WORLD projection, and ACTMEM projection outside the selected policy.

There is no migration, dual read/write, descriptor mapping, or fallback to retired JSON authority data.

## Build and Test

```bash
cd laputa && GOSUMDB=off go test ./...
cd ../mentle && GOSUMDB=off go test ./...
cd ../garden && GOSUMDB=off go test ./...
GOSUMDB=off go test -tags=e2e ./e2e/...
```

## Documentation

- [Selected VIVY Context Policy: ADR-0018](docs/architecture/0018-vivy-unified-context-and-recap-policy.md)
- [Alignment and Evidence Inventory](docs/dev/memory-policy-alignment-2026-10-11.md)
- [Preserved Authority Architecture: ADR-0012](docs/architecture/0012-laputa-markdown-clean-break.md)
- [Implementation Architecture: ADR-0013](docs/architecture/0013-laputa-clean-break-implementation-architecture.md)
- [Mentle Recovery Authority: ADR-0014](docs/architecture/0014-mentle-canonical-authority-and-derived-index-recovery.md)
- [Authority & Recovery GOAL Runbook](docs/bmad/garden-authority-recovery-2026-09/GOAL-EXECUTION-RUNBOOK.md)
- [Documentation Index](docs/README.md)
- [EvoMap Mailbox](docs/architecture/0007-evomap-mailbox.md)
- [EvoMap Hub Transport](docs/architecture/0010-evomap-hub-transport-provider.md)
- [Archived Previous Laputa Contract](docs/archive/2026-08-14-laputa-clean-break/)
