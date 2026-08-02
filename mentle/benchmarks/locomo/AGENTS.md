<!-- Parent: ../../AGENTS.md -->

# mentle/benchmarks/locomo — Long Context Memory Benchmark

**Generated:** 2026-08-01  
**Purpose:** Benchmark retrieval accuracy on long-context and temporal memory scenarios

---

## Purpose

The `locomo` benchmark evaluates:

- **Long-context retrieval** with multi-session memory
- **Temporal reasoning** using knowledge graph
- **Degrade performance** when temporal filters are applied
- **Deep recall** vs Fast recall

---

## Structure

```
locomo/
├── locomo.go                        # Benchmark implementation
├── locomo_test.go                   # Test cases
└── (scenarios and fixtures)
```

---

## Run

```bash
cd mentle
go test -v ./benchmarks/locomo/... -bench=.
```

---

## Metrics

Reports:
- Temporal precision (correct sessions retrieved)
- Cross-session recall
- Degradation impact on latency
- KG query efficiency

---

## MANUAL

When updating:
1. Use realistic temporal scenarios
2. Compare with baseline memory systems
3. Document edge cases

Parent reference: ../../AGENTS.md
