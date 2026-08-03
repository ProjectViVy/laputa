<!-- Parent: ../AGENTS.md -->

# Garden — HTTP Gateway & Activity Orchestration

**Generated:** 2026-08-01  
**Module Path:** `github.com/dashimaki/garden`  
**Go Version:** 1.26.4 or later  
**Status:** Phase 5 active — governed agentic RAG, activity lifecycle, HTTP server

---

## Purpose

Garden is the unified HTTP entry point for Laputa governance and Mentle memory. It provides:

- **HTTP API v2** — canonical memory CRUD, ingest, Fast/Deep recall, governance, evolution, mailbox, admin
- **Activity orchestration** — session lifecycle, event ingestion, recall traces
- **Agentic RAG** — governed context resolution with optional LLM planner
- **Degradation** — graceful fallback when Mentle or LLM is unavailable

The legacy v1 CRUD translator and prefix router were removed (ADR-0008); the HTTP surface is v2-only.

---

## Module Structure

```
garden/
├── .git/                         # Garden submodule (independent repo)
├── config/
│   ├── config.example.yaml       # Example pipeline configuration
│   └── pipelines.yaml            # Runtime pipeline definitions
├── e2e/                          # End-to-end tests (requires -tags=e2e)
│   └── external_e2e_test.go
├── fixtures/                     # Test data and fixtures
├── internal/
│   ├── activity/                 # STM runtime, spool, checkpoints
│   ├── arbiter/                  # Conflict arbitration (read-only)
│   ├── authority/                # Governance projection and writer
│   ├── cognitive/                # MEMRULES loading
│   ├── evolution/                # EvoMap runs/proposals/events
│   ├── ingest/                   # Session ingestion and semantic units
│   ├── lifecycle/                # Session lifecycle management
│   ├── mailbox/                  # EvoMap inbox/outbox
│   ├── pipeline/                 # Pipeline orchestration
│   ├── rag/                      # Planner surface (deterministic + LLM)
│   ├── recall/                   # Fast/Deep recall and traces
│   ├── report/                   # Human-facing report system
│   ├── server/                   # HTTP handlers and routing
│   └── supervision/              # Process supervision and logging
├── main.go                       # HTTP server entry point
├── go.mod                        # Module: github.com/dashimaki/garden
├── go.sum
├── README.md
├── .gitignore
└── garden.exe / garden           # Compiled binary
```

### Subdirectories (Depth 2)

Depth-2 AGENTS.md files exist for:

- **[internal/AGENTS.md](./internal/AGENTS.md)** — server, recall, activity, ingest, evolution
- **[e2e/AGENTS.md](./e2e/AGENTS.md)** — end-to-end tests and external verification

---

## Key Concepts

### HTTP API Contract (v2 only)

```http
POST   /v2/memories                # Canonical memory create
GET    /v2/memories/{id}           # Canonical memory read
GET    /v2/memories                # Canonical memory list
PATCH  /v2/memories/{id}           # Canonical memory update
DELETE /v2/memories/{id}           # Canonical memory delete
POST   /v2/ingest/sessions         # Session-end ingestion
GET    /v2/ingestions/{id}         # Ingestion status
POST   /v2/recall/bootstrap        # Session bootstrap context

POST   /v2/recall/fast            # Fast recall (deterministic, no LLM)
POST   /v2/recall/deep            # Deep recall (explicit, with KG/graph)
GET    /v2/recall/traces/{trace_id} # Retrieve recall trace

POST   /v2/activity/events        # Normalized activity events
GET    /v2/activity/sessions/{id} # Session activity history

POST   /v2/governance/projection  # Read GovernanceProjection
POST   /v2/governance/proposals   # Create or review proposals

POST   /v2/evolution/runs         # Start evolution run
POST   /v2/evolution/proposals    # Submit evolution proposal
GET    /v2/evolution/proposals/{id} # Retrieve proposal details

GET    /v2/pipelines              # Pipeline inspection
GET    /health                    # Health check
```

### Agentic Recall

Combines:

1. **Fast Recall** — deterministic, no LLM, no KG, compact trace
2. **Deep Recall** — explicit expensive recall, optional LLM planner, full RecallTrace
3. **Planner** — deterministic (fallback) or LLM-based (OpenAI-compatible)

### Activity Lifecycle

Session lifecycle management:

- **Session creation** — `session_id`, `scope`, `start_time`
- **Event ingestion** — activity events, material ingest
- **Session end** — idempotent on `session_id + event_id`

---

## Build & Test

### Build

```bash
cd garden
go mod tidy
go build -o garden.exe .
```

### Test (Unit)

```bash
cd garden
GOSUMDB=off go test ./internal/...
```

### Test (E2E)

```bash
cd garden
GOSUMDB=off go test -tags=e2e ./e2e/...
```

### Run HTTP Server

```bash
./garden.exe &
# or
go run . &

# Verify health
curl -s http://127.0.0.1:7373/health
```

Default endpoint: `http://127.0.0.1:7373`

---

## Configuration

Garden uses environment variables and optional YAML config:

| Env Var | Purpose | Default |
|---------|---------|---------|
| `GARDEN_PIPELINE_CONFIG` | Path to pipelines.yaml | `~/.garden/pipelines.yaml` |
| `GARDEN_RAG_BASE_URL` | OpenAI-compatible base URL | (optional) |
| `GARDEN_RAG_API_KEY` | API key for LLM planner | (optional) |
| `GARDEN_RAG_MODEL` | Model name for planner | (optional) |
| `GARDEN_LOG_PATH` | Log file path | `~/.garden/garden.log` |

Without LLM env vars, Garden uses deterministic planner and reports degradation without failing.

### Pipeline Configuration (YAML)

Example `config/pipelines.yaml`:

```yaml
pipelines:
  fast_recall:
    enabled: true
    planner: deterministic
  deep_recall:
    enabled: true
    planner: openai
    openai_base_url: https://api.openai.com/v1
```

---

## Testing Requirements

Before deploying:

```bash
cd garden && GOSUMDB=off go test ./internal/...
GOSUMDB=off go test -tags=e2e ./e2e/...
```

**Exit gates:**

- SearchCards does not return full `Content`
- ReadEvidence enforces per-item and total character budget
- Superseded, deleted, expired cards do not reappear
- Fast Recall does not call Planner, KG, timeline or graph
- Deep Recall always emits `RecallTrace`
- Mentle unavailable leaves governance projection available
- Session end is idempotent on `session_id + event_id`
- Unauthorized authority mutation is rejected and audited

---

## Architecture Boundaries

**Ownership rule (immutable):**

```text
Garden may orchestrate.
Garden may not become authority.
Garden may read from Laputa (read-only governance projection).
Garden may read/write via Mentle facade (memory operations).
```

---

## Example Requests

### Fast Recall

```bash
curl -s -X POST http://127.0.0.1:7373/v2/recall/fast \
  -H "Content-Type: application/json" \
  -d '{
    "scope": "session-123",
    "intent": "What are the key decisions in my project?"
  }'
```

### Deep Recall

```bash
curl -s -X POST http://127.0.0.1:7373/v2/recall/deep \
  -H "Content-Type: application/json" \
  -d '{
    "scope": "session-123",
    "intent": "How do I debug this auth issue?",
    "use_kg": true,
    "use_timeline": true
  }'
```

### Activity Event

```bash
curl -s -X POST http://127.0.0.1:7373/v2/activity/events \
  -H "Content-Type: application/json" \
  -d '{
    "session_id": "session-123",
    "event_type": "code_edit",
    "timestamp": "2026-08-01T10:30:00Z",
    "data": {"file": "main.go", "lines": 42}
  }'
```

### Canonical Memory CRUD (v2)

```bash
# Create
curl -s -X POST http://127.0.0.1:7373/v2/memories \
  -H 'Content-Type: application/json' \
  -d '{"content":"decision text","kind":"decision","scope":"project:garden"}'

# Read
curl -s http://127.0.0.1:7373/v2/memories/mem_<id>

# List
curl -s "http://127.0.0.1:7373/v2/memories?kind=decision&limit=10"

# Update
curl -s -X PATCH http://127.0.0.1:7373/v2/memories/mem_<id> \
  -H 'Content-Type: application/json' \
  -d '{"content":"updated text","expected_version":1}'

# Delete
curl -s -X DELETE http://127.0.0.1:7373/v2/memories/mem_<id>
```

---

## Conventions

- **Go formatting:** standard `gofmt`
- **Error handling:** explicit, structured error responses
- **Concurrency:** request-scoped contexts, graceful shutdown
- **Logging:** structured, optional; trace IDs for request correlation
- **HTTP:** JSON request/response bodies, standard status codes

---

## Migration Waves

Garden is staged across 7 waves. Current status:

| Wave | Goal | Status |
|------|------|--------|
| 0 | Document and baseline | ✓ Complete |
| 1 | Card/Evidence API | ✓ Complete |
| 2 | Fast Recall Core | ✓ Complete |
| 3 | STM Runtime and session semantics | ✓ Complete |
| 4 | Governance application boundary | ✓ Complete |
| 5 | Deep Recall and logical arbiter | ✓ Active |
| 6 | EvoMap/Evolver adapter | Planned |
| 7 | Host adapters and release hardening | Planned |

---

## MANUAL

This document is maintained by the oh-my-claudecode writer agent.

When updating:

1. **Keep this file as single source of truth** for garden module organization
2. **Link to internal/, e2e/ subdirectories** for detailed implementations
3. **Do not duplicate** HTTP contract or router logic
4. **Update when:**
   - New HTTP routes are added
   - Configuration options change
   - Testing requirements evolve
   - Migration waves are completed
   - Dependencies are added/removed
5. **Do not update for:**
   - Implementation details (use internal/AGENTS.md instead)
   - Temporary feature branches
   - Archive material (preserve in docs/archive/)

Parent reference: `../AGENTS.md`
