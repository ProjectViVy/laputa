# External Agent SDK — Conformance Test Inventory

This inventory is a planning artifact. Test identifiers define future acceptance; none is implemented or passed by this document.

| ID | Epic | Scenario | Required result |
|---|---|---|---|
| CON-01 | 0/1 | Discover supported contract | version/capabilities/budgets/error catalog are machine-readable and secret-free |
| CON-02 | 1 | Unknown required version/capability | typed preflight denial before side effect |
| ID-01 | 1 | Token/profile/session binding | server-authorized profile only; binding is immutable |
| ID-02 | 1 | Spoofed actor header | no privilege escalation |
| CTX-01 | 1 | Bootstrap new session | exactly six Frozen Core slots plus bounded requested evidence |
| CTX-02 | 1 | Bootstrap same session after Persona edit/restart | same frozen revision/context hash |
| CTX-03 | 1 | Automatic-context negative sentinel | WORLD, ACTMEM, history, raw path and unbounded tail absent structurally and textually |
| REC-01 | 1 | Search | query-scoped cards retain score/source/revision/status/trace |
| REC-02 | 1 | Expand | bounded evidence preserves provenance and continuation semantics |
| REC-03 | 3 | Explicit resource access | WORLD/ACTMEM/history requires declared capability/tool |
| CAP-01 | 1 | Terminal capture accepted | receipt/status contains request/trace/capture identity |
| CAP-02 | 1/2 | Same event + same content hash | idempotent logical receipt, no duplicate canonical memory |
| CAP-03 | 1/2 | Same event + changed content hash | typed conflict, no overwrite |
| CAP-04 | 1/2 | Canonical commit/index lag | success with `index_pending`, not failure |
| CAP-05 | 2 | Host restart with pending capture | queue resumes safely; host terminal outcome unchanged |
| DEG-01 | 2/4 | Bootstrap/Search timeout or 5xx | permitted host Run degrades/audits rather than fails from memory availability |
| DEG-02 | 2/4 | Capture timeout/429/5xx | retryable queue entry; bounded retry/backoff |
| DEG-03 | 2/4 | 4xx/forbidden/conflict | fail closed; never retry |
| AUTH-01 | 1/3 | Agent Persona proposal | legal proposal only; review decision denied |
| AUTH-02 | 1/3 | Agent protected Persona write/ACTMEM maintenance/index repair | typed forbidden by default |
| MEM-01 | 1 | Remember/Capture write path | enters Mentle facade/canonical outbox only |
| EQ-01 | 3 | REST versus MCP legal request | equivalent domain payload and stable code |
| EQ-02 | 3 | REST versus MCP denial/conflict/unavailable | equivalent structured error fields |
| SDK-01 | 2/4 | Public SDK import boundary | no Garden/Vivy internal, Eino, MCP-internal or storage import |
| VIVY-01 | 2 | Bootstrap lifecycle position | occurs before model request and outside static prompt prefix |
| VIVY-02 | 2 | Context selection audit | bounded refs/hash/budget/degraded event is journaled |
| VIVY-03 | 2 | Terminal RunHook capture | starts only after terminal event; does not append/fake terminal event |
| OPS-01 | 4 | Manifest/release truthfulness | unavailable/stub capability never advertised as supported |

## Harness contract

The future harness has one versioned fixture corpus and three adapters:

```text
fixture corpus + expected domain outcome
  ├─ REST black-box client
  ├─ laputa-sdk-go client
  └─ garden-mcp tool client
```

Each fixture owns: authenticated principal/token fixture, profile/session/turn/event IDs, request payload, expected HTTP/MCP/typed result, request/trace correlation assertions, and explicit absence sentinels. It never asks a transport test to inspect raw Persona files, canonical SQLite, vector state, or host private Journal rows.

### Fixture classes

| Class | Required rows | Cross-transport assertion |
|---|---|---|
| Discovery | CON-01..02 | advertised capability/version/budget/error semantics agree |
| Identity | ID-01..02 | token-derived profile/principal wins; actor metadata cannot elevate |
| Automatic context | CTX-01..03 | six-slot core and absence sentinel are equal semantically |
| Progressive retrieval | REC-01..03 | opaque refs/provenance/bounds agree; no unfiltered recent tail |
| Capture/memory | CAP-01..05, MEM-01 | receipt/idempotency/conflict/index-pending agree |
| Availability | DEG-01..03 | retryability/degraded versus fail-closed classification agrees |
| Authority | AUTH-01..02 | agent proposal/denial matrix agrees |
| Transport | EQ-01..02 | normalized code/request/trace/retryable/reason agree |
| Host boundary | SDK-01, VIVY-01..03 | SDK has no internal imports; lifecycle assertions use observable events |
| Release | OPS-01 | manifest does not advertise unavailable surface |

### Required negative sentinels

The fixture corpus uses distinct synthetic values for `WORLD`, `ACTMEM`, Persona history, raw authority path, unbounded transcript tail, and a spoofed actor label. Every automatic Bootstrap/Search response must be asserted not to contain these values. Explicit-resource fixtures assert the reverse only when their capability is granted.

## Required test isolation

- Use fixture profile/session IDs and synthetic sentinel text, never a real user Persona or raw production palace.
- Use black-box REST/MCP/SDK behavior for conformance; storage probes may verify one canonical outcome but cannot replace transport assertions.
- Fault tests must inject deterministic transport/service failures, not rely on timing accidents.
- All Agent-visible data capture must be bounded/redacted; event tests must assert non-duplication of sensitive full content.
