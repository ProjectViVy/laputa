<!-- Parent: ../../AGENTS.md -->

# laputa/governance/scheduler — Legacy Scheduler Review

This package is not part of the accepted Persona or ACTMEM target. It requires a separate delete-or-rehome decision after dependencies on the retired Governance engine are removed.

- Do not schedule JSON section mutation, Persona changes, ACTMEM maintenance, Skill installation or report persistence through generic Governance APIs.
- Scheduling timing may be preserved only behind a current domain-owned task interface and explicit tests.
- Existing audit/store integrations are deletion targets.

Parent reference: `../../AGENTS.md`
