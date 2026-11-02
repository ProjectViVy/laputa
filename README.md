# Laputa · Garden

Python implementation of role-scoped, plain-text persona evolution for Hermes.

## Current state

| Layer | Status |
|---|---|
| Implementation and synthetic verification | Complete (87 component + 218 host cases, 0 failed/skipped), including a fresh SDK source + distributed patch + installed wheel replay |
| Native SDK extension | Committed locally on top of the pinned base; tree identical to the verified extension |
| Plugin bundle | Installed on the owner's default Hermes profile; native loader and Plugin Doctor pass |
| Gateway restart | Observed on a new process; the existing SOUL and unrelated configuration were unchanged |
| Persona evolution | **Disabled.** No real role sources, no source/write/model consent, no helper calls |
| Public release of this branch | This branch is published as source; the installed copy lives in the owner's Hermes home and is not part of this repository |

Installing the plugin is not the same as enabling evolution. Creating real role documents,
pinning a Mission version, approving a helper model route and allowing automatic writes are
separate owner decisions; each is recorded in [status.json](docs/planning/status.json) and
[installation](docs/planning/INSTALLATION.md).

## First-release contract

- Authoring authority: `<approved_shared_root>/personas/<persona_id>/`, one role per directory.
- Separate opaque UTF-8 documents: MISSION, IDENTITY, RELATIONSHIP, REDLINE, DREAM and DARK (`.MD`). No required headings, blocks, frontmatter or embedded permissions.
- Only IDENTITY/RELATIONSHIP/DREAM/DARK may receive one complete replacement per attempt. Mission must already be assigned and match an owner-approved version; automatic Mission/REDLINE changes are refused.
- Deterministic one-way assembly into native `SOUL.md`. USER/WORLD/notes are not collected or mirrored. Optional private HOST stays local to assembly.
- Persona continuation belongs to the existing managed background review. No Garden database, durable job queue, cron, independent worker, web server or management console.
- Short single-runtime commits with drift checks and exact readback. Separate source/SOUL publications are **not** ACID or global cross-editor CAS.
- Helper input is approved visible text plus editable source context, never inherited history/full SOUL/HOST/tool output/reasoning. One logical auxiliary completion at most; host retries and fallback remain host policy.

## Delivery and verification

The native extension is bundled at `garden_hermes/host_patch/`; the passive plugin bundle is at `garden_hermes/hermes_plugin/`. Its SDK capability is version 1, based on Hermes `bf0203a794a07aac3ff764972ab9087baa4e4f4e`. Unsupported SDKs remain inactive. In-process plugin execution is required; do not relax hosted-isolation policy automatically.

See [operator/developer guide](docs/OPERATOR.md), [development acceptance](docs/planning/DEVELOPMENT-ACCEPTANCE.md), [review](docs/planning/REVIEW.md), and [authorization boundary](docs/planning/IMPLEMENTATION-AUTHORIZATION.md).

```sh
.venv/bin/python tools/verify.py \
  --host-source ../laputa-hermes-dev \
  --host-python "$PWD/.host-venv/bin/python" \
  --component-python "$PWD/.venv/bin/python" \
  --scratch "$HOME/.hermes/cache/scratch/laputa-verification"
```

The verifier uses synthetic inputs and an explicit temporary HOME/profile, runs serially, and does not install anything. `--installed` additionally requires imports from the selected environments rather than the source checkout; it is used for wheel acceptance.

**Real source bootstrap, private-data/model use, activation and public release require separate owner authorization.** The recorded installation used synthetic acceptance only: no real persona text, no real provider call and no observed identity change is claimed.

## Design and history

[PRD](docs/planning/PRD.md) · [architecture](docs/planning/ARCHITECTURE-SPINE.md) · [contracts](docs/planning/CONTRACTS.md) · [Story status](docs/planning/status.json) · [conformance](docs/planning/CONFORMANCE.md).

The original Go implementation and historical work remain archived; they are design references, not this Python implementation or evidence of its acceptance. Existing deployments and other machines are not changed. License: existing repository MIT license.
