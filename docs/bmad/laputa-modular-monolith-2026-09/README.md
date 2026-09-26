# Laputa modular monolith — execution baseline

**Status:** implementation authorized; first Mentle embeddability slice in progress
**Authority:** [ADR-0016](../../architecture/0016-laputa-embeddable-modular-monolith.md), ADR-0012–0014. The earlier [REST-first SDK package](../laputa-external-agent-sdk-2026-09/) remains historical work and must be reconciled by owned hunks; it is not evidence of a shipped embedded backend.

## Ownership and isolation

- Writable Garden lane: `C:/Users/Administrator/Desktop/garden-modular-monolith`, branch `feat/laputa-modular-monolith`, base `1e402835912009fd19fed6442f56084e98a59b12`.
- Prior `C:/Users/Administrator/Desktop/garden-laputa-agent-sdk` worktree is dirty and **read-only** to this lane. No reset, stash, clean, implicit cherry-pick or parallel edits to its files.
- Vivy root is on an unrelated feature branch. Do not change it without its own new isolated worktree and a Garden conformance gate.
- No push, independent public release, data migration, authority semantics change or external network exposure in this batch.

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

## M1 acceptance

- `facade.Options` accepts explicit PalacePath and ModelsDir, with non-empty overrides taking precedence over config. Current `ConfigDir` callers keep existing behavior.
- An isolated host can select a separate palace without mutating its global config. Init/Close and canonical reopen work, without changing canonical schema or memory IDs.
- Existing Garden composition compiles without any call-site change.
- The `CGO_ENABLED=0` test remains a known blocker until M2; a green M1 test must not be described as a pure-Go or Vivy integration result.

## Release boundary

`laputa/`, `mentle/` and `garden/` are already nested Go modules in one Git repository. Module importability is not the same as publishability: independent versioned tags/module path support, model packaging and cross-module compatibility require explicit release evidence. Do not split repositories to make this wave appear complete.
