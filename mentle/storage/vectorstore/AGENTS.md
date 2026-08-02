<!-- Parent: ../../AGENTS.md -->

# mentle/storage/vectorstore — Vector Store Interface

**Generated:** 2026-08-01  
**Purpose:** Public interface for vector storage backends

---

## Purpose

The `vectorstore/` package defines the abstract interface:

- **Pluggable backends** — govector, qdrant, chroma, etc.
- **Common operations** — insert, search, delete
- **Result normalization** — consistent scoring across backends

---

## Structure

```
vectorstore/
├── interface.go                     # VectorStore interface definition
├── types.go                         # Result types
└── (supporting utilities)
```

---

## Interface

```go
type VectorStore interface {
    Insert(ctx context.Context, id string, vector []float32) error
    Search(ctx context.Context, query []float32, k int) ([]SearchResult, error)
    Delete(ctx context.Context, id string) error
    Close(ctx context.Context) error
}

type SearchResult struct {
    ID        string
    Score     float64  // normalized 0-1
    Distance  float64
}
```

---

## Implementing Backends

Each backend (govector, redis, etc.) implements this interface.

See parent [mentle/storage/AGENTS.md](../AGENTS.md) for usage.

---

## Build & Test

```bash
cd mentle
GOSUMDB=off go test ./storage/vectorstore/...
```

---

## MANUAL

Interface is stable. Adding methods requires coordinated backend updates.

Parent reference: ../AGENTS.md
