<!-- Parent: ../AGENTS.md -->

# mentle/storage/chroma — Chroma Vector Storage Backend

**Generated:** 2026-08-01  
**Purpose:** Alternative vector storage implementation (Chroma integration)

---

## Purpose

The `chroma/` subdirectory provides an alternative vector storage backend using Chroma:

- **Chroma client integration** — Python or gRPC-based vector database
- **Vector persistence** — store and retrieve embeddings via Chroma API
- **Compatibility layer** — implements mentle VectorStore interface
- **Optional backend** — can be selected at runtime or build time

---

## Status

Currently not active in production. Offered as pluggable alternative to govector HNSW.

---

## Structure

```
mentle/storage/chroma/
└── (implementation files TBD)
    ├── store.go              # Chroma vector store wrapper
    ├── store_test.go
    ├── config.go             # Chroma connection config
    └── README.md             # Chroma-specific documentation
```

---

## Key Concepts

### Chroma Backend

Chroma is a vector database with:

- **Python-first:** original implementation in Python
- **gRPC API:** language-agnostic access from Go
- **Collection model:** organize vectors into named collections
- **Metadata filtering:** filter results by metadata before distance computation

### Integration Point

Chroma backend implements the `VectorStore` interface:

```go
type VectorStore interface {
    Add(ctx context.Context, vectors []Vector) error
    Search(ctx context.Context, query []float32, k int) ([]Result, error)
    Delete(ctx context.Context, ids []string) error
    Close(ctx context.Context) error
}
```

### Trade-offs vs. govector

| Aspect | govector (HNSW) | Chroma |
|--------|-----------------|--------|
| **Deployment** | in-process | requires daemon/service |
| **Dependencies** | minimal | Python + gRPC |
| **Latency** | <50ms (local) | 50-200ms (network) |
| **Scalability** | single node | distributed (optional) |
| **Persistence** | file-based | Chroma server managed |

---

## Build & Test

### Build

```bash
cd mentle
go build ./storage/chroma/...
```

### Test

```bash
cd mentle
GOSUMDB=off go test ./storage/chroma/...
```

### Integration Test (with Chroma server)

```bash
# Start Chroma server (e.g., Docker)
docker run -p 8000:8000 ghcr.io/chroma-core/chroma:latest

# Run integration tests
CHROMA_API_URL=http://localhost:8000 go test ./storage/chroma/ -tags integration -v
```

---

## Configuration

Chroma backend is configured via environment variables or config struct:

```go
type ChromaConfig struct {
    APIEndpoint string        // e.g., "http://localhost:8000"
    CollectionName string     // e.g., "mentle-vectors"
    Timeout time.Duration     // default 30s
}
```

---

## Performance

| Operation | Target |
|---|---:|
| Add vector | P95 ≤ 50ms (network + Chroma) |
| Search (top-10) | P95 ≤ 100ms (network + Chroma) |
| Delete | P95 ≤ 50ms (network + Chroma) |

Actual latencies depend on Chroma server health and network conditions.

---

## Dependencies

- `chroma-core/chroma` (Python service or language-specific client)
- gRPC client library for Go (if using gRPC API)

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

## MANUAL

When updating:

1. Maintain VectorStore interface compatibility
2. Document performance trade-offs vs. govector
3. Provide example configuration and startup commands
4. Add integration test with containerized Chroma

Parent reference: ../AGENTS.md
