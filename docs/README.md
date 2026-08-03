# Garden Documentation

> **Current architecture:** [MemoryOS vNext Architecture Plan](./architecture/0001-memoryos-vnext-architecture.md)  
> **Status:** proposed - implementation has not started

This directory is the canonical entry point for the next Garden-Laputa MemoryOS transformation.

## Active Documents

| Document | Purpose | Status |
|---|---|---|
| [Architecture Plan](./architecture/0001-memoryos-vnext-architecture.md) | Target architecture, migration sequence, ownership, interfaces, verification gates, and delivery roadmap | proposed |
| [ADR-0002: Laputa Cognitive Partition](./architecture/0002-laputa-cognitive-partition-decision.md) | Accepted partition of Frozen Core, STM, `MEMRULES.MD`, `WORLD.MD`, human-facing reports, removed LTM and deferred migration constraints | accepted |
| [ADR-0003: Operations Console Design](./architecture/0003-operations-console-design.md) | Local-first MemoryOS 运营台: admin layout, layered governance graph, recall trace, materials/evidence, architecture library; workbench-first, MVP-0 read-only first | accepted |
| [ADR-0004: Cognitive Files Migration](./architecture/0004-cognitive-files-migration.md) | Physical `MEMRULES.MD`/`WORLD.MD` representations, validation, no-data-loss migration policy, host projections | accepted |
| [ADR-0005: Report System Design](./architecture/0005-report-system-design.md) | Human-facing report artifact contract, monthly AMBITION / USER SUGGESTIONS modules, orientation read, report metadata placement | accepted |
| [ADR-0006: Semantic Ingestion and Obsidian Source Adapter](./architecture/0006-semantic-ingestion-and-obsidian-adapter.md) | Raw-first `source_artifact`/`semantic_unit` kinds, provenance-preserving semantic units, `SourceAdapter` contract, bounded real-offset evidence read, no second summary authority | accepted |
| [ADR-0007: EvoMap Mailbox](./architecture/0007-evomap-mailbox.md) | Inbox/outbox state machines, evidence refs, privacy gate, retry→dead-letter, Hub disabled by default, `mailbox_items` SQLite persistence, audit of state changes | accepted |
| [ADR-0008: Legacy Compatibility Removal](./architecture/0008-legacy-compatibility-removal.md) | Deletes v1 routes, CRUD translator chain, 06/10/11/12/13/14 registry sections, and 410 compat mechanism; promotes target functionality to v2 | accepted |
| [ADR-0009: AMBITION / USER SUGGESTIONS Modules](./architecture/0009-ambition-user-suggestions-modules.md) | Monthly-only human modules relocated into report-system SQLite after sections 10/11 were deleted; non-binding, user-only write path, monthly report `modules` array | accepted |

## Archive

The pre-MemoryOS redesign documents are preserved without modification at:

[docs/archive/2026-08-01-pre-memoryos-redesign](./archive/2026-08-01-pre-memoryos-redesign/)

They remain source evidence and historical decisions. They are not the implementation contract for the vNext work because they predate the current decisions on governed MemoryOS, progressive recall, lifecycle semantics, and external EvoMap/Evolver integration.
