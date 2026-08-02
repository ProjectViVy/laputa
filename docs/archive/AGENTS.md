<!-- Parent: ../AGENTS.md -->

# docs/archive — Historical Design Records (Pre-vNext)

**Generated:** 2026-08-01  
**Status:** Historical archive, read-only, no modifications  
**Purpose:** Preserve pre-redesign decisions as evidence without endorsing them

---

## Purpose

The `archive/` directory preserves historical design documents from before the vNext redesign:

- **Pre-vNext decisions** on Garden, Laputa, Mentle design
- **Superseded architectures** and discarded alternatives
- **Historical record** for tracing how the system evolved
- **Evidence base** for understanding previous assumptions

These materials are preserved exactly as they were, without modification, because:

1. They provide context for how we arrived at current decisions
2. They document what we tried and why we moved on
3. They are source evidence, not implementation guides
4. They should not be edited, deleted, or reinterpreted

---

## Structure

```
archive/
└── 2026-08-01-pre-memoryos-redesign/        # See 2026-08-01-pre-memoryos-redesign/AGENTS.md
    ├── architecture/                        # Archived ADRs
    ├── legacy-archive/                      # Implementation details
    ├── root/                                # Root docs snapshot
    ├── GARDEN-PLAN-2026-07-08.md            # Previous comprehensive plan
    ├── (other pre-vNext design artifacts)
    └── (all materials dated before 2026-08-01)
```

### Subdirectories (Depth 3+)

- **[2026-08-01-pre-memoryos-redesign/AGENTS.md](./2026-08-01-pre-memoryos-redesign/AGENTS.md)** — pre-vNext snapshot
- **[2026-08-01-pre-memoryos-redesign/architecture/AGENTS.md](./2026-08-01-pre-memoryos-redesign/architecture/AGENTS.md)** — archived ADRs
- **[2026-08-01-pre-memoryos-redesign/legacy-archive/AGENTS.md](./2026-08-01-pre-memoryos-redesign/legacy-archive/AGENTS.md)** — implementation details
- **[2026-08-01-pre-memoryos-redesign/root/AGENTS.md](./2026-08-01-pre-memoryos-redesign/root/AGENTS.md)** — root docs snapshot

---

## What's Archived

All design documents, ADRs, wave specifications, and planning materials from before the vNext redesign decision (2026-08-01).

These include decisions on:

- Previous Garden architecture (before current vNext)
- Earlier Laputa governance model (before current section-based design)
- Earlier Mentle retrieval approach (before current KG/hybrid search)
- Previous HTTP API contracts (before v1/v2 split)
- Earlier migration wave planning

---

## How to Use Archive

### Understand Historical Context

1. Check archive when asking "Why did we design it this way?"
2. Cross-reference to current architecture docs to see what changed
3. Use to explain previous decisions or rejected alternatives

### Do NOT Use Archive As

- Implementation guide (it predates current decisions)
- Authority on current behavior (current docs are authoritative)
- Basis for new features (base on current architecture)

---

## Access Policy

- **Read:** Always permitted (understanding history is valuable)
- **Modify:** Never (preserve exactly as-is for historical accuracy)
- **Delete:** Never (archive is permanent)
- **Link:** Only for historical context, not normative documentation

---

## When Something Is Archived

If a design becomes obsolete:

1. Create a dated subdirectory (e.g., `2026-08-01-phase-name/`)
2. Move the old document there without modification
3. Add a note at the top: "Archived [date] — see architecture/ for current design"
4. Never delete or modify archived content

---

## Current Superseded Content

| Document | Archived | Reason | Current Reference |
|----------|----------|--------|-------------------|
| GARDEN-PLAN-2026-07-08.md | 2026-08-01 | Pre-vNext design | architecture/0001-memoryos-vnext-architecture.md |

---

## Conventions

- One dated subdirectory per archive batch
- All original files preserved without modification
- ISO 8601 date format (YYYY-MM-DD)
- Archive notes explain what was superseded and why

---

## MANUAL

When archiving material:

1. Create dated subdirectory
2. Move files without modification
3. Add reference in this AGENTS.md
4. Do NOT edit archived content
5. Link from architecture/ to explain what changed

When consulting archive:

1. Remember it predates current decisions
2. Cross-reference to current docs
3. Do not use as basis for new work

Parent reference: ../AGENTS.md
