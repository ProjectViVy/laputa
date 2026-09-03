<!-- Parent: ../AGENTS.md -->

# laputa/cmd — CLI and Smoke-Test Entry Points

Command packages are adapters only. New commands call `laputa/persona` or `laputa/actmem` domain services and obey ADR-0012/0013.

- Do not expose the retired JSON section registry, generic Governance mutation, JSON Patch or numbered authority names.
- Persona initialization creates the required five Markdown documents only under the atomic initialization contract.
- Persona protected writes use the Persona review/write-class service; ACTMEM maintenance stays separate.
- Legacy rhythm/eino smoke programs may be retained only as isolated test utilities until their delete-or-rehome review. They must not wire the retired store into production.
- Commands are non-interactive by default, return meaningful exit codes and never print capability tokens.

Run `GOSUMDB=off go test ./...` from the Laputa module after changes.

Parent reference: `../AGENTS.md`
