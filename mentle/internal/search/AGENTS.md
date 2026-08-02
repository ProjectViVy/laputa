<!-- Parent: ../../AGENTS.md -->

# mentle/internal/search — Query Processing & Ranking

**Generated:** 2026-08-01  
**Purpose:** Search query processing, tokenization, and result ranking

---

## Purpose

The `search/` package handles query processing and ranking:

- **Query tokenization** — normalize and split search queries
- **Synonym expansion** — optional semantic term expansion
- **BM25 scoring** — lexical relevance ranking
- **Policy filtering** — remove denied sources from results
- **Deduplication** — merge duplicate results
- **Ranking** — combine scores (BM25, recency, heat)

---

## Structure

```
search/
├── searcher.go       # Main search coordinator
├── ranker.go         # Result ranking and scoring
└── (supporting utilities)
```

---

## Key Concepts

### Search Request

```go
type SearchRequest struct {
    Query      string
    Scope      string
    Wings      []string
    Rooms      []string
    MaxResults int
    Budget     int        // character budget for results
}
```

### Search Result

```go
type SearchResult struct {
    Cards    []MemoryCard
    TotalHits int
    Duration time.Duration
}
```

### Scoring

Combines multiple signals:
1. **BM25** — lexical relevance (30%)
2. **Recency** — last activated time (20%)
3. **Heat** — activation frequency (20%)
4. **Scope** — exact scope match bonus (20%)
5. **Freshness** — recently created bonus (10%)

---

## Testing

```bash
cd mentle
GOSUMDB=off go test -v ./internal/search/...
```

**Behavioral tests:**

- Queries tokenize correctly
- Ranking is deterministic
- Policy filtering works
- Results respect budget
- Dedup removes duplicates

---

## Conventions

- Query normalization is case-insensitive
- Stopwords are filtered
- Scoring is deterministic
- All results are sorted by final score

---

## MANUAL

Keep search focused on query processing. Card retrieval goes to facade.

Parent reference: ../AGENTS.md
