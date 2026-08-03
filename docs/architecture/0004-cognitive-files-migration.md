# ADR-0004: Cognitive Files & Compatibility Migration

**Status:** accepted  
**Date:** 2026-08-03  
**Supersedes:** none  
**Depends on:** ADR-0002 (Laputa Cognitive Partition)

---

## 1. Context

ADR-0002 established the target cognitive partition: Frozen Core (01–04), STM (05), cognitive governance files (MEMRULES.MD, WORLD.MD), human reports (07–09), and removed concepts (06, 13, 14). However, no physical representation exists for MEMRULES.MD or WORLD.MD, and the legacy 14-section registry remains unchanged in code.

This ADR specifies:
1. Physical storage and schema for MEMRULES.MD and WORLD.MD
2. Enforcement boundaries (who reads, who writes, what is projected)
3. Legacy section compatibility policy (no deletion, write-blocking)
4. Audit log retention
5. Context Plane test matrix

---

## 2. MEMRULES.MD — Cognitive Governance Rulebook

### 2.1 Storage

| Property | Value |
|----------|-------|
| Path | `~/.laputa/cognitive/MEMRULES.MD` |
| Format | Markdown with optional YAML front-matter |
| Encoding | UTF-8 |
| Owner | Human (file editor) |

MEMRULES.MD is NOT a governance JSON section. It lives outside the 14-section FileStore at a dedicated `cognitive/` path.

### 2.2 Schema

```markdown
---
version: 1
updated: 2026-08-03T00:00:00Z
---

# Memory Rules

## R1 — Evidence primacy
Mentle raw material and evidence remain the primary source of truth.

## R2 — Claim distinction
Confirmed fact, observation, inference, and hypothesis are distinct categories.

## R3 — Contradiction handling
New contradictory evidence does not silently overwrite prior understanding.

## R4 — User authority
User-confirmed information outranks agent inference.

## R5 — Scope constraint
Scope, time, confidence, provenance, and visibility constrain use.

## R6 — WORLD entry gate
Entry into WORLD requires action relevance and a bounded, reviewable claim.

## R7 — No wholesale injection
WORLD is not copied wholesale into an Agent context.
```

Rules are identified by `## R{n} — {title}` headings. Body text is the rule definition. The front-matter is optional metadata for tooling.

### 2.3 Edit Path

- **Human-only.** No HTTP write endpoint. No agent write interface.
- Edited via file editor (VS Code, vim, etc.)
- Garden detects changes by reading at startup. Hot-reload via mtime polling is optional (future).

### 2.4 Enforcement

- Garden loads rules at boot into a `RulesProvider` interface.
- Rules constrain behavior on ingest, recall, and cognitive-write paths.
- Rules are NEVER injected into ContextView or returned as recall content.
- If MEMRULES.MD is absent, Garden operates with built-in defaults (the 7 rules above hardcoded as fallback).

### 2.5 Go Type

```go
package cognitive

type MemRules struct {
    Path    string
    Version string
    Raw     string
    Rules   []Rule
}

type Rule struct {
    ID    string  // "R1", "R2", etc.
    Title string
    Text  string
}

func LoadMemRules(path string) (*MemRules, error)
func (m *MemRules) Reload() error
```

---

## 3. WORLD.MD — Actionable World Understanding

### 3.1 Storage

| Property | Value |
|----------|-------|
| Path | `~/.laputa/cognitive/WORLD.MD` |
| Format | Structured Markdown (claim-per-section) |
| Encoding | UTF-8 |
| Owners | User (direct edit), AutoDream (future, governed) |

### 3.2 Claim Format

Each claim is a level-2 heading with structured metadata:

```markdown
## [environment] Development machine
- status: confirmed
- confidence: high
- scope: dev, infra
- source: user
- updated: 2026-08-03T00:00:00Z

Windows 11, 64GB RAM, Go 1.26, Node 24. Primary IDE is VS Code.
```

**Field definitions:**

| Field | Values | Required |
|-------|--------|----------|
| status | `confirmed` / `observed` / `inferred` / `hypothesis` / `stale` | yes |
| confidence | `high` / `medium` / `low` | yes |
| scope | comma-separated scope tags | yes |
| source | Mentle card ID (`mem_xxx`) or `user` or `autodream` | yes |
| updated | RFC3339 UTC timestamp | yes |

**Domain** is in the heading brackets: `[environment]`, `[project]`, `[people]`, `[tools]`, etc.

**Body** is bounded to 280 characters per claim.

### 3.3 Write Paths

| Actor | Permission | Constraint |
|-------|-----------|------------|
| User | Direct file edit | Unrestricted |
| AutoDream | Governed API write | Cannot overwrite `confirmed` claims; must set source=autodream |
| Normal agent | None | No write interface exposed |

**Protection rule:** A claim with `status: confirmed` and `source: user` cannot be replaced or deleted by AutoDream. It can only be marked `stale` with an appended note, requiring user review.

### 3.4 Projection API

```http
GET /v2/cognitive/world?scope=dev,infra&budget=2000
```

Response:
```json
{
  "claims": [...],
  "total": 42,
  "projected": 8,
  "budget_chars": 2000,
  "source": "live"
}
```

- Returns only claims matching at least one requested scope.
- Truncates to fit `budget` (default 4000, max 16000 characters).
- NEVER returns the full file.
- If no scope is provided, returns claims with highest confidence first.

### 3.5 Context Rule

WORLD is NOT default context. Garden may project only a scope- and task-relevant, budgeted slice during recall. Evidence for claims comes from Mentle on demand (via `source` card reference).

### 3.6 Go Type

```go
package cognitive

type ClaimStatus string

const (
    ClaimConfirmed  ClaimStatus = "confirmed"
    ClaimObserved   ClaimStatus = "observed"
    ClaimInferred   ClaimStatus = "inferred"
    ClaimHypothesis ClaimStatus = "hypothesis"
    ClaimStale      ClaimStatus = "stale"
)

type WorldClaim struct {
    Domain     string      `json:"domain"`
    Title      string      `json:"title"`
    Status     ClaimStatus `json:"status"`
    Confidence string      `json:"confidence"`
    Scopes     []string    `json:"scopes"`
    Source     string      `json:"source"`
    Updated    time.Time   `json:"updated"`
    Text       string      `json:"text"`
}

type WorldStore struct {
    Path   string
    Claims []WorldClaim
}

func LoadWorld(path string) (*WorldStore, error)
func (w *WorldStore) Project(scopes []string, budgetChars int) ([]WorldClaim, error)
func (w *WorldStore) Save(actor string) error
```

> Errata (2026-08-03): the implemented signature is `Project(scopes []string, budgetChars int) []WorldClaim` — no error return; projection never fails, it only filters and truncates. Code is authoritative.

---

## 4. Legacy 14-Section Compatibility Mapping

### 4.1 Policy

No section is deleted or renamed by this ADR. Only `Compat: true` flags and write-blocking are added. Physical migration (rename, delete, restructure) requires a separate execution PR after this ADR is accepted.

### 4.2 Mapping Table

| Section | Target Status | Action |
|---------|--------------|--------|
| 01-identity | Frozen Core | Keep; add `Frozen: true` to metadata |
| 02-relationship | Frozen Core | Keep; add `Frozen: true` |
| 03-commitment | Frozen Core | Keep; add `Frozen: true` |
| 04-preferences | Frozen Core | Keep; add `Frozen: true` |
| 05-memory_md | STM | Keep unchanged |
| 06-history_md | Removed | `Compat: true`; block all writes; data preserved read-only |
| 07-daily | Reports | Keep unchanged |
| 08-weekly | Reports | Keep unchanged |
| 09-monthly | Reports | Keep unchanged |
| 10-journal_reflective | Future AMBITION | Keep; rename deferred to migration execution |
| 11-proposal_inbox | Future USER_SUGGESTIONS | Keep; rename deferred |
| 12-changelog | Audit infra | Keep; FileAuditLog supersedes; section frozen |
| 13-report_indexes | Removed | `Compat: true`; block all writes |
| 14-aaak_summaries | Removed | `Compat: true`; block all writes |

### 4.3 Compat Write-Blocking

When `Compat: true` is set on a section:
- All write/patch/delete operations return `ErrCompatReadOnly` (HTTP 410 Gone).
- Read operations continue to work (data preservation).
- The section is excluded from ContextView assembly.
- The section appears in the admin UI with a "compat" badge.

### 4.4 Frozen Core Semantics

Sections 01–04 with `Frozen: true`:
- Read once at session bootstrap; cached for session lifetime.
- Mid-session writes are possible (governed, audited) but take effect at next session.
- ContextView always uses the session-start snapshot.

---

## 5. Audit Log Retention

### 5.1 Backend

The existing `FileAuditLog` (append-only JSONL at `~/.laputa/sections/audit/changelog.jsonl`) is the target backend. No new storage engine is introduced.

### 5.2 Retention Policy

| Parameter | Default | Configurable |
|-----------|---------|-------------|
| Max age | 90 days | `GARDEN_AUDIT_MAX_AGE_DAYS` |
| Max size | 50 MB | `GARDEN_AUDIT_MAX_SIZE_MB` |

When either limit is exceeded, rotation occurs:
1. Rename current file to `changelog.{YYYY-MM-DD}.jsonl.bak`
2. Start fresh `changelog.jsonl`
3. Oldest `.bak` files beyond 365 days may be garbage-collected (manual)

### 5.3 Audit Scope

| Operation | Audited |
|-----------|---------|
| Frozen Core mutation (01–04) | Yes |
| WORLD governed write | Yes |
| MEMRULES manual edit (detected at reload) | Yes |
| Evolution proposal accept/reject | Yes |
| Ordinary STM write (05 by agent/system) | No |
| Raw Mentle ingest | No |
| Report generation (07–09) | No |
| Compat section read | No |

---

## 6. Context Plane Test Matrix

These tests define the invariant boundaries of what may appear in a ContextView.

| # | Layer | In ContextView? | Test Name | Assertion |
|---|-------|----------------|-----------|-----------|
| 1 | Frozen Core (01–04) | Yes, at bootstrap | `TestFrozenCoreInContext` | Bootstrap context contains identity/relationship/commitment/preferences |
| 2 | STM (05 working set) | Yes, bounded | `TestSTMInContext` | Working set entries appear within budget |
| 3 | WORLD (scoped slice) | Conditional | `TestWorldScopedProjection` | Only scope-matching claims within budget appear |
| 4 | MEMRULES | Never | `TestMemrulesNeverInContext` | Context string never contains rule text |
| 5 | Full WORLD | Never | `TestFullWorldNeverInContext` | Context never contains all claims |
| 6 | Reports (07–09) | Never | `TestReportsNeverInContext` | Report content never in recall context |
| 7 | Raw Mentle material | Via evidence only | `TestEvidenceBounded` | Evidence respects per-item and total budget |
| 8 | Compat sections (06/13/14) | Never | `TestCompatNeverInContext` | Compat section data never in context |

---

## 7. Migration Execution Policy

This ADR is a design gate. Implementation proceeds in two phases:

**Phase 1 (this PR):** Scaffolding only.
- Add `cognitive` package with types and loaders.
- Add `Compat` and `Frozen` fields to SectionMeta.
- Block writes to compat sections.
- Add read-only `/v2/cognitive/world` endpoint.
- No data migration. No section rename. No file moves.

**Phase 2 (future PR, after acceptance):**
- Create default `MEMRULES.MD` and empty `WORLD.MD` at `~/.laputa/cognitive/`.
- Integrate MEMRULES enforcement into recall/ingest paths.
- Integrate WORLD projection into Fast/Deep Recall.
- Implement audit rotation.
- Write Context Plane integration tests (matrix above).

---

## 8. Consequences

**Positive:**
- MEMRULES and WORLD have concrete physical schemas; downstream code can reference them.
- Compat sections are write-protected, preventing accidental mutation of removed concepts.
- Context Plane boundaries are testable invariants.
- No data loss: all legacy data preserved read-only.

**Negative:**
- Two new files outside the FileStore break the "everything is a section" pattern. This is intentional: cognitive governance files are human-facing documents, not API-managed JSON.
- Compat write-blocking may surprise legacy v1 callers expecting to write to 06/13/14. Mitigated by HTTP 410 + clear error message.

**Neutral:**
- Sections 10/11 retain legacy names until Phase 2 migration. No functional impact.