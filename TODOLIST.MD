# Garden MemoryOS Status

> **Last updated:** 2026-09-03
> **Current contract:** [ADR-0012: Laputa Markdown Clean Break](docs/architecture/0012-laputa-markdown-clean-break.md)
> **Implementation architecture:** [ADR-0013: Laputa Clean-Break Implementation Architecture](docs/architecture/0013-laputa-clean-break-implementation-architecture.md)
> **Mentle recovery authority:** [ADR-0014: Mentle Canonical Authority](docs/architecture/0014-mentle-canonical-authority-and-derived-index-recovery.md)
> **Current phase:** Waves 0–3 implementation complete under the activated GOAL; host adapters remain explicitly Deferred.

This file tracks product-level executable work only. Story-level status, dependencies and execution rules live in the [Authority & Recovery TODOLIST](docs/bmad/garden-authority-recovery-2026-09/TODOLIST.md) and [GOAL Execution Runbook](docs/bmad/garden-authority-recovery-2026-09/GOAL-EXECUTION-RUNBOOK.md).

## Active

- [x] **LAPUTA-MARKDOWN-CLEAN-BREAK-PLAN** `P0`
  ADR-0013 defines package replacement order, file-store contract, Persona review/history API, `ACTMEM.MD` tools, ContextView changes, Console migration, EvoMap candidate boundary, and the test/deletion proof matrix. The activated GOAL executed that order and its final gates passed.

- [x] **AUTHORITY-RECOVERY-GOAL-READINESS** `P0`
  The Waves 0–3 PRD, Architecture Spine, frozen API contract, 91-case test inventory, Story graph, dirty-tree policy and GOAL runbook are accepted. ADR-0014 is accepted. E00-S02, E00-S04, audited Wave 0 baseline adoption and the readiness rerun are complete; the owner activated GOAL execution.

- [x] **LAPUTA-LEGACY-JSON-DELETE** `P0`
  After the execution plan is accepted, remove JSON personality descriptors, `.laputa/sections/*.json` runtime use, section registry APIs, `Commitment`, `Preferences`, `MemoryMD`, JSON Patch, dot-path patching, generic Persona governance mutation, compatibility mappings, and fallback reads. No data migration or dual path.

- [x] **PERSONA-SEVEN-MARKDOWN-AUTHORITIES** `P0`
  Complete: `laputa/persona` serves status/documents/history, initialization/repair, full-document edits and review lifecycle from the DIVA-verified `persona/` layout (immutable `<rev>.md`/`<rev>.diff` snapshots + `log.jsonl`), exposed through the live REST contract and rendered in the Console Persona workspace.

- [x] **ACTMEM-TOOL-ONLY** `P0`
  Implement profile-level `ACTMEM.MD`; prove cross-session and restart continuity; expose one stable read/query tool and deferred maintenance tools; remove all automatic ACTMEM projection.

- [x] **WORLD-TOOL-ONLY** `P0`
  Remove every automatic WORLD projection from bootstrap, Fast Recall, Deep Recall, and ContextView. Preserve explicit tool read/write only.

- [x] **EVOMAP-CAPABILITY-BOUNDARY** `P0`
  Reconcile existing evolution/mailbox/Hub code with ADR-0012: EvoMap owns candidate/proposal review, evaluation, artifact versioning, install permission, privacy gate, mailbox, and Hub policy. Define the bounded candidate input from activity, ACTMEM reflection, and Mentle evidence. No Laputa authority or direct Skill write path.

## Active Supporting Work

- [x] **MENTLE-RECOVERABLE-EVIDENCE** `P1`
  Accepted ADR-0014 establishes canonical SQLite plus transactional `index_jobs` as the sole authority/recovery model and supersedes ADR-0011's WAL-as-authority/backend sequencing. Execute Epic 3 from the Authority & Recovery package after Gate 0 passes.

- [x] **CONSOLE-WORKSPACE-RECONCILIATION** `P1`
  Implemented from `docs/design/garden-memoryos-console-frontend-pre-design.md`: the archived Governance Map / Active Work information architecture is replaced with Persona, Memory, Evolution, and Chat Approval. Materials, evidence, reports, and operations remain supporting surfaces. No mock state.

- [ ] **HOST-ADAPTERS** `P2`
  Deferred until the new Frozen Core, tool-only WORLD/ACTMEM, and EvoMap artifact boundaries have a verified vertical slice.

## Completed Documentation Work

- [x] Archived former active ADRs that conflict with the new contract in `docs/archive/2026-08-14-laputa-clean-break/`.
- [x] Added ADR-0012 as the sole active Laputa architecture contract.
- [x] Rewrote active documentation entry points and module instructions to prohibit JSON Persona continuation and legacy compatibility paths.
- [x] Retained EvoMap mailbox and Hub transport as active, refined capability-domain decisions.
- [x] Accepted ADR-0014 and the full Waves 0–3 GOAL execution contract, including lane locks, evidence rules and per-Wave checkpoint commits.

## Execution closeout

- [x] Four audited Wave checkpoint commits created; no push performed.
- [x] Laputa, Mentle and Garden full Go suites pass; clean-break e2e passes.
- [x] Console typecheck and production build pass.
- [x] Architecture guard enforce scan reports zero target-runtime violations.
- [ ] HOST-ADAPTERS remains Deferred by the accepted scope; no host Skill installation or external adapter was added.

## Historical Status

The former status tracker is preserved unchanged at `docs/archive/2026-08-14-laputa-clean-break/status/TODOLIST-pre-clean-break.MD`. Its completion claims refer to the retired architecture and must not be used as evidence for ADR-0012 implementation.
