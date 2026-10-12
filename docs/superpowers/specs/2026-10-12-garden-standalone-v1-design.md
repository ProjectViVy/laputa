# Standalone Garden v1 — architecture specification for user review

**Date:** 2026-10-12 · **Status:** PROPOSED — WRITTEN-SPEC APPROVAL REQUIRED

**Authority:** This is the single proposed design for standalone Garden. [ADR-0019](../../architecture/0019-garden-standalone-v1.md) records its decision status; the [Epic/Story index](../plans/garden-standalone-v1/index.md) owns proposed dependencies and coverage. No executable Story plans exist, no Story is Ready, and no runtime implementation, data transfer or default switch is authorized by this delivery. After written-spec approval, write and review Story plans and select an execution method before implementation.

## 1. Outcome, scope and evidence

Garden becomes a standalone, local-first application for managing multiple independent persona subjects and their memories. It has one canonical memory backend and one active owner, a full management console, and explicitly activated external-host adapters. Existing VIVY remains one subject and must use the same identity authority visible in Garden. Multiple subjects are not multiple human accounts. The proposed first release has one local human owner; remote/enterprise multi-user operation is not inferred from the request and needs owner confirmation (Q1).

Apply 多快好省: reuse the Go modular monolith, Laputa Markdown authorities, Garden agent API, existing console, exact memory scopes, Mentle adapter and BML library. Add no proxy, broker, external identity server, new agent runtime, scheduler, general plugin marketplace or duplicate memory engine. Basic selected experience → usable fact is included; autonomous persona/rule/skill evolution is excluded. Tencent is a functional UX benchmark, with no API compatibility, L0–L3 taxonomy, team IAM or four-service architecture requirement.

### 1.1 Requirements and acceptance intent

| ID | Required outcome | Observable acceptance |
| --- | --- | --- |
| R1 | Multiple stable persona subjects; VIVY remains single-persona; canonical identity reuse | Two Garden subjects remain distinct; attaching VIVY exposes its actual authority/revisions without copied persona state or a second writer. |
| R2 | Private by default, explicit grants, every retained entrypoint governed | A denied principal cannot learn another subject's cards, counts, evidence, history, collections, Recap or mutation receipts; share then revoke changes subsequent access. |
| R3 | One selected writer; real backend choice and truthful capabilities | Mentle and BML each pass the common contract before selectable; outage/revocation never switches backend; all subjects of an installation use its one database. |
| R4 | One subject Recap; explicit session/worker admission | Same-subject persistent sessions share revisions; restricted sessions and persona-free workers cannot inherit disallowed content; conflicts and clear are effective. |
| R5 | Full native Garden UI/API parity | Registry, Persona editor, actual Recap, memory CRUD/history/lineage, shares, bindings, provider health and saved/written/loaded status operate on actual authority. |
| R6 | Explicit per-host activation and independent controls | Discovery/install is inert; persona sync, recall and capture can each be off while others are on; write conflict, backup, reload and revoke behavior are visible. |
| R7 | Four first-phase host products | OpenClaw and Hermes native adapters, plus Claude Code and Codex CLI file/MCP capabilities, meet the specific matrix in §7; experiments are visibly distinct. |
| R8 | Human-controlled Persona; explicit experience-to-fact | Selected evidence produces a reviewable, revisioned fact with source lineage; no summary or background job changes Persona, Mission, Dream, rules or skills. |
| R9 | Real DeepSeek-only first launch | Minimal key entry, server-side secret persistence, restartable setup, create/attach/skip persona, optional hosts, no repeat wizard after completion; upgrade preserves data. |
| R10 | Deliberate attachment/import/ownership | Existing local and other-machine data receive explicit inventory and takeover/import decisions; simultaneous writers and implicit data copies are rejected. |
| R11 | Preserve accepted policy and cross-repo paths | BML-first VIVY validation, Mentle-through-DIVA validation, LIFE-first resident Recap and CODE exclusion remain; shared DIVA receipts survive cleanup. |
| R12 | Lean clean break with honest evidence | Superseded executable paths and unused dependencies are removed with caller replacements; historical evidence remains; no implementation claim from docs or old tests. |

### 1.2 Source baseline and critical checks

Research input: [Stage 1](../../research/garden-standalone-v1.md), committed at `76baea03e9913b8bfeb35c44cae0a66e4f5eed59`. Local source inspection used Laputa `f44ba00eb6ab5aa6ab9a2f5c09868373e36679f5`, VIVY `65b561f6f8189dc36dd29120a6f8a5f20b7071a6`, DIVA `0a64d1ce152440f1f63d4224a155c4616d2aff5b`. Links to local source below resolve within this branch, whose only changes after that baseline are documentation. These findings are static evidence, not running-product certification. One Stage 1 correction: VIVY does contain checked-in `.agents/skills` in this checkout; its `vivy-eino` skill and integration reference were inspected. No applicable local Supermanagement supplemental references or Codex-host skill were found; the provided Supermanagement/Brainstorming skill contracts were used.

| Verified source | Finding and architectural consequence |
| --- | --- |
| [runtimecore/runtime.go](../../../garden/internal/runtimecore/runtime.go), [agentapi/open.go](../../../garden/agentapi/open.go) | One `ProfileID`, Persona service and ACTMEM store are opened; current backend injection is not a multi-subject registry. Refactor composition to one shared backend owner and subject-bound views, not one full Garden runtime/database per subject. |
| [server/auth.go](../../../garden/internal/server/auth.go), [agentapi/policy.go](../../../garden/agentapi/policy.go) | Principals are role classes; HTTP also permits unauthenticated loopback GET reads. Existing agent provenance is not a unique authenticated identity. Add unique principal records and server-resolved grants; role names remain ceilings, not credentials. |
| [memory/contract.go](../../../garden/memory/contract.go), [scope.go](../../../garden/memory/scope.go), [Mentle adapter](../../../garden/backends/mentle/adapter.go) | Search/Expand/Mutate/MutationLookup are required; scope/v1 is exact. Mentle adapter construction rejects a foreign-subject read union. Reuse subject-bound views over the shared DB, extending management reads rather than bypassing the port. |
| [server.go](../../../garden/internal/server/server.go), [recall_handlers.go](../../../garden/internal/server/recall_handlers.go), [reads.go](../../../garden/agentapi/reads.go), [garden-mcp](../../../garden/cmd/garden-mcp/main.go) | CRUD calls `Facade`, deep recall calls `DeepRecall`, collection reads access Mentle, MCP uses raw routes. Every retained route must converge before isolation acceptance. |
| [VIVY BML scope](https://github.com/ProjectViVy/agent-vivy/blob/65b561f6f8189dc36dd29120a6f8a5f20b7071a6/bml/record.go), [store](https://github.com/ProjectViVy/agent-vivy/blob/65b561f6f8189dc36dd29120a6f8a5f20b7071a6/bml/store.go), [home](https://github.com/ProjectViVy/agent-vivy/blob/65b561f6f8189dc36dd29120a6f8a5f20b7071a6/bml/home.go) | Records have tenant/workspace/session but no persona subject. Store binds one workspace; machine home uses `machine-memory-home`. Neither tenant nor workspace can silently become PersonaID. §5 defines the required mapping work. |
| [VIVY memory provider](https://github.com/ProjectViVy/agent-vivy/blob/65b561f6f8189dc36dd29120a6f8a5f20b7071a6/internal/modules/memory/provider.go) | BML capture is a bounded completion projection with Journal reference, not full original conversation. Do not manufacture missing evidence. |
| [VIVY DIVA factory](https://github.com/ProjectViVy/agent-vivy/blob/65b561f6f8189dc36dd29120a6f8a5f20b7071a6/internal/modules/diva-cognitive/factory.go), [dispatch](https://github.com/ProjectViVy/agent-vivy/blob/65b561f6f8189dc36dd29120a6f8a5f20b7071a6/internal/modules/diva-cognitive/dispatch.go) | DIVA's Go integration lives in VIVY: fixed `diva`, embedded owner, Mentle, `BindEvolutionSource`/`BindEvolution`, Work and human actions. Changes require coordinated VIVY consumers, not deleting packages because their name says evolution. |
| [domain.go](../../../garden/evolution/domain.go), [embedded_domain.go](../../../garden/agentapi/embedded_domain.go), [laputa/evolution](../../../laputa/evolution/contracts.go) | Memory mutation dispatch, effect receipts, lookup/recovery, Mission fences and scope contracts are shared with DIVA. Preserve needed contracts while removing standalone autonomy wiring. |
| [console App](../../../garden/console/src/App.tsx), [Persona](../../../garden/console/src/modules/persona/PersonaPage.tsx), [Memory](../../../garden/console/src/modules/memory/MemoryPage.tsx), [Settings](../../../garden/console/src/pages/SettingsPage.tsx) | Existing single-profile UI is reusable; no registry, grant manager or restartable provider wizard exists. |

### 1.3 Reconciliation and concurrent work

[ADR-0012](../../architecture/0012-laputa-markdown-clean-break.md), [0013](../../architecture/0013-laputa-clean-break-implementation-architecture.md), [0014](../../architecture/0014-mentle-canonical-authority-and-derived-index-recovery.md), [0016](../../architecture/0016-laputa-embeddable-modular-monolith.md) and [0018](../../architecture/0018-vivy-unified-context-and-recap-policy.md) remain the accepted baseline. [Laputa PR #6](https://github.com/ProjectViVy/laputa/pull/6) and [VIVY PR #59](https://github.com/ProjectViVy/agent-vivy/pull/59) were verified merged: both deliver policy documentation only. Their historical test outcomes do not prove new Recap or standalone behavior.

[Laputa #1](https://github.com/ProjectViVy/laputa/issues/1) owns detachability/provider coordination; [#7](https://github.com/ProjectViVy/laputa/issues/7) owns Mentle enhancements, including Chinese lexical correctness, truthful backup and embedding injection. Consume those results where necessary; do not duplicate its batches. [VIVY #57](https://github.com/ProjectViVy/agent-vivy/issues/57) owns session/worker lifecycle; [#58](https://github.com/ProjectViVy/agent-vivy/issues/58) owns unified context/Recap integration. This design supplies Garden-side contracts and integration acceptance, not another VIVY context engine. All four issues were open when checked.

The [OpenClaw design](../../design/garden-openclaw-memory-provider.md), [external SDK contract](../../bmad/laputa-external-agent-sdk-2026-09/api-contract.md), [authority/recovery package](../../bmad/garden-authority-recovery-2026-09/README.md) and [DIVA memory-loop package](https://github.com/ProjectViVy/agent-diva/blob/0a64d1ce152440f1f63d4224a155c4616d2aff5b/docs/plans/diva-next/memory-loop/README.md) are retained by reference. Preserve their scoped evidence and blocked/planned gates. Standalone subject selection, unique principal grants, backend selection and host lifecycle supersede their single-profile assumptions only if this spec is approved. The docs rule against optional backend work is explicitly proposed to be superseded for the user-requested backend product; Mentle's own SQLite/outbox authority remains unchanged.

Remote branch inspection found this branch at Stage 1 and no other branch named for standalone Garden. Adjacent architecture/policy work exists as linked above. This is not proof that no private discussion or unpublished work exists. Recheck remote head before publishing; preserve and reconcile any newly appearing overlapping design by reference, never overwrite it or force-push.

## 2. Architecture choice and responsibility boundaries

| Approach | Benefit | Cost/decision |
| --- | --- | --- |
| Extend existing modular monolith with central registry/policy and thin adapters | Reuses authority, UI, transports and storage; one local service; a single authorization boundary | **Recommended.** Requires route convergence and explicit owner handoff. |
| Keep independent embedded owners and aggregate their UIs/stores | Smaller first UI patch | Reject: does not deliver unified canonical memory or solve writers, isolation and revocation. |
| Introduce memory gateway/proxy plus independent identity and integration services | Broad interception and deployment options | Reject: unnecessary services, credentials and capture surface; poor consent fit. |

Proposed dependency direction:

```text
Console / HTTP / Garden MCP / embedded trusted client / host adapters
                              |
                Garden agentapi application boundary
              auth -> binding -> grant -> scope -> operation
                  /             |                 \
       subject registry      Laputa             memory.Backend
       host/setup state      Persona + Recap    selected adapter only
       (Garden state DB)     (Markdown owner)    Mentle OR BML canonical DB
```

No transport owns business policy. Extend existing `garden/agentapi`, `garden/internal/runtimecore`, `garden/memory` and console modules. Proposed new packages: `garden/internal/registry` (subjects/principals/grants/bindings), `garden/internal/hosts` (file plans/adapters/status), `garden/internal/setup` (wizard/secrets/upgrade coordination), `garden/backends/bml` (adapter over the existing BML Go module). These paths do not yet exist. Keep registry, grants, host state and setup in the existing Garden state SQLite database. They are control metadata, not a second memory-content database. Backend mutation receipts stay transactionally owned by the selected backend. Laputa revision metadata stays with Laputa.

One process holds the local installation ownership lock and opens each subject authority once. Subject handles share the selected backend connection pool; they must not each run index rebuilders or ingestion workers. Host integrations use authenticated calls to that process. Embedding remains available as a mutually exclusive deployment composition for existing VIVY/DIVA use; an embedded and standalone owner cannot open the same installation concurrently. Explicit absolute paths and failure-on-misconfiguration are preserved.

## 3. Identity, persistence and ownership

### 3.1 Distinct identities

| Entity | Proposed key and meaning | Never implies |
| --- | --- | --- |
| Persona subject | Opaque stable `PersonaID`; same value is the memory `SubjectID`; immutable after registration | Permission, agent process, workspace or login |
| Principal | Random `PrincipalID` bound to one human session or individually issued host/worker credential | A subject selected in request JSON |
| Runtime agent | `AgentID` and platform supplied as validated provenance | Authentication or grants |
| Host binding | `BindingID`, principal, selected PersonaID, approved workspace tuples and three independent controls | Activation from install/discovery |
| Workspace | Opaque registered workspace ID with owner-confirmed local path mapping | Universal sharing; empty workspace is exact personal scope, not wildcard |
| Session | Host-instance-qualified session ID bound on admission; subject and permissions fixed for its lifetime | A separate subject, Recap or cross-workspace entitlement |

Retain existing valid subject identifiers such as `diva` when explicitly attaching that authority. Never use arbitrary imported IDs directly as filesystem path components; new directory names are validated opaque IDs and existing authorities use the separately approved canonical path. New subjects get random IDs; display names are editable and non-authoritative. An existing alias is not globally unique evidence: registration also checks authority location and installation identity. A repeated attach of the same authority resolves to the same subject; an ID collision from another installation is an explicit conflict, never merged by name.

### 3.2 Proposed data layout

```text
<GardenHome>/garden.db                  registry/grants/bindings/setup metadata
<GardenHome>/subjects/<PersonaID>/      Laputa config root for a new subject
  persona/                             existing Persona owner layout
  actmem/ACTMEM.MD                      existing ACTMEM location; proposed v3 body
<GardenHome>/memory/                    ONE selected backend's canonical store
<GardenHome>/secrets/                   host-side credentials; never served as content
<GardenHome>/backups/                   explicit local backups/host write recovery
```

These are proposed defaults, not mandatory relocation of existing data. `persona.Open(configDir)` already appends `persona/`, and `actmem.New(configDir)` appends `actmem/`; registry paths identify their common config root, not an extra copy of either leaf. Registry rows hold an approved canonical absolute authority path, backend installation ID, owner mode and current lifecycle state. Existing paths may stay in place after owner handoff. No registry copy of Persona or Recap bodies is editable. Preserve seven ADR-0012 Markdown documents plus existing human-controlled `MISSION.MD`; retain six Frozen Core projections and Mission v2. WORLD is tool-only; ACTMEM is never Frozen Core. Symlink/canonical-path identity must be checked before attach, and one authority path cannot be registered as two subjects.

### 3.3 Existing VIVY/Garden and other-machine data

Attachment is an explicit transaction-like workflow with persistent checkpoints: inventory → owner choice → stop/drain old writer → snapshot → acquire locks and validate → register → enable new client → verify → finalize. Discovery may inspect only owner-selected locations or explicitly approved standard homes; it never imports or activates. The manifest records source installation, paths, subject mapping, schema/version, canonical backend, prior owner and backup identity.

For an existing VIVY/DIVA embedded Garden, the selected mode is **standalone takes ownership, VIVY becomes an authenticated client**. Preserve the same Persona directory, revisions, Recap and selected canonical database. First ship the client composition and ownership fence in VIVY; only then offer takeover. Old direct library opening must refuse while the standalone owner holds the OS lock. A persisted owner-mode marker prevents accidental old-mode restart; a live lock, not the marker alone, prevents concurrency. Existing older binaries that do not honor the marker cannot be considered safe clients: stop them and require the supported version before takeover. A filesystem path alone is insufficient.

If Garden already uses a different memory database, it cannot simply attach another active database and call them unified. Offer: (a) use the existing installation as the entire Garden installation after takeover; or (b) explicit reviewed import into the currently selected writer, with source identity map, provenance, conflict inventory, dry-run counts and backup. No automatic BML→Mentle conversion. A source that remains active after a snapshot is labeled independent; it is not a second writer to the same persona. Reusing the same persona across VIVY and Garden requires actual client handoff, not snapshot import.

Other-machine data requires an owner-exported, versioned bundle and explicit import/ownership transfer. v1 proposes no live cross-machine filesystem sharing, synchronization or remote takeover. Until transfer is confirmed, display an attachment candidate with no active persona/binding. Interrupted handoff resumes from durable checkpoints and keeps writes disabled when ownership is uncertain. Rollback requires stopping the new owner and restoring an identified coherent backup; never auto-resume the old writer after new commits.

## 4. Authorization and the native Garden API

### 4.1 Minimal principal-bound model

Use one local human owner and unique opaque bearer credentials for each opted-in host instance. Persist credential digests and principal state in Garden; plaintext stays only in the designated host secret location. Human browser authentication uses a server-side session established by a one-time local pairing secret; no reusable admin token in browser localStorage, URLs or synced instruction files. Listen on loopback by default; reject untrusted Origin/Host and require same-origin authenticated mutations. Remote serving and network IAM are outside this proposal.

A grant identifies its principal, binding, exact subject/workspace scope, operation set, optional record set, expiry and revision. Principal role is an operation ceiling, not the access decision. Server-side registration supplies the subject and destination; caller `AgentID`, file path, session or model-generated scope cannot expand them. Human owner operations intentionally span owned subjects; automation receives only issued grants. Administrative principal/grant editing is never a model-facing tool.

Derived facts/cards retain the union of their source access requirements in addition to their own scope. Drafting or editing a fact into a less restricted scope cannot launder restricted evidence; reads check both, and revocation invalidates derived access. Explicit sharing must cover the actual disclosed content and required sources. Automated declassification is excluded.

Memory sharing is a separate owner-managed record: source subject, exact source scope and optional item selection, recipient subject, read operations and revision. Default is no share. A host acting for the recipient must also have a principal grant for the share and its workspace ceiling. Sharing is read-only in v1; foreign edits/deletes and recursive re-sharing are denied. Recap and Persona sharing are separate explicit read permissions, never implied by memory sharing. Review displays whether evidence/source content is included, not only card visibility. No public or transitive share graph is needed.

For persona-free workers, issue expiring child credentials restricted to explicit source refs/scopes and operations, intersected with the delegator's live grant. No Persona or Recap by default. Child grants cannot outlive/re-expand a revoked parent. An explicit same-subject persistent-session admission is distinct from giving a worker a role label.

### 4.2 Operation semantics and convergence

Every request follows: authenticate → resolve binding/session → intersect grant/share/workspace limits → validate bounds/revisions → invoke owning service → validate returned scope → recheck revocation before releasing data/committing. Use a bounded in-flight authorization guard spanning backend commit or response release; revocation takes the exclusive side and waits for prior guarded operations to finish/cancel before acknowledging. Do not hold this guard across model generation or file-review interaction. This supplies ordering across Garden metadata and a separate backend transaction without pretending the databases share a transaction. Serialize grant revocation and mutation authorization at the local service boundary. A successful revoke acknowledgement means subsequent operations and not-yet-released responses are denied; already returned data cannot be recalled. Queued captures reauthorize at commit and are discarded or shown as denied when consent is revoked, not silently replayed later.

Proposed native API uses `/api/garden/v1` for subjects, persona documents/reviews/history, Recap, memory cards/items/evidence/history/mutations, grants/shares, hosts, provider status and setup. Host endpoints resolve one binding from the credential; owner UI may select an owned subject. Concrete wire schemas are a post-approval planning artifact. Reuse existing DTOs/error/receipt semantics; this namespace has no Tencent compatibility requirement.

| Operation family | Mandatory behavior |
| --- | --- |
| List/search/collections/counts | Apply exact permitted scope before selection/ranking/aggregation; opaque cursors bind principal, grant revision, scopes and query. Invalidate on permission changes. No global counts or hidden-subject names. |
| Get/expand/lineage/history | Check every referenced item and every source hop at request time; expected revision for expansion; deny or omit unauthorized links without disclosing IDs/titles. A prior card is not a lasting capability. |
| Create/edit/delete | Own subject plus specific write grant; immutable subject/scope on ordinary edit; expected absence/revision; operation ID and payload digest. Cross-scope moves require a separate human-reviewed operation, excluded from ordinary v1 CRUD. |
| Mutation lookup | Bind operation ID to principal/binding and target scope before querying backend; never an unauthenticated receipt oracle. |
| Persona/Recap | Same principal/binding checks, then owning Laputa CAS/history rules; distinct human edit and bounded tool permissions. |
| Export, traces, activity and admin | Metadata can disclose content or existence. Scope every retained viewer; only owner gets installation-wide health. No raw database/Journal/filesystem browse endpoint. |

Retain `/v2` route names only for verified existing callers and make them thin adapters over the same boundary. Unknown/unbound role-only credentials fail with an explicit re-pairing requirement. Upgrade local callers and MCP together. Remove raw Facade/DeepRecall shortcuts; a legacy URL is never permission to use legacy authorization. Direct Mentle MCP is not a tenant boundary: standalone packaging must not expose it against Garden's canonical database. Mentle's standalone library/CLI remains usable for its separate product; that does not create a Garden bypass.

Errors distinguish unauthenticated, forbidden/not-found-without-existence-leak, invalid scope, revision conflict, revoked binding, capability unavailable, backend unavailable, outcome unknown, and ownership conflict. Never turn backend unavailability into an empty successful search or an acknowledged write. UI shows canonical commit and derived-index lag separately.

## 5. One writer and real backend pluggability

### 5.1 Selection and cost

Recommend Mentle for a **new standalone installation**, reusing the shipped adapter and canonical SQLite/outbox. This is a proposed new-install choice, not a switch of VIVY's BML or DIVA's Mentle defaults (Q2). Require a real BML adapter before claiming v1 backend pluggability complete. All subjects share one selected backend database; scope-bound adapters are views of it, not separate stores. Selection is persisted at installation level, not per subject or request. Changing an existing selection requires explicit backup/transfer review and a stopped writer; no hot fallback, dual write or search fan-out to another authority.

The current runtime makes model-less Mentle read-only or spools ingestion. That is inadequate for a fresh minimal-key wizard. Required integration delta: canonical CRUD and usable lexical retrieval must work without an embedding model; embeddings remain optional derived indexing. Coordinate the smallest no-embedding writable mode and Chinese lexical correctness with Laputa #1/#7, using existing facade/outbox. Do not copy their full enhancement backlog. If these prerequisites fail, report setup/backend not ready; do not invent a DeepSeek embedding endpoint or silently select BML. Owner may instead explicitly choose the qualified BML backend.

### 5.2 Shared contract and management reads

Preserve `memory.Backend` Search, Expand, Mutate, MutationStatus, Health, Close and existing exact scope encoding. Mandatory conformance includes bounded cards/evidence, stale-revision failure, source provenance, canonical revision/CAS, durable same-payload replay, changed-payload conflict, tombstones, scoped receipt lookup, restart recovery and honest canonical/index outcomes. A provider may use different ranking or native content formats; Garden does not prescribe Tencent levels or a universal fact taxonomy.

Cross-subject sharing is resolved by Garden: group authorized scopes by subject and obtain subject-bound adapter views over the same selected database. Query only those groups, merge bounded card pages using stable rank/order and scoped continuation state, then validate every result. This preserves the current Mentle adapter’s foreign-subject rejection and single write destination. A read share never obtains a writable foreign adapter. Bound fan-out/result budgets in configuration, report truncation, and invalidate continuation on grant changes; no cross-backend federation or unbounded global search is introduced.

The interface currently lacks full management browsing/history. Add a small **proposed management-read extension** beside it: paged item listing by exact scopes, current item lookup, revision listing/read and scoped collection metadata. Advertise these capabilities explicitly; both offered v1 writers must implement them for the full UI. Do not fake listing through an unverified empty-query search or keep `ListCollections` on raw Mentle. Canonical record/revision history remains in the selected backend; Garden may retain mutation audit metadata but not a shadow content ledger. Older embedded consumers can keep the existing base interface while the product selector requires the extension.

Required mutation receipt stays in the canonical transaction; derived index failure reports committed/degraded, not failed/no-write. A timeout requires status lookup with the same operation ID before retry. Source refs are typed backend/Journal references with revision/range and access requirements; no arbitrary URL/file fetching during expansion. Foreign refs require explicit resolver authorization. A missing source is shown as unavailable; never fabricate a transcript from a completion summary.

### 5.3 BML mapping and outstanding implementation work

Reuse `agent-vivy/bml` as the Go module; do not port a second BML into Garden. Its public module boundary is suitable for an adapter, but not yet sufficient for this contract. Required changes belong with BML owners and shared VIVY contracts:

- Add an explicit stable subject dimension and exact workspace authorization to stored records/queries. Preserve tenant as installation/owner provenance; do not overload tenant or `machine-memory-home` as persona. Garden uses one multi-subject database with row-level subject/workspace tuples. Extend the existing store open/binding mode to support this through explicit Garden composition; existing single-subject VIVY mode remains supported.
- Legacy rows get an owner-confirmed subject mapping in an explicit versioned schema/import operation after backup. Unknown or conflicting mappings remain invisible/unattached. Preserve record IDs where unique, provenance, session IDs and revisions; namespace conflicting imported IDs and record the map. Never infer subjects from display names or content.
- Implement scope-filtered Search/Expand, bounded provenance resolution, management reads, history/tombstones, and durable operation receipts atomic with mutations. Existing record CAS and completion receipts do not establish the entire Garden MutationLookup contract.
- Validate collection semantics and expose unsupported vector/timeline/graph features honestly. BML lexical behavior is provider-owned; basic CRUD/search/lineage and multi-subject isolation are required, not optional flags.
- Provide a Garden client option in VIVY so an attached installation has one writer; its existing direct BML path remains the first isolated validation baseline, not a simultaneous writer against Garden's active DB.

This is real cross-repo cost, not an interface-only promise. Do not call BML selectable until the shared contract and owner-approved attachment tests pass. Backend schema changes and actual data transfer need their own concrete reviewed plans after this specification is approved.

## 6. Subject Recap, context admission and explicit facts

Keep one `ACTMEM.MD` authority/revision stream per subject through Laputa, containing free-form active Recap plus non-content revision/provenance metadata. No per-session recap copies, no UI-local authority, no backend record posing as ACTMEM. Reuse Laputa's revision/history/CAS machinery; update the existing ACTMEM owner rather than introduce a second store. Proposed storage grammar is `actmem/v3`: existing YAML front matter conventions carry schema, subject, revision, parent revision, updated time, operation ID/digest, coverage, required access set and clear generation; the body is one free-form Markdown Recap, without required Pulse/Work fields. Metadata and body commit as one document. Write an immutable revision snapshot before atomically replacing the head, with file/directory durability; only revisions reachable from the committed head count as committed history/receipts, so a crash-created orphan cannot acknowledge a write. Reuse existing filesystem/CAS helpers and add only missing revision-history durability. Metadata is not independently writable. Older v1/v2 content is owner-inspectable historical material until an explicit reviewed conversion; no active dual-read/fallback into old Work semantics.

A session records admitted Persona/Frozen Core revision and Recap revision separately. Subsequent Persona edits affect a new admission, not already frozen session content.

**Restricted-workspace decision:** a free-form paragraph cannot safely be redacted by matching words. Each Recap revision therefore carries a conservative union of required source scopes/grants; permission to see a subject alone is insufficient. Admit the complete bounded projection only if the session can read every required scope. Otherwise admit no Recap, with `recap_scope_denied` visible to the authorized owner. Do not create workspace-specific Recap authorities. Consolidation requires access to the prior Recap and all new sources and preserves the union even if wording drops a detail. Restriction removal requires an explicit human-reviewed rebuild from authorized sources. Owner confirmation Q3 concerns the usability cost of this safe, coarse behavior.

VIVY owns its existing context policy, budgets, segment selection and summarization. Garden exposes authorized Recap read/CAS/clear and covered-source metadata; it adds no scheduler or second compactor. A retiring raw segment remains active until the new Recap commit succeeds. Conflicts return current revision; regenerate/review against that revision rather than blindly replay stale text. Concurrent sessions share the same stream and operation receipts; crash recovery uses committed coverage, not a session's optimistic watermark. No invented token/latency targets; tune on representative Chinese conversations with recorded model input and call cost.

Active clear commits an empty revision, increments a projection generation and establishes a consolidation coverage barrier. Invalidate cached projections and deny replay from older generations. Do not automatically reseed from pre-clear archives or already-covered evidence. A later explicit owner-selected restoration/reconsolidation may use retained sources; new post-clear segments may consolidate normally. Raw history, old Recap revisions and backend records remain until separately targeted deletion. Old prompt text already delivered remains in host history; native clients invalidate at their next context boundary, external hosts may require a new session.

Recap agent-management tools are normally absent. Reuse VIVY’s existing deferred tool search/activation and visibility middleware in [internal/runtime/engine.go](https://github.com/ProjectViVy/agent-vivy/blob/65b561f6f8189dc36dd29120a6f8a5f20b7071a6/internal/runtime/engine.go), with trusted grants attached at the owning API; do not add a second discovery framework. This machinery is source evidence, not proof that ACTMEM tools are already integrated. A small discoverable catalog may describe names/capabilities without their schemas; trusted host activation issues a session-bound operation grant and exposes only the requested inspect/update/clear schemas. Human-authorized write/clear remains CAS-bound. Hosts unable to defer schemas use explicit owner UI/MCP connection activation for this tool group; do not advertise transparent deferred discovery there. Automatic Recap injection is LIFE-first in VIVY, CODE excluded. Supporting Claude/Codex explicit persona/memory tools does not authorize VIVY CODE resident Recap or external coding-host autonomous consolidation.

Selected experience → fact is a separate, simple flow: choose authorized evidence/range → optionally draft a fact using DeepSeek → inspect content/provenance/scope → explicitly save through backend Mutate. Capture-enabled hosts may submit a selected bounded completion event with stable identity; capture does not promote every summary into a fact. UI shows queued/captured/draft/saved distinctly. No automatic pin, Persona mutation, skill generation or rules change follows fact save. Human must-not-forget requests go to existing Laputa edit/review, with Mission/Dream restrictions preserved.

## 7. External hosts: consent, mapping and phase-one acceptance

### 7.1 Common binding lifecycle

Discovery is read-only; installation places inert adapter files only. Bindings start disabled. Owner review selects host instance, persona, workspace, exact paths/config changes, grants and independent `persona_sync`, `recall`, `capture` switches. A file adapter credential has only the memory operations required; the owner authorizes file writes via Garden, not a model-generated instruction. Disabling a switch removes that future capability without enabling another. Revoking the binding revokes credentials and queues; attempts to restore managed files use conflict-aware cleanup, with manual recovery if the user changed them.

Persona mapping is an approved projection of selected existing Persona documents, labeled with subject and source revisions. Default proposal: Identity/Relationship/Redline/User and explicitly selected Dream/Dark; Mission is separately owner-selected; WORLD/Recap are never silently added to static persona files. Show exact rendered content before first activation and when the approved document set changes. Do not claim host instruction precedence makes Garden's projection authoritative over system/project instructions.

For each managed file/block retain original hash/backup, last written whole-file hash, managed-region hash, target revision and adapter version. Re-read and compare before writing; use atomic replacement and the host's compare-and-set API where provided. Preserve all content outside marked Garden regions. A conflict stops writes and offers a diff/rebase or restore choice; never overwrite unknown project/user instructions. Backups are host-side private files, not content served through memory APIs. No self-modifying rule hooks.

Track **saved** (canonical Persona revision), **written** (exact projection/file hash and revision), **loaded** (only a host/session-qualified acknowledgement of the exact projection). A successful file write or hook invocation is not a loaded acknowledgement. Expose `pending_reload`, `unknown`, `conflict`, `revoked` explicitly. Hosts without proof of model admission show loaded unknown; “context submitted” is a separate weaker status. New sessions retain which revision was actually acknowledged. Revocation blocks future Garden access but cannot erase already delivered prompt history, files or transcripts; stale managed snapshots are visible until safely removed/reloaded.

### 7.2 Proposed committed capabilities and limits

“Committed” below means proposed release acceptance if approved, not functionality already shipped. All four hosts are in phase one; reducing this matrix is a material owner decision (Q4). No consumer ChatGPT/Claude web persona control is promised.

| Host | Proposed committed phase-one surface | Experimental/support limit | Incremental cost and acceptance |
| --- | --- | --- | --- |
| OpenClaw | Native memory plugin/slot using Garden search/expand and selected capture; approved workspace `SOUL.md`/`IDENTITY.md`/`USER.md` regions as applicable; independent toggles; explicit MCP fallback transport only when selected | No second default host memory writer for Garden data; context-engine replacement and automatic compaction flush excluded. No loaded claim solely from agent-file RPC | One small TS adapter and contract fixtures. Prove cold-start persona, authorized recall, selected capture, disable/revoke, hash conflict and session reload on pinned host version. |
| Hermes | Native `MemoryProvider` plugin, explicit selection, Garden recall/selected event capture; approved `SOUL.md` region | SOUL is session-cached; existing session stays old until reload/new session. Provider availability probe must not activate or capture. Do not run autonomous memory consolidation/persona editing | One small Python adapter; pin provider lifecycle and teardown. Same consent/isolation/capture checks plus cached-session versus new-session status. |
| Claude Code | Managed region in owner-selected `CLAUDE.md`/local instruction surface; native Garden MCP search/expand/explicit fact save and human-triggered management; separate recall/capture grants | Opt-in hook recall and selected completion capture are experimental. No transcript scraping as a stable API, no auto-memory takeover, no replacement of existing conversation or loaded proof from file write | Shared Garden MCP plus small config/file adapter. Acceptance includes existing instructions preserved, new-session behavior, explicit memory operations, disable/revoke and accurate unknown-loaded state. |
| Codex CLI | Managed region in selected `AGENTS.md`, respecting directory discovery/override precedence; native Garden MCP explicit memory operations | Opt-in lifecycle hook context/capture experimental. Do not create `AGENTS.override.md` to supersede user instructions. Transcript format is not a stable capture contract; no consumer ChatGPT persona integration | Reuse MCP and file machinery; small host config adapter. Test effective instruction discovery, explicit CRUD/source access, conflict/revoke and honest reload status. |

MCP is not an automatic substitute when a native adapter fails; user selects and enables it with its own binding. Do not operate native and MCP capture simultaneously for one event source without shared event IDs/deduplication. CLI capture remains available as explicit selected-experience submission; unqualified automatic lifecycle capture is not a phase-one promise. Experimental hooks remain off by default, return bounded additive context only and fail visibly without blocking unrelated host operation. A host outage never causes credentials or captured payloads to be logged.

### 7.3 Official host evidence and version gate

Checked 2026-10-12 after local source/skill discovery; no installed host binaries were validated. These are integration seams, not conformance results. Exact supported host versions must be pinned in the subsequent Story plans and recorded in release acceptance; latest documentation alone cannot certify a host release.

- OpenClaw [workspace](https://docs.openclaw.ai/concepts/agent-workspace), [agent-file RPC](https://docs.openclaw.ai/gateway/protocol/rpc-talk-config-and-agents), [memory SDK](https://docs.openclaw.ai/plugins/sdk-overview/memory-and-context): file hashes/preconditions and native memory/plugin surfaces support the proposed adapter; use those instead of whole-workspace replacement.
- Hermes [provider plugin](https://github.com/NousResearch/hermes-agent/blob/main/website/docs/developer-guide/memory-provider-plugin.md), [raw provider reference](https://raw.githubusercontent.com/NousResearch/hermes-agent/main/website/docs/developer-guide/memory-provider-plugin.md), [personality](https://github.com/NousResearch/hermes-agent/blob/main/website/docs/user-guide/features/personality.md): native provider registration, `prefetch`, `sync_turn` and session lifecycle are valid seams; the provider reference explicitly warns that workspace changes do not invalidate the cached system prompt. Treat SOUL changes as requiring reload/new-session evidence, not a live patch. Ignore native-memory mirroring hooks: they would create a second Garden write path.
- Claude Code [memory](https://code.claude.com/docs/en/memory), [hooks](https://code.claude.com/docs/en/hooks): instruction loading and additive lifecycle context do not create retroactive prompt replacement.
- Codex [AGENTS.md](https://learn.chatgpt.com/docs/agent-configuration/agents-md), [hooks](https://learn.chatgpt.com/docs/hooks): instruction hierarchy and session-start context are supported; hook completion does not prove model use. Repository/local skills were checked first; no applicable OpenAI-specific local skill/source was available, so official OpenAI documentation was used.

## 8. First launch, DeepSeek, upgrades and UI

### 8.1 Persisted setup

First-launch state is a versioned row in Garden state DB, with step completion and last safe checkpoint. Proposed steps: (1) local owner pairing/home validation; (2) DeepSeek key; (3) storage readiness and optional existing-data choice; (4) optional first persona—attach, create or skip; (5) optional hosts; (6) finish. Only key entry is needed in the normal provider step; endpoint/model have a reviewed shipped default, shown in advanced settings. A skipped persona/host is valid completed setup. A skipped/failed key leaves model-dependent actions unavailable but permits explicit offline completion; normal management remains usable. On restart resume incomplete setup; after finish open console, never rerun wizard because a key expired or a backend became unhealthy.

Use fixed HTTPS DeepSeek endpoint `https://api.deepseek.com` for initial testing, without a general provider chooser or proxy. The [official first-call page](https://api-docs.deepseek.com/) checked on 2026-10-12 documents `deepseek-flash` and `deepseek-v4-pro`; propose the documented Flash identifier as the initial low-cost candidate, subject to account/model availability check at release and wizard validation. This is not a price/performance measurement or an immutable model promise. Send the native provider model ID, never a gateway prefix. No embedding API is assumed. A small explicit test request uses synthetic content, not private memory, and reports access/model errors truthfully.

Secret default: an owner-only host-side file in the private Garden home (directory 0700/file 0600 on Unix; equivalent owner ACL on Windows), referenced by ID from non-secret config. This uses existing filesystem facilities and avoids a mandatory vault dependency. Where a supported packaged OS credential store already exists, it may implement the same resolver. Environment-supplied key is supported without persisting the value. Never send the stored key back to the UI; never store it in localStorage, Persona, host instruction files, logs, export bundles or backups intended for sharing. Fail setup if private permissions cannot be established. Secrets are excluded from ordinary data exports; owner chooses separate protected backup/re-entry. The UI makes persistence choice visible (Q5).

### 8.2 Upgrade and failure ownership

Application schema version is distinct from onboarding completion, provider version and host adapter version. Startup checks versions under exclusive ownership; take a coherent stopped-writer backup before any approved schema upgrade. Supported metadata upgrades are transactional/restartable; backend and Persona format conversions use their owning libraries and an explicit preview when content/scope mapping changes. Never delete/reinitialize on parse failure, rerun first-launch as recovery, or silently migrate Pulse/Work into facts. Unknown versions fail read-only with a recovery explanation. Downgrade requires supported format compatibility or explicit restore, not opening new data with old code.

Backend health and model health are separate. Model failure disables generation only. Backend failure blocks new canonical writes and reports unavailable; reads are served only if the selected backend can honestly provide them. A capture queue is bounded, authorized transient work with explicit pending status; it is not a successful memory receipt or a fallback database. Shutdown stops intake, drains/cancels bounded work, closes stores and releases ownership. No background data conversion at boot.

### 8.3 Console interaction design

Reuse current shell/pages; replace single-profile assumptions with a persistent subject selector and installation status. No chat approval center is reused for Persona review. Wireframe-level flow:

```text
Garden  [Subject: VIVY v]                         [Backend: Mentle · available]
  Personas | Memory | Access & Hosts | Settings

Personas: list -> actual documents -> edit/diff -> save/review -> revision history
Memory:   Recap [full current revision | loaded per session | edit | clear | history]
          Records [scope/filter/search] -> card -> bounded source -> edit/history/delete
Access:   private scopes + explicit shares -> exact recipient/operations/evidence preview
Hosts:    discovered (inactive) -> preview mapping/grants -> enable independent switches
          saved r12 | written r11/hash | loaded unknown -> conflict/reload/revoke
Settings: provider/key status | selected backend/capabilities | setup/upgrade/backup status
```

Recap editor shows the complete active revision, pending edit, CAS conflict and source coverage; owner may compare and resubmit, never silent overwrite. Clear dialog names only the active target and preserved history; source/backend deletion is a separate target. History restore creates a new current revision. Memory detail distinguishes source event time, capture time, revision and provenance; edited facts keep lineage without pretending the original evidence changed. Permission-filtered or unavailable sources are visibly incomplete without exposing hidden titles. Host status does not claim model-loaded state from server saves. Zero-persona state offers create/attach; management works without a model. Backend selection shows required capability checks and a deliberate transfer requirement for existing data.

## 9. Decision and deprecation map

All decisions below are proposed except explicitly preserved accepted policies. Historical ADRs/logs stay intact. Implementation removes superseded executable behavior, DTOs, wiring, config, navigation and dependencies when their last supported caller is replaced; a disabled legacy toggle is not completion.

| Decision | Classification | Exact existing callers/paths and replacement/dependency |
| --- | --- | --- |
| D1: central registry, one process/one backend writer | Keep + extend | `garden/internal/runtimecore/runtime.go`, `garden/agentapi/open.go`, `garden/main.go`: share one backend owner, open subject authorities via registry; VIVY `internal/modules/diva-cognitive/factory.go` needs client/takeover mode before attachment. Preserve embedded library composition for separate installations. |
| D2: unique principals and exact grants | Cut old authorization assumption | `garden/internal/server/auth.go`, `garden/agentapi/policy.go`, session binding in `garden/agentapi/service.go`, console `api/client.ts` and `pages/SettingsPage.tsx`: replace shared role-token access and unauthenticated loopback GET privilege with paired principal/session credentials; AgentID stays provenance. |
| D3: every memory path through Garden | Cut bypass, retain necessary wire names | `garden/internal/server/server.go` CRUD handlers, `recall_handlers.go` deep recall, `material_handlers.go`, `garden/agentapi/reads.go` collections, `garden/cmd/garden-mcp/main.go` search/record_signal: use bound domain operations and backend management reads. Remove direct Facade/DeepRecall fields after callers move. Audit activity/session ingest/admin/report routes for content leaks. |
| D4: native backend product | Keep port; cut implicit Mentle assumption | `garden/memory/*`, `garden/backends/mentle/adapter.go`, `runtimecore` lazy backend binding and ingestion/fast recall: explicit selected destination; new BML adapter reuses VIVY `bml/{record,store,home,search}.go`. No alternate authority selected on failure. |
| D5: one scope-safe free-form Recap | Supersede active Pulse/Work; preserve history | `laputa/actmem/{store,format,ops,capture_pair,scope}.go`, `garden/agentapi/{actmem,reads,bound_reads,embedded_domain}.go`, `garden/internal/ingest`, `garden/internal/server/actmem_handlers.go`, console Memory page; VIVY `diva-cognitive/dispatch.go` Work actions and cognitive DTOs, plus `garden/evolution/domain.go` Work collect/apply and `laputa/evolution/{actmem,effects}.go`. Replace live consumers together; preserve archived versions with explicit human inspection. Do not auto-convert existing data. |
| D6: disable autonomous standalone evolution | Cut app/UI/provider path; capability deferred | `garden/main.go` EvoMap/Hub/mailbox assembly; `garden/internal/evolution/*`; `/v2/evolution/*` and `evolution_handlers.go`; `mailbox_handlers.go`; console `pages/{EvolutionPage,MailboxPage}.tsx`, Settings Hub fields, App/Sidebar/API/i18n entries; `garden/config` and release scripts mentioning EvoMap. Remove after call graph shows no supported consumers; update affected fixtures rather than retain unused production paths. |
| D7: preserve shared explicit-write/recovery infrastructure | Keep, narrow effects | `garden/evolution/domain.go` Apply/Lookup and effect ledger; `garden/agentapi/embedded_domain.go`; `laputa/evolution/{contracts,effects,frozencore}.go`; Mentle mutation receipt lookup. VIVY `diva-cognitive/factory.go` and `internal/cognitivecontract` use these through DIVA. Explicit selected-fact flow reuses memory mutations/receipts. No wholesale `evolution` package deletion. |
| D8: persona/rule/skill self-change excluded | Unwire standalone; defer shared removal | `garden/evolution/domain.go` persona/capability proposal effects, `laputa/evolution/inofy/*`, `laputa/evolution/diva/*`, and VIVY strategy/trigger consumers: standalone composition exposes no autonomous trigger or effects. Preserve code demonstrably required by supported DIVA contracts until a coordinated replacement is accepted; remove only genuinely unused surfaces. No dormant standalone autonomy provider. |
| D9: mailbox is not generic fact review | Cut unneeded standalone path | `garden/internal/mailbox/*` is coupled to capability proposal exchange; don't repurpose it for fact drafts. Shared capability sink references in `garden/evolution/domain.go` must be disentangled before package deletion. Keep shared contracts for live DIVA callers, not an unmounted console/service as a future fallback. |
| D10: native host consent and wizard | Replace single-profile setup/status | `garden/console/src/modules/persona/PersonaPage.tsx` initialization, existing MCP integration and `docs/design/garden-openclaw-memory-provider.md` single-profile assumptions: reuse forms/tools but route via persisted setup/bindings. Do not remove human Persona editor/history or Mission fences. |
| D11: direct Mentle server is not Garden access | Cut from standalone exposure | `mentle/cmd/server/main.go`, `mentle/pkg/mcp`, deployment/package references: no direct endpoint/credential against Garden-owned store. Keep independent Mentle product code unless a repository-wide caller audit proves it unused. |
| D12: no extra platform | Drop | Tencent wire compatibility/taxonomy/team IAM, LLM interception proxy, implicit multi-store aggregation, automatic migration/dual write/fallback, autonomous skills/rules/persona generation and consumer web persona control. |

The table inventories verified file/package callers, not every future changed line. Post-approval Story plans must refresh import/reference searches at pinned revisions, name remaining release/config/test consumers and sequence cross-repo replacements. This is particularly mandatory for D5/D7/D8: DIVA still calls Work and shared evolution contracts. Existing runtime paths cannot be deleted just because a new document calls them obsolete. Remove old executable paths as part of the replacement's acceptance, preserve immutable logs/backups, and annotate superseded architectural instructions when implementation becomes authoritative.

## 10. Verification contract, risks and owner decisions

No full app suites are required for this documentation delivery. Implementation acceptance must focus on real boundaries, not tests that merely mirror getters:

1. Two subjects with personal and restricted workspace data: deny by default across HTTP/MCP/UI/embedded adapters; share/revoke; guessed IDs/cursors/receipt/source-history accesses; concurrent revoke; provenance cannot widen scope.
2. Mentle and BML contract corpus: CRUD/search/expand/history, CAS, deletion/replay, canonical-commit versus index lag, outcome-unknown recovery, provider unavailable and no fallback. Prove one database/one writer per installation, including blocked second process.
3. Same-subject sessions and a persona-free worker: shared Recap CAS, restricted-scope denial, commit-before-unload, concurrent edit, clear-generation barrier and crash restart. Inspect actual VIVY LIFE model inputs; CODE exclusion stays negative control. BML first, then owner Mentle-through-DIVA validation; reused DIVA receipts and Mission fences remain exercised.
4. Fresh setup, restart at each side-effect boundary, skip persona/host, key failure/private persistence, completed setup restart, existing-data attachment and refused simultaneous owner, interrupted import/upgrade, preservation of current data and restore proof.
5. Each committed host capability on an exact recorded version: inactive install, all independent toggle combinations, preservation of user/project instructions, conflict/backup/revoke, new versus existing session revision, selected capture replay and forbidden operations. Experimental hook results are separate.
6. Full UI flow: real authority edit/clear/history, source expansion, share/revoke, provider degradation and saved/written/loaded differences; no display-only fake Recap or success stubs. Targeted static caller search proves legacy bypass/autonomy wiring removed.

Material risks: BML multi-subject/receipt/history changes and VIVY client handoff are cross-repo work; current model-less Mentle cannot satisfy fresh writable setup; host docs do not prove version support; free-form Recap has coarse access semantics; old binaries can violate owner-mode intent unless excluded from takeover. These are explicit requirements/dependencies, not reasons to invent a second database or weaken privacy.

### Owner review questions (proposals, not approvals)

| Question | Recommendation and consequence |
| --- | --- |
| Q1: v1 local owner and machine boundary? | One local human owner, multiple persona subjects and individually granted host principals; local takeover plus explicit other-machine import, no remote service/team accounts. Broader deployment adds authentication/transport/ownership work. |
| Q2: new-install backend and actual alternate delivery? | New standalone defaults to qualified Mentle; ship a qualified BML alternate in v1, with no change to existing VIVY/DIVA defaults. BML mapping/receipts and no-embedding Mentle are real cost; do not advertise alternate support if deferred. |
| Q3: restricted Recap behavior? | One Recap with conservative whole-revision source restrictions; deny projection when any requirement is missing. Simpler and safer, but a restricted event may make shared Recap unavailable to narrower sessions until reviewed rebuild. |
| Q4: host release commitment? | All four hosts in phase one as §7; OpenClaw/Hermes native, Claude/Codex files + explicit MCP committed, their lifecycle hooks experimental. Full automatic lifecycle parity for all four would add version-specific work. |
| Q5: initial packaging/secret persistence? | Private host-side file with equivalent Windows ACL; reuse OS vault only where already available. Confirm initial supported OS/package targets before executable host/setup plans; key remains out of content/config exports. |

Approval of this written specification is the next gate. It permits writing executable Story plans; it does not approve those absent plans, authorize production changes or mark any Story Ready. The proposed index supplies only outcome boundaries, immediate dependencies and acceptance coverage.
