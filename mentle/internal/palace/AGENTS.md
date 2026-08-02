<!-- Parent: ../../AGENTS.md -->

# mentle/internal/palace — Palace Structure & Navigation

**Generated:** 2026-08-01  
**Purpose:** Directory hierarchy (wings, rooms, drawers) for memory organization

---

## Purpose

The `palace/` package manages the memory palace structure:

- **Wings** — major categories (project, agent, domain, etc.)
- **Rooms** — subcategories within wings
- **Drawers** — individual memory items
- **Navigation** — traverse hierarchy efficiently
- **Path resolution** — locate items by path (wing/room/drawer)

---

## Structure

```
palace/
├── palace.go         # Palace structure coordinator
├── palace_test.go
├── graph.go          # Palace graph representation
├── graph_test.go
├── wing.go           # Wing operations
├── drawer.go         # Drawer storage and retrieval
└── (supporting utilities)
```

---

## Key Concepts

### Palace Structure

```
Palace
├── Wing (project)
│   ├── Room (authentication)
│   │   ├── Drawer (token-validation-pattern)
│   │   ├── Drawer (oauth2-flow)
│   │   └── ...
│   ├── Room (database)
│   └── ...
├── Wing (agent-self)
│   ├── Room (identity)
│   └── ...
└── ...
```

### Path Format

`wing/room/drawer` — hierarchical addressing

Example: `project/auth/token-validation-pattern`

### Operations

**CreateWing(ctx, wing):**
- Register new wing
- Initialize metadata

**CreateRoom(ctx, wing, room):**
- Create room in wing
- Link to wing

**StoreDrawer(ctx, wing, room, drawer):**
- Store memory item
- Index by path
- Update timestamps

**RetrieveDrawer(ctx, wing, room, drawer):**
- Fetch item by path
- Return with metadata

**NavigateWing(ctx, wing):**
- List rooms in wing

**NavigateRoom(ctx, wing, room):**
- List drawers in room

---

## Testing

```bash
cd mentle
GOSUMDB=off go test -v ./internal/palace/...
```

**Behavioral tests:**

- Wings and rooms create correctly
- Drawers store and retrieve
- Navigation traverses hierarchy
- Path resolution is accurate
- Dedup prevents duplicate items

---

## Conventions

- Wing and room names are lowercase
- Drawer IDs are UUIDs or deterministic hashes
- All paths are immutable once set
- No circular references

---

## MANUAL

Keep palace focused on hierarchy. Item deduplication goes to facade.

Parent reference: ../AGENTS.md
