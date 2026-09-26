# Epic 1 — Canonical External-Agent Access Plane

**Outcome:** Garden exposes a capability-discovered, identity-bound REST contract for external Agents while preserving authority boundaries.

**Requirements:** FR-CON-01..04, FR-REC-01..05, FR-CAP-01..05, FR-AUTH-01  
**Architecture:** EA-001..008, EA-010..011  
**Dependencies:** Epic 0; ADR-0015 accepted

## Story E01-S01 — Capability discovery manifest

**Acceptance criteria**

- Well-known manifest is generated from one versioned contract source and contains no token/path/authority content.
- Unknown mandatory contract version or unavailable required capability produces a typed preflight failure.
- Only implemented/conformance-tested capabilities are advertised.
- Manifest contract tests cover additive optional capability behavior.

## Story E01-S02 — Authenticated Agent binding

**Acceptance criteria**

- Server binds authorized profile/principal from credential, validates immutable agent/platform/session provenance and rejects mismatch.
- Actor header cannot elevate permissions.
- Request/trace IDs flow through binding, recall and capture.
- No endpoint creates a competing Garden Session/Run for an existing host identity.

## Story E01-S03 — Bootstrap and progressive retrieval profile

**Acceptance criteria**

- Bootstrap uses session-frozen six-document context and bounded evidence only.
- Schema and negative tests prove WORLD/ACTMEM/history/raw-path absence.
- Search is query-scoped and Search → Expand preserves source/revision/status/score/trace.
- Explicit resource calls remain capability-gated and do not create recall dependencies.

## Story E01-S04 — Terminal capture and receipt lifecycle

**Acceptance criteria**

- Capture accepts only terminal phase with event ID/content hash and returns a stable receipt/status model.
- Same event/hash replay is idempotent; changed hash conflicts.
- Capture reaches Mentle facade and exposes canonical success plus index-pending correctly.
- Retryability classification follows EA-010; no capture code writes raw storage.

## Story E01-S05 — Canonical access-plane contract suite

**Acceptance criteria**

- Black-box tests cover discovery, binding, bootstrap, retrieval, capture, Remember, principal denial and typed errors.
- Restart tests prove Frozen Core and capture-status continuity.
- Route/API contract documentation exactly matches live behavior.
