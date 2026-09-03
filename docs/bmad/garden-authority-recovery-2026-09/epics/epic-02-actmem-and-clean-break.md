# Epic 2 — ACTMEM, Frozen Core and Runtime Clean Break

**Outcome:** Garden runs from Markdown Persona + independent ACTMEM, with no retired JSON authority or automatic WORLD/ACTMEM projection.

**Requirements:** FR-ACT-01..05, FR-CTX-01..05, NFR-01..02
**Architecture:** AR-002, AR-005..006, AR-011..012
**Dependencies:** Epic 1 service contracts; Epic 0 scanner
**Exit checkpoint:** atomic composition-root cutover; deletion scanner becomes blocking.

## Story 2.1 — Port ACTMEM model and atomic store from DIVA

**Acceptance criteria**

- Path is `<profile>/actmem/ACTMEM.MD` with DIVA front matter and Pulse/Recap/Work structure.
- Revision CAS, no-op preservation and atomic replacement match DIVA.
- Pulse/Recap/Work and item caps are enforced without silent truncation.
- Missing/malformed/CAS/cap/restart tests pass.
- Store imports neither Persona review nor Mentle.

## Story 2.2 — Port ACTMEM maintenance and capsule lifecycle

**Acceptance criteria**

- Implement append/update for Pulse/Recap and Goal/Open/Next/Constraints/Pointers maintenance.
- Implement complete/drop/edit operations with typed range/section errors.
- Implement bounded session fold and capsule list/read/delete.
- Capsule path validation prevents traversal and invalid projection.
- DIVA parity tests cover no-op, caps and lifecycle.

## Story 2.3 — Build explicit ACTMEM read/query adapters

**Acceptance criteria**

- Stable bounded read/query capability returns at most the DIVA read cap.
- Maintenance routes/tools are separate from the stable read capability.
- No Persona review, Chat Approval or Mentle promotion side effect.
- REST tests prove explicit invocation and restart continuity.

## Story 2.4 — Implement restart-safe session Frozen Core

**Acceptance criteria**

- Capture exactly six bounded Persona projections on first session bootstrap.
- Persist snapshot and source revisions in Garden SQLite keyed by session ID.
- Same session remains unchanged after Persona edits and Garden restart.
- New session sees current Persona revisions.
- FrozenCore type cannot represent WORLD or ACTMEM.

## Story 2.5 — Move checkpoint and report state out of Governance

**Acceptance criteria**

- WorkingSet checkpoint uses Garden SQLite and survives restart.
- Reports/human modules remain Garden-owned SQLite artifacts.
- Neither service imports or writes Laputa governance sections.
- Existing report behavior remains covered, including deterministic time-window tests.

## Story 2.6 — Atomically cut recall/bootstrap to Frozen Core

**Acceptance criteria**

- Fast/Deep/bootstrap/trace assembly uses Frozen Core plus bounded Mentle evidence.
- Remove GovernanceProjection, WorldProjector, ContextView.World and world source/budget/text assembly.
- WORLD and ACTMEM are absent from DTOs and rendered context, not merely empty.
- Degraded Mentle behavior retains Frozen Core without fallback to JSON sections.
- Positive Frozen Core and negative WORLD/ACTMEM tests pass.

## Story 2.7 — Delete the retired runtime composition

**Acceptance criteria**

- `garden/main.go` no longer initializes FileStore/Engine/GovernedService/legacy WorldStore.
- Generic governance and cognitive-world routes are removed.
- Section-backed checkpoint/report adapters are deleted.
- Runtime tests/fixtures asserting retired behavior are deleted or rewritten.
- Blocking deletion scanner reports zero target-runtime violations.

## Story 2.8 — Prove the clean-break vertical slice

**Acceptance criteria**

- Five-file Persona initialization and review work end-to-end.
- ACTMEM survives new session and process restart.
- Frozen Core remains session-immutable.
- Explicit WORLD and ACTMEM reads succeed while automatic responses contain neither.
- Historical `.laputa/sections` files, if present, are ignored.
