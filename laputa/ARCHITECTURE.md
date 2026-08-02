# Laputa Governance Framework

> Pure file-based governance. Zero subprocesses. Mempalace is completely separate.

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                        Laputa Engine                         │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐  │
│  │  Section    │  │  Section    │  │  Section Registry   │  │
│  │  Store      │  │  Snapshot   │  │  (14 sections)      │  │
│  │  Interface  │  │  API        │  │                     │  │
│  └──────┬──────┘  └─────────────┘  └─────────────────────┘  │
└─────────┼───────────────────────────────────────────────────┘
          │
          ▼
┌─────────────────────────────────────────────────────────────┐
│                      FileStore (default)                     │
│              `.laputa/sections/*.json`                       │
└─────────────────────────────────────────────────────────────┘
```

## Target cognitive partition notice

The table below describes the **current Go compatibility registry**, not the accepted Garden MemoryOS cognitive architecture. [Garden ADR-0002](../docs/architecture/0002-laputa-cognitive-partition-decision.md) removes target LTM/`LONGMEM.MD`, `13-report_indexes` and `14-aaak_summaries`; converts #10/#11 into monthly human report modules; moves #12 to infrastructure logging; and introduces `MEMRULES.MD` plus `WORLD.MD`. Do not add features using the legacy meanings below before the separately approved physical migration plan.

## 14 Governance Sections

Based on `laputa-py/baseline/LAPUTA.md` v0.0.6 final.

| # | Section | Write Authority | Schema Owner | Status |
|---|---------|-----------------|--------------|--------|
| 1 | `identity` | agent_self | laputa | stable |
| 2 | `relationship` | agent_self | laputa | stable |
| 3 | `commitment` | user_only | laputa | stable |
| 4 | `preferences` | agent_self | laputa | stable |
| 5 | `memory_md` | agent_self | laputa | stable |
| 6 | `history_md` | agent_self | laputa | stable |
| 7 | `daily` | report_system | report_system | stable |
| 8 | `weekly` | report_system | report_system | stable |
| 9 | `monthly` | report_system | report_system | stable |
| 10 | `journal_reflective` | tbd | tbd | tbd |
| 11 | `proposal_inbox` | tbd | tbd | tbd |
| 12 | `changelog` | tbd | tbd | tbd |
| 13 | `report_indexes` | tbd | tbd | tbd |
| 14 | `aaak_summaries` | tbd | tbd | tbd |

## Core Rules

1. **Laputa writes, Mentle reads** — authority is one-way.
2. **Laputa persists, AutoDream triggers** — single source of truth, scheduling is external.
3. **Laputa accepts all writes, selfinprove is the reviewer** — unified write consumption.

## File Layout

```
.laputa/
├── sections/
│   ├── 01-identity.json
│   ├── 02-relationship.json
│   ├── ...
│   └── 14-aaak_summaries.json
```

## API

```go
engine.Initialize(ctx)              // create all 14 sections
engine.GetSection(ctx, section)     // read one section
engine.SetSection(ctx, section, data)   // replace section
engine.UpdateSection(ctx, section, path, value)  // patch path
engine.DeleteSectionPath(ctx, section, path)     // delete path
engine.ListSections(ctx)            // list all sections
engine.Snapshot(ctx)                // full state with metadata
```

## Rhythm Layer

```
Laputa Engine
    ↓ snapshot
Rhythm Engine (internal/rhythm)
    ↓ prompt
Eino ChatModel
    ↓ report JSON
Laputa Section (daily / weekly / monthly)
```

The rhythm engine is triggered externally (CLI or cronjob), not by a daemon.

## Non-Goals

- Not a DB / vector store / graph DB.
- Not an agent runtime.
- Not a scheduler / cron (rhythm is triggered externally).
- Not auth / identity provider.
- Not a full AutoDream state machine.
- Not Mentle integration.
