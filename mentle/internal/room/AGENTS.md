<!-- Parent: ../../AGENTS.md -->

# mentle/internal/room — Room Detection & Categorization

**Generated:** 2026-08-01  
**Purpose:** Detect and categorize memories into semantic rooms

---

## Purpose

The `room/` package detects memory categories (rooms):

- **Room detection** — identify category from content
- **Room hierarchy** — organize rooms within wings
- **Auto-categorization** — assign new memories to rooms
- **Room metadata** — track room characteristics
- **Cross-room linking** — identify related rooms

---

## Structure

```
room/
├── detector.go       # Room detection service
└── detector_test.go
```

---

## Key Concepts

### Room Types

Common rooms by wing:
- **project wing:** authentication, database, api, ui, deployment
- **agent wing:** identity, learning, constraints, preferences
- **domain wing:** (domain-specific rooms)

### Detection Process

1. **Keyword analysis** — match content keywords to room signatures
2. **Entity recognition** — identify technical entities
3. **Decision type** — is this architectural, tactical, or operational?
4. **Confidence scoring** — how confident in categorization?

### Room Metadata

```go
type RoomMetadata struct {
    Name        string
    Wing        string
    Description string
    Keywords    []string
    Confidence  float64
}
```

---

## Testing

```bash
cd mentle
GOSUMDB=off go test -v ./internal/room/...
```

**Behavioral tests:**

- Room detection matches keywords
- Confidence scores reasonable
- Fallback room used if ambiguous
- No room assignment errors

---

## Conventions

- Room names lowercase
- Keywords lowercase
- Confidence 0.0-1.0

---

## MANUAL

Keep room focused on detection. Storage and indexing go elsewhere.

Parent reference: ../AGENTS.md
