<!-- Parent: ../AGENTS.md -->

# mentle/storage — Vector Store Backend (Persistence)

**Generated:** 2026-08-01  
**Purpose:** HNSW vector storage and persistence layer

---

## Purpose

The `storage/` directory implements the persistent vector store:

- **govector** — HNSW-based vector search backend (via DotNetAge/govector)
- **Persistence** — save/load vector index to disk
- **Recovery** — rebuild index if corrupted

See [mentle/vector/AGENTS.md](../vector/AGENTS.md) for detailed API documentation.

---

## Structure

```
storage/
├── govector/
│   └── store.go                   # HNSW vector store (bbolt) — the active production backend
├── redis/
│   └── store.go                   # Redis drawer storage (linear scan search; optional, not default)
├── sqlite/
│   └── store.go                   # Raw SQLite helper (CGO driver)
└── vectorstore/
    └── vectorstore.go             # Unified Store interface + Open() factory
```

> **Backend status:** `BackendGoVector` is the only backend wired into `facade` (production path). `BackendQdrant`, `BackendChroma`, `BackendLanceDB` are **declared but not implemented** — `vectorstore.Open` returns `"not yet implemented"` for them. `storage/chroma/` and `storage/qdrant/` contain AGENTS.md docs only, no Go code.

---

## Key Components

### HNSW Index

Hierarchical Navigable Small World graph:

- **384-dimensional vectors** (from all-MiniLM-L6-v2)
- **Cosine distance** metric
- **Fast approximate search** in logarithmic time
- **Deterministic results** for same queries

### Persistence

Save and load index from disk:

```bash
./storage/govector/vectors.db   # bbolt file holding the HNSW graph + payloads (not SQLite)
```

`vectors.db` is a **bbolt file** (via DotNetAge/govector `core.Storage`), not a SQLite database — its magic header is `00000000...`, not `SQLite format 3`. The HNSW graph, vectors, and payload metadata all live inside this single bbolt file. There is no separate `hnsw.idx`; the graph is persisted inside `vectors.db`.

### Recovery

If index is corrupted, recreate the store (HNSW graph lives in `vectors.db`; rebuilding from canonical store data is handled by mentle's `repair` CLI / index_jobs replay):

---

## Build & Test

### Build

```bash
cd mentle
go build ./storage/...
```

### Test

```bash
cd mentle
GOSUMDB=off go test ./storage/...
```

---

## Performance

| Operation | Target |
|---|---:|
| Add vector | P95 ≤ 5ms |
| Search (top-10) | P95 ≤ 30ms |
| Rebuild index | O(n log n) |

---

## MANUAL

When updating:

1. Index format is not part of public API (changes allowed internally)
2. Always provide recovery path when changing persistence format
3. Do not expose HNSW internals (M, ef parameters) in facade

Parent reference: ../AGENTS.md
