# Development acceptance — verified, not installed

## Executed layers

| Layer | Component cases | Native/host cases | Failed / skipped |
| --- | ---: | ---: | --- |
| Development source, hermetic synthetic profile | 87 | 218 | 0 / 0 |
| Fresh SDK worktree + distributed patch + new environment + installed wheel | 87 | 218 | 0 / 0 |

Ruff passed. Fresh verification requires `garden_hermes.__file__` inside the new environment, not the checkout. Test assets/config are copied to scratch, so project `pythonpath` settings cannot manufacture an installed-package pass. The distributed SDK patch was applied to base `bf0203a794a07aac3ff764972ab9087baa4e4f4e`; the resulting Git index tree exactly matched development commit `e8236cc2a606820880968260f3e056cccfcfe8bb`. Wheel resources (plugin manifest/entry, patch and upstream license) were checked before replay.

This is serial Songbai implementation and self-review, not an independent reviewer verdict. Counts describe executed test cases, not measured coverage percentages, stochastic inference quality, token spending or production acceptance. All persona/history/request contents are synthetic; no real source bootstrap, private model call, installation, current prompt rewrite or push occurred.

## Evidence map

| Obligations | Exercised evidence / retained boundary |
| --- | --- |
| C01,C26,C34,C37–C39,C41 | `tests/test_persona_markdown.py`, `test_unassigned_mission.py`, `test_role_isolation.py`, `test_readback_alias.py`: explicit role/home/pin, opaque text, identical-hash cross-role negatives, aliases and protected-file refusal |
| C02,C14,C16–C21,C28 | isolated `tests/agent/test_persona_stage.py`; `host_tests/test_native_plugin.py` and `test_plugin_loader.py`: actual finalizer/worker/SDK lease, metadata-before-text consent, remote/missing-surface refusal, unload/cancel, no hidden-history/full-SOUL egress |
| C03–C06,C22,C29 | `tests/test_proposals.py`; native SDK completion matrix: business checks independent of schema, routing/no-change/refusal, finite discriminators, provenance/counterevidence, invalid budgets, duplicates, late/failed calls. Real-model classification/stability accuracy is not claimed. |
| C07–C10,C24–C27,C30,C32,C35 | `tests/test_assemble_apply.py`, `test_storage_boundaries.py`; actual native cancellation token tests; `host_tests/test_crash_recovery.py`: prepared short writes, drift, pending/unknown/rollback, corrupt backup, abrupt process exit and no replay. No ACID/CAS/HTTP abort guarantee. |
| C11–C13,C31 | `tests/test_activation*.py`, `host_tests/test_native_activation.py`: actual outbound chat requests, old cache, genuine DB/history resume, genuinely new session, truncation, wrong home/overlay/unsupported observations. No refresh/reset/forced compression and no certification of a live non-chat transport. |
| C15 | **Deferred, not authorized**: real Songbai/private-provider/installed loop. The synthetic SDK/file/request tests do not substitute for it. |
| C23,C25 | abrupt child exits after source or SOUL rename, restart inspection, no new model call and unchanged recovery marker. No durable task delivery or power-loss guarantee. |
| C33,C36,C40 | `tests/test_role_isolation.py`, `test_storage_boundaries.py`: optional HOST only in local assembly, no USER/WORLD/notes collection, human-source sync with zero model calls. No real shared folder/console/database created. |

Native regressions include background review, finalizer, LLM routing, profile precedence and plugin loading/scope; they are an affected slice, not the full upstream Hermes suite. The verifier uses a whitelist HOME/profile/TMPDIR and an alternative serial pytest harness because the pinned canonical wrapper hardcodes `/var/tmp`.

## Scope boundaries

Only normalized chat identity observation is exercised. Known/unknown overlays, missing observation or incomplete identity stay unverified; observed bytes are not proof of behavior. Hosted plugin isolation and other OS/filesystem/live transport variants are not certified. Auxiliary physical retries/fallback, resource contention and cost require separately approved runtime policy. Authority-sensitive wording triage is heuristic; native security must not depend on persona wording.

Source permissions, current binding, Mission pin and native budget are checked; noncooperating editor/config races still exist after the last check. Temporary private cleanup may be retained when its outcome cannot be established. There is no independent worker, runtime JSON ledger, durable event queue, new memory backend or public control API.

## Reproduction

Use `tools/verify.py` as documented in README and [operator guide](../OPERATOR.md). Add `--installed` with the fresh environment and patched fresh source. JUnit/raw synthetic diagnostic output stays in private scratch; only counts and source coordinates are recorded here.
