<!-- Parent: ../AGENTS.md -->

# Laputa — Governance Module

**Generated:** 2026-08-01  
**Module Path:** `github.com/dashimaki/laputa`  
**Go Version:** 1.26.4 or later  
**Status:** Phase 3+ active — governance lifecycle, policy, and audit

---

## Purpose

Laputa is a file-based governance substrate for AI agents. It provides:

- **Legacy 14-section compatibility registry** stored in `.laputa/sections/*.json`; target cognitive partition is governed by `../docs/architecture/0002-laputa-cognitive-partition-decision.md`
- **Write authority registry** per section
- **Atomic file operations** with cross-process safety
- **Zero subprocesses** — no daemons, no sidecars
- **Rhythm reports** — periodic LLM-generated summaries (daily, weekly, monthly)

All governance state is durable, and no silent mutations are allowed.

---

## Module Structure

```
laputa/
├── .git/                         # Laputa submodule (independent repo)
├── .hermes/                      # Hermes planning artifacts
├── .laputa/                      # File-based governance sections (14 JSON files)
├── cmd/
│   └── eino_smoke/              # Smoke test for eino LLM orchestration
├── governance/
│   └── rhythm/                   # Reporting engine (daily, weekly, monthly)
│       ├── types.go              # RhythmKind, ReportResult, Config
│       ├── prompt.go             # LLM prompt generation
│       ├── generator.go          # Report generation logic
│       ├── generator_test.go
│       └── prompt_test.go
├── scripts/                      # Utility scripts (if present)
├── go.mod                        # Module: github.com/dashimaki/laputa
├── go.sum
├── README.md
├── ARCHITECTURE.md
└── .gitignore
```

### Subdirectories (Depth 2)

Depth-2 AGENTS.md files exist for:

- **[governance/AGENTS.md](./governance/AGENTS.md)** — authority, sections, lifecycle, policy, audit
- **[cmd/AGENTS.md](./cmd/AGENTS.md)** — CLI entry points and smoke tests
- **[scripts/AGENTS.md](./scripts/AGENTS.md)** — (if needed) utility scripts and setup tools

---

## Key Concepts

### Governance Sections

The `.laputa/` directory contains 14 JSON section files, each defining a policy domain:

| Section | Purpose |
|---------|---------|
| `01-identity.json` | Agent identity and principal definitions |
| `02-authority.json` | Write authority registry and capability grants |
| `03-lifecycle.json` | Session and agent lifecycle state |
| `04-policy.json` | Authority constraints and policy rules |
| `05-audit.json` | Audit log and mutation history |
| (+ 9 more domain-specific sections) | — |

Each section is immutable except via explicit, audited mutations.

### No Silent Durable Mutation

The following operations are always explicit and audited:

- Manual `MEMRULES.MD` revision
- Protected/user-confirmed `WORLD.MD` revision or Frozen Core authority mutation
- Skill approval
- Host installation
- Evolution proposal acceptance
- Hub fetch/publish
- Physical deletion

### Rhythm Engine

The `governance/rhythm` package generates periodic reports using an LLM:

```go
type RhythmKind string
const (
    RhythmDaily   RhythmKind = "daily"
    RhythmWeekly  RhythmKind = "weekly"
    RhythmMonthly RhythmKind = "monthly"
)

type ReportResult struct {
    Title         string    `json:"title"`
    Summary       string    `json:"summary"`
    Highlights    []string  `json:"highlights"`
    OpenQuestions []string  `json:"open_questions,omitempty"`
    GeneratedAt   time.Time `json:"generated_at"`
}
```

---

## Build & Test

### Build

```bash
cd laputa
go mod tidy
go build ./...
```

### Test

```bash
cd laputa
GOSUMDB=off go test ./governance/...
GOSUMDB=off go test ./...
```

**Required behavioral tests:**

- Governance initialization and snapshot reads
- Authority mutation enforcement
- Section isolation
- Rhythm report generation (mock and real LLM paths)

### Running Rhythm Reports

```bash
# Mock generator (no API key)
go run ./cmd/eino_smoke -kind daily

# Real LLM (requires OPENAI_API_KEY)
export OPENAI_API_KEY=***
go run ./cmd/eino_smoke -kind daily
```

Supported kinds: `daily`, `weekly`, `monthly`.

---

## Dependencies

### External Packages

| Package | Use | License |
|---------|-----|---------|
| `cloudwego/eino` | LLM orchestration framework | Apache 2.0 |
| `cloudwego/eino-ext/components/model/openai` | OpenAI integration | Apache 2.0 |
| `redis/go-redis/v9` | Redis client (optional governance state) | BSD-2 |

### Shared (via parent garden)

- `google/uuid` — UUID generation
- `go.yaml.in/yaml/v3` — YAML parsing
- `mattn/go-sqlite3` — SQLite3 driver (if governance uses SQLite)

---

## Testing Requirements

Before starting any feature work or migration wave:

```bash
cd laputa && GOSUMDB=off go test ./governance/...
```

**Exit gates:**

- Authority mutations are audited and reversible
- Sections cannot be corrupted by concurrent writes
- Rhythm reports are generated without side effects
- Mock LLM path produces valid ReportResult

---

## Architecture Boundaries

**Ownership rule (immutable):**

```text
Laputa may approve and apply authority.
Laputa may not store material.
Laputa may not retrieve or rank content.
Garden may orchestrate but not hold authority.
Mentle may store and retrieve but not approve.
```

---

## Conventions

- **Go formatting:** standard `gofmt`, no exotic linters
- **Error handling:** explicit, no panics in library code
- **Concurrency:** file-based locking with flock where needed
- **Configuration:** environment variables or YAML for rhythm LLM setup
- **Logging:** structured, optional; no log noise by default

---

## MANUAL

This document is maintained by the oh-my-claudecode writer agent.

When updating:

1. **Keep this file as single source of truth** for laputa module organization
2. **Link to governance/ subdirectory** for policy and authority details
3. **Do not duplicate** rhythm or authority implementation details
4. **Update when:**
   - New governance sections are added
   - New rhythm kinds are introduced
   - Testing requirements change
   - Dependencies are added/removed
5. **Do not update for:**
   - Implementation details (use governance/AGENTS.md instead)
   - Temporary feature branches
   - Archive material (preserve in docs/archive/)

Parent reference: `../AGENTS.md`
