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
│   ├── store.go                   # Vector store implementation
│   ├── store_test.go
│   └── (index management)
└── vectorstore/
    └── interface.go               # Public interface for vector operations
```

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
./storage/govector/hnsw.idx    # Binary HNSW graph
./storage/govector/vectors.db   # SQLite metadata
```

### Recovery

If index is corrupted, rebuild from SQL metadata:

```go
store, err := govector.NewStore("./storage")
if err != nil {
    store.Rebuild(ctx)
}
```

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
