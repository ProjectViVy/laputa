<!-- Parent: ../../AGENTS.md -->

# mentle/pkg/wal — Write-Ahead Log Implementation

**Generated:** 2026-08-01  
**Purpose:** Durable write-ahead logging for all memory operations

---

## Purpose

The `wal/` package implements write-ahead logging:

- **Durability guarantee** — mutations are logged before applied
- **Recovery** — replay committed entries on startup
- **Crash safety** — no data loss on unexpected shutdown
- **Concurrency** — thread-safe entry logging

See parent [mentle/pkg/AGENTS.md](../AGENTS.md) for usage and data format.

---

## Structure

```
wal/
├── wal.go                           # Write-ahead log implementation
├── wal_test.go
├── recovery.go                      # WAL recovery logic
└── (supporting utilities)
```

---

## Key Operations

```go
// Open or create WAL
log, err := wal.Open("./wal")

// Write entry (pending state)
id, err := log.Write(ctx, entry)

// Commit entry
err := log.Commit(ctx, id)

// Recover committed entries
entries, err := log.RecoverCommitted(ctx)

// Clear in-flight entries
err := log.ClearInFlight()
```

---

## Entry Status

- **pending** — written but not committed
- **committed** — safe to replay
- **rolled_back** — should be discarded

---

## Recovery Procedure

On startup:

1. Open WAL directory
2. List all entries
3. Recover committed entries
4. Apply to palace/KG
5. Clear in-flight entries
6. Resume normal operation

---

## Build & Test

```bash
cd mentle
GOSUMDB=off go test ./pkg/wal/...
```

**Mandatory tests:**
- WAL entries are recoverable after crash
- Concurrent writes don't corrupt WAL
- Recovery produces bit-identical state

---

## MANUAL

WAL format is internal. Changes require migration logic. Document format changes in recovery.go.

Parent reference: ../AGENTS.md
