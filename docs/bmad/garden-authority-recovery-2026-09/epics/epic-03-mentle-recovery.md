# Epic 3 — Mentle Single Authority and Recoverable Indexing

**Outcome:** all memory writes converge on canonical SQLite and every secondary index can be observed and safely rebuilt from it.

**Requirements:** FR-MEM-01..08, NFR-01..06
**Architecture:** AR-007..011, AR-013, AR-015
**Dependencies:** Epic 0 baseline; may run in parallel with Epic 1 until adapter cutover
**Exit checkpoint:** fault-oriented recovery review before external adapter/Console work.

## Story 3.1 — Enforce atomic canonical version CAS

**Acceptance criteria**

- Update/delete predicates include expected version where applicable.
- Exactly one affected row is required.
- Concurrent same-version writers yield one success and one typed conflict.
- Idempotency and current CRUD behavior remain intact.

## Story 3.2 — Converge every mutation entry point on Facade

**Acceptance criteria**

- Old MCP add/delete, project mining and import/session paths call `facade.Service`.
- No command/adapter receives direct `Searcher.Store/Delete` mutation authority.
- Every accepted write appears in canonical tables and creates appropriate index work.
- Tests prove REST/MCP/miner consistency and no split-brain path.

## Story 3.3 — Evolve index_jobs into a recoverable outbox

**Acceptance criteria**

- Add job ID/state/attempt/error/next-attempt/lease timestamps with a safe schema migration.
- Claim/apply/complete transitions are atomic and idempotent.
- Poison jobs do not prevent service startup or unrelated job processing.
- Retry is bounded and observable.
- Crash after canonical commit and before index apply is recovered on restart.

## Story 3.4 — Complete embedding identity enforcement

**Acceptance criteria**

- Dimension, metric, normalization, model/provider/version policy is explicit.
- Incompatible mutation is rejected with typed error.
- Identity change creates a reindex request rather than silently mixing vectors.
- Existing and missing identity states have deterministic health semantics.

## Story 3.5 — Replace repair with canonical-derived staged rebuild

**Acceptance criteria**

- Current destructive repair implementation is removed or hard-disabled before replacement.
- Rebuild reads active canonical revisions and current embedding identity.
- Build occurs in staging; verifies identity, active counts and search smoke.
- Only derived vector artifacts are swapped; canonical is untouched.
- Failure rolls back to old index and reports a typed/operator-visible reason.

## Story 3.6 — Define tombstone and BM25 rebuild correctness

**Acceptance criteria**

- Rebuild includes only active current canonical versions.
- Deleted/old physical revisions cannot crowd out active retrieval candidates.
- BM25 rebuild is sourced from canonical active content or a proven equivalent stream, not a capped tombstone-bearing vector listing.
- Health reports tombstone pressure and count divergence.
- Disk persistence of BM25 remains explicitly deferred.

## Story 3.7 — Implement live IndexHealth

**Acceptance criteria**

- Report canonical active count, vector active count, BM25 count, embedding identity, pending/failed job counts, oldest pending age, last rebuild and reasons.
- States are `ok`, `degraded`, or `unavailable` from live probes.
- Garden `/v2/admin/index-health` and component aggregation use the same report.
- Injected divergence and missing identity tests produce the expected degraded reasons.

## Story 3.8 — Make session ingest deterministic and restart-safe

**Acceptance criteria**

- Stable ID derives from session/message identity.
- Re-ingesting identical source produces no duplicate canonical memory.
- Cursor and per-session lease are durable and crash-safe.
- Corrupted/truncated source resumes from a safe boundary.
- Raw source/provenance remains addressable.

## Story 3.9 — Retire legacy business-WAL authority claims

**Acceptance criteria**

- Canonical recovery documentation and runtime no longer claim JSONL WAL can rebuild canonical memory.
- Unused WAL producer/consumer seams are removed or clearly scoped to non-authority diagnostics.
- Service Close releases all retained resources.
- Recovery tests prove authority from SQLite plus transactional outbox, not dual logs.
