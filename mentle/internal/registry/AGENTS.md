<!-- Parent: ../../AGENTS.md -->

# mentle/internal/registry — Memory Registry & Indexing

**Generated:** 2026-08-01  
**Purpose:** Central registry for all memory items with fast lookups

---

## Purpose

The `registry/` package maintains a searchable registry:

- **Unique ID tracking** — UUID or deterministic hashing
- **Fast lookup** — O(1) by ID
- **Index building** — support multiple index strategies
- **Metadata tracking** — revision, timestamp, status
- **Lifecycle tracking** — creation, updates, deletion

---

## Structure

```
registry/
├── registry.go       # Registry service
└── registry_test.go
```

---

## Key Concepts

### Registry Entry

```go
type RegistryEntry struct {
    ID          string
    CardID      string
    Title       string
    Status      string    // "active", "superseded", "deleted"
    Wing        string
    Room        string
    CreatedAt   time.Time
    UpdatedAt   time.Time
    SupersededBy *string
}
```

### Operations

**Register(ctx, card):**
- Assign or validate ID
- Store metadata
- Index for search

**Get(ctx, id):**
- Lookup by ID
- Return metadata

**Update(ctx, id, updates):**
- Update card metadata
- Track version
- Update timestamp

**List(ctx, wing, room):**
- List cards in location
- Respect status filters

**Supersede(ctx, oldID, newID):**
- Mark old card superseded
- Link to replacement

---

## Testing

```bash
cd mentle
GOSUMDB=off go test -v ./internal/registry/...
```

**Behavioral tests:**

- Unique IDs assigned
- Fast O(1) lookup
- Supersede creates chain
- Status filters work

---

## Conventions

- IDs immutable once assigned
- All timestamps UTC
- No deletion, only mark superseded
- Multiple indexes maintained

---

## MANUAL

Keep registry focused on metadata. Content storage goes to storage layer.

Parent reference: ../AGENTS.md
