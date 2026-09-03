<!-- Parent: ../../AGENTS.md -->

# mentle/storage/govector — HNSW Vector Store Implementation

**Generated:** 2026-08-01  
**Purpose:** Core vector storage backend using Hierarchical Navigable Small World (HNSW)

---

## Purpose

The `govector/` package implements persistent vector search:

- **HNSW index** — hierarchical navigable small world graph
- **384-dimensional vectors** — from all-MiniLM-L6-v2
- **Cosine distance** — similarity metric
- **Fast approximate search** — logarithmic time complexity
- **Deterministic results** — same query yields same ranking

See parent [mentle/storage/AGENTS.md](../AGENTS.md) for interface documentation.

---

## Structure

```text
govector/
└── store.go                         # Vector store implementation (single file)
```

The package is a single `store.go`. It wraps `DotNetAge/govector` `core.Collection` + `core.Storage` (bbolt). There is no separate `index.go` or `serialization.go`; HNSW graph persistence is internal to govector's `core.Storage`.

---

## Key Operations

```go
// Create or open store (path + vector dimension are both required)
store, err := govector.NewStore("./vectors.db", 384)

// Add vectors (upsert semantics)
err := store.Add(id, vector, payload)     // single
err := store.AddBatch(points)             // batch

// Search
results, err := store.Search(query, limit, filter)   // top-k with optional filter map

// Delete
err := store.Delete(id)

// List all (zero-vector scan)
results, err := store.ListAll(limit)
```

---

## Performance

| Operation | Target |
|-----------|--------|
| Insert | P95 ≤ 5ms |
| Search (top-10) | P95 ≤ 30ms |
| Search (top-50) | P95 ≤ 80ms |

---

## Build & Test

```bash
cd mentle
GOSUMDB=off go test ./storage/govector/...
```

---

## MANUAL

HNSW parameters (M, ef) are internal and may change without notice. Do not expose in facade API.

Parent reference: ../AGENTS.md
