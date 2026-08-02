<!-- Parent: ../../AGENTS.md -->

# mentle/internal/miner — Project & Conversation Mining

**Generated:** 2026-08-01  
**Purpose:** Extract memories from code, docs, and conversations

---

## Purpose

The `miner/` package extracts structured memories from diverse sources:

- **Project mining** — code files, documentation, commit history
- **Conversation mining** — chat logs, dialogue transcripts
- **Memory extraction** — identify decisions, patterns, insights
- **Categorization** — assign to wings and rooms
- **Normalization** — standardize memory format across sources

---

## Structure

```
miner/
├── miner.go          # Main mining coordinator
├── miner_test.go
├── convo_miner.go    # Conversation-specific mining
├── normalize.go      # Memory normalization
└── (supporting utilities)
```

---

## Key Concepts

### Mining Request

```go
type MiningRequest struct {
    SourceType string           // "code", "doc", "conversation"
    SourceURI  string
    Content    string
    Metadata   map[string]any
    Scope      string
}
```

### Extracted Memory

```go
type ExtractedMemory struct {
    Title      string
    Summary    string
    Wing       string
    Room       string
    Entities   []Entity
    Decisions  []Decision
    Insights   []string
    Timestamp  time.Time
}
```

### Mining Strategies

**Code Mining:**
- Identify functions, classes, modules
- Extract comments and docstrings
- Parse git diffs for changes
- Flag technical decisions

**Conversation Mining:**
- Extract speaker turns
- Identify decisions and commitments
- Extract questions and answers
- Detect emotion and tone

**Doc Mining:**
- Extract sections and hierarchy
- Identify key concepts
- Extract references and links
- Flag actionable items

---

## Testing

```bash
cd mentle
GOSUMDB=off go test -v ./internal/miner/...
```

**Behavioral tests:**

- Code mining extracts function names
- Conversation mining preserves speaker
- Doc mining respects hierarchy
- Normalization produces consistent format
- Wing/room assignment matches scope

---

## Conventions

- All extracted memories are timestamped
- Source URI is preserved
- Extraction is deterministic
- No mutation of source materials

---

## MANUAL

Keep miner focused on extraction. Relationship inference and storage go to facade.

Parent reference: ../AGENTS.md
