# Laputa External Agent SDK

> Historical REST-first execution baseline. [ADR-0016](../../architecture/0016-laputa-embeddable-modular-monolith.md) now governs the library + application split and refines Vivy's transport decision; the status and default transport below describe the prior plan, not the current modular-monolith gate.

BMAD project-sized planning package for the post-clean-break external Agent access plane.

**Status:** Wave 0 complete; Wave 1 in progress (per `sprint-status.yaml`)  
**Date:** 2026-09-04  
**Scope:** `laputa-agent/1` contract, canonical REST profile, thin MCP equivalence, public typed SDK, and AGENT-VIVY first-party lifecycle adapter  
**Precondition:** Garden Authority & Recovery Convergence Waves 0–3 are complete; host adapters were explicitly Deferred.

## Outcome

External Agents can safely consume Laputa through a versioned, capability-discovered API without gaining raw authority/storage access or changing Laputa's authority boundaries.

```text
External Agent / Host Runtime
              │
              ▼
     laputa-agent/1 contract
       ├─ canonical REST profile
       ├─ typed SDKs
       ├─ thin MCP adapter
       └─ native lifecycle adapters
              │
              ▼
Garden domain services → Laputa authority / Mentle canonical facade
```

AGENT-VIVY is the first conformance host, not the SDK itself.

## Artifacts

| Artifact | Purpose |
|---|---|
| [PRD](prd.md) | Product requirements, scope, non-goals, success signals |
| [Architecture Spine](ARCHITECTURE-SPINE.md) | Cross-epic authority, lifecycle, versioning and failure decisions |
| [API Contract](api-contract.md) | Proposed `laputa-agent/1` manifest, resources, DTOs and errors |
| [Test Inventory](test-inventory.md) | Conformance acceptance matrix |
| [Epic 0](epics/epic-00-contract-and-gates.md) | Baseline, ADR acceptance, contract and test gates |
| [Epic 1](epics/epic-01-canonical-access-plane.md) | REST discovery/binding/capture canonical surface |
| [Epic 2](epics/epic-02-sdk-and-vivy-lifecycle.md) | Public SDK and AGENT-VIVY MemoryPort lifecycle bridge |
| [Epic 3](epics/epic-03-mcp-conformance-and-tools.md) | Thin MCP mapping and explicit tools |
| [Epic 4](epics/epic-04-reliability-and-release.md) | Reliability, conformance and release gate |
| [Implementation Readiness](implementation-readiness.md) | Honest readiness verdict and blockers |
| [Sprint Status](sprint-status.yaml) | Dependency order, phase and authorization state |
| [TODOLIST](TODOLIST.md) | Copyable external-subagent assignments |

## Hard boundaries

- Wave 0 is closed. Wave 1–4 execution is authorized within ADR-0015/BMAD scope; story dependencies and coordinator verification still apply.
- The `feat/laputa-agent-sdk` worktree contains uncommitted Wave 1 implementation. Story records here do not mean that code is committed to this main branch or available on the remote.
- Laputa remains Persona/ACTMEM authority; Mentle canonical SQLite remains the sole memory authority.
- REST is canonical; MCP is a thin adapter; SDK is typed transport plus lifecycle helper.
- WORLD, ACTMEM, history, raw authority paths and unbounded tails remain explicit-only.
- AGENT-VIVY owns its existing Session/Run/Journal/Policy/Tool lifecycle; no second Agent runtime is created.
- No compatibility aliases, fallback, dual authority, transparent proxy requirement, or optional vector backend work is in scope.

## Evidence and reference projects

- Binding: [ADR-0012](../../architecture/0012-laputa-markdown-clean-break.md), [ADR-0013](../../architecture/0013-laputa-clean-break-implementation-architecture.md), [ADR-0014](../../architecture/0014-mentle-canonical-authority-and-derived-index-recovery.md), accepted [ADR-0015](../../architecture/0015-laputa-external-agent-contract.md).
- First-party host: `C:/Users/Administrator/Desktop/morediva/diva-go/agent-vivy`.
- TencentDB-Agent-Memory: lifecycle/host-adapter/pending-write reference only — https://github.com/TencentCloud/TencentDB-Agent-Memory
- memsearch: source-first/incremental/progressive-disclosure reference only — https://github.com/zilliztech/memsearch
- BMAD planning references: https://docs.bmad-method.org/plan/choose-a-planning-path/ ; https://docs.bmad-method.org/plan/design-ux-and-architecture/ ; https://docs.bmad-method.org/plan/break-work-into-stories-and-track-it/

The BMAD URLs could not be fetched by the current web extractor (private/internal-address block); they are recorded as source references, while the local accepted BMAD runbook governs this planning package.
