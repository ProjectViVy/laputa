<!-- Parent: ../AGENTS.md -->

# garden/internal/pipeline — Pipeline Orchestration & Configuration

**Generated:** 2026-08-01  
**Purpose:** Define and execute recall and activity processing pipelines

---

## Purpose

The `pipeline/` package orchestrates multi-stage pipelines:

- **Fast recall pipeline** — deterministic, no LLM
- **Deep recall pipeline** — explicit, expensive, with KG/timeline
- **Activity ingest pipeline** — normalize and process activity events
- **Configuration parsing** — YAML pipeline definitions
- **Stage execution** — run stages sequentially or in parallel

---

## Structure

```
pipeline/
├── pipeline.go      # Pipeline execution engine
├── pipeline_test.go
└── config.go        # Configuration parsing (YAML)
```

---

## Key Concepts

### Pipeline Definition

```go
type Pipeline struct {
    Name   string
    Stages []Stage
}

type Stage struct {
    Name    string
    Type    string      // "search", "rank", "filter", "merge", "trace"
    Config  map[string]any
    Timeout time.Duration
}
```

### Built-in Pipelines

**Fast Recall:**
1. Search (deterministic, no LLM)
2. Filter (policy, authority)
3. Rank (BM25, recency)
4. Dedupe
5. Truncate

**Deep Recall:**
1. Plan (optional LLM)
2. Search (expanded scope)
3. Graph (knowledge graph traversal)
4. Timeline (temporal relationships)
5. Filter (policy)
6. Rank (combined scoring)
7. Dedupe
8. Trace (RecallTrace generation)

**Activity Ingest:**
1. Normalize (standardize event schema)
2. Validate (type, timestamp, data)
3. Enrich (add metadata)
4. Store (write to activity backend)

### Configuration (YAML)

```yaml
pipelines:
  fast_recall:
    enabled: true
    stages:
      - name: search
        type: search
        timeout: 80ms
      - name: filter
        type: filter
        timeout: 10ms
  deep_recall:
    enabled: true
    stages:
      - name: plan
        type: plan
        timeout: 5s
```

---

## Testing

```bash
cd garden
GOSUMDB=off go test -v ./internal/pipeline/...
```

**Behavioral tests:**

- Fast recall pipeline executes in ~150ms P95
- Deep recall pipeline generates RecallTrace
- Configuration parses from YAML
- Stage timeouts are enforced
- Parallel stages execute concurrently
- Pipeline errors are propagated

---

## Conventions

- Stage names are unique within pipeline
- Timeouts are cumulative per pipeline
- Configuration is read-only after startup
- Errors in stages are wrapped with context

---

## MANUAL

Keep pipelines declarative. New stage types go to the execution engine, not configuration.

Parent reference: ../AGENTS.md
