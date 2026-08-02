<!-- Parent: ../../AGENTS.md -->

# laputa/governance/web — Governance HTTP API

**Generated:** 2026-08-01  
**Purpose:** HTTP interface to governance engine

---

## Purpose

The `web/` package exposes governance operations via HTTP:

- **Section read/write** — GET/POST /sections/{name}
- **Authority checks** — GET /authority/{scope}/{action}
- **Rhythm reports** — GET /reports/{kind}
- **Audit trail** — GET /audit?since=&limit=
- **Health check** — GET /health

---

## Structure

```
web/
├── server.go         # HTTP server and routes
└── server_test.go
```

---

## Key Endpoints

```
GET    /sections                 # List all sections
GET    /sections/{name}          # Read section
POST   /sections/{name}          # Update section (auth required)
GET    /authority/{scope}/{action} # Check authority
GET    /reports/{kind}           # Fetch rhythm report
GET    /audit?since=&limit=      # Read audit trail
GET    /health                   # Health check
```

---

## Testing

```bash
cd laputa
GOSUMDB=off go test -v ./governance/web/...
```

**Behavioral tests:**

- Section endpoints return correct data
- Authority checks enforce permissions
- Audit trail is immutable (read-only)
- Health check reports accurate status

---

## Conventions

- All responses are JSON
- All times in response are RFC3339
- Errors include error_code and message
- No content filtering at HTTP layer (apply in governance engine)

---

## MANUAL

Keep web focused on HTTP mechanics. Business logic stays in engine.

Parent reference: ../AGENTS.md
