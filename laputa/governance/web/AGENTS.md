<!-- Parent: ../../AGENTS.md -->

# laputa/governance/web — Retired Governance HTTP API

The section/authority/report/audit routes in this package are retired and must not be exposed as a fallback service.

- New HTTP adapters live in Garden and follow the frozen Persona/ACTMEM domain contract.
- Do not add section endpoints, generic Governance mutation, compatibility aliases or direct raw-store access.
- Delete this package after target handlers are live and scanner enforcement proves no runtime consumer remains.

Parent reference: `../../AGENTS.md`
