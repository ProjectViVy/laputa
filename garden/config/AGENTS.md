<!-- Parent: ../AGENTS.md -->

# garden/config — Runtime Configuration Examples

> **2026-10-11 policy supersession:** WORLD remains tool-only. ACTMEM context/schema/tool assumptions below are the baseline; selected VIVY subject-Recap policy overrides only those assumptions. Preserve existing Frozen Core immutability and Mission/Persona authorization. See [ADR-0018](../../docs/architecture/0018-vivy-unified-context-and-recap-policy.md). Original design, interface and verification records below remain evidence of their recorded snapshots; this notice does not change their pass/fail results or claim the new policy is implemented.

Configuration files document deployable Garden runtime options and safe defaults.

## Rules

- Keep examples synchronized with the current schema and environment variables.
- Configuration may select recall/planner/runtime behavior but cannot enable a retired Governance engine, JSON Persona store, automatic WORLD projection or ACTMEM admission outside ADR-0018 or Mentle authority fallback.
- Optional vector backends remain disabled/deferred until they satisfy ADR-0014; configuration must not make a stub backend appear live.
- Capability tokens are supplied through secrets/environment, never committed values. Actor labels do not configure authorization.
- Examples use loopback binding and deterministic planner defaults unless a feature explicitly requires otherwise.

Parent reference: `../AGENTS.md`
