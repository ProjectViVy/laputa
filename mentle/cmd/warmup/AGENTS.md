<!-- Parent: ../../AGENTS.md -->

# mentle/cmd/warmup — Warm-up and Initialization Utility

**Generated:** 2026-08-01  
**Purpose:** Pre-initialize palace storage, embeddings, and indexes for performance profiling

---

## Purpose

The `warmup/` command performs warm-up operations:

- **Load ONNX embedding model** into memory
- **Pre-compute vector index** statistics
- **Initialize connection pools** to storage backends
- **Report timing** for performance profiling

---

## Structure

```
warmup/
├── main.go                          # Warm-up logic
└── (supporting utilities)
```

---

## Build

```bash
cd mentle/cmd/warmup
go build -o warmup .
```

---

## Run

```bash
./warmup --palace ~/.my-palace
```

**Output:**

```
Warm-up Report:
  Model load: 234ms
  Index prep: 45ms
  Storage init: 12ms
  Total: 291ms
```

---

## Use Cases

- **Benchmarking** — establish baseline before running benchmarks
- **Deployment** — pre-warm before accepting production traffic
- **Development** — measure initialization overhead

---

## MANUAL

Warm-up results should be compared before and after optimization changes.

Parent reference: ../AGENTS.md
