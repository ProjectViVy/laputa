# Garden Authority & Recovery Convergence — Architecture Spine

**BMAD phase:** Solutioning
**Status:** Accepted execution spine
**Date:** 2026-09-03
**Inherits:** ADR-0012 and ADR-0013
**Authority reference:** DIVA `agent-diva-laputa`

This spine records only decisions that independently implemented epics could otherwise make incompatibly.

## 1. System shape

```text
                         explicit reads/writes
Host / Console / MCP ─────────────────────────────────────┐
        │ principal-scoped capability                     │
        ▼                                                  ▼
┌──────────────────────────── Garden ─────────────────────────────┐
│ domain HTTP adapters                                             │
│ Persona review │ ACTMEM tools │ Recall │ Memory │ Index health  │
│                                                                  │
│ session_frozen_core ──► ContextView ◄── bounded Mentle evidence │
│      Garden SQLite          ▲                                    │
│                             │                                    │
│ activity/checkpoints/reports/spool/traces — Garden SQLite       │
└──────────────┬───────────────────────────────┬───────────────────┘
               │                               │
       ┌───────▼────────┐              ┌───────▼──────────────────┐
       │ Laputa          │              │ Mentle                   │
       │ persona/*.MD    │              │ canonical.sqlite3       │
       │ history/reviews │              │ transactional index_jobs│
       │ actmem/ACTMEM.MD│              │ vectors.db + BM25        │
       └─────────────────┘              └──────────────────────────┘

WORLD/ACTMEM ── explicit tool lane only
Retired JSON sections ── no runtime edge
```

## 2. Decisions

### AR-001 — Authority partition is closed

- Laputa owns Persona and ACTMEM semantics.
- Mentle owns memory/evidence authority and derived retrieval indexes.
- Garden owns runtime/session/checkpoint/report state and adapters.
- EvoMap owns capability candidate/proposal/artifact lifecycle.
- Chat Approval, Persona Review and EvoMap Review remain separate workflows.

**Rejected:** generic Governance engine as a shared mutation authority.

### AR-002 — DIVA semantic parity is mandatory

Garden's Persona and ACTMEM ports copy DIVA-observable behavior: path layout, caps, normalization, CAS, no-op behavior, write classes, USER/WORLD protections, stale requests, snapshots/diffs, ACTMEM sections and capsules.

A Garden-specific simplification is allowed only when it is behaviorally equivalent and has parity tests.

### AR-003 — Persona multi-file initialization uses staging

Initialization validates all five required documents first, writes a complete `.init-staging/<operation-id>` tree including history, flushes files, then performs one guarded install. On Windows, replacement uses same-volume rename semantics and explicit cleanup/recovery of abandoned staging directories.

Partial live state is never completed automatically; it requires explicit repair.

### AR-004 — Principal class, not actor header, controls writes

- `X-Garden-Actor` is audit metadata only.
- Local write endpoints authenticate a capability token mapped to `user`, `agent`, or `operator` principal.
- User principal may direct-write protected Persona documents.
- Agent principal must create a Persona review for protected documents; only allowed P16 fields/documents may direct-write.
- Operator principal may invoke explicit repair but cannot silently bypass content rules.
- The MCP server receives an agent-scoped token and exposes only agent-legal operations.

Tokens are compared in constant time, never logged, and remain optional only for read-only loopback endpoints during this batch.

### AR-005 — Frozen Core is a derived session snapshot

At the first bootstrap for a session ID, Garden reads exactly six Persona documents, applies fixed caps, and persists the resulting immutable snapshot plus source revisions in Garden SQLite. Repeated bootstrap and process restart with the same session ID return the same snapshot. A new session captures current revisions.

`WORLD.MD` and `ACTMEM.MD` are structurally absent from the Frozen Core type and ContextView DTO.

### AR-006 — Explicit tool lane has no recall dependency

Persona full-document reads, including WORLD, and ACTMEM read/query are separate domain services and adapters. Fast Recall, Deep Recall, planner, bootstrap assembly and background Console polling may not depend on these tool services.

The Console may issue an explicit request only after the user selects WORLD or requests ACTMEM content/query.

### AR-007 — SQLite canonical is the sole Mentle memory authority

`canonical.sqlite3` is the only authority for memory identity, version, status, content, metadata and idempotency.

- SQLite's transactional durability protects canonical state.
- `index_jobs` in the same transaction is the durable outbox for derived index work.
- `vectors.db` and BM25 contain no irreplaceable data.
- The legacy JSONL business WAL is not promoted into a second canonical log and is removed from canonical recovery semantics.

**Rejected:** reconstructing canonical state from legacy drawer WAL; dual-write to SQLite and JSONL; direct Searcher mutation.

### AR-008 — Index jobs are isolated, idempotent state machines

Each job records ID, canonical memory/version, operation, state, attempts, last error, next-attempt time, lease owner/expiry and timestamps. Physical vector IDs remain version-qualified. Processing is idempotent; one poison job degrades health but does not block unrelated jobs or server startup.

Retry follows the established bounded backoff policy. Permanent identity mismatch requires explicit reindex, not silent acceptance.

### AR-009 — Repair only replaces derived artifacts

Repair/reindex reads a consistent active canonical snapshot, captures embedding identity, builds vector/BM25 state in a staging location, verifies counts/search smoke/identity, then atomically swaps only derived artifacts. Canonical SQLite is never renamed, overwritten or reconstructed. Failure preserves the old usable index and emits operator-visible status.

### AR-010 — Atomic optimistic concurrency lives in SQL

Updates/deletes include expected version in the mutation predicate and require exactly one affected row. A pre-read is informational only. Zero affected rows maps to typed version conflict/not-found after disambiguation.

### AR-011 — One mutation entry point per domain

- Memory REST, MCP, miners and importers call `facade.Service`.
- Persona adapters call the Persona policy service, never the raw file writer.
- ACTMEM adapters call the ACTMEM service.
- No adapter receives `Searcher`, raw SQLite DB or file-store mutation capability.

### AR-012 — API cutover is breaking and atomic

The target contract follows ADR-0013:

- `/v2/persona/documents[/{kind}]`
- `/v2/persona/reviews[/{id}/approve|reject]`
- `/v2/persona/history/{kind}[/{revision}]`
- `/v2/actmem` and `/v2/actmem/query`
- `/v2/admin/index-health`

Current `/v2/persona/files`, `/requests`, generic `/v2/governance/*`, and `/v2/cognitive/world` routes are deleted in the same cutover; no aliases or fallback period.

### AR-013 — Health is live evidence, not startup labels

IndexHealth is computed from canonical/index counts, embedding identity, job states, tombstone pressure, and last rebuild. Garden health aggregates this live probe. UI consumers must represent `ok`, `degraded`, and `unavailable` without converting missing endpoints into success.

### AR-014 — Console follows domain workspaces

The stable top-level workspaces are Persona, Memory, Evolution and Chat Approval. Governance Map and WORLD background polling are deleted, not renamed. ACTMEM and Mentle evidence remain visually and semantically distinct inside Memory.

### AR-015 — Optional database backends are deferred

This batch does not add Redis, Qdrant, Chroma or LanceDB. A future backend must implement the vector-store contract, pass capability/health/rebuild tests, and remain a derived index. It may never replace canonical authority by configuration accident.

## 3. Persistence map

| Data | Owner | Persistence | Recoverability |
|---|---|---|---|
| Persona documents/history/reviews | Laputa | Markdown + immutable history/request records | staging/repair under DIVA semantics |
| ACTMEM/capsules | Laputa | Markdown files under `actmem/` | CAS, atomic replace, restart tests |
| Session Frozen Core | Garden | SQLite | immutable by session ID, restart-safe |
| Activity/checkpoint/report/spool/trace | Garden | SQLite | transactional repositories |
| Memory canonical/idempotency/index jobs | Mentle | SQLite | sole authority + transactional outbox |
| Vector HNSW | Mentle | bbolt/govector | rebuild from canonical |
| BM25 | Mentle | memory | rebuild from canonical/index stream |
| EvoMap/mailbox | Garden EvoMap | SQLite | existing lifecycle guarantees |

## 4. Error and degradation rules

- Authority validation/CAS failure: reject; do not mutate current file/row/history.
- Derived index apply failure after canonical commit: return canonical success with explicit `index_pending` state where API contract permits; health becomes degraded and job retries.
- Poison job: quarantine/failed state after bounded attempts; other jobs continue.
- Embedding identity mismatch: block incompatible index mutation; enqueue/require explicit rebuild.
- Frozen Core capture failure: bootstrap fails rather than falling back to retired JSON authority.
- ACTMEM unavailable: explicit tool call fails; recall remains unaffected because it has no dependency.

## 5. Verification architecture

Every epic must provide:

1. focused unit/contract tests;
2. fault/concurrency tests where persistence changes;
3. affected full-suite run;
4. negative boundary/deletion assertions;
5. no-success-with-stub proof for health/UI adapters.

Final gate additionally runs repository deletion scans, Garden e2e under the new contract, Console behavior tests/build, and `git diff --check`.

## 6. Explicitly deferred decisions

- ACTMEM automatic compaction and promotion into Mentle.
- BM25 disk persistence and vector backend pluggability.
- EvoMap host artifact installer.
- Non-loopback multi-user authentication and remote authorization.
- Fate of legacy rhythm/scheduler/wakeup/web packages after dependency removal; they require a separate delete-or-rehome ADR.
