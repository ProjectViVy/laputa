# ADR-0016: Embeddable Laputa Libraries and Modular Monolith

**Status:** accepted direction; M1–M3 implemented in part, M4–M5 gated
**Date:** 2026-09-27
**Decision owner:** project owner
**Refines:** ADR-0012, ADR-0013, ADR-0014
**Supersedes in ADR-0015:** REST as the sole canonical *in-process* access path and the Vivy default transport decision. The external `laputa-agent/1` REST profile remains a supported transport contract.
**Motivation:** [Issue #1 — Detachable mentle](https://github.com/ProjectViVy/laputa/issues/1)

## Decision

Build one domain stack in three importable libraries plus a single distributable Laputa application:

| Unit | Authority or responsibility | Forbidden shortcut |
| --- | --- | --- |
| `laputa/` | Persona Markdown authority, review/history/CAS, ACTMEM | Direct authority mutation by an adapter |
| `mentle/` | Canonical SQLite memory, derived index outbox, bounded cards/evidence | JSONL WAL, vector store or host Journal as second memory authority |
| `garden/` | Frozen Core capture, bounded recall, activity/capture orchestration, capability and profile policy, domain contracts | Automatic WORLD/ACTMEM projection or HTTP-only business policy |
| Laputa application | Compose those libraries; serve REST, MCP and Console | Fork domain semantics into a parallel backend |

The repository already has three Go modules. `garden/main.go` currently composes services while much of the runtime is under `garden/internal/`. Expose only the small stable domain/composition boundary needed by Go hosts; keep internal storage and domain mechanics private. The CLI/application package name must not become a second Persona authority named Laputa. The application remains a valid all-in-one deployment for non-Go consumers.

A Go host such as AGENT-VIVY may use a first-party, in-process `MemoryPort` adapter to the same composed domain services. Vivy owns Session/Run/Journal and terminal-event identity; Laputa/Mentle retain separate on-disk authority and capture idempotency. `std/context-source@v1` supplies read candidates only; terminal capture and explicit protected tools require distinct lifecycle/capability paths. Do not inject model-visible MCP for bootstrap or capture.

External REST and MCP remain supported adapters. One domain contract (DTO/error, principal/binding, budgets, idempotency and capability semantics) governs both in-process and protocol calls. The REST endpoint is the canonical **wire profile**, not the sole source of business rules. An in-process caller may not bypass the same policy by omitting HTTP headers: its principal and profile must be bound by the trusted host composition, never asserted by model content. No implicit loopback-read privilege applies to the embedded adapter.

## Migration and compatibility boundary

1. Freeze a behavioral conformance corpus against current authority, recall, capture, errors and negative WORLD/ACTMEM paths. Keep existing HTTP consumers working; do not migrate Persona or canonical memory data.
2. Make Mentle explicit-path initialization embeddable without changing the canonical catalog, outbox, retrieval or existing default configuration. Address model injection/offline mode and CGO SQLite as separate tested gates; no claim of pure-Go compatibility until `CGO_ENABLED=0` tests pass.
3. Extract Garden domain composition from `main.go`/`internal/` behind a public minimal entrypoint, without leaking raw Persona, vector store or database handles. Make the application call that entrypoint rather than owning alternate logic.
4. Implement Vivy embedded provider only after the same profile, six-slot Frozen Core, budget, capture/replay and denial fixtures pass in-process and through REST. Preserve the existing Vivy Run/Journal lifecycle and its data directory ownership.
5. Decide Vivy default mode from conformance and observed startup/latency/operability measurements. Until then, no change to its shipped provider default is claimed. Document migration of any in-flight ADR-0015 branch as explicit owned hunks, never overwrite or merge dirty worktrees implicitly.

The original `feat/laputa-agent-sdk` worktree has uncommitted changes and is **not** part of this branch; its binding/capture tests are candidate evidence, not already integrated code. The root Garden working tree also has unrelated edits. Neither tree may be reset, cleaned or rebased for this work without a separate review.

## Implementation snapshot (2026-09-27)

The dedicated `feat/laputa-modular-monolith` worktree now contains explicit Mentle palace/model paths, strict local-model startup, a pure-Go SQLite driver for Mentle and Garden state stores, and a fail-closed BM25-only search path. Garden has an importable `garden/agentapi` client with trusted single-profile identity, bound sessions, bounded bootstrap/recall, explicit reads, durable terminal capture and status; `internal/runtimecore` is shared by the monolith. Existing HTTP recall and session-ingest paths route through shared services. An independent local-replacement Go consumer exercises offline `CGO_ENABLED=0` open/bind/recall/capture/replay/restart. This is **local importability**, not remote module publication or Vivy integration.

The REST↔embedded fixtures cover automatic context, explicit WORLD/ACTMEM, session freezing, budgets and principal denial. MCP tests preserve backend error envelopes and pin its existing tools, but MCP is not yet a cross-transport bootstrap/capture implementation. No new Agent wire profile or Vivy default switch is claimed. Current full-suite and outstanding transport gates are recorded in the [execution handoff](../bmad/laputa-modular-monolith-2026-09/README.md) and [conformance inventory](../bmad/laputa-modular-monolith-2026-09/conformance.md); a partial test pass is not a release gate.

## Non-negotiable conformance gates

- DIVA parity for Persona/ACTMEM; seven authority files and explicit-only WORLD/ACTMEM; exact CAS/review semantics.
- Mentle canonical SQLite is the sole memory authority; every accepted mutation and `index_jobs` outbox entry share one transaction; derived indexes are rebuildable.
- Both transports bind one server-authorized profile and principal; agents cannot self-grant operator/user capabilities or approve Persona reviews.
- Captures use stable terminal event identity and content hash; identical replay is idempotent and changed content conflicts. Timeout or derived-index lag never fabricates a failed canonical commit.
- Equivalent in-process, REST and MCP results/errors on shared positive and negative fixtures, including restart and degraded-mode behavior. A proposed transport advertises only what its executable conformance tests prove.

## First executable slice

In a dedicated worktree, add explicit `PalacePath` and `ModelsDir` to Mentle facade initialization with tests first, preserving the legacy `ConfigDir` path and canonical behavior. This is a prerequisite, not completion of the new deployment architecture. No Vivy or Garden application switchover is authorized by that slice alone.
