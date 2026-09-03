# TODOLIST — Garden MemoryOS

> **Last updated:** 2026-08-04  
> **Current focus:** Gate I complete — console mailbox UI (`/mailbox` page: inbox review approve/reject + outbox delivery state, pure frontend per user scope; mailbox routes added to root AGENTS.md). Next = candidate batches: Obsidian adapter runtime (ADR-0006 §4), publish EvolutionEvent third element, semantic privacy review (ADR-0007), memories/governance/cognitive-world console UI, sidecar Evolver, host adapters (Wave 7).  
> **Architecture source:** [0001 MemoryOS vNext Architecture](docs/architecture/0001-memoryos-vnext-architecture.md) + [ADR-0002 Cognitive Partition](docs/architecture/0002-laputa-cognitive-partition-decision.md) + [ADR-0004 Cognitive Files Migration](docs/architecture/0004-cognitive-files-migration.md) + [ADR-0005 Report System Design](docs/architecture/0005-report-system-design.md) + [ADR-0006 Semantic Ingestion](docs/architecture/0006-semantic-ingestion-and-obsidian-adapter.md) + [ADR-0007 EvoMap Mailbox](docs/architecture/0007-evomap-mailbox.md) + [ADR-0008 Legacy Compatibility Removal](docs/architecture/0008-legacy-compatibility-removal.md) + [ADR-0009 AMBITION / USER SUGGESTIONS Modules](docs/architecture/0009-ambition-user-suggestions-modules.md) + [ADR-0010 EvoMap Hub Transport Provider](docs/architecture/0010-evomap-hub-transport-provider.md)  
> This file tracks execution status only. It must not restate design contracts.

**Legend:** ✅ done · 🚧 active · ⬜ planned · ⛔ blocked  
**Modules:** `[L]` Laputa · `[M]` Mentle · `[G]` Garden · `[R]` Report System

---

## 🔧 Now — documentation and migration boundary

- [x] `[G]` Commit/record the existing documentation transition only after reviewing all pre-existing dirty worktrees. (Gate E commit `3c824e8`; pre-existing `tsconfig.tsbuildinfo`/`.agents`/`.omc` reviewed and excluded)
- [x] Record a current baseline-test snapshot for Laputa governance, Mentle facade, Garden internal and Garden E2E suites. (220/220 green at `3c824e8` — see `docs/baseline-test-snapshot-2026-08-03.md`)
- [x] Add this `TODOLIST.MD` to version control and retain links from root project guidance.
- [x] Produce a **separate, approved physical migration plan** before changing the legacy 14-section Go registry or persisted files (done: ADR-0008, executed in Gate E).

**Guard:** legacy sections `06-history_md`/LTM, `10-journal_reflective`, `11-proposal_inbox`, `12-changelog`, `13-report_indexes`, and `14-aaak_summaries` are deleted from the registry (ADR-0008). They must receive no new target-architecture features; writes to those names fail as unknown section.

---

## ✅ Architecture decision: Laputa Cognitive Partition (2026-08-02)

- [x] Deleted the target-architecture LTM / `LONGMEM.MD` layer; Mentle remains the only material/evidence universe.
- [x] Defined `MEMRULES.MD` as human-maintained cognitive governance, enforced by Garden and excluded from ContextView.
- [x] Defined `WORLD.MD` as user-editable, AutoDream-participating, bounded actionable world understanding; never default-injected.
- [x] Reclassified `07–11` as the human-facing report system: daily, weekly, monthly, AMBITION and USER SUGGESTIONS.
- [x] Moved change/audit history out of section content and deferred an independent EvoMap mailbox design.
- [x] Deleted `13-report_indexes` and `14-aaak_summaries` as target Laputa concepts; retained AAAK only as a future Mentle semantic-unit ingestion principle.

Reference: [ADR-0002](docs/architecture/0002-laputa-cognitive-partition-decision.md).

---

## ✅ Wave 0 — Document & baseline freeze

Archived pre-vNext docs and established active documentation navigation. The active architecture is now reconciled with ADR-0002; no production Go runtime migration was performed by this decision.

---

## ✅ Waves 1–6 — Core implementation (post-ADR-0002 aligned)

- [x] `[M]` **Wave 1:** Card/Evidence API — `SearchCards` (no full content), `ReadEvidence` (budgeted), lifecycle filtering.
- [x] `[G]` **Wave 2:** Fast Recall Core — deterministic recall, governance projection, rank/filter/dedupe, `/v2/recall/fast`.
- [x] `[G]` **Wave 3:** STM Runtime — activity events, working set, checkpoint to `05-memory_md`, `/v2/activity/*`.
- [x] `[G]` **Raw-first gap fix:** transient spool, raw `source_artifact` ingest (no truncation), drain worker, removed `06-history_md` from projection.
- [x] `[L][G]` **Wave 4:** Governance application boundary — `GovernedService` with actor-aware authorization, `FileAuditLog` (append-only JSONL), `/v2/governance/*` routes, legacy v1 CRUD locked down. Exit gate: non-authorized actor cannot mutate authority through v2 or compatibility paths.
- [x] `[G]` **Wave 5:** Deep Recall and logical arbiter — `DeepService` with explicit capability gate, KG/timeline expansion, `RecallTrace` (SQLite-persisted), deterministic `arbiter.Arbiter` producing conflict proposals (never writes), `/v2/recall/deep` + `/v2/recall/traces/{id}`. Exit gate: no default request can trigger KG/timeline/graph.
- [x] `[G]` **Wave 6:** EvoMap/Evolver adapter — `evolution.Service` with bounded `EvolverProvider` interface, evidence bundle policy gate, candidate normalization + leakage detection, SQLite proposal/event store, `/v2/evolution/*` routes (7 endpoints), Hub disabled by default. Exit gate: Evolver failure degrades only evolution, never Fast Recall or authority safety.

---

## Planned implementation sequence — reset after ADR-0002

> The older Wave 1–7 implementation checklist is superseded as an execution plan where it assumes LTM promotion, `LONGMEM.MD`, legacy section semantics, or reuse of `proposal_inbox`. Retain it only as historical design input until the tasks below are approved and decomposed.

### ✅ Gate A — Cognitive files and compatibility migration design

**Goal:** specify physical representations, validation, migration/no-data-loss policy and host projections before implementation.

- [x] `[L][G]` Define `MEMRULES.MD` storage location, schema, human-only edit path and Garden enforcement boundary.
- [x] `[L][G]` Define `WORLD.MD` claim format, provenance/status/confidence/visibility convention, user edit path and bounded projection API.
- [x] `[L]` Map legacy `06-history_md` and the 14-section registry to compatibility-only behavior; define read-only/data-preservation migration.
- [x] `[L]` Define append-only change/audit log backend and retention; exclude ordinary STM edits and raw Mentle ingest.
- [x] `[G]` Define Context Plane test matrix: Frozen Core + STM + scoped WORLD, excluding full WORLD/MEMRULES/reports/raw material.
- [x] `[L][G]` **Phase 1:** ADR-0004 written; `cognitive` package scaffolded; compat write-blocking (410); `/v2/cognitive/world` endpoint.
- [x] `[L][G]` **Phase 2:** default `MEMRULES.MD`/`WORLD.MD` creation (`GARDEN_COGNITIVE_DIR`), main.go wiring (endpoint live), MEMRULES load + builtin fallback + manual-edit audit, WORLD projection in Fast/Deep Recall (incl. degraded path), audit rotation (`GARDEN_AUDIT_MAX_AGE_DAYS`/`GARDEN_AUDIT_MAX_SIZE_MB`), Context Plane matrix tests (8/8), compat 410 on `/v2/governance/mutations`.

**Exit:** architecture review accepts a concrete migration ADR and test plan. No registry rename/delete before this gate.

**Status:** Complete. ADR-0004 accepted; Phase 1 + Phase 2 implemented and tested. Physical registry migration was executed in Gate E (ADR-0008).

### ✅ Gate B — Report-system expansion design

**Goal:** make reports human-facing and add optional monthly modules without turning them into authority or default context.

- [x] `[R]` Specify daily/weekly/monthly report catalog and readable artifact contract.
- [x] `[R]` Specify monthly-only `AMBITION` and `USER SUGGESTIONS` modules, including non-binding language and user controls.
- [x] `[R][G]` Define bounded Mentle-outage orientation read; reports must never replace raw transient-spool recovery.
- [x] `[L]` Retire `13-report_indexes` from the target model; locate report metadata inside the report subsystem.

**Status:** Complete. ADR-0005 accepted; artifact contract + SQLite migration, dual-write publisher (07/08/09, report_system actor), optional LLM enrichment, v2 endpoints (list/latest/generate/orientation), Console Reports page shipped. AMBITION / USER SUGGESTIONS modules were specified in ADR-0005 §8 and delivered in Gate F (ADR-0009).

**Exit:** report data model and acceptance tests approved.

### ✅ Gate C — Mentle semantic ingestion and EvoMap mailbox design

**Goal:** preserve AAAK's useful chunking lesson without retaining a Laputa AAAK section; isolate external evolution communication.

- [x] `[M]` Design Obsidian source adapter and provenance-preserving semantic units: source path, heading path, block offsets, hash, scope/tags and bounded evidence read.
- [x] `[M]` Verify that semantic-unit ingestion does not create a second summary authority.
- [x] `[G]` Design separate EvoMap mailbox: inbox/outbox, state machine, evidence refs, privacy gate, evaluation, approval/rejection, retry and dead letter.
- [x] `[G]` Keep EvoMap Hub disabled and prevent mailbox data from carrying private raw corpus/personality files.

**Status:** Complete. ADR-0006 + ADR-0007 accepted. Core implementation shipped: `source_artifact`/`semantic_unit` kinds in Mentle facade (also fixes the pre-existing e2e ingestion hang — e2e suite green 8.7s), provenance metadata + real-offset bounded evidence read, `IngestSemanticUnits` (idempotent, zero Laputa writes), `internal/mailbox` store + state machines + privacy gate (evolution.CheckOutbound) + retry→dead-letter, v2 read-only mailbox endpoints. Obsidian adapter runtime form intentionally design-only (ADR-0006 §2); Console mailbox/ingestion views deferred.

**Exit:** mailbox and Obsidian-ingestion designs are approved independently.

### ✅ Gate D — Operations Console (ADR-0003)

**Goal:** deliver the local-first MemoryOS 运营台: workbench-first admin UI (sidebar/top-bar/main), layered governance graph, recall trace, materials/evidence, architecture library, i18n (English default + 中文 toggle), dev-server iteration with Garden `/v2/admin/*` aggregation.

- [x] `[G]` Define `/v2/admin/*` read-only aggregation contract: overview, components, spool, audit; every field carries `source: live | compat | accepted-design`.
- [x] `[G]` Stand up frontend dev server (Vite-class) proxying `/v1/*` + `/v2/*` to Garden `127.0.0.1:7373`; prod target = Garden-embedded static bundle. (dev server + Go-embed prod bundle both done)
- [x] `[F]` MVP-0 pages: Overview (4 health axes), Governance Map (layered graph + 4-layer node state + Inspector), Pipeline/ingestion/recall trace read, Architecture Library L0–L5.
- [x] `[F]` i18n scaffold: English default, 中文 catalog, top-bar language toggle; every shipped string externalized.
- [x] `[F]` Materials & Evidence two-layer model: Card discovery → bounded Evidence read; no full-text dump in search results. (`/v2/materials/*` adapter + three-pane page)
- [x] `[F]` Recall Trace replay: selected / read / rejected paths visible for a real trace id.
- [x] `[F]` Legacy 14-section registry and `06/13/14` visually marked compatibility-only with ADR-0002 mapping.
- [x] `[F]` Design-preview mode toggle (`source: accepted-design` labels); production zero-mock enforced.
- [x] `[G][F]` Safety banner reflects real degraded state (mentle/pipeline/planner, pending_mentle); loopback-only admin API.

**Exit:** ADR-0003 acceptance criteria §9 all pass with real data; no write path in MVP-0.

### ✅ Gate E — Legacy compatibility removal (ADR-0008)

**Goal:** delete the entire legacy compatibility surface — v1 HTTP routes, CRUD translator chain, `06/10/11/12/13/14` registry sections, and the 410 compat mechanism — because no host uses the legacy Laputa runtime.

- [x] `[G]` Promote v1-carried target functionality to v2: `/v2/memories` (canonical CRUD), `/v2/ingest/sessions`, `/v2/ingestions/{id}`, `/v2/pipelines*`, `/v2/recall/bootstrap`; delete all `/v1/*` routes.
- [x] `[G][M]` Delete the CRUD translator chain: `internal/crud`, `internal/router`, `mentle/facade/crud.go`, `legacy_key`/`backfillCanonical`.
- [x] `[L]` Converge the governance registry to 01-05 + 07-09; remove `Compat`/`tbd` machinery and the 410 mapping; delete repo section files for removed sections.
- [x] `[G]` Remove wakeup `06-history_md` writes (`SyncTurn`/`OnSessionEnd` + CLI actions); converge `rag` to the planner surface; update console to v2; rewrite e2e against v2.

**Status:** Complete. ADR-0008 accepted; full verification green (laputa governance, mentle facade, garden internal, e2e 5.3s). Legacy files on user machines are no longer created/read but are not actively deleted.

**Exit:** physical migration plan approved and executed without data loss (inert legacy JSON files remain on disk, unreferenced).

### ✅ Gate F — AMBITION / USER SUGGESTIONS modules (ADR-0009)

**Goal:** deliver the ADR-0005 §8 monthly-only human modules without reviving sections 10/11 — relocated into report-system SQLite, non-binding, user-only write path, referenced from monthly reports.

- [x] `[R]` Write ADR-0009: modules relocate to the report system (`human_modules` table in garden state db); supersede the ADR-0005 §8 carrier sentence (sections 10/11 deleted by ADR-0008).
- [x] `[R]` Module store + CRUD in `internal/report`: `kind ∈ {ambition, suggestion}`, content ≤ 2000 runes, `active ⇄ dismissed`, no hard delete, no governance/audit dependency.
- [x] `[R]` Monthly-only attachment: monthly reports carry `modules: ["AMBITION","USER SUGGESTIONS"]` for in-window active entries; daily/weekly never do; artifact JSON additive (legacy rows read back `modules == []`).
- [x] `[G]` v2 endpoints `GET/POST /v2/reports/modules`, `PATCH /v2/reports/modules/{id}` with 200/201/400/404/503 paths.
- [x] `[F]` Console Modules page (AMBITION + USER SUGGESTIONS columns, create/edit/dismiss/reactivate, i18n en/zh); first console write path (`post`/`patch` clients).
- [x] `[G]` Tests: report unit (CRUD/bounds/state machine/monthly-only/legacy compat), server handlers, e2e lifecycle + monthly `modules` array, Context Plane regression (module content never in `/v2/recall/*`).

**Status:** Complete. ADR-0009 accepted; full verification green (garden internal 13 pkgs, e2e 5.2s, console `tsc --noEmit`; laputa/mentle regression untouched).

**Exit:** ADR-0009 test matrix §5 all pass; modules are non-binding and user-only with no ContextView projection.

### ✅ Gate G — EvoMap Hub transport (GEP-A2A) provider (ADR-0010)

**Goal:** promote `cmd/evomap-demo` to the formal in-process `EvoMapProvider` in `garden/internal/evolution` — read/write surface already verified against the live hub; hermetic mock-hub tests for CI.

- [x] `[G]` ADR-0010: in-process GEP-A2A transport refines ADR-0001 §9.2; provider run = local record of one hub round trip (publish-mode / discovery-mode); sidecar reserved for a future real Evolver.
- [x] `[G]` `HubClient` in `internal/evolution/hub.go`: hello (idempotent auto-register), heartbeat, free search, paid fetch (CLI-only), validate, publish, report; `CanonicalHash`; typed errors (`ErrNoCredentials`/`ErrUnauthorized`/`ErrHubUnreachable`); node_secret redacted everywhere; session-only endpoints (revoke/decision) never implemented.
- [x] `[G]` `EvoMapProvider`: `CheckOutbound` gate before any network call; publish gated by `GARDEN_EVOMAP_HUB_PUBLISH=1` + bundle policy; deterministic Gene+Capsule pair; `evomap_runs` persistence; `PollRun` local read + heartbeat + MaxLifetime timeout; `Candidates` maps hub search results (confidence from GDI) with local fallback; `Status` liveness view.
- [x] `[G]` Wiring: `main.go` env config (`GARDEN_EVOMAP_HUB_URL`/`CREDS`/`HUB_PUBLISH`/`AUTO_REGISTER`); degraded (not fatal) without creds/hub; `GET /v2/evolution/hub/status`.
- [x] `[G]` CLI: `cmd/evomap-demo` → `cmd/evomap` thin wrapper (hello/heartbeat/fetch/search/validate/publish/report); only path to paid fetch tier.
- [x] `[G]` Tests: `hubtest` hermetic GEP-A2A mock; hub client (auth/redaction/errors/determinism), provider (both modes, gates, degradation, timeout, integration), server (hub/status 200/503, HTTP run flow), e2e (real binary + mock hub: auto-register → run → status).

**Status:** Complete. ADR-0010 accepted; full verification green (garden internal 13 pkgs, e2e 13.8s, live `cmd/evomap heartbeat` against evomap.ai OK; laputa/mentle regression untouched).

**Exit:** ADR-0010 §6 test strategy all pass; degradation paths verified; live-hub verification recorded in the research doc §10.

---

### ✅ Gate H — Console API audit + Evolution panel

**Goal:** comprehensive front/back API cross-audit to make the Operations Console fully usable; surface Gate F/G results in the console.

- [x] `[G]` Audit matrix: all 14 frontend-called endpoints verified against backend contracts; SPA fallback, CORS posture, degradation table recorded in `docs/dev/reference/2026-08-03-console-api-audit.md`.
- [x] `[G]` Frontend fixes: ModulesPage dismiss/reactivate `disabled={busy}` race; `client.ts` empty-body guard; missing `--ink-1` CSS token.
- [x] `[G]` Backend bug (found live): `UpdateModule` returned zero `created_at` — fixed + regression test.
- [x] `[G]` New read-only endpoints: `GET /v2/evolution/runs` and `GET /v2/evolution/proposals` (store `ListRuns`/`ListProposals`, limit 1–100), coexisting with the by-id routes.
- [x] `[G]` Console `/evolution` page: hub status card (claim link, credits, heartbeat), runs list + start-run form + candidate → proposal flow, proposals list + leakage report + two-step approve/reject; zh/en i18n.
- [x] `[G]` Docs: removed nonexistent `POST /v2/governance/proposals` from root and garden AGENTS.md; list endpoints added to all endpoint tables.

**Status:** Complete. Full verification green (internal all pkgs, e2e 10.9s, `tsc -b` clean); live-server pass: 13 GET endpoints 200, discovery run 202→completed, no-creds instance degrades to 503 while lists/admin stay 200; module `created_at` fix re-verified live.

**Exit:** audit doc §6 verification table complete; honest note — no browser automation in this environment, frontend verified via tsc + bundle assertions + curl shapes.

---

### ✅ Gate I — Console mailbox UI (pure frontend)

**Goal:** surface the EvoMap mailbox (ADR-0007 §5) in the console; scope confirmed with user — no backend changes (no outbox retry endpoint).

- [x] `[G]` Console `/mailbox` page: inbox/outbox two-column layout (8s polling), expandable items with payload/evidence/facts, two-step approve/reject with optional reason for `received`/`evaluating` inbox items, dead-letter items shown with leakage report (violations/warnings/checked refs), zh/en i18n, sidebar nav + route + badge state colors.
- [x] `[G]` Docs: mailbox routes added to root `AGENTS.md` API table (previously missing entirely); audit doc §5 marks mailbox UI delivered.
- [x] `[G]` Verified: seeded SQLite via temporary `cmd/seedmailbox` (inbox received/approved/rejected; outbox queued/acked/dead-letter with `leakage.clean=false`), live server curls for three GETs + approve/reject 200/404/409, `tsc -b` clean, bundle assertions, `go test ./internal/...` green; seed tool removed after verification.

**Status:** Complete. Verification green (internal all pkgs, `tsc -b` clean, live-server shape checks on seeded mailbox); honest note — no browser automation in this environment, frontend verified via tsc + bundle assertions + curl shapes.

**Exit:** seeded mailbox live-verified end to end; user-confirmed scope (pure frontend) honored; seed tooling deleted.

---

## Deferred legacy implementation waves

- ⛔ Old “STM → LTM promotion” work is removed; do not implement it.
- ⛔ Old `proposal_inbox` work is not the EvoMap mailbox; do not implement it as such.
- ⛔ `12-changelog` is not a user-facing Laputa section; do not create new section-content features around it.
- ⛔ `13-report_indexes` and `14-aaak_summaries` are deleted target concepts; do not revive them.
- ⬜ Fast Recall/Card/Evidence and raw-first STM implementation work may resume only after Gate A reconciles their contracts with `MEMRULES.MD` and `WORLD.MD`.
- ⬜ EvoMap adapter and host adapters remain downstream of Gates A–C.

---

## Maintenance

- Check a box only when its stated exit gate has real verification output.
- Update **Last updated** and **Current focus** on every edit.
- Preserve historical archive material; do not rewrite it to match current terminology.
- Never reset, restore, format or commit unrelated dirty worktree files.
