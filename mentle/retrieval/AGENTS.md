<!-- Parent: ../AGENTS.md -->

# mentle/retrieval — Vector Search & BM25 Ranking

**Generated:** 2026-08-01  
**Purpose:** Semantic search, lexical ranking, and hybrid retrieval

---

## Purpose

The `retrieval/` package implements hybrid search combining semantic vectors and lexical ranking:

- **Vector search** using ONNX embeddings (hugot, no external daemons)
- **BM25 lexical ranking** for keyword-based retrieval
- **Hybrid fusion** combining both scores with configurable weights
- **Temporal filtering** with knowledge graph validity checks
- **Result deduplication** and ranking by combined score

Hybrid search powers both Fast Recall (deterministic ranking) and Deep Recall (with LLM reranking).

---

## Structure

```
retrieval/
├── vector.go                      # Vector search interface
├── vector_test.go
├── bm25.go                        # BM25 ranking implementation
├── bm25_test.go
├── hybrid.go                      # Hybrid fusion and ranking
├── hybrid_test.go
├── ranker.go                      # Result ranking and deduplication
├── ranker_test.go
└── types.go                       # Common types and interfaces
```

### Subdirectories (Depth 3)

No formal depth-3 AGENTS.md required; implementations are self-contained.

---

## Search Modes

### Vector Search (Semantic)

Query embedding is compared to stored embeddings using cosine similarity:

```go
// Embed query
queryVec, err := embedder.Embed(ctx, query)

// Search in HNSW index
results, err := vectorStore.Search(ctx, queryVec, topK)

// Results ranked by cosine similarity (0.0 to 1.0, higher is better)
```

Model: `sentence-transformers/all-MiniLM-L6-v2` (384-dimensional)  
Index: HNSW (Hierarchical Navigable Small World) via govector  
Truncation: 400 Unicode code-points max per query

### BM25 (Lexical)

Keyword-based ranking using TF-IDF:

```go
// Tokenize query and documents
tokens := bm25.Tokenize(query)

// Compute BM25 score for each document
score, err := bm25.Score(ctx, documentID, tokens)

// Results ranked by BM25 score
```

Implementation: pure Go, no external libraries  
Corpus statistics: stored in database for score computation

### Hybrid Fusion

Combine both scores with weighted average:

```go
// Normalize both scores to [0, 1]
vectorScore := normalizeVectorScore(vectorRawScore)
bm25Score := normalizeBM25Score(bm25RawScore)

// Weighted combination
hybridScore = alpha * vectorScore + (1 - alpha) * bm25Score

// Default alpha = 0.7 (favor semantic)
```

---

## Ranking

### Deduplication

Multiple results may reference the same memory card (via different fragments):

```go
// Group by card ID
// Keep highest-scoring result per card
deduped := deduplicateByCard(results)
```

### Temporal Filtering

Filter out expired or out-of-validity-window results:

```go
if result.ValidTo != nil && time.Now().After(*result.ValidTo) {
    continue  // Expired
}
if time.Now().Before(result.ValidFrom) {
    continue  // Not yet valid
}
```

### Final Ranking

Final results ranked by:

1. Hybrid score (primary)
2. Heat score (recency/frequency, tiebreaker)
3. Card ID (deterministic tiebreaker)

---

## Build & Test

### Build

```bash
cd mentle
go build ./retrieval/...
```

### Test

```bash
cd mentle
GOSUMDB=off go test ./retrieval/...
```

### Test Specific

```bash
GOSUMDB=off go test -v ./retrieval/ -run TestHybridSearch
GOSUMDB=off go test -v ./retrieval/ -run TestBM25Ranking
GOSUMDB=off go test -v ./retrieval/ -run TestDeduplication
```

---

## Performance Targets

| Operation | Target |
|---|---:|
| Vector embedding (batch) | P95 ≤ 50 ms |
| Vector search (HNSW) | P95 ≤ 30 ms |
| BM25 ranking (top-K) | P95 ≤ 10 ms |
| Hybrid fusion | P95 ≤ 5 ms |
| Total retrieval | P95 ≤ 80 ms |

---

## Configuration

Hybrid parameters (configurable):

```go
type HybridConfig struct {
    VectorWeight   float64  // default 0.7
    BM25Weight     float64  // default 0.3
    MaxResults     int      // default 50
    DedupByCard    bool     // default true
    TemporalFilter bool     // default true
}
```

---

## Testing Requirements

Before starting feature work:

```bash
cd mentle && GOSUMDB=off go test ./retrieval/...
```

**Mandatory behavioral tests:**

- Vector search returns consistent results (deterministic)
- BM25 scores are stable
- Hybrid fusion respects weight parameters
- Deduplication keeps highest-scoring result per card
- Temporal filtering excludes expired/not-yet-valid results
- Query truncation at 400 runes does not crash embedding

---

## Integration with Facade

Facade uses retrieval for SearchCards:

```go
// In mentle/facade/cards.go
results, err := retrieval.HybridSearch(ctx, query, scope, maxResults)
cards := convertToCardCandidates(results)
```

---

## Conventions

- **Go formatting:** standard `gofmt`
- **Scoring:** normalized to [0, 1] for comparison
- **Timestamps:** UTC, ISO 8601 format
- **Character truncation:** Unicode code-points, not bytes
- **Determinism:** same query always produces same ranking (except heat decay)

---

## MANUAL

When updating:

1. Keep search deterministic where possible
2. Hybrid weights should be configurable, not hardcoded
3. Temporal filtering must be enforced consistently
4. Update when:
   - Search algorithm changes
   - Hybrid weights are tuned
   - Deduplication rules evolve
5. Do not update for:
   - Individual index optimization (use internal AGENTS.md)

Parent reference: ../AGENTS.md
