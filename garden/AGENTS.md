# Garden — HTTP Gateway and Runtime Orchestration

> **2026-10-11 policy supersession:** WORLD remains tool-only. ACTMEM context/schema/tool assumptions below are the baseline; selected VIVY subject-Recap policy overrides only those assumptions. Preserve existing Frozen Core immutability and Mission/Persona authorization. See [ADR-0018](../docs/architecture/0018-vivy-unified-context-and-recap-policy.md). Original design, interface and verification records below remain evidence of their recorded snapshots; this notice does not change their pass/fail results or claim the new policy is implemented.

**Current contract:** [`../docs/architecture/0012-laputa-markdown-clean-break.md`](../docs/architecture/0012-laputa-markdown-clean-break.md)

Garden is the local HTTP and runtime orchestration module. It delivers bounded ContextViews, accepts activity and raw-first ingestion, exposes Mentle evidence, aggregates console state, and hosts EvoMap integration. It does not own personality authority or capability-artifact authority.

## Boundaries

| Subsystem | Garden responsibility | Garden prohibition |
| --- | --- | --- |
| Laputa | Read the bounded Frozen Core projection; surface Persona and ACTMEM operations through their dedicated contracts | Store Persona JSON, project full authority files, auto-inject `WORLD.MD`, admit ACTMEM outside the selected VIVY policy, or route Persona through generic governance mutation |
| Mentle | Ingest materials, discover cards, read bounded evidence, report index health | Promote material to Persona, ACTMEM, or EvoMap authority |
| EvoMap | Start bounded candidate/proposal work, expose mailbox and Hub status | Create, install, publish, or approve a SOP/Skill outside EvoMap |
| Chat Approval | Surface dangerous runtime authorization | Reuse it for Persona review, ACTMEM maintenance, or ordinary memory operations |

## Context Discipline

`/v2/recall/bootstrap`, `/v2/recall/fast`, `/v2/recall/deep`, and every ContextView assembler must obey ADR-0012:

- Frozen Core preserves the existing Mission v2 slot plus bounded projections of `IDENTITY.MD`, `RELATIONSHIP.MD`, `REDLINE.MD`, `USER.MD`, `DREAM.MD`, and `DARK.MD`, captured at session start.
- `WORLD.MD` stays tool-only. Selected ACTMEM Recap admission belongs to the existing VIVY context policy; baseline ContextView endpoints remain governed by their current tested contract.
- Cards are not evidence; full evidence remains an explicit bounded read.
- ContextView is disposable output, not a durable authority or memory store.

## EvoMap

The `internal/evolution` and `internal/mailbox` packages retain their mailbox, privacy gate, retry/dead-letter, evaluation, and GEP-A2A transport responsibilities. EvoMap is the only capability-artifact domain.

## Current Implementation State

The Garden runtime is cut over to the clean-break routes and domain adapters. Retired routes are absent and the architecture guard is a blocking zero-violation gate. New code must not add compatibility paths.

## Testing

```bash
cd garden
GOSUMDB=off go test ./internal/...
GOSUMDB=off go test -tags=e2e ./e2e/...
```

Required baseline tests must prove that automatic WORLD projection, ACTMEM projection outside the selected policy, JSON Persona bodies, `memory_md`, and generic Persona mutation cannot reappear.
