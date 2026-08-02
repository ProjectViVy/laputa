# mempalace-go

**Give your AI a memory. No API key required.**

A Go implementation of [MemPalace](https://github.com/milla-jovovich/mempalace) — a memory system for AI assistants that implements the [Model Context Protocol (MCP)](https://modelcontextprotocol.io/) over stdio.

## Overview

mempalace-go provides a single, portable binary with no Python dependencies or persistent daemon processes. It exposes memory operations as MCP tools that AI clients (like Claude Desktop, Cursor, or other MCP-compatible editors) can invoke to store, search, and recall contextual information.

### Key Features

- **Portable Binary**: Single executable, no runtime dependencies
- **MCP Server**: Stdio-based JSON-RPC protocol for AI client integration
- **Vector Search**: Semantic memory retrieval using ONNX embeddings (hugot)
- **Knowledge Graph**: SQLite-based entity relationship tracking with temporal validity
- **WAL-based Storage**: Write-ahead log for durable memory operations
- **CLI Tools**: Project mining, search, repair, and palace management

## Architecture

```
┌─────────────┐     MCP/JSON-RPC     ┌───────────────┐
│   MCP      │◄────────────────────►│  mempalace-go │
│   Client   │     stdio            │    server     │
└─────────────┘                     └───────┬───────┘
                                            │
              ┌─────────────────────────────┼──────────────────────────┐
              │                             │                          │
         ┌────▼──────┐              ┌───────▼──────┐          ┌───────▼────────┐
         │  Hugot    │              │   Memory     │          │  Knowledge     │
         │  (ONNX)   │              │   Stack      │          │  Graph (SQLite)│
         └────┬──────┘              └───────▲──────┘          └────────────────┘
              │                             │
         ┌────▼────────────┐          ┌─────▼────┐
         │ Embedding Model │          │ Vector   │
         │ all-MiniLM-L6   │          │ Store    │
         │ (feature extr.) │          │(govector)│
         └─────────────────┘          └──────────┘
```

### Embedding Pipeline

mempalace-go uses [hugot](https://github.com/knights-analytics/hugot) for ONNX-based embeddings — no external processes or llamafiles are required. Models are downloaded once and run natively in Go:

1. Text input → hugot feature extraction pipeline
2. ONNX runtime produces 384-dimensional embeddings
3. Vector store (govector with HNSW index) handles similarity search

## Installation

### Prerequisites

- Go 1.26.2 or later
- ONNX embedding model (auto-downloaded by hugot on first run)

## Building

### Standard Build (pure Go, no CGo)

```bash
go build -o mempalace-go .
# or
make build
```

No native dependencies required. Uses hugot's pure-Go backend.

### ORT Build (Apple Silicon acceleration)

The ORT build enables [ONNX Runtime](https://onnxruntime.ai/) with the CoreML execution provider on darwin/arm64, which allows embedding inference to run on the GPU or ANE where the model graph is supported.

**Step 1: Download `libtokenizers.a`**

The ORT backend requires `libtokenizers.a`, a pre-built Rust library for fast BERT tokenization. Download the darwin/arm64 build once:

```bash
mkdir -p ~/lib
curl -fSL https://github.com/daulet/tokenizers/releases/download/v1.26.0/libtokenizers.darwin-aarch64.tar.gz \
    | tar -xz -C ~/lib/
```

**Step 2: Build with ORT tag**

```bash
export CGO_LDFLAGS="-L${HOME}/lib"
make build-ort
# equivalent to: go build -tags ORT ./...
```

**Step 3 (optional): Install system-wide to avoid the env var**

If you prefer not to set `CGO_LDFLAGS` every time, copy the library to a standard system path:

```bash
sudo cp ~/lib/libtokenizers.a /usr/local/lib/libtokenizers.a
# Then: make build-ort  (no CGO_LDFLAGS needed)
```

### Makefile targets

| Target | Description |
|---|---|
| `make build` | Pure Go build, no CGo |
| `make build-ort` | ORT + CoreML build (requires `libtokenizers.a`) |
| `make test` | Run all tests (pure Go) |
| `make test-ort` | Run all tests with ORT backend |
| `make bench-perf` | Run embedding/storage timing benchmark (pure Go) |
| `make bench-perf-ort` | Run benchmark with ORT + CoreML |

## Configuration

mempalace-go uses a config file at `~/.mempalace/config.json`:

```json
{
  "palace_path": "~/.mempalace/palace",
  "model_name": "sentence-transformers/all-MiniLM-L6-v2",
  "models_dir": "~/.mempalace/models",
  "collection_name": "default"
}
```

### Configuration Options

| Field | Description | Default |
|-------|-------------|---------|
| `palace_path` | Path to the memory palace data | `~/.mempalace/palace` |
| `model_name` | Hugging Face model name for embeddings | `sentence-transformers/all-MiniLM-L6-v2` |
| `models_dir` | Directory for cached ONNX models | `~/.mempalace/models` |
| `collection_name` | Collection identifier | `default` |

## Quick Start

### 1. Initialize a Palace

```bash
./mempalace-go init ~/.my-palace
```

### 2. Run as MCP Server

```bash
./mempalace-go server
```

### 3. Mine Project Files

```bash
./mempalace-go mine /path/to/project --mode projects
```

### 4. Search Memories

```bash
./mempalace-go search "authentication flow"
```

## CLI Commands

### Core Commands

| Command | Description |
|---------|-------------|
| `mempalace-go init [dir]` | Initialize a new memory palace |
| `mempalace-go mine [dir] --mode [projects\|convos]` | Mine files or conversations into the palace |
| `mempalace-go search [query]` | Search memories by query |
| `mempalace-go wake-up --wing [wing]` | Show L0 + L1 context for a wing |
| `mempalace-go status` | Show palace configuration status |

### Maintenance Commands

| Command | Description |
|---------|-------------|
| `mempalace-go repair` | Rebuild palace vector index from WAL files |
| `mempalace-go compress` | Compress palace storage |
| `mempalace-go split` | Split palace data |
| `mempalace-go hook` | Manage hooks |

### Advanced Commands

| Command | Description |
|---------|-------------|
| `mempalace-go mcp` | Run MCP server (alias for `server`) |
| `mempalace-go instructions` | Show usage instructions |
| `mempalace-go onboard` | Interactive onboarding |
| `mempalace-go bench` | Run benchmarks |

### Global Flags

- `--palace`: Override palace path from config

## MCP Tools

When running as an MCP server, mempalace-go exposes the following tools:

### Memory Operations

| Tool | Description | Parameters |
|------|-------------|------------|
| `search` | Search memories by query | `query`, `wing` (optional), `room` (optional) |
| `wake` | Wake up memory with wing context | `wing` (optional) |
| `recall` | Recall memories from wing/room | `wing`, `room`, `count` (default: 10) |

### Palace Management

| Tool | Description | Parameters |
|------|-------------|------------|
| `mempalace_status` | Get palace status and overview | None |
| `mempalace_list_wings` | List all wings with drawer counts | None |
| `mempalace_list_rooms` | List rooms within a wing | `wing` (optional) |
| `mempalace_get_taxonomy` | Get full wing → room → count tree | None |

### Drawer Operations

| Tool | Description | Parameters |
|------|-------------|------------|
| `mempalace_add_drawer` | Add content to a wing/room | `content`, `wing`, `room`, `source` (optional) |
| `mempalace_delete_drawer` | Delete a drawer by ID | `id` |
| `mempalace_check_duplicate` | Check if content already exists | `content`, `wing` (optional), `room` (optional) |

### Knowledge Graph

| Tool | Description | Parameters |
|------|-------------|------------|
| `kg_query` | Query knowledge graph for entity relationships | `entity`, `as_of` (optional), `direction` (default: "outgoing") |
| `mempalace_kg_add` | Add fact to knowledge graph | `subject`, `predicate`, `object`, `valid_from`, `valid_to`, `confidence` |
| `mempalace_kg_invalidate` | Mark facts as ended | `subject`, `predicate`, `object`, `valid_to` |
| `mempalace_kg_timeline` | Get chronological entity story | `entity` |
| `mempalace_kg_stats` | Get knowledge graph statistics | None |

### Graph Navigation

| Tool | Description | Parameters |
|------|-------------|------------|
| `mempalace_traverse` | Walk the palace graph from a room | `room`, `max_hops` (default: 3) |
| `mempalace_find_tunnels` | Find rooms bridging two wings | `wing_a`, `wing_b` |
| `mempalace_graph_stats` | Get palace graph connectivity | None |

### Agent Diary (AAAK)

| Tool | Description | Parameters |
|------|-------------|------------|
| `mempalace_diary_write` | Write AAAK diary entry for a specialist agent | `agent`, `content`, `wing` (optional) |
| `mempalace_diary_read` | Read recent diary entries | `agent` (optional), `wing` (optional), `limit` (default: 10), `hours` (default: 24) |
| `mempalace_get_aaak_spec` | Get AAAK dialect reference specification | None |

### Advanced Search

| Tool | Description | Parameters |
|------|-------------|------------|
| `mempalace_deep_search` | L3 deep semantic search with full results | `query`, `wing` (optional), `room` (optional), `count` (default: 20) |

## LongMemEval Benchmark Results

The `bench longmemeval` command runs the [LongMemEval](https://arxiv.org/abs/2410.10813) session-retrieval benchmark (500 questions, ~49 sessions each). Results below are for the `session` granularity, `raw` mode, top-50 retrieval, on Apple M-series (pure-Go GoMLX backend, no CoreML).

### Accuracy (500 questions)

| Metric      | Score  |
|-------------|--------|
| Recall@5    | 92.8%  |
| Recall@10   | 96.2%  |
| NDCG@10     | 0.809  |

| Type                       | n   | R@5   | R@10   | NDCG@10 |
|----------------------------|-----|-------|--------|---------|
| knowledge-update           |  78 | 98.7% | 100.0% | 0.835   |
| multi-session              | 133 | 94.7% |  98.5% | 0.823   |
| single-session-assistant   |  56 | 96.4% |  98.2% | 0.950   |
| single-session-preference  |  30 | 90.0% |  96.7% | 0.797   |
| temporal-reasoning         | 133 | 90.2% |  94.7% | 0.755   |
| single-session-user        |  70 | 85.7% |  88.6% | 0.753   |

### Embedding Speed

| Phase                     | Before (baseline) | After (cache + 4 workers) |
|---------------------------|-------------------|---------------------------|
| Per-question embed (avg)  | ~8.3 s            | ~100 ms (cache hit)       |
| Cache build (18,362 sessions) | N/A          | ~11 min (one-time, 4 workers) |
| Total 500-question run    | ~69 min (est.)    | ~12 min                   |

**Key optimisations applied:**

1. **Cross-question session cache** — unique session texts across all 500 questions are embedded once upfront. The 18,362 unique sessions are split across 4 parallel workers (each with its own hugot/GoMLX instance). After the cache is warm, each `buildCorpus` call is a pure in-memory map lookup (~100 ms for vector store writes vs 8.3 s of GEMM before).

2. **Rune-based truncation** — texts are truncated to 400 Unicode code-points before embedding. Word-count limits fail on long URLs and Unicode math symbols (a single "word" can produce hundreds of subword tokens, crashing GoMLX's shape-specific JIT graphs). At 400 runes, the worst case (all ASCII URLs) produces ≤ 400 tokens, safely under the 512-position limit.

3. **Chunk size 32 → 64** — hugot's `RunPipeline` is called with batches of 64 texts instead of 32, halving the number of forward-pass invocations per question.



```
mempalace-go/
├── cmd/
│   ├── cli/              # CLI command implementations
│   │   ├── main.go       # Root command (init, mine, search, etc.)
│   │   ├── bench.go      # Benchmark command
│   │   ├── compress.go   # Compress command
│   │   ├── hook.go       # Hook command
│   │   ├── instructions.go # Instructions command
│   │   ├── mcp.go        # MCP server command
│   │   ├── onboard.go    # Onboarding command
│   │   └── split.go      # Split command
│   └── server/           # Standalone MCP server
│       └── main.go       # Server entry point
├── pkg/
│   ├── mcp/              # MCP server implementation
│   └── wal/              # Write-ahead log
├── internal/
│   ├── config/           # Configuration management
│   ├── embedder/         # ONNX embedding models (hugot)
│   ├── layers/           # Memory stack operations
│   ├── miner/            # Project/conversation mining
│   ├── palace/           # Palace graph structure
│   ├── search/           # Semantic search
│   ├── kg/               # Knowledge graph (SQLite)
│   ├── diary/            # Agent diary (AAAK)
│   ├── extractor/        # Memory extraction
│   ├── dialect/          # Text dialect handling
│   ├── entity/           # Entity detection
│   └── sanitizer/        # Input sanitization
├── storage/
│   └── govector/         # Vector storage backend
├── integration/          # Integration tests
├── benchmarks/           # Performance benchmarks
├── docs/                 # Documentation
└── main.go               # Entry point
```

## Storage Model

- **Vector Store**: `vectors.db` (govector with HNSW index)
- **Knowledge Graph**: `knowledge_graph.sqlite3` (temporal RDF-style triples)
- **WAL Directory**: `wal/` (write-ahead log for durability)
- **Diary**: `diary/` (agent-specific AAAK entries)

## Embedding Models

mempalace-go uses [hugot](https://github.com/knights-analytics/hugot) for ONNX-based embeddings. The default model is `sentence-transformers/all-MiniLM-L6-v2` which produces 384-dimensional vectors.

Models are auto-downloaded from Hugging Face on first use and cached in the configured `models_dir` (default: `~/.mempalace/models`).

## Testing

```bash
# Run unit tests
go test ./...

# Run integration tests
go test -v ./integration/ -run TestMCP

# Run benchmarks
go test -bench=. ./benchmarks/

# Or use the integration test runner
./run_integration_tests.sh
```

## Integration with AI Clients

### Claude Desktop

Add to your Claude Desktop MCP configuration:

```json
{
  "mcpServers": {
    "mempalace": {
      "command": "/path/to/mempalace-go",
      "args": ["server"]
    }
  }
}
```

### Cursor / VS Code

Configure in your MCP settings to point to the mempalace-go binary with the `server` argument.

## AAAK Dialect

mempalace-go supports the **AAAK** (Aphantix Abstraction Annotating Kit) dialect for structured symbolic summaries:

- **Entity Codes**: 3-letter uppercase codes (e.g., KAI, MAX, PRI)
- **Topics**: Frequency-based with proper noun boosting
- **Emotion Codes**: vul, joy, fear, trust, grief, wonder, rage, etc.
- **Flag Codes**: DECISION, ORIGIN, CORE, PIVOT, TECHNICAL

Use `mempalace-go get_aaak_spec` to retrieve the full specification.

## Dependencies

- [cobra](https://github.com/spf13/cobra) - CLI framework
- [viper](https://github.com/spf13/viper) - Configuration management
- [hugot](https://github.com/knights-analytics/hugot) - ONNX embedding runtime
- [govector](https://github.com/DotNetAge/govector) - Vector storage with HNSW
- [sqlite](https://pkg.go.dev/modernc.org/sqlite) - Pure Go SQLite driver

## License

MIT — see [LICENSE](LICENSE).


## Credits

This is a Go port of the original [mempalace](https://github.com/milla-jovovich/mempalace) Python project by milla-jovovich. The port provides a simpler deployment model with no Python dependencies or daemon processes.
