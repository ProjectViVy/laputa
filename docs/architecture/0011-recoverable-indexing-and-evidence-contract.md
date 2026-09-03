# ADR-0011: Recoverable Indexing & Evidence Contract

**Status:** proposed
**Date:** 2026-08-04
**Driver:** Python MemPalace ↔ Mentle/Garden source-grounded gap audit (2026-08-04, see `docs/dev/reference/2026-08-04-mempalace-python-gap-audit.md`).
**Depends on:** ADR-0012 §6 (Laputa authority and Mentle evidence boundary), ADR-0006 (raw-first `source_artifact`, `SourceAdapter`, bounded evidence read).
**Refined by:** accepted ADR-0014. ADR-0014 supersedes this proposal's WAL-as-canonical-recovery and optional-backend sequencing. KG provenance, embedding identity, deterministic session ingest, live health and authority-boundary goals remain useful inputs.

---

## 1. Context

A source-grounded comparison between the official Python MemPalace reference (`mempalace @ 939a076`) and the current Go implementation (`mentle/garden/laputa @ c6cdcee`) shows that the **scaffolding is in place** but several recovery and identity contracts are **"can run, cannot be trusted"**. The goal of this ADR is to harden those contracts without smuggling governance into Mentle.

The audit identifies five material gaps:

1. **WAL durability.** `mentle/pkg/wal/wal.go:42-69, 85-103` uses `O_APPEND|O_CREATE|O_WRONLY` without `O_SYNC`/`fsync`, segments by calendar day, and parses lines without partial-line recovery. A crash mid-write leaves a half-line that breaks replay; a corrupted segment cannot be replayed cleanly.
2. **Embedding identity.** `mentle/internal/embedder/hugot.go:34-37` defaults to `MiniLM-L6-v2` but exposes no `Identity/Dimension/Provider`. `mentle/facade/facade.go:51-77` hard-codes 384 dims. Model swap produces silently wrong cosine distances.
3. **Vector backend.** `mentle/storage/vectorstore/vectorstore.go:48-80` advertises `BackendQdrant|Chroma|LanceDB` but each returns `"not yet implemented"`. The interface is honest, the implementations are stubs.
4. **KG provenance & temporal semantics.** `mentle/internal/kg/knowledge_graph.go:94-116` `AddTriple` uses `time.Now().String()` as a hash seed, does not persist `source_closet/source_file/extracted_at`, has no `source_drawer_id/adapter_name`; `QueryEntity` "default" branch only returns outgoing triples (`kg.go:149-155`); `asOf` is a raw string compare with no RFC3339 normalization or inverted-interval rejection.
5. **Session ingest atomicity.** `mentle/internal/miner/convo_miner.go:303-324` still calls `generateID()` for every run, so the same `(session_id, message_uuid)` resolves to different drawer IDs across runs. There is no cursor, no source lock, no transactional replace of prior session chunks.

The audit also reaffirms what **must not migrate into Mentle**:

- KG fact-checking, persona auto-promotion, weight/heat semantics → Laputa.
- Long-form user profile, authority/permissions, audit semantics → Laputa.
- Graph/timeline traversal policy → Laputa (Mentle provides the substrate).

## 2. Decisions

### 2.1 Mentle is the recovery and evidence layer; it does not absorb governance

All five gaps are addressed in Mentle (`pkg/wal`, `internal/embedder`, `storage/vectorstore`, `internal/kg`, `internal/miner`). None of them introduces:

- a write path to Laputa sections,
- persona inference or user-profile promotion,
- `MEMRULES.MD` or `WORLD.MD` reads,
- an `audit_log` entry from automated index work.

The Gate E boundary audit (`E8`) verifies this by grep over the changed surface for forbidden tokens (`governance`, `audit_log`, `MEMRULES`, `WORLD`, `persona`) and reports results before exit.

### 2.2 WAL contract

`mentle/pkg/wal.WAL` becomes a recoverable append-only log with the following hard properties:

| Property | Decision |
|---|---|
| Durability | every write is `f.Write + f.Sync` (or `O_SYNC` if the host allows it); durable before the call returns |
| Segmentation | fixed-size segments (e.g. 64 MiB), rotated by byte count, not by day |
| Checkpoint | a `wal_checkpoints` table in `facade.Catalog` stores `(segment, byte_offset, last_committed_job_id, captured_at)` |
| Replay | `ReplayFrom(checkpoint)` re-applies `add`/`delete`/`reindex` entries; `delete` is idempotent (matched by drawer id + version) |
| Partial-line recovery | a torn entry is detected and skipped with `torn_offset` recorded; the reader resumes at the next `\n` |
| Reader audit | `ReadAll()` returns `([]Entry, TornOffsets []int64)` so callers can observe degradation, not hide it |

### 2.3 Embedding identity

```go
type Identity struct {
    Model     string    // e.g. "sentence-transformers/all-MiniLM-L6-v2"
    Dimension int       // 384 / 768 / 1024
    Metric    string    // "cosine" | "dot" | "euclidean"
    Normalize bool      // whether vectors are L2-normalized
    Provider  string    // "ort-coreml" | "ort-cpu" | "go-onnx" | "remote"
    Version   string    // build/version id of the embedder
    CapturedAt time.Time
}
```

`mentle/internal/embedder.Embedder` exposes `Identity() Identity`. `facade.Catalog` persists the active identity in `embedding_identity` (one row). `Facade.CreateMemory` enforces dimension and metric consistency:

- **dim mismatch** → reject write with `ErrEmbeddingDimensionMismatch` (typed, surfaced through MCP / HTTP).
- **metric mismatch** → reject write with `ErrEmbeddingMetricMismatch`.
- **identity mismatch on read** → degrade Fast Recall with explicit warning; refuse Deep Recall unless a `reindex_job` is queued.

Identity is captured once at embedder construction and updated only by an explicit `recapture` action that queues a reindex job.

### 2.4 Real vector backends

`storage/vectorstore.Open` becomes honest:

| Backend | Decision |
|---|---|
| `BackendGoVector` | unchanged (local HNSW); `govector.NewStore` returns `Store` directly |
| `BackendRedis` | unchanged (linear scan + Redis keyword search); still scanned for `StorageCapabilities.LexicalSearch` |
| `BackendQdrant` | **implement in-process Qdrant substitute via SQLite FTS5 + an exact-vector candidate set** for Gate E; flag in metadata that this is a local stand-in (no external Qdrant in MVP) |
| `BackendChroma` | implement only as an opt-in adapter behind a build tag (`-tags=chroma`) so the binary stays hermetic; keep stubbed path in default build |
| `BackendLanceDB` | remain stubbed with a typed error; revisit when a Go-native client exists |

The interface gains:

```go
type StorageCapabilities struct {
    LexicalSearch      bool
    VectorSearch       bool
    BackendArtifactDetect bool // can probe local data dir for prior format / version
}
type Store interface {
    Search(ctx, query, limit, filter) ([]SearchResult, error)
    Add(ctx, id, vector, payload) error
    AddBatch(ctx, points) error
    Delete(ctx, id) error
    ListAll(ctx, limit) ([]SearchResult, error)
    LexicalSearch(ctx, query, limit) ([]SearchResult, error) // optional; nil if !cap.LexicalSearch
    Capabilities() StorageCapabilities
    Health(ctx) HealthReport
    Close() error
}
```

`Health(ctx)` returns `{BackendType, VectorCount, BM25DocCount, EmbeddingIdentity, LastReindexJob, Status: ok|degraded|unavailable, Reasons []string}`.

### 2.5 KG provenance & temporal semantics

`internal/kg.AddTriple` gains explicit provenance:

```go
type Triple struct {
    ID              string
    Subject         string
    Predicate       string
    Object          string
    ValidFrom       time.Time
    ValidTo         *time.Time
    Confidence      float64
    SourceCloset    string    // e.g. "obsidian-vault" or "claude-code-jsonl"
    SourceFile      string    // path or content-address
    SourceDrawerID  string    // canonical drawer ref into Mentle
    AdapterName     string    // SourceAdapter identifier (ADR-0006)
    AdapterVersion  string
    ExtractedAt     time.Time
}
```

- Migration: add `source_drawer_id`, `adapter_name`, `adapter_version` columns; backfill `extracted_at` from `valid_from` when missing; preserve existing rows.
- Indexes: `(source_drawer_id)`, `(adapter_name)`, `(extracted_at)`.
- `QueryEntity(name, opts)` `default` mode returns the **bidirectional UNION** (subject OR object), not just outgoing.
- `asOf` accepts RFC3339 and a `[valid_from, valid_to]` pair; inverted intervals (`valid_to < valid_from`) are rejected with `ErrInvalidTemporalInterval`; both `at` and `range` query modes are supported.
- MCP schema updated to expose the new fields and accept provenance on write.

### 2.6 Session atomic ingest

Conversation ingest gains a stable, recoverable identity:

- Drawer ID = `sha1("session:" + session_id + ":" + message_uuid)` (matches Python MemPalace `sweeper.py:8-23`).
- Cursor = `max(message_timestamp)` per `session_id`, persisted in `session_cursor` table.
- Re-ingestion: cursor `<=` semantics; the same `(session_id, message_uuid)` is idempotent.
- Source lock: per-session writer lease backed by a SQLite `session_lock` row (`session_id`, `owner_pid`, `acquired_at`, `ttl`); cross-process PID/lease prevents concurrent hooks from racing the same session.
- Transactional replace: re-ingest of an existing session first deletes prior session chunks in a transaction, then writes new ones; if the writer dies mid-replace, the partial state is rolled back and the cursor does not advance.
- Raw envelope: the original JSONL line and tool blocks are persisted as a `source_artifact` (ADR-0006), with a `normalizer_version` field that records the parser that produced the projection.

### 2.7 Index health endpoint (Mentle side)

`facade.Service.IndexHealth(ctx) HealthReport` joins:

- `embedding_identity` row (existence, version match)
- `index_jobs` (pending count, failed count, oldest pending age)
- vector store `Health(ctx)` (vector count, BM25 doc count, identity match)
- WAL `torn_offset` count and last replay result
- GoVector structural stats if applicable

The MCP `health` tool calls this real probe instead of returning the current static OK (`cmd/server/main.go:904-918`). Garden exposes it as `/v2/admin/index-health` and aggregates it under `/v2/admin/components`.

### 2.8 Failure semantics (cross-cutting)

| Condition | Behavior |
|---|---|
| WAL replay divergence | `/v2/admin/index-health` returns 410; operator-visible banner; Fast Recall degraded with `torn_offset > 0` warning |
| Embedding identity mismatch on read | Fast Recall degraded; Deep Recall refused unless `reindex_job` queued |
| Vector backend unavailable | automatic lexical-only fallback (`LexicalSearch`); results tagged `source: bm25_fallback` in trace |
| KG inverted interval | reject write with typed error |
| Session ingest mid-replace crash | rollback; cursor unchanged; next run resumes idempotently |

## 3. HTTP / MCP Surface

| Surface | New / changed |
|---|---|
| `mentle/pkg/mcp` | `health` returns real probe; `kg_add_triple` accepts provenance; `kg_query_entity` supports bidirectional default + `as_of` RFC3339 |
| `garden/internal/server` | `GET /v2/admin/index-health` reuses `IndexHealth`; aggregated under existing `/v2/admin/components` |
| `mentle/facade` | `CreateMemory` typed errors: `ErrEmbeddingDimensionMismatch`, `ErrEmbeddingMetricMismatch`; `IndexHealth(ctx) HealthReport` |

## 4. Test Matrix (TDD-first; red → green)

| # | Test | Assertion |
|---|------|-----------|
| 1 | WAL torn-line recovery | kill writer mid-line → reader returns zero torn entries after recovery; `TornOffsets` records the offset |
| 2 | WAL replay idempotence | replay same segment twice → catalog identical; `delete` matched by id+version |
| 3 | Embedding identity mismatch | write 384-dim → switch to 768 → second write rejected with typed error; `reindex_job` queued |
| 4 | Real vector backend | `vectorstore.Open(BackendQdrant)` returns a Store backed by SQLite FTS+exact; `Capabilities().LexicalSearch=true`; `Health(ctx)` returns non-empty `BM25DocCount` |
| 5 | KG provenance round-trip | triple inserted with full provenance → `QueryEntity` returns it for both subject and object lookups; inverted interval rejected with typed error |
| 6 | Session atomic ingest | ingest same JSONL twice → no duplicate drawers; IDs match across runs; rewind cursor → re-ingest only new messages; corrupted mid-line → next clean run resumes |
| 7 | Index health endpoint | hand-injected divergence (vector count vs catalog count) → `IndexHealth` reports `degraded: vector_count_mismatch`; missing identity row → `degraded: embedding_identity_missing` |
| 8 | Boundary audit | CI grep over changed surface for `governance|audit_log|MEMRULES|WORLD|persona` → 0 hits in Mentle |

## 5. Non-goals

- No bundled/sidecar Evolver, no host skill auto-install, no EvoMap hub publish in this batch.
- No KG fact-checking, persona auto-promotion, or weight/heat semantics migration into Mentle.
- No silent recovery; every degradable axis is exposed via `IndexHealth` and surfaced in the trace.
- No rewrite of Laputa sections, governance, or audit semantics.
- No rewriting of the Garden ingest contract — Gate E only normalizes what comes **into** Mentle; Garden's idempotent ingest + transient spool remain unchanged.

## 6. Consequences

- **Mentle becomes trustworthy enough to survive a crash and a model swap.** Replays and reindexes are first-class operations, not folklore.
- **The boundary audit (`E8`) prevents governance drift.** Every Gate E change is grep-checked against the ownership rule before exit.
- **`/v2/admin/index-health` is the operator's single source of truth** for "is this index trustworthy right now". The Console can surface it directly without inventing its own probes.
- **The 84-day MemPalace article (`docs/dev/reference/2026-08-02-mempalace-84d-long-memory-case.md`) validates the chosen direction.** Its two real risks — index broken for 5 minutes, raw memory lost on rollback — are exactly what `E1`, `E2`, `E5`, `E6` make impossible.
- **KG provenance becomes a substrate, not a persona authority.** Mentle records evidence for a future `WORLD.MD` claim; Persona review decides whether an allowed proposal changes that file.

## 7. References

- `docs/architecture/0012-laputa-markdown-clean-break.md` §6 (Laputa authority and Mentle evidence boundary)
- `docs/architecture/0006-semantic-ingestion-and-obsidian-adapter.md` (raw-first `source_artifact`, `SourceAdapter`, bounded evidence)
- `docs/dev/reference/2026-08-02-mempalace-84d-long-memory-case.md` (84-day article; validates direction)
- `docs/dev/reference/2026-08-04-mempalace-python-gap-audit.md` (synthesizes the three subagent reports; cites exact source paths and line ranges)
- `mentle/pkg/wal/wal.go:42-69, 85-103`
- `mentle/internal/embedder/hugot.go:34-37`
- `mentle/storage/vectorstore/vectorstore.go:48-80`
- `mentle/internal/kg/knowledge_graph.go:94-116, 149-155`
- `mentle/internal/miner/convo_miner.go:303-324`
- `mentle/cmd/server/main.go:904-918`
