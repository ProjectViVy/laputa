# laputa/governance — Retired JSON Engine

The current `governance` package is a legacy implementation baseline. Its section registry, JSON file store, dot-path patching, `map[string]any` content bodies, `Commitment`, `Preferences`, `memory_md`, and generic mutation service do not describe Garden's target architecture.

The current contract is [`../../docs/architecture/0012-laputa-markdown-clean-break.md`](../../docs/architecture/0012-laputa-markdown-clean-break.md).

## Replacement Target

The replacement must provide:

- Markdown authority documents: `IDENTITY.MD`, `RELATIONSHIP.MD`, `REDLINE.MD`, `USER.MD`, `DREAM.MD`, `DARK.MD`, `WORLD.MD`.
- A separate `ACTMEM.MD` activity-memory contract.
- Atomic complete-document writes with per-document limits.
- Append-only complete snapshots and textual before/after diffs.
- Bounded, session-frozen Frozen Core projection for the first six documents only.
- Persona-specific content review and direct-write exceptions, not generic governance approval.

## Prohibited Target Behavior

- No JSON personality descriptors or `.laputa/sections/*.json` authority store.
- No JSON Patch, dot-path patch, JSON schema, compatibility map, migration, dual read/write, or fallback path.
- No automatic `WORLD.MD` or `ACTMEM.MD` context projection.
- No EvoMap proposal/mailbox or Skill lifecycle storage.

Do not expand the legacy engine. New implementation work must be introduced through the deletion-first clean-break plan, with tests that prove retired contracts cannot be reached.
