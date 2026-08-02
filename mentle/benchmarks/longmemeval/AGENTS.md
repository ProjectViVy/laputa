<!-- Parent: ../../AGENTS.md -->

# mentle/benchmarks/longmemeval — LongMemEval Benchmark Harness

**Generated:** 2026-08-01  
**Purpose:** Implementation of LongMemEval (500-question benchmark for long-term memory retrieval)

---

## Purpose

LongMemEval is the canonical benchmark for Mentle's retrieval quality:

- **500 real-world questions** with temporal context
- **Top-50 retrieval** evaluation
- **Recall@5, Recall@10, NDCG@10** metrics
- **Session-aware retrieval** (questions reference sessions not in candidate pool)

---

## Structure

```
longmemeval/
├── longmemeval.go                   # Benchmark implementation
├── longmemeval_test.go              # Test suite
├── fixtures/                        # Test data
│   └── questions.json               # 500 questions
├── results/                         # Benchmark results
│   └── baseline.json                # Baseline results
└── (supporting utilities)
```

---

## Key Types

```go
type Entry struct {
    QuestionID         string
    Question           string
    Answer             interface{}
    AnswerSessionIDs   []string
    QuestionDate       string
    HaystackSessions   [][]interface{}
    HaystackSessionIDs []string
    HaystackDates      []string
}

type Result struct {
    Mode        string
    Granularity string
    TotalQ      int
    Recall5     float64
    Recall10    float64
    NDCG10      float64
    PerType     map[string]TypeResult
    PerK        map[int]KResult
}
```

---

## Run

```bash
cd mentle
go test -v ./benchmarks/longmemeval/... -bench=.
make bench-perf
```

---

## Baseline Results

Target performance (from phase 3):

| Metric | Score |
|--------|-------|
| Recall@5 | 92.8% |
| Recall@10 | 96.2% |
| NDCG@10 | 0.809 |

---

## Configuration

Set via environment:

```bash
LONGMEMEVAL_DATA_PATH=./fixtures/questions.json
LONGMEMEVAL_OUTPUT_PATH=./results/
LONGMEMEVAL_TOP_K=50
```

---

## MANUAL

When updating:
1. Keep 500-question dataset stable for consistency
2. Document any question type changes
3. Report results against baseline
4. Use make bench-perf for reproducible runs

Parent reference: ../../AGENTS.md
