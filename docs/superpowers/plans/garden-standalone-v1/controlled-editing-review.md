# Controlled editing and native Persona increment — spec/plan review and handoff

**Date:** 2026-10-12. **Review kind:** planner self-review under Superpowers brainstorming/writing-plans and Supermanagement. This is not an independent runtime/code review.
**Runtime baseline:** `3c4cfffe33d92ad1105ad31dec259c901cc24a4e`, `feat/garden-standalone-v1`.
**Authoritative artifacts:** [specification §7](../../specs/2026-10-12-garden-standalone-v1-design.md#7-controlled-editing-and-automatic-host-projection), [index](index.md), [host source pins](../../../research/garden-host-pins-2026-10-12.md), [corrected research](../../../research/garden-standalone-v1-controlled-editing-research-2026-10-12.md).

## Scope and authorization

The latest owner continuation asks for architecture, concrete Epic/Story plans, reviews and documentation publication on the existing branch. The initial research-only restriction no longer limits this documentation delivery. This session implements no product code, installs no host, runs no product tests, and changes no user host configuration. The original GPT-6.1 Sol High native development session remains the implementation executor; no per-task agents or new development branch are requested.

The shell and Git remote read were available. Remote and local baseline matched the fixed SHA before planning. The supplemental Supermanagement architecture/coordination/story-plan resources were requested through skills.read and returned `failed to read skill resource`; its main SKILL.md and the available Superpowers skills supply the applied workflow. This does not block the concrete plans or imply those supplemental files were read.

## Specification review: findings and resolutions

| Finding | Resolution in the authoritative design |
| --- | --- |
| UI editing was only local state; new polling could attach a fresh revision to an old draft. | §7.2 fixes draft/base/profile on Acquire; viewer polling cannot mutate the editing baseline; reconnection requires reread/reacquire/reconciliation. |
| A lease at HTTP alone leaves embedded/review/init/repair bypasses. | One Persona service gate covers all retained commits; multi-kind operations preflight all affected kinds. Proof-bearing HTTP editor writes remain owner-only. |
| SaveAgentP16 without a base can commit stale generated content as latest. | D13/S14 remove the no-base method after supported caller inventory; no unsafe compatibility wrapper. Searches in this repository and available VIVY/DIVA/INOFY/pens non-test Go sources found no production `.SaveAgentP16(` callers. |
| Current GetDocument reads live Markdown with the latest history revision, so external edits can masquerade as committed data. | Snapshot compares live hash to committed history; drift blocks projection. Editor base_hash detects completed drift; no claim of CAS against external writers. |
| Plain edit state survives failed cleanup and old tabs could release a new holder. | Process-local 120-second lease, random token, generation and 30-second renewal; restart invalidates tokens; old proof never downgrades to token-free write. |
| Retry after human Save or edit-and-cancel could replay stale content. | Fresh source/hash/edit-generation/binding preview, 60-second validity, recheck on confirm and actual host pull. Pending persists; previews do not survive restart. |
| First native activation/configuration had no concrete API. | ConfigureNativeHost/native-config defines exact tuple/scope/kinds and activation pending; first preview/confirm shares retry machinery; controls remain independently explicit. |
| Switching projection mode could lose cleanup guidance. | Reject native reconfiguration while manual-file cleanup remains; file Preview/Apply reject native mode. No hidden mode transition through FilePlan. |
| Native output could be confused with file written or loaded. | §7.4 distinguishes saved/written/offered/loaded; native written is not applicable, loaded unknown absent qualified evidence. No generic observer framework. |
| OpenClaw expectedHash RPC was treated too strongly; Hermes/Codex native hooks were previously missed. | Fixed source evidence records OpenClaw RPC's independent-shell race and all exact native hook signatures. No SOUL replacement, Hermes profile split or Codex workspace narrowing. |
| Dynamic host context could silently redefine Frozen Core or priority. | Explicit separate external contribution; existing Garden/VIVY Frozen Core immutable. Hermes user-context, Codex developer context and Claude reminder semantics stated; old host history can remain. |
| “No runtime authorized/no Story ready” pointers contradicted delivered S01/S02/S03/S08 and the latest continuation. | ADR-0019, docs entry and authoritative index updated; historical records retain original snapshots and pass/fail outcomes. |

Result: no unresolved design ambiguity blocks the first S14→S15→S16 sequence. Actual host behavior remains an evidence requirement, not a documentation claim. A material requirement to replace native highest-priority SOUL or change user profile/workspace would need a separate decision; those expansions are not selected.

## Plan review: coverage and contract consistency

| Requirement | Owning task(s) | Evidence required from implementation |
| --- | --- | --- |
| U1 viewer/edit lifecycle | S14 Tasks 1, 3 | Two tabs, fixed base/draft, profile switch, expiry/reconnect, lost-save response |
| U2 shared commit gate | S14 Task 2 | Direct/embedded/review/init/repair all guarded; no-base writer removed; stale revisions denied |
| U3 durable pending/fresh retry | S15 Tasks 1–3 | Coalescing, Save/Cancel keeps pending, generation drift, confirmation/actual-pull recheck, restart/offline |
| U4 actual automatic host delivery | S16 Task 2; S17 Task 2; S10 Task 2 | Pinned native prompt receives new committed revision next supported boundary without manual file copying |
| U5 truthful status/revoke | S15 Tasks 1–3 plus adapter qualification | offered is not written/loaded; no replay after revoke; old admitted context not claimed erased |
| U6 preservation/removal | S14 Task 2; S15 Task 1; S16/S17 Task 2; S10 Tasks 1–2 | No shared-user-file overwrite; no profile/workspace migration; obsolete active writer prescriptions removed |
| U7 governance/control/memory continuity | Every new plan; S09 Tasks 1–3; S10 Task 3 | WORLD/ACTMEM excluded; no autonomous generation; independent memory operations and required native memory evidence |

Contract owners are singular: S14 owns EditLease/EditProof/EditorWrite/EditStatus/ProjectionSnapshot and WithProjectionSnapshot; S15 owns NativeBindingConfig/NativePromptRequest/PromptProjection/RetryPreview plus ConfigureNative/PreparePrompt/RetryPreview/ConfirmRetry. Installation names are ConfigureNativeHost/PrepareHostPrompt/PreviewHostRetry/ConfirmHostRetry. S16/S17/S10 consume the same transport rather than defining separate retry queues. S09 extends the same plugin packages; S12 reuses the editor/status UI instead of rebuilding it.

Each new or materially revised plan includes scope, existing/proposed files, consumed/produced contracts, ordered checkbox tasks, focused failure cases, verification commands, removal and exact residual gates. Review Focus items map to the owning tasks. Product test commands are future instructions; the planning session did not run them. Tests use existing fixtures/stdlib and actual isolated-host observation, not a new test framework or scheduler-dependent filesystem “proof.”

## Dependency review

The index is the single dependency source: **17 nodes, 23 immediate edges, 9 topological waves**. The new path is S03→S14; S08+S14→S15; S15→S16/S17/S10; S16+S17→S09. S12 consumes S09/S10 plus its existing S07/S11 prerequisites. S13 follows S12. Other original edges and independent S05 backend delivery remain.

S14/S15/S16 are sequenced for first useful automatic-update delivery. S17 and S10 are logically independent after S15 but shared files and the preserved single executor call for sequential edits, not parallel agents. Runtime availability gates do not become artificial DAG nodes. S09's remaining memory mapping does not gate Persona qualification.

Documentation-only validation **PASS**: 18 owned Markdown files; 17 nodes / 23 edges / 9 waves; 173 relative links/anchors; 25 explicitly listed existing increment source paths; fixed research source targets resolved locally. No source test/build was run. Checks covered the table-derived graph against Mermaid, known IDs, self-edges, cycles, redundant transitive edges and displayed waves; relative Markdown targets/anchors; required plan sections; classified existing/proposed files; doc-only diff and whitespace. The publication handoff reports the actual resulting commit. No runtime PASS is created by this review.

## Remaining blockers and exact first handoff

| Item | Remaining gate | Effect |
| --- | --- | --- |
| S14 | No product/prerequisite blocker; refresh source/caller inventory in execution checkout | Ready first; implement one local lease + shared gate + UI slice |
| S15 | S14 methods must exist and pass its scoped checks | Then implement native coordinator/owner UI; S08 already exists |
| S16 | S15; actual OpenClaw 2026.10.2 embedded runner and observable prompt | Source contract is ready; no host acceptance until real isolated evidence |
| S17 | S15; actual Hermes 0.21.7 persistent CLI/gateway request composition | Same-profile additive user-context only; no migration/frozen-SOUL replacement |
| S10 | S15; actual Codex 0.162.1 / Claude 2.1.296 plugin loader and prompt evidence | Shared bridge plan executable after S15; each host's runtime gate remains, especially closed Claude binary |
| S09 | S16/S17; exact memory provider and explicit-selected-capture mappings in Task 1 | Memory code remains blocked on named signatures; do not substitute Persona hooks or full transcript ingestion |
| Broader standalone | Q2/Q3, attachment/package/backend/cross-repo gates already in index | Not resolved or newly authorized by this increment |

Original executor handoff:

1. Inspect actual worktree/branch/status; fetch the published `feat/garden-standalone-v1` docs commit and integrate without overwriting concurrent work. No replacement branch or force push.
2. Read spec §7, this review, index and [S14](S14.md); execute S14 in the original GPT-6.1 Sol High native session using executing-plans.
3. After S14 evidence, execute [S15](S15.md), then [S16](S16.md). Preserve necessary S08 shared-file safety throughout.
4. Continue [S17](S17.md) and [S10](S10.md); finish S09's exact memory contract gate before its code. Record unsupported paths precisely rather than removing automatic updating from scope.
5. Update only actual Story evidence/status after implementation. This plan/review is not a runtime completion report or authorization to configure the user's real hosts.


## Independent implementation pre-flight review

Fresh-context Astra review of docs commit `4a8f713` and actual baseline source found no Critical or product/architecture blocker to S14→S15→S16. The executor accepts its narrow corrections: intercept dirty profile selection before route remount, fence delayed editor attempts; route public sync disable through the host coordinator to invalidate previews; enforce persona_get Operations independently of Recall; implement the missing AccessPage host/control flow; use one whole-document USER editor and commit explicitly inspected drift as a new revision. These are included in S14/S15 and execution ledgers, without a new owner review gate or routine parameter question. Native source pins were independently checked; actual runtime qualification still remains mandatory.
