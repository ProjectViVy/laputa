<!-- Parent: ../../AGENTS.md -->

# mentle/internal/sanitizer — Input Sanitization & Validation

**Generated:** 2026-08-01  
**Purpose:** Clean and validate all input data

---

## Purpose

The `sanitizer/` package ensures input safety:

- **Content sanitization** — remove malicious markup, scripts
- **Size validation** — enforce reasonable limits
- **Format validation** — ensure well-formed data
- **Encoding normalization** — handle encoding issues
- **Injection prevention** — protect against injection attacks

---

## Structure

```
sanitizer/
├── sanitizer.go      # Sanitization service
└── sanitizer_test.go
```

---

## Key Functions

**SanitizeText(input string) string:**
- Remove HTML/XML tags
- Decode entities
- Normalize whitespace
- Return safe text

**ValidateCardSize(card MemoryCard) error:**
- Check title length
- Check content length
- Check summary length
- Enforce limits

**ValidateInput(input map[string]any) error:**
- Type checking
- Format validation
- Required field checking

---

## Testing

```bash
cd mentle
GOSUMDB=off go test -v ./internal/sanitizer/...
```

**Behavioral tests:**

- HTML tags removed
- Injection attempts blocked
- Size limits enforced
- Encoding normalized

---

## Conventions

- HTML tags always removed
- Maximum content size: 1MB
- Maximum title size: 512 chars
- No exceptions for any input

---

## MANUAL

Apply sanitizer to all external input before processing.

Parent reference: ../AGENTS.md
