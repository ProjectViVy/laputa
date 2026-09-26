# ADR-0015: Laputa External Agent Contract and Lifecycle SDK

**Status:** accepted; Vivy default transport and in-process access precedence refined by [ADR-0016](0016-laputa-embeddable-modular-monolith.md)
**Date:** 2026-09-04  
**Decision owner:** project owner  
**Accepted by:** owner authorization on 2026-09-04  
**Depends on:** ADR-0012, ADR-0013, ADR-0014  
**Planned by:** `docs/bmad/laputa-external-agent-sdk-2026-09/`

## Context

The completed Garden Authority & Recovery batch made Persona, ACTMEM, Frozen Core, Mentle canonical memory, REST, and `garden-mcp` domain-consistent. It deliberately left host lifecycle adapters deferred.

External Agents still lack one explicit, versioned contract for capability discovery, authenticated host identity binding, bounded bootstrap, progressive memory retrieval, terminal-turn capture, degradation, and REST/MCP equivalence. Existing surfaces are not substitutes:

- Garden `/v2/*` is a domain API, not yet a frozen external-Agent profile;
- `garden-mcp` is a thin loopback adapter but lacks a lifecycle/capture contract and contract discovery;
- AGENT-VIVY owns Session, Run, Journal, Policy, Tool, and local `vivy.rpc.v1`, but it is not a Laputa client SDK;
- raw Persona Markdown and Mentle storage are authorities, never external SDK storage APIs.

TencentDB-Agent-Memory demonstrates host-adapter separation, bounded recall/capture/tools/degradation responsibilities, pending writes, and identity-aware lifecycle integration. memsearch demonstrates source-first derived indexing and Search → Expand → Source progressive disclosure. Neither project defines Laputa authority semantics.

## Decision

### 1. External-access plane has three non-interchangeable layers

```text
laputa-agent/1 contract
  ├── canonical Garden REST profile
  ├── thin garden-mcp adapter
  └── native host lifecycle adapters
        └── first reference: AGENT-VIVY MemoryPort (transport revised in ADR-0016)
```

REST is the canonical external wire profile. MCP maps its tools to the same domain result/error contract. Under ADR-0016, the same domain services also have a policy-equivalent in-process composition for Go hosts; a native host adapter maps existing lifecycle identity without manufacturing a second Session or Run model.

### 2. Freeze a scoped `laputa-agent/1` contract

Discovery is exposed at `GET /.well-known/laputa-agent.json`. It publishes protocol version, supported transports, capabilities, principal requirements, budgets, automatic-vs-explicit context policy, and stable error codes. It contains no secret, filesystem path, or authority content.

The initial semantic surface is:

```text
Discover
Bind(existing authenticated host identity)
Bootstrap
Search → Expand
Capture
Remember
Persona.Get / Persona.Propose
ACTMEM.Read / ACTMEM.Query
```

The contract is a selected profile over existing `/v2` domain resources. It does not declare every Garden route a public SDK method. HTTP route versions and `laputa-agent/1` protocol versions evolve independently.

### 3. Bind host identity; do not trust asserted authorization

A binding includes `profile_id`, `agent_id`, `platform`, `session_id`, `turn_id`, and `event_id` as routing/audit/provenance values. Principal and capabilities are established by the authenticated server-side credential. `X-Garden-Actor` remains audit metadata and cannot grant permissions.

`laputa-agent/1` v1 is a **single-service, single-profile** deployment contract. Garden configures one canonical profile identifier (default `default`); a request-supplied `profile_id` must exactly match it and cannot select another authority directory. A valid Agent bearer token establishes the Agent principal, not the profile. Multi-profile token-to-profile mapping is deferred rather than simulated with a client assertion.

`Bind` creates an immutable client-side session handle. Garden validates that envelope on every Agent-profile request; it does not create a second service-side Session or Run, and it does not let an Agent choose another profile.

### 4. Preserve automatic-versus-explicit context boundaries

Automatic Bootstrap may contain only:

- the session-frozen six-slot Frozen Core; and
- caller-requested bounded Mentle cards/evidence.

The following are explicit, capability-gated tools only: full Persona including WORLD, Persona history/reviews, ACTMEM, raw evidence/source, activity, diagnostics, and mutations. WORLD, ACTMEM, history, raw profile paths, and unbounded tails are structurally excluded from automatic context.

### 5. Capture is terminal, asynchronous, durable and idempotent

Host adapters invoke capture only after the host Run reaches a terminal event. Capture carries stable `session_id`, `turn_id`, `event_id`, `content_hash`, phase, bounded content/provenance, and idempotency metadata.

Garden returns an accepted receipt/status handle when processing is asynchronous. Network timeout, 429, and 5xx may enter a bounded durable retry queue; deterministic 4xx must not retry. Capture failure must not reverse a completed host Run. Canonical memory writes still go through Mentle `facade.Service`; derived-index lag is surfaced as `index_pending`, never as a false canonical failure.

### 6. Persona and ACTMEM retain domain-specific authority rules

An Agent may create Persona proposals when its capability allows it, but cannot approve/reject review or direct-write protected Persona authority. P16 remains limited by DIVA parity rules. ACTMEM maintenance remains separately capability-gated and is not a side effect of capture or recall.

### 7. First-party conformance host is AGENT-VIVY

AGENT-VIVY is the first-party conformance host. Its target integration is an internal `MemoryPort` provider, not its user-plugin ABI; no live Vivy provider is claimed by this ADR. This ADR originally selected `laputa-sdk-go` over Garden REST as the default transport. ADR-0016 replaces that default-only decision with an embeddable-domain target; Vivy's shipped default is not changed until shared conformance and operational evidence establish it. Neither mode uses model-visible MCP for pre-run bootstrap or terminal capture. The original REST adapter sketch remains a supported external-transport option:

```text
agent-vivy/internal/memory/
  port.go
  laputa/client.go
  laputa/lifecycle.go
  laputa/capture_queue.go
  laputa/tools.go
```

It maps existing Vivy `SessionID`, `RunID`, Journal provenance and terminal `RunHook` events. Bootstrap occurs before `model.request`; selected-memory provenance is journaled without duplicating sensitive bodies. Capture is delivered from a restart-safe queue. Neither Eino, Vivy `internal/*`, Vivy JSON-RPC, nor Garden/Mentle storage internals become part of the public SDK.

## 8. Compatibility scope is external access only

Compatibility is limited to external Agent adoption: existing loopback Garden domain routes and capability-token mechanisms remain available for their current consumers while `laputa-agent/1` is an opt-in Agent profile. This does **not** relax ADR-0012/0014 clean-break rules: no JSON Persona authority, old/new authority mapping, alias/fallback authority reads, dual-write, authority migration, or derived store promoted to canonical state.

## Consequences

### Positive

- External Agent integrations get stable discovery, typed errors, lifecycle semantics, and conformance gates.
- Garden retains one authority per domain and one canonical memory write path.
- REST, MCP, and first-party host adapters can be verified against the same behavior.
- AGENT-VIVY validates a real lifecycle integration without becoming a universal framework dependency.

### Costs

- Discovery, session/capture status, typed error normalization, and conformance tests must be implemented before claiming external-Agent compatibility.
- A public typed client needs strict versioning and backward-compatibility discipline.
- Host adapters need a durable queue and auditable context-selection events.

## Rejected alternatives

- Make `garden-mcp` the only external API: it cannot perform invisible pre-run bootstrap and terminal capture lifecycle safely.
- Expose raw Persona/profile Markdown paths: bypasses authority policy, CAS, review, and security boundaries.
- Make Milvus/vector/BM25/JSONL a client authority: violates ADR-0014.
- Export AGENT-VIVY internals or its `vivy.rpc.v1` as the public Laputa SDK: couples clients to a private host runtime/control plane.
- Copy TencentDB L0–L3 automatic Persona mutation: violates Laputa review/CAS authority.
- Let memsearch scan whole Persona/profile trees: violates explicit tool and capability boundaries.

## Acceptance gates

1. Manifest capability/version negotiation is machine-readable and secret-free.
2. Profile/agent/session binding is authenticated and immutable for a handle.
3. Bootstrap contains exactly the allowed automatic context; negative tests prove WORLD/ACTMEM/history absence.
4. Search → Expand preserves source, canonical revision/status, score, and trace provenance.
5. Identical `event_id + content_hash` capture replay is idempotent; changed content conflicts.
6. Timeout/5xx/429 degrades a host Run without blocking it; deterministic client errors do not retry.
7. Persona/ACTMEM capability matrix and DIVA CAS/review semantics are preserved.
8. REST, SDK, and MCP return equivalent domain results and stable error codes.
9. Frozen Core, capture status, and derived-index recovery survive restart.
10. SDK exposes no authority filesystem path, raw database handle, or host-runtime internal type.
