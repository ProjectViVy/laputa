<!-- Parent: ../AGENTS.md -->

# garden/fixtures — Contract Test Data

Fixtures are non-sensitive examples for current domain contracts and isolated tests.

## Rules

- Keep fixtures synchronized with the frozen API contract and current Go/TypeScript DTOs.
- Persona bodies are Markdown strings using the seven closed document kinds.
- Frozen Core fixtures contain only Identity, Relationship, Redline, User, Dream and Dark projections.
- WORLD and ACTMEM may appear only in explicit tool fixtures; they never appear in automatic ContextView fixtures.
- Memory mutation fixtures target canonical Facade semantics and may include `index_pending`; they do not model direct vector/WAL writes.
- Do not add GovernanceProjection, numbered authority sections, JSON Patch, `memory_md`, compatibility aliases or real user data.
- Legacy fixtures are deleted or moved to a dated archive when their owning cutover Story lands; tests must not discover archive fixtures automatically.

Parent reference: `../AGENTS.md`
