<!-- Parent: ../AGENTS.md -->

# garden/internal/server — Domain HTTP Adapters

This package owns HTTP mechanics and composition of domain handlers. Business mutation rules remain in Persona, ACTMEM, Mentle Facade, Garden runtime and EvoMap services.

## Target contract

Use `docs/bmad/garden-authority-recovery-2026-09/api-contract.md` as the exact route, DTO, principal and error source. The clean-break groups are:

- `/v2/persona/documents`, `/v2/persona/reviews`, `/v2/persona/history`, initialization and repair;
- `/v2/actmem` and `/v2/actmem/query` plus separately authorized maintenance;
- `/v2/memories` through canonical Mentle Facade;
- `/v2/admin/index-health` from live probes;
- existing recall, activity, evolution and mailbox domain routes that remain valid.

At atomic cutover, remove `/v2/persona/files`, `/v2/persona/requests`, `/v2/governance/*` and `/v2/cognitive/world`; no aliases or fallback period.

## Rules

- Capability tokens establish `read`, `user`, `agent` or `operator`; `X-Garden-Actor` is audit metadata only.
- Handlers call policy/domain services, never raw file writers, Searcher mutation, vector stores or raw SQLite handles.
- Errors use the stable envelope and codes in the API contract without leaking token or filesystem details.
- WORLD content loads only through an explicit Persona document read. ACTMEM loads only through explicit ACTMEM calls.
- Missing/stub probes are unavailable, not healthy.
- Shared route wiring in `server.go` is an integration lock under the GOAL runbook.

## Verification

```bash
cd garden
GOSUMDB=off go test ./internal/server/...
```

Contract tests cover success DTOs, malformed/unknown fields, principal denial, domain errors, old-route absence and structural WORLD/ACTMEM omission.

Parent reference: `../AGENTS.md`
