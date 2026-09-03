<!-- Parent: ../AGENTS.md -->

# laputa/governance/store — Retired JSON Store

This package is a clean-break deletion target. Its JSON section files, registry metadata, `map[string]any` bodies and section mutation API are not supported architecture.

- Do not add features, compatibility reads, migration, aliases or new consumers.
- Production Persona work belongs in `laputa/persona`; ACTMEM belongs in `laputa/actmem`.
- Existing tests are useful only to understand deletion impact and must not be counted as target acceptance.
- Remove the package after all runtime callers have moved and the architecture scanner enforces zero references.

Parent reference: `../AGENTS.md`
