<!-- Parent: ../AGENTS.md -->

# mentle/examples — Usage Examples and Quick-Start Guides

**Generated:** 2026-08-01  
**Purpose:** Code examples, tutorials, and quick-start guides

---

## Purpose

The `examples/` directory provides:

- **Quick-start code** — minimal working examples
- **Common patterns** — reusable code snippets
- **Tutorial guides** — step-by-step walkthroughs
- **Client examples** — SDK usage from external tools

Examples help users get started quickly with Mentle.

---

## Structure

```
examples/
├── quickstart.md                  # Getting started in 5 minutes
├── basic_search.go                # Simple search example
├── advanced_retrieval.go          # Complex retrieval scenarios
├── mcp_client.py                  # Python MCP client example
├── embedding_custom.go            # Custom embedding pipeline
└── (language-specific examples)
```

---

## Example: Basic Search

```go
package main

import (
    "context"
    "log"
    "github.com/dashimaki/mentle"
)

func main() {
    palace, err := mentle.Open("~/.my-palace")
    if err != nil {
        log.Fatal(err)
    }
    defer palace.Close()

    results, err := palace.SearchCards(context.Background(), "authentication flow", 10)
    if err != nil {
        log.Fatal(err)
    }

    for _, card := range results {
        println(card.Title, card.Summary)
    }
}
```

---

## Quick-Start Guide

```markdown
# Mentle Quick Start

## 1. Initialize

```bash
mentle init ~/.my-palace
```

## 2. Mine Project

```bash
mentle mine /path/to/project --mode projects
```

## 3. Search

```bash
mentle search "my question"
```

## 4. Store New Memory

```bash
mentle store "My insight" --wing architecture --room system-design
```

Done!
```

---

## Conventions

- **Go examples:** runnable with `go run`
- **Documentation:** clear comments explaining each step
- **Error handling:** explicit (not hidden)
- **Realistic:** use actual data, not toy examples

---

## MANUAL

When adding examples:

1. Make examples runnable (go run, python script, etc.)
2. Include comments explaining key concepts
3. Test before committing
4. Link from main README

Parent reference: ../AGENTS.md
