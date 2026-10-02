# DIVA cognitive integration — acceptance evidence (S09)

Date: 2026-10-02. Source pins: laputa monorepo lane `feat/diva-cognitive` (see git log; merged to local main as `f639e36`), agent-vivy `401f38eb`, INOFY `71e2c9b`, agent-diva `d96e396d` (reference only, unchanged).

Verification commands and results — all executed on the pinned revisions:

| Command | Result |
| --- | --- |
| `laputa/`: `CGO_ENABLED=0 GOSUMDB=off go test ./...` | PASS |
| `mentle/`: `CGO_ENABLED=0 GOSUMDB=off go test ./...` | PASS |
| `garden/`: `CGO_ENABLED=0 GOSUMDB=off go test ./...` | PASS |
| `garden/`: `GOSUMDB=off go test -tags=e2e ./e2e/...` | PASS (clean-break e2e + `TestDivaCognitiveDecisiveEndToEnd`) |
| `agent-vivy`: `just ci` | PASS (fmt, vet, full test incl. `sdk/internal/conformance`, UI gates, plugin-ci) |
| `agent-diva`: `just ci` | PASS (GUI vitest + build, baseline `d96e396d`) |

## Requirement → evidence mapping

- **R1** (one personality authority): `garden/evolution.Domain` binds Persona/ACTMEM as sole authorities; no second personality store added anywhere. e2e: `TestDivaCognitiveDecisiveEndToEnd`, `garden/evolution/domain_test.go`.
- **R2** (workspace isolation): ingest `workspace` column + `Window(workspace,...)`; `agentapi.Capture` forwards `Binding.WorkspaceID`; `Scope` validated at Domain construction. e2e scope-visible collection: `TestCollectProjectsCommittedWindow`.
- **R3** (automatic consolidation under enabled policy): `cognitive_service.go` wake loop + `Evaluate` gate; policy-approved commits without per-item confirmation; `TestCognitiveWakeCoalescesToOneActiveRun`, `TestCognitiveNotifyInputFeedsWindow`.
- **R4** (DIVA strategy, INOFY reuse): `laputa/evolution/diva` + `StartCognitiveWorkflow` on the INOFY workflow path (`TrustedStrategyDIVA`); no second engine.
- **R5** (Mission): human-only `KindMission` writes (user direct), AutoDream/agent/forged-actor refused, stale pin → `mission_revision_changed` at admission. e2e assertions in `TestDivaCognitiveDecisiveEndToEnd`; unit `TestCognitiveStaleMissionBlocks`.
- **R6** (DREAM conversational only): `SaveAgentP16(KindDream)` succeeds; Mission via same path refused; effect union contains no mission/dream kinds (closed `PersonaRequestKind`).
- **R7** (Garden native governance): bound Domain in `garden/evolution`; backend replaceable via `memory.Backend` port (fake backend exercised beside real services).
- **R8** (optional backends in-repo): Mentle adapter `garden/backends/mentle` + `memory.Backend` contract; test backend used for decisive scenario.
- **R9** (authority separation): work_patch→ACTMEM, memory_mutation→memory backend, persona_request→Persona review, capability_proposal→EvoMap proposer; separate durable receipts per effect.
- **R10** (Markdown ACTMEM): `ApplyWorkPatch` into `actmem.Store`; scoped read verified.
- **R11** (strategy semantics in Laputa only): all eligibility/strategy code in `laputa/evolution`; ViVy supplies runtime ports (`Domain`, `CognitiveSource`, `CognitiveMissionSource`, `CognitiveCaptureSink`) only.

## Gate → evidence mapping

- **G1**: mission write/deny matrix in decisive e2e + `TestCognitiveStaleMissionBlocks` + S02 unit suite (`persona` package).
- **G2**: persona eight-kind roster in e2e (`Documents != 8` — caught and fixed a stale 7-count assertion left by S02); FrozenCore v2 wire in `agentapi` tests.
- **G3**: scope-encoded ingest workspace + window reads; dedupe replay returns original seq.
- **G3a**: actmem round-trip/scope in `laputa/actmem` suite + domain work_patch apply.
- **G4**: real Mentle adapter AND minimal fake backend both bound through `memory.Backend`; failure → `partial` (`TestUnavailableBackendPartialOutcome`).
- **G5**: `memory_mutation` commits without per-item confirmation; backend op-id idempotency (restart replay → zero extra mutations); reflection notes durable but never evidence (`TestReflectionNoteIsDurableButNotEvidence`, `TestCollectSkipsReflectionNotes`).
- **G6**: conversational `SaveAgentP16(KindDream)` accepted; autodream/agent `Write(KindMission)` and `SaveAgentP16(KindMission)` refused; closed request vocabulary cannot encode mission/dream effects.
- **G7**: one admission path (`cognitiveAttempt`) for loop + `TriggerCognitive`; `TestCognitiveNoNewInputCallsNoModel` (zero model calls without new input), `TestCognitiveDisabledPolicyBlocks`, `TestCognitiveFailedRunRetriesWithNewKey`, `TestCognitiveWakeCoalescesToOneActiveRun`, `TestCognitiveCaptureCursorFollowsAcceptance`.
- **G8 real desktop path**: **PENDING** — blocked on external DIVA-NEXT-P0 desktop bridge (S08). ViVy-side loop/capture/status are tested; the real desktop joint run was not performed.
- **G9**: all CGO=0 module suites PASS; garden e2e PASS incl. console-embedded binary; `just ci` PASS; INOFY suite PASS earlier (baseline green at pin); independent consumer build not separately exercised (agent-vivy builds against the monorepo via its replace pin — same tree).

## Limitations / not performed

- No configured live-model run — all model behavior is scripted fixtures; no semantic-quality claims.
- G8 real desktop joint acceptance: pending external bridge; not claimed.
- agent-diva desktop smoke: not exercised (GUI unit + build only; reference repo unchanged).
- SQLite/PostgreSQL observerhost conformance: existing storage-conformance suites pass under `just ci`; no new postgres-specific capture fixture added.
- Supervisor-run replacement after process restart is covered by design + unit recovery tests, not by a full crash-loop e2e.

## Changed contracts during execution

`contracts.md` changelog records: `CaptureReceipt.Seq`/ingest seq + window semantics (S07), Mission/frozen-core v2 (S02), scoped ACTMEM grammar (S03), backend receipt/status vocabulary (S04), strategy DTOs (S05), trusted-strategy admission (S06). No unrecorded interface change shipped.

## Known stale-assertion fix recorded

`garden/e2e/e2e_test.go` expected 7 persona documents; S02's eight-file roster made it 8. Assertion updated to the authorized roster (MISSION.MD per ADR-0017), not removed.

Rollback note: new runs can be stopped by disabling the trigger policy (`UpdateCognitivePolicy`) or `StopCognitiveLoop`; code rollback is schema-compatible (new `seq` column reads use `rowid`; new effect ledgers are additive files).
