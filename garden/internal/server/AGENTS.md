<!-- Parent: ../AGENTS.md -->

# garden/internal/server — HTTP Server & Request Handling

**Generated:** 2026-08-01  
**Purpose:** HTTP server setup, middleware, and request/response handling

---

## Purpose

The `server/` package sets up the HTTP server and handles request lifecycle:

- **Server initialization** — configure port, TLS, timeouts
- **Middleware** — logging, tracing, error recovery
- **Request parsing** — unmarshal JSON bodies
- **Response formatting** — consistent JSON responses
- **Graceful shutdown** — drain connections, cleanup resources

---

## Structure

```
server/
├── server.go        # HTTP server implementation
├── handlers.go      # Route handlers
└── server_test.go
```

---

## Key Endpoints

### v2

```
POST   /v2/memories                # Canonical memory create
GET    /v2/memories/{id}           # Canonical memory read
GET    /v2/memories                # Canonical memory list
PATCH  /v2/memories/{id}           # Canonical memory update
DELETE /v2/memories/{id}           # Canonical memory delete
POST   /v2/ingest/sessions         # Session-end ingestion
GET    /v2/ingestions/{id}         # Ingestion status
POST   /v2/recall/bootstrap        # Session bootstrap context
GET    /health                     # Health check
```

```
POST   /v2/recall/fast            # Fast recall
POST   /v2/recall/deep            # Deep recall
GET    /v2/recall/traces/{id}     # Retrieve trace
POST   /v2/activity/events        # Ingest event
GET    /v2/activity/sessions/{id} # Session history
POST   /v2/governance/projection  # Read governance
POST   /v2/governance/mutations   # Governed mutation (audited)
GET    /v2/governance/audit       # Audit trail
POST   /v2/evolution/runs         # Evolution run
GET    /v2/evolution/runs         # List runs (newest first, read-only)
GET    /v2/evolution/proposals    # List proposals (newest first, read-only)
GET    /v2/evolution/hub/status   # EvoMap provider liveness (ADR-0010)
GET    /v2/mailbox/inbox          # EvoMap inbox
GET    /v2/mailbox/outbox         # EvoMap outbox
GET    /v2/admin/overview         # Admin overview
GET    /v2/materials/cards        # Card discovery
GET    /v2/cognitive/world        # WORLD projection
```

---

## Configuration

Environment variables:
- `GARDEN_PORT` — HTTP port (default: 7373)
- `GARDEN_HOST` — bind address (default: 127.0.0.1)
- `GARDEN_TIMEOUT` — request timeout (default: 30s)
- `GARDEN_LOG_LEVEL` — log level (default: info)

---

## Testing

```bash
cd garden
GOSUMDB=off go test -v ./internal/server/...
```

**Behavioral tests:**

- Server starts and listens on configured port
- Request timeout is enforced
- Graceful shutdown closes connections
- Health check returns 200 OK
- All routes return appropriate status codes

---

## Conventions

- All responses are JSON
- Errors include error_code and message
- Trace IDs are included in response headers
- Logging is structured

---

## MANUAL

Keep server focused on HTTP mechanics. Business logic goes to the service packages.

Parent reference: ../AGENTS.md
