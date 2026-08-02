<!-- Parent: ../../AGENTS.md -->

# laputa/cmd/laputa — Main Laputa CLI (Placeholder)

**Generated:** 2026-08-01  
**Purpose:** Main command-line interface for Laputa governance operations

---

## Purpose

The `laputa` directory is reserved for main CLI entry points and commands (currently minimal).

Future commands:
- `laputa init` — initialize governance sections
- `laputa audit` — view mutation history
- `laputa backup` — snapshot sections to tarball
- `laputa restore` — restore from backup
- `laputa status` — show current governance state

---

## Structure

```
laputa/
├── main.go                          # CLI entry point (if implemented)
└── (command files as added)
```

---

## Current Status

Currently a placeholder. See `eino_smoke/` for working smoke tests.

---

## MANUAL

When implementing main CLI commands, add them to this directory and document here.

Parent reference: ../../AGENTS.md
