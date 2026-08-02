<!-- Parent: ../../AGENTS.md -->

# mentle/internal/bm25 — BM25 Lexical Ranking

**Generated:** 2026-08-01  
**Purpose:** Lexical search scoring using BM25 algorithm

---

## Purpose

The `bm25/` package implements BM25 ranking:

- **Term frequency (TF)** — how often a term appears in a document
- **Inverse document frequency (IDF)** — how rare a term is across corpus
- **Document length normalization** — adjust for document size
- **Configurable parameters** — K1 and b for tuning

---

## Structure

```
bm25/
├── bm25.go           # BM25 scorer implementation
└── bm25_test.go
```

---

## Key Concepts

### BM25 Formula

```
score = IDF(term) * (TF * (K1 + 1)) / (TF + K1 * (1 - b + b * (doc_len / avg_doc_len)))
```

Where:
- **K1** — tuning parameter (default: 1.5, higher = more term frequency weight)
- **b** — document length normalization (default: 0.75, 0.0 = no normalization)
- **IDF** — log((N - df + 0.5) / (df + 0.5))

### Scorer Interface

```go
type Scorer struct {
    K1              float64
    B               float64
    AvgDocLength    float64
    CorpusStats     map[string]int  // term -> document frequency
}

func (s *Scorer) Score(term string, termFreq, docLen int) float64
```

### Usage

```go
scorer := bm25.NewScorer(1.5, 0.75)
score := scorer.Score("authentication", 3, 1024)
```

---

## Testing

```bash
cd mentle
GOSUMDB=off go test -v ./internal/bm25/...
```

**Behavioral tests:**

- Scoring is deterministic
- Higher TF increases score
- Lower IDF (common terms) decreases score
- Length normalization prevents long docs from dominating
- K1 and b parameters tune correctly

---

## Conventions

- All scores are non-negative
- Corpus stats are frozen after initialization
- Scores are normalized to 0.0-1.0 range
- Parameters K1 and b have reasonable defaults

---

## MANUAL

Keep BM25 focused on scoring math. Ranking aggregation goes to search package.

Parent reference: ../AGENTS.md
