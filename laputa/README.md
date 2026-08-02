# Laputa

> Go implementation of the Laputa governance framework.

## What It Is

A pure file-based governance substrate for AI agents.

- **14 governance sections** in `.laputa/sections/*.json`
- **Write authority registry** per section
- **Atomic file operations** with cross-process safety in mind
- **Zero subprocesses** — no daemons, no sidecars
- **Mempalace is completely separate**

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "github.com/dashimaki/laputa"
)

func main() {
    ctx := context.Background()
    store, _ := laputa.NewFileStore(".laputa")
    engine := laputa.NewEngine(store)
    _ = engine.Initialize(ctx)

    snap, _ := engine.Snapshot(ctx)
    fmt.Println(snap["schema_version"])
}
```

## Build & Test

```bash
go build ./...
go test ./...
```

## Rhythm Reports

Generate periodic reports using an LLM:

```bash
# Mock generator (no API key)
go run ./cmd/laputa -kind daily

# Real LLM
export OPENAI_API_KEY=*** run ./cmd/laputa -kind daily -api-key $OPENAI_API_KEY
```

Supported kinds: `daily`, `weekly`, `monthly`.

## Design Basis

This implementation follows:

- `laputa-py/baseline/LAPUTA.md` v0.0.6 final
- `laputa-py/baseline/MENTLE.md` v0.2
- `laputa-work/DECISIONS.md`

See `C:\Users\Administrator\Desktop\DIVA\docs\` for full design lineage.

## License

MIT
