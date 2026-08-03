<!-- Parent: ../AGENTS.md -->

# laputa/.laputa — Governance Sections Storage

**Generated:** 2026-08-01  
**Purpose:** File-based JSON storage for 14 governance sections

---

## Purpose

The `.laputa/sections/` directory stores the complete governance state for Laputa as JSON files:

- **14 governance sections** — identity, relationships, commitments, preferences, memory, history, daily/weekly/monthly reports, journal, proposals, changelog, indexes, summaries
- **Single source of truth** for agent authority, policy, and audit
- **File-based durability** — atomic writes with cross-process safety
- **Zero daemon** — all operations are explicit and audited

---

## Structure

```
.laputa/
└── sections/
    ├── 01-identity.json           # Agent identity and principal definitions
    ├── 02-relationship.json       # Relationships and resonance
    ├── 03-commitment.json         # Commitments and red lines
    ├── 04-preferences.json        # Agent preferences and settings
    ├── 05-memory_md.json          # Long-term memory summaries
    ├── 07-daily.json              # Daily rhythm reports
    ├── 08-weekly.json             # Weekly rhythm reports
    └── 09-monthly.json            # Monthly rhythm reports
```

### Subdirectories (Depth 2)

- **[sections/AGENTS.md](./sections/AGENTS.md)** — detailed section schema and governance semantics

---

## Key Concepts

### Governance Sections (8 Total)

The legacy 14-section registry was reduced to the target model by ADR-0008 (06-history_md, 10-journal_reflective, 11-proposal_inbox, 12-changelog, 13-report_indexes, 14-aaak_summaries removed).

| # | Section | Purpose | Mutability |
|---|---------|---------|-----------|
| 01 | identity | Agent role, capabilities, constraints, SOP | explicit |
| 02 | relationship | Relationships with other agents, resonance signals | explicit |
| 03 | commitment | Commitments and red-line constraints | explicit |
| 04 | preferences | Preference settings, mode selections | explicit |
| 05 | memory_md | STM summary and highlights | explicit |
| 07 | daily | Daily rhythm reports (generated) | append-only |
| 08 | weekly | Weekly rhythm reports (generated) | append-only |
| 09 | monthly | Monthly rhythm reports (generated) | append-only |

### No Silent Durable Mutation

Every change to governance state is:
- **Explicit:** caller must invoke `SetSection()` with new data
- **Audited:** change is logged to `FileAuditLog` (ADR-0002; the legacy section-12 changelog is removed) with timestamp and authority
- **Reversible:** prior state is retained in the audit log for rollback

### File Format

Each section is a JSON object:

```json
{
  "_meta": {
    "updated_at": "2026-08-01T12:34:56Z",
    "version": "1.0"
  },
  "...section-specific fields..."
}
```

All sections include `_meta` with `updated_at` (RFC3339) and `version` (immutable schema version).

---

## Build & Test

No build artifacts. Storage layer is runtime-only.

### Verify Sections Exist

```bash
ls -la .laputa/sections/
```

Expected: 14 JSON files.

### Test Section Access

```bash
cd laputa
go test ./governance/... -v
```

---

## Schema Evolution

When changing a section schema:

1. Increment `_meta.version` (e.g., 1.0 → 1.1)
2. Add migration logic to `laputa.Engine.GetSection()`
3. Write test for migration
4. Commit with migration reference

---

## Architecture Boundaries

**Ownership rule (immutable):**

```text
Laputa owns all 14 sections.
Mentle may read sections (for memory scoping).
Garden may read sections (for governance projection).
Evolver may propose changes (write to proposal_inbox).
No external process may write sections directly.
```

---

## Conventions

- **Atomic writes:** file operations use temp + rename for safety
- **Locking:** file-based flock when concurrent access is possible
- **Backup:** prior state retained in changelog for audit
- **Schema:** JSON schema version in `_meta` for migrations
- **Timestamps:** all dates in RFC3339 (UTC)

---

## MANUAL

This document is maintained by the oh-my-claudecode writer agent.

When updating:

1. Add new section files to sections/ subdirectory
2. Update section table above with # and Purpose
3. Update sections/AGENTS.md with detailed schema
4. Do not delete sections; archive superseded schemas to docs/archive/

Parent reference: `../AGENTS.md`
