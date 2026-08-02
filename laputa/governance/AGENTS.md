<!-- Parent: ../AGENTS.md -->

# laputa/governance — Authority, Sections, Lifecycle, and Policy

**Generated:** 2026-08-01  
**Purpose:** Governance engine, section management, write authority, and audit

---

## Purpose

The `governance/` package implements Laputa's core responsibility:

- **14 governance sections** stored in JSON files for durability
- **Write authority registry** per section (who can write, read, delete)
- **Atomic file operations** with cross-process locking (flock)
- **Rhythm reports** — periodic LLM-generated summaries (daily, weekly, monthly)
- **Immutable audit trail** — all mutations are logged and reversible

No governance state is lost to silent mutations; every write is explicit and tracked.

---

## Structure

```
governance/
├── rhythm/
│   ├── types.go                  # RhythmKind, ReportResult, Config
│   ├── prompt.go                 # LLM prompt generation
│   ├── generator.go              # Report generation logic
│   ├── generator_test.go
│   └── prompt_test.go
├── engine.go                      # Main governance engine
├── engine_test.go                 # Engine tests
└── (supporting types and utilities)
```

### Subdirectories (Depth 3)

- **rhythm/** — Report generation (daily, weekly, monthly rhythm)

---

## Key Concepts

### Governance Sections (legacy compatibility registry)

All are currently defined as constants in `engine.go`. This is not the target Garden cognitive partition: [ADR-0002](../../docs/architecture/0002-laputa-cognitive-partition-decision.md) removes target LTM/`LONGMEM.MD`, report-index and AAAK section semantics; introduces `MEMRULES.MD` and `WORLD.MD`; and requires a separate physical migration plan. Do not implement new features against the legacy meanings below.

Current constants:

| Name | Purpose | Write Authority |
|------|---------|-----------------|
| 01-identity | Agent role, capabilities, constraints, agentic RAG config | agent_self |
| 02-relationship | Relationships, resonance with agents | agent_self |
| 03-commitment | Commitments, red lines, agentic RAG denied sources | user_only |
| 04-preferences | Learning preferences and customization | agent_self |
| 05-memory_md | STM summary and highlights (MEMORY.MD source) | agent_self |
| 06-history_md | Timeline and history index (LONGMEM.MD source) | agent_self |
| 07-daily | Daily reports from rhythm engine | report_system |
| 08-weekly | Weekly reports from rhythm engine | report_system |
| 09-monthly | Monthly reports from rhythm engine | report_system |
| 10-journal_reflective | Reflective journal entries | tbd |
| 11-proposal_inbox | Evolution proposals awaiting review | tbd |
| 12-changelog | Durable mutation changelog | tbd |
| 13-report_indexes | Report index metadata | tbd |
| 14-aaak_summaries | AAAK dialect summaries (agent-specific) | tbd |

Each section is stored as a JSON file in `.laputa/sections/` directory.

### Write Authority

Four authority levels:

- **agent_self** — the agent process may write
- **user_only** — user (human) must explicitly write
- **report_system** — rhythm engine writes daily/weekly/monthly reports
- **tbd** — future use, not yet implemented

### Rhythm Engine

Generates periodic LLM-based reports:

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

Reports are generated using OpenAI API (configurable endpoint) and stored in the appropriate section.

### File Store

Implements atomic, cross-process-safe writes:

- **flock (gofrs/flock)** for cross-process advisory locking
- **Orphan lock cleanup** to recover from crashed processes
- **TryLock with retry** to avoid indefinite hangs on Windows
- **RWMutex** for in-process coordination
- **Metadata tracking** (`_meta.updated_at`, `_meta.version`)

---

## Build & Test

### Build

```bash
cd laputa
go mod tidy
go build ./governance/...
```

### Test

```bash
cd laputa
GOSUMDB=off go test ./governance/...
GOSUMDB=off go test ./...
```

### Test Specific Package

```bash
GOSUMDB=off go test -v ./governance/rhythm/...
```

---

## Usage

### Initialize Laputa

```go
import "github.com/dashimaki/laputa/governance"

store, err := governance.NewFileStore("~/.laputa")
if err != nil {
    log.Fatal(err)
}

engine := governance.NewEngine(store)
if err := engine.Initialize(context.Background()); err != nil {
    log.Fatal(err)
}
```

### Read a Section

```go
ctx := context.Background()
identity, err := engine.GetSection(ctx, governance.SectionIdentity)
if err != nil {
    log.Fatal(err)
}
```

### Update a Section

```go
err := engine.UpdateSection(ctx, governance.SectionIdentity, "role", "my-agent")
if err != nil {
    log.Fatal(err)
}
```

### Generate a Rhythm Report

```go
config := rhythm.Config{
    APIKey: os.Getenv("OPENAI_API_KEY"),
    BaseURL: "https://api.openai.com/v1",
    Model: "gpt-4",
}
gen := rhythm.NewGenerator(engine, config)
result, err := gen.Generate(ctx, rhythm.RhythmDaily)
if err != nil {
    log.Fatal(err)
}
```

---

## Testing Requirements

Before starting any feature work:

```bash
cd laputa && GOSUMDB=off go test ./governance/...
```

**Exit gates:**

- Authority mutations are audited and reversible
- Sections cannot be corrupted by concurrent writes
- Cross-process locks prevent lost updates
- Rhythm reports are generated without side effects
- Mock LLM path produces valid ReportResult

---

## Conventions

- **Go formatting:** standard `gofmt`
- **Error handling:** explicit, wrapped with context
- **Concurrency:** file-based locking (flock) for cross-process safety
- **JSON storage:** 2-space indent, always valid JSON
- **Configuration:** environment variables for OpenAI API
- **Logging:** structured, optional; no noise by default

---

## MANUAL

When updating:

1. Keep sections stable; new sections require design review
2. Document write authority for each section
3. Do not add silent mutations; all changes must be explicit
4. Update when:
   - New section is added
   - New rhythm kind is introduced
   - Write authority changes
   - Testing requirements evolve
5. Do not update for:
   - Implementation details (inline code comments)
   - Temporary branches

Parent reference: ../AGENTS.md
