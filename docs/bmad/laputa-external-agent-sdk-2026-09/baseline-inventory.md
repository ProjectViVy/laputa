# E00-S01 — External Agent SDK Dirty-Tree and Ownership Baseline

**Captured:** 2026-09-04  
**Mode:** explicit scoped ownership; no move/reset/stash/clean/stage/commit  
**Scope owner:** BMAD Coordinator  
**Source edits by this Story:** none

## Owner decision

> Do not move, revert, or auto-commit; keep the current dirty tree explicitly scoped.

Every later Story records its own start snapshot and claims only declared paths/hunks. No worker owns a pre-existing dirty file merely because it needs an adjacent change.

## Garden baseline

```text
Repository: C:/Users/Administrator/Desktop/garden
Branch: main
HEAD: 5158f9b9c38d943682d0d844e4a2f8e1c4a139f2
Upstream delta: ahead 12 / behind 0
Tracked unstaged paths: 4
Tracked staged paths: 0
Untracked paths: 112701
```

| Class | Paths/count | Ownership decision |
|---|---|---|
| Current SDK planning documents | `docs/README.md`, `docs/architecture/AGENTS.md`, `docs/architecture/0015-*`, `docs/bmad/laputa-external-agent-sdk-2026-09/**` | Coordinator-owned documentation scope |
| Existing archived TODO modification | `docs/archive/2026-08-14-laputa-clean-break/status/TODOLIST-pre-clean-break.MD` | pre-existing / do not touch |
| Existing generated Console output | `garden/console/tsconfig.tsbuildinfo` | pre-existing generated / do not touch |
| Tool state | `.agents/**` (1), `.omc/**` (25) | pre-existing / excluded |
| Research snapshots | `.workspace/**` (2 repository roots) | excluded, read-only evidence only |
| Existing unrelated doc | `docs/dev/reference/2026-08-04-mempalace-python-gap-audit.md` | pre-existing / do not touch |
| Test-cache artifacts | `e2e-tmp/**` (92577), `garden/e2e-tmp/**` (20081) | pre-existing generated cache / excluded from all diffs and claims |

The prior untracked counts include the current planning package's 15 documents. They are documentation deliverables; no runtime behavior follows from their presence.

## AGENT-VIVY baseline

```text
Repository: C:/Users/Administrator/Desktop/morediva/diva-go/agent-vivy
Branch: main
HEAD: 17722669c7fbabf1e28631b3acd9e7436955c450
Upstream delta: ahead 140 / behind 0
Tracked unstaged paths: 1
Tracked staged paths: 0
Untracked paths: 0
```

| Path | Classification | Ownership decision |
|---|---|---|
| `studio` (submodule pointer/worktree state) | pre-existing, outside Laputa Provider scope | do not touch |

## Future Story ownership rules

1. Garden canonical REST stories may modify only declared Garden route/domain/contract files after E00-S02 freezes them.
2. `garden/internal/server/server.go`, MCP registration, canonical error fixtures and shared conformance fixtures are serial integration locks.
3. The future public SDK module location remains undecided until E00-S02. No child creates it before that decision.
4. Vivy work occurs in an isolated declared worktree after Garden canonical semantics are available; `studio/` stays excluded.
5. No Story uses the cache trees above as a test fixture, source, or ownership signal.

## Verification scope

Captured with `git rev-parse HEAD`, `git branch --show-current`, upstream delta, porcelain v2, staged/unstaged name+stat, and full untracked inventory. A separate count-only pass classified all 112701 untracked Garden paths by top-level directory to avoid treating test cache as work.

No destructive Git operation, staging, commit, source edit, test edit, configuration edit, or runtime operation occurred in E00-S01.