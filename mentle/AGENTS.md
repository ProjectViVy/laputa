<!-- Parent: ../AGENTS.md -->

# Mentle — Material & Retrieval Module

**Generated:** 2026-08-01  
**Module Path:** `github.com/dashimaki/mentle`  
**Go Version:** 1.26.4 or later  
**Status:** Phase 3+ active — hybrid search, knowledge graph, WAL durability

---

## Purpose

Mentle is a Go implementation of MemPalace — a memory system for AI assistants. It provides:

- **Vector search** using ONNX embeddings (hugot, no external daemons)
- **Knowledge graph** with SQLite-based entity relationship tracking
- **WAL-based storage** for durable memory operations
- **MCP server** for AI client integration (Claude Desktop, Cursor, etc.)
- **Hybrid search** combining semantic and lexical retrieval (BM25)
- **Taxonomy** organized as palace wings and rooms

All memory operations are portable; single binary, no Python dependencies.

---

## Module Structure

```
mentle/
├── .git/                         # Mentle submodule (independent repo)
├── benchmarks/                   # LongMemEval benchmark harness
│   ├── longmemeval/             # Session retrieval benchmark
│   └── metrics.go               # Performance metrics
├── cmd/
│   └── cli/                      # CLI command implementations
│       ├── main.go               # Root command (init, mine, search, etc.)
│       ├── mcp.go                # MCP server command
│       ├── split.go              # Data splitting command
│       ├── bench.go              # Benchmark command
│       ├── compress.go           # Compression command
│       └── hook.go               # Hook management
├── docs/                         # Internal documentation
│   ├── hybrid-search.md          # Search algorithm design
│   └── superpowers/              # Design specs and plans
├── examples/                     # Usage examples (if present)
├── integration/                  # Integration tests
│   ├── cli_test.go
│   ├── mcp_test.go
│   └── llama.log                 # Integration log
├── internal/
│   ├── bm25/                     # BM25 lexical ranking
│   ├── config/                   # Configuration management
│   ├── diary/                    # AAAK diary (agent-specific entries)
│   ├── dialect/                  # Text dialect handling
│   ├── entity/                   # Entity detection
│   ├── extractor/                # Memory extraction
│   ├── instructions/             # Usage instructions
│   ├── kg/                        # Knowledge graph (temporal RDF)
│   ├── miner/                    # Project and conversation mining
│   ├── palace/                   # Palace structure (wings, rooms, drawers)
│   ├── registry/                 # Memory registry
│   ├── room/                      # Room detection and categorization
│   ├── sanitizer/                # Input sanitization
│   └── search/                   # Semantic search
├── models/
│   └── onnx/                     # ONNX embedding model files
│       ├── model.onnx            # all-MiniLM-L6-v2 (384-dim)
│       ├── config.json
│       ├── tokenizer.json
│       ├── special_tokens_map.json
│       └── tokenizer_config.json
├── pkg/
│   ├── mcp/                      # MCP server implementation
│   │   ├── protocol.go           # JSON-RPC protocol
│   │   ├── server_test.go
│   │   └── tools.go              # MCP tool definitions
│   └── wal/                      # Write-ahead log
│       ├── wal.go
│       └── wal_test.go
├── storage/
│   ├── govector/                 # Vector storage backend (HNSW)
│   │   ├── store.go
│   │   └── store_test.go
│   └── vectorstore/              # Vector store interface
├── Makefile                      # Build targets (build, test, bench-perf, etc.)
├── go.mod                        # Module: github.com/dashimaki/mentle
├── go.sum
├── README.md
├── LICENSE
├── .gitignore
├── run_integration_tests.sh
└── integration.test              # Integration test binary
```

### Subdirectories (Depth 2)

Depth-2 AGENTS.md files exist for:

- **[internal/AGENTS.md](./internal/AGENTS.md)** — search, KG, palace, mining, entity, dialect
- **[pkg/AGENTS.md](./pkg/AGENTS.md)** — MCP server, WAL, protocol
- **[storage/AGENTS.md](./storage/AGENTS.md)** — govector, vector store backend
- **[cmd/AGENTS.md](./cmd/AGENTS.md)** — CLI commands and MCP server entry points
- **[benchmarks/AGENTS.md](./benchmarks/AGENTS.md)** — LongMemEval benchmark harness

---

## Key Concepts

### Palace Structure

Memory is organized hierarchically:

- **Wing** — category or context (e.g., "architecture", "debugging", "temporal-reasoning")
- **Room** — subcategory or entity (e.g., a specific design decision or agent diary)
- **Drawer** — individual memory items (embeddings, metadata, references)

### Vector Search

- **Model:** `sentence-transformers/all-MiniLM-L6-v2` (384-dimensional)
- **Runtime:** hugot (ONNX native Go, no llamafile)
- **Index:** HNSW (Hierarchical Navigable Small World)
- **Truncation:** 400 runes max to avoid tokenization explosion

### Hybrid Search

Combines:

1. **Semantic search** — vector similarity (cosine)
2. **Lexical search** — BM25 ranking
3. **Temporal filtering** — knowledge graph validity

### Knowledge Graph

Temporal RDF-style triples with SQLite backend:

```
subject -> predicate -> object
valid_from: time
valid_to: time (nullable)
confidence: float
```

Supports:

- Entity timelines
- Temporal reasoning
- Fact invalidation
- Relationship traversal

### WAL (Write-Ahead Log)

Durability via write-ahead log in `wal/` directory. All mutations are logged before applied.

### AAAK Dialect

Structured symbolic summaries for agent diaries:

- **Entity Codes:** 3-letter uppercase (KAI, MAX, PRI)
- **Topics:** frequency-based with proper noun boosting
- **Emotion Codes:** vul, joy, fear, trust, grief, wonder, rage, etc.
- **Flag Codes:** DECISION, ORIGIN, CORE, PIVOT, TECHNICAL

---

## Build & Test

### Build (Pure Go)

```bash
cd mentle
go mod tidy
go build ./...
make build
```

### Build (ORT + CoreML, Apple Silicon)

Requires `libtokenizers.a`:

```bash
mkdir -p ~/lib
curl -fSL https://github.com/daulet/tokenizers/releases/download/v1.26.0/libtokenizers.darwin-aarch64.tar.gz \
    | tar -xz -C ~/lib/

export CGO_LDFLAGS="-L${HOME}/lib"
make build-ort
```

### Test

```bash
cd mentle
GOSUMDB=off go test ./...
make test
```

### Integration Tests

```bash
./run_integration_tests.sh
go test -v ./integration/ -run TestMCP
```

### Benchmarks

```bash
make bench-perf                 # Pure Go benchmark
make bench-perf-ort             # ORT + CoreML benchmark
go test -bench=. ./benchmarks/
```

---

## Performance Targets

| Operation | Target |
|---|---:|
| warm vector search (P95) | ≤ 80 ms |
| BM25 lexical filter (P95) | ≤ 10 ms |
| bounded evidence read (P95) | ≤ 40 ms |
| knowledge graph query (P95) | ≤ 30 ms |

### LongMemEval Results (500 questions, top-50 retrieval)

| Metric | Score |
|--------|-------|
| Recall@5 | 92.8% |
| Recall@10 | 96.2% |
| NDCG@10 | 0.809 |

---

## Dependencies

### External Packages (Mentle-specific)

| Package | Use | License |
|---------|-----|---------|
| `DotNetAge/govector` | Vector storage with HNSW | MIT |
| `knights-analytics/hugot` | ONNX embedding runtime | Apache 2.0 |
| `gomlx/gomlx` + `gomlx/onnx-gomlx` | ML execution engine | Apache 2.0 |
| `spf13/cobra` | CLI framework | Apache 2.0 |
| `spf13/viper` | Configuration management | MIT |
| `redis/go-redis/v9` | Cache/async indexing | BSD-2 |

### Shared (via parent garden)

- `google/uuid` — UUID generation
- `go.yaml.in/yaml/v3` — YAML parsing
- `mattn/go-sqlite3` — SQLite3 driver

---

## Testing Requirements

Before starting any feature work:

```bash
cd mentle && GOSUMDB=off go test ./facade/...
```

**Exit gates:**

- SearchCards does not return full `Content`
- ReadEvidence enforces per-item and total character budget
- Knowledge graph temporal queries are consistent
- Vector search is deterministic and repeatable
- WAL recovery produces bit-identical state
- Integration tests pass with MCP protocol

---

## Architecture Boundaries

**Ownership rule (immutable):**

```text
Mentle may store and retrieve.
Mentle may not promote authority.
Mentle may not approve policy.
Mentle may not hold governance state.
```

---

## Conventions

- **Go formatting:** standard `gofmt`
- **Embedding truncation:** 400 runes (Unicode code-points) max
- **Batch size:** 64 texts per hugot `RunPipeline` invocation
- **Concurrency:** worker pools for embedding; file-based locking for palace
- **Configuration:** `~/.mempalace/config.json` (configurable paths, model names)

---

## Quick Start

### Initialize a Palace

```bash
go run ./cmd/cli init ~/.my-palace
```

### Run as MCP Server

```bash
go run ./cmd/cli server
```

### Mine Project Files

```bash
go run ./cmd/cli mine /path/to/project --mode projects
```

### Search Memories

```bash
go run ./cmd/cli search "authentication flow"
```

---

## MANUAL

This document is maintained by the oh-my-claudecode writer agent.

When updating:

1. **Keep this file as single source of truth** for mentle module organization
2. **Link to internal/, pkg/, storage/ subdirectories** for implementation details
3. **Do not duplicate** search algorithms or KG logic
4. **Update when:**
   - New internal packages are added
   - Vector store backend changes
   - MCP tools are added/removed
   - Performance targets are adjusted
   - Dependencies are added/removed
5. **Do not update for:**
   - Implementation details (use subdirectory AGENTS.md instead)
   - Temporary feature branches
   - Benchmark results (document in benchmarks/AGENTS.md)
   - Archive material (preserve in docs/)

Parent reference: `../AGENTS.md`
