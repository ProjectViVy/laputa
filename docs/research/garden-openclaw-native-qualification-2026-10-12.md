# OpenClaw native Persona qualification — 2026-10-12

This records the actual S16 experiment, not a version-range or native-memory certification. [Index](../superpowers/plans/garden-standalone-v1/index.md) owns Story status; [S16](../superpowers/plans/garden-standalone-v1/S16.md) owns acceptance and [S09](../superpowers/plans/garden-standalone-v1/S09.md) owns native recall/capture.

Tested: **OpenClaw 2026.10.2 (b8952d4)**, source `b8952d477f0c932d8ddc6da5c89cab5a169b6ff1`, Node 24.19.0, Linux x86_64, actual embedded runner. Reused the installed pinned package and SDK. A built plugin was staged separately from the host dependency tree, referencing the existing SDK; no second heavy install. All host state, configuration, credentials and synthetic authority were isolated under `.superpowers/sdd/S16/`. No user profile, host instructions, credentials or production data were configured/migrated.

## Invocation and observation

The existing build-tagged `TestPersonaBrowserFixture` served actual Garden owner/binding/Persona APIs. A standard-library local OpenAI-compatible provider captured actual HTTP model request JSON and returned a fixed synthetic assistant response. The native runner, plugin loader, permissions, history and prompt assembly were real; the model response was not used as evidence of loading.

```sh
openclaw plugins registry --refresh
openclaw plugins install --link --no-enable --force --accept-capabilities /absolute/isolated/plugin-stage
openclaw plugins enable garden
openclaw plugins doctor
openclaw agent --local --agent main --session-key agent:main:garden-qualification --message 'synthetic boundary' --json --timeout 30
```

`OPENCLAW_STATE_DIR` and `OPENCLAW_CONFIG_PATH` selected the temporary home. `--force` and capability consent applied only to that known local fixture. Native conversation access and prompt injection grants were explicit; Garden PersonaSync was separately enabled and fresh retry confirmed. Native config used the pinned `agents.entries`, `modelPolicy.allow` and provider `agentRuntime.id` schema. Normal user updates require inspecting existing enablement: `--no-enable` preserves a previously enabled registration.

Main turns retained actual session `c88184cb-e2c0-4b70-8c7d-2cbdb9c3755c` and its directory/history. Captured request N+1 includes N's synthetic assistant message. Existing SOUL.md, AGENTS.md and USER.md SHA-256s stayed identical. Scope fingerprint: `6796a40a640f2be75e14047f2a4566f0080b0274e354fb88cb7acd8f8db58eec` (main + newline + canonical original workspace).

## Observed requests and boundaries

The completed expanded run captured **15 actual model HTTP requests**. Positive main-workspace requests retained the original native SOUL; current system/developer content was inspected separately from retained conversational history. The deliberate workspace-mismatch turn used the alternate native workspace and received no Garden contribution.

| Actual native case | Current model request observation |
| --- | --- |
| N → owner save N+1 → same-session next turn | Identity revision 1 → 2 appears automatically, with one current Garden system block per request. Native base and prior assistant history survive. |
| Acquire edit lease | No current Garden block; Garden records durable pending. |
| Human save with exact lease proof | Pending survives Save; next turn still omits Garden. |
| Armed preview → later owner save → stale confirmation | Confirmation 409; next turn omits Garden. |
| New preview + confirm → next turn | Exactly latest identity revision 4, pending cleared after offered. |
| Native conversation permission denied | Native registry refuses the typed hook; no current Garden block. Native doctor inspection reports the denied permission. |
| Native prompt-injection permission denied | Native registry refuses the typed hook; no current Garden block. Native doctor inspection reports the denied permission. |
| PersonaSync off, then explicitly restored + fresh retry | Off omits; restoration offers revision 4. Recall/Capture controls remain separately off in this fixture. |
| Garden offline | Static omission diagnostic; no cached Garden block. |
| Configured agent mismatch / actual native workspace mismatch | Static unsupported-scope diagnostic; no current Garden block. Original workspace restored and original files preserved. |
| Native subagent session key | Actual runner invoked; no current Garden block. Main binding is not inherited. |
| Unsupported native harness | Native runtime policy rejects `unqualified-fixture-harness` with `owner-plugin-not-activatable` before any model request. No fabricated callback context was substituted. |
| Credential/binding revoked | Actual next turn omits Garden; owner status revoked. |

The SDK trims the outer contribution before composition. Extracting `[Garden Persona …]` through its closing label and restoring the canonical final newline gives the same SHA-256 as Garden offered bytes:

- Identity revision 2: `sha256:4dcd0d2961bd7a6d6574afd86e873eaad39bdf8b747de5ad51f4dbbd2fe3400b`.
- Identity revision 4 after fresh retry and control restoration: `sha256:03f0bd9b284894aed68235b312e5b023f5de46dc0b0dda9829f94c7b8be4a86c`.

Other source revisions remain relationship/redline/user 1. Garden status remains **offered**, without written or loaded receipt; final revocation retains historical offered metadata but does not claim delivery or erasure. Original/prior context may remain in native history. No system replacement, SOUL writer, memory-slot replacement, automatic capture or Persona fallback was added.

## Regression and review

Actual qualification found an Important defect missed by callback-only checks: the CLI first registers full, then builds scoped **discovery** runtime registries. A full-only entry guard lost the hook before model assembly. The added regression failed RED; the minimal `full || discovery` guard passed GREEN. Native loader still rejects disabled entries, and native conversation/prompt policies govern `api.on`. Metadata/setup/tool scans remain excluded. Enabled inspection may populate a private lazy callback registry; it performs no credential read, Garden request, prompt contribution or file write merely from discovery.

Pinned source references: `src/agents/runtime-plugins.ts`, `src/plugins/{loader.ts,loader-registration-plan.ts,registry-state.ts,registry-api.ts,registry-registrars-tools-hooks.ts}`. The independent reviewer verified these paths, the regression, actual requests/revisions/hashes and main-session/history preservation. Fresh `npm test`: **5/5 PASS**, including one-second abort, exact metadata-only request, wrong scope/worker/harness/version omission and failure without cache/secret diagnostics.

Supporting synthetic artifacts are preserved at `/workspace/garden-standalone-v1-evidence-20261012/S16/`: `qualify_openclaw.py`, `provider-requests.json`, `native-results.json`, exact per-case CLI logs, `code-review.md`, and `positive-first-pass/`. The script and model bodies contain only isolated synthetic data; the credential file and native state databases are excluded. This qualifies only the exact embedded Persona path above. Other hosts, OS/version ranges and native memory remain separate evidence gates.
