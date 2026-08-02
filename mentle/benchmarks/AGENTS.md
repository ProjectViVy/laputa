<!-- Parent: ../AGENTS.md -->

# mentle/benchmarks — LongMemEval Benchmark Harness

**Generated:** 2026-08-01  
**Purpose:** Memory retrieval evaluation and performance metrics

---

## Purpose

The `benchmarks/` directory contains evaluation tools for Mentle's retrieval performance:

- **LongMemEval benchmark** — standard evaluation harness for memory retrieval systems
- **Metrics computation** — Recall@K, NDCG, F1 score calculation
- **Session-level evaluation** — measure retrieval across multi-turn sessions
- **Performance profiling** — latency and throughput measurement

LongMemEval is a published benchmark for memory systems; Mentle targets top-50 retrieval performance.

---

## Structure

```
benchmarks/
├── longmemeval/
│   ├── dataset.go                 # LongMemEval dataset loading
│   ├── runner.go                  # Benchmark execution
│   ├── dataset_test.go
│   └── runner_test.go
├── metrics.go                     # Metric computation (Recall, NDCG, F1)
├── metrics_test.go
└── profiles/                      # Saved benchmark results
```

---

## Metrics

### Recall@K

Fraction of relevant items appearing in top-K results:

```
Recall@5 = (items_found_in_top5) / (total_relevant_items)
```

Mentle target: **Recall@5 = 92%+**

### NDCG@K

Normalized Discounted Cumulative Gain with binary relevance:

```
NDCG@10 = DCG@10 / IDCG@10
```

Penalizes incorrect ranking of relevant items.

Mentle target: **NDCG@10 = 0.80+**

### F1 Score

Token-level overlap between retrieved and ground-truth text:

```
F1 = 2 * (precision * recall) / (precision + recall)
```

---

## Running Benchmarks

### Full Benchmark

```bash
cd mentle
make bench-perf
```

### With Profiling

```bash
go test -bench=. -cpuprofile=cpu.prof -memprofile=mem.prof ./benchmarks/
go tool pprof cpu.prof
```

### Specific Test

```bash
go test -v ./benchmarks/ -run TestLongMemEval
```

---

## Output

Benchmark results (example):

```
BenchmarkLongMemEval/recall@5
  Recall@5:  92.8%
  Recall@10: 96.2%
  NDCG@10:   0.809
  Latency (P95): 78ms
```

Results are saved to `profiles/` for historical tracking.

---

## Build & Test

### Build

```bash
cd mentle
go build ./benchmarks/...
```

### Test

```bash
cd mentle
GOSUMDB=off go test ./benchmarks/...
```

---

## Performance Targets

| Metric | Target |
|---|---:|
| Recall@5 | ≥ 92% |
| Recall@10 | ≥ 96% |
| NDCG@10 | ≥ 0.80 |
| Latency (P95) | ≤ 80ms |

---

## Testing Requirements

Before releases:

```bash
cd mentle && make bench-perf
```

Check that:
- [ ] Recall@5 ≥ 92%
- [ ] Recall@10 ≥ 96%
- [ ] NDCG@10 ≥ 0.80
- [ ] P95 latency ≤ 80ms

---

## Conventions

- **Go formatting:** standard `gofmt`
- **Metrics:** always reported as fractions (0.0 to 1.0) or percentages
- **Latency:** reported in milliseconds, P95 and P99 latiles
- **Dataset:** LongMemEval v1 (500 questions, standard corpus)

---

## MANUAL

When updating:

1. Keep benchmark dataset stable for historical comparison
2. Document new metrics before including them
3. Save results with timestamps for trend analysis
4. Update when:
   - New metrics are defined
   - Performance targets change
   - Benchmark infrastructure updates
5. Do not update for:
   - Individual algorithm tweaks (benchmark before/after)

Parent reference: ../AGENTS.md
