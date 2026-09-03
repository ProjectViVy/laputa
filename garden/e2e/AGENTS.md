<!-- Parent: ../AGENTS.md -->

# garden/e2e — Clean-Break End-to-End Tests

The e2e suite proves complete Garden behavior with real process, persistence and HTTP boundaries. Tests compile under the `e2e` tag and use isolated temporary profiles/state.

## Required target proofs

- Five-file Persona initialization, user direct write and agent review work through the accepted domain API.
- ACTMEM survives session and process restart and is available only through explicit read/query/maintenance calls.
- A session's six-document Frozen Core remains immutable after Persona edits and restart; a new session captures current revisions.
- Bootstrap, Fast Recall, Deep Recall, traces and assembled context structurally omit WORLD and ACTMEM.
- Historical `.laputa/sections` files are ignored and never imported or used as fallback.
- Mentle canonical commits and pending index jobs survive restart; derived-index failure is visible as degraded health.
- Old Persona `/files`/`requests`, generic `/v2/governance/*` and `/v2/cognitive/world` routes are absent after cutover.
- MCP/Console use the same live domain contracts and principal rules.

Mentle-unavailable recall may degrade to Frozen Core plus available bounded evidence behavior; it must never fall back to a GovernanceProjection. LLM-unavailable Deep Recall may use its deterministic fallback while still emitting a trace.

## Test discipline

```bash
cd garden
GOSUMDB=off go test -tags=e2e ./e2e/...
```

- Use random available ports rather than assuming 7373 is free.
- Set explicit timeouts and clean up child processes and temporary state.
- Do not use real user profiles, credentials or API keys.
- Retired-behavior tests are rewritten/deleted by their owning cutover Story; passing them is not acceptance evidence.

Parent reference: `../AGENTS.md`
