# Shared implementation contracts

Contract revision: `diva-cognitive/v1-review-1`. Owner: S01. Consumers: S02–S09. These are proposed implementation contracts, not already exported APIs. Changes update this file and every affected Story before execution. Architecture requirements R1–R11 remain authoritative; this file makes their transport/storage details concrete.

## 1. Identity, scope and admission

All timestamps are UTC. Wire field names are snake_case. Strict decoders reject unknown fields, trailing JSON, unknown enum values and duplicate JSON keys. JSON carries DTOs; Persona/ACTMEM content remains Markdown.

`Scope` has `subject_id:string`, `kind:"personal"|"workspace"`, `workspace_id:string`. All subject IDs are nonempty; personal requires empty workspace_id; workspace requires nonempty workspace_id. Exact tuple equality defines scope equality. An absent scope is invalid, never global. Scope IDs are host-issued opaque identifiers; paths are not identities.

`RunBinding` has `subject_id`, `workspace_id`, `destination_id`, `policy_revision`, `strategy_digest` (nonempty strings) and `mission_revision:uint64`. It is populated by trusted composition, not model input. `mission_revision=0` identifies explicitly unassigned Mission and cannot admit mission-driven autonomous effects. Domain capabilities bind the authenticated principal and immutable allowed scopes on construction; JSON cannot construct or enlarge them. Read mounts never imply write grants.

`Window` has `source_id:string`, `after:uint64`, `through:uint64`; after <= through. It identifies (after, through] in one durable source. An empty range is no-new-input. No comparison across unrelated source IDs is permitted. A run freezes this window before inference.

`StrategyID` is `diva/v1`. Strategy identity includes semantic definition, exact prompt/schema bytes and implementation identity. UI presentation does not change program identity. Host binding adds the policy, Mission revision, destination and source window. Restart uses the same admitted values.

## 2. Strategy DTOs and signatures

Proposed Go types live in `laputa/evolution/contracts.go`. All wire DTOs below map directly to typed Go structs; use `uint64` for revisions/cursors, `int64` for Unix milliseconds, `string` for text/opaque IDs, and explicit enum types. Do not use `map[string]any` for authority bodies or effect payloads.

```go
func Evaluate(wake Wake, policy TriggerPolicy, state TriggerState) Eligibility

type Model interface {
    Infer(context.Context, ModelRequest) (ModelReply, error)
}
type Domain interface {
    Collect(context.Context, Window) (EvidenceBatch, error)
    Apply(context.Context, Effect) (EffectReceipt, error)
    Lookup(context.Context, string) (EffectReceipt, error)
}
```

| DTO | Exact fields |
| --- | --- |
| Wake | now_unix_ms:int64, manual:bool, new_activity:bool, foreground_busy:bool |
| TriggerPolicy | enabled:bool, min_interval_ms:int64 (>=0, explicitly configured; zero means no additional interval gate) |
| TriggerState | last_completed_unix_ms:int64, active_run_id:string |
| Eligibility | run:bool, reason:disabled/active/no_new_input/foreground_busy/interval/not_before_clock/eligible |
| Input | binding:RunBinding, window:Window |
| SourceRef | source_id:string, record_id:string, revision:uint64, scope:Scope; all IDs nonempty |
| Entry | id:string, section:pulse/recap/work, field:goal/open/next/constraints/pointers or empty for rings, scope:Scope, session_id:string, event_id:string, occurred_at:string, body:string, sources:SourceRef[] |
| AuthorityView | kind:string (closed eight-kind roster), revision:uint64, content:string; bounded, authorized Markdown only |
| EvidenceBatch | window:Window, activity_revision:uint64, entries:Entry[], persona:AuthorityView[], sources:SourceRef[] |
| ModelRequest | stage:prepare/reconcile/reflect, prompt:string, input_json:JSON bytes, output_schema:JSON bytes |
| ModelReply | output_json:JSON bytes |

Eligibility order: disabled, active, no new input, automatic foreground busy, backward clock, automatic elapsed interval, eligible. Backward clock postpones automatic work until clock catches up; manual ignores clock/interval/foreground gates but not disabled/active/no-input. `min_interval_ms=0` is valid and still coalesces active runs. No periodic polling is added to the library. Configuration persistence and timer delivery belong to the host.

The `laputa/evolution/diva` package owns prompt text and validation policy. `laputa/evolution/inofy` exports `Definition() (inofy.Definition,error)`, `Descriptors() []inofy.NodeDescriptor`, `NewExecutor(domain evolution.Domain, model evolution.Model) inofy.NodeExecutor`. It imports the strategy package; the root evolution package imports neither its descendants nor Garden/ViVy, preventing cycles. Limits are set by Host CompileOptions/RunRequest, not increased by strategy parameters.

Fixed node types/IDs: `laputa.evolution.collect@1` / collect, prepare@1 / prepare, reconcile@1 / reconcile, reflect@1 / reflect, effects@1 / effects, finish@1 / finish (all names have the same prefix). Catalog implementation IDs include the pinned implementation revision. All nodes are bounded calls; their data links and ordering follow the six-stage sequence. Empty batches yield no-change, including zero subsequent model calls. No node dynamically creates Agent tools or imports a model-generated executable graph.

## 3. ACTMEM versioned Markdown format

Filename: `actmem/ACTMEM.MD`; one per subject. Header starts and ends with `---` and contains YAML keys exactly `schema`, `revision`, `updated`, `entries`. Schema value is `laputa.actmem/v2`. `entries` maps stable entry IDs to section/field/scope/session/event/time/source metadata from Entry, excluding body. Reject unknown keys, duplicate keys, YAML aliases/tags and missing associations. Use the existing dependency closure where adequate; selecting a safe parser is an S03 implementation detail, not permission to introduce a second authority.

Body contains exactly these top-level sections in order: `## Pulse`, `## Recap`, `## Work`. Each entry is enclosed by exact full-line delimiters:

```markdown
<!-- actmem-entry:e_0123456789abcdef0123456789abcdef -->
Human-readable Markdown body.
<!-- /actmem-entry:e_0123456789abcdef0123456789abcdef -->
```

IDs match `e_[0-9a-f]{32}`. Fresh IDs are generated by the writer and retained on edits. Delimiters are recognized only as full lines, not Markdown semantics. A body containing either reserved delimiter prefix on a full line is rejected with format error; it is never silently reinterpreted. Every entry appears once, in its declared section, with matching closing ID and one header record. Missing/duplicate/misplaced IDs, orphan header metadata and arbitrary unassociated text are invalid v2 input. Empty sections are allowed. Work field is metadata; render its familiar Goal/Open/Next/Constraints/Pointers headings in UI/projection without creating another authority.

Order: Pulse/Recap chronological append order with ID tie-break; Work field order goal/open/next/constraints/pointers, preserving accepted entry order within each field. Normalize line endings to LF at admission. Existing body limits count Unicode code points, not bytes, excluding delimiters/front matter; body limit and request-byte limit are both checked. Mission retains its separate architecture limit. Header/source overhead is additionally bounded by the existing API request-size ceiling. Empty/deleted entries do not keep unbounded header records.

Legacy Markdown remains owner-readable but unclassified entries are excluded from Agent projections. Human classification is an explicit save operation using the same parser/write kernel. No auto-upgrade that silently makes old content global, no retired JSON personality import.

Public scoped operations use `ReadRequest{sections:[], max_chars:uint32}`, `WorkChange{kind:add|replace|complete|drop, entry_id:string, field:WorkField, body:string, sources:[]}`, `WorkPatch{base_revision:uint64, changes:[]}`. Add has empty entry_id; the writer allocates it. Replace/complete/drop require a visible existing Work ID in the admitted writable scope. Complete/drop have no new body. A separate permitted drop operation can remove visible ring entries; tools cannot append ring content. Scope and event metadata are never editable model fields. System append is a host-only capability carrying a typed Entry with trusted origin. Human whole-document save has a separate authenticated principal and base revision.

Output is `ActivityResult{changed:bool, revision:uint64, entries:[]Entry}` filtered to caller scope. Global document revision may reveal that some activity occurred; content, IDs and counts of hidden entries remain absent. This is content isolation, not a traffic-analysis guarantee. A scoped reader cannot use the revision to fetch hidden entries.

Caps remain Pulse/Recap/Work 1600 each, single Pulse 280, Recap 200, explicit read 1200, capsule 800. Work overflow rejects; ring overflow evicts oldest entries. A raw human save cannot bypass caps. Scoped conflict: one retry only if Work did not change and only rings advanced. Other Work changes conflict even across scopes; first delivery favors integrity over additional conflict optimizations.

Capsules preserve entry scope metadata. Archive-first fold uses deterministic IDs from the session plus ordered source-entry IDs/content digest; writes matching archive once, then CAS-removes unchanged source entries. Duplicate archive is harmless; mismatched content conflicts. Crash recovery never promotes capsules into the active head automatically.

## 4. Closed evolution effects

`Effect` has `operation_id:string`, `payload_digest:string`, `kind:EffectKind`, and exactly one typed payload. Digest is SHA-256 of normalized typed payload plus effective scope/destination/target/expected revision. Operation ID comes from committed INOFY OperationKey plus stable candidate index, never from model text. Domain overwrites no caller scope: it validates against the prebound target scope. Unknown fields and Mission/Dream variants reject before mutation.

| Kind | Payload |
| --- | --- |
| work_patch | WorkPatch from §3; no scope metadata override |
| memory_mutation | operation:create/update/tombstone, record_id:string, expected_revision:uint64, expected_absent:bool, body:string, sources:[]SourceRef, inference:observed/inferred; scope is the bound destination scope |
| persona_request | kind:identity/relationship/dark/user_observations/world, base_revision:uint64, proposed_markdown:string, reason:string, sources:[]SourceRef; authority's existing finer rules still apply |
| capability_proposal | name:string, description:string, proposed_artifact:string, sources:[]SourceRef; submits to EvoMap only, no install/publish permission |
| reflection_note | body:string, sources:[]SourceRef; report-only and excluded from fresh training/evidence collection |

Model reflection response is `ReflectionOutput{candidates:[]Candidate, no_change_reason:string}`. Candidate carries kind and one payload only; operation IDs/digests are assigned after validation. Empty candidates requires a nonempty no_change_reason. Nonempty candidates requires empty no_change_reason. Bounded output size follows Host limits; no mandatory number of changes.

`EffectReceipt` fields: operation_id, payload_digest, status:applied/no_change/submitted/rejected/unknown, target_ref:string, revision:uint64, error_code:string. Unused ref/code is empty and unused revision is zero. A Persona/capability proposal returns submitted, never applied. Lookup absence is a typed `effect_not_found` error; ambiguous crash state is unknown, never absence. Lookup is scoped by the bound Domain.

Durability: backend mutations use transactionally persisted receipts. Work/Persona paths without proven atomic receipt are non-replayable; an unknown outcome stops the graph as recovery_required. A workflow Journal entry alone does not prove the target write committed. Do not add a parallel ACTMEM store to manufacture that proof. Known receipts allow memory-effect deduplication; no effect is blindly retried after timeout. Reuse an operation ID with changed digest -> idempotency_conflict.

## 5. Frozen Core and authority

FrozenCore v2 has `schema_version:"laputa.frozen-core/v2"`, `session_id`, `captured_at`, `sections:[]FrozenSection`. FrozenSection uses `kind` (named enum), `content`, `source_revision:uint64`, `source_hash:string`. Exact ordered kinds are mission, identity, relationship, redline, user, dream, dark. WORLD and ACTMEM never appear. Seven slots are validated explicitly, not by arbitrary extensible map.

Unassigned Mission slot has empty content/revision zero and explicit mission_status:unassigned at the envelope level; assigned has mission_status:assigned and a positive source_revision. No generated Mission. Existing six projection limits remain unchanged. Add Mission without shifting existing numeric kind identities in storage. User-facing human operations authenticate outside this envelope. Old v1 is not silently relabelled v2.

The `/v2/recall/*` `frozen_core` field and `agentapi` `BootstrapResponse`/`ContextView` carry this envelope directly (`agentapi.FrozenCore = evolution.FrozenCoreV2`). `personactx` is the sole v2 assembler. A session admitted under the numeric six-slot v1 shape is never relabelled: reading it returns `recovery_required` and the host must open a new session.

Mutation checks occur again before effects: `RunBinding.CheckMissionRevision` re-verifies the pin — changed Mission -> mission_revision_changed; revoked scope -> authority_denied. New sessions observe applied revisions; admitted session snapshot remains immutable. Direct agent Dream tool uses the existing P16 revision-aware domain path with reason; evolution Effect intentionally cannot encode Dream or Mission writes.

## 6. Backend boundary and receipts

Package `garden/memory` owns the backend DTOs. Reuse existing agentapi card/evidence values through a one-way contract import or explicit adapter conversion; no vendor type crosses this boundary. The backend does not import agentapi if agentapi imports memory.

```go
type Backend interface {
    Capabilities() Capabilities
    Search(context.Context, AuthorizedSearch) (CardPage, error)
    Expand(context.Context, AuthorizedExpansion) (EvidencePage, error)
    Mutate(context.Context, AuthorizedMutation) (MutationReceipt, error)
    MutationStatus(context.Context, string) (MutationReceipt, error)
    Health(context.Context) (Health, error)
    Close() error
}
```

Backend instances are bound to subject and write destination; MutationStatus cannot reveal another instance's records. AuthorizedSearch: scopes[]Scope, query, cursor, limit, budget_chars. AuthorizedExpansion: scope, card_id, expected_revision, budget_chars. A missing/changed revision fails rather than combining check and later refetch of different content. Cursor is opaque and bound to scope/query; replay under another binding fails.

AuthorizedMutation: scope, destination_id, operation_id, payload_digest, operation, record_id, expected_revision, expected_absent, body, sources, inference. Create requires expected_absent=true, revision=0 and a host-assigned record_id; update/tombstone require expected_absent=false and positive revision. Scope/destination must equal the bound writer. MutationReceipt matches EffectReceipt plus canonical/index status (accepted/completed/pending/ready/failed as applicable); canonical commit is distinct from index readiness.

Capabilities has explicit booleans search, expand, mutate, mutation_lookup, vector, timeline, knowledge_graph. First four are required for the selected primary writer. Health carries available/degraded/unavailable, reason_code and derived_index_state. CardPage has items[]MemoryCard, next_cursor; EvidencePage has items[]EvidenceFragment, truncated:bool. No hidden total counts. Scope/revision/status are included in card and evidence envelopes so Garden can verify adapter results. Existing card-before-evidence and validity rules apply.

Mentle mapping: `facade/canonical.go` CreateMemory/UpdateMemory/DeleteMemory SQL transactions and idempotency table; `facade/outbox.go` index jobs. Add receipt semantics for all three mutation forms within the same canonical transaction. Do not use an index as recovery authority. Persist operation ID/digest/scope/destination together; no hash-only cross-scope deduplication.

## 7. Errors and contract-test cases

Stable codes: invalid_schema, invalid_scope, authority_denied, revision_conflict, idempotency_conflict, effect_not_found, outcome_unknown, mission_revision_changed, actmem_cap_exceeded, actmem_format_error, backend_unavailable, capability_unavailable, recovery_required. They map to existing domain errors rather than creating duplicate exceptions. Only transient classified failures are retryable; unknown outcome is not auto-retryable. Error messages omit hidden scope/record content.

S01 defines test fixtures (not production behavior) for:

- TestScopeExactMatch: A/X != A/Y != B/X; empty workspace with workspace kind rejected.
- TestEffectStrictUnion: unknown kind, two payloads, Dream/Mission, duplicate JSON keys all fail.
- TestFrozenCoreV2: seven named kinds only; six-slot v1 and WORLD/ACTMEM insertion fail.
- TestEffectReplayDigest: same op/digest gives same receipt; changed payload/scope conflicts.
- TestActmemGoldenGrammar: valid multiline entries round-trip; delimiter collision/orphan scope metadata fails.

Each owning Story adds behavior tests listed in its plan. S01 fixtures do not assert unimplemented integration success.

## 8. Changelog — interface changes recorded during execution

### S03 — scoped ACTMEM ops (diva-cognitive/v1-review-1, amended)

- New stable codes join section 7: `actmem_unclassified_head`, `actmem_revision_conflict`,
  `actmem_fold_conflict`, `actmem_invalid_edit`, `actmem_invalid_entry`,
  `actmem_scope_forbidden`, `actmem_capsule_invalid`, `actmem_malformed`,
  `actmem_storage_error`. HTTP map: 400 invalid_* / malformed / format /
  unclassified / scope; 404 capsule; 409 revision + fold conflicts; 413 cap.
- `PUT /v2/actmem` is owner-only (`actmem_save` op, user capability): body is
  `{markdown, base_revision}` — a validated v2 whole-save whose stored
  revision is always head+1. Free-text section fields are gone.
- `POST /v2/actmem/maintenance` operations are now the typed set
  `system_append` (entry{section,scope,field,session_id,event_id,body,sources}),
  `work_patch` (contracts WorkPatch), `fold_session` (session_key). The retired
  `append_pulse`/`append_recap`/`edit_work`/`complete_open_item`/`drop_item`
  operations are gone — no compatibility mapping.
- `GET /v2/actmem` is scope-split: read/user/operator principals receive the
  owner document projection; agent principals receive scoped `entries[]`
  bounded by `ActmemWorkspace` (host-configured, never request-supplied).
- `POST /v2/actmem/query` executes under the caller's derived scope; hidden
  workspaces contribute no hits.
- `DELETE /v2/actmem/capsules/{name}` is owner-only (`actmem_save`).
- agentapi gains `actmem_save` (user-only) and typed service ops
  `ReadActivity`, `ApplyWorkPatch`, `AppendActivity`, `FoldSession` alongside
  the scoped `QueryACTMEM`; contract DTOs alias evolution types.
- Fold capsules record `session_key`/`created_at`/`fold_digest`/`entries`
  front matter; reader tolerates both legacy and fold formats.

### S04 — injected backend, scoped captures, mutation receipts (diva-cognitive/v1-review-1, amended)

- `memory.AuthorizedSearch` gains `Collection`; `AuthorizedExpansion.Scope`
  becomes `Scopes []evolution.Scope` (the caller admits a union; it never
  declares which scope a record lives in). Scope strings use `scope/v1`
  encoding (`memory.EncodeScope`/`DecodeScope`/`ScopeVisible`); requests carry
  no raw scope anywhere.
- `memory.Backend` port implemented by `backends/mentle.Adapter` (canonical
  facade only, never search indexes): `Search`, `Expand`,
  `Mutate`, `MutationStatus`, `Capabilities`, `Health`, `Close`,
  `BoundScope`, `BoundDestination`, plus `CreateMemory` for the ingest writer.
  One backend binds one scope+destination; `AuthorizedMutation.MatchesWriter`
  rejects foreign writes; injected `runtimecore.Config.Backends` wins and a
  revoked destination never falls back.
- `agentapi.CardSearch` drops `Scope`; `agentapi.EvidenceRead` gains
  `Items []EvidenceRef{CardID, ExpectedRevision}` — `GET
  /v2/materials/cards/{id}/evidence` now requires `expected_revision` and
  expands by exact ID + revision, never search-hit ids.
- `facade` gains `Mutate`/`MutationStatus`/`MutationRequest`/`MutationReceipt`/
  `ErrMutationNotFound`, the `mutation_receipts` schema (registered inside
  `canonicalSchema`), `OpenCatalogService` (writable no-model catalog with
  BM25 projection), `Service.IsReadOnly`, and `ErrReadOnly`. Canonical
  Create/Update/Delete and `outbox` transaction boundaries are unchanged;
  receipts record inside those same transactions.
- `hybrid.Searcher` gains `Lexical`, `IndexBM25Drawer`, `RemoveBM25` for the
  no-model projection path in `applyIndexJob`.
- ingest `TransientEntry` and the `ingestions` row carry the encoded scope
  (derived from the stored `workspace` column); undecodable spool scopes fail
  visibly as `failed`, never re-routed.
- `MemoryCard`/`EvidenceFragment` live in `garden/memory/cards.go`;
  `agentapi` keeps type aliases (breaks the memory→agentapi import edge).
