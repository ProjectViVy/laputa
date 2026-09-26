# Laputa External Agent SDK — Architecture Spine

**BMAD phase:** Solutioning  
**Status:** Planning baseline; ADR-0015 accepted (implementation status tracked in `sprint-status.yaml`)  
**Date:** 2026-09-04  
**Authority:** ADR-0012/0013/0014; DIVA wins for Persona/ACTMEM semantics.

## 1. System shape

```text
 External host Agent                         explicit tool lane
 ┌─────────────────┐                 ┌──────────────────────────┐
 │ existing session │                 │ Persona / ACTMEM / trace │
 │ run + journal    │                 └────────────┬─────────────┘
 └────────┬────────┘                              │ capability-gated
          │ native lifecycle adapter               ▼
          ▼                        ┌───────────────────────────────┐
 ┌───────────────────────┐         │ Garden canonical REST profile │
 │ laputa-sdk-go          │────────►│ discovery · bind · recall     │
 │ transport + DTO only   │         │ capture · memory · typed errs │
 └───────────────────────┘         └───────┬──────────────┬────────┘
                                            │              │
                                   ┌────────▼─────┐  ┌─────▼──────────┐
                                   │ Laputa        │  │ Mentle facade  │
                                   │ Persona/ACTMEM│  │ canonical SQLite│
                                   └──────────────┘  │ index_jobs      │
                                                      └────────────────┘

 garden-mcp = thin ListTools/CallTool adapter over the same domain contract
```

## 2. Decisions

### EA-001 — Contract-first, REST-canonical

`laputa-agent/1` is a language-neutral semantic contract. Canonical REST is its reference transport. Typed SDKs serialize the contract; MCP projects a constrained tool interface. No adapter may define a separate memory, identity, authorization, or error behavior.

### EA-002 — Capability discovery precedes feature use

`GET /.well-known/laputa-agent.json` is the only discovery endpoint. The manifest includes contract/version/transports/capabilities/budgets/automatic-context/explicit-only/error catalog. Unknown mandatory version/capability fails before a side-effecting request. Manifest values never contain credentials or local paths.

### EA-003 — Server-authorized immutable binding

`AgentBinding` contains only host identity/provenance fields. Token-derived principal and allowed profile are verified server-side. A typed `SessionHandle` is immutable. No request header/body assertion, including actor metadata, may increase access.

### EA-004 — Bootstrap has a narrow automatic lane

Bootstrap uses the existing session-frozen six-slot ContextView and optionally bounded, caller-requested Mentle evidence. Its response type cannot represent WORLD, ACTMEM or Persona history. Static host instructions stay cache-stable; dynamic context is a separately audit-able preamble.

### EA-005 — Retrieval is progressive disclosure

The external Agent profile uses `POST /v2/agent/search` then `POST /v2/agent/evidence/{card_id}`. Each required AgentBinding is JSON body data, not a header or query assertion. Search returns bounded cards/references; Expand resolves selected evidence/source under a separate budget. Both preserve source, revision/status, score, trace, and continuation metadata. This borrows memsearch's disclosure pattern but uses Mentle canonical revisions, not a filesystem collection. Ordinary `GET /v2/materials/*` routes are separately read-gated domain interfaces and cannot bypass the Agent profile.

### EA-006 — Completed-turn Capture is an outbox protocol

Capture accepts one terminal host event. The idempotency identity is `profile_id + session_id + event_id + content_hash`; replays with identical payload converge, and event reuse with changed content conflicts. Server processing may return `202 Accepted` and a receipt handle. The host queue persists/retries only retryable transport/service failures and exposes pending/failed status.

### EA-007 — One write entry point per authority

Remember/capture enter `facade.Service`; Persona proposal uses Persona policy/review; ACTMEM remains its dedicated service. No adapter receives raw Persona file writer, canonical SQLite, Searcher, vector store, or JSONL business-log mutation access.

### EA-008 — Explicit Agent powers are narrow

Agent default writes: capture, Remember, and Persona proposal if capability permits. Agent cannot review-approve/reject, protected direct-write, maintain ACTMEM, or invoke repair. P16 is separately declared and requires current DIVA-parity proof. This does not merge Persona Review, Chat Approval or EvoMap review.

### EA-009 — AGENT-VIVY is a host adapter, not a dependency of SDK

Public SDK core has no Eino/Vivy imports. The Vivy internal `MemoryPort` maps its existing Session/Run/Journal/RunHook/Tool lifecycle. Bootstrap runs before model request; a bounded context-selection event records trace/references/hash/budget/degradation; terminal hooks enqueue capture after completion. `vivy.rpc.v1`, Face, ChannelHost, user-plugin ABI and outbound MCP client remain separate.

The initial public Go client is owned as a separate module at `C:/Users/Administrator/Desktop/garden/sdk/laputa-agent-go` (module path to be frozen as `github.com/dashimaki/laputa-agent-go`). This repository-root `sdk/` placement is accepted. It is not added to the `garden`, `laputa`, or `mentle` modules, and no `go.work` is introduced in Wave 0. Extraction to an independent repository is deferred until AGENT-VIVY plus a second black-box client pass conformance.

### EA-010 — Degradation distinguishes availability from authorization

Data-plane timeout/5xx/429 may produce a degraded Bootstrap/Search/Capture delivery result and must not block an already executable host Run. Authentication, capability denial, malformed request, version conflict, and authority CAS failures are deterministic fail-closed errors and are never retried.

### EA-011 — Protocol and route versioning are independent

`laputa-agent/1` is a semantic version line. `/v2` remains Garden domain routing. Additive optional manifest capabilities can ship within `/1`; behaviorally breaking meaning requires `/2` or a separately negotiated capability. No compatibility alias is introduced for retired Garden routes.

## 3. Persistence and audit map

| Data | Owner | Rule |
|---|---|---|
| Persona/ACTMEM | Laputa | Existing authority/CAS/review semantics unchanged |
| Frozen Core | Garden | Existing session-immutable SQLite snapshot |
| Selected-memory event | Host Journal + Garden trace | Bounded refs/hash/budget only; no duplicated sensitive full body |
| Capture receipt/status | Garden runtime store | Restart-safe, idempotent, auditable |
| Canonical memory/index work | Mentle facade | SQLite authority + transactional `index_jobs` |
| Host queue | Host runtime (e.g., Vivy) | Bounded retry spool; not a second memory authority |

## 4. Required public SDK shape

```go
type Client interface {
    Discover(ctx context.Context) (Manifest, error)
    Bind(binding AgentBinding) (*SessionHandle, error)
}

type SessionHandle interface {
    Bootstrap(ctx context.Context, req BootstrapRequest) (ContextView, error)
    Search(ctx context.Context, req SearchRequest) (SearchPage, error)
    Expand(ctx context.Context, req ExpandRequest) (EvidencePage, error)
    Capture(ctx context.Context, req CompletedTurn) (CaptureReceipt, error)
    Remember(ctx context.Context, req RememberRequest) (MemoryReceipt, error)
}
```

Persona/ACTMEM clients are explicit subclients; neither is called by Bootstrap.

## 5. Cross-epic verification invariants

1. Bind never creates or mutates a host Session/Run.
2. Bootstrap is frozen for an existing session across Persona revision changes/restart.
3. Automatic response schemas cannot carry WORLD/ACTMEM/history.
4. Capture cannot mutate canonical memory outside the Facade.
5. Agent principal cannot approve Persona review or repair index.
6. REST/MCP/SDK normalize one stable error result.
7. Host capture retry cannot change a terminal host Run result.
8. Public SDK imports no Garden/Vivy internal package or storage implementation.
