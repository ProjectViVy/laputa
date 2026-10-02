# DIVA Cognitive Integration — Delivery Index

[Architecture](../../specs/2026-10-02-diva-cognitive-integration-design.md) · [Shared contracts](contracts.md) · [Package checks](verification.md)


Planning authorization: the owner requested detailed architecture and scheduling after discussing Laputa-owned evolution. This is a reviewable plan, not implementation authorization. Keep this index as the sole status/dependency source. One file per Story is supplied. This index is the sole dependency/status authority; contracts.md is the sole cross-Story DTO/format authority.

Schedule basis: dependency waves, not invented calendar dates or person-day estimates. No assigned execution capacity or accepted desktop-bridge completion date exists in this session. Logical parallel eligibility does not authorize parallel agents. Within a wave, serialize shared-file edits and limit active work to review capacity.

| Story | Epic | Requirements | Immediate predecessors / supplied input | Plan state |
| --- | --- | --- | --- | --- |
| [S01](S01.md) Shared contracts and compatibility fixtures | A | R1–R11 | None; reviewed shared design | Done |
| [S02](S02.md) Mission authority and session projection | A | R1,R5,R6,R9 | S01 | Done |
| [S03](S03.md) Scoped Markdown ACTMEM | A | R2,R9,R10 | S01 | Done |
| [S04](S04.md) Optional backend and scope-safe capture | B | R2,R3,R7,R8,R9 | S01 | Done |
| [S05](S05.md) Laputa DIVA strategy library | C | R3,R4,R6,R11 | S01 | Done |
| [S06](S06.md) ViVy trusted strategy runtime adapter | D | R4,R5,R6,R11 | S01 | Done |
| [S07](S07.md) Bound cognitive service, capture and automatic wakeups | D | R1,R2,R3,R4,R7,R11 | S02, S03, S04, S05, S06 | Done |
| [S08](S08.md) DIVA cognitive controls and read models | E | R1,R5,R6,R10,R11 | S07 | Blocked: desktop bridge |
| [S09](S09.md) Cross-repository acceptance and delivery evidence | E | R1–R11 | S08 | Done |

Topological waves: {S01} → {S02, S03, S04, S05, S06} → {S07} → {S08} → {S09}.

| Wave | Delivery milestone | Exit condition |
| --- | --- | --- |
| 1 | Shared contracts | S01 strict schemas/fixtures reviewed; compatibility deltas explicit |
| 2 | Authority, activity, memory, strategy and runtime foundations | S02–S06 accepted independently; S05/S06 can use fakes while data services are built |
| 3 | Headless cognitive loop | S07 proves real domain writes, automatic triggers and recovery |
| 4 | Desktop product connection | S08 plus external DIVA-NEXT-P0 bridge evidence |
| 5 | DIVA phase-one acceptance | S09 covers complete product and failure paths |

The structural critical chain is S01 -> slowest accepted S02/S03/S04/S05/S06 -> S07 -> S08 -> S09. This is dependency depth, not a measured time forecast. The canonical Mentle mutation hooks are now mapped in S04. The external desktop-bridge contract remains an identified readiness gap for S08. Independent work need not wait for desktop completion. INOFY engine changes are not on this schedule; add a focused upstream Story only if a conformance failure proves an API gap. Garden Fast/Deep S12 remains separate.

Shared-file conflicts: S01 owns initial `garden/agentapi/contract.go`; S02 extends Persona projection after S01. S03/S04 must serialize common agentapi composition edits. S05 owns Laputa evolution source and its dependency pin; S04 module-identity changes must settle before rebasing that pin. S06 exclusively owns runtime admission/executor changes; S07 follows its accepted interface. S08 coordinates with DIVA-NEXT-P0 on desktop.ts rather than replacing that work. These are editing constraints, not invented logical DAG edges.

Global constraints for every Story: requirements R1–R11; no second runtime/Journal/personality authority; one Markdown ACTMEM authority; tool-only WORLD/ACTMEM; human-only Mission; explicit conversational DREAM tool; no default BML/backend switch; no implicit migration or publishing; documentation and commit text in English. Proposed file paths are additions, not claims they exist. Before editing any repository, read root and all applicable nested AGENTS.md/skills and preserve unrelated work.

Review focus and assigned checks: forged host identity (S01/S06), project leakage through pointers/capsules (S03/S04), lost effect acknowledgement (S04/S05/S06), Mission changes mid-run (S02/S06/S07), UI reporting a proposal as applied (S08/S09).

The plan has not run product tests. Document checks validate Story IDs, requirement coverage, dependency endpoints/cycles and required plan fields only. Downstream readiness requires accepted predecessor evidence; a plan file or wave number is not that evidence.


## Readiness and handoff

- S01–S07 and S09 are Done; evidence and the acceptance record live in acceptance.md and the merged lane history.

- S08 is explicitly Blocked on the separate DIVA-NEXT-P0 bridge API, selected revision and desktop build/smoke command. The owning bridge delivery supplies these, and the S08 planner attaches them before release.
- Root/nested AGENTS.md instruction changes are not made by this package. The owner-approved Mission/backend architecture changes must be reconciled with the seven-file/Mentle-only guards before affected code is landed. No broad instruction rewrite is authorized.
- Execution recommendation: native, one Story at a time initially. Enable concurrent work only after shared interfaces land and the owner selects delegation.
- This package is an artifact prepared for the Laputa repository, not an uploaded branch or an assertion that repositories were modified. No implementation or runtime tests were run.
