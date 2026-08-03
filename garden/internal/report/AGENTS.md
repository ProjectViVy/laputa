<!-- Parent: ../AGENTS.md -->

# garden/internal/report — Report System

**Generated:** 2026-08-03  
**Purpose:** ADR-0005 human-facing report system + ADR-0009 human modules

---

## Purpose

The `report/` package implements the unified report system (ADR-0005): deterministic / LLM-enriched window reports persisted to Garden state SQLite, dual-written into Laputa sections 07/08/09 via `GovernedPublisher`, plus a bounded orientation read for Mentle outage. It also owns the AMBITION / USER SUGGESTIONS human modules (ADR-0009).

---

## Structure

```
report/
├── service.go       # Service, report generation, window, orientation
├── enrich.go        # optional LLM enricher (rhythm.ArtifactGenerator)
├── publisher.go     # GovernedPublisher → section writes (report_system actor)
├── modules.go       # human_modules table, Module CRUD, monthly attachment
└── *_test.go
```

---

## Key Concepts

### Report artifact

`Report` (JSON) carries identity (`cadence, window_start, window_end`), bounded lists (`highlights, completed, decisions, goals, open_loops`, each ≤ 10 × 240 runes), `source_ids/source_hash`, `revision`, `generator`, and `modules` (monthly only). Persisted in the `reports` table with the artifact fields in one JSON `artifact` column; `migrateArtifactColumn` adds the column on old databases.

### Generation

`Generate(ctx, cadence, now)` is source-idempotent: `INSERT OR IGNORE` keyed on `(cadence, window_start, source_hash)`. An empty window never produces a report (`ErrNotFound`). The hourly loop and lazy `latest` miss share this path. Monthly reports attach `modules` names for active entries created inside the window (`moduleNamesInWindow`); daily/weekly never do.

### Human modules (ADR-0009)

`human_modules` table: user-only write path (no agent writes, no GovernedService/audit), `kind ∈ {ambition, suggestion}`, `status ∈ {active, dismissed}` (no hard delete), content ≤ 2000 runes. Non-binding; never projected into ContextView.

---

## HTTP Surface (registered in internal/server)

| Route | Behavior |
|-------|----------|
| `GET /v2/reports?cadence=&limit=` | history, newest first |
| `GET /v2/reports/latest?cadence=` | latest artifact; lazy generation on miss |
| `POST /v2/reports/generate` | synchronous idempotent generation |
| `GET /v2/reports/orientation?budget=` | bounded orientation read with spool-recovery note |
| `GET /v2/reports/modules?kind=&status=` | list modules (`active` default, `all`/`dismissed`) |
| `POST /v2/reports/modules` | create module (201) |
| `PATCH /v2/reports/modules/{id}` | edit content and/or dismiss/reactivate |

---

## Testing

```bash
cd garden
GOSUMDB=off go test -v ./internal/report/...
```

**Behavioral tests:**

- Generation is source-idempotent and revision increments per `(cadence, window_start)`
- Artifact derivation bounds and decision filtering
- Legacy schema migration (pre-artifact rows read back with defaults, `modules == []`)
- Publication idempotency and failure isolation (publisher error leaves SQLite intact)
- Enricher fallback keeps `generator == "deterministic"`
- Module CRUD, validation bounds, `active ⇄ dismissed` state machine
- Monthly-only attachment: daily/weekly carry no modules; dismissed/out-of-window entries excluded
- Orientation truncates to budget and carries the spool-recovery note

---

## Conventions

- All timestamps stored as UTC RFC3339Nano
- Reports are point-in-time snapshots; re-generating an unchanged window returns the saved row
- Module writes are user actions only; nothing in this package mutates authority

---

## MANUAL

Keep report generation non-fatal on publication failure; SQLite is the source of truth for HTTP reads. Modules stay non-binding and out of ContextView.

Parent reference: ../AGENTS.md
