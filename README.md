# LAPUTA

[中文文档](README_CN.md)

Laputa is a specification/protocol for persona and memory governance, with small
implementations adapted to each agent. Garden supplies a reusable governance
workflow and optional management application. The owner's 2026-10-31 direction
is recorded in [ADR-0016](docs/architecture/0016-laputa-protocol-and-native-agent-governance.md);
the new adapters and workflow are not implemented by this documentation update.

## Current Direction

| Part | Responsibility |
| --- | --- |
| Laputa | Persona ownership, read/update and context-delivery semantics; integration boundaries for memory and Recap |
| Host agent | Its authoritative persona files/APIs, working-context engine, execution and native evolution |
| Long-term backend | Canonical memory and retrieval; replaceable without another persona authority |
| Garden | Collect corpus, invoke an agent, apply governed updates, verify effective results and retain outcomes |

Prefer updating the agent's existing authority over creating parallel Laputa
files. Hermes and OpenClaw are intended adapter examples, not verified
integrations. The new Vivy prefers a small Rust persona implementation, retains
full MemPalace for now and owns Recap in its harness. A later backend such as
Mem0 remains a design possibility. Garden need not run for Vivy AutoDream to work.

A unified Web console and Garden management of Vivy's internal Laputa remain
later discussion. Protocol messages, adapters and workflow packaging need
detailed design; no universal Go runtime, file layout or REST deployment is required.

## Existing Go Implementation

The following describes the existing Garden MemoryOS profile, not requirements
for all Laputa consumers. [ADR-0012: Laputa Markdown Clean Break](docs/architecture/0012-laputa-markdown-clean-break.md)
remains its contract, scoped by ADR-0016.

[ADR-0013](docs/architecture/0013-laputa-clean-break-implementation-architecture.md) defines the clean-break implementation order. [ADR-0014](docs/architecture/0014-mentle-canonical-authority-and-derived-index-recovery.md) makes canonical SQLite the sole Mentle memory authority and treats vector/BM25 indexes as rebuildable projections.

```text
Laputa  = seven Markdown personality authorities + Persona review/history + ACTMEM semantics
Mentle  = raw material, evidence, retrieval, index, provenance
Garden  = activity/runtime orchestration, bounded ContextView, host integration, console
EvoMap  = capability candidates, proposal review, artifact lifecycle, mailbox, Hub policy
```

Laputa authority is one profile, one directory, and exactly these seven uppercase Markdown files:

```text
IDENTITY.MD      RELATIONSHIP.MD  REDLINE.MD  USER.MD
DREAM.MD         DARK.MD          WORLD.MD
```

`ACTMEM.MD` is the profile-wide cross-session activity memory. It is not a personality authority and is never default prompt context.

## Context Discipline

| Lane | Content |
| --- | --- |
| Frozen Core | Bounded session-frozen projections of `IDENTITY`, `RELATIONSHIP`, `REDLINE`, `USER`, `DREAM`, and `DARK` |
| Dynamic loading | Empty in v1 |
| Tool access | Full `WORLD.MD`, full `ACTMEM.MD`, full authority documents, Mentle evidence, reports, and history |

Neither `WORLD.MD` nor `ACTMEM.MD` may enter Fast Recall, Deep Recall, bootstrap, or automatic ContextView assembly.

## EvoMap

EvoMap is retained and owns all capability-artifact lifecycle: evolution candidates, proposals, evaluation, versioning, installation permission, mailbox, privacy gate, and Hub publication policy. Laputa does not produce, install, or publish Skills.

## Clean Break

The following are retired and must not appear in new runtime design: JSON Persona descriptors, `.laputa/sections/*.json` authority state, `Commitment`, `Preferences`, `MemoryMD`, `MEMORY.MD`, `memory_md`, JSON Patch, generic Persona governance mutation, automatic WORLD projection, and automatic ACTMEM projection.

There is no migration, dual read/write, descriptor mapping, or fallback to retired JSON authority data.

## Build and Test

```bash
cd laputa && GOSUMDB=off go test ./...
cd ../mentle && GOSUMDB=off go test ./...
cd ../garden && GOSUMDB=off go test ./...
GOSUMDB=off go test -tags=e2e ./e2e/...
```

## Documentation

- [Current Direction: ADR-0016](docs/architecture/0016-laputa-protocol-and-native-agent-governance.md)
- [Existing Go Profile: ADR-0012](docs/architecture/0012-laputa-markdown-clean-break.md)
- [Implementation Architecture: ADR-0013](docs/architecture/0013-laputa-clean-break-implementation-architecture.md)
- [Mentle Recovery Authority: ADR-0014](docs/architecture/0014-mentle-canonical-authority-and-derived-index-recovery.md)
- [Authority & Recovery GOAL Runbook](docs/bmad/garden-authority-recovery-2026-09/GOAL-EXECUTION-RUNBOOK.md)
- [Documentation Index](docs/README.md)
- [EvoMap Mailbox](docs/architecture/0007-evomap-mailbox.md)
- [EvoMap Hub Transport](docs/architecture/0010-evomap-hub-transport-provider.md)
- [Archived Previous Laputa Contract](docs/archive/2026-08-14-laputa-clean-break/)
