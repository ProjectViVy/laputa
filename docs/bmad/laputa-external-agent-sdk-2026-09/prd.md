# Laputa External Agent SDK — PRD

**BMAD phase:** Planning  
**Status:** Planning baseline; see `sprint-status.yaml` for execution state  
**Date:** 2026-09-04  
**Scope:** Brownfield Garden/Laputa/Mentle plus first-party AGENT-VIVY integration planning

## 1. Problem

Garden has completed its authority and recovery cutover, but an external Agent has no single supported way to discover capabilities, bind an existing host session, obtain bounded context, progressively retrieve evidence, capture terminal turns, or handle degradation. Calling arbitrary `/v2` endpoints or mounting MCP tools alone leaves lifecycle semantics, versioning, typed errors, and conformance undefined.

## 2. Product outcome

A host Agent can adopt `laputa-agent/1` and:

1. discover the server contract and allowed capabilities without leaking secrets;
2. bind its existing authenticated identity and session without creating a competing Session/Run model;
3. obtain a six-slot session-frozen bootstrap plus bounded requested evidence;
4. use Search → Expand progressive disclosure for memory/evidence;
5. deliver completed-turn capture asynchronously, idempotently, and without failing its primary run;
6. invoke explicit Persona/ACTMEM tools only when capability permits;
7. receive equivalent domain semantics through REST, typed SDK, and MCP.

## 3. Users and jobs

| User | Job |
|---|---|
| Host runtime developer | Integrate a memory/persona authority without importing Garden internals or inventing lifecycle semantics |
| AGENT-VIVY developer | Map existing Session/Run/Journal/RunHook into bootstrap and terminal capture with auditability |
| Generic external Agent developer | Use capability discovery, typed requests/errors and progressive recall safely |
| Profile owner | Permit bounded Agent access while retaining Persona review/CAS and explicit-only protected surfaces |
| Operator | Diagnose pending capture/index degradation without falsely declaring canonical writes failed |

## 4. Functional requirements

### Contract and identity

- **FR-CON-01** — Publish a secret-free `laputa-agent/1` capability manifest at a stable well-known endpoint.
- **FR-CON-02** — Declare protocol version, transport, capability matrix, principal requirements, request budgets, automatic/explicit context policy, and stable errors.
- **FR-CON-03** — Bind `profile_id`, `agent_id`, `platform`, `session_id`, `turn_id`, and `event_id` to an authenticated request context; only the server credential establishes principal/capability.
- **FR-CON-04** — Keep a client binding immutable and prohibit second Session/Run creation by the SDK.

### Context and retrieval

- **FR-REC-01** — Agent-profile Bootstrap (`POST /v2/agent/bootstrap`) returns the persisted six-document Frozen Core for the validated binding session and optional bounded Mentle evidence.
- **FR-REC-02** — Agent-profile Search (`POST /v2/agent/search`) returns query-scoped card results with score, source, canonical revision/status, trace and expansion reference.
- **FR-REC-03** — Agent-profile Expand (`POST /v2/agent/evidence/{card_id}`) returns bounded evidence/source content with provenance and continuation metadata.
- **FR-REC-04** — WORLD, ACTMEM, Persona history/reviews, raw authority paths and unbounded tails are absent from automatic Bootstrap and Recall.
- **FR-REC-05** — Explicit Persona, ACTMEM, evidence and trace endpoints remain separately capability-gated.

### Capture and canonical memory

- **FR-CAP-01** — Capture accepts terminal completed-turn payloads with stable event identity and content hash.
- **FR-CAP-02** — Identical capture replay is idempotent; changed content under the same event identity returns a typed conflict.
- **FR-CAP-03** — Asynchronous capture returns acceptance/receipt/status and supports durable retry/restart recovery.
- **FR-CAP-04** — `Remember` maps to Mentle canonical `facade.Service`; no SDK route writes a vector store, JSONL log, raw SQLite or Persona Markdown.
- **FR-CAP-05** — Canonical success with delayed derived index is represented as `index_pending`, not false failure.

### Typed SDK and host adapter

- **FR-SDK-01** — Provide a framework-neutral typed client with Discover, Bind, Bootstrap, Search, Expand, Capture, Remember, Persona.Get/Propose, and ACTMEM.Read/Query.
- **FR-SDK-02** — Do not expose Eino, AGENT-VIVY internal packages, Garden internal packages, raw HTTP details, MCP internal types, SQLite handles, or authority paths in the public SDK API.
- **FR-VIVY-01** — Inject an internal Vivy MemoryPort through runtime composition; bootstrap precedes `model.request` but does not alter the cache-stable static instruction prefix.
- **FR-VIVY-02** — Persist a bounded selected-memory/provenance audit event for model-visible context.
- **FR-VIVY-03** — Trigger terminal capture from the existing RunHook/Journal lifecycle; a capture delivery failure does not change a terminal Vivy Run outcome.

### MCP and authorization

- **FR-MCP-01** — Garden MCP maps its tool surface to canonical REST/domain results and stable error envelope.
- **FR-MCP-02** — MCP cannot independently implement retrieval, direct storage mutation, authorization, or a second error taxonomy.
- **FR-AUTH-01** — Agent may capture/remember/propose only as explicitly granted; it cannot approve Persona review, directly edit protected Persona, maintain ACTMEM by default, or repair/rebuild indexes.

## 5. Non-functional requirements

- **NFR-01:** deterministic protocol negotiation and additive-version discipline.
- **NFR-02:** context and evidence budgets are explicit and enforced.
- **NFR-03:** timeout/5xx/429 can degrade data-plane access without blocking a host primary Run; authorization/validation failures fail closed.
- **NFR-04:** REST/MCP/SDK preserve request ID, trace ID, stable code, retryability and domain semantics.
- **NFR-05:** all sensitive content is excluded from diagnostic/event duplication unless specifically authorized and bounded.
- **NFR-06:** restart preserves Frozen Core immutability, capture queue/status and derived-index recovery semantics.
- **NFR-07:** Windows remains a first-class development/test environment; no Unix-only hook is the required integration path.

## 6. Non-goals

- Creating a new generic Agent runtime, Session service, Journal, policy engine, or control plane.
- Exporting Vivy `vivy.rpc.v1`, Face, ChannelHost or user-plugin ABI as Laputa protocol.
- TencentDB L0–L3 compatibility, automatic Persona generation, or a transparent LLM proxy in v1.
- memsearch-style scanning of Persona/profile trees or filesystem-path authorization.
- Optional vector backend work, canonical storage change, or any Persona/ACTMEM semantic change.
- Direct Agent approval of Persona review, direct protected Persona writes, or default ACTMEM maintenance.

## 7. Success signals

- A contract test client can discover, bind, bootstrap, Search → Expand, Capture and Remember with stable typed behavior.
- Negative conformance cases prove automatic context never contains WORLD/ACTMEM/history/raw authority paths.
- A Vivy integration passes a terminal capture replay/restart/degraded-run test without changing Vivy runtime authority.
- REST and MCP return equivalent results/codes for the same legal request and denial.
- All public package boundaries pass an import/API scan proving no host or storage internals leak.
