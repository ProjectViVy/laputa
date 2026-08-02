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

```
govector/
├── store.go                         # Vector store implementation
├── store_test.go                    # Test suite
├── index.go                         # HNSW index management
├── serialization.go                 # Index persistence
└── (supporting utilities)
```

---

## Key Operations

```go
// Create or open store
store, err := govector.NewStore("./storage")

// Add vectors
err := store.Insert(ctx, id, vector)

// Search
results, err := store.Search(ctx, query, k)  // top-k

// Rebuild index
err := store.Rebuild(ctx)
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
