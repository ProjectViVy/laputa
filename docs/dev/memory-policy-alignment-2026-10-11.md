# Memory policy alignment — 2026-10-11

**Scope:** documentation reconciliation only; selected [ADR-0018](../architecture/0018-vivy-unified-context-and-recap-policy.md) is accepted owner direction, with engineering and implementation pending. No runtime, tests, scripts, migrations, canonical data, backend defaults or issue bodies are changed here. The lead publishes the documentation branch and draft PR; no merge is authorized.

**Workspace:** clean mounted `/workspace/laputa` on `work`, isolated documentation worktree `/workspace/doc-alignment/laputa`, branch `docs/memory-policy-alignment-20261011`.
**Refreshed upstream:** `origin/main` = `4b2bec2cc2ab3374b7ec1ab8d448e612c1e4db24`, including merged [Laputa PR #5](https://github.com/ProjectViVy/laputa/pull/5). The mounted checkout and unrelated work are preserved. Human-configured commit identity is retained.

## Current contract and historical boundaries

VIVY [issue #58](https://github.com/ProjectViVy/agent-vivy/issues/58) owns the unified context foundation. LIFE is the first default-enabled pilot and coding is excluded from phase 1. Persistent same-subject sessions share one free-form AI-organized Recap. Existing runtime/context/Journal/policy remains the integration path. [Issue #57](https://github.com/ProjectViVy/agent-vivy/issues/57) role-only workers gain no automatic persona, Recap, history or permissions.

Backend providers own concrete CRUD/search, storage, indexing and returned content; VIVY owns authorized scope, admission and budgets. BML is the first validation backend; the owner's supported Mentle path through DIVA must use the same capability-aware interface. This is no default switch, dual write or data migration.

Revision-aware consolidation organizes the previous Recap and retiring raw segment, commits the next revision, then unloads raw CONTEXT. Raw evidence remains retained. Human edits and concurrent session updates must conflict safely rather than overwrite newer content. Active Recap delete clears the head, advances revision and invalidates active projections; raw evidence, older revisions, archives and backend data stay retained unless separately and explicitly targeted through authorized tools.

A dedicated UI needs full human inspect/edit/delete, with reviewed flows and wireframes still open. Agent ACTMEM inspect/update/delete tools must be normally hidden, discovered and explicitly activated when needed; the exact mechanism is unresolved. Human control actions and Garden `agentapi` methods do not prove VIVY model-visible ACTMEM tools exist today.

WORLD remains tool-only. Frozen Core retains the existing six bounded Persona projections and recorded human-controlled Mission v2 slot; no new authority is introduced. It remains immutable for an admitted session, with later source revisions observed through existing session/admission rules. Guide “must not be forgotten” requests through the existing human Laputa governance workflow; ordinary Recap edits create no permanent pin. Summary/consolidation paths cannot automatically write Mission, Dream or frozen Persona content.

Earlier Pulse/Recap/Work grammar, fixed caps/timers, Work reconciliation and ACTMEM exclusion results remain recorded implementation baseline. This change adds supersession notices rather than deleting consumers or rewriting pass/fail evidence. Existing API/SDK/MCP/desktop contracts remain unchanged until a reviewed compatibility plan and implementation proof exist. The dated logs and archive snapshots are unchanged.

## Supporting integration evidence and limits

The lead reverified merged [VIVY PR #46](https://github.com/ProjectViVy/agent-vivy/pull/46) (`59a673ac`) and [PR #48](https://github.com/ProjectViVy/agent-vivy/pull/48) (`e67e6ff`); `BML-FIRST-INIT-CANCEL` was fixed by PR #48. [CI run 38095646661](https://github.com/ProjectViVy/agent-vivy/actions/runs/38095646661) for `ce33bc8776b9f52ab13bed5f2cdaf8ebad01f666` completed all four jobs successfully, including supported DIVA memory integration. This is the developer regression baseline, not proof that ADR-0018 has shipped.

Original local [real-backend verification](../logs/2026-10-09-diva-memory-real-backend/verification.md), [environment follow-up](../logs/2026-10-09-memory-test-environment/verification.md) and [capture receipt recovery](../logs/2026-10-09-capture-receipt-recovery/verification.md) retain their actual results and limitations. An initial failing or skipped baseline is not rewritten by later successful tests. The [log index](../logs/README.md) distinguishes those snapshots from the selected policy.

The DIVA developer chain preserves:

- [v3.0.0 S08–S11 closeout](https://github.com/ProjectViVy/agent-diva/blob/0a64d1ce152440f1f63d4224a155c4616d2aff5b/docs/logs/2026-10-memory-loop-chain/v3.0.0-s08-s11-closeout/summary.md): restart recall and six negative empty/disabled controls; correction/delete followed by restart and three old-run replays.
- [v3.1.0 V19 boundary](https://github.com/ProjectViVy/agent-diva/blob/0a64d1ce152440f1f63d4224a155c4616d2aff5b/docs/logs/2026-10-memory-loop-chain/v3.1.0-s08-v19-boundary/summary.md): prior no-injection baseline, human UI and absent agent ACTMEM read tools. The no-injection decision is now historical for the selected policy.
- [C01 terminal before capture](https://github.com/ProjectViVy/agent-diva/blob/0a64d1ce152440f1f63d4224a155c4616d2aff5b/docs/logs/2026-10-memory-loop-chain/v2.3.0-s10-c01-terminal-before-capture/summary.md), [C02 capture before acknowledgement](https://github.com/ProjectViVy/agent-diva/blob/0a64d1ce152440f1f63d4224a155c4616d2aff5b/docs/logs/2026-10-memory-loop-chain/v2.2.0-s10-c02-capture-before-ack/summary.md), [C03 canonical before caller receipt](https://github.com/ProjectViVy/agent-diva/blob/0a64d1ce152440f1f63d4224a155c4616d2aff5b/docs/logs/2026-10-memory-loop-chain/v2.4.0-s10-c03-canonical-before-caller-receipt/summary.md), [C04 partial effect batch](https://github.com/ProjectViVy/agent-diva/blob/0a64d1ce152440f1f63d4224a155c4616d2aff5b/docs/logs/2026-10-memory-loop-chain/v2.5.0-s10-c04-partial-effect-batch/summary.md), [C05 effects before watermark](https://github.com/ProjectViVy/agent-diva/blob/0a64d1ce152440f1f63d4224a155c4616d2aff5b/docs/logs/2026-10-memory-loop-chain/v2.6.0-s10-c05-effects-before-watermark/summary.md), and [C06 canonical before index](https://github.com/ProjectViVy/agent-diva/blob/0a64d1ce152440f1f63d4224a155c4616d2aff5b/docs/logs/2026-10-memory-loop-chain/v2.7.0-s10-c06-canonical-before-index/summary.md): the actual scoped crash-cut/replay proofs.

These pointers preserve real proofs and do not close installed-product Windows, real-model, unknown/partial recovery, or workspace acceptance matrices. Historical names such as Work/capsules are not silently reinterpreted as new Recap acceptance.

## Engineering decisions still open

1. Free-form Recap storage/revision/provenance representation and compatibility for current Work reconcile, captured Pulse/Recap, folds, evolution, REST/MCP/SDK and desktop consumers. Do not introduce a second authority to make compatibility easy.
2. Safe admission/disclosure of subject-level Recap across authorized workspaces and role-only workers; subject continuity is not an unrestricted memory grant.
3. Concurrent consolidation and human-edit conflict resolution, commit-before-retire recovery, active projection/cache invalidation, and the clear/reconsolidation lifecycle. Older revisions remain recoverable; targeted clear does not imply erasure or automatic restore.
4. Deferred inspect/update/delete discovery and explicit activation, with no always-on schemas or autonomous maintenance grant.
5. Reviewed full Recap UI flows/wireframes, experimental thresholds/budgets, provider capability tests and remaining installed-product matrices. Fixed historical limits do not settle these experiments.

## Verification in this environment

- `git diff --check`: passed.
- Local Markdown file-link scan: all changed/new documents resolve; scoped notices point to ADR-0018. Archive links are labelled historical. The full active-doc scan excludes fenced code and archived snapshots.
- README/AGENTS policy scan: current instructions preserve WORLD exclusion while explicitly permitting selected-policy ACTMEM admission; remaining absolute exclusion claims in historical plans/ADRs are covered by supersession notices. No source or test is changed.
- All four prescribed Go commands were attempted: `GOSUMDB=off go test ./...` in Laputa, Mentle and Garden, and `GOSUMDB=off go test -tags=e2e ./e2e/...` in Garden. Each stopped immediately with `Go: Unknown option: test`: `/usr/bin/go` is not the Go language toolchain. No runtime test passed in this documentation workspace.
- Prescribed cognitive planning command `python3 scripts/check_plan.py` was attempted from its package directory and failed because that checker is absent in this snapshot. It is not replaced or added in this docs-only change.

## Changed/current and historical path inventory

Current policy, instruction and evidence entry points are edited in English. `TODOLIST.MD` adds current unfinished work while preserving completed baseline text. Historical contracts/plans receive scoped notices; dated result bodies and archives are unchanged. The legacy translated overview/JSON architecture also get notices and corrected historical link targets.

This reconciliation changes 71 Markdown documents: 20 current entry points and 51 historical/baseline notices or references.

### Current entry points

- [AGENTS.md](../../AGENTS.md)
- [README.md](../../README.md)
- [TODOLIST.MD](../../TODOLIST.MD)
- [docs/AGENTS.md](../AGENTS.md)
- [docs/README.md](../README.md)
- [docs/architecture/0018-vivy-unified-context-and-recap-policy.md](../architecture/0018-vivy-unified-context-and-recap-policy.md)
- [docs/architecture/AGENTS.md](../architecture/AGENTS.md)
- [docs/dev/memory-policy-alignment-2026-10-11.md](memory-policy-alignment-2026-10-11.md)
- [docs/logs/README.md](../logs/README.md)
- [garden/AGENTS.md](../../garden/AGENTS.md)
- [garden/config/AGENTS.md](../../garden/config/AGENTS.md)
- [garden/e2e/AGENTS.md](../../garden/e2e/AGENTS.md)
- [garden/fixtures/AGENTS.md](../../garden/fixtures/AGENTS.md)
- [garden/internal/AGENTS.md](../../garden/internal/AGENTS.md)
- [garden/internal/rag/AGENTS.md](../../garden/internal/rag/AGENTS.md)
- [garden/internal/server/AGENTS.md](../../garden/internal/server/AGENTS.md)
- [laputa/AGENTS.md](../../laputa/AGENTS.md)
- [laputa/README.md](../../laputa/README.md)
- [laputa/governance/AGENTS.md](../../laputa/governance/AGENTS.md)
- [laputa/governance/wakeup/AGENTS.md](../../laputa/governance/wakeup/AGENTS.md)

### Historical/baseline documents with scoped notice or reference changes

- [README_CN.md](../../README_CN.md)
- [docs/architecture/0012-laputa-markdown-clean-break.md](../architecture/0012-laputa-markdown-clean-break.md)
- [docs/architecture/0013-laputa-clean-break-implementation-architecture.md](../architecture/0013-laputa-clean-break-implementation-architecture.md)
- [docs/architecture/0015-laputa-external-agent-contract.md](../architecture/0015-laputa-external-agent-contract.md)
- [docs/architecture/0016-laputa-embeddable-modular-monolith.md](../architecture/0016-laputa-embeddable-modular-monolith.md)
- [docs/architecture/0017-diva-cognitive-contract-deltas.md](../architecture/0017-diva-cognitive-contract-deltas.md)
- [docs/bmad/garden-authority-recovery-2026-09/ARCHITECTURE-SPINE.md](../bmad/garden-authority-recovery-2026-09/ARCHITECTURE-SPINE.md)
- [docs/bmad/garden-authority-recovery-2026-09/GOAL-EXECUTION-RUNBOOK.md](../bmad/garden-authority-recovery-2026-09/GOAL-EXECUTION-RUNBOOK.md)
- [docs/bmad/garden-authority-recovery-2026-09/README.md](../bmad/garden-authority-recovery-2026-09/README.md)
- [docs/bmad/garden-authority-recovery-2026-09/TODOLIST.md](../bmad/garden-authority-recovery-2026-09/TODOLIST.md)
- [docs/bmad/garden-authority-recovery-2026-09/api-contract.md](../bmad/garden-authority-recovery-2026-09/api-contract.md)
- [docs/bmad/garden-authority-recovery-2026-09/baseline-inventory.md](../bmad/garden-authority-recovery-2026-09/baseline-inventory.md)
- [docs/bmad/garden-authority-recovery-2026-09/epics/epic-00-baseline-and-gates.md](../bmad/garden-authority-recovery-2026-09/epics/epic-00-baseline-and-gates.md)
- [docs/bmad/garden-authority-recovery-2026-09/epics/epic-02-actmem-and-clean-break.md](../bmad/garden-authority-recovery-2026-09/epics/epic-02-actmem-and-clean-break.md)
- [docs/bmad/garden-authority-recovery-2026-09/epics/epic-04-adapters-console-release.md](../bmad/garden-authority-recovery-2026-09/epics/epic-04-adapters-console-release.md)
- [docs/bmad/garden-authority-recovery-2026-09/implementation-readiness.md](../bmad/garden-authority-recovery-2026-09/implementation-readiness.md)
- [docs/bmad/garden-authority-recovery-2026-09/prd.md](../bmad/garden-authority-recovery-2026-09/prd.md)
- [docs/bmad/garden-authority-recovery-2026-09/test-inventories.md](../bmad/garden-authority-recovery-2026-09/test-inventories.md)
- [docs/bmad/laputa-external-agent-sdk-2026-09/ARCHITECTURE-SPINE.md](../bmad/laputa-external-agent-sdk-2026-09/ARCHITECTURE-SPINE.md)
- [docs/bmad/laputa-external-agent-sdk-2026-09/README.md](../bmad/laputa-external-agent-sdk-2026-09/README.md)
- [docs/bmad/laputa-external-agent-sdk-2026-09/TODOLIST.md](../bmad/laputa-external-agent-sdk-2026-09/TODOLIST.md)
- [docs/bmad/laputa-external-agent-sdk-2026-09/api-contract.md](../bmad/laputa-external-agent-sdk-2026-09/api-contract.md)
- [docs/bmad/laputa-external-agent-sdk-2026-09/epics/epic-00-contract-and-gates.md](../bmad/laputa-external-agent-sdk-2026-09/epics/epic-00-contract-and-gates.md)
- [docs/bmad/laputa-external-agent-sdk-2026-09/epics/epic-01-canonical-access-plane.md](../bmad/laputa-external-agent-sdk-2026-09/epics/epic-01-canonical-access-plane.md)
- [docs/bmad/laputa-external-agent-sdk-2026-09/epics/epic-03-mcp-conformance-and-tools.md](../bmad/laputa-external-agent-sdk-2026-09/epics/epic-03-mcp-conformance-and-tools.md)
- [docs/bmad/laputa-external-agent-sdk-2026-09/epics/epic-04-reliability-and-release.md](../bmad/laputa-external-agent-sdk-2026-09/epics/epic-04-reliability-and-release.md)
- [docs/bmad/laputa-external-agent-sdk-2026-09/evidence-ledger.md](../bmad/laputa-external-agent-sdk-2026-09/evidence-ledger.md)
- [docs/bmad/laputa-external-agent-sdk-2026-09/prd.md](../bmad/laputa-external-agent-sdk-2026-09/prd.md)
- [docs/bmad/laputa-external-agent-sdk-2026-09/test-inventory.md](../bmad/laputa-external-agent-sdk-2026-09/test-inventory.md)
- [docs/bmad/laputa-modular-monolith-2026-09/README.md](../bmad/laputa-modular-monolith-2026-09/README.md)
- [docs/bmad/laputa-modular-monolith-2026-09/conformance.md](../bmad/laputa-modular-monolith-2026-09/conformance.md)
- [docs/bmad/laputa-modular-monolith-2026-09/vivy-handoff.md](../bmad/laputa-modular-monolith-2026-09/vivy-handoff.md)
- [docs/design/garden-memoryos-console-frontend-pre-design.md](../design/garden-memoryos-console-frontend-pre-design.md)
- [docs/design/garden-openclaw-memory-provider.md](../design/garden-openclaw-memory-provider.md)
- [docs/superpowers/plans/2026-10-02-diva-cognitive/S01.md](../superpowers/plans/2026-10-02-diva-cognitive/S01.md)
- [docs/superpowers/plans/2026-10-02-diva-cognitive/S02.md](../superpowers/plans/2026-10-02-diva-cognitive/S02.md)
- [docs/superpowers/plans/2026-10-02-diva-cognitive/S03.md](../superpowers/plans/2026-10-02-diva-cognitive/S03.md)
- [docs/superpowers/plans/2026-10-02-diva-cognitive/S04.md](../superpowers/plans/2026-10-02-diva-cognitive/S04.md)
- [docs/superpowers/plans/2026-10-02-diva-cognitive/S05.md](../superpowers/plans/2026-10-02-diva-cognitive/S05.md)
- [docs/superpowers/plans/2026-10-02-diva-cognitive/S06.md](../superpowers/plans/2026-10-02-diva-cognitive/S06.md)
- [docs/superpowers/plans/2026-10-02-diva-cognitive/S07.md](../superpowers/plans/2026-10-02-diva-cognitive/S07.md)
- [docs/superpowers/plans/2026-10-02-diva-cognitive/S08.md](../superpowers/plans/2026-10-02-diva-cognitive/S08.md)
- [docs/superpowers/plans/2026-10-02-diva-cognitive/S09.md](../superpowers/plans/2026-10-02-diva-cognitive/S09.md)
- [docs/superpowers/plans/2026-10-02-diva-cognitive/acceptance.md](../superpowers/plans/2026-10-02-diva-cognitive/acceptance.md)
- [docs/superpowers/plans/2026-10-02-diva-cognitive/contracts.md](../superpowers/plans/2026-10-02-diva-cognitive/contracts.md)
- [docs/superpowers/plans/2026-10-02-diva-cognitive/index.md](../superpowers/plans/2026-10-02-diva-cognitive/index.md)
- [docs/superpowers/plans/2026-10-02-diva-cognitive/verification.md](../superpowers/plans/2026-10-02-diva-cognitive/verification.md)
- [docs/superpowers/specs/2026-10-02-diva-cognitive-integration-design.md](../superpowers/specs/2026-10-02-diva-cognitive-integration-design.md)
- [examples/vivy-embed-smoke/README.md](../../examples/vivy-embed-smoke/README.md)
- [garden/README.md](../../garden/README.md)
- [laputa/ARCHITECTURE.md](../../laputa/ARCHITECTURE.md)
