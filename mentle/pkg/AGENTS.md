<!-- Parent: ../AGENTS.md -->

# mentle/pkg — MCP Server and WAL (Write-Ahead Log)

**Generated:** 2026-08-01  
**Purpose:** MCP protocol server and durable write-ahead logging

---

## Purpose

The `pkg/` directory contains reusable packages:

- **MCP server** — JSON-RPC protocol implementation for Claude Desktop/Cursor integration
- **WAL (Write-Ahead Log)** — durable logging for all mutations

---

## Structure

```
pkg/
├── mcp/
│   ├── protocol.go                # JSON-RPC protocol handling
│   ├── server.go                  # MCP server main logic
│   ├── server_test.go
│   ├── tools.go                   # Tool definitions (search, store, etc.)
│   └── tools_test.go
└── wal/
    ├── wal.go                     # Write-ahead log implementation
    ├── wal_test.go
    └── recovery.go                # WAL recovery on startup
```

### Subdirectories (Depth 3)

- **mcp/** — MCP server and protocol
- **wal/** — Write-ahead log and recovery

---

## MCP Server

### Purpose

Enable Mentle to integrate with Claude Desktop and Cursor as an MCP resource server.

### Protocol

- **JSON-RPC 2.0** over stdio
- **Tools exposed** — search, store, read, etc.
- **Async calls** — request/response with request ID tracking

### Tools

| Tool | Purpose | Input | Output |
|------|---------|-------|--------|
| search | Search memories | query, scope, max_results | cards, scores |
| store | Store new memory | content, wing, room, tags | card_id |
| read | Read evidence | card_id, budget | fragments |
| list_wings | List categories | — | wings |
| list_rooms | List entity collections | wing | rooms |

### MCP Server Startup

```bash
./mentle server --listen 127.0.0.1:9999
```

Claude Desktop config:

```json
{
  "mcpServers": {
    "mentle": {
      "command": "./mentle",
      "args": ["server", "--listen", "127.0.0.1:9999"]
    }
  }
}
```

---

## WAL (Write-Ahead Log)

### Purpose

Ensure durability: all mutations are logged before applied.

If Mentle crashes:
- Committed entries are recovered from WAL
- In-flight entries are rolled back
- No data loss, no corruption

### Operations Logged

- Store memory (card creation)
- Update memory (modification)
- Delete memory (removal)
- Update knowledge graph
- Store activity events

### Recovery

On startup, WAL is replayed:

```go
log, err := wal.Open("./wal")
if err != nil {
    log.Fatal(err)
}

// Replay committed entries
entries, err := log.RecoverCommitted(ctx)
if err != nil {
    log.Fatal(err)
}

// Apply entries to palace/KG
for _, entry := range entries {
    applyEntry(ctx, entry)
}

// Clear in-flight entries
log.ClearInFlight()
```

### Entry Format

Each WAL entry is a JSON record:

```json
{
  "id": "wal-entry-123",
  "timestamp": "2026-08-01T10:30:00Z",
  "operation": "store",
  "data": {...},
  "status": "committed"
}
```

Status values: `pending`, `committed`, `rolled_back`

---

## Build & Test

### Build

```bash
cd mentle
go build ./pkg/...
```

### Test

```bash
cd mentle
GOSUMDB=off go test ./pkg/...
```

### Test MCP Protocol

```bash
GOSUMDB=off go test -v ./pkg/mcp/ -run TestJSONRPC
```

### Test WAL Recovery

```bash
GOSUMDB=off go test -v ./pkg/wal/ -run TestRecovery
```

---

## Testing Requirements

Before starting feature work:

```bash
cd mentle && GOSUMDB=off go test ./pkg/...
```

**Mandatory behavioral tests:**

- MCP tools serialize/deserialize correctly
- WAL entries are recoverable after crash
- Concurrent writes don't corrupt WAL
- Recovery produces bit-identical state

---

## Conventions

- **Go formatting:** standard `gofmt`
- **JSON-RPC:** follow spec exactly, no extensions
- **WAL entries:** always JSON, UTF-8 encoded
- **Error handling:** explicit, wrapped with context

---

## MANUAL

When updating:

1. MCP tools must be idempotent where possible
2. WAL format changes require migration logic
3. Do not silently skip WAL entries
4. Update when:
   - New MCP tools are added
   - WAL entry types are added
   - Recovery logic changes
5. Do not update for:
   - Individual entry contents (schema-independent)

Parent reference: ../AGENTS.md
