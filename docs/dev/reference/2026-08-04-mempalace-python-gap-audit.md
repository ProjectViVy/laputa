# Python MemPalace ↔ Mentle/Garden Capability Gap Audit

**Date:** 2026-08-04
**Author:** 松本
**Baselines:** Python MemPalace `939a076baf0b349e1f5b3a7e27ad1d545364f18b` (package version `3.4.0`), Mentle / Garden / Laputa `c6cdceee0830036c258e50523874ea5b9693810e`.
**Scope:** Source-grounded comparison only. No code was modified; no tests were run.
**Driver:** identify genuine engineering gaps that should become Gate E (ADR-0011) sub-tasks; explicitly exclude capabilities already covered by Laputa.
**Method:** three parallel read-only subagents (`deleg_240d3643`) covering ingestion/archive/repair/source adapters; storage/embedding/retrieval/dedup/index health; MCP/ops/taxonomy/KG/LLM/onboarding/hook boundaries — each cited Python source paths and line ranges, and each compared against the current Go implementation. This document synthesizes those reports plus the post-audit file reads.

---

## 1. Headline finding

The current Go implementation has **all the scaffolding** (canonical memory, idempotent ingest, hybrid BM25 + vector + RRF, Fast/Deep Recall separation, trace contracts, governance filter) but **lacks the hard contracts** that turn "can run" into "can be trusted after a crash, a model swap, or a corrupt transcript". The 84-day MemPalace article (`docs/dev/reference/2026-08-02-mempalace-84d-long-memory-case.md`) validates this reading: the failure modes the author flags (index broken for 5 minutes, raw memory lost on rollback) are not addressed by scaffolding alone.

Five gaps surface repeatedly across the three subagent reports. They are listed below with the exact source evidence and the proposed Gate E sub-task.

## 2. Gap 1 — WAL durability

### 2.1 Evidence in current Mentle

`mentle/pkg/wal/wal.go:42-69, 85-103`:

```go
f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
...
_, err = f.Write(append(data, '\n'))
return err
```

```go
filename := filepath.Join(w.dir, fmt.Sprintf("write_log_%s.jsonl", time.Now().Format("2006-01-02")))
...
lines := splitLines(string(data))
for _, line := range lines {
    if line == "" { continue }
    if err := json.Unmarshal([]byte(line), &entry); err != nil { continue }
    entries = append(entries, entry)
}
```

- No `O_SYNC` / `f.Sync()` — durability depends on the OS page cache.
- Segmentation is by calendar day, not by byte count; long-running days produce single huge files.
- `readFile` does not detect torn lines; a mid-write crash leaves a half-line that JSON-decodes as an empty entry and silently disappears.
- There is no `checkpoint` / `watermark` table; replay always starts from the oldest segment.

### 2.2 Evidence in Python MemPalace

Python's `palace_ingest` and WAL layer keep `commit_id`-keyed recovery on the SQLite-backed palace: WAL entries are paired with checkpoint commits, replay uses `commit_id ≤ X`, and partial lines are recovered by reading to the next newline.

### 2.3 Classification

| Bucket | Decision |
|---|---|
| Already stronger | — |
| Genuine gap / candidate to add | **Yes — Gate E sub-task E1 (WAL contract)** |
| Deferred | — |
| Reject (Laputa owns it) | — |

### 2.4 Implementation owner

Mentle (`pkg/wal`). No Laputa involvement.

## 3. Gap 2 — Embedding identity

### 3.1 Evidence in current Mentle

`mentle/internal/embedder/hugot.go:34-37`:

```go
if modelName == "" {
    modelName = "sentence-transformers/all-MiniLM-L6-v2"
}
```

`mentle/facade/facade.go:51-77` hard-codes 384 dims in the embedder pipeline. There is no `Identity()`, no `Dimension()`, no `Provider()` method on `Embedder`. Switching to a 768-dim model produces silently wrong cosine distances with no typed error.

### 3.2 Evidence in Python MemPalace

`mempalace/embedding.py:6-17, 133-151, 238-265`: tracks model id, MRL truncation, device/provider fallback (CUDA/CoreML/DirectML/CPU). Model switch is gated by an explicit re-embed step.

### 3.3 Classification

| Bucket | Decision |
|---|---|
| Already stronger | — |
| Genuine gap / candidate to add | **Yes — Gate E sub-task E2 (Embedding identity)** |
| Deferred | — |
| Reject (Laputa owns it) | — |

### 3.4 Implementation owner

Mentle (`internal/embedder` + `facade.Catalog.embedding_identity` + `Facade.CreateMemory` typed errors). No Laputa involvement.

## 4. Gap 3 — Vector backend reality

### 4.1 Evidence in current Mentle

`mentle/storage/vectorstore/vectorstore.go:48-80`:

```go
case BackendQdrant:  return nil, fmt.Errorf("qdrant: not yet implemented")
case BackendChroma:  return nil, fmt.Errorf("chroma: not yet implemented")
case BackendLanceDB: return nil, fmt.Errorf("lancedb: not yet implemented")
```

The interface is honest; the implementations are stubs. `govector` is reachable only by bypassing `vectorstore.Open` and using `govector.NewStore` directly. `redis` is reachable only through `storage/redis.Open`. `Health(ctx)` does not exist on the interface.

### 4.2 Evidence in Python MemPalace

`mempalace/backends/base.py:233-311, 403-410` and `mempalace/backends/registry.py:56-157, 171-199, 207-225`: a single `Store` contract with `Capabilities`, `Health(ctx)`, `BackendArtifactDetect`, collection/namespace isolation, and real implementations for Chroma, SQLite exact, Qdrant, pgvector.

### 4.3 Classification

| Bucket | Decision |
|---|---|
| Already stronger | — |
| Genuine gap / candidate to add | **Yes — Gate E sub-task E3 (Real vector backends)** |
| Deferred | — |
| Reject (Laputa owns it) | — |

### 4.4 Implementation owner

Mentle (`storage/vectorstore` + a local stand-in backed by SQLite FTS5 + exact vector candidate set). No Laputa involvement.

## 5. Gap 4 — KG provenance and temporal semantics

### 5.1 Evidence in current Mentle

`mentle/internal/kg/knowledge_graph.go:94-116, 149-155`:

```go
func (kg *KnowledgeGraph) AddTriple(subject, predicate, obj, validFrom, validTo string, confidence float64) (string, error) {
    subID := entityID(subject)
    objID := entityID(obj)
    pred := strings.ToLower(strings.ReplaceAll(predicate, " ", "_"))
    hash := sha256.Sum256([]byte(time.Now().String()))
    ...
    _, err = kg.db.Exec(`INSERT INTO triples (id, subject, predicate, object, valid_from, valid_to, confidence) VALUES (...)`, ...)
}
```

`QueryEntity(name)` `default` branch returns only `subject = ?` rows (`kg.go:149-155`); the bidirectional union does not actually fire on the default path. `asOf` is a raw string compare; inverted intervals are accepted; RFC3339 normalization is absent.

Schema declares `source_closet / source_file / extracted_at` but `AddTriple` does not accept or write them. `source_drawer_id / adapter_name / adapter_version` are missing entirely.

### 5.2 Evidence in Python MemPalace

`mempalace/knowledge_graph.py:155-175, 237-360, 364-527`: triples carry `source_closet`, `source_file`, `source_drawer_id`, `adapter_name`, `extracted_at`; MCP writes (`mcp_server.py:1753-1807`) accept provenance; `asOf` is RFC3339-normalized with inverted-interval rejection.

### 5.3 Classification

| Bucket | Decision |
|---|---|
| Already stronger | — |
| Genuine gap / candidate to add | **Yes — Gate E sub-task E4 (KG provenance + temporal correctness)** |
| Deferred | — |
| Reject (Laputa owns it) | — |

### 5.4 Implementation owner

Mentle (`internal/kg`). No Laputa involvement. KG fact-checking, persona auto-promotion, and weight/heat semantics stay in Laputa.

## 6. Gap 5 — Session atomic ingest

### 6.1 Evidence in current Mentle

`mentle/internal/miner/convo_miner.go:303-324`:

```go
// generate a random id for the drawer
drawerID := fmt.Sprintf("convo_%s", generateID())
```

No `session_id + message_uuid` stable ID. No cursor. No source lock. Re-ingestion produces different drawer IDs across runs; partial re-ingest cannot resume.

### 6.2 Evidence in Python MemPalace

`mempalace/sweeper.py:8-23, 147-177, 183-205, 229-299`: deterministic IDs from `(session_id, message_uuid)`, timestamp cursor with `<=` semantics, rewind-resume, tie-safe timestamps, verbatim tool block preservation.

### 6.3 Classification

| Bucket | Decision |
|---|---|
| Already stronger | — |
| Genuine gap / candidate to add | **Yes — Gate E sub-task E5 (Session atomic ingest)** |
| Deferred | — |
| Reject (Laputa owns it) | — |

### 6.4 Implementation owner

Mentle (`internal/miner` + `facade.session_cursor` + `facade.session_lock`). Garden's idempotent ingest + transient spool remain unchanged.

## 7. Gap 6 — Index health is a static claim

### 7.1 Evidence in current Mentle

`mentle/cmd/server/main.go:904-918`:

```go
// health tool returns Status: OK, Vector DB: connected
```

There is no real probe; the response is constant text. `facade.Catalog.index_jobs` exists (`facade/canonical.go:114-117`) and replay is implemented (`canonical.go:364-405`), but no surface joins vector count, BM25 doc count, embedder identity, and pending/failed jobs.

### 7.2 Classification

| Bucket | Decision |
|---|---|
| Already stronger | — |
| Genuine gap / candidate to add | **Yes — Gate E sub-task E6 (Index health endpoint)** |
| Deferred | — |
| Reject (Laputa owns it) | — |

### 7.3 Implementation owner

Mentle (`facade.Service.IndexHealth`). Garden surfaces as `/v2/admin/index-health` under existing `/v2/admin/components`.

## 8. P1 gaps (raw envelope, source adapter, dedup, collision, query sanitize, lexical fallback)

Source-grounded, lower priority than the P0 list above:

- **Raw envelope + normalized projection.** Current `internal/miner/normalize.go:121-155` collapses Claude Code messages to `string(message.content)`; tool_use/tool_result blocks are lost. Persist the original JSONL line as a `source_artifact` (ADR-0006) with a `normalizer_version` and bounded raw read.
- **Real source adapter contract.** Python `mempalace/sources/base.py:68-153, 164-245` and `mempalace/sources/registry.py:31-162` define `SourceRef`, version, schema, transform declaration, `is_current`. Current Mentle couples filesystem / project / conversation paths into miner/CLI; no plugin registry.
- **Near-duplicate maintenance.** Python `mempalace/dedup.py:53-129, 152-208`: per-source grouping, length-preferred retention, vector cosine-distance threshold, dry-run report, batched delete. Current Mentle `facade/cards.go:74-96` only collapses same-`Memory.ID` versions; no semantic near-duplicate path.
- **ID collision pre-check.** Python `mempalace/collision_scan.py:41-99, 102-121`: merges `(source_file, chunk_index)` from incoming/existing; rejects same-physical-id-different-provenance. Current Mentle `internal/hybrid/searcher.go:137-162` and `storage/govector/store.go:32-57` upsert by `drawer.ID` with no provenance key.
- **Query sanitization.** Python `mempalace/query_sanitizer.py:29-72, 105-191`: surrogate cleanup, question extraction, tail extraction, audit fields. Current Mentle `internal/hybrid/searcher.go:78-101` and `facade/retrieval.go:42-54` only check empty string.
- **Lexical fallback when vector fails.** Current `internal/hybrid/searcher.go:94-97` fails the whole search if vector fails; Python falls back to SQLite FTS (`mempalace/searcher.py:388-412, 450-545, 599-620`).

## 9. P2 gaps (export, sync, format miner, hook adapter, MCP runtime)

- **`mentle export --format markdown|jsonl`.** Python `mempalace/exporter.py:1-12, 68-208`: paginated Markdown export; symlink-safe. Current Mentle CLI has no `export`.
- **`mentle sync --dry-run`.** Python `mempalace/sync.py` reports `missing/gitignored/out-of-scope`; current CLI has no `sync`.
- **Binary format miner.** Python `mempalace/format_miner.py:1-34, 118-133, 152-189, 315-449, 538-582, 693-742`: PDF/DOCX/PPTX/XLSX/RTF/EPUB with retryable vs permanent-skip status. Current Mentle supports text/code/config extensions only (`internal/miner/miner.go:19-34`).
- **Harness hook adapter.** Garden already accepts `precompact/session_end` with content-hash verification and spool (`garden/internal/ingest/service.go:100-218`); what's missing is the Claude/Codex JSONL → event id/hash/timeout/retry adapter package, not the ingest pipeline itself.
- **Mentle MCP runtime.** `pkg/mcp/server.go:80-200` is line-oriented stdio JSON-RPC; Python `mcp_server.py:299-449, 3115-3204` adds writer lease, dependency health, request deadline/cancellation, idle handle cleanup, optional HTTP transport.

## 10. What is explicitly excluded (covered by Laputa, not a Mentle gap)

These are present in Python MemPalace and not migrated into Mentle, by ownership rule:

- **Automatic user-profile authority promotion.** Laputa decides what becomes a `MEMRULES.MD` / `WORLD.MD` claim; Mentle provides the substrate.
- **Fact-checking / entity contradiction policy.** Laputa runs the conservative `fact_checker.py` analog; Mentle only guarantees that any KG fact points back to its drawer (`E4` provenance).
- **Hebbian/Ebbinghaus dynamic weights / cross-memory reinforcement.** Laputa decides policy; Mentle already implements heat decay at the presentation layer (`facade/cards.go:141-147`) and stops there.
- **Long-form persona / identity / commitment.** Laputa owns Frozen Core (ADR-0002 §3.1) and audit; no Mentle write path may touch them.
- **Audit / authority / rollback.** Laputa's `governance/audit.go:54-119` and `governed.go:67-150` are unchanged.
- **Report semantics.** Garden's `internal/report` + Laputa's `governance/rhythm/generator.go:41-70` already produce governed daily/weekly/monthly artifacts; no Mentle re-implementation.

## 11. Boundary verification

Gate E sub-task `E8` runs a CI grep over the changed Mentle surface for these tokens:

```
governance|audit_log|MEMRULES|WORLD|persona
```

Expected hits: 0 in Mentle. Any non-zero hit blocks Gate E exit and forces a re-design that respects ownership.

## 12. References

- `docs/architecture/0001-memoryos-vnext-architecture.md` §3-§6, §13
- `docs/architecture/0002-laputa-cognitive-partition-decision.md` §3
- `docs/architecture/0006-semantic-ingestion-and-obsidian-adapter.md`
- `docs/architecture/0011-recoverable-indexing-and-evidence-contract.md`
- `docs/dev/reference/2026-08-02-mempalace-84d-long-memory-case.md`
- `mentle/pkg/wal/wal.go:42-69, 85-103`
- `mentle/internal/embedder/hugot.go:34-37`
- `mentle/storage/vectorstore/vectorstore.go:48-80`
- `mentle/internal/kg/knowledge_graph.go:94-116, 149-155`
- `mentle/internal/miner/convo_miner.go:303-324`
- `mentle/cmd/server/main.go:904-918`
- `mentle/internal/hybrid/searcher.go:78-101, 137-162`
- `mentle/facade/cards.go:74-96, 141-147`
- `mentle/facade/canonical.go:114-117, 364-405`
- `garden/internal/ingest/service.go:100-218`
- `laputa/governance/audit.go:54-119`
- `laputa/governance/governed.go:67-150`
- `laputa/governance/rhythm/generator.go:41-70`