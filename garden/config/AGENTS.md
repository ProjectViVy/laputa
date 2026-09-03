<!-- Parent: ../AGENTS.md -->

# garden/config — Runtime Configuration Examples

Configuration files document deployable Garden runtime options and safe defaults.

## Rules

- Keep examples synchronized with the current schema and environment variables.
- Configuration may select recall/planner/runtime behavior but cannot enable a retired Governance engine, JSON Persona store, automatic WORLD/ACTMEM projection or Mentle authority fallback.
- Optional vector backends remain disabled/deferred until they satisfy ADR-0014; configuration must not make a stub backend appear live.
- Capability tokens are supplied through secrets/environment, never committed values. Actor labels do not configure authorization.
- Examples use loopback binding and deterministic planner defaults unless a feature explicitly requires otherwise.

Parent reference: `../AGENTS.md`
