<!-- Parent: ../../AGENTS.md -->

# laputa/governance/store — File-Based Governance Store

**Generated:** 2026-08-01  
**Purpose:** Atomic, cross-process-safe JSON store for governance sections

---

## Purpose

The `store/` package provides durable governance state:

- **JSON file persistence** — sections stored as JSON files
- **Cross-process locking** — flock (gofrs/flock) for safety
- **Atomic writes** — write-to-temp, fsync, rename pattern
- **Metadata tracking** — version numbers, last-updated timestamps
- **Orphan lock recovery** — cleanup stale locks from crashed processes

---

## Structure

```
store/
├── store.go         # File store implementation
└── store_test.go
```

---

## Key Concepts

### File Store

```go
type FileStore struct {
    Root      string        // ~/.laputa directory
    RWMutex   sync.RWMutex  // in-process coordination
    // locks managed per-section
}
```

### Operations

**GetSection(ctx, name):**
- Read JSON file for section
- Return as map[string]any
- Handles missing sections gracefully

**UpdateSection(ctx, name, updates):**
- Acquire write lock (flock)
- Read current state
- Merge updates
- Write to temp file
- Fsync to disk
- Rename to final location
- Release lock

**ListSections():**
- Return all section names
- No locking (read-only enumeration)

---

## Locking Strategy

**Write Lock (exclusive):**
- Acquired before any mutation
- Held until fsync complete
- Cross-process safe via flock
- Orphan cleanup: checks PID, clears stale locks

**Read Lock (shared):**
- Advisory only; used for in-process coordination
- Multiple readers concurrent
- Writers block readers

---

## Testing

```bash
cd laputa
GOSUMDB=off go test -v ./governance/store/...
```

**Behavioral tests:**

- Concurrent writes don't corrupt data
- Cross-process locking prevents lost updates
- Orphan lock cleanup works
- Metadata (_meta.version) updated correctly
- Sections survive server restarts

---

## Conventions

- All JSON is 2-space indented
- Metadata is hidden in _meta key
- Filenames are lowercase section names
- All times are UTC

---

## MANUAL

Keep store focused on persistence mechanics. Business logic goes to engine.go.

Parent reference: ../AGENTS.md
