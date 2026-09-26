# Epic 3 — MCP Conformance and Explicit Tool Surface

**Outcome:** `garden-mcp` becomes a truthful thin adapter for the same external-Agent contract; explicit resources remain outside automatic context.

**Requirements:** FR-MCP-01..02, FR-REC-05, FR-AUTH-01  
**Architecture:** EA-001, EA-004..005, EA-007..008  
**Dependencies:** Epic 1; may run after canonical REST cases are frozen

## Story E03-S01 — Map MCP tools to canonical domain methods

**Acceptance criteria**

- Tool registration/call mapping delegates to canonical REST/domain client and does not implement a second search/write/auth layer.
- Every tool declares required capability, input schema, bounded output and structured success/error fields.
- Tool names and descriptions distinguish automatic bootstrap from explicit resource access.

## Story E03-S02 — Normalize structured MCP errors/results

**Acceptance criteria**

- MCP carries code, request ID, trace ID, retryability and reason equivalently to REST.
- Text-only error output is rejected by protocol tests.
- Unauthorized/forbidden/conflict/unavailable/index-pending remain distinguishable to an Agent.

## Story E03-S03 — Explicit Persona and ACTMEM tool gate

**Acceptance criteria**

- Full Persona/WORLD, history/reviews, ACTMEM and raw evidence have separate explicit tools/capabilities.
- Agent proposal can create a review but cannot approve/reject or protected direct-write.
- ACTMEM maintenance and operator repair remain absent from default Agent tools.
- Tool use never auto-injects data into later Bootstrap without an explicit host request.

## Story E03-S04 — REST/MCP equivalence conformance suite

**Acceptance criteria**

- Same legal request yields equivalent domain payload/error code through REST and MCP.
- Negative test proves MCP cannot bypass profile/principal/canonical write authority.
- Search/Expand results retain provenance and bounded disclosure under both transports.
