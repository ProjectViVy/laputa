# ADR-0014: Mentle Canonical Authority and Derived-Index Recovery

**Status:** accepted
**Date:** 2026-09-03
**Decision owner:** project owner
**Refines:** ADR-0011 recovery and embedding-identity work
**Supersedes:** ADR-0011 WAL-as-canonical-recovery and next-backend sequencing; retains its KG provenance, embedding identity, session ingest, health and boundary goals
**Implements:** BMAD Garden Authority & Recovery Convergence, AR-007 through AR-010

## 1. Context

Current Mentle runtime has one viable memory authority and two incompatible mutation/recovery models:

1. Garden REST uses `canonical.sqlite3`, transactionally creates `index_jobs`, then applies govector/BM25 updates.
2. Legacy MCP/miner paths can mutate `Searcher` directly and append a JSONL WAL afterward, bypassing canonical memory, version, idempotency and audit records.

The JSONL WAL implementation has durable append/checkpoint primitives, but canonical create/update/delete does not write it and no runtime consumer restores canonical state from it. The existing repair command moves the whole palace, reads the wrong post-move WAL path and creates a 1024-dimensional vector store while production uses 384 dimensions.

Treating this WAL as canonical recovery would create a second authority and require atomic coordination between SQLite and files that does not exist. Optional vector backends do not solve that authority problem.

## 2. Decision

### 2.1 Canonical SQLite is the sole memory authority

`canonical.sqlite3` exclusively owns logical memory ID, content, metadata, version, status, idempotency and audit state. A successful canonical transaction is the authority commit.

No JSONL log, vector store, BM25 structure, Redis key or backend-specific payload may be used to reconstruct or override canonical memory state.

### 2.2 `index_jobs` is the transactional derived-index outbox

Every canonical create/update/delete transaction records its derived-index operation in `index_jobs` before commit. A worker claims and applies jobs after commit.

The outbox records:

- stable job ID;
- canonical memory ID and version;
- operation and required immutable payload/reference;
- state (`pending`, `leased`, `retry`, `failed`);
- attempts and last error;
- next-attempt time;
- lease owner/expiry;
- created/updated timestamps.

Application is idempotent. A failed/poison job degrades index health but never rolls back canonical authority, blocks unrelated jobs, or prevents the server from starting.

### 2.3 Vector and lexical indexes are disposable

The current govector/bbolt HNSW and in-memory BM25 are derived projections. They may be deleted and rebuilt from active current canonical revisions plus the current embedding identity.

Physical vector IDs may remain version-qualified, but normal retrieval and rebuilt indexes must exclude inactive/old canonical revisions before candidate starvation can occur.

### 2.4 Repair is a staged derived-index rebuild

Repair/reindex:

1. opens canonical SQLite read-only or at a consistent snapshot;
2. reads active current revisions;
3. captures and validates embedding identity;
4. builds derived artifacts in a same-volume staging directory;
5. verifies identity, counts and search smoke tests;
6. atomically swaps only derived artifacts;
7. retains/rolls back to the prior index on any failure.

Repair never renames, deletes, overwrites or reconstructs `canonical.sqlite3`.

### 2.5 Optimistic concurrency is enforced in SQL

Expected-version update/delete operations use the version in the SQL predicate and require one affected row. Pre-read comparison is not a correctness boundary.

### 2.6 All mutation adapters call the Facade

REST, MCP, CLI, project mining, session ingestion and provider adapters call `facade.Service`. Adapters are not given mutable `Searcher`, raw vector store or raw canonical DB handles.

### 2.7 JSONL WAL is not canonical recovery

Legacy business-WAL producers and recovery claims are retired. Any retained append-only diagnostics must be clearly named/scoped as non-authoritative and must not participate in memory reconstruction. Unused WAL resources are removed and closed cleanly.

### 2.8 Optional backends are deferred

Redis, Qdrant, Chroma and LanceDB are outside this batch. A future vector backend must:

- implement the actual vector-store interface;
- expose capabilities/health;
- pass rebuild, identity and fault tests;
- remain disposable and subordinate to canonical SQLite.

## 3. Live health contract

`IndexHealth` reports at least:

- canonical active count;
- current active vector count;
- BM25 document count;
- embedding identity and compatibility;
- pending/retry/failed job counts;
- oldest pending age;
- tombstone/old-version pressure;
- last successful rebuild and current rebuild state;
- machine-readable degradation reasons.

Garden `/v2/admin/index-health` and component health aggregate this live report rather than startup-time labels.

## 4. Failure semantics

| Failure | Required behavior |
|---|---|
| Canonical transaction fails | no authority change and no visible index job |
| Crash after canonical commit, before index apply | pending job survives and is replayed |
| Crash after index apply, before job completion | idempotent replay converges |
| Transient index failure | retry with bounded backoff; health degraded |
| Poison job | isolate after bounded attempts; unrelated work continues |
| Embedding identity mismatch | reject incompatible apply; require explicit rebuild |
| Staged rebuild failure | canonical and prior usable index remain untouched |
| Count divergence | expose degraded state and reason; never silently claim healthy |

## 5. Migration and clean-break rule

This is a runtime clean break, not a data migration:

- existing canonical SQLite remains authority;
- existing derived indexes may be rebuilt;
- legacy direct-Searcher mutation endpoints are removed or rewritten to Facade in one cutover;
- no dual-write grace period;
- no import from legacy drawer WAL into canonical without a separately reviewed one-time data recovery procedure.

## 6. Acceptance gates

1. Concurrent same-version writers produce exactly one success and one typed conflict.
2. Every runtime mutation entry point produces canonical state and transactional index work.
3. Crash-point tests prove outbox replay before/after index apply.
4. A poison job does not block startup or unrelated jobs.
5. Deleting/corrupting only derived index artifacts is recoverable from canonical.
6. Repair monitoring proves canonical SQLite is never renamed or overwritten.
7. Embedding identity mismatch is explicit and prevents mixed-index writes.
8. IndexHealth detects injected count/job/identity divergence.
9. Repeated session ingest is deterministic and idempotent.
10. Mentle boundary scan reports no Persona/WORLD/ACTMEM authority dependency.

## 7. Consequences

### Positive

- one durable memory authority;
- no cross-store atomicity fiction;
- indexes can fail or be replaced without losing memory;
- operational degradation becomes observable and repairable;
- future backends remain implementation details rather than authority choices.

### Cost

- `index_jobs` requires schema/worker/health work;
- current MCP/miner direct paths must be rewritten;
- old repair and WAL claims must be removed;
- canonical backup/integrity procedures remain operational requirements because SQLite is the authority.

## 8. Non-goals

- replacing canonical SQLite with Redis or a vector database;
- BM25 disk persistence;
- automatic Persona/ACTMEM promotion;
- distributed multi-host consensus;
- importing all historical drawer/WAL records into canonical.
