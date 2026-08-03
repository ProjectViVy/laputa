<!-- Parent: ../AGENTS.md -->

# garden/internal/evolution — Evolution Runs and EvoMap Hub Transport

**Generated:** 2026-08-03  
**Purpose:** Evolution runs, proposals, and the in-process GEP-A2A hub transport (ADR-0010)

---

## Purpose

The `evolution/` package owns the bounded Evolver integration:

- **Run lifecycle** — `Service.StartRun/GetRun/ListRuns` over an `EvolverProvider`, run store, event chain
- **Proposals** — candidate → `NormalizeCandidate` leakage gate → user review; `ListProposals` for console enumeration
- **Hub transport (ADR-0010)** — `HubClient` (GEP-A2A v1.0.0) + `EvoMapProvider`, the in-process hub I/O layer
- **Policy** — `HubPolicy` (hub publish disabled by default), `CheckOutbound` mechanical privacy gate (ADR-0007 §4)

---

## Structure

```
evolution/
├── adapter.go       # EvolverProvider interface, ProviderLimits, RunStatus
├── bundle.go        # EvolutionEvidenceBundle, GeneCandidate, proposals
├── service.go       # Run/proposal/event orchestration, HubStatus
├── store.go         # evolution_runs/proposals/candidates + evomap_runs tables
├── policy.go        # HubPolicy, bundle validation, CheckOutbound
├── normalize.go     # NormalizeCandidate leakage report
├── events.go        # EventStore
├── hub.go           # GEP-A2A HubClient (hello/heartbeat/search/fetch/validate/publish/report)
├── evomap.go        # EvoMapProvider (EvolverProvider implementation)
└── hubtest/         # Hermetic mock of the GEP-A2A surface (tests only)
```

---

## Key Concepts

### Provider run model (ADR-0010 §2.1)

The EvoMap Hub is a signal marketplace, not a synchronous evolver — **there is no run lifecycle on the hub**. A provider run is a local record of one hub round trip (`evomap_runs` table):

- **publish-mode** — `GARDEN_EVOMAP_HUB_PUBLISH=1` **and** `bundle.Policy.PublicationAllowed`; the bundle's boundary fields are materialized into a Gene+Capsule pair, `CanonicalHash`-addressed, `validate` → `publish`. Run id: `evomap_<hash>`.
- **discovery-mode** (default) — no publish; candidates come from the free `search` endpoint by bundle trigger. Run id: `evomap_disc_<uuid>`.

`PollRun` is a local state read with an optional heartbeat refresh; a `running` run older than `ProviderLimits.MaxLifetime` is failed locally.

### Gates

- Outbound payloads always pass `CheckOutbound` (prohibited evidence refs / keys) before any network call → `ErrLeakageDetected`.
- The provider never calls session-only endpoints (`revoke`, `decision` — account-page operations, verified 401 for node_secret clients).
- The paid `fetch_full_content` tier is reachable only through `cmd/evomap`, never from the server path.
- `node_secret` is redacted from every error string (`redactSecret`); credentials file is written 0600.

### Degradation

Missing credentials or an unreachable hub → provider is nil → `components["evolution"]="degraded"`, evolution endpoints return 503 (`ErrProviderUnavailable`). No fatal, no impact on other modules.

---

## Testing

```bash
cd garden
GOSUMDB=off go test -v ./internal/evolution/...
```

All hub tests run against the hermetic `hubtest` mock (envelope shape, Bearer auth, canonical-hash + required-field validation, 401 for session-only ops). The live hub is exercised only manually:

```bash
cd garden && GOSUMDB=off go run ./cmd/evomap heartbeat
cd garden && GOSUMDB=off go run ./cmd/evomap search "<signal>"
```

**Behavioral tests:**

- Hello registers once and saves credentials; re-registration is a no-op
- Unauthorized (401) and unreachable-hub errors map to typed errors; error strings never contain the node_secret
- Publish-mode runs validate → publish a deterministic Gene+Capsule pair; bundle policy blocks publish
- Leaky bundles are rejected before any network call
- Discovery-mode candidates map hub search results (confidence from GDI score); publish-mode includes the own capsule
- Hub loss after a run → local fallback (degraded), no error
- MaxLifetime timeout fails stale runs
- `Service.HubStatus` reports provider liveness or `ErrProviderUnavailable`

---

## Conventions

- Hub protocol details (envelope, asset schema, canonical hash) live in `docs/dev/reference/2026-08-03-evomap-gep-a2a-integration-research.md`
- New A2A endpoint support requires a mock-hub counterpart in `hubtest/` first
- Never log or print `node_secret`; use `redactSecret` when wrapping hub errors

Parent reference: ../AGENTS.md
