<!-- Parent: ../AGENTS.md -->

# garden/internal/report — Activity and Recall Reporting

**Generated:** 2026-08-01  
**Purpose:** Generate reports from activity and recall traces

---

## Purpose

The `report/` package generates human-readable summaries:

- **Activity report** — what happened in a session (events, changes)
- **Recall report** — trace of recall operations (planner decisions, cards found)
- **Trace export** — serialize RecallTrace and ActivityTrace for storage
- **Aggregation** — summarize across multiple sessions or time windows

---

## Structure

```
report/
├── service.go       # Report generation service
└── service_test.go
```

---

## Key Concepts

### Activity Report

Summarizes session activity:
```go
type ActivityReport struct {
    SessionID string
    StartTime time.Time
    EndTime   *time.Time
    EventCount int
    Summary   string
    Highlights []string
}
```

### Recall Report

Traces a recall operation:
```go
type RecallReport struct {
    TraceID      string
    Intent       string
    Mode         string        // "fast" or "deep"
    PlannerMode  string        // "deterministic" or "llm"
    CardsFound   int
    EvidenceChars int
    Duration     time.Duration
}
```

### Service Operations

**GenerateActivityReport(ctx, sessionID):**
- Fetch activity trace
- Summarize events
- Extract highlights
- Return ActivityReport

**GenerateRecallReport(ctx, traceID):**
- Fetch recall trace
- Summarize planner decisions
- Report card/evidence counts
- Return RecallReport

---

## Testing

```bash
cd garden
GOSUMDB=off go test -v ./internal/report/...
```

**Behavioral tests:**

- Activity reports contain accurate event counts
- Recall reports trace planner decisions
- Reports are human-readable
- Reports preserve all important details

---

## Conventions

- Reports are generated on-demand, not persisted
- All timestamps are UTC
- Reports are immutable (generated once)

---

## MANUAL

Keep reports focused on summarization. Storage goes to lifecycle or persistence layer.

Parent reference: ../AGENTS.md
