<!-- Parent: ../../AGENTS.md -->

# mentle/internal/embedder — Embedding Generation

**Generated:** 2026-08-01  
**Purpose:** Generate vector embeddings for semantic search

---

## Purpose

The `embedder/` package generates embeddings:

- **ONNX-based inference** — local embedding models via hugot
- **Batch processing** — efficient vectorization of multiple texts
- **Caching** — avoid recomputing embeddings for same input
- **Model management** — download and cache models locally
- **Deterministic** — same input always produces same embedding

---

## Structure

```
embedder/
├── hugot.go          # ONNX embedding implementation
└── hugot_test.go
```

---

## Key Concepts

### Embedder Interface

```go
type Embedder interface {
    Embed(ctx context.Context, text string) ([]float32, error)
    EmbedBatch(ctx context.Context, texts []string) ([][]float32, error)
    Dimension() int
    ModelName() string
}
```

### ONNX Embedder

Uses hugot to load and run ONNX models:
- Model path: `models/onnx/` directory
- Auto-download if missing
- Dimension: model-dependent (default: 384)
- Deterministic results

### Usage

```go
embedder, err := hugot.New("all-MiniLM-L6-v2")
if err != nil {
    log.Fatal(err)
}

embedding, err := embedder.Embed(ctx, "hello world")
if err != nil {
    log.Fatal(err)
}
```

---

## Testing

```bash
cd mentle
GOSUMDB=off go test -v ./internal/embedder/...
```

**Behavioral tests:**

- Embedding dimension is consistent
- Deterministic (same input = same output)
- Batch processing works
- Model loading works

---

## Conventions

- All embeddings are float32
- Dimension is fixed per model
- No mutation of models
- Models are read-only after load

---

## MANUAL

Keep embedder focused on inference. Storage and indexing go to vector or storage packages.

Parent reference: ../AGENTS.md
