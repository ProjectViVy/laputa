# Epic 2 — Typed SDK and AGENT-VIVY Lifecycle Bridge

**Outcome:** a framework-neutral public client exists, and AGENT-VIVY proves the first native lifecycle integration without leaking host internals.

**Requirements:** FR-SDK-01..02, FR-VIVY-01..03, NFR-02..06  
**Architecture:** EA-003..006, EA-009..010  
**Dependencies:** Epic 1 contract suite; AGENT-VIVY worktree/branch plan separately approved

## Story E02-S01 — Publish SDK core and transport contract

**Acceptance criteria**

- Public module exports DTOs, manifest/discovery, immutable binding, Bootstrap/Search/Expand/Capture/Remember and typed errors.
- SDK imports neither Garden/Vivy internal packages nor Eino/MCP/storage types.
- Timeout, correlation ID, idempotency and retry classification are explicit client options.
- Contract-generated or hand-written types match the frozen API exactly.

## Story E02-S02 — Build resilient client semantics

**Acceptance criteria**

- Retry is limited to network/5xx/429 and respects deadline/backoff; 4xx/CAS conflict never retries.
- Capture supports receipt polling and reports pending/degraded states accurately.
- No helper silently converts unavailable data-plane result into authority success.
- Unit tests cover serialization, error normalization, idempotency key handling and version negotiation.

## Story E02-S03 — Add Vivy internal MemoryPort composition seam

**Acceptance criteria**

- MemoryPort is a narrow Vivy-owned interface injected through runtime composition, not `sdk/plugin`.
- Existing SessionID/RunID/provenance map directly; no duplicate host Session/Run/Journal appears.
- Dynamic bootstrap is placed before model request outside cache-stable static instruction construction.
- Memory selection/provenance is journaled as bounded refs/hash/budget/degraded state.

## Story E02-S04 — Add terminal capture outbox bridge

**Acceptance criteria**

- Existing terminal RunHook/Journaling starts capture after completed/failed/canceled terminal event as policy permits.
- Queue is durable, bounded, restart-safe and flushes/shuts down deterministically.
- Pending/failed delivery is observable; delivery failure cannot alter terminal Run result.
- Replay uses stable host event ID/content hash and preserves idempotency.

## Story E02-S05 — Vivy first-party conformance suite

**Acceptance criteria**

- A real Vivy run validates bootstrap audit → model request → terminal event → capture delivery.
- Timeout/5xx produces a degraded but completed permitted run.
- Restart preserves pending queue and does not change Frozen Core for existing session.
- No test uses Vivy RPC, Face, ChannelHost or outbound MCP as a substitute lifecycle path.
