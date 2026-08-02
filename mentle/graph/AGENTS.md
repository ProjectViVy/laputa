<!-- Parent: ../AGENTS.md -->

# mentle/graph — Knowledge Graph (Temporal RDF)

**Generated:** 2026-08-01  
**Purpose:** Entity relationships, temporal reasoning, and knowledge graph queries

---

## Purpose

The `graph/` package implements a temporal RDF-style knowledge graph backed by SQLite:

- **Entity relationships** modeled as subject → predicate → object triples
- **Temporal validity** for each triple (valid_from, valid_to timestamps)
- **Confidence scores** to track assertion strength
- **Fact invalidation** when new information supersedes old facts
- **Temporal reasoning** and relationship traversal

The knowledge graph powers Deep Recall's entity-focused retrieval and enables reasoning about how relationships change over time.

---

## Structure

```
graph/
├── triple.go                      # Triple representation and interface
├── triple_test.go
├── store.go                       # SQLite-backed triple store
├── store_test.go
├── query.go                       # Query builder and execution
├── query_test.go
├── temporal.go                    # Temporal validity and reasoning
├── temporal_test.go
└── types.go                       # Common types and enums
```

### Subdirectories (Depth 3)

No formal depth-3 AGENTS.md required; implementations are self-contained.

---

## Key Concepts

### Triple

A fact with temporal validity:

```go
type Triple struct {
    ID         string     // UUID
    Subject    string     // entity or identifier
    Predicate  string     // relationship type
    Object     string     // target entity or value
    ValidFrom  time.Time  // when this fact became true
    ValidTo    *time.Time // when this fact ceased to be true (nil = ongoing)
    Confidence float64    // 0.0 to 1.0
    Source     string     // where this fact came from (card ID, etc.)
    Tags       []string   // optional categorization
}
```

### Relationships

Common predicate examples:

- `mentions` — entity A mentions entity B
- `depends_on` — module A depends on module B
- `authored_by` — artifact authored by person
- `related_to` — semantic relationship
- `supersedes` — new fact supersedes old fact
- `contradicts` — facts that conflict

### Temporal Validity

Each triple has a lifespan:

```go
// Fact is valid if: now >= ValidFrom AND (ValidTo is nil OR now < ValidTo)
func (t *Triple) IsValid(at time.Time) bool {
    if at.Before(t.ValidFrom) {
        return false
    }
    if t.ValidTo != nil && !at.Before(*t.ValidTo) {
        return false
    }
    return true
}
```

### Queries

Temporal query examples:

```go
// Find all relationships of person Alice
triples, err := store.Query(ctx, &QueryTriple{
    Subject: "alice",
    As:       time.Now(),
})

// Find all subjects related to debugging
triples, err := store.Query(ctx, &QueryTriple{
    Object: "debugging",
    ValidAt: time.Now(),
})

// Historical: who did Bob work with in June?
triples, err := store.Query(ctx, &QueryTriple{
    Subject: "bob",
    Predicate: "works_with",
    ValidAt: time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC),
})
```

---

## SQLite Schema

Triples are stored in a single table with temporal indexes:

```sql
CREATE TABLE triples (
    id TEXT PRIMARY KEY,
    subject TEXT NOT NULL,
    predicate TEXT NOT NULL,
    object TEXT NOT NULL,
    valid_from DATETIME NOT NULL,
    valid_to DATETIME,
    confidence REAL,
    source TEXT,
    tags TEXT,  -- JSON array
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_subject_valid ON triples(subject, valid_to);
CREATE INDEX idx_object_valid ON triples(object, valid_to);
CREATE INDEX idx_predicate_valid ON triples(predicate, valid_to);
```

---

## Build & Test

### Build

```bash
cd mentle
go build ./graph/...
```

### Test

```bash
cd mentle
GOSUMDB=off go test ./graph/...
```

### Test Specific

```bash
GOSUMDB=off go test -v ./graph/ -run TestTemporalQuery
GOSUMDB=off go test -v ./graph/ -run TestTripleStore
```

---

## Usage Example

### Add a Fact

```go
store := graph.NewStore(db)
triple := &graph.Triple{
    ID:        uuid.New().String(),
    Subject:   "module-auth",
    Predicate: "depends_on",
    Object:    "module-crypto",
    ValidFrom: time.Now(),
    ValidTo:   nil,  // ongoing
    Confidence: 0.95,
    Source:    "card-2024-08-01",
}
err := store.AddTriple(ctx, triple)
```

### Query Relationships

```go
// Find all dependencies of module-auth at current time
results, err := store.Query(ctx, &graph.QueryTriple{
    Subject:   "module-auth",
    Predicate: "depends_on",
    ValidAt:   time.Now(),
})
```

### Invalidate a Fact

```go
// Mark triple as no longer valid (fact changed)
now := time.Now()
err := store.Invalidate(ctx, tripleID, now)
```

---

## Testing Requirements

Before starting feature work:

```bash
cd mentle && GOSUMDB=off go test ./graph/...
```

**Mandatory behavioral tests:**

- Triples are stored and retrieved with full temporal data
- Temporal queries respect ValidFrom and ValidTo
- Invalidated facts are not returned by queries
- Confidence scores are preserved
- Concurrent writes do not corrupt the store
- Indexes provide consistent query performance

---

## Performance Targets

| Operation | Target |
|---|---:|
| Add triple | P95 ≤ 2 ms |
| Query by subject | P95 ≤ 30 ms |
| Temporal query | P95 ≤ 30 ms |
| Invalidate fact | P95 ≤ 5 ms |

---

## Conventions

- **Go formatting:** standard `gofmt`
- **Timestamps:** UTC, ISO 8601 format
- **UUIDs:** `google/uuid` for triple IDs
- **Confidence:** 0.0 to 1.0 float64
- **JSON tags:** marshaling for storage and API responses

---

## MANUAL

When updating:

1. Keep schema stable; new predicates don't require schema changes
2. Temporal validity is immutable once stored
3. Use Invalidate() to retire facts, don't delete
4. Update when:
   - New query types are added
   - Temporal reasoning rules change
   - Performance targets are adjusted
5. Do not update for:
   - Individual fact entries (use StoreActivity instead)

Parent reference: ../AGENTS.md
