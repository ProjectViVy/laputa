# BMAD Implementation Readiness — Laputa External Agent SDK

**Historical assessment:** Wave 0 entry snapshot; superseded for current execution state by `sprint-status.yaml` (Wave 0 done, Wave 1 in progress).  
**Assessment date:** 2026-09-04  
**Verdict:** **CONCERNS**  
**Implementation authorized:** **Yes — Wave 0 gate work only**

## Decision

Planning is sufficiently concrete and the owner has accepted ADR-0015 plus explicitly authorized execution. Implementation is authorized only for the Wave 0 gate: live evidence, contract freeze, ownership isolation, and conformance-harness design. No public endpoint or SDK source may begin until the remaining P0 gate items are closed.

## Complete planning artifacts

| Artifact | State |
|---|---|
| PRD | complete planning draft |
| Architecture Spine | proposed cross-epic decisions |
| API Contract | proposed; budget/route details require live audit freeze |
| Epics/Stories | 5 epics / 22 stories, dependency-ordered |
| Conformance Inventory | defined; not implemented |
| Sprint Status | execution authorized; Wave 0 gate in progress |
| External Agent TODOLIST | active, coordinator-dispatched execution rules |
| ADR-0015 | accepted by owner; Vivy built-in preferred provider confirmed |

## Concerns and clearing stories

| ID | Severity | Concern | Clear with |
|---|---|---|---|
| C-EXT-01 | P0 | ADR-0015 is accepted, but exact public version/budget/capability semantics are not frozen for independent implementers. | E00-S02 contract freeze |
| C-EXT-02 | P0 | Exact live route/DTO/error gaps for manifest, binding and capture status must be re-audited against current Garden HEAD before implementation. | E00-S01 evidence ledger |
| C-EXT-03 | P0 | Garden and AGENT-VIVY are separate repositories with existing dirty/unrelated changes; no branch/worktree ownership plan is approved. | E00-S01 scope baseline + owner branch decision |
| C-EXT-04 | P1 | Typed SDK module ownership, initial language package/release process and API codegen policy are not decided. | E00-S02 |
| C-EXT-05 | P1 | Current Garden local auth policy is loopback-oriented; external/remote deployment posture is explicitly out of scope and must not be implicitly widened. | E01-S02 implementation design confirms local contract only |
| C-EXT-06 | P1 | No live conformance harness exists yet; existing green tests cannot prove new protocol equivalence. | E00-S03 then E01-S05/E03-S04 |

## Remaining execution gate

1. E00-S01 captures and reviews the current Garden/Vivy route and lifecycle evidence plus explicit dirty-tree ownership baseline.
2. E00-S02 freezes API budgets, exact canonical route reuse/gaps, SDK module ownership and cross-repository worktree plan.
3. E00-S03 confirms one REST/SDK/MCP conformance matrix.
4. Only after E00-S01 through E00-S04 are reviewed may public endpoint/SDK source Stories begin.

## Authorization boundary

The owner explicitly authorized execution on 2026-09-04. This authorizes Wave 0 only. It does not authorize a child to jump into E01–E04, create a public SDK module, modify runtime code, or set a Story to `done` without Coordinator verification.

## Authorization verdict

**CONCERNS** remains intentional. The owner accepted the architecture and opened Wave 0; its purpose is to prevent public REST/SDK/Vivy implementation from choosing incompatible routes, budgets, package ownership, or worktree boundaries. The next valid action is completion and review of the E00 evidence/contract gate.
