# Garden MemoryOS — Agents & Architecture Guide

**Generated:** 2026-08-01  
**Status:** vNext architecture proposed — implementation phase 0  
**Repository:** `garden/` workspace (3 Go modules + governance layer)

---

## Purpose

Garden MemoryOS is a governed memory system for continuous AI agents. It connects personal work materials to recalled context and reusable capability without treating storage, retrieval, or evolution output as authority by themselves.

The system separates concerns into three independent Go modules:

- **Laputa** — identity, authority, lifecycle, policy, audit
- **Mentle** — material, evidence, retrieval, taxonomy, graph  
- **Garden** — source ingestion, recall orchestration, ContextView assembly

Each module can degrade gracefully. None holds authority over the others.

---

## Key Files

- `README.md` — high-level overview, baseline verification commands
- `TODOLIST.MD` — live KANBAN-style status tracker (done / active / next) for the vNext migration
- `docs/architecture/0001-memoryos-vnext-architecture.md` — complete architecture plan, migration waves, HTTP contracts
- `docs/README.md` — documentation navigation and archive policy
- `go.work` — Go workspace configuration (if present; garden uses go.mod replace directives)
- Top-level module directories: `laputa/`, `mentle/`, `garden/`

---

## Repository Layout

```
garden/
├── .agents/              # Agent-specific tools and configurations
├── .omc/                 # oh-my-claudecode state and memory
├── docs/                 # Active and archived documentation
│   ├── README.md
│   ├── architecture/     # vNext ADRs and design documents
│   │   └── 0001-memoryos-vnext-architecture.md
│   └── archive/          # Pre-vNext design history
├── laputa/               # Go governance module
├── mentle/               # Go material and retrieval module
├── garden/               # Go HTTP gateway and orchestration
└── probe/                # (optional) diagnostic or test utilities
```

### Subdirectories (Depth 1)

Each major component has its own AGENTS.md:

- **[laputa/AGENTS.md](./laputa/AGENTS.md)** — governance, identity, lifecycle, policy, audit
- **[mentle/AGENTS.md](./mentle/AGENTS.md)** — canonical material, evidence, retrieval, taxonomy, knowledge graph
- **[garden/AGENTS.md](./garden/AGENTS.md)** — HTTP server, recall gateway, activity orchestration, ContextView assembly
- **[docs/AGENTS.md](./docs/AGENTS.md)** — documentation, architecture decisions, migration tracking
- **[.agents/AGENTS.md](./.agents/AGENTS.md)** — agent tools and orchestration metadata
- **[.omc/AGENTS.md](./.omc/AGENTS.md)** — oh-my-claudecode workspace state and memory

---

## Working for AI Agents

### Go Conventions

All three modules follow these conventions:

- **Go version:** 1.26.4 or later
- **Module path format:** `github.com/dashimaki/{module}`
- **Workspace setup:** garden uses `go.mod` replace directives to reference sibling modules:
  ```go
  replace (
    github.com/dashimaki/laputa => ../laputa
    github.com/dashimaki/mentle => ../mentle
  )
  ```
- **Test tags:** Use `-tags=e2e` for end-to-end integration tests
- **Environment:** `GOSUMDB=off` disables sum database checks for local development
- **Code style:** standard Go formatting (`gofmt`); no exotic linter plugins without explicit justification

### Testing Requirements

Before starting any migration wave or feature work, verify the baseline:

```bash
cd laputa && GOSUMDB=off go test ./governance/...
cd ../mentle && GOSUMDB=off go test ./facade/...
cd ../garden && GOSUMDB=off go test ./internal/...
GOSUMDB=off go test -tags=e2e ./e2e/...
```

**Required test layers:**

| Layer | Command | Required Proof |
|-------|---------|---|
| Laputa | `cd laputa && GOSUMDB=off go test ./governance/...` | section, store, authority behavior |
| Mentle | `cd mentle && GOSUMDB=off go test ./facade/...` | canonical, cards, evidence, lifecycle |
| Garden unit | `cd garden && GOSUMDB=off go test ./internal/...` | recall, activity, server, compatibility |
| Garden E2E | `GOSUMDB=off go test -tags=e2e ./e2e/...` | real process, HTTP, degradation |

**Mandatory behavioral tests:**

- SearchCards does not return full `Content`
- ReadEvidence enforces per-item and total character budget
- Superseded, deleted, expired and out-of-scope cards do not reappear
- Fast Recall does not call Planner, KG, timeline or graph
- Deep Recall always emits `RecallTrace`
- Mentle unavailable leaves governance projection and safe Fast Recall available
- Session end is idempotent on `session_id + event_id`
- Unauthorized authority mutation is rejected and audited

### Architecture Boundaries

**Ownership rule (immutable):**

```text
Garden may orchestrate.
Garden may not become authority.
Mentle may store and retrieve.
Mentle may not promote authority.
Evolver may propose capability.
Evolver may not persist authority.
Laputa may approve and apply authority.
```

**No silent high-impact mutation.** The following are always explicit and audited:

- Manual `MEMRULES.MD` revision
- Protected/user-confirmed `WORLD.MD` revision or a significant world-model correction
- Frozen Core authority mutation
- Skill approval
- Host installation
- Evolution proposal acceptance
- Hub fetch/publish
- Physical deletion

Ordinary STM working-set/checkpoint edits are lightweight revisions, not high-impact governance mutations. See `docs/architecture/0002-laputa-cognitive-partition-decision.md` for the accepted partition; legacy LTM promotion is removed.

### Key Runtime Modes

| Mode | Default | LLM | KG/graph | Writes | Trace |
|---|---:|---:|---:|---:|---:|
| Fast Recall | yes | no | no | no | compact request trace |
| Deep Recall | explicit | optional | explicit | no | required `RecallTrace` |
| Session Ingest | lifecycle | optional later | no default | activity/material | ingestion trace |
| Evolution Run | explicit/background | external Evolver may use | evidence scope only | proposal only | evolution event chain |
| Authority Apply | explicit governed action | no default | no default | Laputa/Mentle coordinated | audit + rollback ref |

### HTTP API Contracts

**New v2 endpoints (vNext):**

```http
POST   /v2/recall/fast              # deterministic default recall
POST   /v2/recall/deep              # explicit expensive recall
GET    /v2/recall/traces/{trace_id} # retrieve recall trace

POST   /v2/activity/events          # normalized activity events
GET    /v2/activity/sessions/{id}   # session activity history

POST   /v2/governance/projection    # read GovernanceProjection
POST   /v2/governance/proposals     # create or review proposals

POST   /v2/evolution/runs           # start evolution run
POST   /v2/evolution/proposals      # submit evolution proposal
GET    /v2/evolution/proposals/{id} # retrieve proposal details
```

**Legacy v1 routes (compatibility during migration):**

```http
POST   /v1/memories               # legacy CRUD translator
GET    /v1/memories/{key}         # legacy read
DELETE /v1/memories/{key}         # legacy delete
POST   /v1/context/resolve        # Fast or Deep adapter
POST   /v1/context/bootstrap      # Fast Recall bootstrap
GET    /v1/pipelines              # read-only inspection
GET    /health                    # health check
```

The translator must add deprecation metadata and must not expose new internal fields.

---

## Dependencies

### External Packages (Shared)

| Package | Use | License |
|---------|-----|---------|
| `google/uuid` | UUID generation | Apache 2.0 |
| `go.yaml.in/yaml/v3` | YAML parsing | MIT |
| `mattn/go-sqlite3` | SQLite3 driver | MIT |

### Laputa-Specific

| Package | Use |
|---------|-----|
| `cloudwego/eino` | LLM orchestration framework |
| `cloudwego/eino-ext/components/model/openai` | OpenAI integration |
| `redis/go-redis/v9` | Redis client (optional governance state) |

### Mentle-Specific

| Package | Use |
|---------|-----|
| `DotNetAge/govector` | Vector storage with HNSW |
| `knights-analytics/hugot` | ONNX-based embedding runtime |
| `gomlx/gomlx` + `gomlx/onnx-gomlx` | ML execution engine |
| `spf13/cobra` | CLI framework |
| `spf13/viper` | Configuration management |
| `redis/go-redis/v9` | Cache/async indexing |

### Garden-Specific

Garden combines Laputa and Mentle via replace directives. No additional runtime dependencies beyond those two modules.

---

## Key Concepts

### MemoryCard (Candidate)

A card-only object for candidate discovery. Must not contain full content.

```go
type MemoryCard struct {
    ID             string
    Kind           string
    Collection     string
    Scope          string
    Title          string
    Summary        string      // bounded, safe for discovery
    SourceRef      string
    Revision       int
    Status         string
    ValidFrom      time.Time
    ValidTo        *time.Time
    SupersededBy   *string
    Tags           []string
    HeatScore      float64
    LastActivated  *time.Time
    CandidateScore float64
}
```

### EvidenceFragment

Evidence is read only after a card passes policy and ranking.

```go
type EvidenceFragment struct {
    CardID       string
    MaterialRef  string
    SourceURI    string
    SourceRev    string
    Excerpt      string      // bounded by character budget
    StartOffset  int
    EndOffset    int
    ContentHash  string
    Validity     string
    EvidenceRefs []string
}
```

Every evidence read enforces item and character budgets.

### ContextView

Disposable final context assembled for one request.

```go
type ContextView struct {
    TraceID       string
    Scope         string
    Mode          string
    Governance    GovernanceProjection
    Cards         []MemoryCard
    Evidence      []EvidenceFragment
    Context       string
    BudgetChars   int
    Degraded      bool
    Warnings      []string
    RecallTraceID *string
}
```

A `ContextView` is not written to Laputa authority and is not automatically promoted to memory.

---

## Migration Waves

The vNext implementation is staged into 7 waves. Each wave has specific files, actions, and exit gates.

| Wave | Goal | Key Files |
|------|------|-----------|
| **0** | Document and baseline freeze | archive manifest, test snapshots |
| **1** | Card/Evidence API | `mentle/facade/{cards,evidence}.go` |
| **2** | Fast Recall Core | `garden/internal/recall/` |
| **3** | STM Runtime and session semantics | `garden/internal/activity/`, checkpoint state |
| **4** | Governance application boundary | `laputa/governance` mutation service |
| **5** | Deep Recall and logical arbiter | `garden/internal/recall/deep.go` |
| **6** | EvoMap/Evolver adapter | `garden/internal/evolution/` |
| **7** | Host adapters and release hardening | Hermes, Claude Code, Codex integration |

See `docs/architecture/0001-memoryos-vnext-architecture.md` Section 11 for detailed actions and exit gates.

---

## TODOLIST Tracker

`TODOLIST.MD` at the repo root is the single source of truth for execution status (what is done / active / next). It complements, but does not replace, this file and the architecture doc:

- **Architecture doc** = design contract (what & why)
- **AGENTS.md** = stable module/runtime reference (how it fits together)
- **TODOLIST.MD** = live status (where we are now)

Rules for maintaining it:

1. Use the status markers (✅ 🚧 ⬜ ⛔) and module tags (`[L] [M] [G]`) defined in its legend.
2. A checkbox is only checked when the wave's exit-gate test passes.
3. The Section 16 approval gate must be fully checked before any Wave 1 runtime code.
4. On every edit, refresh **Last updated** and **Current focus**.
5. Collapse completed waves to one line to keep the file scannable (< 30 s).
6. Never duplicate architecture detail — link `docs/architecture/0001-memoryos-vnext-architecture.md`.

---

## Performance Targets

Targets are hypotheses until measured:

| Operation | Target |
|---|---:|
| warm GovernanceProjection | P95 ≤ 5 ms |
| SearchCards | P95 ≤ 80 ms |
| filter/rank/dedupe | P95 ≤ 10 ms |
| bounded ReadEvidence | P95 ≤ 40 ms |
| Fast Recall total | P95 ≤ 150 ms (excluding host transport) |
| governance-only degradation | P95 ≤ 30 ms |
| Deep Recall | separate budget and SLO |

---

## Build & Development

### Quick Build

```bash
# Laputa
cd laputa
go mod tidy
go build ./...

# Mentle
cd ../mentle
go mod tidy
go build ./...

# Garden (depends on both)
cd ../garden
go mod tidy
go build -o garden.exe .
```

### Running Garden HTTP Server

```bash
cd garden
./garden.exe &
```

Default endpoint: `http://127.0.0.1:7373`

Health check:
```bash
curl -s http://127.0.0.1:7373/health
```

### Configuration

Garden uses environment variables and optional YAML config:

- `GARDEN_PIPELINE_CONFIG` — path to pipelines.yaml (default: `~/.garden/pipelines.yaml`)
- `GARDEN_RAG_BASE_URL` — OpenAI-compatible base URL (optional)
- `GARDEN_RAG_API_KEY` — API key for LLM planner (optional)
- `GARDEN_RAG_MODEL` — model name for planner (optional)

Without LLM env vars, Garden uses deterministic planner and reports degradation without failing.

---

## MANUAL

This document was created via the oh-my-claudecode writer agent.

When updating:

1. **Keep this file as the single source of truth** for root-level workspace coordination
2. **Link to vNext architecture** (`docs/architecture/0001-memoryos-vnext-architecture.md`) for detailed design decisions
3. **Do not duplicate** architecture details; cross-reference instead
4. **Update this file when:**
   - New top-level directories are added
   - Go versions or conventions change
   - Testing requirements evolve
   - HTTP contracts are finalized
   - Migration waves are completed
5. **Do not update this file for:**
   - Implementation details within a single module (use module AGENTS.md instead)
   - Temporary debugging or feature branches
   - Archive material (preserve in `docs/archive/`)

Depth-1 AGENTS.md files exist for:
- `laputa/AGENTS.md`
- `mentle/AGENTS.md`
- `garden/AGENTS.md`
- `docs/AGENTS.md`
- `.agents/AGENTS.md`
- `.omc/AGENTS.md`

Each describes its own subdirectories, testing, and build requirements.

For questions about the overall vision, architecture decisions, or module responsibilities, refer to the active design set:

- **Entry point:** `docs/README.md`
- **Complete plan:** `docs/architecture/0001-memoryos-vnext-architecture.md`
- **Module READMEs:** `{laputa,mentle,garden}/README.md`
