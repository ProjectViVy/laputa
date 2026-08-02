<!-- Parent: ../AGENTS.md -->

# mentle/taxonomy — Palace Structure (Wings, Rooms, Drawers)

**Generated:** 2026-08-01  
**Purpose:** Memory organization hierarchy and categorical indexing

---

## Purpose

The `taxonomy/` package implements the Palace structure for organizing memories hierarchically:

- **Wings** — top-level categories (e.g., "architecture", "debugging", "decisions", "temporal-reasoning")
- **Rooms** — subcategories or entities within wings (e.g., "auth-system" room in "architecture" wing)
- **Drawers** — individual memory items (cards, fragments, evidence)

The taxonomy provides categorical organization, navigation, and scoped retrieval.

---

## Structure

```
taxonomy/
├── palace.go                      # Palace and wing management
├── palace_test.go
├── room.go                        # Room and categorical discovery
├── room_test.go
├── drawer.go                      # Individual item storage and retrieval
├── drawer_test.go
├── hierarchy.go                   # Path traversal and navigation
├── hierarchy_test.go
└── types.go                       # Common types and interfaces
```

### Subdirectories (Depth 3)

No formal depth-3 AGENTS.md required; implementations are self-contained.

---

## Hierarchy

### Palace

Root container for all memories. Metadata:

- `name` — palace name
- `owner` — agent identifier
- `created_at` — timestamp
- `wings` — map of wing name → wing data

### Wing

Top-level category. Examples:

- `architecture` — system design decisions, patterns, diagrams
- `debugging` — bugs, debugging sessions, root causes
- `decisions` — strategic and tactical decisions with rationale
- `temporal-reasoning` — time-based facts and timelines
- `relationships` — agent relationships and collaborations
- `learning` — techniques, approaches, what worked/didn't

Each wing contains rooms and metadata:

- `name` — wing name
- `description` — purpose and scope
- `created_at` — timestamp
- `rooms` — map of room name → room data

### Room

Subcategory or entity within a wing. Examples:

- In `architecture` wing: `auth-system`, `database`, `api-gateway`, `frontend`
- In `debugging` wing: `memory-leak-2024`, `race-condition-auth`, `performance-issue`

Each room contains:

- `name` — room name
- `entity_id` — optional identifier for entity or topic
- `description` — room purpose
- `drawers` — list of memory items

### Drawer

Individual memory item with content and metadata:

- `id` — UUID
- `title` — concise label
- `content` — full text
- `kind` — "document", "decision", "snippet", "diary", etc.
- `tags` — optional labels
- `created_at` — timestamp
- `updated_at` — last modification
- `heat_score` — recency/frequency signal

---

## Navigation

### Paths

Memory items are addressed by path:

```
palace:/wing/room/drawer_id

palace:/architecture/auth-system/card-2024-08-01-auth-flow
palace:/debugging/race-condition-auth/snippet-123
palace:/decisions/central-cache/decision-xyz
```

### Traversal

```go
// Get all items in a wing
items, err := palace.GetWing(ctx, "architecture")

// Get all items in a room
items, err := palace.GetRoom(ctx, "architecture", "auth-system")

// Get a specific drawer
item, err := palace.GetDrawer(ctx, "architecture", "auth-system", "card-2024-08-01")
```

### Discovery

```go
// List all wings
wings, err := palace.ListWings(ctx)

// List all rooms in a wing
rooms, err := palace.ListRooms(ctx, "architecture")

// Search across taxonomy
results, err := palace.SearchTaxonomy(ctx, "query")
```

---

## Build & Test

### Build

```bash
cd mentle
go build ./taxonomy/...
```

### Test

```bash
cd mentle
GOSUMDB=off go test ./taxonomy/...
```

### Test Specific

```bash
GOSUMDB=off go test -v ./taxonomy/ -run TestPalaceStructure
GOSUMDB=off go test -v ./taxonomy/ -run TestPathTraversal
```

---

## Testing Requirements

Before starting feature work:

```bash
cd mentle && GOSUMDB=off go test ./taxonomy/...
```

**Mandatory behavioral tests:**

- Wings are created and listed correctly
- Rooms are scoped to wings (no cross-wing confusion)
- Drawers are stored and retrieved by full path
- Path navigation is deterministic
- Wing/room names don't conflict with special characters
- Concurrent writes don't corrupt hierarchy

---

## Conventions

- **Go formatting:** standard `gofmt`
- **Naming:** wings and rooms use lowercase with hyphens (e.g., "auth-system")
- **Paths:** always start with `palace:/` prefix
- **IDs:** UUID format for drawers
- **Timestamps:** UTC, ISO 8601 format

---

## Integration

Taxonomy is used by:

- **Facade** — scoped searches by wing/room
- **Mining** — auto-categorize extracted memories into wings
- **Search** — filter results by categorical scope
- **Activity** — route events to appropriate rooms

---

## MANUAL

When updating:

1. Keep hierarchy stable; wing names should not change frequently
2. Support adding new wings without breaking existing code
3. Path syntax is immutable (don't change `palace:/` prefix)
4. Update when:
   - New wing types are introduced
   - Room discovery rules evolve
   - Navigation patterns are added
5. Do not update for:
   - Individual drawer operations (use facade instead)

Parent reference: ../AGENTS.md
