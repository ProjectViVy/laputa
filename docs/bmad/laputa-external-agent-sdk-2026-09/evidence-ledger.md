# E00-S01 — Live Surface Evidence Ledger

**Status:** done  
**Date:** 2026-09-04  
**Method:** parent baseline + two read-only independent audits  
**Source/test/config changes:** none  
**Commit:** no

## Locked repositories

| Repository | Branch / HEAD | Baseline | Scope rule |
|---|---|---|---|
| Garden | `main` / `5158f9b9c38d943682d0d844e4a2f8e1c4a139f2`, ahead 12 | 4 tracked unstaged; 112701 untracked paths, dominated by pre-existing test cache | only Coordinator documentation is owned; cache/tool state/research snapshots excluded |
| AGENT-VIVY | `main` / `17722669c7fbabf1e28631b3acd9e7436955c450`, ahead 140 | pre-existing `M studio`; no staged/untracked paths | root worktree and `studio/` excluded; provider must use new declared worktree |

## Garden evidence ledger

| Path:line | Observed fact | `laputa-agent/1` decision / gap |
|---|---|---|
| `garden/internal/server/server.go:62-133` | Existing `/v2` domain REST routes include memory, session ingest/status, bootstrap, Persona, ACTMEM, materials, trace and index health. | Reuse domain routes selectively; **P0** new `/.well-known/laputa-agent.json` is required. |
| `garden/internal/server/auth.go:13-176` | Token-derived principals are `read/user/agent/autodream/operator`; `X-Garden-Actor` is audit-only; loopback read can be tokenless. | Preserve actor rule; **P0** external binding must authenticate/profile-bind agent/platform/session/turn rather than rely on loopback read. |
| `garden/internal/server/server.go:394-428`; `internal/recall/fast.go:26-143`; `internal/recall/context.go:10-20` | Bootstrap returns session Frozen Core, bounded evidence, trace/degraded/warnings; context assembly is Frozen Core + evidence. | Reuse Bootstrap; **P1** add binding, context hash and external response profile. WORLD/ACTMEM/history remain absent automatically. |
| `garden/internal/server/material_handlers.go:18-91` | Query-scoped Search and bounded Expand exist. Search default 20/max 100; Expand per-item 800/max 4000, total 4000/max 16000. | Reuse Search→Expand; **P0** both handlers require principal/binding gates before external SDK exposure. |
| `garden/internal/server/server.go:335-369`; `internal/ingest/service.go:25-253` | Session ingest is async `202`, persists status, survives restart, has event/hash idempotency and changed-content conflict; canonical write calls `MemoryWriter`/Mentle Facade path. | Reuse capture foundation; **P0** add terminal host semantics, AgentBinding/provenance, structured receipt/error and principal-gated status. |
| `garden/internal/server/server.go:537-560` | Current error envelope has top-level code/message/error/retryable/request ID/details. | **P1** normalize external contract code/reason/trace semantics across REST/SDK/MCP; do not claim existing shape is equivalent. |
| `garden/internal/server/persona_api.go:32-545` | Persona read/propose/review/CAS and legal agent P16 behavior exist. | Explicit Persona mapping is available; P16 is excluded from v1 manifest until separately capability-frozen. |
| `garden/internal/server/actmem_handlers.go:24-255` | Read/query are explicit and bounded, but maintenance/capsule delete allow Agent. | **P0** default external Agent capability must deny ACTMEM maintenance/delete. |
| `garden/internal/server/activity_handlers.go:48-76`; `server.go:359-369`; `admin_handlers.go:19-142` | Activity, ingestion status and several admin reads lack principal checks. | **P0** explicit external diagnostics/status must be capability-gated. |
| `garden/cmd/garden-mcp/main.go:370-400,643-679,797-839,1071-1082` | MCP is stdio HTTP client with 14 tools; `memory_search` composes routes locally; errors are text-only; no bootstrap/capture tool. | **P0** not yet thin or structured-equivalent; retain MCP as explicit tool adapter, never Vivy lifecycle transport. |

## AGENT-VIVY evidence ledger

| Path:line | Observed fact | Provider mapping |
|---|---|---|
| `internal/domain/session.go:21-29`; `internal/runtime/service.go:285-289` | Existing Session identity/Run entry accepts SessionID. | Map directly; do not create a second Session. |
| `internal/domain/run.go:5-39,84-94`; `runtime/service.go:346-408` | Existing Run state/durable start lifecycle. | Map RunID to turn/provenance. |
| `internal/storage/contracts.go:33-64`; `internal/domain/event.go:8-48,102-137` | Journal is ordered, append-only, exactly-one-terminal runtime history. | Journal memory selection/capture audit only; never replace Mentle. |
| `internal/runtime/service.go:36-40,2185-2235` | RunHook runs after durable terminal event/state and is advisory. | Capture enqueue seam; delivery failure cannot mutate terminal Run. |
| `internal/runtime/service.go:70-114`; `internal/app/app.go:409-442` | ServiceDeps/composition root are injection seams. | Add Vivy-owned MemoryPort/capture outbox here, not plugin ABI. |
| `internal/runtime/prompt.go:17-58`; `runtime/service.go:1129-1203` | Static instruction is cache-stable; dynamic preamble is assembled before model request. | Bootstrap enters dynamic preamble only and produces bounded provenance audit. |
| `internal/tools/tools.go:17-40,169-200,352-385`; `runtime/engine.go:127-148` | Tool contract/registry/HITL surface exists. | Explicit memory tools reuse this layer after capability gate; no Eino in public SDK. |
| `internal/config/config.go:95-105,557-576,605-623` | Strict config; no MemoryPort/Laputa section; `Providers` means LLM providers. | Add separate `memory` config, not an LLM provider; config/error behavior must be explicit. |

## Gate conclusion

E00-S01 closes the evidence/ownership concern. It proves a reuse-first implementation path, but it does not declare `laputa-agent/1` live. E00-S02 still must establish isolated worktrees and freeze exact external contract diffs before source Stories E01+ may start.
