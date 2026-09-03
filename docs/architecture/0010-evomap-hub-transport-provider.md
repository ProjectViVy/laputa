# ADR-0010: EvoMap Hub Transport (GEP-A2A) Provider

**Status:** accepted  
**Date:** 2026-08-03  
**Refines:** ADR-0012 §5 (EvoMap capability boundary).
**Depends on:** ADR-0012 §5 (EvoMap capability boundary), ADR-0007 (mailbox privacy gates), ADR-0006 (bounded evidence).

---

## 1. Context

The previous design proposed a local Evolver MCP server or bounded CLI process and prohibited bundled Evolver code before license and supply-chain review. The read/write surface of the EvoMap Hub (GEP-A2A v1.0.0) has since been verified end-to-end against the live hub (2026-08-03: hello/heartbeat/fetch/search/validate/publish/report; see `docs/dev/reference/2026-08-03-evomap-gep-a2a-integration-research.md`).

Key findings that shape this decision:

- GEP-A2A is a **marketplace/signal-network protocol** (publish/fetch/search/report), not a synchronous evolver API — there is **no run lifecycle** on the hub.
- Asset management (`revoke`, `decision`, own-asset list) is **session-only**; a node_secret A2A client receives 401 and has no way to authenticate those calls.
- Publish is gated by Hub-side validation (Gene+Capsule pairs, content-addressed `asset_id = sha256(canonical JSON)`, required fields, non-trivial validation assertions).
- `search`/`search_summary`/`validate` are free; `fetch_full_content`/`advanced_search` consume credits.
- The GEP-A2A transport is a **pure HTTP protocol client** — it bundles no Evolver code, no third-party binaries, and adds no supply-chain surface beyond the Go standard library.

## 2. Decision: in-process GEP-A2A transport provider

Garden MAY implement the hub transport as an in-process `EvoMapProvider` inside `garden/internal/evolution`, rather than spawning a sidecar. The sidecar remains the future home for an actual local Evolver (code, models, sandboxed execution); the in-process provider is only the hub I/O layer.

### 2.1 Provider run model (hub has no run lifecycle)

A provider "run" is a **local record of one hub round trip**:

- **publish-mode** (requires `GARDEN_EVOMAP_HUB_PUBLISH=1` **and** `bundle.Policy.PublicationAllowed`): the bundle's boundary fields are materialized into a Gene+Capsule asset pair, `CanonicalHash`-addressed, `validate` → `publish`; the run status is the hub's published status (quarantine/candidate).
- **discovery-mode** (default): no publish; the run records the bundle signals, and `Candidates` returns hub search results for those signals.

`PollRun` is a local state read with an optional heartbeat refresh; it never pretends the hub has an asynchronous state machine. A run older than `ProviderLimits.MaxLifetime` that is still `running` is failed locally.

### 2.2 Boundaries and gates

- **Input boundary:** only `EvolutionEvidenceBundle` fields leave the process; the provider runs `CheckOutbound` (ADR-0007 §4 mechanical gate) before any network call and rejects leaky bundles with `ErrLeakageDetected`.
- **Hub publish:** disabled by default (`GARDEN_EVOMAP_HUB_PUBLISH` defaults to 0) and additionally requires `bundle.Policy.PublicationAllowed`. `HubPolicy.CanPublishHub` still requires explicit approval.
- **Hub fetch:** discovery uses the **free** `search` endpoint only. Paid `fetch_full_content` is reachable only through the manual CLI (`cmd/evomap`), never from the server path.
- **Session-only operations** (`revoke`, `decision`, own-asset management) are **never** called by Garden; the client does not implement them.
- **Secrets:** node credentials stay in `~/.evomap/node.json` (0600); `node_secret` is redacted from every error string and log line, and is never printed.
- **Degradation contract:** missing credentials or hub unreachability at startup yield a nil provider and `components["evolution"]="degraded"` — no fatal, no impact on other modules (fast recall, governance, reports all stay available).

### 2.3 Auto-registration

When credentials are absent, the provider may call `hello` at startup to register a node and print the `claim_url` (24 h window; free; binds the node to an EvoMap account). Controlled by `GARDEN_EVOMAP_AUTO_REGISTER` (default on).

## 3. Configuration

| Env var | Default | Meaning |
|---------|---------|---------|
| `GARDEN_EVOMAP_HUB_URL` | `https://evomap.ai` | Hub base URL (also used by tests) |
| `GARDEN_EVOMAP_CREDS` | `~/.evomap/node.json` | Credentials file path |
| `GARDEN_EVOMAP_HUB_PUBLISH` | `0` | Master switch for publish-mode runs |
| `GARDEN_EVOMAP_AUTO_REGISTER` | `1` | Auto-`hello` when no credentials exist |

## 4. HTTP Contract Addition

| Route | Behavior |
|-------|----------|
| `GET /v2/evolution/hub/status` | provider liveness: node_id, claimed, credit_balance, survival_status, last_heartbeat; 503 when provider is nil |

Existing `/v2/evolution/*` routes are unchanged; with a provider present they become functional (202 on start, run polling, proposals).

## 5. Non-goals

- No bundled/sidecar Evolver in this batch; a bounded process remains deferred.
- No `revoke`/`decision`/asset management via A2A (session-only; account page).
- No automatic credit spend; no scheduled background evolution.
- No change to `HubPolicy` semantics for proposals/host artifacts.

## 6. Test Strategy

All automated tests are hermetic against an `httptest` mock of the GEP-A2A surface (envelope shape, Bearer auth, canonical-hash + required-field validation, search/report, 401 for session-only ops). Live-hub verification is manual and recorded in the research document. Degradation paths (no creds, unreachable hub, MaxLifetime timeout, leaky bundle) are first-class tests.

## 7. Consequences

- `garden/cmd/evomap-demo` is promoted to the library (`internal/evolution`) and slimmed into `cmd/evomap`, a manual ops CLI (the only path to paid `fetch_full_content`).
- `evolution` component can become `ok` on this machine (credentials already exist at `~/.evomap/node.json`).
- In-process transport is allowed; a sidecar is reserved for actual Evolver execution.
