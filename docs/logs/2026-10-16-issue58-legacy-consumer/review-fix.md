# Consolidated independent-review fix

The independent review of native checkpoint
`3c365207d295221c4a3c5667fc71a6a465ee59f8` found two Important defects in
ingestion metadata validation. This increment fixes only those two findings;
it does not activate VIVY BML or change dependency pins.

An applying/unknown attempt now requires a present, non-null original
`before.revision` and Before object. Missing/null metadata returns an error
before projection or resaving. The original activity_json, including source,
error and incomplete Before, remains byte-for-byte unchanged. Explicit revision
zero remains legitimate: genuine archived captures recover native revision 1;
the complete nonzero snapshot recovers the actual native revision 3. Native
archive proof does not supply a missing original revision.

The shared ingestion decoder now validates the entire JSON document before
using the existing strict decoder. Extra closing delimiters and a second JSON
value are rejected. Existing unknown-field/duplicate-key rejection remains in
place. The shared evolution decoder and its other consumers are unchanged.

The new committed native fixtures run actual capture, fold, v3 policy and
archive lookup. They exercise ActivityProjection, projectActivity, drainActivity,
real Start, HighWatermark and ArchiveSession and assert unchanged raw activity
JSON on every rejection. Exact authority/archive bytes and current C1 head,
including logical revision/coverage, are unchanged. Original worker/restart,
legacy/header and Windows test source remains intact.

## Author verification

All commands used the unchanged `/tmp/issue58-localonly-runner.py` with offline
dependencies, credential-name stripping and a non-forwarding deny proxy. Each
recorded zero external requests and zero forwarding. Counts include Go parent
tests and subtests; package results are separate.

| Run | Tests pass/fail/skip | Packages pass/fail | Seconds |
| --- | --- | --- | --- |
| New metadata RED before production changes | 8/12/0 | 0/1 | 1.573 |
| New metadata GREEN | 20/0/0 | 1/0 | 1.372 |
| Garden focused normal | 181/0/0 | 3/0 | 10.365 |
| Garden focused race | 181/0/0 | 3/0 | 42.326 |
| ACTMEM normal | 120/0/0 | 1/0 | 0.365 |
| ACTMEM race | 120/0/0 | 1/0 | 3.282 |

Garden focused packages: `./internal/ingest ./internal/runtimecore ./agentapi`.
Owner package: `./actmem`. All use `go test -json -p=1 -count=1`; race adds
`-race`. Full commands and unchanged raw logs are in the host evidence directory
`S5-reimplementation/native-producer-fix-logs`. There were no compile-error or
interrupted test attempts. Gofmt and diff whitespace checks pass.

Source checkpoint identity and per-file hashes are recorded in the separate host
fix report after commit, avoiding a self-referential commit hash here. Runner
metadata names the baseline HEAD because verification preceded commit; recorded
source hashes bind the tested working bytes to the committed fixed checkpoint.

Actual Windows execution is NOT RUN. Root owns unchanged independent-probe
verification, fixed affected gates, exact remote publication, dependency pin and
normal module-resolution qualification. This is a source-frozen native fix
checkpoint, not acceptance of consumer activation or whole S5/S6.
