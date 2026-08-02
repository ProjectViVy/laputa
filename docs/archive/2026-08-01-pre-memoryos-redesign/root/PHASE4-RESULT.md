# Phase 4 Result — Independent E2E Test Entry

> **Date**: 2026-07-14
> **Plan**: `GARDEN-PLAN.md` §2
> **Previous**: `PHASE3-RESULT.md`

## Summary

Phase 4 completes the fourth independent test entry point. The new E2E test builds the real Garden executable, starts it as a subprocess, and verifies its public HTTP API over a local TCP connection.

## Deliverable

```
garden/e2e/
└── e2e_test.go    # gated by //go:build e2e
```

The test uses only temporary resources:

- a randomly assigned loopback port;
- a temporary executable;
- temporary governance storage and log directories;
- process cleanup through `testing.T.Cleanup`.

It polls `/health`, then performs the complete governance-backed CRUD flow:

1. `POST /v1/memories` writes `section:01-identity`.
2. `GET /v1/memories/{key}` confirms the persisted values.
3. `GET /v1/memories?prefix=...` confirms list routing.
4. `DELETE /v1/memories/{key}` confirms removal.

No mock server or external API credentials are used. The test exercises the compiled Garden process, lifecycle wiring, HTTP server, CRUD handler, router, and governance file store together.

## Independent Test Entries

| Entry | Command |
|---|---|
| governance | `cd laputa && go test ./governance/...` |
| facade | `cd mentle && go test ./facade/...` |
| garden unit | `cd garden && go test ./internal/...` |
| garden E2E | `cd garden && GOSUMDB=off go test -tags=e2e ./e2e/...` |

The build tag keeps E2E out of normal test runs.

## Verification

```text
cd garden
GOSUMDB=off go test ./...
ok   github.com/dashimaki/garden/internal/crud
ok   github.com/dashimaki/garden/internal/lifecycle
ok   github.com/dashimaki/garden/internal/router
ok   github.com/dashimaki/garden/internal/server
ok   github.com/dashimaki/garden/internal/supervision

GOSUMDB=off go test -tags=e2e ./e2e/...
ok   github.com/dashimaki/garden/e2e  4.272s
```

## Result

All five phases in `GARDEN-PLAN.md` are now implemented. Garden has four separate Go test entry points, including an opt-in full-process HTTP E2E test.
