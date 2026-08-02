<!-- Parent: ../AGENTS.md -->

# mentle/cmd — CLI Commands and Entry Points

**Generated:** 2026-08-01  
**Purpose:** Command-line interface for memory operations, mining, and MCP server

---

## Purpose

The `cmd/` directory contains CLI commands for interacting with Mentle:

- **Main CLI root** (`main.go`) — dispatcher for subcommands
- **MCP server command** (`mcp.go`) — start as MCP server for Claude Desktop/Cursor
- **Memory mining** (`mine.go`) — extract memories from project files
- **Search command** (`search.go`) — query memories from the command line
- **Data splitting** (`split.go`) — split memory data for benchmarking
- **Compression** (`compress.go`) — compress palace storage
- **Hook management** (`hook.go`) — integrate with version control hooks

---

## Structure

```
cmd/
├── cli/
│   ├── main.go                   # Root command dispatcher
│   ├── mcp.go                    # MCP server command
│   ├── mine.go                   # Memory mining command
│   ├── search.go                 # Search command
│   ├── split.go                  # Data splitting command
│   ├── compress.go               # Compression command
│   ├── hook.go                   # Hook management command
│   └── (supporting utilities)
└── go.mod/go.sum                 # If cmd is a separate module (usually not)
```

---

## Commands

### init

Initialize a new Palace:

```bash
go run ./cmd/cli init ~/.my-palace
```

Sets up directory structure, ONNX model, and SQLite storage.

### server (MCP)

Start Mentle as MCP server:

```bash
go run ./cmd/cli server --port 9999
```

Listens on localhost:9999 (or configured port) for MCP JSON-RPC requests from Claude Desktop/Cursor.

### mine

Extract memories from a directory or project:

```bash
go run ./cmd/cli mine /path/to/project --mode projects
go run ./cmd/cli mine /path/to/project --mode conversations
```

Modes:
- `projects` — extract from code, docs, commit messages
- `conversations` — extract from chat logs or dialogue
- `mixed` — both

### search

Search memories from the command line:

```bash
go run ./cmd/cli search "authentication flow"
```

Returns top-K results (default K=10) with snippets.

### split

Split memory data for benchmarking:

```bash
go run ./cmd/cli split /path/to/palace --ratio 0.8 --output benchmarks/
```

Splits into train/test sets for LongMemEval evaluation.

### compress

Compress palace storage (optimize HNSW index):

```bash
go run ./cmd/cli compress ~/.my-palace
```

Rebuilds indexes to improve query performance.

### hook

Manage git hooks for auto-mining on commits:

```bash
go run ./cmd/cli hook install ~/.my-palace
go run ./cmd/cli hook uninstall
```

---

## Build

### Build CLI Binary

```bash
cd mentle
go build -o mentle ./cmd/cli
```

Or use Makefile:

```bash
make build
```

### Build MCP Server

```bash
make build-mcp
```

---

## Usage in Integration

### From Go Code

```go
import "github.com/dashimaki/mentle/cmd/cli"

// Use CLI utilities directly
result := cli.Search(context.Background(), palace, "query")
```

### From Shell

```bash
./mentle init ~/.palace
./mentle mine /path --mode projects
./mentle search "what is this?"
./mentle server --listen 0.0.0.0:9999
```

---

## Testing

### Test Commands

```bash
cd mentle
GOSUMDB=off go test ./cmd/...
```

### Integration Tests

```bash
./run_integration_tests.sh
go test -v ./integration/ -run TestCLI
```

---

## Conventions

- **Root command:** `main.go` uses `cobra` for subcommand dispatch
- **Subcommands:** each in its own file (`mine.go`, `search.go`, etc.)
- **Error handling:** explicit, exit codes reflect status
- **Output:** JSON (with `--json` flag) or human-readable text
- **Flags:** long flags for stability (`--mode`, `--port`), short for quick use (`-m`, `-p`)

---

## MANUAL

When adding commands:

1. Create new file (e.g., `newcmd.go`) in `cmd/cli/`
2. Register in root dispatcher (`main.go`)
3. Document usage in this file
4. Add integration tests in `../integration/`

Parent reference: ../AGENTS.md
