<!-- Parent: ../../AGENTS.md -->

# mentle/internal/kg — Knowledge Graph Operations

**Generated:** 2026-08-01  
**Purpose:** Temporal RDF-based knowledge graph for entity relationships

---

## Purpose

The `kg/` package manages knowledge graph operations:

- **Temporal RDF** — resource-description-framework with time dimension
- **Entity relationships** — link entities with typed edges
- **Temporal reasoning** — "when did X lead to Y?"
- **Graph traversal** — find related entities and insights
- **Query support** — SPARQL-like querying

---

## Structure

```
kg/
├── knowledge_graph.go  # KG engine
├── kg_test.go
└── (supporting utilities)
```

---

## Key Concepts

### Temporal Triple

```go
type TemporalTriple struct {
    Subject     string    // entity
    Predicate   string    // relationship type (caused, influenced, related_to)
    Object      string    // entity
    StartTime   time.Time // when relationship began
    EndTime     *time.Time // when relationship ended
    Confidence  float64   // 0.0-1.0
}
```

### Query Examples

**Find decisions caused by KAI:**
```
?decision caused_by KAI
```

**Find entities influenced by decision D1 after 2026-01-01:**
```
?entity influenced_by D1 AFTER 2026-01-01
```

### Operations

**AddTriple(ctx, triple):**
- Add or update triple
- Track temporal validity

**QueryPattern(ctx, pattern):**
- Pattern matching
- Return matching triples

**TraversePath(ctx, start, depth):**
- BFS/DFS graph traversal
- Return related entities

---

## Testing

```bash
cd mentle
GOSUMDB=off go test -v ./internal/kg/...
```

**Behavioral tests:**

- Triples stored correctly
- Pattern queries return matches
- Temporal queries respect time boundaries
- Graph traversal finds paths

---

## Conventions

- Subject and object are entity IDs
- Predicate is lowercase with underscores
- Confidence 0.0-1.0
- Time dimension is UTC

---

## MANUAL

Keep KG focused on graph structure. Inference and reasoning go elsewhere.

Parent reference: ../AGENTS.md
