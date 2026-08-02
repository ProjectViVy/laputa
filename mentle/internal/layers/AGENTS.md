<!-- Parent: ../../AGENTS.md -->

# mentle/internal/layers — Multi-Layer Memory Organization

**Generated:** 2026-08-01  
**Purpose:** Organize memories across semantic, temporal, and social layers

---

## Purpose

The `layers/` package organizes memories into layers:

- **Semantic layer** — topics, concepts, relationships
- **Temporal layer** — timeline, causality, evolution
- **Social layer** — agents, relationships, resonance
- **Technical layer** — architecture, patterns, decisions
- **Cross-layer linking** — relationships across layers

---

## Structure

```
layers/
├── layers.go         # Multi-layer coordinator
└── layers_test.go
```

---

## Key Concepts

### Layer Definition

```go
const (
    LayerSemantic   = "semantic"
    LayerTemporal   = "temporal"
    LayerSocial     = "social"
    LayerTechnical  = "technical"
)
```

### Layer Operations

**Index(ctx, card, layer):**
- Add card to layer index
- Create relationships to other cards in layer

**Query(ctx, layer, criterion):**
- Search within single layer
- Return cards and relationships

**Bridge(ctx, card, layer1, layer2):**
- Link card across layers
- Record relationships between layers

---

## Testing

```bash
cd mentle
GOSUMDB=off go test -v ./internal/layers/...
```

**Behavioral tests:**

- Cards indexed to correct layers
- Cross-layer links created
- Queries return appropriate results
- No circular relationships

---

## Conventions

- Each card can appear in multiple layers
- Relationships are typed (semantic, temporal, social, technical)
- No mutation of past relationships

---

## MANUAL

Keep layers focused on organization. Inference goes to kg or graph packages.

Parent reference: ../AGENTS.md
