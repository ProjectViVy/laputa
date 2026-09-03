# ADR-0013: Laputa Clean-Break Implementation Architecture

**Status:** accepted design
**Date:** 2026-08-14
**Decision owner:** project owner
**Implements:** ADR-0012
**Refines:** ADR-0006, ADR-0007, ADR-0009, ADR-0010, and ADR-0011 within their existing boundaries
**Supersedes as implementation guidance:** the current `laputa/governance` JSON section engine, Garden `authority` projection, generic governance HTTP mutation routes, legacy cognitive WORLD projection, and section-backed activity/report paths.

---

## 1. Goal

Define the deletion-first Go architecture that implements ADR-0012 without a JSON compatibility path.

The target must establish a Markdown Persona store, separate `ACTMEM.MD`, session-frozen Frozen Core, tool-only `WORLD.MD`/`ACTMEM.MD`, Persona-specific review/history, and a bounded EvoMap candidate input. It must retain Garden's runtime and Mentle/EvoMap boundaries.

This ADR is an implementation design. It does not authorize retaining any legacy runtime path while the replacement is introduced.

## 2. Design Paradigm

**Explicit authority documents plus disposable runtime views.**

```text
Profile directory
  persona/                         Laputa authority and history (DIVA-verified layout)
    IDENTITY.MD                    complete Markdown document
    RELATIONSHIP.MD                complete Markdown document
    REDLINE.MD                     complete Markdown document
    USER.MD                        complete Markdown document
    DREAM.MD                       optional complete Markdown document
    DARK.MD                        optional complete Markdown document
    WORLD.MD                       explicit tool-only complete document
    history/<FILE_NAME>/<rev>.md   append-only complete snapshots (immutable)
    history/<FILE_NAME>/<rev>.diff unified diff produced at write time
    history/<FILE_NAME>/log.jsonl  append-only revision metadata (newest first)
    requests/<uuid>.json           agent-proposed PersonaChangeRequest entries
  ACTMEM.MD                        profile-wide activity memory, outside persona/

Garden runtime
  session Frozen Core              in-memory immutable snapshot at session start
  activity / checkpoint / spool    SQLite runtime state, never Persona or ACTMEM
  recall ContextView               disposable; Frozen Core + Mentle evidence only
  EvoMap SQLite stores             candidate, proposal, artifact, mailbox, Hub state

Mentle
  raw material -> cards -> explicit bounded evidence reads -> provenance
```

No target component reads `~/.laputa/sections`, maps a numbered section to an authority file, or accepts a structured Persona body.

## 3. Architectural Decisions

### AD-1: One Markdown document store replaces the section registry

Create `laputa/persona` as Laputa's public Persona package. It defines a closed `DocumentKind` enum for the seven exact filenames, document limits, atomic full-document writes, list/read access, initialization, revision metadata, and snapshot history.

The store accepts `string` content only. It exposes no `map[string]any`, `Patch`, dot-path mutation, registry mutation, or generic schema extension API.

| Kind | File | Limit | Frozen Core |
| --- | --- | ---: | ---: |
| Identity | `IDENTITY.MD` | 800 | 200 |
| Relationship | `RELATIONSHIP.MD` | 600 | 120 |
| Redline | `REDLINE.MD` | 400 | 200 |
| User | `USER.MD` | 800 | 160 |
| Dream | `DREAM.MD` | 40 | 10 |
| Dark | `DARK.MD` | 300 | 60 |
| World | `WORLD.MD` | 1000 | never |

A document write validates visible character count before its atomic replace. It writes a complete prior/current snapshot and revision metadata under the Persona history root in the same durability boundary. A failed validation or failed history write leaves the current authority document unchanged.

`Initialize` is a single guarded operation: only if all five required files (`IDENTITY.MD`, `RELATIONSHIP.MD`, `REDLINE.MD`, `USER.MD`, `WORLD.MD`) are absent may it atomically create those five documents and their initial revisions. Partial pre-existing state is an integrity error requiring explicit user repair. It never creates `DREAM.MD` or `DARK.MD`.

### AD-2: Persona review is a dedicated content workflow

Create `laputa/persona/review` or an adjacent `laputa/persona` review service with proposal records separate from Chat Approval Center and EvoMap.

A Persona change proposal contains an immutable before revision, full proposed Markdown body, document kind, actor, reason, and textual diff. Approving it re-validates the current revision; a stale base revision rejects rather than overwriting later user edits.

| Write class | Target documents | Action |
| --- | --- | --- |
| User direct write | Identity, Relationship, Redline, User preferences, World | atomic document write + history |
| Agent reviewed write | Identity, Relationship, Redline, World, User preferences | create Persona review; only approval writes |
| Agent direct write | Dream, Dark, User observations | atomic document write + history |
| ACTMEM maintenance | `ACTMEM.MD` | direct write; no Persona review |

The service does not reuse `governance.GovernedService`, `MutationRequest`, generic audit authorization, or Chat Approval Center state.

### AD-3: ACTMEM is one explicit whole-file capability

Create `laputa/actmem` with its own path configuration and complete-file atomic write semantics. `ACTMEM.MD` is profile-wide but physically outside the Persona authority directory.

V1 exposes a stable read/query capability that returns bounded text or an explicit query result. It does not add automatic promotion into Mentle, new activity checkpoint persistence, a generic mutation endpoint, or a default prompt projection. Direct daily maintenance writes remain a lightweight activity action with a small append-only operation log only if implementation needs operator diagnostics; they are not Persona history and not Approval Center records.

### AD-4: Frozen Core is captured once per session

Create `garden/internal/personactx` as Garden's consumer-facing projection adapter. At session start it reads the first six Persona documents through a narrow Laputa reader interface, truncates each by the fixed Frozen Core budget, and stores the resulting immutable `FrozenCore` with a session id and source revisions.

Frozen Core is the only Laputa content automatically available to ContextView. It contains document projections, not JSON-derived policy fields. The content is not refreshed during the session.

`WORLD.MD` and `ACTMEM.MD` never implement the Frozen Core interface and are structurally absent from `FrozenCore`, `FastRequest`, `DeepRequest`, `ContextView`, bootstrap responses, and ContextView assembly functions.

### AD-5: Explicit tool reads are separate from recall

Garden receives two narrow read services:

- Persona tool read: explicit full document read for a named authority document, including `WORLD.MD`.
- ACTMEM tool read/query: explicit bounded read/query of `ACTMEM.MD`.

The HTTP adapter may expose these tools for the local Console and host adapters, but they must not be called by `FastService`, `DeepService`, bootstrap, planner, graph traversal, or automatic admin polling. `WORLD.MD` is a complete Markdown document; there is no `WorldStore.Project`, scope/budget claim projection, or `WorldClaim` target model.

### AD-6: Runtime checkpoint state stays in Garden SQLite

`activity.WorkingSet`, checkpoint recovery, activity events, transient spool, recall traces, and session records remain Garden runtime data. Replace the section-backed `activity.Checkpointer` with a SQLite-backed checkpoint repository in `garden/internal/activity`.

This prevents both failure modes: working-set state masquerading as Laputa authority, and `ACTMEM.MD` being used as a runtime serialization format.

### AD-7: Recall becomes Frozen Core plus Mentle evidence

Replace `garden/internal/authority.GovernanceProjection` with `personactx.FrozenCore`. `FastService` and `DeepService` retain card discovery, bounded evidence reads, recall traces, scope filtering, degradation behavior, and explicit Deep Recall capabilities.

Remove these target fields and behaviors:

- `GovernanceProjection`, `IdentityRef`, `AllowedKinds`, `DeniedSources`, `WorkingSetRefs`, and section-derived `FrozenRefs`.
- `ContextView.World`, world budget, world scope splitting, `WorldProjector`, and `worldContext`.
- `memory_md` as a working-set or recall source.

Recall's runtime policy remains in Garden. If `MEMRULES.MD` survives a later implementation decision, it is loaded only as a Garden policy input and cannot add authority content to ContextView.

### AD-8: Reports leave Laputa

`garden/internal/report` remains a Garden-owned report and human-module service. Replace `report.GovernedPublisher` with a report repository backed by the existing Garden SQLite state database or a dedicated report table under that database.

Daily, weekly, and monthly reports remain human/report artifacts. They do not become Persona documents, `ACTMEM.MD`, or EvoMap capability artifacts, and they are not automatic ContextView sources.

### AD-9: EvoMap remains the sole artifact lifecycle authority

Retain `garden/internal/evolution`, `garden/internal/mailbox`, their SQLite stores, privacy gate, Hub transport, review state, retries, and dead-letter behavior.

Add a narrow `EvolutionCandidateInput` assembler in `garden/internal/evolution` rather than granting EvoMap a Laputa store dependency. The assembler may accept:

- Activity outcome/reflection references.
- An explicit bounded `ACTMEM.MD` excerpt provided by an authorized caller.
- Mentle evidence references and bounded evidence metadata.
- Recall trace references.

It may not receive Persona document bodies, Persona history, an authority store, or a direct artifact install/write capability. Existing `PortableSkill` and `HostArtifact` types may remain as EvoMap-owned lifecycle records, but host installation stays blocked until EvoMap approval and a future host adapter contract.

### AD-10: HTTP becomes resource-oriented and domain-specific

Remove generic governance routes. Add domain-specific local HTTP groups after their packages exist:

| Group | Target routes | Notes |
| --- | --- | --- |
| Persona read | `GET /v2/persona/documents`, `GET /v2/persona/documents/{kind}` | `kind` is one of the closed document identifiers; complete Markdown returned explicitly |
| Persona write/review | `PUT /v2/persona/documents/{kind}`, `POST /v2/persona/reviews`, `GET /v2/persona/reviews`, `POST /v2/persona/reviews/{id}/approve`, `POST /v2/persona/reviews/{id}/reject`, `GET /v2/persona/history/{kind}` | user direct writes and agent-reviewed writes follow AD-2 |
| ACTMEM tool | `GET /v2/actmem`, `POST /v2/actmem/query`, `PUT /v2/actmem` | never consumed by recall or automatic Console polling |
| Session/context | existing bootstrap/fast/deep endpoints with new Frozen Core metadata | response cannot contain World/ACTMEM automatic projection |
| Evolution | existing `/v2/evolution/*` and `/v2/mailbox/*` | retained, then extend candidate-input endpoint only after explicit schema acceptance |

The final response DTOs use document and revision identifiers. They do not accept `action`, `path`, `value`, JSON Patch, section names, or arbitrary object bodies.

### AD-11: Console has four workspaces, not a generic Governance Map

Replace the old Governance Map route/data model with four domain workspaces:

1. **Persona**: document viewer/editor, bounded Frozen Core preview, dedicated review queue, and revision history.
2. **Memory**: explicit ACTMEM read/query/maintenance view plus Mentle materials/evidence and runtime activity surfaces. It clearly separates tool-only ACTMEM from Mentle evidence.
3. **Evolution**: existing EvoMap runs, candidates, proposals, artifacts, mailbox, privacy state, and Hub state.
4. **Chat Approval**: dangerous runtime-operation requests only; it does not render Persona review or ACTMEM maintenance as approvals.

The Console must not automatically fetch or expose `WORLD.MD` as a dashboard panel. A user opens it by explicit Persona document selection. All pending surfaces must label their truth source as `live`, `accepted-design`, or `deferred`; production views use no mock state.

## 4. Package and File Plan

### Create

| Path | Responsibility |
| --- | --- |
| `laputa/persona/model.go` | closed document kinds, filenames, limits, revision/document DTOs, domain errors |
| `laputa/persona/store.go` | authority directory layout, reads, atomic whole-document writes, initialization, locking |
| `laputa/persona/history.go` | append-only snapshot/revision metadata and textual diff generation |
| `laputa/persona/review.go` | Persona proposal state and compare-and-swap approval workflow |
| `laputa/persona/*_test.go` | initialization, limits, atomicity, history, stale review, no Dream creation |
| `laputa/actmem/store.go` | distinct ACTMEM read/write/query boundary |
| `laputa/actmem/*_test.go` | restart continuity, bounded query, no Persona side effects |
| `garden/internal/personactx/frozen.go` | session-frozen six-document projection and revision references |
| `garden/internal/personactx/*_test.go` | exact projections, immutability, World/ACTMEM exclusion |
| `garden/internal/activity/checkpoint_store.go` | SQLite checkpoint persistence for `WorkingSet` |
| `garden/internal/activity/checkpoint_store_test.go` | restart restore and no Laputa dependency |
| `garden/internal/server/persona_handlers.go` | Persona resource/read/review endpoints |
| `garden/internal/server/actmem_handlers.go` | explicit ACTMEM read/query/maintenance endpoints |
| `garden/internal/server/persona_handlers_test.go` | HTTP contract and authorization/write-class checks |
| `garden/internal/server/actmem_handlers_test.go` | explicit-only route behavior and bounded read checks |
| `garden/internal/evolution/candidate_input.go` | bounded runtime/ACTMEM/evidence candidate assembler |
| `garden/internal/evolution/candidate_input_test.go` | rejects Persona bodies and unbounded content |
| `garden/console/src/pages/PersonaPage.tsx` | Persona workspace |
| `garden/console/src/pages/MemoryPage.tsx` | ACTMEM/Mentle/activity workspace |
| `garden/console/src/pages/ChatApprovalPage.tsx` | runtime approval workspace |

### Replace

| Existing path | Replacement |
| --- | --- |
| `laputa/governance/engine.go` | `laputa/persona` and `laputa/actmem`; remove after callers move |
| `laputa/governance/governed.go` | Persona review service and domain-specific server handlers; remove |
| `garden/internal/authority/projection.go` | `garden/internal/personactx/frozen.go` |
| `garden/internal/activity/checkpoint.go` | SQLite checkpoint repository |
| `garden/internal/recall/fast.go` and `context.go` | Frozen Core plus Mentle-only ContextView assembly |
| `garden/internal/report/publisher.go` | Garden SQLite report publisher |
| `garden/internal/server/authority_handlers.go` | Persona handlers; remove generic mutation API |
| `garden/internal/server/cognitive_handlers.go` | explicit Persona `WORLD.MD` document read; remove world projection API |
| `garden/main.go` | new composition root with Persona, ACTMEM, SQLite checkpoints, and no JSON/cognitive WorldStore wiring |
| Console `GovernanceMap.tsx` / `data/governance.ts` | Persona, Memory, Evolution, and Chat Approval workspace data/views |

### Delete after replacement tests pass

- `laputa/governance/engine.go`, `governed.go`, their JSON SectionStore tests, and all exported section constants/registry APIs.
- `laputa/governance/cognitive/world.go` and target-facing World claim/default code.
- Generic governance audit/mutation routes and Console types/callers.
- Section-backed report publisher and section-backed activity checkpoint.
- All tests, fixtures, labels, route docs, and seed content that make `.laputa/sections`, numbered names, JSON Patch, `memory_md`, `Commitment`, or `Preferences` appear usable.

Existing `laputa/governance/rhythm`, scheduler, wakeup, and web packages require a separate deletion-or-rehome review before their code is carried forward. They must not be connected to the Markdown Persona store by default merely because they live under the legacy package.

## 5. Deletion-First Execution Order

### Phase 0: Contract and test harness

1. Add target package tests that prove the seven-file model, required-five initialization rule, limits, complete-document atomic writes, textual history, review concurrency, and ACTMEM restart continuity.
2. Add repository scanner tests that fail on target-runtime imports of `laputa/governance`, `.laputa/sections`, `SectionMemoryMD`, JSON Patch, or `WorldProjector` after the cutover phase.
3. Do not change HTTP routes or Console before a target store test suite exists.

### Phase 1: Introduce isolated target storage

1. Implement `laputa/persona` and `laputa/actmem` with no import of `laputa/governance`.
2. Implement `garden/internal/personactx` against a narrow Persona read interface.
3. Prove five-file initialization, direct/reviewed writes, snapshots, and Frozen Core capture in unit tests.
4. Keep new packages unwired to the old composition root until Phase 2's full route replacement is ready; this is staging, not compatibility.

### Phase 2: Replace Garden runtime dependencies atomically

1. Move WorkingSet persistence from `SectionMemoryMD` to Garden SQLite.
2. Replace `GovernanceProjection` with `FrozenCore` in Fast Recall, Deep Recall, bootstrap, traces, and their tests.
3. Remove `WorldProjector`, `ContextView.World`, world assembly, scope projection, and `GET /v2/cognitive/world` in the same change.
4. Replace report publishing with Garden SQLite in the same change.
5. Rewire `garden/main.go` so no target runtime initializes `governance.NewFileStore`, a JSON audit log, a legacy cognitive WorldStore, or `GovernedService`.

### Phase 3: Replace APIs and Console surfaces

1. Add Persona and ACTMEM domain routes and complete HTTP tests.
2. Delete `/v2/governance/projection`, `/v2/governance/mutations`, `/v2/governance/audit`, and `/v2/cognitive/world`.
3. Replace Console Governance Map with Persona, Memory, Evolution, and Chat Approval workspaces. Preserve existing Evolution/Mailbox pages and APIs; do not move them into Persona.
4. Ensure WORLD is shown only after an explicit Persona document request. Ensure ACTMEM is fetched only after an explicit Memory workspace request/action.

### Phase 4: Reconcile EvoMap input and remove legacy code

1. Add `EvolutionCandidateInput` validation and candidate assembly from explicit bounded ACTMEM excerpt, activity/trace refs, and Mentle evidence refs.
2. Prove candidate assembly rejects Persona body/history input and cannot create/install a host artifact.
3. Remove legacy governance imports, packages, tests, fixtures, documentation references, and Console labels.
4. Run deletion scans before declaring the clean break complete.

## 6. HTTP and Data Invariants

- Persona writes are complete Markdown replacements. No partial field mutation endpoint exists.
- `WORLD.MD` and `ACTMEM.MD` are absent from all automatic ContextView, bootstrap, Fast Recall, Deep Recall, planner, and trace-content fields.
- Frozen Core contains bounded projections of exactly Identity, Relationship, Redline, User, Dream, and Dark; it never reads World or ACTMEM.
- `ACTMEM.MD` does not share a table, directory, revision namespace, review queue, or write policy with Persona documents.
- Mentle evidence remains source-addressed and bounded. It cannot write Persona, ACTMEM, or EvoMap lifecycle state.
- EvoMap reads only a bounded candidate input and owns every state transition after candidate creation. No Laputa package imports EvoMap.
- Chat Approval Center cannot approve a Persona content proposal and cannot be used as an ACTMEM write wrapper.
- Existing historical JSON files remain inert. The running program never probes for their presence.

## 7. First Vertical Slice Acceptance Matrix

| Proof | Test level | Required result |
| --- | --- | --- |
| Required-five initialization | Laputa unit | all five absent creates exactly five files and revisions; partial state fails; Dream/Dark absent |
| Limits and atomicity | Laputa unit | over-limit write fails without changing authority or history; successful write advances revision |
| Persona review | Laputa unit | agent Identity change remains pending; approved current revision writes; stale approval fails |
| Frozen Core | Garden unit | six bounded documents captured at session start; later source edit does not change same session snapshot |
| Tool-only World/ACTMEM | Garden unit + HTTP | Fast/Deep/bootstrap responses and ContextView contain neither; explicit Persona/ACTMEM read succeeds |
| Restart behavior | integration | ACTMEM and SQLite checkpoint survive restart independently |
| Mentle boundary | Garden unit | cards/evidence work without granting Mentle authority writes |
| EvoMap boundary | evolution unit | bounded candidate accepts evidence/reflection refs; rejects Persona bodies; cannot auto-install/publish |
| Legacy deletion | repository scan | no target-runtime import or route/DTO/fixture uses JSON section concepts |
| Console truth | Console integration | no Governance Map/world auto-poll; Persona/Memory/Evolution/Chat Approval routes use live domain APIs only |

## 8. Keep, Cut, Deferred, Drop

| Classification | Implementation decision |
| --- | --- |
| Keep | Garden activity/event/spool/trace runtime; Mentle cards/evidence/provenance; EvoMap and mailbox stores/policy; reports and human modules; local admin aggregation; degraded Mentle behavior |
| Cut | SectionStore, Engine section registry, JSON authority persistence, generic governed mutation/audit, JSON World claim projection, section-backed working-set checkpoint, section-backed report publisher, generic governance Console route |
| Deferred | ACTMEM query semantics beyond bounded read; ACTMEM-to-Mentle promotion; full Persona review UX; exact revision metadata storage format; EvoMap candidate payload schema; artifact installer/host adapter contract; fate of legacy rhythm/scheduler/wakeup/web packages |
| Drop | Migration/import/dual-read/dual-write/fallback; JSON descriptor aliases; automatic WORLD/ACTMEM injection; World scope projection; user-defined authority kinds; any direct Laputa Skill/SOP creation |

## 9. Risks and Controls

| Risk | Control |
| --- | --- |
| A staged package coexists indefinitely with the JSON engine | Phase 2 requires composition-root replacement and deletion in the same accepted change; scanner tests prohibit old imports afterward |
| Persona review becomes a generic approval clone | Separate package, data types, routes, and tests; no dependency on Chat Approval Center or `GovernedService` |
| ACTMEM becomes a hidden new LTM store | Whole-file tool contract, no automatic Mentle promotion, no recall injection, explicit deferred promotion policy |
| WORLD returns through a new scoped API | No World projection type/interface/field remains; only an explicit whole-document Persona read exists |
| EvoMap loses artifact ownership | Candidate assembler only supplies bounded input; all proposal/review/install/publish transitions remain in `internal/evolution` |
| Existing reports or checkpoints silently disappear | Rehome them to Garden SQLite before legacy code deletion; tests prove restart and report publication |
| Console displays accepted-design data as live | Endpoint-specific truth source and no mock fallback; new routes are hidden until backend routes exist |

## 10. Explicit Non-Goals

- No runtime implementation in this ADR.
- No source-data migration from JSON sections.
- No automatic persona extraction from Mentle or ACTMEM.
- No automatic ACTMEM compaction, tool proliferation, or BML promotion.
- No host adapter installation or automatic Skill/SOP publication.
- No reintroduction of a generic governance registry under a different name.
