<!-- Parent: ../AGENTS.md -->

# mentle/facade — Memory Card and Evidence API

**Generated:** 2026-08-01  
**Purpose:** High-level facade for memory card discovery and evidence reading

---

## Purpose

The `facade/` package provides a clean, policy-aware API for Garden and external clients:

- **SearchCards** — discover memory candidates without revealing full content
- **ReadEvidence** — retrieve bounded evidence fragments with character budgets
- **StoreActivity** — ingest and normalize activity events into memory
- **Lifecycle management** — card status (active, superseded, deleted, expired)

The facade enforces that:
- Cards are searchable but content-limited (no full Content field)
- Evidence is always bounded by per-item and total character budgets
- Superseded, deleted, and expired cards are never returned
- All operations enforce scope and validity checks

---

## Structure

```
facade/
├── cards.go                       # SearchCards implementation
├── cards_test.go
├── evidence.go                    # ReadEvidence implementation
├── evidence_test.go
├── activity.go                    # Activity event ingestion
├── activity_test.go
├── policy.go                      # Scope and validity checking
├── policy_test.go
└── types.go                       # Common types and interfaces
```

### Subdirectories (Depth 3)

No formal depth-3 AGENTS.md required; implementations are self-contained.

---

## Key Operations

### SearchCards

Discover memory candidates by query:

```go
func (f *Facade) SearchCards(ctx context.Context, query string, scope string, maxResults int) ([]MemoryCard, error)
```

Returns:
- MemoryCard structs (ID, Kind, Collection, Title, Summary only — NO Content)
- Ranked by semantic similarity + heat score + recency
- Filtered to active, in-scope, unexpired cards
- Limited to maxResults

### ReadEvidence

Retrieve bounded evidence fragments:

```go
func (f *Facade) ReadEvidence(ctx context.Context, cardID string, budgetPerItem int, budgetTotal int) ([]EvidenceFragment, error)
```

Returns:
- EvidenceFragment structs (excerpts, source URIs, offsets)
- Enforced per-item budget (excerpt truncated to budgetPerItem chars)
- Enforced total budget (sum of all excerpts ≤ budgetTotal chars)
- Stops adding fragments when total budget is exhausted

### StoreActivity

Ingest and normalize activity events:

```go
func (f *Facade) StoreActivity(ctx context.Context, sessionID string, event ActivityEvent) error
```

Normalizes event data, extracts memories if applicable, stores in appropriate collection.

---

## Data Types

### MemoryCard

Candidate card for recall (no full content):

```go
type MemoryCard struct {
    ID            string
    Kind          string              // "document", "decision", "snippet", etc.
    Collection    string              // "architecture", "debugging", "timeline", etc.
    Scope         string              // session, project, or global
    Title         string              // concise label
    Summary       string              // bounded summary for discovery
    SourceRef     string              // reference to origin (file, URL, etc.)
    Revision      int                 // edit count
    Status        string              // "active", "superseded", "deleted", "expired"
    ValidFrom     time.Time
    ValidTo       *time.Time
    SupersededBy  *string             // ID of replacement card, if superseded
    Tags          []string
    HeatScore     float64             // recency/frequency signal (0-1)
    LastActivated *time.Time
    CandidateScore float64            // ranking score from search
}
```

### EvidenceFragment

Bounded excerpt from a card:

```go
type EvidenceFragment struct {
    CardID       string
    MaterialRef  string              // reference to source material
    SourceURI    string              // file path or URL
    SourceRev    string              // version or timestamp
    Excerpt      string              // bounded excerpt
    StartOffset  int                 // character offset in full content
    EndOffset    int                 // character offset (exclusive)
    ContentHash  string              // SHA256 of excerpt
    Validity     string              // "valid", "suspect", "stale"
    EvidenceRefs []string            // citations or cross-refs
}
```

---

## Validation & Enforcement

### No Full Content in Cards

SearchCards never returns the full Content field. Cards are metadata only.

### Character Budgets

ReadEvidence enforces budgets:

```go
// Per-item budget: truncate any excerpt exceeding budgetPerItem
if len(excerpt) > budgetPerItem {
    excerpt = excerpt[:budgetPerItem]
}

// Total budget: stop adding fragments when sum >= budgetTotal
totalChars += len(excerpt)
if totalChars >= budgetTotal {
    break
}
```

### Scope Enforcement

Only return cards matching the requested scope:

```go
if card.Scope != scope && card.Scope != "global" {
    continue  // Skip out-of-scope cards
}
```

### Lifecycle Filters

Skip cards with invalid status:

```go
if card.Status == "deleted" || card.Status == "superseded" || 
   (card.ValidTo != nil && time.Now().After(*card.ValidTo)) {
    continue
}
```

---

## Build & Test

### Build

```bash
cd mentle
go build ./facade/...
```

### Test

```bash
cd mentle
GOSUMDB=off go test ./facade/...
```

### Test Specific

```bash
GOSUMDB=off go test -v ./facade/ -run TestSearchCards
GOSUMDB=off go test -v ./facade/ -run TestReadEvidence
```

---

## Testing Requirements

Before starting feature work:

```bash
cd mentle && GOSUMDB=off go test ./facade/...
```

**Mandatory behavioral tests:**

- SearchCards does not return full Content field
- ReadEvidence enforces per-item budget (no excerpt > budgetPerItem)
- ReadEvidence enforces total budget (sum(excerpts) ≤ budgetTotal)
- Superseded, deleted, and expired cards are never returned
- Out-of-scope cards are never returned
- Invalid memory IDs return error

---

## Integration with Garden

Garden's Mentle adapter uses the facade:

```go
// In garden/internal/router/mentle_adapter.go
cards, err := mentle.SearchCards(ctx, intent, scope, 10)
evidence, err := mentle.ReadEvidence(ctx, cardID, 200, 2000)
```

---

## Conventions

- **Go formatting:** standard `gofmt`
- **Error handling:** explicit, wrapped with context
- **Character counting:** Unicode code-points, not bytes
- **Timestamps:** UTC, ISO 8601 format
- **Status values:** "active", "superseded", "deleted", "expired" (enum-like)

---

## MANUAL

When updating:

1. Keep facade stable; it's the public API
2. Do not expose internal storage details
3. Enforce budgets and validity checks rigorously
4. Update when:
   - New card status is added
   - Budget calculations change
   - Scope rules evolve
5. Do not update for:
   - Internal search algorithm details (use internal/AGENTS.md)

Parent reference: ../AGENTS.md
