<!-- Parent: ../AGENTS.md -->

# mentle/internal — Core Search, Mining, and Entity Detection

**Generated:** 2026-08-01  
**Purpose:** Internal subsystems for search, memory extraction, entity detection, and palace operations

---

## Purpose

The `internal/` directory contains Mentle's core subsystems (not exposed in the public facade):

- **Search** — query processing and ranking
- **Mining** — extract memories from project files and conversations
- **Entity detection** — identify entities, relationships, and topics
- **BM25** — lexical ranking implementation
- **Palace management** — palace structure operations
- **Configuration** — runtime configuration and defaults
- **Sanitization** — input cleaning and validation

---

## Subdirectories (Depth 3+)

```
internal/
├── bm25/                         # BM25 lexical ranking
├── config/                       # Configuration management
├── diary/                        # AAAK diary (agent-specific entries)
├── dialect/                      # Text dialect handling
├── entity/                       # Entity detection and extraction
├── extractor/                    # Memory extraction from sources
├── instructions/                 # Usage instructions and prompts
├── kg/                           # Knowledge graph operations (see graph/AGENTS.md)
├── miner/                        # Project and conversation mining
├── palace/                       # Palace structure operations
├── registry/                     # Memory registry and indexing
├── room/                         # Room detection and categorization
├── sanitizer/                    # Input sanitization
└── search/                       # Query processing and ranking
```

### Documented Subdirectories

- **[kg/AGENTS.md](./kg/AGENTS.md)** — Knowledge graph backend (temporal RDF)

---

## Key Subsystems

### Search

Query processing, tokenization, scoring:

- Normalize and tokenize queries
- Expand with synonyms if configured
- Apply policy filtering
- Rank and deduplicate results

### Mining

Extract memories from diverse sources:

- **Project mining** — code files, docs, commit history
- **Conversation mining** — chat logs, dialogue transcripts
- **Extraction** — identify key entities, decisions, patterns
- **Categorization** — assign to wings and rooms

### Entity Detection

Identify and extract entities from text:

- **Named entities** — people, organizations, locations
- **Technical entities** — modules, APIs, data structures
- **Concepts** — abstract ideas, patterns, techniques
- **Cross-references** — links between entities

### Diary (AAAK)

Structured symbolic summaries for agent diaries:

- **Entity codes** — 3-letter uppercase (KAI, MAX, PRI)
- **Topics** — frequency-based with proper noun boosting
- **Emotion codes** — vul, joy, fear, trust, grief, wonder, rage, etc.
- **Flag codes** — DECISION, ORIGIN, CORE, PIVOT, TECHNICAL

### Palace Operations

Directory and navigation management:

- Create wings and rooms
- Store drawers (memory items)
- Retrieve by path
- Navigate hierarchy

### BM25

Lexical ranking for keyword-based retrieval:

- Term frequency (TF) and inverse document frequency (IDF)
- BM25 scoring with configurable K1 and b parameters
- Corpus statistics maintenance

---

## Build & Test

### Build

```bash
cd mentle
go build ./internal/...
```

### Test

```bash
cd mentle
GOSUMDB=off go test ./internal/...
```

### Test Specific Subsystem

```bash
GOSUMDB=off go test -v ./internal/entity/...
GOSUMDB=off go test -v ./internal/miner/...
GOSUMDB=off go test -v ./internal/bm25/...
```

---

## Testing Requirements

Before starting feature work:

```bash
cd mentle && GOSUMDB=off go test ./internal/...
```

**Mandatory behavioral tests:**

- Entity detection extracts entities correctly
- Mining produces valid memory structures
- BM25 scoring is deterministic
- Sanitization removes malicious input safely
- Palace operations maintain hierarchy integrity

---

## Conventions

- **Go formatting:** standard `gofmt`
- **No direct use from external packages** — use facade instead
- **Configuration:** environment variables and YAML
- **Logging:** internal, optional; no spam
- **Error handling:** explicit, wrapped with context

---

## MANUAL

When updating:

1. Keep internal subsystems focused on single responsibility
2. Expose operations via facade (mentle/facade/)
3. Do not add external API surface to internal/ packages
4. Update when:
   - New mining strategies are added
   - Entity detection is improved
   - Palace operations evolve
5. Do not update for:
   - Individual configuration tuning
   - Temporary debugging

Parent reference: ../AGENTS.md
