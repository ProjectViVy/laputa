# Pre-GOAL Execution Contract Record

**Date:** 2026-09-03
**Status:** complete
**Runtime implementation:** not started
**GOAL activation:** false

## Decisions recorded

- Full Waves 0–3 scope.
- ADR-0014 accepted as written.
- At most two disjoint execution lanes.
- Existing intended dirty work is audited for adoption in Wave 0.
- One Coordinator-owned checkpoint commit per Wave; never push.
- Implementation begins only after the owner explicitly says `运行 GOAL`.

## Documentation changes

- Added `GOAL-EXECUTION-RUNBOOK.md` with objective, precedence, roles, locks, Story state machine, lane schedule, dirty-tree control, verification and commit rules.
- Synchronized PRD, Architecture Spine, BMAD README/TODOLIST, readiness, baseline inventory and sprint status.
- Accepted ADR-0014 and recorded how it refines ADR-0011.
- Updated root documentation/status entry points.
- Replaced active stale Agent guidance for Garden adapters/tests, Laputa legacy packages and Mentle WAL/recovery with the accepted contracts.

## Live probe evidence

- `go test ./internal/server -run TestMonthlyReportModulesViaHTTP -count=1`: failed as recorded by C-02; no report generated for the fixed historical window.
- `go test ./persona/... -count=1`: passed.
- `go test ./facade/... -count=1`: passed.
- `npm exec tsc -- --noEmit -p tsconfig.json`: passed.
- `npm exec vite -- build`: passed.
- Go was resolved at `C:\Program Files\Go\bin\go.exe` because the shell PATH did not contain `go`.

No production source was changed, no file was staged or committed, and no GOAL was created.
