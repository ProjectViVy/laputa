# Garden

Unified CLI / HTTP entry for Laputa governance and mentle memory.

## Module

```
github.com/dashimaki/garden
```

Depends on sibling modules via `go.mod` replace:

- `../laputa` → `github.com/dashimaki/laputa/governance`
- `../mentle` → `github.com/dashimaki/mentle/facade`

## HTTP API (v2)

| Method | Route | Body / params |
|--------|-------|---------------|
| POST | `/v2/memories` | canonical `{"content","kind","scope",...}` |
| GET | `/v2/memories/{id}` | — |
| GET | `/v2/memories` | `?kind=&status=&limit=` |
| PATCH | `/v2/memories/{id}` | `{"content","expected_version"}` |
| DELETE | `/v2/memories/{id}` | — |
| POST | `/v2/ingest/sessions` | session-end transcript |
| GET | `/v2/ingestions/{id}` | ingestion status |
| POST | `/v2/recall/bootstrap` | `{"intent","budget_chars"}` |
| GET | `/health` | — |

The legacy v1 CRUD translator was removed (ADR-0008); the HTTP surface is v2-only.

```bash
./garden.exe &
curl -s -X POST http://127.0.0.1:7373/v2/memories \
  -H 'Content-Type: application/json' \
  -d '{"content":"decision text","kind":"decision","scope":"project:garden"}'
curl -s http://127.0.0.1:7373/v2/memories
curl -s http://127.0.0.1:7373/health
```

## Build

```bash
cd garden
go mod tidy
go build -o garden.exe .
go test ./internal/...
```

## Governed Agentic RAG

Garden runs Fast/Deep recall as governed pipelines. Laputa supplies
read-only policy and governance evidence; Mentle supplies hybrid memory, KG,
and timeline retrieval. The response is a compact, cited context package:

```bash
curl -s -X POST http://127.0.0.1:7373/v2/recall/fast \
  -H "Content-Type: application/json" \
  -d '{"query":"What decisions constrain the current task?","budget_chars":4000}'

curl -s http://127.0.0.1:7373/v2/pipelines
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
