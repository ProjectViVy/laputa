# ADR-0005: Report System Design

**Status:** accepted  
**Date:** 2026-08-03  
**Supersedes:** none  
**Depends on:** ADR-0001 §7.4 (human reports and emergency orientation), ADR-0002 §3.4 (07–11 reclassification), ADR-0004 (13-report_indexes compat)

---

## 1. Context

Before this ADR, two report implementations existed and were not connected:

- **Garden `internal/report`** (live, served over HTTP): deterministic, LLM-free generation into a SQLite `reports` table, hourly loop plus on-demand lazy generation, single endpoint `GET /v1/reports/latest`. Source-idempotent via `source_hash` in the primary key.
- **Laputa `governance/rhythm`** (not wired into garden): LLM-based (eino) generation writing into sections `07-daily`/`08-weekly`/`09-monthly`, reachable only from the deprecated `laputa` CLI.

ADR-0001 §7.4 defines the target contract: reports are human-facing continuity artifacts with an explicit artifact shape, canonical copies under 07/08/09, optional LLM generation, and a bounded orientation use during Mentle outage. None of that was implemented.

This ADR specifies the unified report system: catalog, artifact contract, dual-write storage, generator policy, orientation read, and the retirement of `13-report_indexes`.

## 2. Report Catalog

| Cadence | Window (UTC) | Canonical section |
|---------|--------------|-------------------|
| daily | midnight → midnight | `07-daily` |
| weekly | Monday 00:00 → +7d | `08-weekly` |
| monthly | 1st 00:00 → +1 month | `09-monthly` |

- Windows are computed in UTC and are stable functions of the clock (Garden `report.window()` semantics are retained).
- Generation is triggered (a) at startup, (b) hourly, (c) on user request via `POST /v2/reports/generate`, and (d) lazily on a `latest` cache miss. All paths are idempotent per `(cadence, window_start, source_hash)`.
- An empty window produces no report (`ErrNotFound`), never an empty artifact.

## 3. Artifact Contract

Each report is a compact human-readable snapshot per ADR-0001 §7.4:

| Field | Type | Bound |
|-------|------|-------|
| cadence, window_start, window_end | identity | — |
| scope | string | `"mentle_active"` when generated from Mentle listing |
| title, summary | string | summary ≤ 4000 chars |
| highlights | []string | ≤ 10 items × 240 runes |
| completed | []string | ≤ 10 × 240 runes |
| decisions | []string | ≤ 10 × 240 runes |
| goals | []string | ≤ 10 × 240 runes |
| open_loops | []string | ≤ 10 × 240 runes |
| open_questions | []string | retained for v1 compatibility |
| source_ids / source_refs | []string | Mentle card IDs backing the report |
| source_hash | string | `sha256:` of joined source IDs |
| revision | int | count of prior reports for the same `(cadence, window_start)` + 1 |
| generator | string | `"deterministic"` \| `"llm"` |
| generated_at | RFC3339 | UTC |

Deterministic derivation (no LLM): `decisions` = window memories with `kind == "decision"`; `completed` = highlights; `goals` and `open_loops` are empty unless an LLM enricher fills them.

Reports answer "这一时间窗口做了什么、结果怎样、现在卡在哪里？" They never contain the only copy of raw activity, tool output, or source evidence; Mentle remains the material universe.

## 4. Storage: Dual Write

| Store | Role |
|-------|------|
| Garden SQLite `reports` table | Generator cache and query index; source of truth for HTTP reads; survives Mentle outage |
| Sections 07/08/09 | Canonical durable human-facing copy |

Mechanics:

1. Generation writes SQLite first (`INSERT OR IGNORE`, PK `(cadence, window_start, source_hash)`).
2. Only when a new row is actually inserted, Garden publishes the report into the matching section via `GovernedService.Mutate` with `actor = report_system` (authorized for 07/08/09 by `AuthorityReport`).
3. Section entries carry `source_hash`; a section entry with the same hash is never duplicated (publication is idempotent).
4. The section `reports` array is capped at 200 entries; the oldest entries are dropped first.
5. Publication failure is non-fatal: it is logged, SQLite remains readable, and report generation never blocks ingestion, STM checkpointing, transient-spool recovery, or WORLD/MEMRULES governance (ADR-0001 §7.4).

Known trade-off: if publication fails after the SQLite insert, that report is not retried automatically (the next generation with the same source hash is a no-op). A force re-publish path may be added later; it is out of scope here.

Report metadata (cadence catalog, revisions, last-generated) lives inside the report subsystem: the SQLite `reports` table and the section `_meta` maintained by the laputa engine. `13-report_indexes` is not revived (see §8).

## 5. Generators

- **Deterministic default**: bounded template over the window's active Mentle memories. No LLM, no network. Always available when Mentle listing is available.
- **Optional LLM upgrade**: when `GARDEN_RAG_BASE_URL`, `GARDEN_RAG_API_KEY`, and `GARDEN_RAG_MODEL` are all set, an enricher backed by `laputa/governance/rhythm`'s OpenAI-compatible generator fills `goals`, `completed`, `decisions`, and `open_loops` before the SQLite insert.
- The enricher runs under a 30-second timeout. Any error, timeout, or unparseable response falls back to the deterministic artifact; `generator` stays `"deterministic"`. On success the bounds in §3 are re-enforced and `generator` is set to `"llm"`.
- Enrichment runs inside the generation path (hourly loop goroutine or explicit generate request); it never blocks recall, ingest, or admin endpoints.

## 6. HTTP Contract

| Route | Behavior |
|-------|----------|
| `GET /v1/reports/latest?cadence=` | unchanged legacy contract (retained) |
| `GET /v2/reports?cadence=&limit=` | history, newest first; default limit 20, max 100 |
| `GET /v2/reports/latest?cadence=` | latest with full artifact fields; lazy generation on miss |
| `POST /v2/reports/generate` | user-requested synchronous generation, idempotent |
| `GET /v2/reports/orientation?budget=` | bounded orientation read (§7) |

All endpoints return 503 when the report service is unavailable; the orientation endpoint degrades to an empty orientation with a warning instead of an error when no report exists.

## 7. Mentle-Outage Orientation Read

When Mentle is unavailable, the latest daily report provides a bounded orientation read: `GET /v2/reports/orientation?budget=` (default 2000, max 8000 characters) assembled from the latest daily report's summary and artifact.

Constraints (ADR-0001 §7.4 bootstrap order):

```text
Laputa Core Prompt
  -> MEMORY.MD checkpoint / working_set
  -> latest matching human report (only if needed for scope orientation)
  -> bounded Laputa transient read for missing recent detail
```

- The orientation read is served from local SQLite; it works while Mentle is down.
- It is never automatically injected into every session and is not default context.
- It can never stand in for the mandatory transient-spool drain: after Mentle recovery, raw data is still rehydrated by `event_id + content_hash`, and reports are never replayed as source material. Every orientation response carries an explicit `note` field stating this.

## 8. AMBITION / USER SUGGESTIONS (Specification Only)

Per ADR-0002 §3.4, `AMBITION` and `USER SUGGESTIONS` are optional **monthly-only** human modules:

- Carried by the sections ADR-0002 repurposes for them (`10-journal_reflective` → AMBITION, `11-proposal_inbox` → USER SUGGESTIONS); physical rename remains deferred to the migration-execution PR, as with Gate A.
- Content is **non-binding**: ambitions and suggestions never become commitments, never alter Frozen Core, and are never projected into ContextView by default.
- User controls: creation, edit, and dismissal are user actions; no agent write path.
- Implementation is deferred; the report artifact reserves no required dependency on them. A future monthly report MAY attach a `modules` array identifying included optional modules.

## 9. Retirement of 13-report_indexes

`13-report_indexes` is write-blocked (`Compat: true`, HTTP 410) since ADR-0004. This ADR finalizes its target status: registry status moves from `tbd` to `compat`. Report metadata lives in the report subsystem (§4). No new feature may target section 13.

## 10. Test Matrix

| # | Test | Assertion |
|---|------|-----------|
| 1 | Artifact derivation | deterministic artifact fields populated within bounds; decisions filtered by kind |
| 2 | Revision | revision increments per `(cadence, window_start)` when source set changes |
| 3 | Legacy schema migration | opening a pre-artifact database upgrades without data loss |
| 4 | Publication actor | section write uses `report_system` actor via GovernedService |
| 5 | Publication idempotency | duplicate `source_hash` is not appended twice |
| 6 | Publication failure isolation | publisher error leaves Generate successful and SQLite intact |
| 7 | Enricher fallback | LLM error/timeout keeps deterministic artifact and `generator == "deterministic"` |
| 8 | Section cap | reports array capped at 200 entries, oldest dropped |
| 9 | v2 endpoints | list/latest/generate 200/400/503 paths |
| 10 | Orientation bound | orientation truncates to budget and carries the spool-recovery note |
| 11 | Reports never authority | reports appear in no ContextView projection (Context Plane, ADR-0004 §6 test 6) |

## 11. Consequences

**Positive:**
- One unified report system: deterministic and LLM paths share one artifact contract and one storage pipeline.
- Canonical human-readable copies survive Mentle outage (sections 07/08/09 + local SQLite).
- Orientation during outage is bounded and explicitly non-substitutive for spool recovery.
- `13-report_indexes` is fully retired from the target model.

**Negative:**
- Dual write introduces a second persistence path; bounded by idempotent publication and non-fatal failure handling.
- The section reports cap (200) means very old canonical copies roll off; SQLite history is unbounded.

**Neutral:**
- AMBITION / USER SUGGESTIONS remain specification-only until a later batch.
- The deprecated `laputa` CLI rhythm wiring stays untouched; its generator library is reused, its process model is not.
