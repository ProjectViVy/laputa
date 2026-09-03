# Epic 1 — Persona Authority Parity and Write Safety

**Outcome:** the existing Go Persona slice becomes DIVA-compatible, atomic and safe to expose through domain adapters.

**Requirements:** FR-PER-01..08, FR-API-02, NFR-01..03
**Architecture:** AR-002..004, AR-011
**Dependencies:** Epic 0 API/error contract and test inventories
**Exit checkpoint:** Persona service parity and policy review before Garden runtime cutover.

## Story 1.1 — Lock DIVA text, CAS and no-op parity

**Acceptance criteria**

- Normalize and count text exactly as the DIVA contract requires.
- Existing revision plus `base_revision=0` returns typed conflict.
- CAS is enforced at the commit boundary.
- Identical normalized content returns `changed=false` and does not create revision/history.
- Tests cover conflict/no-op/over-limit without file mutation.

## Story 1.2 — Make history immutable and failure-atomic

**Acceptance criteria**

- Snapshot/diff creation uses create-new semantics; collisions fail rather than overwrite.
- History metadata preserves actor, source, reason and base revision.
- Failed history or authority install leaves prior document/history coherent.
- Direct writes stale same-kind pending reviews with `decided_at`.
- Acceptance retains proposal actor/source/reason; sibling stale persistence errors are surfaced.

## Story 1.3 — Implement atomic five-file initialization and repair

**Acceptance criteria**

- Validate all required documents before any live mutation.
- Stage five files plus initial history on the same volume and install as one guarded operation.
- DREAM/DARK are not created.
- Partial live state produces a typed integrity error.
- Explicit repair validates/reconstructs only owner-selected inconsistent Persona artifacts; no legacy JSON import.
- Crash/failure tests prove no half-initialized profile is accepted as ready.

## Story 1.4 — Enforce Persona write classes

**Acceptance criteria**

- User, agent and operator principals are verified independently of `X-Garden-Actor`.
- Agent protected-document changes create reviews and cannot direct-write.
- Direct agent writes are limited to DREAM, DARK and USER Observations.
- USER Preferences/Observations scope rules and WORLD preservation/R6 rules match DIVA.
- Agent cannot approve/reject its own or any Persona review through the agent capability.

## Story 1.5 — Complete review lifecycle semantics

**Acceptance criteria**

- Review creation requires ready profile, legal actor/kind, valid bounded normalized proposal and one pending per kind.
- Stale base is persisted as `stale` with decision timestamp.
- List/filter/read/approve/reject operations have stable typed errors.
- Review files and transitions survive restart.
- Full DIVA parity matrix passes.

## Story 1.6 — Complete Persona domain adapter tests

**Acceptance criteria**

- Initialize, document list/read/write, review list/create/approve/reject, history and repair APIs use the frozen contract.
- HTTP tests cover principal class, cap, CAS, no-op, stale, malformed payload and stable error codes.
- Raw file writer is not exposed to handlers.
- Existing `/files` and `/requests` names are not kept as aliases at final cutover.
