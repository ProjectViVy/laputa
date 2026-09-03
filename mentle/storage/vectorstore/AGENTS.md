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

```text
vectorstore/
└── vectorstore.go                     # Store interface, BackendType constants, Open() factory
```

The package is a single file. It defines the unified `Store` interface plus an `Open(cfg)` factory that dispatches by `BackendType`. It does **not** declare `interface.go`/`types.go` (those names are historical documentation only).

## Interface

The canonical interface is in `vectorstore.go` (see source for the authoritative definition):

```go
type Store interface {
    Search(ctx, query []float32, limit int, filter map[string]any) ([]SearchResult, error)
    Add(ctx, id string, vector []float32, payload map[string]any) error
    AddBatch(ctx, points []Point) error
    Delete(ctx, id string) error
    ListAll(ctx, limit int) ([]SearchResult, error)
    Close() error
}

type BackendType string

const (
    BackendGoVector BackendType = "govector"   // active in production (facade)
    BackendRedis    BackendType = "redis"
    BackendQdrant   BackendType = "qdrant"     // NOT implemented — Open() returns error
    BackendChroma   BackendType = "chroma"     // NOT implemented — Open() returns error
    BackendLanceDB  BackendType = "lancedb"    // NOT implemented — Open() returns error
)
```

## Implementing Backends

Only **govector** implements this interface today, and it does so via `storage/govector.Store` (constructed directly by `facade` — `vectorstore.Open(BackendGoVector)` deliberately tells you to use `govector.NewStore` directly). Redis has its own `palace.Drawer`-based API in `storage/redis` and is not wired through this interface. Qdrant/Chroma/LanceDB have no implementation.

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
