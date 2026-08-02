<!-- Parent: ../AGENTS.md -->

# garden/fixtures — Test Data and Fixtures

**Generated:** 2026-08-01  
**Purpose:** Reusable test data for unit and integration tests

---

## Purpose

The `fixtures/` directory contains test data, mock objects, and example payloads:

- **JSON payloads** for recall requests, activity events, governance updates
- **Mock data structures** for testing without real backends
- **Example responses** to verify unmarshaling and response handling
- **Test utilities** for fixture generation and cleanup

---

## Structure

```
fixtures/
├── recall_fast_request.json        # Example Fast Recall request payload
├── recall_deep_request.json        # Example Deep Recall request payload
├── activity_event.json             # Example activity event payload
├── governance_projection.json      # Example governance projection response
├── recall_response.json            # Example successful recall response
├── error_response.json             # Example error response
└── (utilities for fixture generation as needed)
```

---

## Usage in Tests

Load a fixture:

```go
import (
    "encoding/json"
    "os"
)

data, err := os.ReadFile("fixtures/recall_fast_request.json")
if err != nil {
    t.Fatalf("load fixture: %v", err)
}

var req RecallFastRequest
if err := json.Unmarshal(data, &req); err != nil {
    t.Fatalf("unmarshal fixture: %v", err)
}
```

---

## Example Payloads

### Fast Recall Request

```json
{
  "scope": "session-123",
  "intent": "What are the key decisions in my project?",
  "max_cards": 10
}
```

### Activity Event

```json
{
  "session_id": "session-123",
  "event_type": "code_edit",
  "timestamp": "2026-08-01T10:30:00Z",
  "data": {
    "file": "main.go",
    "lines_changed": 5
  }
}
```

### Governance Projection Response

```json
{
  "scope": "session-123",
  "authority": "agent_self",
  "allowed_capabilities": ["governance", "recall"],
  "denied_wings": ["debug"]
}
```

---

## Conventions

- JSON format, 2-space indentation
- Filenames describe the type (request, response, event, etc.)
- Use realistic but non-sensitive example data
- Keep payloads representative of actual usage
- Do not commit real user data or API keys

---

## MANUAL

When updating:

1. Keep examples synchronized with actual API contracts
2. Document payload structure in comments
3. Add new fixtures when new request types are added
4. Remove obsolete fixtures when old API routes are retired

Parent reference: `../AGENTS.md`
