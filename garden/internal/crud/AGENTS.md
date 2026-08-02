<!-- Parent: ../AGENTS.md -->

# garden/internal/crud — Legacy CRUD Translator

**Generated:** 2026-08-01  
**Purpose:** HTTP v1 compatibility layer translating legacy CRUD operations to v2 backends

---

## Purpose

The `crud/` package provides backward compatibility for v1 HTTP clients by translating CRUD operations into v2 routing:

- **POST /v1/memories** — write to section: or memory: backend
- **GET /v1/memories/{key}** — read from appropriate backend
- **GET /v1/memories?prefix=&limit=** — list with prefix filtering
- **DELETE /v1/memories/{key}** — remove memory

No new features; v1 is frozen and deprecated.

---

## Structure

```
crud/
├── crud.go         # CRUD handler implementation
└── crud_test.go    # Unit tests
```

---

## Key Responsibilities

### Write Handler

Accepts POST body with `key` and `value` fields:
- Parses key prefix to determine backend (section: vs memory:)
- Adds deprecation metadata to response
- Routes to appropriate Router backend

### Read Handler

Retrieves by key, delegates to Router:
- Returns full object if exists
- 404 if not found
- No content filtering (use v2 for policy-aware reads)

### List Handler

Accepts `prefix` and `limit` query parameters:
- Default prefix: `section:` if omitted
- Returns array of objects
- Limit capped at 100 for safety

### Delete Handler

Removes by key:
- 204 No Content on success
- 404 if not found
- Idempotent (repeated deletes are safe)

---

## Testing

```bash
cd garden
GOSUMDB=off go test -v ./internal/crud/...
```

**Behavioral tests:**

- Write with section: prefix routes to Laputa
- Write with memory: prefix routes to Mentle
- Read returns exact object stored
- List respects limit and prefix
- Delete is idempotent
- All v1 responses include deprecation header

---

## Conventions

- All v1 routes emit `Deprecation: true` in response headers
- No new v1 routes are added (feature requests go to v2)
- Error responses use standard HTTP status codes
- No content transformation or filtering

---

## MANUAL

Do not expand this package. v1 is deprecated; all new work goes to v2.

Parent reference: ../AGENTS.md
