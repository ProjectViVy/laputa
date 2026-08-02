<!-- Parent: ../../AGENTS.md -->

# mentle/pkg/mcp — MCP Protocol Server Implementation

**Generated:** 2026-08-01  
**Purpose:** JSON-RPC 2.0 protocol server for Claude Desktop/Cursor integration

---

## Purpose

The `mcp/` package implements the Model Context Protocol:

- **JSON-RPC 2.0** message handling
- **Tool registration** and dispatch
- **Request/response routing**
- **Async call tracking**

See parent [mentle/pkg/AGENTS.md](../AGENTS.md) for detailed tool documentation.

---

## Structure

```
mcp/
├── protocol.go                      # JSON-RPC protocol handling
├── protocol_test.go
├── server.go                        # MCP server main logic
├── server_test.go
├── tools.go                         # Tool definitions
├── tools_test.go
└── (supporting utilities)
```

---

## Key Types

```go
// JSON-RPC request/response
type Message struct {
    JSONRPC string          `json:"jsonrpc"`
    ID      interface{}     `json:"id,omitempty"`
    Method  string          `json:"method,omitempty"`
    Params  json.RawMessage `json:"params,omitempty"`
    Result  json.RawMessage `json:"result,omitempty"`
    Error   *Error          `json:"error,omitempty"`
}

// Tool definition
type Tool struct {
    Name        string
    Description string
    InputSchema json.Schema
}
```

---

## Exposed Tools

| Tool | Purpose |
|------|---------|
| search | Search memories |
| store | Store new memory |
| read | Read evidence |
| list_wings | List categories |
| list_rooms | List entity collections |

---

## Build & Test

```bash
cd mentle
GOSUMDB=off go test ./pkg/mcp/...
```

---

## MANUAL

When adding tools:
1. Define tool schema in `tools.go`
2. Implement handler
3. Register in server startup
4. Add tests
5. Update parent AGENTS.md

Parent reference: ../AGENTS.md
