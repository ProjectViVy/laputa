<!-- Parent: ../AGENTS.md -->

# mentle/integration — Integration Tests and Harness

**Generated:** 2026-08-01  
**Purpose:** Full-system integration tests and test utilities

---

## Purpose

The `integration/` directory contains end-to-end integration tests:

- **CLI tests** — test mentle CLI commands with real palace
- **MCP server tests** — verify MCP protocol compliance
- **LLM tests** — integration with embedding models
- **Full pipelines** — memory mining → storage → search → retrieval

Tests use real data, real backends, and complete workflows.

---

## Structure

```
integration/
├── cli_test.go                    # CLI command integration tests
├── mcp_test.go                    # MCP server protocol tests
├── llama.log                      # Integration test log (if present)
└── run_integration_tests.sh       # Test harness script
```

---

## Running Integration Tests

### All Integration Tests

```bash
./run_integration_tests.sh
```

### Specific Test

```bash
go test -v ./integration/ -run TestCLI
go test -v ./integration/ -run TestMCP
```

### With Verbosity

```bash
go test -v -run Integration ./integration/ 2>&1 | tee integration.log
```

---

## Test Scenarios

### CLI Integration

- Initialize palace
- Mine project files
- Search memories
- Store new memories
- List wings and rooms

### MCP Server

- Start MCP server
- Client connects
- Send tool calls
- Verify responses

### Full Workflow

- Init palace
- Mine files → create cards
- Vector embed cards
- Search by query
- Retrieve evidence

---

## Exit Gates

Before release, integration tests must pass:

```bash
./run_integration_tests.sh
# All tests pass with no errors
```

---

## Build & Test

### Build Integration Tests

```bash
cd mentle
go build -o integration.test ./integration/
```

### Run from Binary

```bash
./integration.test
```

---

## Conventions

- **Test data:** use fixtures/ directory for test files
- **Cleanup:** tests clean up created data after run
- **Isolation:** each test is independent
- **Timeouts:** explicit context.WithTimeout for each operation
- **Assertions:** standard Go testing patterns

---

## MANUAL

When adding integration tests:

1. Use `*testing.T` for assertions
2. Clean up test data after test
3. Document test scenarios
4. Run before committing

Parent reference: ../AGENTS.md
