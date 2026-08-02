<!-- Parent: ../../AGENTS.md -->

# mentle/storage/sqlite — SQLite Metadata and Auxiliary Storage

**Generated:** 2026-08-01  
**Purpose:** SQLite backend for metadata, temporal constraints, and auxiliary data

---

## Purpose

The `sqlite/` package provides durable storage for:

- **Memory metadata** — card info, validity periods, collection membership
- **Knowledge graph** — temporal RDF-style triples
- **Indexes** — supporting efficient queries
- **WAL recovery** — replay committed entries

---

## Structure

```
sqlite/
├── store.go                         # SQLite connection and operations
├── store_test.go                    # Test suite
├── schema.go                        # Database schema
├── queries.go                       # Standard SQL queries
└── (supporting utilities)
```

---

## Schema

Core tables:

- **cards** — memory metadata (ID, kind, status, validity)
- **triples** — knowledge graph (subject, predicate, object, valid_from, valid_to)
- **collections** — taxonomy (wing, room, drawer hierarchy)
- **wal_log** — write-ahead log entries

---

## Key Operations

```go
store, err := sqlite.Open("./memory.db")

// Store card metadata
err := store.StoreCard(ctx, card)

// Query by temporal range
cards, err := store.CardsByTimeRange(ctx, start, end)

// Add RDF triple
err := store.AddTriple(ctx, subject, predicate, object, validFrom, validTo)
```

---

## Build & Test

```bash
cd mentle
GOSUMDB=off go test ./storage/sqlite/...
```

---

## MANUAL

Schema is stable. Breaking changes require migration. Document migrations in schema.go.

Parent reference: ../AGENTS.md
