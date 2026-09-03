# Garden Authority & Recovery Convergence — PRD

**BMAD phase:** Planning
**Status:** Accepted execution-planning baseline
**Date:** 2026-09-03
**Scope:** Existing brownfield system; next multi-epic batch
**Binding inputs:** ADR-0012, ADR-0013, DIVA Persona/ACTMEM implementation

## 1. Why

Garden currently has working Persona read/write primitives and Mentle memory CRUD, but the system is not yet operating under one coherent authority model:

- retired JSON Governance remains wired into startup, recall, reports, checkpoints, HTTP, tests, and Console;
- WORLD is still automatically projected into ContextView;
- ACTMEM has no Garden implementation;
- Persona write/review exists but diverges from DIVA write-class, CAS, validation, atomicity, and history semantics;
- Mentle has canonical SQLite plus legacy direct-index/WAL mutation paths, an unsafe repair command, and no real index-health contract;
- Garden's full test gate is red and some passing tests still assert retired behavior.

This batch turns the accepted clean-break design into a verifiable runtime and establishes one recoverable Mentle write authority before optional storage backends are considered.

## 2. Product outcome

At completion, a local Garden profile has:

1. one Markdown Persona authority with DIVA-compatible write/review/history behavior;
2. one independent, restart-safe ACTMEM authority;
3. one session-frozen six-document Frozen Core and no automatic WORLD/ACTMEM projection;
4. one canonical memory authority in SQLite, with disposable/rebuildable vector and lexical indexes;
5. domain-specific REST/MCP/Console surfaces that do not expose retired governance concepts;
6. a green, deletion-first acceptance gate proving the old JSON runtime cannot return.

## 3. Users and jobs

| User | Job |
|---|---|
| Profile owner | Initialize, inspect, edit and repair Persona; review agent proposals without accidental overwrite |
| Local agent | Read allowed Persona/ACTMEM data explicitly, propose protected Persona changes, write ordinary memories through one canonical path |
| Operator | Determine whether canonical memory and indexes are healthy; safely rebuild derived indexes |
| Developer | Implement independently reviewable stories without choosing conflicting authority, API or recovery semantics |

## 4. Functional requirements

### Persona

- **FR-PER-01** — Store exactly seven closed Markdown document kinds using the DIVA directory/history layout.
- **FR-PER-02** — Atomically initialize exactly the five required documents; partial pre-existing state must fail with an integrity/repair error.
- **FR-PER-03** — Enforce exact revision CAS; `base_revision=0` must not bypass an existing revision.
- **FR-PER-04** — Treat normalized identical content as no-op without advancing revision.
- **FR-PER-05** — Enforce user-direct, agent-reviewed, and agent-direct write classes by authenticated principal and document scope.
- **FR-PER-06** — Validate request actor/kind, profile readiness, content cap, USER scope and WORLD protection before persisting a review.
- **FR-PER-07** — Persist immutable snapshots/diffs and DIVA-compatible stale/accepted/rejected request state.
- **FR-PER-08** — Provide explicit initialization, document, request-list/decision, history and repair APIs.

### ACTMEM

- **FR-ACT-01** — Store `<profile>/actmem/ACTMEM.MD` independently of Persona and Mentle.
- **FR-ACT-02** — Implement DIVA-compatible revision CAS, no-op behavior, Pulse/Recap/Work structure and capacity limits.
- **FR-ACT-03** — Support bounded read/query plus maintenance operations and capsule lifecycle.
- **FR-ACT-04** — Preserve ACTMEM across sessions and Garden restart.
- **FR-ACT-05** — Never inject ACTMEM automatically into bootstrap, Fast Recall, Deep Recall, planner or ContextView.

### Context and clean break

- **FR-CTX-01** — Capture bounded projections of exactly Identity, Relationship, Redline, User, Dream and Dark once per session.
- **FR-CTX-02** — Preserve a session's Frozen Core across profile edits; persist the derived snapshot in Garden SQLite for same-session process restart.
- **FR-CTX-03** — Remove GovernanceProjection, WorldProjector, ContextView.World and all automatic WORLD assembly.
- **FR-CTX-04** — Move WorkingSet checkpoint and report publication state to Garden SQLite.
- **FR-CTX-05** — Remove runtime discovery/read/write of `.laputa/sections`, generic Governance mutation routes and retired DTOs.

### Mentle recovery

- **FR-MEM-01** — Route REST, MCP, CLI/miner and provider writes through `facade.Service` and canonical SQLite.
- **FR-MEM-02** — Use canonical SQLite as the sole memory authority; vector and BM25 indexes are derived and replaceable.
- **FR-MEM-03** — Use transactional `index_jobs` as the outbox, with attempts, last error, availability/lease state and poison-job isolation.
- **FR-MEM-04** — Enforce optimistic concurrency atomically in SQL using `WHERE version=?` and `RowsAffected`.
- **FR-MEM-05** — Rebuild derived indexes from active canonical revisions using embedding identity; never rebuild authority from the legacy JSONL WAL.
- **FR-MEM-06** — Replace the unsafe repair command with staging, verification, atomic vector-store swap and rollback.
- **FR-MEM-07** — Expose live IndexHealth covering canonical/index counts, embedding identity, pending/failed jobs, tombstones and last rebuild.
- **FR-MEM-08** — Make conversation/session ingest IDs deterministic and re-ingest idempotent.

### Adapters and Console

- **FR-API-01** — Use one clean-break REST contract: `/v2/persona/documents`, `/v2/persona/reviews`, `/v2/actmem`, and `/v2/admin/index-health`; remove old aliases atomically.
- **FR-API-02** — Protect write-class endpoints with principal-scoped local capability tokens; `X-Garden-Actor` remains audit metadata only.
- **FR-MCP-01** — MCP reads and writes must map to domain APIs and never bypass canonical/review rules.
- **FR-MCP-02** — `memory_search` must return query-matched results only.
- **FR-UI-01** — Replace Governance Map with Persona, Memory, Evolution and Chat Approval workspaces backed only by live APIs.
- **FR-UI-02** — WORLD and ACTMEM full content may load only after an explicit user action.

## 5. Non-functional requirements

- **NFR-01 Durability:** successful authority writes survive process crash and restart; failed atomic operations leave prior authority valid.
- **NFR-02 Integrity:** no dual authority, compatibility path, silent truncation or silent recovery divergence.
- **NFR-03 Security:** actor labels are not authorization; protected writes require a verified principal class.
- **NFR-04 Observability:** degraded index/recovery state is explicit and machine-readable.
- **NFR-05 Portability:** all tests pass on the current Windows/MSYS environment; no Unix-only atomicity assumptions.
- **NFR-06 Testability:** each story begins with a failing contract/fault test and ends with focused plus affected full-suite verification.
- **NFR-07 Scope:** no unrelated refactor and no ownership claim over pre-existing dirty changes.

## 6. Non-goals

- Redis, Qdrant, Chroma or LanceDB production backend implementation.
- BM25 on-disk persistence before canonical rebuild correctness is proven.
- Persona extraction or automatic promotion from Mentle/ACTMEM.
- Automatic ACTMEM-to-Mentle promotion or autonomous compaction policy.
- Host artifact installation, autonomous Skill publication or new EvoMap transport.
- Migration, aliases, fallback reads or dual-write compatibility for retired JSON sections.

## 7. Success signals

- All vertical-slice acceptance tests pass, including crash/restart, stale CAS, write-class and explicit-only context tests.
- Repository deletion scan reports zero target-runtime references to retired JSON Persona/Governance/WORLD projection concepts.
- Every memory mutation entry point is proven to create/update canonical state and corresponding recoverable index work.
- Index rebuild from canonical succeeds after deleting/corrupting only derived index artifacts.
- Garden, Laputa, Mentle and Console gates are green.
- Runtime and Console expose no route, field or status that presents accepted-design/stub behavior as live.

## 8. Traceability

Implementation is divided into five epics under `epics/`. Architecture decisions are in `ARCHITECTURE-SPINE.md`. Readiness findings and the no-code planning gate are in `implementation-readiness.md`.
