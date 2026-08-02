<!-- Parent: ../../AGENTS.md -->

# mentle/benchmarks/membench — Memory Benchmark Suite

**Generated:** 2026-08-01  
**Purpose:** General-purpose memory benchmark harness for performance testing

---

## Purpose

The `membench` harness provides benchmarking infrastructure:

- **Micro-benchmarks** for individual operations
- **Macro-benchmarks** for full retrieval pipelines
- **Load testing** under concurrent workloads
- **Memory profiling** (allocation, GC pressure)

---

## Structure

```
membench/
├── membench.go                      # Core benchmark harness
├── membench_test.go                 # Test cases
├── workloads/                       # Standard workload definitions
└── (supporting utilities)
```

---

## Run

```bash
cd mentle
go test -v ./benchmarks/membench/... -bench=.
make bench-perf
```

---

## Standard Workloads

- **Vector search** — 10K queries
- **Embedding generation** — 1K documents
- **Knowledge graph traversal** — 5K entity chains
- **Hybrid search** — semantic + lexical fusion
- **Concurrent retrieval** — 10 concurrent clients

---

## Metrics

Reports:
- P50, P95, P99 latency
- Throughput (ops/sec)
- Memory allocation (bytes)
- GC pause time

---

## MANUAL

When updating:
1. Add new workloads as subdirectories
2. Keep standard workloads stable for comparison
3. Document any methodology changes

Parent reference: ../../AGENTS.md
