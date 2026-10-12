# Hermes native Persona qualification — 2026-10-12

[Index](../superpowers/plans/garden-standalone-v1/index.md) owns Story status; [S17](../superpowers/plans/garden-standalone-v1/S17.md) owns acceptance. This experiment qualifies exact **Hermes0.21.7**, source `71d34bad1045610252eda02ee79549436896e784`, CPython3.14.7, Linux x86_64 persistent CLI. Gateway, other builds/platforms and S09 native memory remain unqualified.

The immutable source's frozen core lock was installed once in an isolated environment. Its nativepm CPython archive matched SHA256 `0ab3305457051cd3e7c031857e005f1bda17c218a1990567dacaaac6dd1d14f0`. All synthetic configuration, credentials, native home/history and plugin installation stayed under `.superpowers/sdd/S17/`. Existing native-created SOUL and workspace AGENTS were preserved; no real user profile or production data was changed.

## Native execution

The build-tagged Garden fixture served actual owner/binding/Persona APIs. A local synthetic provider captured real model HTTP bodies; fixed assistant replies were not evidence of loading. Native loader, runtime, hook assembly and SQLite history were unchanged.

```sh
HERMES_HOME=/absolute/isolated/existing-home python -m hermes_cli.main --version
HERMES_HOME=/absolute/isolated/existing-home python -m hermes_cli.main plugins enable garden
HERMES_HOME=/absolute/isolated/existing-home python -m hermes_cli.main chat --cli --provider fixture -m fixture-model --resume SESSION -q 'synthetic boundary' --run-budget 20
```

The plugin directory was copied only into the fixture's `plugins/garden`; settings referenced a private credential file and matching existing home. Source `PYTHONPATH` and the exact isolated Python selected the pinned build. The native `plugins doctor` probe reported no hooks because it imports into a fresh unconfigured temporary home with connects blocked; actual model captures prove configured activation. Garden PersonaSync and fresh retry were separately enabled; recall/capture stayed off.

## Actual observations

One continuous PTY process (PID61236), existing session `20261012_203504_b149bf`, performed N → owner Save N+1 → next turn without restart or profile migration. Main model requests were indices2(seed),7(N),8(N1), separately counted from metadata probes and auxiliary title calls. Current-user Garden revisions1/2 matched offered hashes `6bc7bf589d4f71a98f44bc3dd0b2b4ca66bbfa6a0036cfaf8d9b768ede5ea7fa` / `21f5d8f0365b8d4386048605dfe9b0478a0da8957f5db3ae62d226a0262bebb4`. Canonical final newline was restored when hashing the captured block. All three system hashes remained `1465bd72172d213acc11cefb6208ab164445f5922721a674cb70f08903c081b7`, containing the full original native SOUL. Six committed SQLite rows preserved seed/N/N1 user and assistant history; raw human text stayed clean and api_content retained the additive context.

| Actual native case | Observation |
| --- | --- |
| Same-process N→N1 | Newest revision in current user context; unchanged system, SOUL/AGENTS, existing profile/session and prior assistant history. |
| Continuous edit lock→Save | Current Garden omitted; pending survives save. |
| Preview→later revision→stale confirm | Confirmation409, next current contribution omitted. |
| Fresh preview/confirm | Latest identity revision4 offered at next turn. |
| PersonaSync off/restored | Off omits; restored with fresh retry offers. Other controls independent. |
| Projection endpoint closed/restored | Real same-session native command omits current Garden on connection refusal, then restores it. |
| Native cache-parity ephemeral fork / parent-owned AIAgent | Full actual model pipelines omit current Garden and issue zero projection requests. |
| Revoke | Same-session real CLI omits new Garden context; owner status revoked; native system/files unchanged. |

Expanded continuous lifecycle used session `20261012_204530_5ae61c` and nine main model requests, followed by five separate remaining-path model requests and two focused revoke model requests. Its PTY offline input never reached native submission (no SQLite user row), so offline was qualified through the normal same-session command path instead. A historical remaining script variable shadow stopped before Revoke; the focused corrected Revoke run passed. These fixture limitations are retained in raw evidence and do not count as product failures or simulated hook qualification.

The contribution is additive **user context**, below native SOUL/system priority. Older api_content remains in native history and may replay; disable/revoke prevents new contributions, not erasure. Status is offered only, with no written/loaded receipt. No system section, MemoryProvider, transcript transfer, cached fallback, instruction writer or autonomous producer is added.

## Focused tests and evidence

Nine standard-library tests passed on pinned Python3.14.7. Independent review reproduced and corrected environment-proxy credential routing and streamed bodies escaping an inactivity timeout, then removed unbounded localhost name lookup. Direct numeric loopback transport shares one monotonic deadline across connect/headers/body and synchronously closes ownership. Independent slow-header/body requests ended around1.002s with owned descriptors closed and process FD count5→5. No repeated heavy dependency installation or broad passed-suite rerun was needed.

Synthetic evidence: `/workspace/garden-standalone-v1-evidence-20261012/S17/`, separated into `positive-first-pass/`, `continuous-lifecycle/`, `remaining-lifecycle/`, `revoke-lifecycle/`; actual request bodies, projection metadata/auth-reference comparisons, native logs, fixture scripts and independent review report. Credentials and native SQLite databases are excluded from the evidence copy. The original isolated database remains available for read-only inspection.
