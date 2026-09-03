# ADR-0008: Legacy Compatibility Removal

**Status:** accepted  
**Date:** 2026-08-03  
**Supersedes:** ADR-0001 §2.5 (current compatibility facts), §10.5 (compatibility routes), §16 checklist item "legacy route compatibility period defined"; ADR-0004 §4 (legacy 14-section compatibility mapping and 410 write-blocking)  
**Depends on:** ADR-0002 (cognitive partition), ADR-0004 (cognitive files), ADR-0005 (report system), ADR-0006 (semantic ingestion), ADR-0007 (EvoMap mailbox)

---

## 1. Context

The vNext migration retained a legacy compatibility surface: the v1 HTTP routes (`/v1/memories`, `/v1/context/resolve`, `/v1/context/bootstrap`, `/v1/sessions`, `/v1/ingestions/{id}`, `/v1/pipelines*`, `/v1/reports/latest`), the CRUD translator chain (`garden/internal/crud` + `garden/internal/router` + `mentle/facade/crud.go` + `legacy_key`/`backfillCanonical`), the legacy 14-section governance registry (with `06-history_md`, `10-journal_reflective`, `11-proposal_inbox`, `12-changelog`, `13-report_indexes`, `14-aaak_summaries`), and the `Compat` write-blocking mechanism (HTTP 410 Gone).

The compatibility layer was staged for removal once hosts migrated. **No host uses the legacy Laputa runtime anymore.** The decision is to delete the entire legacy compatibility surface rather than preserve it.

## 2. Decisions

### 2.1 Registry convergence (8 sections)

`laputa/governance` now exposes exactly the target-model sections:

| Section | Role |
|---|---|
| 01-identity, 02-relationship, 03-commitment, 04-preferences | Frozen Core (Frozen: true) |
| 05-memory_md | STM checkpoint |
| 07-daily, 08-weekly, 09-monthly | Human-facing reports (report_system authority) |

Removed constants, registry entries, default data, and repo-committed JSON files for `06-history_md`, `10-journal_reflective`, `11-proposal_inbox`, `12-changelog`, `13-report_indexes`, `14-aaak_summaries`. Their responsibilities are already carried by target-model subsystems: reports (SQLite, ADR-0005) for 10/11, `FileAuditLog` for 12, and ADR-0002/ADR-0006 for 06/13/14. The `tbd` authority and `compat` status concepts are removed with them.

Writes to removed section names now fail as **unknown section** (HTTP 400), not 410.

### 2.2 Compat mechanism removal

- `SectionInfo.Compat` field, `ErrCompatReadOnly`, and the 410 `compat_read_only` mapping are deleted.
- `06/13/14` write-blocking tests are replaced by unknown-section behavior tests.

### 2.3 v1 routes deleted; target functionality promoted to v2

All v1 routes are removed. Functionality still part of the target model is promoted:

| Old v1 | New v2 |
|---|---|
| POST/GET/GET-list/PATCH/DELETE `/v1/memories` (canonical branches) | `/v2/memories`, `/v2/memories/{id}` |
| POST `/v1/sessions` | `POST /v2/ingest/sessions` |
| GET `/v1/ingestions/{id}` | `GET /v2/ingestions/{id}` |
| POST `/v1/context/bootstrap` | `POST /v2/recall/bootstrap` |
| GET `/v1/pipelines*` | `GET /v2/pipelines*` |
| GET `/v1/reports/latest` | `GET /v2/reports/latest` (already registered) |
| POST `/v1/context/resolve` | removed (no target equivalent; /v2/recall/* supersedes) |

The legacy `{key, value, meta}` payload shape is gone; `/v2/memories` accepts only the canonical `{content, kind, scope, ...}` contract.

### 2.4 Translator chain deleted

- `garden/internal/crud` and `garden/internal/router` packages deleted.
- `mentle/facade/crud.go` (legacy `Write`/`Read`/`List`/`Forget`) deleted; `legacy_key` column, `backfillCanonical`, and the legacy `insertMemory` parameter removed from `canonical.go`.
- `Server.Handler *crud.Handler` replaced by `Server.Facade *facade.Service`; memory handlers call the facade directly.
- Pre-existing databases keep an orphaned `legacy_key` column harmlessly; no data migration is performed.

### 2.5 RAG convergence

`garden/internal/rag` keeps the planner surface only (`Planner`, `RulePlanner`, `OpenAIPlanner`, `FallbackPlanner`, `types.go`) required by Deep Recall. The `Resolver` implementation (`service.go`) and governance policy resolver (`policy.go`), which referenced `06-history_md`, are deleted.

### 2.6 Wakeup history writes removed

`laputa/governance/wakeup` no longer writes history: `SyncTurn` and `OnSessionEnd` (which appended to `06-history_md`, bypassing GovernedService) are deleted, together with their CLI actions (`sync-turn`, `session-end`) and the scheduler daemon's session-end hook.

## 3. Data notes

- Repo-committed section files for removed sections are deleted.
- User-machine files under `~/.laputa/sections/*.json` for removed sections are **not** actively deleted by the code; `Initialize` no longer creates them and nothing reads or writes them. They remain inert historical data.

## 4. Non-goals

- No host-adapter work (Wave 7 remains deferred).
- No Obsidian adapter runtime (ADR-0006 remains design-only).
- No rewrite of archived docs (`docs/archive/`) or historical ADR text.

## 5. Consequences

- API surface is v2-only; legacy clients must migrate or are unsupported.
- `garden/console` updated to `/v2/pipelines`; the legacy registry node is removed from the governance map.
- e2e suite rewritten against v2 endpoints and no longer exercises legacy CRUD.
- Test matrices updated: laputa registry tests assert 8 sections with `stable` status only.
