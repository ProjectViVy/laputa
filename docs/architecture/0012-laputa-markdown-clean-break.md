# ADR-0012: Laputa Markdown Clean Break and EvoMap Capability Boundary

**Status:** accepted
**Date:** 2026-08-14
**Decision owner:** project owner
**Supersedes:** ADR-0001, ADR-0002, ADR-0003, ADR-0004, ADR-0005, and ADR-0008 as active guidance. Their original texts are preserved in `docs/archive/2026-08-14-laputa-clean-break/`.
**Refines:** ADR-0007 and ADR-0010. EvoMap remains the evolution and capability-artifact domain.
**Does not supersede:** ADR-0006 semantic ingestion, ADR-0009 human report modules, or ADR-0011 recovery and evidence work, except where they name retired Laputa contracts.

---

## 1. Decision

Garden MemoryOS adopts the current Laputa product contract without a Garden-specific variant.

Laputa is the human-readable personality and cognitive-governance surface. It is not a generic JSON section registry, a long-term memory store, an activity scheduler, a report database, an EvoMap mailbox, or a Skill manager.

The current runtime must be rebuilt as a clean break. It must not read, write, map, import, migrate, or fall back to the retired JSON personality descriptors.

## 2. Authority Files

One Diva/profile has one authority directory. The seven authority files live in that same directory, use uppercase names, and contain Markdown bodies only:

| File | Responsibility | Initial setup | Context lane |
| --- | --- | --- | --- |
| `IDENTITY.MD` | Agent identity, personality, expression, current body/form | yes | Frozen Core, 200-character projection |
| `RELATIONSHIP.MD` | User, relationship, and the agent's view of that relationship | yes | Frozen Core, 120-character projection |
| `REDLINE.MD` | User red lines and actions that require asking first | yes | Frozen Core, 200-character projection |
| `USER.MD` | User self-described preferences and agent observations | yes, preferences only | Frozen Core, 160-character projection |
| `DREAM.MD` | Agent's own wish | no | Frozen Core, strict 10-character projection |
| `DARK.MD` | `FEAR` and `SHADOW` exhibits | no | Frozen Core, 60-character projection |
| `WORLD.MD` | Current actionable environment | yes | tool-only; never default context |

Body limits are fixed: Identity 800, Relationship 600, Redline 400, User 800, Dream 40, Dark 300, World 1000 visible characters. Writes over a limit fail; no silent truncation is allowed.

The v1 authority type set is closed. Adding an eighth authority type is an architecture decision, not a user-editable registry change.

## 3. ACTMEM and Context Lanes

`ACTMEM.MD` is the profile-wide cross-session activity memory. It is not a personality authority file, BML/Mentle evidence, a session checkpoint, or a long-term-memory substitute.

Context has exactly three lanes:

| Lane | Rule | Current content |
| --- | --- | --- |
| Frozen Core | Captured at session start and immutable for that session | Bounded projections of the first six authority files |
| Dynamic loading | Refreshable bounded projection | empty in v1 |
| Tool access | Not automatically inserted into the prompt | full `WORLD.MD`, full `ACTMEM.MD`, full authority bodies, history, Mentle evidence, reports |

`WORLD.MD` and `ACTMEM.MD` must not be projected by Fast Recall, Deep Recall, bootstrap, or any automatic ContextView assembly. The stable tool surface has one `actmem` read/query capability. ACTMEM maintenance capabilities are discoverable only through the deferred tool lane.

Garden retains activity events, session checkpoints, raw-first ingest, recall traces, cards, and evidence reads as runtime mechanisms. They do not become an alternate authority file or a substitute for `ACTMEM.MD`.

## 4. Write and History Rules

- User edits to `IDENTITY.MD`, `RELATIONSHIP.MD`, `REDLINE.MD`, `USER.MD` preferences, and `WORLD.MD` save directly.
- Agent changes to identity, relationship, redline, world, or user preferences require Persona-specific content review, not the Chat Approval Center and not a generic governance state machine.
- Agent writes to `DREAM.MD`, `DARK.MD`, and user observations may be direct writes; the user may edit or delete them.
- Persona changes retain append-only complete Markdown snapshots and textual before/after diffs. Complete history is never default context.
- Initial setup appears only when `IDENTITY.MD`, `RELATIONSHIP.MD`, `REDLINE.MD`, `USER.MD`, and `WORLD.MD` are all absent. It atomically writes those five files and initial history. It never creates `DREAM.MD`.
- ACTMEM maintenance is a lightweight direct-write activity. It does not create Persona review, Approval Center requests, generic governance records, or EvoMap proposals.

The old `GovernedService` mutation abstraction may remain only as a temporary implementation seam while being removed. It is not a target-model API for Persona or ACTMEM.

## 5. EvoMap Is the Capability Domain

EvoMap remains a first-class Garden subsystem and is the only domain that owns capability artifacts.

```text
activity events + ACTMEM reflection + Mentle evidence
                        |
                        v
          bounded Evolution candidate/proposal
                        |
                        v
       EvoMap review, evaluation, versioning, mailbox, Hub policy
                        |
                        v
          EvoMap-authorized SOP / Skill artifact lifecycle
```

EvoMap owns proposal state, review, evaluation, artifact versioning, install permission, outbound privacy filtering, Hub publication policy, retry, and dead-letter behavior. Existing mailbox and GEP-A2A transport guarantees remain valid.

Laputa does not create, install, publish, or manage Skills. Garden does not bypass EvoMap to create an artifact. AutoDream-like reflection may produce a bounded EvoMap candidate, but it cannot apply the candidate or alter Persona/BML/Mentle authority as part of that action.

The retired `11-proposal_inbox` is not revived. EvoMap mailbox persistence remains separate from Laputa authority files.

## 6. Module Boundaries

| Module | Owns | Does not own |
| --- | --- | --- |
| Laputa | Seven Markdown authorities, Persona review/history, ACTMEM file semantics | Mentle material, report storage, EvoMap artifacts, generic JSON governance sections |
| Garden | Activity/runtime orchestration, bounded ContextView delivery, host integration, console aggregation | Personality authority, automatic WORLD/ACTMEM prompt injection, capability artifact authority |
| Mentle | Raw materials, evidence, retrieval, indexing, provenance | Persona, ACTMEM, approval, Skill lifecycle |
| EvoMap | Evolution candidates/proposals, evaluation, artifact lifecycle, mailbox and Hub policy | Persona files, ACTMEM ownership, Mentle evidence authority |
| Chat Approval Center | Dangerous runtime-operation authorization | Persona content review, memory CRUD, ACTMEM maintenance, EvoMap artifact lifecycle |

`MEMRULES.MD`, if retained, is a Garden runtime-policy input only. It is not a Laputa authority file, a Frozen Core document, or an additional personality type. It governs evidence, privacy, scope, and write-path behavior; it does not compete with `REDLINE.MD`.

## 7. Mandatory Clean-Break Deletions

The new runtime must delete rather than translate these concepts:

- `.laputa/sections/*.json` personality files and all `map[string]any` authority bodies.
- `01-identity`, `02-relationship`, `03-commitment`, `04-preferences`, and `05-memory_md` as target-model names.
- `Commitment`, `Preferences`, `MemoryMD`, `MEMORY.MD`, `memory_md`, `history_md`, and `LONGMEM.MD` authority contracts.
- JSON Patch, dot-path patch, JSON editor, JSON preview, JSON schema validation, and JSON fallback for personality content.
- Automatic WORLD projection, scoped WORLD projection, and any automatic ACTMEM projection in ContextView.
- Generic governance/proposal/approval workflows for Persona or ACTMEM.
- Any route, DTO, test fixture, seed file, console label, or migration path that makes a retired JSON contract appear usable.

Historical JSON data may remain physically on a user machine as inert evidence. New code must not discover it, read it, import it, or use it as fallback data.

## 8. Keep, Cut, Deferred, Drop

| Classification | Items |
| --- | --- |
| Keep | Mentle evidence and retrieval boundaries; Garden activity, raw-first ingest, cards/evidence, recall trace, degraded operation; EvoMap mailbox, privacy gate, Hub transport, explicit publication gating; human report artifacts and modules |
| Cut | JSON authority registry, JSON section file store for Persona, `memory_md`, `Commitment`, `Preferences`, automatic WORLD/STM context projection, generic Persona governance mutation API |
| Deferred | Persona storage path implementation, revision-store schema, ACTMEM compaction algorithm, ACTMEM-to-Mentle promotion policy, one-read-tool wiring, EvoMap candidate schema, SOP-versus-Skill package shape, host adapter changes |
| Drop | Any compatibility migration, dual read/write, legacy descriptor mapping, legacy JSON fallback, retired proposal-inbox semantics, user-defined authority types |

## 9. Implementation Gate

This decision authorizes documentation and design work only. Production implementation begins only after a deletion-first execution plan defines the replacement packages, HTTP contract, console migration, test matrix, and proof that no legacy JSON personality path remains.

The first vertical slice must prove all of the following:

1. Five-file initialization and seven-file Markdown authority store work with atomic write, bounded projections, textual history, and session-frozen Frozen Core.
2. `WORLD.MD` is accessible only by an explicit tool read and is absent from automatic ContextView assembly.
3. `ACTMEM.MD` survives a new session and restart, is absent from automatic ContextView assembly, and is reachable through one explicit read/query capability.
4. An EvoMap candidate can reference bounded evidence and ACTMEM reflection without receiving authority to write Persona or directly create/install a Skill.
5. Repository scans show no target runtime use of retired JSON Persona names, JSON Patch, `memory_md`, or automatic WORLD/ACTMEM projection.
