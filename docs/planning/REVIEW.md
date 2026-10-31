# Serial safety/correctness review

Reviewer/executor: Songbai. This is a separate self-review pass over changed contracts, source/diffs, fault cases, imports and packaged execution; no independent agent was used. No `[verified]` independent-review label is claimed.

## Findings resolved, ordered by severity

| Severity | Finding fixed | Code evidence |
| --- | --- | --- |
| P1 | Missing surface must not upgrade an explicitly remote Agent into classic CLI | isolated `agent/persona_stage.py:51`, current `capture_current_turn`; remote-platform regressions |
| P1 | Native unsupported/exception early gates leaked managed completion ownership | isolated `agent/background_review.py:1239`, `_run_review_in_thread`; native early-gate regressions |
| P1 | Invalid/late inference or unknown callback result could look completed | isolated `agent/persona_stage.py`, `invoke`; `garden_hermes/plugin.py:21`, strict outcome handling |
| P1 | Unsafe file kinds/permissions or aliases could block or falsely pass post-publication readback | `garden_hermes/markdown.py:25`, `garden_hermes/apply.py:20`; real FIFO, world-writable and alias-race regressions |
| P1 | A partial publication, final receipt failure or crashed process must not be called fully persisted or blindly retried | `garden_hermes/apply.py:117`; `host_tests/test_crash_recovery.py`; distinct observed/recorded states |
| P1 | Corrupt previous recovery must not be silently replaced | `garden_hermes/recovery.py:150`; startup/material validation and corrupt-backup regression |
| P1 | Native identity budget could change after registration | `garden_hermes/plugin.py:21`, `PersonaController.current`; native-budget drift regression |
| P2 | Unpublished preparation leaked on drift/cancellation | `garden_hermes/recovery.py:66`; cleanup keeps referenced/uncertain material |
| P2 | Model output discriminator/schema and evidence attribution needed independent validation | `garden_hermes/proposal.py:43`, `garden_hermes/review.py:12`; finite results and strict business guards |

Test-harness corrections are not misreported as production defects: SDK structured input has an instruction/schema header before payload; genuine session resume requires history + DB, not merely a reused ID. Both are reflected in the executable native fixtures.

## Module coverage

| Module coordinate | Review |
| --- | --- |
| `garden_hermes/__init__.py:1` | inspected; covered by the component/SDK acceptance suite |
| `garden_hermes/activation.py:8` | inspected; covered by the component/SDK acceptance suite |
| `garden_hermes/apply.py:20` | inspected; covered by the component/SDK acceptance suite |
| `garden_hermes/assemble.py:6` | inspected; covered by the component/SDK acceptance suite |
| `garden_hermes/domain.py:21` | inspected; covered by the component/SDK acceptance suite |
| `garden_hermes/hermes_plugin/__init__.py:1` | inspected; covered by the component/SDK acceptance suite |
| `garden_hermes/host.py:4` | inspected; covered by the component/SDK acceptance suite |
| `garden_hermes/markdown.py:19` | inspected; covered by the component/SDK acceptance suite |
| `garden_hermes/plugin.py:21` | inspected; covered by the component/SDK acceptance suite |
| `garden_hermes/proposal.py:11` | inspected; covered by the component/SDK acceptance suite |
| `garden_hermes/recovery.py:22` | inspected; covered by the component/SDK acceptance suite |
| `garden_hermes/review.py:8` | inspected; covered by the component/SDK acceptance suite |

The separate SDK diff covers finalizer capture, background-review phase ownership/cleanup, native SDK owned registration, immutable projection and spawn/defer/requeue parameter forwarding. Original cache/tool/history contracts were retained and affected regressions rerun. Static AST scanning found no product `eval`, `exec`, shell=True or pickle deserialization paths; Ruff passed. This does not establish semantic safety for arbitrary natural-language instructions.

## Accepted / deferred risks

- Cooperative single-runtime writer only; no global editor/config CAS, two-file ACID, hardware durability guarantee or instant revocation of an admitted short commit.
- Native helper completion may retry/fallback or continue remotely after cancellation. Real egress/resource/cost verification is unapproved, not inferred from mocks.
- Wording triage is conservative and incomplete. Transport/tool authorization remains native; no QQ/group tool-isolation claim is made.
- The pure activation helper covers the exercised chat request form and known overlay metadata. Real current-provider observation, other OS/filesystems, hosted plugin isolation and actual private persona inference remain unverified.
- Real installation/bootstrap/pilot is excluded. No blocking defect remains in the exercised authorized development scope; these exclusions are not green production acceptance.
