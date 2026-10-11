<!-- Parent: ../AGENTS.md -->

# garden/fixtures — Contract Test Data

> **2026-10-11 policy supersession:** WORLD remains tool-only. ACTMEM context/schema/tool assumptions below are the baseline; selected VIVY subject-Recap policy overrides only those assumptions. Preserve existing Frozen Core immutability and Mission/Persona authorization. See [ADR-0018](../../docs/architecture/0018-vivy-unified-context-and-recap-policy.md). Original design, interface and verification records below remain evidence of their recorded snapshots; this notice does not change their pass/fail results or claim the new policy is implemented.

Fixtures are non-sensitive examples for current domain contracts and isolated tests.

## Rules

- Keep fixtures synchronized with the frozen API contract and current Go/TypeScript DTOs.
- Persona bodies are Markdown strings using the closed existing kinds, including the recorded human-only Mission addition; this policy adds no kind.
- Frozen Core fixtures preserve the existing six Persona projections and the recorded Mission v2 slot/version semantics. ACTMEM is never a frozen slot.
- WORLD appears only in explicit tool fixtures. Preserve baseline ACTMEM omission fixtures; selected-policy Recap fixtures require explicit scope/budget admission and do not relabel baseline cases.
- Memory mutation fixtures target canonical Facade semantics and may include `index_pending`; they do not model direct vector/WAL writes.
- Do not add GovernanceProjection, numbered authority sections, JSON Patch, `memory_md`, compatibility aliases or real user data.
- Legacy fixtures are deleted or moved to a dated archive when their owning cutover Story lands; tests must not discover archive fixtures automatically.

Parent reference: `../AGENTS.md`
