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

### v1 (Legacy Compatibility)

```
POST   /v1/memories               # CRUD write
GET    /v1/memories/{key}         # CRUD read
GET    /v1/memories?prefix=       # CRUD list
DELETE /v1/memories/{key}         # CRUD delete
POST   /v1/context/resolve        # Recall (adapter)
GET    /health                    # Health check
```

### v2 (vNext)

```
POST   /v2/recall/fast            # Fast recall
POST   /v2/recall/deep            # Deep recall
GET    /v2/recall/traces/{id}     # Retrieve trace
POST   /v2/activity/events        # Ingest event
GET    /v2/activity/sessions/{id} # Session history
POST   /v2/governance/projection  # Read governance
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

Keep server focused on HTTP mechanics. Business logic goes to router and other packages.

Parent reference: ../AGENTS.md
