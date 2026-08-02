<!-- Parent: ../../AGENTS.md -->

# mentle/examples/hybrid_search — Hybrid Search Example

**Generated:** 2026-08-01  
**Purpose:** Runnable example demonstrating hybrid search (semantic + lexical) in Mentle

---

## Purpose

This example shows how to use Mentle's hybrid search:

- **Semantic search** — vector similarity
- **Lexical search** — BM25 ranking
- **Fusion** — combining scores
- **Temporal filtering** — knowledge graph constraints

---

## Structure

```
hybrid_search/
├── main.go                          # Example entry point
├── fixtures/                        # Sample data
│   ├── documents.json               # Test documents
│   └── queries.json                 # Test queries
└── (supporting code)
```

---

## Build

```bash
cd mentle/examples/hybrid_search
go build -o hybrid_search .
```

---

## Run

```bash
./hybrid_search
```

**Output:** Demonstrates hybrid search results with semantic and lexical scores.

---

## Key Concepts Illustrated

1. **Initialize palace** with vector storage and knowledge graph
2. **Index documents** with embeddings
3. **Execute semantic search** (vector similarity)
4. **Execute lexical search** (BM25)
5. **Fuse results** using both rankers
6. **Apply temporal filters** from KG

---

## Code Structure

- Load sample documents
- Create palace instance
- Index into vector store and KG
- Run hybrid search
- Display results with scores

---

## MANUAL

Keep this example simple and well-commented. Update when hybrid search algorithm changes.

Parent reference: ../../AGENTS.md
