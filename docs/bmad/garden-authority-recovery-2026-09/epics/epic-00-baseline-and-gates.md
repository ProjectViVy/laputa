# Epic 0 — Baseline, Contract and Deletion Gates

**Outcome:** establish a trustworthy implementation baseline and executable clean-break constraints before runtime changes.

**Requirements:** NFR-06, NFR-07, FR-CTX-05
**Architecture:** AR-001, AR-012, Verification Architecture
**Dependencies:** none
**Exit checkpoint:** owner reviews baseline inventory and implementation branches may start.

## Story 0.1 — Isolate the implementation baseline

**Intent:** prevent the existing dirty working tree from contaminating ownership and verification.

**Acceptance criteria**

- Record current branch, tracked/untracked changes, and which files belong to the prior Persona slice.
- Establish an owner-approved baseline strategy: commit existing accepted work, create an isolated worktree/branch, or explicitly scope the dirty tree.
- No destructive Git operation occurs without confirmation.
- A later story can identify its own diff unambiguously.

## Story 0.2 — Restore the pre-implementation quality gate

**Intent:** ensure the batch does not start from a falsely green baseline.

**Acceptance criteria**

- Fix the date-dependent `TestMonthlyReportModulesViaHTTP` fixture by deriving source time from the generation window or injecting a clock.
- `go test ./...` passes in Garden, Laputa and Mentle.
- Console build passes.
- Results are recorded as baseline evidence; no retired-behavior test is counted as target-contract proof.

## Story 0.3 — Freeze the target API and error contract

**Intent:** eliminate `/files` versus `/documents` and `/requests` versus `/reviews` drift before adapters multiply.

**Acceptance criteria**

- Produce one route/DTO/error-code table for Persona, ACTMEM and IndexHealth using AR-012.
- Define principal requirements per write endpoint.
- Define breaking removal list; no alias or compatibility window.
- REST, MCP and Console stories cite the same contract.

## Story 0.4 — Add deletion and boundary scanner tests

**Intent:** make clean-break regression mechanically impossible.

**Acceptance criteria**

- Scanner covers runtime references to `laputa/governance`, `.laputa/sections`, numbered authority names, `SectionMemoryMD`, JSON Patch, `GovernanceProjection`, `WorldProjector`, `/v2/governance` and `/v2/cognitive/world`.
- Scanner distinguishes dated archive/document references from target runtime references.
- Mentle boundary scan rejects Persona/WORLD/ACTMEM authority dependencies.
- Test is initially allowed to report known violations, then becomes blocking at Epic 2 cutover.

## Story 0.5 — Convert architecture decisions into test inventories

**Intent:** ensure each risky decision has an owner and proof before coding.

**Acceptance criteria**

- Persona parity matrix maps each DIVA behavior to a test.
- ACTMEM parity matrix maps store/query/capsule behavior to a test.
- Mentle fault matrix maps crash points, poison jobs, CAS and rebuild rollback to tests.
- Context negative matrix proves WORLD/ACTMEM absence by type, response and assembled text.
