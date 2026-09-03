# Garden MemoryOS — Agents & Architecture Guide

**Current architecture:** [`docs/architecture/0012-laputa-markdown-clean-break.md`](docs/architecture/0012-laputa-markdown-clean-break.md)

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

- One profile has one authority directory with exactly `IDENTITY.MD`, `RELATIONSHIP.MD`, `REDLINE.MD`, `USER.MD`, `DREAM.MD`, `DARK.MD`, and `WORLD.MD`.
- Authority bodies are Markdown only. JSON bodies, JSON Patch, and `map[string]any` authority content are prohibited.
- `ACTMEM.MD` is profile-wide activity memory. It is not a personality file, BML/Mentle evidence, or a session checkpoint.
- Frozen Core contains bounded, session-frozen projections of the first six authority files only.
- `WORLD.MD` and `ACTMEM.MD` are tool-only. They must not enter bootstrap, Fast Recall, Deep Recall, or automatic ContextView assembly.
- Persona content review is separate from the Chat Approval Center. ACTMEM maintenance is direct write. EvoMap owns capability artifacts.
- Retired JSON descriptors have no migration, compatibility mapping, dual read/write, discovery, or fallback path.

## Engineering Rules

1. Read ADR-0012 before changing `laputa/`, `garden/internal/authority`, `garden/internal/activity`, `garden/internal/recall`, `garden/internal/evolution`, mailbox, server contracts, or Console information architecture.
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
- Existing source code still implements the retired model. It is an implementation baseline to delete, not a reason to alter ADR-0012.
