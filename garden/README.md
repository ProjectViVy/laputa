# Garden

Unified CLI / HTTP application and importable Garden domain entry for Laputa governance and Mentle memory.

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
| POST | `/v2/ingest/sessions` | explicit `precompact` or `session_end` transcript, `session_id` and durable event identity |
| GET | `/v2/ingestions/{id}` | ingestion status |
| POST | `/v2/recall/bootstrap` | `{"session_id","intent","budget_chars"}` |
| GET | `/health` | — |

The legacy v1 CRUD translator was removed (ADR-0008); the HTTP surface is v2-only.

```bash
./garden.exe &
curl -s http://127.0.0.1:7373/health
curl -s http://127.0.0.1:7373/v2/memories  # loopback read
```

## Build

```bash
cd garden/console
npm install --no-package-lock --ignore-scripts
npm run build
cd ..
CGO_ENABLED=0 go build -o garden.exe .
CGO_ENABLED=0 GOSUMDB=off go test ./... -count=1
CGO_ENABLED=0 GOSUMDB=off go test -tags=e2e ./e2e/... -count=1
```

Memory writes require a trusted User/Agent capability bearer token configured via
`GARDEN_CAPABILITY_USER_TOKEN`/`GARDEN_CAPABILITY_AGENT_TOKEN`; loopback GET
without a token remains a read-only convenience, not a write grant. Keep the
HTTP listener on loopback unless a separate authentication/exposure decision is
made. See [ADR-0016](../docs/architecture/0016-laputa-embeddable-modular-monolith.md).

## In-process Go host

Import `github.com/dashimaki/garden/agentapi`, not `garden/internal/*` or the
HTTP server. `agentapi.Open` fixes absolute storage/model paths and trusted
profile/principal/agent/platform; `BindSession` supplies only the host session
ID. `Bootstrap`/`FastRecall` expose bounded Frozen Core, explicit reads use
separate methods, and `Capture` accepts durable terminal events only.

The [independent smoke module](../examples/vivy-embed-smoke/) actually calls
Open→BindSession→Bootstrap→Capture→Close under `CGO_ENABLED=0`. The
[Vivy handoff](../docs/bmad/laputa-modular-monolith-2026-09/vivy-handoff.md)
lists future MemoryPort/RunHook work and offline/lexical degradation. Neither
local `replace` imports nor passing this smoke means Vivy is integrated or the
modules have been published.

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
