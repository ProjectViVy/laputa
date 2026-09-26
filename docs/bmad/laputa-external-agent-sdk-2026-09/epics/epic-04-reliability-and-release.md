# Epic 4 — Reliability, Conformance and Release Gate

**Outcome:** adapters are operationally safe under restart/degradation and the protocol can be claimed as supported only with evidence.

**Requirements:** all FR; NFR-01..07  
**Architecture:** EA-004..011  
**Dependencies:** Epics 1–3

## Story E04-S01 — Fault and restart matrix

**Acceptance criteria**

- Inject timeout, 429, 5xx, service restart, host restart, duplicate capture and changed-payload conflict.
- Verify primary host Run behavior, queue persistence, receipt state, Frozen Core immutability and derived-index pending/recovery.
- Authorization failures remain fail-closed and are not retried.

## Story E04-S02 — Security and boundary audit

**Acceptance criteria**

- Scan public SDK and transport DTOs for raw file paths, DB handles, internal imports, secrets and authority content leaks.
- Verify token/principal/profile binding and actor-header non-escalation.
- Verify no automatic context schema/path includes WORLD/ACTMEM/history/raw source.

## Story E04-S03 — Multi-host conformance sample

**Acceptance criteria**

- AGENT-VIVY remains required first-party conformance client.
- A second minimal black-box SDK/client sample proves no Vivy dependency is necessary.
- Samples do not create a new generic Agent runtime or copy TencentDB proxy semantics.

## Story E04-S04 — Release evidence and support contract

**Acceptance criteria**

- Publish exact supported manifest/version/capability list, limits, deployment/auth prerequisites and degradation behavior.
- Publish adapter guide, operator capture-pending diagnostics and rollback boundaries.
- Run full conformance matrix, affected Garden/Vivy suites, static API/import scan and `git diff --check`.
- Mark protocol supported only after owner acceptance; otherwise retain experimental/planned status.
