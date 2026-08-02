<!-- Parent: ../AGENTS.md -->

# garden/internal/rag — Agentic Recall Planning & LLM Integration

**Generated:** 2026-08-01  
**Purpose:** Recall orchestration with deterministic and LLM-based planning

---

## Purpose

The `rag/` package implements recall planning and LLM integration:

- **Deterministic planner** — keyword matching, lexical ranking (always available)
- **OpenAI-compatible LLM planner** — configurable endpoint for advanced intent resolution
- **Governance policy enforcement** — respect denied sources, wings, rooms
- **Graceful degradation** — deterministic planner used if LLM unavailable
- **Service interface** — high-level recall API used by router

---

## Structure

```
rag/
├── planner.go          # Planner interface (deterministic + LLM)
├── planner_test.go
├── openai.go           # OpenAI-compatible LLM adapter
├── openai_test.go
├── policy.go           # Governance policy enforcement
├── service.go          # High-level recall service
├── service_test.go
└── types.go            # Shared types (PlanResult, RecallRequest, etc.)
```

---

## Key Components

### Planner Interface

```go
type Planner interface {
    Plan(ctx context.Context, intent string, scope string) (PlanResult, error)
    Degraded() bool
}

type PlanResult struct {
    SearchTerms  []string
    Wings        []string
    Rooms        []string
    UseKG        bool
    UseTimeline  bool
    EstimatedCost int
}
```

### Deterministic Planner

Always available fallback:
- Keyword extraction from intent
- Frequency analysis of terms
- Simple wing/room suggestions
- No external calls

### OpenAI Planner

LLM-based planning for complex intent:
- Configurable base URL, model, API key
- Prompt engineering for deterministic output
- Timeout protection
- Automatic fallback to deterministic planner on error
- Degradation flag in response

### Policy Enforcement

Applies governance constraints:
- Filter wings by authority
- Exclude denied sources
- Respect scope restrictions
- Budget enforcement (token/cost limits)

### Recall Service

High-level API combining planner + policy + backend:
- FastRecall(ctx, request) — deterministic, no LLM, ~150ms
- DeepRecall(ctx, request) — explicit expensive recall, with full trace
- Both return ContextView with cards, evidence, and trace

---

## Configuration

Environment variables:
- `GARDEN_RAG_BASE_URL` — OpenAI-compatible endpoint
- `GARDEN_RAG_API_KEY` — API key
- `GARDEN_RAG_MODEL` — Model name (default: gpt-4)
- `GARDEN_RAG_TIMEOUT` — LLM request timeout (default: 10s)

---

## Testing

```bash
cd garden
GOSUMDB=off go test -v ./internal/rag/...
```

**Behavioral tests:**

- Deterministic planner runs without external calls
- OpenAI planner respects timeout and falls back to deterministic
- Policy filtering removes denied sources
- Wing/room filtering respects authority
- FastRecall never calls planner or KG
- DeepRecall always generates RecallTrace
- Degradation flag accurate when LLM unavailable

---

## Conventions

- Planner interface allows dependency injection for testing
- All LLM calls are timeoutted
- Errors are wrapped with context
- Degradation is reported, not silent

---

## MANUAL

Keep policy enforcement separate from planner logic. New LLM integrations go to openai.go or new adapter files.

Parent reference: ../AGENTS.md
