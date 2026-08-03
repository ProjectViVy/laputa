# ADR-0009: AMBITION / USER SUGGESTIONS Modules

**Status:** accepted  
**Date:** 2026-08-03  
**Supersedes:** ADR-0005 §8 carrier sentence ("Carried by the sections ADR-0002 repurposes for them (`10-journal_reflective` → AMBITION, `11-proposal_inbox` → USER SUGGESTIONS); physical rename remains deferred to the migration-execution PR, as with Gate A")  
**Depends on:** ADR-0002 §3.4 (monthly human modules), ADR-0005 (report system), ADR-0008 (deletion of sections 10/11)

---

## 1. Context

ADR-0005 §8 specified `AMBITION` and `USER SUGGESTIONS` as optional **monthly-only** human modules, non-binding, with a user-only write path. Their originally intended carriers were the legacy sections `10-journal_reflective` (→ AMBITION) and `11-proposal_inbox` (→ USER SUGGESTIONS).

ADR-0008 (Gate E) then deleted both sections from the governance registry: no host uses the legacy Laputa runtime, and the registry converged to 01-05 + 07-09. The §8 carrier sentence is therefore obsolete. This ADR relocates the modules into the report subsystem (Garden state SQLite), preserving every ADR-0005 §8 guarantee and the reserved `modules` array on monthly reports.

## 2. Decisions

### 2.1 Storage: report-system SQLite

Modules live in a `human_modules` table in the Garden state database, owned by `garden/internal/report` (same connection as the `reports` table). No Laputa section is created, renamed, or written.

```sql
CREATE TABLE IF NOT EXISTS human_modules(
  id TEXT PRIMARY KEY,
  kind TEXT NOT NULL CHECK(kind IN ('ambition','suggestion')),
  content TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'active' CHECK(status IN ('active','dismissed')),
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
```

### 2.2 Non-binding human content (unchanged from ADR-0005 §8)

- Ambitions and suggestions never become commitments, never alter Frozen Core, and are never projected into ContextView by default.
- **User-only write path:** creation, edit, and dismissal are user actions. There is no agent write path, and module writes do **not** go through `GovernedService`/audit — they are human-facing report content, not authority mutations.
- No hard delete; dismissal (`active → dismissed`) is the terminal user action and is reversible (`dismissed → active`).

### 2.3 Bounds

- `content`: non-empty, ≤ 2000 runes.
- `kind`: `ambition` | `suggestion` only; unknown kinds are rejected.

### 2.4 Monthly report attachment

- A monthly report (`cadence == "monthly"`) MAY attach `modules: ["AMBITION", "USER SUGGESTIONS"]` — the names of the modules that have **active** entries created inside that month's window (`created_at ∈ [window_start, window_end)`).
- The array identifies included modules only; entry content is read through the module endpoints, keeping report artifacts compact.
- Daily and weekly reports never carry `modules`.
- The `modules` field is additive to the report artifact JSON; pre-existing report rows unmarshal to an empty array (no schema migration).

## 3. HTTP Contract

| Route | Behavior |
|-------|----------|
| `GET /v2/reports/modules?kind=&status=` | list; `kind` required (`ambition`\|`suggestion`), `status` optional (`active` default \| `dismissed` \| `all`) |
| `POST /v2/reports/modules` | create `{kind, content}` → 201 |
| `PATCH /v2/reports/modules/{id}` | edit `{content}` and/or `{status}`; 404 unknown id |

Validation failures (bad kind, empty or over-long content, bad status) → 400. Report service unavailable → 503.

## 4. Non-goals

- No agent write path, no LLM enrichment of module content.
- No ContextView projection, no WORLD claims, no Frozen Core interaction.
- No Laputa section writes of any kind.
- No EvoMap / mailbox integration.

## 5. Test Matrix

| # | Test | Assertion |
|---|------|-----------|
| 1 | Module CRUD | create/list/update round-trip; 404 on unknown id |
| 2 | Validation bounds | bad kind, empty/over-long content → error; content ≤ 2000 runes |
| 3 | State machine | `active ⇄ dismissed`; list filters by status |
| 4 | Monthly-only attachment | monthly report carries `modules` for in-window active entries; daily/weekly never do; dismissed and out-of-window entries excluded |
| 5 | Artifact compatibility | pre-existing report rows read back with `modules == []` |
| 6 | Endpoints | 200/201/400/404/503 paths |
| 7 | Context plane regression | module content appears in no `/v2/recall/*` projection |

## 6. Consequences

- ADR-0005 §8's deferred "later batch" is delivered without reviving sections 10/11 (ADR-0008 stays intact).
- Report subsystem gains one table and three endpoints; monthly reports can reference the human modules.
- Users manage ambitions/suggestions through the console Modules page; no agent can write them.
