# ADR-0017: DIVA Cognitive Contract Deltas vs ADR-0012/0014/0016

**Status:** proposed record (S01 shared contracts frozen; downstream stories not started)
**Date:** 2026-10-02
**Decision owner:** project owner
**Refines:** ADR-0012, ADR-0014, ADR-0016
**Contract revision:** `diva-cognitive/v1-review-1` (`docs/superpowers/plans/2026-10-02-diva-cognitive/contracts.md`)
**Evidence:** `laputa/evolution/`, `laputa/evolution/testkit/`, `garden/memory/`, `garden/agentapi/` (S01, commit `547e35d`)

## Purpose

Record the deliberate contract deltas frozen by Story S01 against the three
governing ADRs, so downstream stories implement against reviewed changes
instead of re-deriving them. This ADR is a change record; it grants no new
authority and enables no production inference or write path.

## Mission and Frozen Core (vs ADR-0012)

ADR-0012 fixes Frozen Core at six bounded projections of the first six
authority files. The v2 envelope adds a leading `mission` slot:

- Exact ordered v2 kinds: `mission, identity, relationship, redline, user,
  dream, dark` — seven named slots, validated explicitly. WORLD and ACTMEM
  never appear.
- The envelope carries `schema_version:"laputa.frozen-core/v2"` and
  `mission_status:unassigned|assigned`. An unassigned mission slot has empty
  content and `source_revision:0`; an assigned slot requires a positive
  source revision. No generated Mission.
- The persona storage intenum is unchanged (`KindIdentity=0` …
  `KindDark=6`). Mission is a new named wire kind, not a renumbered integer.
- The existing `agentapi.FrozenCore` six-slot wire is unchanged. v1 data
  labelled v2, v2 data with a WORLD/ACTMEM slot, and wrong-order slots are
  rejected by `evolution.DecodeFrozenCoreV2`. Old v1 is not silently
  relabelled v2.
- Classification: v1 six-slot wire — **Keep** (existing wire profile);
  six-slot data labelled v2 — **Drop** (rejected); automatic Mission
  generation — **Drop** (envelope requires explicit status).

## Scope (vs ADR-0012/0016)

- New trusted tuple `evolution.Scope{subject_id, kind, workspace_id}` with
  `kind: personal|workspace`. Comparison is exact-tuple equality, never
  prefix-based. An absent scope is invalid input.
- `agentapi.Binding` gains `workspace_id` (additive, `omitempty`) and
  `agentapi.Config` gains `WorkspaceID`. The host issues it at `Open`; a
  request carrying a conflicting binding already fails `invalid_binding` —
  that enforcement now covers workspace. Empty `workspace_id` is the
  implicit personal workspace.
- `Binding.TrustedScope()` derives the contract scope: `ProfileID` is the
  stable subject id (per the ADR-0016 embedding contract, the host binds the
  profile; no model-supplied grant exists).
- Deliberate behavioral change: requests bearing a mismatched
  `workspace_id` are now rejected; previously the field did not exist.

## Backend contract (vs ADR-0014)

- New package `garden/memory` owns the section 6 backend boundary:
  `Backend` port (`Capabilities`, `Search`, `Expand`, `Mutate`,
  `MutationStatus`, `Health`, `Close`), `AuthorizedSearch`,
  `AuthorizedExpansion`, `AuthorizedMutation`, `MutationReceipt`,
  `CardPage`, `EvidencePage`, `Capabilities`, `Health`.
- Backend instances are bound to one subject and one write destination;
  mutation scope/destination must equal the bound writer exactly
  (`invalid_scope` on mismatch).
- `MutationReceipt` keeps canonical commit distinct from derived-index
  readiness (`canonical_status` vs `index_status`). Mentle canonical SQLite
  remains the sole memory authority; receipts persist inside the canonical
  transaction in the downstream S03 backend story, never via an index.
- `agentapi.EvidenceFragment` envelopes carry `scope`, `revision`, `status`
  (additive, `omitempty`) so Garden can verify adapter results. Facade
  reads use explicit adapter conversion; no vendor type crosses the
  boundary.
- Classification: canonical SQLite authority — **Keep**; hash-only
  cross-scope deduplication — **Drop** (operation id + digest + scope +
  destination persisted together); WAL-as-authority — **Drop** (unchanged
  from ADR-0014).

## Trusted strategy binding (vs ADR-0016)

- `evolution.RunBinding{subject_id, workspace_id, destination_id,
  policy_revision, strategy_digest, mission_revision}` is populated by the
  trusted host composition, never by model input. `mission_revision=0` means
  unassigned and cannot admit mission-driven autonomous effects.
- `evolution.Model` and `evolution.Domain` are consumer-defined ports; the
  evolution library imports no Garden/ViVy code and no INOFY runtime. The
  fixed node identities `laputa.evolution.{collect,prepare,reconcile,
  reflect,effects,finish}@1` are frozen.
- The closed `Effect` union (`work_patch`, `memory_mutation`,
  `persona_request`, `capability_proposal`, `reflection_note`) rejects
  Mission/Dream variants, unknown kinds, mismatched payload keys, unknown
  fields and duplicate JSON keys before mutation. Effect digests cover the
  normalized payload plus effective scope/destination; reuse of an
  operation id with a changed digest yields `idempotency_conflict`.
- ACTMEM v2 grammar (`laputa.actmem/v2`) is frozen: YAML front matter keys
  exactly `schema/revision/updated/entries`, sections `## Pulse`,
  `## Recap`, `## Work` in order, full-line `e_[0-9a-f]{32}` delimiters.
  Legacy unclassified Markdown stays owner-readable but is excluded from
  agent projections; no auto-upgrade.
- Classification: legacy unclassified ACTMEM records — **Deferred** (explicit
  owner decision; conversion, if ever, is a separate revisioned tool);
  model-authored scope grants — **Drop**.

## Deliberate breaking changes

None at the wire level: every addition is a new type or an `omitempty`
field. Deliberate behavioral tightenings recorded above: strict JSON
decoders reject previously tolerated unknown fields/duplicate keys;
mismatched `workspace_id` on requests is rejected; invalid v2 ACTMEM input
fails with `actmem_format_error` instead of silent reinterpretation.

## Guard updates required by the accepted changes

Recorded only — AGENTS.md and guard tables are not edited without explicit
owner permission:

- Root `AGENTS.md`: "Frozen Core contains bounded, session-frozen
  projections of the first six authority files only" needs v2 wording (seven
  named slots including the mission slot), and the seven-file authority
  list needs MISSION.MD added (the design's eight-file roster, pinned as
  `evolution.AuthorityKinds`).
- `laputa/AGENTS.md`: "Frozen Core is a bounded projection of the first six
  authority files" needs the same v2 update, and its authority-file list
  gains MISSION.MD.
- `garden/AGENTS.md`: Context Discipline's six-file frozen list needs the
  v2 roster plus `mission_status` envelope; the new `garden/memory` backend
  boundary package should be named in the boundary table.
- `docs/architecture/AGENTS.md`: Active ADRs table needs a 0017 row.
- `garden/internal/architectureguard`: guard needles do not yet cover
  v2-specific regressions (WORLD/ACTMEM in a v2 frozen slot, model-written
  `workspace_id` grants, Dream/Mission effect variants); add needles when
  the corresponding production paths land (S02–S06).
- `docs/AGENTS.md` read order may add this record under `architecture/`.

## Out of scope for S01

No runtime, Journal, prompt text, INOFY definition, backend implementation,
or UI work is delivered here; downstream Stories S02–S07 implement behavior
against these frozen contracts. S08 remains blocked on the external desktop
bridge (DIVA-NEXT-P0).
