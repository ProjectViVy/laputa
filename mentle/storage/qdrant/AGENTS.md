<!-- Parent: ../AGENTS.md -->

# mentle/storage/qdrant — Qdrant Vector Storage Backend

**Generated:** 2026-08-01  
**Purpose:** Alternative vector storage implementation (Qdrant integration)

---

## Purpose

The `qdrant/` subdirectory provides an alternative vector storage backend using Qdrant:

- **Qdrant client integration** — gRPC-based vector search engine
- **Vector persistence** — store and retrieve embeddings via Qdrant API
- **Distributed search** — supports multi-node Qdrant clusters
- **Compatibility layer** — implements mentle VectorStore interface
- **Optional backend** — can be selected at runtime or build time

---

## Status

Currently not active in production. Offered as pluggable alternative to govector HNSW.

---

## Structure

```
mentle/storage/qdrant/
└── (implementation files TBD)
    ├── store.go              # Qdrant vector store wrapper
    ├── store_test.go
    ├── config.go             # Qdrant connection config
    └── README.md             # Qdrant-specific documentation
```

---

## Key Concepts

### Qdrant Backend

Qdrant is a vector search engine with:

- **Written in Rust:** high performance, low memory footprint
- **gRPC API:** native gRPC support with Go client library
- **Point concept:** vectors stored as "points" with payload metadata
- **Collection model:** vectors organized into collections
- **Distributed ready:** supports clustering for horizontal scaling

### Integration Point

Qdrant backend implements the `VectorStore` interface:

```go
type VectorStore interface {
    Add(ctx context.Context, vectors []Vector) error
    Search(ctx context.Context, query []float32, k int) ([]Result, error)
    Delete(ctx context.Context, ids []string) error
    Close(ctx context.Context) error
}
```

### Trade-offs vs. govector

| Aspect | govector (HNSW) | Qdrant |
|--------|-----------------|--------|
| **Deployment** | in-process | standalone service |
| **Language** | Go-native | Rust (gRPC) |
| **Latency** | <50ms (local) | 30-150ms (network) |
| **Scalability** | single node | distributed (native) |
| **Persistence** | file-based | Qdrant server managed |
| **Query filtering** | post-search | pre-search payload filter |

---

## Build & Test

### Build

```bash
cd mentle
go build ./storage/qdrant/...
```

### Test

```bash
cd mentle
GOSUMDB=off go test ./storage/qdrant/...
```

### Integration Test (with Qdrant server)

```bash
# Start Qdrant server (e.g., Docker)
docker run -p 6333:6333 qdrant/qdrant:latest

# Run integration tests
QDRANT_URL=http://localhost:6333 go test ./storage/qdrant/ -tags integration -v
```

---

## Configuration

Qdrant backend is configured via environment variables or config struct:

```go
type QdrantConfig struct {
    Endpoint       string        // e.g., "localhost:6333"
    CollectionName string        // e.g., "mentle-vectors"
    VectorSize     int           // e.g., 384 (for all-MiniLM-L6-v2)
    Timeout        time.Duration // default 30s
}
```

---

## Performance

| Operation | Target |
|---|---:|
| Add vector | P95 ≤ 30ms (network + Qdrant) |
| Search (top-10) | P95 ≤ 80ms (network + Qdrant) |
| Delete | P95 ≤ 30ms (network + Qdrant) |
| Payload filter + search | P95 ≤ 100ms (pre-search filter) |

Actual latencies depend on Qdrant server health, network conditions, and collection size.

---

## Dependencies

- `qdrant/go-client` (Qdrant gRPC client for Go)
- Qdrant server instance (self-hosted or managed service)

---

## Architecture Boundaries

**Ownership rule (immutable):**

```text
mentle/storage may provide multiple backends.
Each backend implements VectorStore interface.
Mentle/facade chooses backend at initialization.
No backend may hold authority or policy.
```

---

## Advanced Features

### Payload Filtering

Qdrant supports filtering on metadata before computing distance:

```go
// Example: filter by scope before search
filter := &qdrant.Filter{
    Must: []*qdrant.Condition{{
        Field: "scope",
        Match: "architecture",
    }},
}
results, err := store.SearchWithFilter(ctx, query, k, filter)
```

### Point Versioning

Qdrant maintains versioned snapshots for point-in-time recovery:

```go
snapshot, err := store.Snapshot(ctx)
defer snapshot.Close()
results, err := store.SearchSnapshot(ctx, query, k, snapshot)
```

---

## MANUAL

When updating:

1. Maintain VectorStore interface compatibility
2. Document Qdrant-specific features (payload filtering, versioning)
3. Provide example Qdrant server deployment
4. Add integration test with containerized Qdrant
5. Document scaling considerations for multi-node clusters

Parent reference: ../AGENTS.md
