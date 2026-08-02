<!-- Parent: ../AGENTS.md -->

# garden/e2e — End-to-End Integration Tests

**Generated:** 2026-08-01  
**Purpose:** Full-system integration tests with real processes and HTTP endpoints

---

## Purpose

The `e2e/` directory contains end-to-end tests that verify Garden as a complete system:

- **Real HTTP server** (not mocked)
- **Real backend connections** (Laputa, Mentle)
- **Complete request/response cycles** (no test fakes)
- **Degradation scenarios** (backend failures, timeouts)

Tests require `-tags=e2e` to compile and run.

---

## Structure

```
e2e/
├── external_e2e_test.go       # End-to-end test suite
└── (integration test utilities as needed)
```

---

## Key Test Scenarios

### HTTP API Contract Verification

- POST `/v2/recall/fast` returns valid RecallResponse
- POST `/v2/recall/deep` includes RecallTrace
- POST `/v2/activity/events` accepts and stores events
- GET `/health` responds with status

### Degradation Paths

- Mentle unavailable → Fast Recall still works (governance-only)
- LLM unavailable → Deep Recall falls back to deterministic planner
- Both unavailable → health check returns degraded status

### Error Handling

- Invalid requests return 400 with error message
- Unauthorized governance mutations return 403
- Missing required fields return 422
- Server errors return 500 with trace ID

---

## Build & Test

### Run E2E Tests

```bash
cd garden
GOSUMDB=off go test -tags=e2e ./e2e/...
```

### Run with Output

```bash
GOSUMDB=off go test -v -tags=e2e ./e2e/... -run TestName
```

### Run All Tests (Unit + E2E)

```bash
GOSUMDB=off go test -tags=e2e ./...
```

---

## Test Requirements

Before running E2E tests:

1. **Port 7373 must be available** (Garden HTTP listen port)
2. **Laputa governance store must be accessible** (`.laputa/` directory or configured path)
3. **Mentle backend must be accessible** (SQLite or configured Mentle service)

Tests will start a Garden instance on 127.0.0.1:7373 and shut it down cleanly.

---

## Exit Gates

All E2E tests must pass before release:

- [ ] Fast Recall returns results within P95 latency
- [ ] Deep Recall includes RecallTrace
- [ ] Activity events are persisted and retrievable
- [ ] Mentle unavailable does not crash Fast Recall
- [ ] LLM unavailable triggers graceful fallback
- [ ] Unauthorized mutations are rejected and audited
- [ ] Session end is idempotent on session_id + event_id

---

## Conventions

- Use `TestNameDescribesBehavior` format
- Each test is independent (no shared state)
- Cleanup: delete test files and temp data after test
- Timeouts: set explicit `context.WithTimeout` for each HTTP call
- Assertions: use standard `testing.T` patterns, no frameworks

---

## MANUAL

When updating:

1. Keep tests focused on complete request/response cycles
2. Document any external service dependencies
3. Add new degradation scenarios as features are added
4. Run full suite before committing

Parent reference: `../AGENTS.md`
