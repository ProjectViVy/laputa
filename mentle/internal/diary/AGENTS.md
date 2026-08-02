<!-- Parent: ../../AGENTS.md -->

# mentle/internal/diary — AAAK Diary (Agent-Specific Entries)

**Generated:** 2026-08-01  
**Purpose:** Structured symbolic summaries for agent learning diaries

---

## Purpose

The `diary/` package stores structured diary entries:

- **Entity codes** — 3-letter uppercase identifiers (KAI, MAX, PRI)
- **Topic frequency** — track important topics by occurrence count
- **Emotion codes** — symbolic emotions (vul, joy, fear, trust, grief, wonder, rage)
- **Flag codes** — mark special entries (DECISION, ORIGIN, CORE, PIVOT, TECHNICAL)
- **Agent-specific** — diary entries tailored to agent identity

---

## Structure

```
diary/
├── diary.go          # Diary service
└── diary_test.go
```

---

## Key Concepts

### Diary Entry

```go
type DiaryEntry struct {
    ID         string
    Timestamp  time.Time
    Entities   []EntityCode    // [KAI, MAX, PRI]
    Topics     map[string]int  // topic -> frequency
    Emotions   []EmotionCode   // [vul, joy, fear, trust, grief, wonder, rage]
    Flags      []FlagCode      // [DECISION, ORIGIN, CORE, PIVOT, TECHNICAL]
    Summary    string
}

type EntityCode string
type EmotionCode string
type FlagCode string
```

### Valid Codes

**Entity Codes:** 3-letter uppercase (KAI, MAX, PRI, etc.)

**Emotion Codes:** vul, joy, fear, trust, grief, wonder, rage, disgust, surprise, anticipation

**Flag Codes:** DECISION (important decision), ORIGIN (origin story), CORE (core identity), PIVOT (turning point), TECHNICAL (technical insight)

### Operations

**Record(ctx, entry):**
- Store diary entry
- Index by timestamp
- Index by entity and flag for search

**Query(ctx, entityCode):**
- Find entries mentioning entity
- Return with highest frequency topics

**Search(ctx, flag):**
- Find entries with specific flag
- Return in chronological order

---

## Testing

```bash
cd mentle
GOSUMDB=off go test -v ./internal/diary/...
```

**Behavioral tests:**

- Entity codes are 3 letters
- Emotions are valid
- Flags are recognized
- Topic frequency tracked
- Queries return correct entries

---

## Conventions

- All timestamps UTC
- Topic names lowercase
- Entity codes uppercase
- No mutation of past entries

---

## MANUAL

Keep diary focused on structured storage. Interpretation and learning go elsewhere.

Parent reference: ../AGENTS.md
