<!-- Parent: ../../AGENTS.md -->

# mentle/internal/hybrid — Hybrid Search (Lexical + Vector)

**Generated:** 2026-08-01  
**Purpose:** Combine BM25 lexical and vector semantic search

---

## Purpose

The `hybrid/` package combines search modalities:

- **Lexical search** — BM25 keyword matching
- **Vector search** — semantic similarity via embeddings
- **Score normalization** — combine heterogeneous scores
- **Reranking** — apply governance policy to results
- **Budget enforcement** — respect character and result limits

---

## Structure

```
hybrid/
├── searcher.go       # Hybrid search coordinator
└── (supporting utilities)
```

---

## Key Concepts

### Hybrid Search

Two-phase approach:

**Phase 1: Candidate Selection**
- Run BM25 search (lexical)
- Run vector search (semantic)
- Merge candidate sets

**Phase 2: Ranking**
- Normalize scores (0.0-1.0)
- Weight combinations (default: 0.5 lexical + 0.5 vector)
- Apply policy filters
- Rank final results

### Configuration

```go
type HybridConfig struct {
    LexicalWeight float64 // default: 0.5
    VectorWeight  float64 // default: 0.5
    VectorModel   string  // embedding model name
    TopK          int     // initial candidates before rerank
}
```

---

## Testing

```bash
cd mentle
GOSUMDB=off go test -v ./internal/hybrid/...
```

**Behavioral tests:**

- Lexical and vector results merge
- Score normalization works
- Weighting adjusts ranking
- Policy filtering applies
- Budget respected

---

## Conventions

- Weights sum to 1.0 (normalized)
- All scores normalized to 0.0-1.0
- Deterministic ranking for same query

---

## MANUAL

Keep hybrid focused on search orchestration. Individual search modes go elsewhere.

Parent reference: ../AGENTS.md
