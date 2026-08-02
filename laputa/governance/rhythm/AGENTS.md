<!-- Parent: ../../AGENTS.md -->

# laputa/governance/rhythm — Rhythm Reports (Daily, Weekly, Monthly)

**Generated:** 2026-08-01  
**Purpose:** LLM-based periodic report generation for governance summaries

---

## Purpose

The `rhythm/` package generates periodic LLM-based reports:

- **Daily reports** — daily summary of activity and insights
- **Weekly reports** — weekly aggregation and trends
- **Monthly reports** — monthly reflection and patterns
- **LLM integration** — OpenAI-compatible endpoint for report generation
- **Report storage** — save to governance sections (07-daily, 08-weekly, 09-monthly)

---

## Structure

```
rhythm/
├── types.go          # RhythmKind, ReportResult, Config
├── prompt.go         # Prompt generation for LLM
├── prompt_test.go
├── generator.go      # Report generation service
├── generator_test.go
├── rhythm.go         # Rhythm engine (orchestration)
├── rhythm_test.go
└── integration_test.go
```

---

## Key Concepts

### Rhythm Kind

```go
type RhythmKind string
const (
    RhythmDaily   RhythmKind = "daily"
    RhythmWeekly  RhythmKind = "weekly"
    RhythmMonthly RhythmKind = "monthly"
)
```

### Report Result

```go
type ReportResult struct {
    Title         string    `json:"title"`
    Summary       string    `json:"summary"`
    Highlights    []string  `json:"highlights"`
    OpenQuestions []string  `json:"open_questions,omitempty"`
    GeneratedAt   time.Time `json:"generated_at"`
}
```

### Configuration

```go
type Config struct {
    APIKey  string
    BaseURL string
    Model   string
    Timeout time.Duration
}
```

### Generator Service

**Generate(ctx, kind):**
- Fetch input data (activity, memories, governance state)
- Prepare prompt via prompt.go
- Call OpenAI-compatible LLM
- Parse response into ReportResult
- Write to appropriate section (07, 08, or 09)
- Return result

---

## Prompt Generation

The `prompt.go` package generates deterministic prompts:

- Activity summary (what happened)
- Memory highlights (what was learned)
- Governance state snapshot (who/what changed)
- Prompt engineering ensures JSON response
- Temperature and max_tokens configurable

---

## Testing

```bash
cd laputa
GOSUMDB=off go test -v ./governance/rhythm/...
```

**Behavioral tests:**

- Prompt generation is deterministic
- Report generation calls LLM with correct parameters
- ReportResult is valid JSON
- Reports are stored to correct sections
- Timeout is enforced
- Mock LLM path produces valid results

---

## Conventions

- All reports are timestamped (UTC)
- Highlights are short strings (<100 chars each)
- Summary is bounded (max 1000 chars)
- No mutations outside rhythm package
- All LLM calls are timeoutted

---

## MANUAL

Keep rhythm focused on periodic reports. New report types require design review.

Parent reference: ../AGENTS.md
