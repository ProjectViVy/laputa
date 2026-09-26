# `laputa-agent/1` — Proposed API Contract Freeze

**Status:** Living contract freeze. Discovery is implemented; every other capability remains proposed until its implementation and conformance evidence are recorded.  
**Canonical transport:** current Garden loopback HTTP JSON; HTTPS is a future deployment transport, not a v1 advertised capability  
**Discovery:** `GET /.well-known/laputa-agent.json`

## 1. Manifest

```json
{
  "contract": "laputa-agent",
  "version": "1",
  "transports": ["http"],
  "capabilities": [
    "bootstrap", "memory.search", "memory.expand", "session.capture",
    "memory.remember", "persona.read", "persona.propose",
    "actmem.read", "actmem.query"
  ],
  "automatic_context": ["frozen_core", "bounded_evidence"],
  "explicit_only": ["world", "actmem", "persona_history", "persona_mutation", "raw_source"],
  "budgets": {
    "bootstrap_chars": {"default": 8000, "min": 256, "max": 64000},
    "search_limit": {"default": 20, "min": 1, "max": 100},
    "expand_per_item_chars": {"default": 800, "max": 4000},
    "expand_total_chars": {"default": 4000, "max": 16000}
  },
  "principal_requirements": {"session.capture": "agent", "persona.propose": "agent"},
  "errors": ["invalid_request", "unauthenticated", "forbidden", "not_found", "conflict", "unavailable", "index_pending" ]
}
```

Discovery initially advertises only the live loopback `http` transport. It must not announce `https` until Garden actually owns a TLS listener, and it must not announce `mcp-stdio` until the MCP adapter meets the same external-contract conformance gate.

## 2. Identity envelope

Every bound request carries:

```json
{
  "profile_id": "server-authorized profile reference",
  "agent_id": "host implementation identity",
  "platform": "vivy|openclaw|hermes|custom",
  "session_id": "existing host conversation identity",
  "turn_id": "existing host run identity",
  "event_id": "stable terminal event identity"
}
```

Only `agent_id`, `platform`, `session_id`, `turn_id`, and `event_id` are caller provenance. Token/capability authorization determines principal; v1 accepts `profile_id` only when it exactly equals the server's configured single canonical profile (default `default`). A request body/header cannot select a different profile or expand access. `X-Garden-Actor` is audit metadata only.

## 3. Canonical semantic mapping

| Contract method | Canonical Garden basis | Semantics |
|---|---|---|
| Discover | `GET /.well-known/laputa-agent.json` | **implemented** discovery endpoint; only current loopback HTTP transport and conformance-proven capabilities may be advertised |
| Bind | `POST /v2/agent/bind` | **implemented** stateless validation endpoint; Agent bearer + configured single profile; returns validated immutable binding metadata and creates no Garden Session/Run |
| Bootstrap | `POST /v2/agent/bootstrap` | **new Agent profile** over existing recall service; required AgentBinding + Agent bearer; reuses six-slot Frozen Core and bounded evidence without altering ordinary `/v2/recall/bootstrap` semantics |
| Search | `POST /v2/agent/search` | **new Agent profile** over Materials provider; required AgentBinding + Agent bearer; query-scoped cards with source/revision/score/trace |
| Expand | `POST /v2/agent/evidence/{card_id}` | **new Agent profile** over Materials provider; required AgentBinding + Agent bearer; bounded evidence/source by opaque card reference |
| Domain materials hardening | `GET /v2/materials/*` | retain ordinary domain routes but add read-principal gate; they are not an external SDK binding bypass |
| Capture | `POST /v2/ingest/sessions` + `GET /v2/ingestions/{id}` | adapt/profile: current ingest already has async `202`, `session_id + event_id + content_hash` idempotency and changed-hash conflict; add authenticated profile/agent/platform/turn binding, terminal phase semantics, structured receipt/error, and principal-gated status read |
| Remember | `POST /v2/memories` | reuse only through Mentle Facade; add AgentBinding/idempotency profile |
| Persona.Get/Propose | `/v2/persona/documents*`, `/reviews*` | explicit only; review/direct/P16 rules preserved |
| ACTMEM | `/v2/actmem*` | explicit only; maintenance is not default Agent capability |
| Trace/health | recall trace / index health | explicit diagnostic capability; manifest declares access |

Exact Garden route/DTO names remain frozen by Epic 0 after live route audit. This contract deliberately avoids adding aliases where a compliant existing route exists.

## 4. Requests and responses

### Agent-profile context and retrieval endpoints

The external SDK never relies on loopback read exemption or smuggles identity through ad-hoc headers. It uses dedicated Agent-profile routes with a required `binding` object in the JSON body:

```text
POST /v2/agent/bootstrap
POST /v2/agent/search
POST /v2/agent/evidence/{card_id}
```

Each route requires an Agent bearer token, validates the same single-profile AgentBinding as `/v2/agent/bind`, and invokes the existing domain recall/material service. Existing ordinary `/v2/recall/bootstrap` and `GET /v2/materials/*` remain domain routes; the latter must gain a read-principal gate but cannot serve as an external SDK binding bypass.

### Bootstrap

```json
{
  "binding": {"profile_id":"p", "agent_id":"a", "platform":"vivy", "session_id":"s"},
  "intent": "user message intent summary",
  "evidence_budget": {"cards": 8, "chars": 6000}
}
```

Response contains `frozen_core` with exactly six named documents, bounded `evidence[]`, `trace_id`, `context_hash`, `budget`, and `degraded`. It contains no `world`, `actmem`, `history`, authority path, or unbounded transcript.

### Search and Expand

Search request includes binding, query, scope and bound limit. Each result includes opaque expansion reference, source reference, canonical revision/status, score, trace ID and snippet budget. Expand accepts an opaque reference and a character budget; it returns bounded content, provenance and continuation only when capability permits.

### Capture

```json
{
  "binding": {"profile_id":"p", "agent_id":"a", "platform":"vivy", "session_id":"s", "turn_id":"r", "event_id":"r:terminal:12"},
  "content_hash": "sha256:...",
  "phase": "completed",
  "completed_turn": {"bounded": "redacted/limited host result"},
  "provenance": {"model": "host metadata", "context_trace_id": "..."}
}
```

Success is either final `200` receipt or `202 Accepted` receipt with `capture_id`, `status`, `request_id`, and polling reference. Same event/content returns the original logical receipt. Same event with different hash returns `409 conflict`.

### Remember

Remember includes content, allowed metadata/source provenance and `Idempotency-Key` derived from `session_id + event_id + content_hash`. Canonical commit is success even if output reports `index_pending`.

## 5. Error envelope

```json
{
  "error": {
    "code": "conflict",
    "message": "stable human-readable summary",
    "request_id": "...",
    "trace_id": "...",
    "retryable": false,
    "reason": "event_content_hash_mismatch"
  }
}
```

| Code | Retryable | Meaning |
|---|---:|---|
| `invalid_request` | no | schema, budget or semantic validation error |
| `unauthenticated` | no | no valid credential |
| `forbidden` | no | principal/capability denial |
| `not_found` | no | authorized resource absent |
| `conflict` | no | CAS/idempotency/event-content mismatch |
| `unavailable` | yes | temporary service/index/network unavailability |
| `rate_limited` | yes | retry after declared delay |
| `index_pending` | no | canonical write succeeded; derived work remains pending |

MCP must encode all fields structurally, not only an error string.

## 6. Principal matrix

| Action | read | agent | user | operator |
|---|---:|---:|---:|---:|
| Discover / Bootstrap / Search / Expand | yes | yes | yes | diagnostics only |
| Capture / Remember | no | granted capability | granted capability | no default |
| Persona read/history | granted | granted | granted | diagnostics only |
| Persona proposal | no | granted | optional | no default |
| Persona protected direct write / review decision | no | no | yes | no |
| ACTMEM read/query | granted | granted | granted | diagnostics only |
| ACTMEM maintenance | no | no default | granted | no default |
| Index repair/rebuild | no | no | no | granted |

P16 is excluded until DIVA parity and an explicit future capability declaration are accepted.
