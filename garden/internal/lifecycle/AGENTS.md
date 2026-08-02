<!-- Parent: ../AGENTS.md -->

# garden/internal/lifecycle — Session and Activity Lifecycle Management

**Generated:** 2026-08-01  
**Purpose:** Session creation, event ingestion, and session termination with idempotency

---

## Purpose

The `lifecycle/` package manages session state and activity events:

- **Session creation** — initialize scope, start_time, session_id
- **Event ingestion** — normalize and store activity events
- **Session end** — mark session complete (idempotent on session_id + event_id)
- **Activity tracing** — record recall and evolution activities
- **Checkpoint state** — save intermediate session state

---

## Structure

```
lifecycle/
├── lifecycle.go      # Session and event lifecycle
└── lifecycle_test.go
```

---

## Key Concepts

### Session Lifecycle

```go
type Session struct {
    ID        string    // unique session identifier
    Scope     string    // scope (user, project, agent)
    StartTime time.Time
    EndTime   *time.Time
    Status    string    // "active", "ended"
    Events    []Event
}

type Event struct {
    ID        string
    Type      string    // "code_edit", "recall", "evolution", etc.
    Timestamp time.Time
    Data      map[string]any
}
```

### Create Session

Initialize a new session:
- Generate session_id
- Record start_time
- Set scope
- Status = "active"

### Ingest Event

Normalize and store activity:
- Parse event_type (code_edit, recall, evolution, etc.)
- Attach to session
- Write to activity store
- Idempotent on event_id within session

### End Session

Mark session complete:
- Set EndTime
- Status = "ended"
- Idempotent on session_id + final event_id
- Cannot reopen ended session

---

## Testing

```bash
cd garden
GOSUMDB=off go test -v ./internal/lifecycle/...
```

**Behavioral tests:**

- Session creation records start_time
- Events are ingested in order
- Session end is idempotent on session_id + event_id
- Cannot ingest events after session end
- Scope is preserved across lifecycle
- Status transitions are valid

---

## Conventions

- All timestamps in UTC
- Event IDs are unique within session
- Session ID format: deterministic or UUID (configurable)
- No mutation of past events
- All operations are logged

---

## MANUAL

Keep lifecycle focused on state management. Business logic for activity interpretation goes to router or rag packages.

Parent reference: ../AGENTS.md
