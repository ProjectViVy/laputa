<!-- Parent: ../AGENTS.md -->

# garden/internal/supervision — Process Supervision & Logging

**Generated:** 2026-08-01  
**Purpose:** Graceful shutdown, process health, and structured logging

---

## Purpose

The `supervision/` package manages process lifecycle and observability:

- **Graceful shutdown** — drain requests, close connections, cleanup
- **Health monitoring** — background health checks on backends
- **Structured logging** — request correlation, trace IDs, metrics
- **Signal handling** — respond to SIGTERM, SIGINT
- **Resource cleanup** — close files, release locks, cleanup temp data

---

## Structure

```
supervision/
├── supervision.go      # Supervision service
└── supervision_test.go
```

---

## Key Concepts

### Supervisor

Manages process lifecycle:
```go
type Supervisor struct {
    Server    *http.Server
    Router    *router.Router
    Lifecycle *lifecycle.Lifecycle
}

func (s *Supervisor) Start(ctx context.Context) error
func (s *Supervisor) Shutdown(ctx context.Context) error
func (s *Supervisor) Health() HealthStatus
```

### Health Check

Periodic monitoring:
- Laputa connectivity (read-only health check)
- Mentle connectivity (search card count)
- Disk space (log directory)
- Memory usage
- Request latency percentiles

### Graceful Shutdown

On SIGTERM:
1. Stop accepting new requests
2. Drain in-flight requests (timeout: 30s)
3. Close database connections
4. Close file handles
5. Exit with status 0

### Logging

Structured logging with request context:
- Trace ID (correlation)
- Request method and path
- Response status and latency
- Error stack traces (if any)
- Degradation warnings

---

## Testing

```bash
cd garden
GOSUMDB=off go test -v ./internal/supervision/...
```

**Behavioral tests:**

- Health check reports accurate backend status
- Graceful shutdown drains connections
- Logging includes trace IDs
- Signal handling exits cleanly

---

## Conventions

- All logging is structured (JSON)
- Trace IDs are UUIDs
- Timeouts are configurable
- No stdout noise; all output is logged

---

## MANUAL

Keep supervision focused on process lifecycle. Observability details go to logging infrastructure.

Parent reference: ../AGENTS.md
