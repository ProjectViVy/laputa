<!-- Parent: ../../AGENTS.md -->

# mentle/internal/extractor — Memory Extraction from Sources

**Generated:** 2026-08-01  
**Purpose:** Extract structured memories from diverse source materials

---

## Purpose

The `extractor/` package extracts memories from raw materials:

- **Pattern-based extraction** — rules for decisions, techniques, insights
- **LLM-optional extraction** — use external LLM for complex reasoning
- **Normalization** — standardize extracted format
- **Deduplication** — avoid duplicate extractions
- **Source attribution** — track extraction provenance

---

## Structure

```
extractor/
├── general.go        # General-purpose extractor
└── general_test.go
```

---

## Key Concepts

### Extraction Request

```go
type ExtractionRequest struct {
    SourceType string           // "code", "doc", "conversation"
    Content    string
    Context    map[string]any   // metadata
}
```

### Extracted Item

```go
type ExtractedItem struct {
    Type        string         // "decision", "technique", "insight", "question"
    Title       string
    Description string
    Evidence    string         // supporting text from source
    Confidence  float64        // 0.0-1.0
    SourceRef   string         // reference back to source
}
```

### Extraction Strategies

**Decision Extraction:**
- Pattern: "decided to", "we will", "must"
- Extract: choice made, rationale, alternatives

**Technique Extraction:**
- Pattern: "approach", "method", "pattern", "best practice"
- Extract: name, steps, applicable context

**Insight Extraction:**
- Pattern: "learned", "discovered", "realized", "key insight"
- Extract: observation, implication, action

**Question Extraction:**
- Pattern: "?", "what", "how", "why"
- Extract: question, context, urgency

---

## Testing

```bash
cd mentle
GOSUMDB=off go test -v ./internal/extractor/...
```

**Behavioral tests:**

- Decisions extracted with high confidence
- Techniques identified correctly
- Source reference accurate
- Dedup prevents duplicates

---

## Conventions

- Confidence 0.0-1.0
- All extracted items include source reference
- No mutation of source materials

---

## MANUAL

Keep extractor focused on pattern matching. Storage and indexing go elsewhere.

Parent reference: ../AGENTS.md
