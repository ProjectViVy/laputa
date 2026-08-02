<!-- Parent: ../../AGENTS.md -->

# mentle/benchmarks/convomem — Conversation Memory Benchmark

**Generated:** 2026-08-01  
**Purpose:** Benchmark Mentle's retrieval performance on conversation-based queries

---

## Purpose

The `convomem` benchmark harness evaluates:

- **Retrieval accuracy** on multi-turn conversations
- **Vector search latency** under conversation workloads
- **Hybrid search fusion** effectiveness
- **Recall and NDCG** metrics

---

## Structure

```
convomem/
├── convomem.go                      # Benchmark harness
├── convomem_test.go                 # Test suite
└── (supporting utilities)
```

---

## Run

```bash
cd mentle
go test -v ./benchmarks/convomem/... -bench=.
```

---

## Metrics

Reports:
- Recall@5, Recall@10
- NDCG@10
- P95 latency
- False positive rate

---

## MANUAL

When updating:
1. Keep benchmark harness stable
2. Document new metrics
3. Compare against baseline (LongMemEval)

Parent reference: ../../AGENTS.md
