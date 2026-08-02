<!-- Parent: ../../AGENTS.md -->

# mentle/cmd/cli — Main CLI Entry Point

**Generated:** 2026-08-01  
**Purpose:** Root command dispatcher for Mentle command-line interface

---

## Purpose

The `cli/` directory implements the main CLI entry point for Mentle:

- **Root command dispatcher** — routes subcommands
- **Cobra framework** for command registration
- **Subcommands** — init, mine, search, server, bench, etc.

See parent [mentle/cmd/AGENTS.md](../AGENTS.md) for command documentation.

---

## Structure

```
cli/
├── main.go                          # Root command and dispatcher
├── init.go                          # init command
├── mine.go                          # mine command
├── search.go                        # search command
├── server.go                        # server command (MCP)
├── bench.go                         # bench command
├── mcp.go                           # MCP server variant
├── compress.go                      # compress command
├── split.go                         # split command
├── hook.go                          # hook command
├── (other commands)
└── (supporting utilities)
```

---

## Build

```bash
cd mentle
go build -o mentle ./cmd/cli
```

---

## Run

```bash
./mentle init ~/.my-palace
./mentle mine /path --mode projects
./mentle search "query"
./mentle server --listen 127.0.0.1:9999
```

---

## Conventions

- **Root command** — dispatcher in `main.go`
- **Each subcommand** — separate file (`*.go`)
- **Flag registration** — in command struct
- **Error handling** — exit codes reflect status

---

## MANUAL

When adding subcommands:
1. Create new file (e.g., `newcmd.go`)
2. Register in root dispatcher (`main.go`)
3. Document in parent AGENTS.md

Parent reference: ../AGENTS.md
