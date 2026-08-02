<!-- Parent: ../../AGENTS.md -->

# mentle/cmd/server — MCP Server Standalone Entry Point

**Generated:** 2026-08-01  
**Purpose:** Standalone MCP server entry point for Claude Desktop/Cursor integration

---

## Purpose

The `server/` directory provides a simplified entry point for running Mentle as an MCP resource server:

- **Minimal CLI** — just `--listen` and `--config`
- **JSON-RPC protocol** over stdio or TCP
- **Direct Claude integration** via MCP config

---

## Structure

```
server/
├── main.go                          # MCP server entry point
└── (server utilities)
```

---

## Build

```bash
cd mentle/cmd/server
go build -o mentle-server .
```

---

## Run

```bash
./mentle-server --listen 127.0.0.1:9999
```

---

## Claude Desktop Config

```json
{
  "mcpServers": {
    "mentle": {
      "command": "./mentle-server",
      "args": ["--listen", "127.0.0.1:9999"]
    }
  }
}
```

---

## MANUAL

This is a simplified wrapper around the main `cli/server` command. Maintain parity with the primary CLI.

Parent reference: ../AGENTS.md
