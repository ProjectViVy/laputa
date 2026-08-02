<!-- Parent: ../AGENTS.md -->

# mentle/vector — Vector Storage Backend (HNSW)

**Generated:** 2026-08-01  
**Purpose:** HNSW-based vector store for semantic search index

---

## Purpose

The `vector/` package (or `storage/govector/`) implements the vector storage backend:

- **HNSW (Hierarchical Navigable Small World)** index for fast approximate nearest neighbor search
- **Vector persistence** to disk with recovery
- **Index optimization** and rebuild
- **Configurable distance metrics** (cosine, euclidean)

---

## Structure

```
storage/
└── govector/
    ├── store.go                   # Vector store implementation
    ├── store_test.go
    └── (index management utilities)
```

---

## Key Concepts

### HNSW Index

Hierarchical Navigable Small World graph:

- **Approximate nearest neighbor search** in logarithmic time
- **Configurable M (max neighbors per node)** — default 16
- **Configurable ef_construction** — default 200 (higher = better quality, slower build)
- **Search-time ef (exploration factor)** — default 100 (higher = more accurate, slower)

### Vector Properties

- **Dimension:** 384 (from all-MiniLM-L6-v2 model)
- **Type:** float32
- **Distance metric:** cosine similarity (default)
- **Normalization:** vectors are normalized to unit length

### Persistence

- **Format:** HNSW graph serialized to disk
- **Recovery:** index is rebuilt on load if corrupted
- **Incremental updates:** new vectors added without full rebuild (usually)
- **Optimization:** periodic index rebuild for better search performance

---

## Usage

### Create Store

```go
store, err := govector.NewStore("./storage/vectors")
if err != nil {
    log.Fatal(err)
}
defer store.Close()
```

### Add Vector

```go
// Each vector is labeled with a memory ID
err := store.Add(ctx, "card-2024-08-01-auth-flow", embedding)
if err != nil {
    log.Fatal(err)
}
```

### Search

```go
// Find top-K nearest neighbors
results, err := store.Search(ctx, queryEmbedding, 10)
if err != nil {
    log.Fatal(err)
}

// Each result includes: ID, Distance (cosine similarity), Score
for _, result := range results {
    fmt.Printf("ID: %s, Distance: %.4f\n", result.ID, result.Distance)
}
```

### Rebuild Index

```go
// Full rebuild for optimization
err := store.Rebuild(ctx)
if err != nil {
    log.Fatal(err)
}
```

---

## Build & Test

### Build

```bash
cd mentle
go build ./storage/govector/...
```

### Test

```bash
cd mentle
GOSUMDB=off go test ./storage/govector/...
```

---

## Performance

| Operation | Target |
|---|---:|
| Add vector | P95 ≤ 5 ms |
| Search (top-10) | P95 ≤ 30 ms (warm index) |
| Index rebuild | O(n log n) with n vectors |

---

## Configuration

Tunable parameters:

| Parameter | Default | Purpose |
|---|---|---|
| M (HNSW) | 16 | max neighbors per node |
| ef_construction | 200 | construction quality |
| ef_search | 100 | search exploration |
| distance_metric | cosine | similarity measure |

---

## Testing Requirements

- Index persists vectors correctly
- Search results are deterministic
- Index rebuild produces equivalent results
- Corrupted index is recovered gracefully

---

## MANUAL

When updating:

1. HNSW parameters (M, ef) should be configurable
2. Distance metric should be swappable
3. Persistence format must be version-aware
4. Do not update for individual vector operations (use facade instead)

Parent reference: ../AGENTS.md
