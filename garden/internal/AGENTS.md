<!-- Parent: ../AGENTS.md -->

# garden/internal — Core Recall, Activity, and Routing

**Generated:** 2026-08-01  
**Purpose:** HTTP routing, recall orchestration, activity lifecycle, and Mentle integration

---

## Purpose

The `internal/` directory contains Garden's core business logic:

- **HTTP request routing** by key prefix (section: vs memory:)
- **Fast and Deep Recall** orchestration
- **Activity lifecycle** management (sessions, events, traces)
- **LLM planner** integration (OpenAI-compatible)
- **Governance projection** reading (read-only from Laputa)
- **Process supervision** and graceful degradation

---

## Structure

```
internal/
├── crud/                          # CRUD translator (v1 legacy compatibility)
│   ├── crud.go                    # Handler interface and routing logic
│   └── crud_test.go               # Unit tests for CRUD behavior
├── lifecycle/                     # Session and event lifecycle
│   ├── lifecycle.go               # Session creation, event ingestion, end
│   └── lifecycle_test.go
├── pipeline/                      # Pipeline orchestration
│   ├── pipeline.go                # Pipeline definition and execution
│   ├── pipeline_test.go
│   └── config.go                  # Pipeline configuration parsing
├── rag/                           # Agentic RAG (recall planning)
│   ├── planner.go                 # Planner interface (deterministic + LLM)
│   ├── planner_test.go
│   ├── openai.go                  # OpenAI-compatible LLM adapter
│   ├── openai_test.go
│   └── policy.go                  # Governance policy enforcement
├── router/                        # HTTP request routing
│   ├── router.go                  # Main router logic
│   ├── router_test.go
│   ├── governance.go              # Governance projection reading
│   ├── mentle_adapter.go          # Mentle backend adapter
│   └── mentle_adapter_test.go
└── supervision/                   # Process supervision and logging
    ├── supervision.go             # Shutdown, logging, metrics
    └── supervision_test.go
```

### Subdirectories (Depth 3)

No formal depth-3 AGENTS.md required; implementation details are documented inline.

---

## Key Components

### Router

Routes requests by key prefix:

| Prefix | Backend | Purpose |
|--------|---------|---------|
| `section:` | Laputa (governance) | Authority, lifecycle, policy |
| `memory:` | Mentle | Material, evidence, retrieval |

All requests go through the unified router before reaching backend-specific handlers.

### Recall

Implements two modes:

1. **Fast Recall** — deterministic, no LLM, no KG, ~150ms P95
2. **Deep Recall** — explicit expensive recall, optional LLM planner, full trace

Both return RecallResponse with cards and evidence.

### Activity Lifecycle

Manages session state:

- Session creation (scope, start_time)
- Event ingestion (normalized activity events)
- Session end (idempotent on session_id + event_id)

### LLM Planner

Deterministic planner (always available) or LLM-based:

- **Deterministic:** keyword matching, lexical ranking, no external calls
- **OpenAI-compatible:** configurable endpoint, model, API key

Planner helps disambiguate intent for recall target selection.

### Governance Adapter

Read-only access to Laputa governance:

- GovernanceProjection for current scope
- Authority checks (who can write, read, delete)
- Policy enforcement (denied sources, wings, rooms)

### Mentle Adapter

Facade to Mentle backend:

- SearchCards (with policy filtering)
- ReadEvidence (with budget enforcement)
- StoreActivity (for event ingestion)

---

## Build & Test

### Build

```bash
cd garden
go mod tidy
go build ./internal/...
```

### Test (Unit)

```bash
cd garden
GOSUMDB=off go test ./internal/...
```

### Test (Specific Package)

```bash
GOSUMDB=off go test -v ./internal/router/...
GOSUMDB=off go test -v ./internal/rag/...
```

---

## Key Interfaces

### Router Interface

```go
type Backend interface {
    Write(ctx context.Context, key, value string, meta map[string]any) (string, error)
    Read(ctx context.Context, key string) (map[string]any, error)
    List(ctx context.Context, prefix string, limit int) ([]map[string]any, error)
    Forget(ctx context.Context, key string) (bool, error)
}
```

### Planner Interface

```go
type Planner interface {
    Plan(ctx context.Context, intent string, scope string) (PlanResult, error)
    Degraded() bool
}
```

---

## Testing Requirements

Before starting feature work:

```bash
cd garden && GOSUMDB=off go test ./internal/...
```

**Mandatory behavioral tests:**

- SearchCards does not return full Content
- ReadEvidence enforces per-item and total character budget
- Fast Recall does not call Planner, KG, or graph
- Deep Recall always emits RecallTrace
- Session end is idempotent on session_id + event_id
- Unauthorized mutations are rejected with audit

---

## Conventions

- **Go formatting:** standard `gofmt`
- **Error handling:** explicit, wrapped with context
- **Concurrency:** request-scoped contexts, graceful shutdown
- **Logging:** structured, optional; trace IDs for correlation
- **HTTP:** JSON request/response bodies, standard status codes

---

## MANUAL

When updating:

1. Keep each package focused on one responsibility
2. Link to subdirectory implementations for details
3. Do not duplicate router logic or recall algorithms
4. Update when:
   - New HTTP routes are added
   - Planner behavior changes
   - Activity semantics evolve
   - Governance policy changes
5. Do not update for:
   - Implementation details (use package README instead)
   - Temporary branches

Parent reference: ../AGENTS.md
