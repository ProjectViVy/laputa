<!-- Parent: ../../AGENTS.md -->

# mentle/internal/entity — Entity Detection & Extraction

**Generated:** 2026-08-01  
**Purpose:** Extract entities (people, concepts, technical items) from text

---

## Purpose

The `entity/` package identifies and extracts entities:

- **Named entity recognition** — people, organizations, locations
- **Technical entity extraction** — modules, APIs, data structures
- **Concept extraction** — abstract ideas and patterns
- **Cross-reference detection** — links between entities
- **Entity normalization** — canonicalize entity names

---

## Structure

```
entity/
├── detector.go       # Entity detection service
├── detector_test.go
└── (supporting utilities)
```

---

## Key Concepts

### Entity Types

```go
const (
    EntityPerson       = "person"
    EntityOrganization = "organization"
    EntityLocation     = "location"
    EntityModule       = "module"
    EntityAPI          = "api"
    EntityConcept      = "concept"
    EntityDecision     = "decision"
    EntityTechnique    = "technique"
)
```

### Extracted Entity

```go
type Entity struct {
    Name        string
    Type        string
    Aliases     []string
    Confidence  float64
    Context     string
    References  []string
}
```

### Detection Process

1. **Linguistic parsing** — identify noun phrases and proper nouns
2. **Pattern matching** — recognize technical terms (CamelCase, CONSTANT_CASE)
3. **Frequency analysis** — promote repeated mentions
4. **Coreference resolution** — connect pronouns to entities
5. **Deduplication** — merge equivalent entities

---

## Testing

```bash
cd mentle
GOSUMDB=off go test -v ./internal/entity/...
```

**Behavioral tests:**

- Named entities detected correctly
- Technical terms recognized
- Confidence scores reasonable
- Dedup merges equivalent entities
- Aliases captured

---

## Conventions

- Confidence 0.0-1.0
- Names are normalized to title case
- All contexts are preserved for debugging

---

## MANUAL

Keep entity detection focused on extraction. Relationship inference goes to kg package.

Parent reference: ../AGENTS.md
