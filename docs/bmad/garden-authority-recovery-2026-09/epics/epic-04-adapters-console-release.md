# Epic 4 — Domain Adapters, MCP, Console and Release Gate

**Outcome:** users and agents access the new authority/recovery model through truthful, domain-specific interfaces; all legacy surfaces are removed.

**Requirements:** FR-API-01..02, FR-MCP-01..02, FR-UI-01..02
**Architecture:** AR-004, AR-006, AR-011..014
**Dependencies:** Epics 1–3 domain services and cutover contracts
**Exit checkpoint:** complete release/readiness gate and owner acceptance.

## Story 4.1 — Apply principal-scoped local capability authentication

**Acceptance criteria**

- User/agent/operator credentials map to principal classes and are compared in constant time.
- Tokens never appear in logs, errors, audit bodies or Console storage readable by unrelated origins.
- Persona write/review/repair and ACTMEM maintenance endpoints enforce the matrix from AR-004.
- Read-only loopback policy is documented explicitly.
- Spoofed `X-Garden-Actor` cannot elevate permissions.

## Story 4.2 — Cut over REST to the stable domain contract

**Acceptance criteria**

- Persona documents/reviews/history/repair, ACTMEM and IndexHealth routes follow Story 0.3.
- Old `/files`, `/requests`, `/v2/governance/*` and `/v2/cognitive/world` routes return not found and are absent from docs/types.
- DTOs contain no arbitrary JSON authority body, action/path/value mutation or World projection type.
- Contract tests cover success, typed failures and auth classes.

## Story 4.3 — Align Garden MCP with domain rules

**Acceptance criteria**

- Persona/ACTMEM tools expose only operations legal for the MCP agent principal.
- Protected Persona mutation creates a review; it cannot direct-write or decide reviews.
- Memory write uses canonical facade via REST.
- Index health is read-only and truthful.
- Protocol tests exercise list/call/typed-error behavior.

## Story 4.4 — Correct `memory_search` semantics

**Acceptance criteria**

- Every result is demonstrably matched by the query or explicitly labeled as a separate recent-memory section.
- No unfiltered recent tail is represented as matching evidence.
- Ranking/source/provenance fields are preserved.
- Empty/degraded/unavailable cases have distinct MCP responses and tests.

## Story 4.5 — Complete Persona and Memory workspaces

**Acceptance criteria**

- Persona supports initialization, full-document edit, review queue/decision and immutable history using live APIs.
- Memory explicitly separates ACTMEM, Mentle evidence and Garden activity.
- WORLD and ACTMEM full text loads only after explicit user action.
- Stale edits preserve draft and present conflict/reload flow.
- No unavailable operation appears as a live button.

## Story 4.6 — Replace legacy Console information architecture

**Acceptance criteria**

- Top-level core workspaces are Persona, Memory, Evolution and Chat Approval.
- Governance Map, WorldClaimsPanel, compat labels and retired DTOs are deleted.
- Mailbox is presented under Evolution without changing its domain ownership.
- Operations uses live IndexHealth and runtime probes.
- UI tests prove no background WORLD/ACTMEM request.

## Story 4.7 — Reconcile bounded EvoMap candidate input

**Acceptance criteria**

- Candidate input accepts bounded ACTMEM excerpt, activity/trace references and Mentle evidence references.
- It rejects Persona bodies/history and unbounded content.
- Candidate creation cannot install/publish a host artifact.
- Existing EvoMap review/privacy/mailbox ownership remains unchanged.

## Story 4.8 — Run final project acceptance gate

**Acceptance criteria**

- Garden, Laputa and Mentle full suites pass.
- New Garden e2e proves Frozen Core, explicit WORLD/ACTMEM and canonical recovery behavior.
- Console test/build passes using live contract fixtures.
- Deletion/boundary scans pass.
- `git diff --check` passes for owned changes.
- Architecture/PRD/story traceability has no orphan requirement.
- Operator runbook documents backup, rebuild, degraded health and rollback.
