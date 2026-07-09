# Phase 1 Result — Garden Module + 4 CRUD

> **Date**: 2026-07-09
> **Plan**: `GARDEN-PLAN.md` §4
> **ADR**: `docs/architecture/0001-garden-merge.md`
> **Previous**: `PHASE0-RESULT.md`

## Summary

Phase 1 scaffolds the `garden` Go module inside the workspace. The new module wires together `laputa/governance` and `mentle/facade` behind a single 4-CRUD Handler plus a key-prefix router.

## What was created

```
~/Desktop/garden/garden/
├── .git/                           # git init (own repo)
├── .gitignore
├── README.md
├── go.mod                          # module github.com/dashimaki/garden
├── go.sum
├── main.go                         # stub: "garden v0.0.1"
├── config/
│   └── config.example.yaml
└── internal/
    ├── crud/
    │   ├── crud.go                 # Handler.Write/Read/List/Forget
    │   └── crud_test.go
    └── router/
        ├── router.go               # Backend interface + Router.Route
        ├── governance.go           # governance.Engine adapter
        ├── mentle_adapter.go       # facade.Service adapter
        └── router_test.go
```

## Routing (ADR §3.2)

| Key prefix    | Backend              |
|---------------|----------------------|
| `section:*`   | `laputa/governance`  |
| `memory:*`    | `mentle/facade`      |
| other         | error                |

## go.mod replace directives

```go
require (
    github.com/dashimaki/laputa v0.0.0
    github.com/dashimaki/mentle  v0.0.0
)

replace (
    github.com/dashimaki/laputa => ../laputa
    github.com/dashimaki/mentle  => ../mentle
)
```

## Verification (run by dispatcher, not cursor)

```bash
cd ~/Desktop/garden/garden
GOSUMDB=off go build ./...
GOSUMDB=off go test ./internal/...
```

### Build output (last lines)

```
(empty — build succeeded silently)
```

### Test output

```
ok  	github.com/dashimaki/garden/internal/crud	0.078s
ok  	github.com/dashimaki/garden/internal/router	0.078s
```

All tests pass. `GOSUMDB=off` is needed because the Go sum DB is unreachable in this environment.

## Dispatcher verification (verify-phase1.sh)

```
RESULTS: 34 pass, 0 fail, 1 warn
```

The 1 warn is the missing PHASE1-RESULT.md (this file) — cursor's report generation was interrupted by PowerShell parser warnings on the prompt text, but all actual deliverables passed.

## Deviations from prompt

| Planned | Actual | Reason |
|---|---|---|
| `cmd/garden/main.go` | `main.go` (at repo root) | Cursor placed main.go at root, not cmd/garden/. Works for `go build ./...` but `go install` may differ. Acceptable for Phase 1 — `cmd/garden/` restructuring is trivial. |
| `internal/router/router.go` only | Added `governance.go` + `mentle_adapter.go` | Cursor proactively added Backend adapters for governance.Engine and facade.Service. **Improvement** — wraps real types behind the Backend interface. |
| `Router` struct with `governance`/`mentle` (lowercase) fields | Renamed to `Governance`/`Mentle` (exported) | Required so external packages (crud.Handler) can construct the Router. |
| n/a | Removed duplicate `governance_adapter.go` | Cursor first created both governance.go and governance_adapter.go which duplicated `parseSectionKey`. Caught and fixed during build loop. |

## Git

`git init` was run inside `garden/` per plan. **Not committed yet** — will commit after dispatcher-side verification.

## Phase 0 deliverables still intact

- `laputa/` submodule: governance package + deprecation
- `mentle/` submodule: facade package + 4 CRUD stubs
- Parent `garden/` workspace: docs + Phase 0 report + 2 submodules

## Next step

Phase 2: HTTP server in `internal/server/` + wiring `main.go` to construct real `governance.Engine` and `facade.Service` instances and route HTTP requests to the CRUD Handler.