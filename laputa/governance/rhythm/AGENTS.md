<!-- Parent: ../../AGENTS.md -->

# laputa/governance/rhythm — Legacy Report Implementation

This package requires the delete-or-rehome review mandated by ADR-0013. It must not write reports into Laputa sections or become a Persona/ACTMEM consumer by default.

- Human daily/weekly/monthly reports are Garden-owned artifacts persisted in Garden SQLite.
- Do not connect report generation to the retired JSON store, Persona history, ACTMEM authority or automatic ContextView assembly.
- Preserve reusable pure generation code only when an owning Garden report Story explicitly rehomes and tests it.
- Existing section-backed paths are deletion targets, not compatibility requirements.

Parent reference: `../../AGENTS.md`
