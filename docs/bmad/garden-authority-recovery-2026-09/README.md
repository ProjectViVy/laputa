# Garden Authority & Recovery Convergence

BMAD planning package for the next Garden/Laputa/Mentle batch.

**Status:** Waves 0–3 execution complete; implementation readiness = **PASS**
**Code changes in this batch:** audited clean-break implementation; host adapters remain Deferred

The owner activated `运行 GOAL` on 2026-09-03. Waves 0–3 are complete and the final gate evidence is recorded in `sprint-status.yaml`, `implementation-readiness.md`, and `implementation/`; host adapters remain Deferred by scope.

## Artifacts

| Artifact | Purpose |
|---|---|
| [PRD](prd.md) | What/why, requirements and success signals |
| [Architecture Spine](ARCHITECTURE-SPINE.md) | Shared cross-epic technical decisions |
| [Baseline Inventory](baseline-inventory.md) | Dirty-tree ownership and future Story diff boundaries |
| [Frozen API Contract](api-contract.md) | Persona/ACTMEM/IndexHealth/Memory routes, DTOs, errors and principals |
| [Implementation Readiness](implementation-readiness.md) | BMAD readiness gate and blocking concerns |
| [Sprint Status](sprint-status.yaml) | Ordered tracking state and implementation authorization |
| [Manual Assignment TODOLIST](TODOLIST.md) | 可直接复制派工的 Wave、Story、依赖、允许范围和验收清单 |
| [GOAL Execution Runbook](GOAL-EXECUTION-RUNBOOK.md) | Durable GOAL state machine, lane scheduling, locks, dirty-tree adoption, evidence and Wave commits |
| [Test Inventories](test-inventories.md) | Persona/ACTMEM/Context/Mentle executable acceptance matrix |
| [Pre-GOAL record](implementation/pre-goal-execution-contract.md) | Historical owner decisions and readiness probe evidence |
| [Operator Runbook](operator-runbook.md) | Canonical backup, staged derived-index rebuild, live health and rollback |
| [ADR-0014](../../architecture/0014-mentle-canonical-authority-and-derived-index-recovery.md) | Accepted canonical-only Mentle recovery decision |
| [Epic 0](epics/epic-00-baseline-and-gates.md) | Baseline, API and deletion gates |
| [Epic 1](epics/epic-01-persona-authority.md) | Persona parity and write safety |
| [Epic 2](epics/epic-02-actmem-and-clean-break.md) | ACTMEM, Frozen Core and runtime clean break |
| [Epic 3](epics/epic-03-mentle-recovery.md) | Canonical memory and recoverable indexes |
| [Epic 4](epics/epic-04-adapters-console-release.md) | REST/MCP/Console and final release gate |

## Execution waves

| Wave | Work | Parallelism | Checkpoint |
|---|---|---|---|
| 0 | Epic 0 | Serial | Green baseline + stable contracts + accepted recovery correction |
| 1 | Epic 1 and Epic 3 | Parallel, separate ownership | Persona parity review; Mentle fault/recovery review |
| 2 | Epic 2 | Mostly serial cutover | Blocking deletion scan + clean-break vertical slice |
| 3 | Epic 4 | Adapter/UI streams may parallelize after API cutover | Full release gate |

## Hard boundaries

- No Redis/Qdrant/Chroma/LanceDB work in this batch.
- No JSON compatibility, migration, fallback, alias or dual-write path.
- No new Persona write consumer before write-class/auth parity is complete.
- No repair operation may rename or reconstruct canonical SQLite.
- No WORLD/ACTMEM background load or automatic ContextView projection.
- No implementation begins until the project owner explicitly authorizes it.
- At most two workers may run concurrently, and only with disjoint declared paths and locks.
- Only the GOAL Coordinator updates project status or creates the audited Wave checkpoint commits; no worker pushes.

## Official BMAD references used

- https://docs.bmad-method.org/plan/choose-a-planning-path/
- https://docs.bmad-method.org/plan/define-requirements-and-a-specification/
- https://docs.bmad-method.org/plan/design-ux-and-architecture/
- https://docs.bmad-method.org/plan/break-work-into-stories-and-track-it/
