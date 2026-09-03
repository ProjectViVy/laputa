# Garden Authority & Recovery — GOAL Execution Runbook

**Status:** accepted execution contract; GOAL active and Waves 0–3 complete
**Date:** 2026-09-03
**Scope:** Waves 0–3 in this planning package
**Implementation authorization:** true; the owner activated this GOAL for the current execution
**Commit policy:** one audited checkpoint commit per Wave; never push

This runbook controls how the accepted PRD, Architecture Spine, API contract, Epics, test inventories and manual TODOLIST are executed as one durable GOAL. It adds orchestration rules; it does not replace Story requirements.

## 1. GOAL contract

When the owner explicitly activates the GOAL, use this objective without a token budget:

> Implement and verify Garden Authority & Recovery Convergence across Waves 0–3: establish DIVA-compatible Persona and ACTMEM authority, restart-safe Frozen Core, remove retired JSON Governance and automatic WORLD/ACTMEM projection, make canonical SQLite the sole recoverable Mentle authority, cut REST/MCP/Console/EvoMap to the accepted domain contracts, pass every recorded gate, and create audited per-wave commits without pushing.

The GOAL is complete only when:

- every Story in Waves 0–3 is `done` and every Wave exit gate has current evidence;
- the blocking architecture/deletion scanner reports zero target-runtime violations;
- Garden, Laputa and Mentle suites, clean-break/recovery e2e, and Console checks pass;
- ADRs, API types, runtime wiring, Console behavior and active Agent instructions agree;
- four Wave checkpoint commits exist and no GOAL-owned change remains uncommitted;
- unrelated user changes that remain dirty are listed and explicitly excluded from GOAL ownership.

The GOAL must not be marked complete because the implementation is mostly present, a stub compiles, an old test passes, or the remaining budget is small.

## 2. Source-of-truth precedence

Use the first applicable source in this order:

1. The nearest active `AGENTS.md`, provided it agrees with accepted ADRs.
2. ADR-0012, ADR-0013 and ADR-0014.
3. `ARCHITECTURE-SPINE.md` and `api-contract.md`.
4. The owning Epic and `test-inventories.md`.
5. This runbook, `TODOLIST.md` and `sprint-status.yaml`.
6. The Story implementation record and current source code.

Additional interpretation rules:

- DIVA `agent-diva-laputa` is the behavioral authority for Persona and ACTMEM path layout, normalization, caps, CAS, no-op, history, review, write-class and capsule semantics.
- `api-contract.md` is authoritative for REST DTOs, stable error codes, principal classes and the atomic removal list.
- Existing runtime code is a baseline to replace, not a compatibility contract.
- `docs/archive/**`, legacy fixtures and retired tests are historical evidence only.
- A discovered conflict is escalated to the Coordinator before implementation continues; an Agent may not silently select the legacy behavior.

## 3. Roles and concurrency

### 3.1 Coordinator

One GOAL Coordinator owns scheduling and integration. Only the Coordinator may:

- change `sprint-status.yaml`, readiness verdicts or project-level status;
- assign or release a shared-file lock;
- declare a Wave gate passed;
- stage and create a Wave checkpoint commit;
- mark the GOAL complete or blocked.

The Coordinator does not claim a worker's implementation until the Story evidence and diff have been reviewed.

### 3.2 Workers

- One Story has one owning worker and one implementation record.
- At most two workers may be `in-progress` at once.
- Concurrent Stories must have disjoint declared path sets and no shared lock.
- If only one agent is available, preserve the same ordering and execute the lanes serially.
- Workers never edit project status directly and never commit or push.

### 3.3 Shared locks

| Lock | Paths/authority | Exclusive owners |
|---|---|---|
| `STATUS` | sprint status, readiness, root/BMAD status summaries | Coordinator |
| `COMPOSITION` | `garden/main.go`, `garden/internal/server/server.go` | designated Integrator |
| `PERSONA_CORE` | Persona service/history/review and shared tests | one Persona Story at a time |
| `MENTLE_CANONICAL` | canonical schema, Facade mutation boundary, index-job migration | one Mentle Story at a time |
| `CONSOLE_SHELL` | App, Sidebar, shared API types, i18n, global styles | E04-S06 Integrator |
| `MODULE_DEPS` | each module's `go.mod`/`go.sum` | current module owner or Integrator |
| `WAVE_COMMIT` | staging area and commit | Coordinator |

E02-S06 and E02-S07 are always serial and use the same Integrator unless an explicit implementation record transfers all locks and current diff. Feature Console work in E04-S05 stays under feature modules; shared shell changes wait for E04-S06.

## 4. Story state machine

The only Story states are:

```text
backlog -> ready -> claimed -> in-progress -> verifying -> done
                         \-> blocked <-/
```

- `ready`: all dependencies and the Wave entry gate pass.
- `claimed`: Coordinator records owner, allowed paths, locks and baseline fingerprint.
- `in-progress`: the implementation record exists and the first failing/contract test is recorded.
- `verifying`: implementation is frozen except for defects found by the declared verification ladder.
- `done`: acceptance criteria, owned diff review and required tests all pass.
- `blocked`: progress requires an owner decision, unavailable external dependency, unsafe dirty-tree overlap, or the same failure remains after three evidence-backed attempts using materially different hypotheses.

An unaffected lane may continue while another Story is blocked. A blocked shared integration Story blocks every dependent Story.

## 5. Story start protocol

Before modifying a file, the worker must:

1. Read root and nearest `AGENTS.md`, accepted ADRs, the Architecture Spine, API contract, test inventory, baseline inventory and owning Epic.
2. For Persona/ACTMEM, inspect the current DIVA reference implementation and tests; do not rely on recollection.
3. Record in `implementation/eNN-sNN.md`:
   - Story ID, owner and start time;
   - branch, HEAD and upstream divergence;
   - `git status --porcelain=v1 -uall` summary;
   - target-file pre-existing diff and allowed path list;
   - dependencies, required test IDs and acquired locks.
4. Run the narrowest current test that demonstrates the missing/divergent behavior.
5. Stop before writing if an allowed path contains an unexplained concurrent change that cannot be isolated by hunk.

No worker may use `reset`, `checkout`, `clean`, `stash`, `rebase`, broad recursive deletion, `git add .` or `git add -A` to simplify the baseline.

## 6. Implementation and evidence protocol

- Implement the smallest complete Story contract; do not absorb adjacent backlog work.
- Use domain services for writes. Adapters do not receive raw Persona stores, mutable Searcher/vector handles or raw canonical DB handles.
- Do not add migration, JSON compatibility, aliases, fallback reads, dual writes or automatic WORLD/ACTMEM projection.
- Persistence changes require failure-atomic, restart and concurrency evidence appropriate to the Story.
- Tests must report real commands, exit codes and relevant output. Distinguish baseline failures, new regressions and unrelated worktree noise.
- A test asserting retired JSON Governance or automatic WORLD projection is rewritten/deleted in its owning cutover Story and cannot count as positive evidence.
- A health/UI path cannot report success from a stub, accepted-design fixture, missing endpoint or startup label.

Every implementation record ends with:

```text
Story: E##-S##
Status: done | blocked
Allowed paths:
Actual changed files/hunks:
Tests and exact results:
Acceptance criteria results:
Pre-existing issues preserved:
Unfinished/blocking condition:
Locks released:
Commit: none (included later in Wave N checkpoint)
```

## 7. Lane schedule

### Wave 0 — Baseline and executable gates

Run at most two workers:

```text
QA:    E00-S02 ─┐
Guard: E00-S04 ─┼─> Gate 0-B readiness rerun ─> Wave 0 commit
ADR-0014 accepted┘
```

Gate 0 passes only when the monthly report test is deterministic, all three Go modules and Console checks are green, the scanner reports current violations without false positives, ADR-0014 is accepted, and readiness is `PASS`.

### Wave 1 — Persona and Mentle foundations

Run two module-isolated lanes:

```text
Persona: E01-S01 -> S02 -> S03 -> S04 -> S05 -> S06
Mentle:  E03-S01 -> S02 -> S03 -> S04 -> S05 -> S06 -> S07 -> S08 -> S09
```

Persona Stories are serial because they overlap the service/history/review boundary. Mentle Stories are serial by default because canonical, outbox, embedding identity and rebuild semantics converge in shared Facade/schema code. The Coordinator may split a Mentle Story only after recording disjoint path and lock sets. Wave 1 commits only after both Epic exit reviews pass.

### Wave 2 — ACTMEM and runtime clean break

Two lanes may run until integration:

```text
ACTMEM:         E02-S01 -> S02 -> S03 ─┐
Garden runtime: E02-S04 -> S05 ────────┼-> S06 -> S07 -> S08
```

S06–S08 are serial integration work. S07 turns the scanner from report to enforce mode. The Wave cannot commit while any target runtime import, route, DTO, fixture or automatic context path exposes retired Governance/WORLD behavior.

### Wave 3 — Domain adapters and release

```text
E04-S01 -> S02
             ├-> MCP S03 -> S04 ───────┐
             ├-> Console S05 -> S06 ───┼-> S08
             └-> EvoMap S07 ───────────┘
```

After S02, run at most two branches concurrently. S08 is a single-owner release integration Story.

## 8. Dirty-tree adoption and drift control

At GOAL activation, the Coordinator refreshes `baseline-inventory.md` and classifies every current change:

- `accepted`: directly supports accepted architecture or an existing intended Persona/Mentle/Console slice and passes its available checks;
- `preserved`: unrelated or ownership cannot be established;
- `generated`: `.omc/**`, `e2e-tmp/**`, runtime databases, Console `dist/**` and other build output;
- `blocked`: conflicts with the accepted contract or overlaps a Story in a way that cannot be safely separated.

Wave 0 adopts accepted content using explicit pathspecs or hunk staging. Generated content is never committed. Preserved content remains untouched and listed. A blocked overlap pauses its dependent lane rather than being reset or overwritten.

Every Story rechecks status and target diffs at start, before verification and at handoff. New unexplained drift under a claimed lock is recorded immediately and the affected Story stops.

## 9. Verification ladder

For each Story:

1. focused unit/policy/contract tests mapped by Test Inventory ID;
2. persistence fault, concurrency and restart tests where applicable;
3. affected package and module suites;
4. boundary/deletion assertions;
5. `git diff --check` on owned changes.

Wave gates add:

| Wave | Required evidence |
|---|---|
| 0 | monthly report regression; all Go modules; Console typecheck/build; scanner report; readiness PASS |
| 1 | Persona parity matrix; Mentle canonical/outbox/recovery fault matrix; affected Garden adapters |
| 2 | ACTMEM restart; Frozen Core immutability; structural WORLD/ACTMEM absence; ignored legacy sections; scanner enforce; clean-break e2e |
| 3 | REST/MCP/Console/EvoMap contract tests; all Go/e2e/Console checks; scanner enforce; traceability and operator runbook |

### Windows command rules

Resolve Go once. If `Get-Command go` fails, use `C:\Program Files\Go\bin\go.exe`; if neither exists, mark the environment preflight blocked. Set `GOSUMDB=off` for repository Go tests.

To avoid modifying the tracked `tsconfig.tsbuildinfo` during routine checks, use:

```powershell
npm exec tsc -- --noEmit -p tsconfig.json
npm exec vite -- build
```

The generated `dist/` directory is verification output, never Story ownership.

## 10. Wave commits

Only the Coordinator may stage. Use explicit pathspecs, inspect `git diff --cached --name-status` and `git diff --cached`, then run `git diff --cached --check` before committing.

| Wave | Commit message |
|---|---|
| 0 | `chore: establish authority recovery execution gates` |
| 1 | `feat: complete persona authority and mentle recovery` |
| 2 | `feat: cut over markdown authority and frozen context` |
| 3 | `feat: complete domain adapters console and release gates` |

Do not push. Do not amend a completed Wave checkpoint automatically. A failed hook or verification returns the Wave to integration; only owned defects may be changed.

## 11. Pause, recovery and final handoff

On interruption, the Coordinator records current Story states, owners, locks, last passing commands, next exact action and uncommitted owned paths. Resumption starts by checking that this fingerprint still matches the working tree.

Mark the overall GOAL `blocked` only after the same blocking condition has recurred for at least three GOAL turns and no safe lane can make meaningful progress. Mark it `complete` only after the completion conditions in section 1 are all met.
