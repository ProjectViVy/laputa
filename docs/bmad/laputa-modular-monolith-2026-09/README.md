# Laputa modular monolith — execution baseline

**Status:** implementation underway; local embedded consumer works, cross-transport/release gates open
**Authority:** [ADR-0016](../../architecture/0016-laputa-embeddable-modular-monolith.md), ADR-0012–0014. The earlier [REST-first SDK package](../laputa-external-agent-sdk-2026-09/) remains historical work and must be reconciled by owned hunks; it is not evidence of a shipped embedded backend.

## Ownership and isolation

- Writable Garden lane: `C:/Users/Administrator/Desktop/garden-modular-monolith`, branch `feat/laputa-modular-monolith`, base `1e402835912009fd19fed6442f56084e98a59b12`.
- Prior `C:/Users/Administrator/Desktop/garden-laputa-agent-sdk` worktree is dirty and **read-only** to this lane. No reset, stash, clean, implicit cherry-pick or parallel edits to its files.
- Vivy root is on an unrelated feature branch. Do not change it without its own new isolated worktree and a Garden conformance gate.
- No push, independent public release, data migration, authority semantics change or external network exposure in this batch.
- [Cross-transport conformance inventory](conformance.md) records existing regression anchors and open gates; it is not proof that embedded mode passes.

## Dependency-driven waves

| Wave | Deliverable | Required evidence and stop gate |
| --- | --- | --- |
| M0 | ADR-0016 and exact baseline/conformance inventory | Policy ownership frozen; existing REST/MCP claims distinguished from roadmap. |
| M1 | Mentle facade explicit config/path injection | RED→GREEN tests for explicit PalacePath/ModelsDir and unchanged ConfigDir/default behavior; canonical SQLite/outbox intact; `go test ./...`. |
| M2 | Pluggable embedding/offline mode and pure-Go storage driver | Tests for lexical-only, identity mismatch, startup offline, existing canonical catalog/KG reopen, transaction/index-job restart; `CGO_ENABLED=0` full module gate. No claimed pure-Go support before this passes. |
| M3 | Export minimal Garden domain composition (Frozen Core, bounded recall, policy, capture), wire app via it | Same-domain in-process and HTTP fixtures; six-slot frozen session, protected context absent, capability denial, capture retry/idempotency; Garden Go tests/e2e and Console build prerequisite. |
| M4 | Keep Laputa app REST/MCP/Console as adapters; retire duplicate handler logic | Black-box embedded/REST/MCP result/error equivalence and restart proof. No speculative capability advertisement. |
| M5 | Vivy first-party MemoryPort adapter in separate worktree | Reuse Session/Run/Journal and terminal RunHook; bootstrap before model request, no double Session/authority; full Vivy `just ci` and live path; default deployment decision only after measurements. |

Each wave has its own reviewed commit. No wave may mark a later one complete because packages happen to compile.

## Implementation handoff (2026-09-27)

| Wave | Current evidence | Remaining gate |
| --- | --- | --- |
| M1 | Explicit `PalacePath`/`ModelsDir` and ambient-config isolation tests landed in Mentle; existing config and default paths retained. | Final tree CGO=0 and default-CGO full tests passed; remote import/tag remains a release decision. |
| M2 | Pure-Go SQLite changes landed in Mentle canonical/KG and Garden state stores, with reopen/migration/restart tests. Strict local-model startup and fail-closed BM25-only mode landed. | Mentle and Garden CGO=0 and default full suites, plus CGO=0 vet passed; backend switch/derived-index reconciliation across transports remains a separate contract gate. |
| M3 | Public `garden/agentapi` Open/BindSession client, shared `internal/runtimecore`, policy, bounded recall, explicit reads and durable terminal capture/status landed. Garden main, HTTP recall and legacy session ingest call shared services. The independent `examples/vivy-embed-smoke` module uses local `replace` directives and passed an offline pure-Go run. | Garden full CGO=0/default tests, e2e, independent consumer and selected five-package race test passed. `Start` SELECT/Scan retry and Close tests characterize existing behavior; no dedicated rows.Err fault injection. |
| M4 | REST↔embedded focused fixtures exist for context/denial/freeze/budget and terminal capture. MCP pins its existing tools and now retains both structured backend envelopes on dual `memory_search` failure. | No complete embedded↔REST↔MCP black-box corpus or authenticated Agent-profile wire coverage. Do not advertise unsupported bootstrap/capture. |
| M5 | No Vivy source changed. The example is a Vivy-*style* Go consumer only. | Separate Vivy worktree, MemoryPort/RunHook integration, `just ci`, live path and deployment decision remain open. |

Verification ledger (final code tree before documentation-only edits): Mentle, Laputa and Garden each passed `CGO_ENABLED=0 GOSUMDB=off go test ./... -count=1`, CGO=0 `go vet ./...`, and default-CGO full tests. Garden e2e passed with CGO=0; race passed for `agentapi`, ingest, runtimecore, server and garden-mcp. Independent consumer passed CGO=0 `go test`, `go run`, `go vet`, `go build`; `go run` printed `PASS: external consumer; offline pure-Go open/bind/bootstrap/recall/explicit WORLD/capture/replay/restart/close`, and `go list -deps` excluded Garden server/Console/command packages. Earlier `core.Ingest.Writer` compilation and lexical-only failures were transient concurrent-edit snapshots, not current blockers. This does not prove a released remote module, full Agent-profile conformance or Vivy integration.

Handoff boundaries: do not edit Vivy or the old SDK worktree, publish modules, or advertise unproven Agent/MCP capabilities. Keep documentation changes in a separately reviewed commit from any final code fix.

## M1 acceptance

- `facade.Options` accepts explicit PalacePath and ModelsDir, with non-empty overrides taking precedence over config. Current `ConfigDir` callers keep existing behavior.
- An isolated host can select a separate palace without mutating its global config. Init/Close and canonical reopen work, without changing canonical schema or memory IDs.
- Existing Garden composition compiles without any call-site change.
- M1 alone never established pure-Go or Vivy integration. Pure-Go driver changes now pass the Mentle CGO=0 full-module gate; this does not imply a Vivy integration.

## Release boundary

`laputa/`, `mentle/` and `garden/` are already nested Go modules in one Git repository. Module importability is not the same as publishability: independent versioned tags/module path support, model packaging and cross-module compatibility require explicit release evidence. Do not split repositories to make this wave appear complete.
