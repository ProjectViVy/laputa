# Standalone Garden v1: source-grounded research handoff

**Stage:** 1 research only · **Checked:** 2026-10-12 · **Status:** evidence and decisions for the separate Stage 2 architecture session; not an accepted implementation specification. No runtime code was changed.

## Executive summary

Laputa already has a useful base for a standalone Garden: the Garden runtime composes the existing Laputa Persona and ACTMEM authorities, has a scoped memory `Backend` port, exposes an in-process `agentapi`, and ships one Mentle adapter, an HTTP console, and a stdio MCP process. The public scope model already names a `SubjectID`, scopes records by exact subject/workspace tuples, and requires Search, Expand, Mutate, and MutationLookup from a selected primary writer.

The present product is still single-profile. Startup opens one persona directory and one ACTMEM store. There is no persona registry, cross-subject grant store, end-user backend selector, persisted first-launch wizard, or external-host attachment manager. The only production backend assembled by the ordinary Garden executable is Mentle. VIVY has a real separate BML memory module; DIVA's current Garden composition binds one `diva` profile and Mentle. Neither existing path proves that the two stores can be combined under one authority. BML records have tenant/workspace/session scope fields, but no persona subject field; its machine-home database is bound to one workspace. Do not copy or dual-write those stores to make the UI appear unified.

There is also a material route split: the newer `agentapi` path scopes search and source expansion from a bound identity, while legacy `/v2/memories` CRUD, `/v2/recall/deep`, MCP memory calls, and collection reads still use Mentle-specific services. Garden must own authorization and all CRUD, search, and source expansion through one domain boundary before multi-subject isolation can be claimed.

The lowest-overhead direction to test in Stage 2 is one standalone Garden service and console, reusing Laputa's identity and ACTMEM owners, one selected canonical memory authority, and thin per-host adapters. Keep the adapters opt-in. Use the Tencent project as a functional benchmark for management and integrations, not as an architecture or data-model template. OpenClaw and Hermes expose useful native file and lifecycle surfaces; Claude Code and Codex CLI have instruction-file and hook surfaces that can be researched after the first two integrations. A new general LLM proxy, provider marketplace, team/role system, or L0–L3 memory taxonomy is not needed to meet the confirmed requirements.

## Evidence baseline

The three current remote `main` refs were fetched and checked with `git ls-remote` on 2026-10-12. These are the baselines used below; older pins in archived plans are historical evidence, not current heads.

| Checkout | Remote main commit | Relevant current evidence |
|---|---|---|
| [ProjectViVy/laputa](https://github.com/ProjectViVy/laputa/tree/f44ba00eb6ab5aa6ab9a2f5c09868373e36679f5) | `f44ba00eb6ab5aa6ab9a2f5c09868373e36679f5` | ADR-0018 is accepted as owner policy, with runtime implementation still pending. Garden is a single-profile composition with an injectable backend port and one shipped Mentle adapter. |
| [ProjectViVy/agent-vivy](https://github.com/ProjectViVy/agent-vivy/tree/65b561f6f8189dc36dd29120a6f8a5f20b7071a6) | `65b561f6f8189dc36dd29120a6f8a5f20b7071a6` | VIVY's BML module and its memory context/observer provider are wired into the module catalog. The current BML home is machine-wide within the VIVY data directory, not a Garden persona registry. |
| [ProjectViVy/agent-diva](https://github.com/ProjectViVy/agent-diva/tree/0a64d1ce152440f1f63d4224a155c4616d2aff5b) | `0a64d1ce152440f1f63d4224a155c4616d2aff5b` | The DIVA cognitive factory opens one Garden owner, fixes its profile to `diva`, and uses Mentle as its destination. Its detailed P0 and memory-loop plans are not product acceptance. |

No `.agents/skills` checkout directory was present in these three repos. Repository `AGENTS.md` and nested Garden/docs instructions were read and applied. The Laputa work branch was created from the fetched `origin/main`; at the time of inspection no remote branch named `feat/garden-standalone-v1` existed.

### Laputa source map

- [ADR-0018: unified context and Recap policy](../architecture/0018-vivy-unified-context-and-recap-policy.md) — accepted policy; no runtime/API/schema/backend migration implemented by that document.
- [ADR-0016: embeddable modular monolith](../architecture/0016-laputa-embeddable-modular-monolith.md) — composition and public package boundary.
- [`garden/memory/contract.go`](../../garden/memory/contract.go) and [`garden/memory/scope.go`](../../garden/memory/scope.go) — backend capability contract, authorized DTOs, exact scope encoding.
- [`garden/internal/runtimecore/runtime.go`](../../garden/internal/runtimecore/runtime.go), [`garden/main.go`](../../garden/main.go), [`garden/agentapi/reads.go`](../../garden/agentapi/reads.go), [`garden/agentapi/policy.go`](../../garden/agentapi/policy.go) — current runtime and API composition.
- [`garden/internal/server/server.go`](../../garden/internal/server/server.go), [`garden/internal/server/recall_handlers.go`](../../garden/internal/server/recall_handlers.go), [`garden/internal/server/material_handlers.go`](../../garden/internal/server/material_handlers.go) — HTTP route families and where they enter the domain.
- [`garden/backends/mentle/adapter.go`](../../garden/backends/mentle/adapter.go), [`garden/cmd/garden-mcp/main.go`](../../garden/cmd/garden-mcp/main.go) — only shipped backend adapter and existing MCP client-facing process.
- [`garden/console/src/App.tsx`](../../garden/console/src/App.tsx), [`PersonaPage.tsx`](../../garden/console/src/modules/persona/PersonaPage.tsx), [`MemoryPage.tsx`](../../garden/console/src/modules/memory/MemoryPage.tsx), [`SettingsPage.tsx`](../../garden/console/src/pages/SettingsPage.tsx) — existing UI surfaces.

### VIVY and DIVA source map

- [VIVY unified context/memory policy](https://github.com/ProjectViVy/agent-vivy/blob/65b561f6f8189dc36dd29120a6f8a5f20b7071a6/docs/architecture/VIVY-UNIFIED-CONTEXT-MEMORY.md) and [host contracts](https://github.com/ProjectViVy/agent-vivy/blob/65b561f6f8189dc36dd29120a6f8a5f20b7071a6/docs/architecture/VIVY-MEMORY-HOST-CONTRACTS.md).
- [BML record model](https://github.com/ProjectViVy/agent-vivy/blob/65b561f6f8189dc36dd29120a6f8a5f20b7071a6/bml/record.go), [BML home](https://github.com/ProjectViVy/agent-vivy/blob/65b561f6f8189dc36dd29120a6f8a5f20b7071a6/bml/home.go), [VIVY memory provider](https://github.com/ProjectViVy/agent-vivy/blob/65b561f6f8189dc36dd29120a6f8a5f20b7071a6/internal/modules/memory/provider.go), and [module wiring](https://github.com/ProjectViVy/agent-vivy/blob/65b561f6f8189dc36dd29120a6f8a5f20b7071a6/internal/modules/defaults/catalog.go).
- [DIVA P0 design](https://github.com/ProjectViVy/agent-diva/blob/0a64d1ce152440f1f63d4224a155c4616d2aff5b/docs/plans/diva-next/p0-design.md), [memory-loop plan/status](https://github.com/ProjectViVy/agent-diva/blob/0a64d1ce152440f1f63d4224a155c4616d2aff5b/docs/plans/diva-next/memory-loop/README.md), and [bundled VIVY config](https://github.com/ProjectViVy/agent-diva/blob/0a64d1ce152440f1f63d4224a155c4616d2aff5b/agent-diva-gui/src-tauri/resources/vivy.default.yaml).

## Requirements carried into Stage 2

These are confirmed owner requirements, not recommendations introduced by this report.

1. Standalone Garden supports multiple independent persona subjects in one canonical memory authority, private by default with explicit sharing. Garden shows and edits the existing VIVY persona through the one authoritative identity source. VIVY itself remains a single persona unless separately decided.
2. Persistent sessions of the same persona subject share one subject-level ACTMEM Recap. Persona-free delegated workers are bounded execution contexts, not persona subjects. Persona subject, authenticated client principal, runtime agent, host/workspace, and session must remain distinct.
3. Garden owns authorization plus memory CRUD, search, and bounded source expansion. Keep a pluggable memory backend. Preserve the current BML-on-VIVY and Mentle-through-DIVA paths for initial validation; add no default dual write or secondary authority.
4. Match TencentDB-Agent-Memory's user-facing management and integration experience functionally, without Tencent API or wire compatibility. Prioritize OpenClaw and Hermes. Research supported Claude Code and Codex CLI persona/context and memory surfaces. Garden's first-class UI includes subject registry/editor, the actual loaded memory and Recap, private/shared permissions, bindings and revocation, provider health, source lineage, and edits.
5. Per external host instance, explicit opt-in is required before Garden manages designated persona config files or distributes updates. Passive discovery or installation does not enable a connection, hook, grant, recall, or capture. Persona sync, memory recall, and experience capture have independent controls. Preserve existing instructions; use conflict checks and backups; distinguish saved, written, and loaded revisions; describe host reload delays honestly. A revoke affects future Garden-authorized reads; it cannot erase context already supplied to a running host.
6. Defer and unwire autonomous evolution that changes persona, rules, or skills. A selected experience may become a usable fact with provenance. Persona edits and user-approved distribution remain independent of autonomous evolution. The frozen layer remains under human authority.
7. Ship a real persisted first-launch wizard with resume and no repeat-on-boot behavior; upgrades retain existing data. DeepSeek is the only initial test model provider, with minimal API-key setup and host-side secret handling. Offer authorized discovered VIVY/Garden data, create a persona, or skip. A different store or machine requires explicit attach/import decisions. Host setup is optional and off by default.
8. Follow 多快好省: few components, low overhead, little maintained code. Real cross-persona isolation is required correctness, not an optional enterprise feature. Inventory and remove superseded runtime paths during implementation; retain no compatibility fallback without a concrete caller. Stage 1 does not delete runtime code.

## Current state versus required product

### Garden identity and scope

`runtimecore.Config` accepts one `PersonaDir`, one `ProfileID`, and an optional `Backends` map. `Open` loads one Laputa Persona service and one ACTMEM store from that persona directory. The normal executable assembles that one profile. `ProfileID` is a binding label, not a selector for multiple persona authorities. The DIVA factory likewise fixes `profileID = "diva"` and opens one owner.

The memory scope type already includes `SubjectID` and `WorkspaceID`. The stored `scope/v1` value is an exact tuple; malformed or empty values stay invisible. The Backend contract states that instances are bound to a subject/write destination and caller-supplied scope does not grant authority. These are useful isolation primitives. They are not a persona registry or a sharing system: `agentapi.Binding.TrustedScope()` is derived from the current single profile/workspace, and no persistent cross-subject grant/revocation protocol is exposed. The internal injected backend map can bind a foreign scope, but injecting one is an internal composition operation, not a user-ready provider selection or authorization decision.

Every new API must keep these concepts separate:

| Concept | Current evidence | Garden v1 meaning to settle |
|---|---|---|
| Persona subject | Laputa Persona authority; runtime currently has one profile | Stable ID for a separately managed personality and its private data. |
| Authenticated principal | Garden capability/auth contract | Human or client allowed to perform an operation; it is not the persona ID. |
| Runtime agent | `Binding.AgentID` and host metadata | Adapter/runtime provenance. Current `AgentID` is not an authorization grant. |
| Host binding | Not currently a Garden registry | Explicit mapping from an opted-in host instance to a persona subject and allowed operations. |
| Workspace | Optional exact Garden scope | Project context; not a persona and not an implicit sharing grant. |
| Session | Host-supplied conversation/run identity | Execution/history key. Same-subject persistent sessions share Recap; workers do not become subjects. |

### Memory authority and backend wiring

Laputa's `memory.Backend` is a useful port. A primary writer must provide Search, Expand, Mutate, and MutationLookup; vector, timeline, and graph behavior is optional. Mutations carry exact scope, destination, operation ID, revision/absence preconditions, and provenance sources. This is a backend contract, not yet a configured provider product.

The production adapter shipped in Laputa is Mentle. Ordinary `garden/main.go` does not provide `Config.Backends`; `runtimecore` lazily binds Mentle for the current profile when available. A supplied backend map is internal and scope-keyed. There is no UI or persisted configuration to choose, attach, validate, or report another provider. `ListCollections` in `agentapi` still reads `runtime.Mentle` directly, so not every read follows the interface.

The current VIVY BML module is real code, not a Garden backend. It opens one `bml.NewHome(cfg.DataDirectory())`; the `memory` module exposes context recall and a completed-run observer. That observer records a bounded `run.completed` summary with a deterministic receipt and experience-journal reference; it does not write a raw transcript. BML records include `tenant_id`, `workspace_id`, and optional `session_id`, and the typed store checks one bound workspace. No `persona_subject_id` is present in the record scope. BML has record CRUD and FTS recall, but the code inspected does not expose the full Garden `Search/Expand/MutationLookup` adapter contract as a ready provider. A mapping from BML tenant/workspace to Garden persona subject is an implementation-blocking decision, not a safe assumption.

DIVA currently binds its Garden subject/profile to `diva`, `workspace_id` empty, `destination_id` `mentle`, and uses the Laputa `agentapi` with Mentle. DIVA's existing P0 plan explicitly describes Mentle as selected for that composition and disallows speculative BML fallback. Its memory-loop plan lists S01–S03 as Blocked and later story statuses as Planned; it contains useful focused development evidence but does not claim end-to-end product acceptance. Keep that work distinct and reuse only evidence that its own gates mark valid.

**Backend conclusion:** no user-ready alternate Garden backend is present in the inspected mainline. BML is a plausible compatibility adapter candidate, not a drop-in alternate. A Garden UI should report the selected authority and health, not imply that a backend is pluggable merely because a Go interface exists. Standalone v1 needs one selected backend/provider and one canonical database shared by its subjects; subject-bound adapter instances may still enforce exact scope against that same database. Provider absence or revocation must fail visibly and must never fall back to another writer.

### Authorization and API paths

The newer `agentapi` exposes bound persona and ACTMEM reads, `SearchCards`, and bounded `ReadEvidence`; search and expansion derive permitted scopes from a binding. This is the closest current path to the confirmed Garden-owned boundary.

Other paths still need convergence:

- `/v2/memories` create/list/get/update/delete call the Mentle facade directly and accept current canonical DTO scope. They do not share the subject-grant decision made by `agentapi`.
- `/v2/recall/deep` calls `DeepRecall.Recall` directly from a request body; it is not bound through the same subject/session API.
- `agentapi.ListCollections` calls Mentle directly instead of the selected backend.
- `garden-mcp` is an existing stdio adapter, and its memory search/`record_signal` calls use the legacy `/v2/memories` route family. Its prior OpenClaw integration evidence does not prove multi-subject authorization or automatic lifecycle capture.

Stage 2 must route every retained API and host adapter through one authorization/domain boundary. Route names can remain as thin adapters when there is a real caller, but no parallel raw-Mentle compatibility writer should remain solely to preserve an old implementation path.

### UI and first launch

Garden's console is already functional for one profile: Persona displays authority files, edit/review, history/revisions, and initialization. Memory displays explicit ACTMEM reads/queries, Mentle cards with evidence fragments, and activity. Settings shows local preferences, a session capability token, pipelines, and EvoMap Hub status.

The console does not have a subject registry, subject switch, share grants, host bindings/revocation, backend selection, source-edit workflow for all records, host saved/written/loaded revisions, separate host toggles, DeepSeek API setup, or a persisted onboarding wizard. Existing `PersonaInitialization` is a page-level one-time persona form, not a restartable first-launch flow with upgrade/data-preservation semantics.

The user-facing console should make canonical content and actual host state inspectable: for the selected subject, show the current Persona revision, subject Recap revision, backend authority/status, memory item and source lineage, permissions, connected host binding, and each host's last written and last confirmed loaded revision. Do not claim a file write changed an already-running prompt.

## Upstream integration research

All upstream links below were checked 2026-10-12. Host docs evolve; pin the selected host versions before implementation.

### TencentDB-Agent-Memory as the functional benchmark

The [v2.0.2 release](https://github.com/TencentCloud/TencentDB-Agent-Memory/releases/tag/v2.0.2) points to commit `99a1a2c251741658917e3fa700bf8e337ad15a5f`. Its repository separates MemoryCore, MemoryKnowledge, MemoryPanel, and MemoryProxy. The release adds MongoDB support inside its storage implementation, OpenCode and DeepSeek Harness adapters, Pi integration, external-source import/status UX, and management-panel improvements. The project provides OpenClaw and Hermes adapters, memory search/write flows, one management panel for users, teams, agents, assets, versions, access/visibility, and bindings, plus an optional transparent LLM proxy.

The user-facing parity target for Garden is the workflow capability: create/manage persona subjects; inspect and edit actual memories with provenance; control private/shared visibility; manage host bindings; see provider and host status; integrate supported agents; and review changed state. Garden does not need Tencent endpoints, metadata models, or service decomposition. Tencent's Mongo `STORE_MODE` and MemoryCore API are implementation-specific storage choices, not evidence of an arbitrary Garden memory-engine provider contract. Its L0–L3 taxonomy, skill/wiki/code-graph assets, automatic skill evolution, team IAM, and proxy are not to be transplanted. Basic selected experiences should become provenance-linked Garden facts under human control.

### OpenClaw

Official [agent workspace](https://docs.openclaw.ai/concepts/agent-workspace) documentation defines `AGENTS.md` as session-loaded operating instructions, `SOUL.md` as per-session persona/tone, `USER.md` as user model, and optional `MEMORY.md` as curated long-term context. The official [SOUL guide](https://docs.openclaw.ai/concepts/soul) says SOUL content is injected into normal sessions. OpenClaw also provides plugin and memory/context hooks; its built-in `memorySearch` is a file/index feature, not a generic external memory database plug-in slot.

Laputa already has [`docs/design/garden-openclaw-memory-provider.md`](../design/garden-openclaw-memory-provider.md) and a shipped [`garden-mcp`](../../garden/cmd/garden-mcp/main.go). The design recorded an OpenClaw MCP-server integration and a tested stdio flow. This is a significant overlap to preserve. It is only a tool/API adapter today: it must be reconciled with the current one-subject routes, unified backend ownership, host file distribution, explicit consent, independent recall/capture controls, and truthful loaded-revision reporting. Re-evaluate its older statements about MCP versus native plugin surfaces against pinned current OpenClaw docs instead of copying them unchanged.

### Hermes

Hermes's official [SOUL/personality guide](https://github.com/NousResearch/hermes-agent/blob/main/website/docs/user-guide/features/personality.md) describes `SOUL.md` in `HERMES_HOME` as the primary identity slot in the system prompt; an existing file is not overwritten by Hermes. Its [memory provider guide](https://github.com/NousResearch/hermes-agent/blob/main/website/docs/user-guide/features/memory-providers.md) and [provider SDK guide](https://github.com/NousResearch/hermes-agent/blob/main/website/docs/developer-guide/memory-provider-plugin.md) expose lifecycle hooks for prompt context, prefetch, turn sync, session end, pre-compression, and tools. Discovery only enumerates a provider; it does nothing until `memory.provider` selects it. Once selected, Hermes can automatically inject context, prefetch, sync turns, and add tools while built-in memory remains active.

This is a strong native adapter surface, but a provider must not silently activate capture just because the plugin was installed or discovered. Opt-in and the three independent toggles must control the actual hooks, not just the Garden UI. The provider receives host identity/session/workspace context; these values are inputs to a Garden host binding, not proof of persona subject or permission. Hermes session prompt caching and process restart requirements mean file writes need separate `written` and `loaded` status.

### Claude Code and Codex CLI

Claude Code's [memory documentation](https://code.claude.com/docs/en/memory) distinguishes user/project `CLAUDE.md` instructions from automatic memory. It loads those at conversation start; auto memory loads a bounded portion, and instructions are contextual rather than a hard authorization boundary. [Hooks](https://code.claude.com/docs/en/hooks) provide lifecycle integration but cannot remove context already supplied to a running session. A Garden adapter may manage a designated instruction block and use hooks only after explicit activation; Claude-owned auto memory should not become a second Garden authority.

Codex documentation describes repository [`AGENTS.md`](https://learn.chatgpt.com/docs/agent-configuration/agents-md) instructions and configurable [hooks](https://learn.chatgpt.com/docs/hooks), including session-start additional context. This is a supported context-injection seam, not a general Garden memory-provider or revocation API. Keep Codex CLI in research until the initial host adapters are working. Do not conflate ChatGPT product Memories with Codex CLI's workspace/session surfaces.

### Model setup

DeepSeek's [first-call documentation](https://api-docs.deepseek.com/) documents an OpenAI-compatible API base URL and an API key. It is feasible to support one DeepSeek test configuration without a general provider framework. Keep the key in the host's secret store/environment, never in persona markdown, synced host files, or a committed config. The selected initial DeepSeek model and the host's supported model fields should be checked again at Stage 2; this report does not pin a model name as a product decision.

## Integration alternatives and recommended boundary

| Option | What it gives | Cost / limitation | Assessment for Garden v1 |
|---|---|---|---|
| Garden-managed host files only | Simple persona distribution to `SOUL.md`, `AGENTS.md`, `CLAUDE.md`, or equivalent; no extra daemon per host | Static snapshot; does not provide bounded recall, capture, source edits, or immediate revoke. A running session keeps prior context. | Required as a consented persona-sync surface, insufficient alone. |
| MCP tools only | Reuses shipped `garden-mcp`; explicit memory search, source expansion, and writes; low runtime complexity | Host must call tools; does not prove session capture or automatic recall. Current MCP routes include legacy Mentle paths. | Keep as an explicit, user-invoked adapter after it uses central subject authorization. |
| Native host plugin/hooks | Can report session lifecycle, prefetch bounded context, and optionally capture completed selected experiences | Per-host code/config maintenance; auto hooks have stronger side effects and reload/lifecycle differences. | Preferred for OpenClaw/Hermes if their lifecycle API can keep three toggles genuinely independent. Keep adapters thin. |
| Transparent model proxy | May observe requests/turns across otherwise unsupported clients | Adds an extra service and credential/inference boundary; broad interception/capture; more components; weak fit for explicit host consent and least-scope toggles. | Do not add for v1. Tencent has this optional component; Garden requirements do not require it. |
| Markdown export / indexed folder | Minimal setup and easy inspection | One-way snapshot, stale until rewritten/reindexed, no authorized CRUD or revocation. | Optional export only; never treat it as canonical memory or a provider. |
| BML as a Garden backend | Preserves current VIVY store and typed provenance/record model | No persona-subject mapping or Garden `Expand`/mutation-receipt adapter; workspace-bound machine home. | Evaluate as a compatibility adapter only after subject and canonical-writer decisions. No automatic import or dual write. |
| Tencent MemoryCore as an external backend | Mature gateway, panel and agent integrations | Separate authority, API, server and metadata model; not a Garden adapter; imports/migrations and semantics differ. | Benchmark the user experience; not a v1 default backend. |

A practical low-overhead shape to validate in Stage 2 is one Garden process/UI, Laputa as the single Persona/ACTMEM owner, one canonical memory backend chosen explicitly, one central authorization decision per request, and small OpenClaw/Hermes adapters that register only after per-instance opt-in. Use the existing MCP adapter for explicit user-invoked operations and add lifecycle code only for capabilities whose toggles the host can enforce separately. This is a candidate direction, not an accepted architecture decision.

### Host state contract

For every opted-in host instance, store a binding to one Garden subject and separate booleans for persona sync, recall, and capture. Host discovery is read-only and creates no binding. Installation writes no active config. When the owner enables a surface, Garden must:

1. show the exact host instance, paths, proposed changes, and selected persona;
2. preserve unrelated instructions and config, check the observed file revision/hash before writing, and make a recoverable backup;
3. write only the designated managed file/section, then report `saved` (Garden authority revision), `written` (host file revision/hash), and `loaded` (host-confirmed active revision or `unknown/pending reload`);
4. bind memory calls to the selected subject and only the granted operations; keep passive discovery, persona sync, recall, and capture separate;
5. on grant revocation, block subsequent Garden reads/writes and report the revocation revision. Existing prompt contents, transcripts, exported files, or cached host sessions cannot be retroactively recalled.

A host's project workspace or runtime identity is not a substitute for the chosen persona. The adapter must not switch subjects because a session ID, workspace path, agent name, or thread changes.

## Overlapping work: preserve and reconcile

I inspected the public issue/PR pages, current remote branch heads, and the linked mainline documents. No same-purpose standalone Garden branch or open PR was visible during this inspection. Relevant work is already underway in adjacent scopes:

| Existing item | Status/evidence | Overlap and boundary |
|---|---|---|
| [Laputa ADR-0018](../architecture/0018-vivy-unified-context-and-recap-policy.md), [merged PR #6](https://github.com/ProjectViVy/laputa/pull/6), [issue #1](https://github.com/ProjectViVy/laputa/issues/1) | ADR-0018 is accepted policy; PR #6 merged documentation only; issue #1 remains the coordination issue. | Direct overlap on Recap, session/worker boundary, BML/Mentle, and owner boundaries. Reuse ADR-0018; do not restate its policy as new architecture or claim it is implemented. Issue #1 explicitly coordinates VIVY #57/#58 and backend work. |
| [Laputa issue #7](https://github.com/ProjectViVy/laputa/issues/7) | Open Mentle/MemPalace capability enhancement plan. | Separate memory-engine capability research. Coordinate alternate-backend decisions with #1/#7; do not make Garden an excuse to duplicate the engine enhancement or silently add a second canonical store. |
| [Laputa OpenClaw design](../design/garden-openclaw-memory-provider.md) | Existing design says stdio MCP Phase 1 is implemented and records verification. | Preserve it as adapter history; reconcile its single-profile/raw-route assumptions and older SDK research. This report does not replace or overwrite it. |
| [Laputa Garden Authority & Recovery package](../bmad/garden-authority-recovery-2026-09/README.md) and [external agent SDK contract](../bmad/laputa-external-agent-sdk-2026-09/api-contract.md) | Frozen earlier one-profile/API contracts; ADR-0018 supersession notes are already attached. | These include exact single-profile identity, fixed scope, and prior ACTMEM assumptions. They must be consciously amended/superseded in Stage 2; do not implement their stale premises as the multi-persona design. |
| [VIVY issue #57](https://github.com/ProjectViVy/agent-vivy/issues/57), [#58](https://github.com/ProjectViVy/agent-vivy/issues/58), [#60](https://github.com/ProjectViVy/agent-vivy/issues/60), [merged PR #59](https://github.com/ProjectViVy/agent-vivy/pull/59) | #57/#58 are open design/integration issues; #60 is a separate offload/on-demand recall proposal; PR #59 merged memory policy docs only. | #57 owns one VIVY subject, shared subject Recap, and role-only worker boundaries; #58 owns the VIVY context/Recap UI and provider policy. #60 is not an accepted new CODE pipeline. Preserve the VIVY multi-persona question as undecided; standalone Garden must not force its answer. |
| [DIVA P0 design](https://github.com/ProjectViVy/agent-diva/blob/0a64d1ce152440f1f63d4224a155c4616d2aff5b/docs/plans/diva-next/p0-design.md), [memory-loop plan](https://github.com/ProjectViVy/agent-diva/blob/0a64d1ce152440f1f63d4224a155c4616d2aff5b/docs/plans/diva-next/memory-loop/README.md), branches `docs/diva-next-detailed-architecture-20261003`, `DIVA-NEXT-P0`, and `feat/memory-loop-verification-20261009` | P0 fixes one embedded Garden owner and Mentle; memory-loop README marks formal gates blocked/planned. | This is DIVA desktop integration and memory validation work, not standalone Garden. Reuse its proof artifacts only within their stated gates; don't create a parallel DIVA memory loop. |

Laputa's current remote branch list includes `docs/memory-policy-alignment-20261011` and other non-Garden features; VIVY has the matching policy-doc branch plus other features; DIVA has P0, memory-loop, and policy-doc branches. The checked heads contained no `feat/garden-standalone-v1` purpose match. The public GitHub PR lists showed the relevant #6/#59 as merged and no open same-purpose PR at inspection. GitHub CLI authentication was unavailable, so this conclusion is based on fetched remote refs and public issue/PR pages, not an authenticated organization-wide search.

## Superseded decisions and cleanup inventory

The following is an inventory for Stage 2, not a deletion list executed here.

### Keep and reuse

- Laputa Persona as the only identity authority; Garden registry entries must point to those records rather than copy identity markdown into a second database.
- Subject-level ACTMEM Recap policy from ADR-0018; current files and consumers still include Pulse/Recap/Work and need a migration-free implementation decision.
- `garden/memory.Backend`, exact scope encoding, mutation receipts, source expansion, and the Mentle adapter where they satisfy the selected canonical backend contract.
- `garden/agentapi` as the policy/domain boundary, extended to resolve an explicitly authorized subject. Existing REST/MCP integrations can be thin callers after they use this boundary.
- Explicit selected experience -> reviewable provenance-linked fact. Keep the frozen/persona authority human controlled.

### Candidate cut or unwire

- Autonomous EvoMap capability evolution from the standalone Garden first-run path: `garden/internal/evolution` run/candidate/proposal/Hub services, `/v2/evolution/*`, the console `EvolutionPage`, and its navigation/provider setup. This is the runtime self-evolution surface, not the minimal user-approved fact flow.
- Any auto-enablement of OpenClaw/Hermes hooks or background turn capture; host installation/discovery alone must remain inert.
- The independent direct-Mentle data paths in legacy `/v2/memories`, `/v2/recall/deep`, `agentapi.ListCollections`, and the MCP call sites if they cannot be made thin authorized adapters. Preserve endpoint names only where a real caller needs them.
- Old UI/status pieces that exist only for autonomous evolution, after callers and release packaging are traced.

`garden/internal/mailbox` and its UI currently support evolution proposal exchange/review. Inventory its call graph before cutting; retain none of it merely as an unused future fallback. Do not delete `garden/evolution` or `laputa/evolution` by package name alone: `garden/evolution.Domain`, effect receipts, exact scopes, and backend mutation lookup are shared with the current DIVA composition and may be required for explicit user-approved writes. Separate the autonomy provider from these domain/receipt contracts first.

### Defer or drop

- Defer new-team/role administration, remote multi-node deployment, arbitrary provider marketplace, automatic skill/rule/persona generation, large-scale import, and host support beyond the requested first integrations.
- Drop automatic BML↔Mentle migration or mirroring; default dual write; silent attach/import across stores or machines; Tencent wire/API compatibility; and Tencent's four-layer memory taxonomy.
- Drop any fallback route that changes the selected authority when a backend is unavailable. Report unavailable/degraded and require explicit reconfiguration.

## Implementation-blocking decisions for Stage 2

1. **Canonical subject ID and storage layout.** The confirmed product invariant is one canonical memory database shared by all standalone Garden subjects. Decide its owner/layout, assign each Laputa/VIVY persona one stable Garden subject ID, and discover any existing VIVY home only with user authorization. Specify how an old BML tenant/workspace and a DIVA Mentle `ProfileID` relate to that database. Preserve the existing paths for validation, but treat a different store as an explicit attach/import decision; never create two writers for one subject or silently make separate stores look unified.
2. **Principal, subject, binding, and grant rules.** Define the small local-user/client principal model, subject ownership, explicit subject-to-subject share record, which operations sharing grants, workspace treatment, and revocation behavior. A binding must be resolved server-side; `AgentID`, session, workspace, request body, file path, or host-discovered installation cannot grant access.
3. **Cross-subject authorization boundary.** Decide how the bound subject and explicit shares become authorized scope lists for Search, Expand, CRUD, Recap, and source lineage. Route retained HTTP/MCP/API surfaces through this one boundary. Existing backend scope checking is a useful second guard, not the grant authority.
4. **One selected canonical writer.** Select one primary backend for the shared Garden database; new Garden subjects cannot be split across independent authorities. Define required capabilities/status, identity mapping, revocation, failure behavior, health reporting, and whether BML gains an adapter capable of serving that shared subject model. No default dual write or fallback; do not choose a backend that cannot preserve provenance, source expansion, revision control, and mutation lookup.
5. **ACTMEM Recap and persistent sessions.** Define the canonical subject-level Recap path/revision/CAS and session admission rule. Reconcile current Pulse/Recap/Work storage and DIVA's current per-session/task workers with ADR-0018. Specify that persona-free workers have bounded data and no subject Recap unless explicitly authorized as a session of that same subject.
6. **Host mapping and consent UX.** For OpenClaw and Hermes, choose which authority sections map to their persona/instruction files and how user text outside a Garden-managed block is preserved. Pick native plugin versus MCP/file-sync roles. Freeze a per-host opt-in record and independent sync/recall/capture toggles. Provide conflict/back-up/recovery and saved/written/loaded status. Confirm the adapter can stop capture when disabled and can identify selected experiences without storing unrestricted transcripts.
7. **Initial model secret and setup flow.** Choose the minimum DeepSeek config (base URL, model, secret location) and persisted wizard state. Decide how an authorized existing Garden/VIVY path is discovered and how “attach”, “import”, “create”, and “skip” differ. Upgrades must not recreate state or rerun setup; no background migration.
8. **Evolution removal boundary.** Trace `internal/evolution`, `garden/evolution`, mailbox, DIVA callers, API routes, console and MCP. Disable self-changing persona/rules/skills while retaining exactly the minimal provenance/review contracts needed for selected facts and explicitly human edits.
9. **Ship boundary and acceptance.** Confirm whether Stage 2 includes OpenClaw and Hermes adapters only, with Claude Code/Codex CLI research results recorded for later. Acceptance must prove actual same-subject continuity and private-default cross-subject denial/share/revoke, not claim a broad enterprise security system.

## Stage 1 verification limits

This is a repository and upstream-doc research report, not product behavior evidence. No application tests were run and no runtime files were edited. The report was checked against the listed remote main commits, current source paths, existing issues/plans/branches, and the cited upstream docs/release. DIVA's existing test and memory-loop evidence is described only according to its own plan status. Stage 2 must pin host versions, convert these blockers into an architecture decision, and inventory deletion callers before any runtime change.
