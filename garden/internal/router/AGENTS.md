<!-- Parent: ../AGENTS.md -->

# garden/internal/router — HTTP Request Routing & Backend Dispatch

**Generated:** 2026-08-01  
**Purpose:** Unified HTTP router dispatching requests to Laputa (governance) or Mentle (memory) backends

---

## Purpose

The `router/` package implements the core routing logic that dispatches all requests:

- **Prefix-based routing** — section: to Laputa, memory: to Mentle
- **Governance projection** — read-only access to authority policies
- **Mentle adapter** — facade to memory operations with policy filtering
- **Request parsing and validation** — normalize v1/v2 payloads
- **Error handling** — structured responses with degradation flags

---

## Structure

```
router/
├── router.go              # Main router implementation
├── router_test.go         # Router tests
├── governance.go          # Governance adapter (read-only Laputa)
├── mentle_adapter.go      # Mentle backend facade
└── mentle_adapter_test.go
```

---

## Key Components

### Router

Central dispatcher:
- Inspects key prefix to determine backend
- Calls appropriate handler (Laputa or Mentle)
- Returns response with optional degradation flag

### GovernanceAdapter

Read-only projection of Laputa state:
- GetGovernanceProjection(ctx, scope) — current authority, policy, wings, rooms
- CheckAuthority(ctx, scope, action, key) — verify write permission
- GetDeniedSources(ctx, scope) — list of excluded sources for RAG
- No mutations; reads only

### MentleAdapter

Facade to Mentle operations:
- SearchCards(ctx, scope, query) — find candidate cards
- ReadEvidence(ctx, cardID, budget) — fetch excerpts with character limits
- StoreActivity(ctx, session, event) — record activity
- Policy filtering applied at search time

### Request Router

Routes by prefix and HTTP method:
- Prefix: `section:` → Laputa
- Prefix: `memory:` → Mentle
- No prefix → default to Laputa (legacy)

---

## Testing

```bash
cd garden
GOSUMDB=off go test -v ./internal/router/...
```

**Behavioral tests:**

- section: prefix routes to Laputa
- memory: prefix routes to Mentle
- Governance projection reads are accurate
- Mentle adapter respects policy filters
- Authority checks pass/fail correctly
- Degradation flag set when backends unavailable

---

## Conventions

- Router never mutates governance (read-only)
- All backend operations are request-scoped
- Errors are wrapped with context
- Timeout applied to all backend calls

---

## MANUAL

Keep router focused on dispatch logic. Backend-specific details go to governance.go or mentle_adapter.go.

Parent reference: ../AGENTS.md
