<!-- Parent: ../../AGENTS.md -->

# mentle/internal/dialect — Text Dialect Handling

**Generated:** 2026-08-01  
**Purpose:** Handle different text representations (AAAK, standard, etc.)

---

## Purpose

The `dialect/` package manages text representation variants:

- **AAAK dialect** — symbolic/structured form for agent learning
- **Standard dialect** — human-readable natural language
- **Encoding** — convert between representations
- **Decoding** — parse encoded forms
- **Round-trip** — ensure no loss during conversion

---

## Structure

```
dialect/
├── encoder.go        # Encoding service
└── encoder_test.go
```

---

## Key Concepts

### Dialects

**AAAK Dialect:**
```
ENTITIES[KAI,MAX] EMOTIONS[joy,wonder] FLAGS[DECISION,TECHNICAL] TOPIC[auth] 
SUMMARY: established oauth2 pattern
REFS: project/auth/token-validation
```

**Standard Dialect:**
```
Entity mentions: KAI, MAX
Emotions: joy, wonder
This is a decision about technical matters.
Summary: established oauth2 pattern
References: project/auth/token-validation
```

### Encoding

**StandardToAAAK(text string, metadata):**
- Extract entities, emotions, flags
- Generate AAAK representation

**AAKToStandard(aaak string):**
- Parse AAAK form
- Generate readable form

---

## Testing

```bash
cd mentle
GOSUMDB=off go test -v ./internal/dialect/...
```

**Behavioral tests:**

- Round-trip conversion preserves data
- Encoding is deterministic
- Decoding works for all valid forms
- Malformed input handled gracefully

---

## Conventions

- All dialects preserve meaning
- Round-trip is lossless
- Encoding is deterministic

---

## MANUAL

Keep dialect focused on representation conversion. Storage goes elsewhere.

Parent reference: ../AGENTS.md
