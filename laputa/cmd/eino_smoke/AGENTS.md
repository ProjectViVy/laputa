<!-- Parent: ../../AGENTS.md -->

# laputa/cmd/eino_smoke — Smoke Test for LLM Integration

**Generated:** 2026-08-01  
**Purpose:** Quick verification of eino + OpenAI integration for rhythm reporting

---

## Purpose

The `eino_smoke` command is a minimal smoke test that verifies:

- **Eino framework** loads and initializes correctly
- **OpenAI integration** connects and authenticates
- **Rhythm report generation** produces valid ReportResult
- **Mock LLM path** works without API key

---

## Structure

```
eino_smoke/
├── main.go                          # Entry point and smoke test logic
└── (supporting utilities as needed)
```

---

## Build

```bash
cd laputa/cmd/eino_smoke
go build -o eino_smoke .
```

---

## Run

### Mock Path (No API Key)

```bash
./eino_smoke -kind daily
```

Output: Generates a mock ReportResult without calling OpenAI.

### Real LLM Path

```bash
export OPENAI_API_KEY=sk-...
./eino_smoke -kind daily
```

Output: Generates a ReportResult using real LLM.

**Supported kinds:**
- `daily`
- `weekly`
- `monthly`

---

## Output

Produces JSON ReportResult:

```json
{
  "title": "Daily Report",
  "summary": "...",
  "highlights": [...],
  "open_questions": [...],
  "generated_at": "2026-08-01T10:30:00Z"
}
```

---

## Exit Codes

- `0` — success
- `1` — initialization failed
- `2` — API call failed
- `3` — invalid kind argument

---

## MANUAL

Smoke test for verification only. Not part of production workflow.

Parent reference: ../../AGENTS.md
