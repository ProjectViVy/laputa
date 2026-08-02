# TODOLIST — Garden MemoryOS

> **Last updated:** 2026-08-02  
> **Current focus:** Gate D — Operations Console: `/v2/admin/*` backend + MVP-0 frontend (React/Vite) built & verified on live data; next = Materials & Evidence (needs Mentle `/v2/materials/*`), design-preview toggle, Go-embed prod bundle  
> **Architecture source:** [0001 MemoryOS vNext Architecture](docs/architecture/0001-memoryos-vnext-architecture.md) + [ADR-0002 Cognitive Partition](docs/architecture/0002-laputa-cognitive-partition-decision.md)  
> This file tracks execution status only. It must not restate design contracts.

**Legend:** ✅ done · 🚧 active · ⬜ planned · ⛔ blocked  
**Modules:** `[L]` Laputa · `[M]` Mentle · `[G]` Garden · `[R]` Report System

---

## 🔧 Now — documentation and migration boundary

- [ ] `[G]` Commit/record the existing documentation transition only after reviewing all pre-existing dirty worktrees.
- [ ] Record a current baseline-test snapshot for Laputa governance, Mentle facade, Garden internal and Garden E2E suites.
- [ ] Add this `TODOLIST.MD` to version control and retain links from root project guidance.
- [ ] Produce a **separate, approved physical migration plan** before changing the legacy 14-section Go registry or persisted files.

**Guard:** `06-history_md`/LTM, `13-report_indexes`, and `14-aaak_summaries` must receive no new target-architecture features. Existing files/data remain read-only compatibility evidence until the migration plan exists.

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

### ⬜ Gate A — Cognitive files and compatibility migration design

**Goal:** specify physical representations, validation, migration/no-data-loss policy and host projections before implementation.

- [ ] `[L][G]` Define `MEMRULES.MD` storage location, schema, human-only edit path and Garden enforcement boundary.
- [ ] `[L][G]` Define `WORLD.MD` claim format, provenance/status/confidence/visibility convention, user edit path and bounded projection API.
- [ ] `[L]` Map legacy `06-history_md` and the 14-section registry to compatibility-only behavior; define read-only/data-preservation migration.
- [ ] `[L]` Define append-only change/audit log backend and retention; exclude ordinary STM edits and raw Mentle ingest.
- [ ] `[G]` Define Context Plane test matrix: Frozen Core + STM + scoped WORLD, excluding full WORLD/MEMRULES/reports/raw material.

**Exit:** architecture review accepts a concrete migration ADR and test plan. No registry rename/delete before this gate.

### ⬜ Gate B — Report-system expansion design

**Goal:** make reports human-facing and add optional monthly modules without turning them into authority or default context.

- [ ] `[R]` Specify daily/weekly/monthly report catalog and readable artifact contract.
- [ ] `[R]` Specify monthly-only `AMBITION` and `USER SUGGESTIONS` modules, including non-binding language and user controls.
- [ ] `[R][G]` Define bounded Mentle-outage orientation read; reports must never replace raw transient-spool recovery.
- [ ] `[L]` Retire `13-report_indexes` from the target model; locate report metadata inside the report subsystem.

**Exit:** report data model and acceptance tests approved.

### ⬜ Gate C — Mentle semantic ingestion and EvoMap mailbox design

**Goal:** preserve AAAK's useful chunking lesson without retaining a Laputa AAAK section; isolate external evolution communication.

- [ ] `[M]` Design Obsidian source adapter and provenance-preserving semantic units: source path, heading path, block offsets, hash, scope/tags and bounded evidence read.
- [ ] `[M]` Verify that semantic-unit ingestion does not create a second summary authority.
- [ ] `[G]` Design separate EvoMap mailbox: inbox/outbox, state machine, evidence refs, privacy gate, evaluation, approval/rejection, retry and dead letter.
- [ ] `[G]` Keep EvoMap Hub disabled and prevent mailbox data from carrying private raw corpus/personality files.

**Exit:** mailbox and Obsidian-ingestion designs are approved independently.

### 🚧 Gate D — Operations Console (ADR-0003)

**Goal:** deliver the local-first MemoryOS 运营台: workbench-first admin UI (sidebar/top-bar/main), layered governance graph, recall trace, materials/evidence, architecture library, i18n (English default + 中文 toggle), dev-server iteration with Garden `/v2/admin/*` aggregation.

- [x] `[G]` Define `/v2/admin/*` read-only aggregation contract: overview, components, spool, audit; every field carries `source: live | compat | accepted-design`.
- [x] `[G]` Stand up frontend dev server (Vite-class) proxying `/v1/*` + `/v2/*` to Garden `127.0.0.1:7373`; prod target = Garden-embedded static bundle. (dev server done; Go-embed prod bundle still pending)
- [x] `[F]` MVP-0 pages: Overview (4 health axes), Governance Map (layered graph + 4-layer node state + Inspector), Pipeline/ingestion/recall trace read, Architecture Library L0–L5.
- [x] `[F]` i18n scaffold: English default, 中文 catalog, top-bar language toggle; every shipped string externalized.
- [ ] `[F]` Materials & Evidence two-layer model: Card discovery → bounded Evidence read; no full-text dump in search results. (blocked on Mentle `/v2/materials/*` adapter)
- [x] `[F]` Recall Trace replay: selected / read / rejected paths visible for a real trace id.
- [x] `[F]` Legacy 14-section registry and `06/13/14` visually marked compatibility-only with ADR-0002 mapping.
- [ ] `[F]` Design-preview mode toggle (`source: accepted-design` labels); production zero-mock enforced. (per-item `accepted-design` labels done; global toggle pending)
- [x] `[G][F]` Safety banner reflects real degraded state (mentle/pipeline/planner, pending_mentle); loopback-only admin API.

**Exit:** ADR-0003 acceptance criteria §9 all pass with real data; no write path in MVP-0.

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
