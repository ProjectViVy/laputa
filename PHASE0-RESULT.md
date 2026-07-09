# Phase 0 Result — Garden Workspace Merge

> **Date**: 2026-07-09  
> **Plan**: `GARDEN-PLAN.md` §3  
> **ADR**: `docs/architecture/0001-garden-merge.md`

## Summary

Phase 0 infrastructure refactor is complete. Both Go repositories were physically moved into `~/Desktop/garden/`, module paths updated, and top-level `governance` / `facade` packages extracted.

## Completed Steps

### A. Physical move

| Source | Destination | Status |
|---|---|---|
| `~/Desktop/projects/laputa/` | `~/Desktop/garden/laputa/` | Done |
| `~/Desktop/projects/mempalace-go-redis-v2/` | `~/Desktop/garden/mentle/` | Done |

Git history preserved (`.git/` directories intact in both repos).

### B. mentle module rename

- `github.com/dashimaki/mempalace-go-redis` → `github.com/dashimaki/mentle`
- All `.go` import paths updated
- `go mod tidy` + `go build ./...` green

### C. laputa → governance package

| Change | Detail |
|---|---|
| `laputa.go` | → `governance/engine.go` (`package governance`) |
| `internal/rhythm/` | → `governance/rhythm/` |
| `internal/scheduler/` | → `governance/scheduler/` |
| `internal/store/redis/` | → `governance/store/` |
| `internal/wakeup/` | → `governance/wakeup/` |
| `internal/web/` | → `governance/web/` |
| `laputa_test.go` | → `governance/engine_test.go` |
| `cmd/laputa/main.go` | Updated imports + deprecation comment; kept functional as :7373 fallback |

Import alias `laputa "github.com/dashimaki/laputa/governance"` preserves existing `laputa.Engine` references in sub-packages.

Module path unchanged: `github.com/dashimaki/laputa`.

### D. mentle facade package

New files:

```
mentle/facade/
├── facade.go   Service + Init/Close (assembly logic from cmd/server)
├── crud.go     Write/Read/List/Forget stubs (Phase 1 implementation)
└── facade_test.go
```

`cmd/server/main.go` refactored to call `facade.Service.Init()` then register MCP tools.

## Verification

```bash
cd ~/Desktop/garden/laputa
go build ./...                              # OK
go test ./governance/...                    # OK (6 packages)

cd ~/Desktop/garden/mentle
go build ./...                              # OK
go test ./facade/...                        # OK (2 tests)
```

## Workspace layout (post Phase 0)

```
~/Desktop/garden/
├── README.md
├── GARDEN-PLAN.md
├── PHASE0-RESULT.md          ← this file
├── docs/architecture/0001-garden-merge.md
├── laputa/                   module github.com/dashimaki/laputa
│   ├── governance/           ← new top-level package
│   └── cmd/laputa/           ← deprecated fallback binary
└── mentle/                   module github.com/dashimaki/mentle
    ├── facade/               ← new top-level package
    ├── internal/             ← 17 internal packages unchanged
    └── cmd/server/           ← uses facade
```

## Not in scope (deferred)

| Item | Phase |
|---|---|
| `garden/` module + CRUD router | Phase 1 |
| HTTP server on :7373 | Phase 2 |
| Lifecycle / supervision | Phase 3 |
| 4 independent test entry points | Phase 4 |
| facade CRUD real implementations | Phase 1 |
| cmd/server MCP smoke test | Phase 2 |

## Known issues / notes

1. **Network**: `go mod tidy` required `GOSUMDB=off` in this environment due to sumdb connectivity; `go.sum` restored from git when needed.
2. **mentle startup**: Phase 0 validates compile + facade unit tests only; full MCP server startup (embedder/models) not exercised.
3. **Git commits**: Changes are unstaged; per plan, laputa/mentle/garden root each get their own commits when ready.

## Next step

Phase 1: create `~/Desktop/garden/garden/` module with `replace` directives pointing to `../laputa` and `../mentle`, implement 4 CRUD actions + key-prefix router.
