# Epic 0 — Contract, Baseline and Conformance Gates

**Outcome:** implementation starts only after contract names, ownership, route mapping, dirty-tree scope and cross-repository evidence are auditable.

**Requirements:** FR-CON-01..04, NFR-01..07  
**Architecture:** EA-001..003, EA-011  
**Dependencies:** ADR-0015 accepted by owner

## Story E00-S01 — Freeze live-surface evidence ledger

**Allowed future implementation scope:** documentation only.

**Acceptance criteria**

- Record Garden route/DTO/error/principal evidence with path:line and identify exact existing route reuse versus genuine gap.
- Record AGENT-VIVY Session/Run/Journal/RunHook/Tool/Config seams with path:line; label proposed adapter paths as future work.
- Record TencentDB and memsearch fact → borrow/reject decision ledger from pinned local snapshots and official URLs.
- Lock both repository HEAD/branch/status; classify dirty changes as owned, excluded or blocking.

## Story E00-S02 — Accept ADR and freeze public boundary

**Acceptance criteria**

- Owner accepts, edits or rejects ADR-0015 before source work begins.
- Manifest fields, required capabilities, semantic version rules, principal matrix, budgets and stable error codes are fixed without placeholder values.
- Public SDK import boundary and module/version ownership are fixed.
- No route, capability, or adapter is marked live without implementation evidence.

## Story E00-S03 — Create cross-transport conformance harness design

**Acceptance criteria**

- One fixture matrix drives REST, SDK and MCP equivalent tests.
- Harness includes success, malformed input, unauthenticated, forbidden, conflict, unavailable, index-pending and restart cases.
- Test data contains explicit negative sentinel values proving WORLD/ACTMEM/history exclusion.
- Harness is black-box at transport boundary and does not inspect raw storage as a substitute for behavior.

## Story E00-S04 — Establish planning-to-implementation gate

**Acceptance criteria**

- Run focused current baseline tests only after live command/path audit; preserve actual output as evidence.
- Validate Markdown links, YAML, unique story IDs and Story-count parity.
- Produce owner decision: authorize implementation or retain planning-only state.
- No source/test/config/fixture modification is made while authorization remains false.
