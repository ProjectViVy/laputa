# Garden

Unified CLI / HTTP entry for Laputa governance and mentle memory.

## Module

```
github.com/dashimaki/garden
```

Depends on sibling modules via `go.mod` replace:

- `../laputa` → `github.com/dashimaki/laputa/governance`
- `../mentle` → `github.com/dashimaki/mentle/facade`

## CRUD

| Action | Key prefix | Backend |
|--------|------------|---------|
| write  | `section:` | governance |
| read   | `section:` | governance |
| list   | `section:` | governance |
| forget | `section:` | governance |
| write  | `memory:`  | mentle facade |
| read   | `memory:`  | mentle facade |
| list   | `memory:`  | mentle facade |
| forget | `memory:`  | mentle facade |

## Build

```bash
cd garden
go mod tidy
go build -o garden.exe .
go test ./internal/...
```

## HTTP API

| Method | Route | Body / params |
|--------|-------|---------------|
| POST | `/v1/memories` | `{"key","value","meta?"}` |
| GET | `/v1/memories/{key}` | — |
| GET | `/v1/memories` | `?prefix=&limit=` (default prefix `section:`) |
| DELETE | `/v1/memories/{key}` | — |
| GET | `/health` | — |

```bash
./garden.exe &
curl -s -X POST http://127.0.0.1:7373/v1/memories \
  -H 'Content-Type: application/json' \
  -d '{"key":"section:01-identity","value":"{\"agent\":\"matsumoto\"}"}'
curl -s http://127.0.0.1:7373/v1/memories/section:01-identity
curl -s http://127.0.0.1:7373/health
```

## Governed Agentic RAG

Garden now runs `agentic_recall_v1` as a governed pipeline. Laputa supplies
read-only policy and governance evidence; Mentle supplies hybrid memory, KG,
and timeline retrieval. The response is a compact, cited context package:

```bash
curl -s -X POST http://127.0.0.1:7373/v1/context/resolve \
  -H "Content-Type: application/json" \
  -d '{"intent":"What decisions constrain the current task?","session_id":"demo"}'

curl -s http://127.0.0.1:7373/v1/pipelines
```

Optional OpenAI-compatible planning uses `GARDEN_RAG_BASE_URL`,
`GARDEN_RAG_API_KEY`, and `GARDEN_RAG_MODEL`. Without them, Garden uses the
deterministic planner and reports planner degradation without failing the
request. Set `GARDEN_PIPELINE_CONFIG` to a YAML file such as
`config/pipelines.yaml`; the default is `~/.garden/pipelines.yaml`, with the
built-in configuration used when the file does not exist.

## Phases

- **Phase 1**: module skeleton + CRUD router
- **Phase 2**: HTTP server on `:7373`
- **Phase 3** (current): lifecycle + supervision + `~/.garden/garden.log`
- **Phase 4**: complete — opt-in, full-process HTTP e2e tests (`go test -tags=e2e ./e2e/...`)
- **Phase 5**: governed Pipeline Runtime + Agentic RAG context resolution

See `../GARDEN-PLAN.md` and `../docs/architecture/0001-garden-merge.md`.
