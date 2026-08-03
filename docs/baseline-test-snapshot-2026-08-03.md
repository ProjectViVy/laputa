# Baseline Test Snapshot — 2026-08-03

**Commit:** `3c824e8` (Gate E — ADR-0008 executed)
**Go:** 1.26.4 windows/amd64
**Command prefix:** `GOSUMDB=off go test -count=1 -v`

## Results

| Suite | Packages | Tests | Pass | Fail | Duration |
|-------|---------:|------:|-----:|-----:|---------:|
| Laputa `./governance/...` | 7 | 49 | 49 | 0 | 0.32–0.43s/pkg |
| Mentle `./facade/...` | 1 | 20 | 20 | 0 | 0.63s |
| Garden `./internal/...` | 13 | 150 | 150 | 0 | 0.28–2.93s/pkg |
| Garden E2E `-tags=e2e ./e2e/...` | 1 | 1 | 1 | 0 | 5.63s |
| **Total** | **22** | **220** | **220** | **0** | — |

## Packages (garden internal)

activity · arbiter · authority · cognitive · evolution · ingest · lifecycle · mailbox · pipeline · recall · report · server · supervision — all `ok`.

## Notes

- E2E is a single end-to-end test covering the real binary over HTTP (v2 surface only after ADR-0008).
- This snapshot is the pre-Gate-F baseline; Gate F (AMBITION / USER SUGGESTIONS modules, ADR-0009) must not regress any of these suites.
