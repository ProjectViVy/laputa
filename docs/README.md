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

## Archive

The pre-MemoryOS redesign documents are preserved without modification at:

[docs/archive/2026-08-01-pre-memoryos-redesign](./archive/2026-08-01-pre-memoryos-redesign/)

They remain source evidence and historical decisions. They are not the implementation contract for the vNext work because they predate the current decisions on governed MemoryOS, progressive recall, lifecycle semantics, and external EvoMap/Evolver integration.
