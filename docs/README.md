# Garden Documentation

> **Selected context policy:** [ADR-0018](./architecture/0018-vivy-unified-context-and-recap-policy.md), accepted owner direction with implementation pending.
> **Current evidence and open gates:** [2026-10-11 alignment inventory](./dev/memory-policy-alignment-2026-10-11.md).
> **Preserved authority architecture:** [ADR-0012: Laputa Markdown Clean Break](./architecture/0012-laputa-markdown-clean-break.md)
> **Status:** architecture and GOAL execution contract accepted; implementation readiness = PASS; Waves 0–3 complete

This directory is the canonical entry point for Garden MemoryOS documentation.

**Standalone Garden enhancement:** [Specification](./superpowers/specs/2026-10-12-garden-standalone-v1-design.md), [ADR-0019](./architecture/0019-garden-standalone-v1.md), [authoritative 17-Story index](./superpowers/plans/garden-standalone-v1/index.md), and [increment review](./superpowers/plans/garden-standalone-v1/controlled-editing-review.md). S01/S02/S03 and conservative S08 are implemented; automatic host updating remains pending. The reviewed continuation starts S14 → S15 → S16 in the original native executor. Q2/Q3 remain open; the unrelated historical completion status above does not certify standalone delivery.

## Read First

1. [ADR-0018: VIVY Unified Context and Subject Recap Policy](./architecture/0018-vivy-unified-context-and-recap-policy.md) — current policy scope, targeted deletion, deferred tools, lifecycle and backend boundaries.
2. [ADR-0012: Laputa Markdown Clean Break](./architecture/0012-laputa-markdown-clean-break.md) — authority files, ACTMEM, context lanes, EvoMap boundary, deletions, and implementation gate.
3. [ADR-0013: Laputa Clean-Break Implementation Architecture](./architecture/0013-laputa-clean-break-implementation-architecture.md) — replacement packages, deletion order, API boundary, Console workspaces, and acceptance matrix.
4. [ADR-0014: Mentle Canonical Authority](./architecture/0014-mentle-canonical-authority-and-derived-index-recovery.md) — canonical SQLite, transactional index outbox, disposable indexes and safe rebuild.
5. [ADR-0016: Embeddable Laputa Libraries and Modular Monolith](./architecture/0016-laputa-embeddable-modular-monolith.md) — three reusable libraries plus one application; refines Vivy transport decision.
6. [Laputa modular-monolith execution baseline](./bmad/laputa-modular-monolith-2026-09/) — ownership, dependency waves and conformance gates under ADR-0016.
   [Vivy in-process handoff](./bmad/laputa-modular-monolith-2026-09/vivy-handoff.md) — importable local Go API, host lifecycle mapping, offline modes and outstanding Vivy-side work.
7. [ADR-0015: Laputa External Agent Contract](./architecture/0015-laputa-external-agent-contract.md) — external `laputa-agent/1` wire profile, refined by ADR-0016.
8. [Laputa External Agent SDK package](./bmad/laputa-external-agent-sdk-2026-09/) — historical REST-first execution baseline; replan under ADR-0016 before further integration.
9. [Garden Authority & Recovery planning package](./bmad/garden-authority-recovery-2026-09/) — completed clean-break PRD, API contract, Epics, test inventory, TODOLIST and GOAL execution runbook.
10. [Console Frontend Pre-Design](./design/garden-memoryos-console-frontend-pre-design.md) — four-workspace information architecture, data/state rules, UI migration, and acceptance criteria.
11. [ADR-0007: EvoMap Mailbox](./architecture/0007-evomap-mailbox.md) — durable mailbox and privacy-state semantics, constrained by ADR-0012.
12. [ADR-0010: EvoMap Hub Transport](./architecture/0010-evomap-hub-transport-provider.md) — Hub transport and publication gates, constrained by ADR-0012.
13. [ADR-0006: Semantic Ingestion](./architecture/0006-semantic-ingestion-and-obsidian-adapter.md) — raw-first evidence boundary.
14. [ADR-0011: Recoverable Indexing and Evidence Contract](./architecture/0011-recoverable-indexing-and-evidence-contract.md) — proposed evidence/recovery work, refined by ADR-0014.

## Active Documents

| Document | Purpose | Status |
| --- | --- | --- |
| [ADR-0018: VIVY Unified Context and Subject Recap Policy](./architecture/0018-vivy-unified-context-and-recap-policy.md) | Selected VIVY foundation; LIFE pilot; one subject Recap, human UI, CAS/conflicts, deferred tools, scope/budget ownership | accepted direction; implementation pending |
| [ADR-0017: Cognitive Contract Deltas](./architecture/0017-diva-cognitive-contract-deltas.md) | Existing Mission/Frozen Core v2, backend and scoped ACTMEM record; active ACTMEM policy refined by ADR-0018 | proposed record; implementation evidence recorded |
| [ADR-0012: Laputa Markdown Clean Break](./architecture/0012-laputa-markdown-clean-break.md) | Preserved Garden-Laputa authority contract: Markdown authority files, `ACTMEM.MD`, WORLD exclusion and baseline ACTMEM policy refined by ADR-0018, EvoMap capability ownership, clean-break deletion rules | accepted |
| [ADR-0013: Laputa Clean-Break Implementation Architecture](./architecture/0013-laputa-clean-break-implementation-architecture.md) | Accepted implementation design: Markdown store, Persona review/history, ACTMEM tools, Frozen Core, deletion order, HTTP and Console replacement | accepted design |
| [ADR-0014: Mentle Canonical Authority](./architecture/0014-mentle-canonical-authority-and-derived-index-recovery.md) | Canonical SQLite authority, transactional index outbox, rebuildable derived indexes and non-destructive repair | accepted |
| [ADR-0016: Embeddable Laputa Libraries and Modular Monolith](./architecture/0016-laputa-embeddable-modular-monolith.md) | Importable Laputa/Mentle/Garden domain stack and one Laputa application; in-process Vivy option with shared conformance | accepted direction; incremental implementation |
| [Modular-monolith execution baseline](./bmad/laputa-modular-monolith-2026-09/) | Owned waves M0–M5 and explicit conformance/release gates | local M1–M3 implementation; M4 wire and M5 Vivy gates remain |
| [ADR-0015: Laputa External Agent Contract](./architecture/0015-laputa-external-agent-contract.md) | External `laputa-agent/1` REST/MCP wire profile and native adapters | accepted; Vivy transport refined by ADR-0016 |
| [Laputa External Agent SDK BMAD package](./bmad/laputa-external-agent-sdk-2026-09/) | Prior REST-first story baseline; isolate its uncommitted Wave 1 worktree before integration | replanning under ADR-0016 |
| [Authority & Recovery GOAL package](./bmad/garden-authority-recovery-2026-09/) | Execution contract, frozen API, Story graph, tests, dirty-tree controls, Wave gates and implementation evidence | executed; complete |
| [Operator Runbook](./bmad/garden-authority-recovery-2026-09/operator-runbook.md) | Canonical backup, derived-index rebuild, live health and rollback procedures | active |
| [Console Frontend Pre-Design](./design/garden-memoryos-console-frontend-pre-design.md) | Overall Console design: Persona, Memory, Evolution, Chat Approval workspaces and legacy UI removal | implemented |
| [ADR-0006: Semantic Ingestion](./architecture/0006-semantic-ingestion-and-obsidian-adapter.md) | Raw-first evidence and bounded evidence-read contract | accepted |
| [ADR-0007: EvoMap Mailbox](./architecture/0007-evomap-mailbox.md) | EvoMap inbox/outbox, privacy gate, retry/dead-letter semantics | accepted, refined |
| [ADR-0009: AMBITION / USER SUGGESTIONS](./architecture/0009-ambition-user-suggestions-modules.md) | Human report modules outside Laputa authority | accepted |
| [ADR-0010: EvoMap Hub Transport](./architecture/0010-evomap-hub-transport-provider.md) | GEP-A2A transport with explicit publication gates | accepted, refined |
| [ADR-0011: Recoverable Indexing](./architecture/0011-recoverable-indexing-and-evidence-contract.md) | Provenance, identity, ingest and health inputs; recovery/backend sequencing superseded by ADR-0014 | proposed, refined |

## Archive

[2026-08-14 Laputa clean break archive](./archive/2026-08-14-laputa-clean-break/) preserves the former active architecture. It is evidence only and must not guide new code.

[2026-08-01 pre-MemoryOS archive](./archive/2026-08-01-pre-memoryos-redesign/) preserves earlier history.

## Rule

No active document may introduce or recommend a JSON personality descriptor, `01-05` authority-section model, `MEMORY.MD`/`memory_md`, automatic WORLD projection or ACTMEM admission outside the selected ADR-0018 policy, a generic Persona governance workflow, or any legacy fallback. Those concepts exist only in the archives.
