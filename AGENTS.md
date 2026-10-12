# Garden MemoryOS — Agents & Architecture Guide

**Selected VIVY context policy:** [ADR-0018](docs/architecture/0018-vivy-unified-context-and-recap-policy.md) refines the ACTMEM rules below; accepted direction, implementation pending.

**Authority architecture:** [`docs/architecture/0012-laputa-markdown-clean-break.md`](docs/architecture/0012-laputa-markdown-clean-break.md)

**Implementation contracts:** [ADR-0013](docs/architecture/0013-laputa-clean-break-implementation-architecture.md), [ADR-0014](docs/architecture/0014-mentle-canonical-authority-and-derived-index-recovery.md)

**Execution contract:** [GOAL Execution Runbook](docs/bmad/garden-authority-recovery-2026-09/GOAL-EXECUTION-RUNBOOK.md)

Garden MemoryOS is a local-first governed memory operating system. Its modules have non-overlapping authority:

```text
Laputa  -> personality authority, Persona review/history, ACTMEM semantics
Mentle  -> raw material, evidence, retrieval, indexing, provenance
Garden  -> activity/runtime orchestration, bounded ContextView, host integration, console
EvoMap  -> capability candidates/proposals, artifacts, mailbox, Hub policy
```

## Non-negotiable Laputa Contract

- One profile has one authority directory. Preserve the seven ADR-0012 documents (`IDENTITY.MD`, `RELATIONSHIP.MD`, `REDLINE.MD`, `USER.MD`, `DREAM.MD`, `DARK.MD`, `WORLD.MD`) and existing human-only `MISSION.MD` recorded by ADR-0017; no new kind is introduced by this policy.
- Authority bodies are Markdown only. JSON bodies, JSON Patch, and `map[string]any` authority content are prohibited.
- `ACTMEM.MD` is profile-wide activity memory. It is not a personality file, BML/Mentle evidence, or a session checkpoint.
- Frozen Core preserves the six established bounded Persona projections and existing human-controlled Mission v2 slot recorded by ADR-0017. It is immutable for an admitted session; this policy adds no authority slot.
- `WORLD.MD` remains tool-only and absent from automatic context. Under the selected ADR-0018 policy, ACTMEM is one persistent subject-level free-form Recap admitted only through VIVY scope/budget policy; it never enters Frozen Core. Baseline transports retain their tested behavior until explicitly revised.
- Persona content review is separate from the Chat Approval Center. ACTMEM updates use the shared revision-aware authority boundary; concurrent human/consolidation edits must conflict rather than lose updates. Active Recap deletion commits a new revision and invalidates projections without erasing raw/archive/backend records. EvoMap owns capability artifacts.
- Retired JSON descriptors have no migration, compatibility mapping, dual read/write, discovery, or fallback path.

## Engineering Rules

1. Read ADR-0018 and the preserved ADR-0012 authority contract before changing `laputa/`, `garden/internal/authority`, `garden/internal/activity`, `garden/internal/recall`, `garden/internal/evolution`, mailbox, server contracts, or Console information architecture.
2. Preserve Mentle's card-before-evidence boundary. A ContextView is disposable output, never authority.
3. EvoMap owns proposal state, evaluation, artifact lifecycle, installation permission, mailbox, privacy gate, and Hub publication policy. No other module creates or installs a Skill.
4. `MEMRULES.MD`, if present, is a Garden runtime-policy input only. It is not a Persona authority, Frozen Core document, or eighth type.
5. New architecture work must classify legacy behavior as Keep, Cut, Deferred, or Drop. Do not silently retain a legacy surface.
6. Mentle canonical SQLite is the sole memory authority. `index_jobs` is its transactional derived-index outbox; JSONL WAL, vector stores and BM25 cannot reconstruct or override canonical memory.

## Baseline Tests

```bash
cd laputa && GOSUMDB=off go test ./...
cd ../mentle && GOSUMDB=off go test ./...
cd ../garden && GOSUMDB=off go test ./...
GOSUMDB=off go test -tags=e2e ./e2e/...
```

## Documentation

- [`docs/README.md`](docs/README.md) is the current documentation index.
- [`docs/archive/2026-08-14-laputa-clean-break/`](docs/archive/2026-08-14-laputa-clean-break/) contains the superseded section/JSON architecture as historical evidence.
- Runtime implementation and historical tests are evidence, not acceptance of ADR-0018. Plan compatibility for Work reconcile/capture consumers; this documentation task authorizes no source, test, script or migration changes.
- Agent ACTMEM inspect/update/delete tools must be normally hidden, discovered and explicitly activated when needed; the engineering mechanism remains unresolved. No always-on schemas or autonomous maintenance.
- Consolidation cannot automatically write Mission, Dream or frozen Persona content. Guide human “must not be forgotten” requests through existing Laputa governance; Recap edits do not permanently pin information.
