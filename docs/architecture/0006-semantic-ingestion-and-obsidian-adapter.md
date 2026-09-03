# ADR-0006: Semantic Ingestion and Obsidian Source Adapter

**Status:** accepted  
**Date:** 2026-08-03  
**Supersedes:** none  
**Depends on:** ADR-0012 §6 (Mentle owns materials/evidence, not authority), which supersedes the former Laputa section references; raw-first ingestion remains this ADR's subject.

---

## 1. Context

Historical note: the retired AAAK pipeline demonstrated that external note content is most useful when decomposed into small, provenance-preserving semantic units for indexing, not compressed into a Laputa summary file.

Current state at decision time:

- Mentle `facade.CreateMemory` accepts kinds `fact/preference/decision/session_digest/note` only. Garden's session ingest worker writes `kind="source_artifact"`, which is rejected outright — the root cause of the long-standing e2e ingestion hang.
- `Memory.Source` carries `{Type, SessionID, EventID}` only; provenance beyond that rides untyped `Metadata` (a `source_uri` precedent exists).
- `MemoryCard.SourceRef` is populated from `Source.Type`, not a real reference.
- `ReadEvidence` synthesizes offsets (`0..len(excerpt)`) and hashes the full memory content, not the excerpt.
- No source adapter abstraction exists anywhere in the repository.

Gate C requires: an Obsidian source adapter design, provenance-preserving semantic units (source path, heading path, block offsets, hash, scope/tags, bounded evidence read), and verification that semantic-unit ingestion creates no second summary authority.

## 2. Decision Summary

1. Unify the kind model: `source_artifact` (raw original) and `semantic_unit` (derived index unit) join the facade allow-list.
2. Specify (do not yet implement) the Obsidian source adapter behind a generic `SourceAdapter` contract.
3. Specify semantic-unit decomposition: front-matter → heading path → blocks with byte offsets, per-unit content hash, scope/tags inheritance.
4. Carry provenance in a structured, enumerated metadata key set; extend `Memory.Source` minimally.
5. Semantic units are index material only: no Laputa section writes, no automatic WORLD claims, authority, skills, or EvoMap uploads.
6. `ReadEvidence` supports real stored offsets when present; budgets unchanged.

## 3. Kind Model

| Kind | Meaning | Written by |
|---|---|---|
| `source_artifact` | Raw original content, persisted before acknowledgement | Garden ingest worker, source adapters |
| `semantic_unit` | Derived index unit referencing a `source_artifact` | Semantic decomposition step |
| `fact/preference/decision/session_digest/note` | unchanged | unchanged |

Rules:

- A `semantic_unit` must carry `source_ref` metadata pointing at its parent `source_artifact` memory id (or `source_path` + `content_hash` when the artifact is file-backed).
- Decomposition is derivable and re-runnable; losing semantic units never loses the raw artifact.
- Unknown kinds remain rejected.

## 4. Source Adapter Contract (specification only)

First adapters are read-only and bounded:

```go
type SourceAdapter interface {
    Name() string
    Scan(ctx context.Context) ([]SourceItem, error)
    Read(ctx context.Context, ref string, budgetChars int) (EvidenceFragment, Revision, error)
}

type SourceItem struct {
    Ref           string    // adapter-local stable reference (e.g. vault-relative path)
    SourcePath    string    // human-readable origin path
    SourceRev     string    // mtime/hash revision token
    ContentHash   string    // sha256 of full content
    ObservedAt    time.Time
    Scope         string
    Tags          []string
}
```

Obsidian specifics (specified, implementation deferred to a later batch):

- Vault location, enablement, and sync trigger (endpoint vs scheduled) are deliberately undecided here.
- The adapter reads only; it never writes back into the vault (`read_only=true`).
- Scan results feed the raw-first path: persist `source_artifact` first, then decompose.

## 5. Semantic Unit Decomposition (specification)

For one markdown note:

1. **Front-matter** (`---` YAML block): extract `tags`, `scope`, aliases; never indexed as a unit itself.
2. **Heading path**: ordered array of ancestor headings, e.g. `["Architecture", "Recall", "Budgets"]`. H1–H6 levels define nesting.
3. **Blocks**: maximal paragraph/list/code segments under a heading. Each block is one semantic unit.

Each unit carries: `source_path`, `heading_path`, `start_offset`/`end_offset` (byte offsets into the artifact content), `content_hash` (unit content), `source_revision`, `observed_at`, `sync_state` (`current|stale`), inherited `scope`/`tags`.

Bounds: units longer than the 64 KiB memory cap are split at block boundaries; empty blocks are skipped.

## 6. Provenance Contract

Structured metadata keys (enumerated, stable):

| Key | Type | Meaning |
|---|---|---|
| `source_path` | string | origin file/vault-relative path |
| `source_ref` | string | parent artifact memory id (semantic_unit only) |
| `heading_path` | []string | ancestor heading chain |
| `start_offset` / `end_offset` | int | byte offsets into artifact content |
| `content_hash` | string | sha256 of unit content |
| `source_revision` | string | adapter revision token |
| `observed_at` | RFC3339 | when observed |
| `sync_state` | string | `current` \| `stale` |
| `lifecycle`, `collection` | string | existing conventions unchanged |

`Memory.Source` extension (minimal): add `URI string` and `Revision string` fields; `Type` gains value `import` for adapter-originated material (already in the allow-list). Session provenance is unchanged.

## 7. No Second Summary Authority

Semantic-unit ingestion must not recreate AAAK's failure mode:

- No Laputa authority file is written anywhere in the ingestion path. Units never create `WORLD.MD` content, Persona mutations, Skills, or EvoMap uploads automatically.
- The original source remains authoritative; Mentle never writes back.
- Units participate in card search and bounded evidence read only; heat alone establishes nothing.

Acceptance test: an ingestion run asserts zero mutations against every Laputa authority file and `ACTMEM.MD`.

## 8. Bounded Evidence Read

`ReadEvidence` behavior:

- When a memory carries valid `start_offset`/`end_offset` metadata, the excerpt is cut from stored content by those offsets (clamped to content length and per-item budget).
- `ContentHash` reports the excerpt's hash when real offsets are used; full-content hashing remains for legacy memories. (Trade-off: keeps existing assertions stable for legacy rows while making new evidence self-verifying.)
- Per-item and total character budgets are enforced exactly as today.

## 9. Test Matrix

| # | Test | Layer |
|---|---|---|
| 1 | `source_artifact` and `semantic_unit` accepted by CreateMemory; unknown kind still rejected | mentle facade |
| 2 | Provenance metadata round-trips through CreateMemory → ListMemories | mentle facade |
| 3 | Real-offset evidence excerpt + excerpt hash; budget clamps | mentle facade |
| 4 | Legacy memories keep synthetic offsets + full-content hash | mentle facade |
| 5 | `IngestSemanticUnits` idempotent on `source_path+content_hash` | garden |
| 6 | Ingestion writes zero Laputa sections | garden |
| 7 | e2e ingestion completes (hang eliminated) | garden e2e |

## 10. Consequences

- The e2e ingestion hang is fixed by the kind allow-list change.
- Future adapters (Obsidian first) implement `SourceAdapter` without further contract changes.
- ReadEvidence gains a behavioral branch; legacy semantics preserved.
- No console surface in this batch; material pages already display evidence generically.
