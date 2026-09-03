<!-- Parent: ../AGENTS.md -->

# Documentation — Current Contracts and Execution Records

`docs/` is the canonical documentation entry point. Active implementation work is governed by accepted ADRs and the Authority & Recovery execution package; archived material is evidence only.

## Read order

1. `architecture/0012-laputa-markdown-clean-break.md`
2. `architecture/0013-laputa-clean-break-implementation-architecture.md`
3. `architecture/0014-mentle-canonical-authority-and-derived-index-recovery.md`
4. `bmad/garden-authority-recovery-2026-09/ARCHITECTURE-SPINE.md`
5. `bmad/garden-authority-recovery-2026-09/api-contract.md`
6. `bmad/garden-authority-recovery-2026-09/GOAL-EXECUTION-RUNBOOK.md`
7. The owning Epic, test inventory and Story implementation record.

## Ownership

- `architecture/`: durable decisions and their status.
- `bmad/garden-authority-recovery-2026-09/`: accepted execution planning, API contract, Story graph, tests, status and evidence.
- `design/`: accepted or proposed UX/adapter inputs; never live-state evidence.
- `dev/`: implementation references and audits.
- `archive/`: immutable historical evidence; never a current contract.

## Rules

- ADR-0012 controls Persona, ACTMEM, Frozen Core, WORLD and EvoMap boundaries.
- ADR-0014 controls Mentle authority and recovery: canonical SQLite is authoritative; `index_jobs` is the transactional outbox; vectors/BM25 are derived.
- New architecture decisions require an ADR. Story execution detail belongs in BMAD Epic/implementation records.
- Every active legacy behavior is classified Keep, Cut, Deferred or Drop. Do not silently preserve an old route, DTO, fixture or fallback.
- Do not add JSON Persona authority, `memory_md`, automatic WORLD/ACTMEM context, generic Persona Governance, WAL-as-canonical recovery or optional backend work.
- Internal links are relative, dates are ISO 8601, and status words must agree across the ADR index, BMAD readiness and sprint status.

## Documentation verification

- Check that every active link resolves and archived links are clearly labelled historical.
- Scan active README/AGENTS files for superseded architecture claims.
- Run `git diff --check` on owned documentation changes.
- Do not claim a runtime contract as implemented until its tests and implementation record exist.

Parent reference: `../AGENTS.md`
