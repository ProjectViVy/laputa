<!-- Parent: ../../AGENTS.md -->

# laputa/governance/wakeup — Legacy Event Provider Review

This package predates the Markdown clean break and requires a separate delete-or-rehome decision.

- Do not watch `.laputa/sections`, numbered authority names or generic Governance changes.
- Do not trigger automatic WORLD projection, ACTMEM admission outside the selected [ADR-0018](../../../docs/architecture/0018-vivy-unified-context-and-recap-policy.md) policy, or direct Persona/Skill mutation.
- Reusable event notification code may be retained only when a current domain owns the event and the dependency boundary is explicit.

Parent reference: `../../AGENTS.md`
