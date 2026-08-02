<!-- Parent: ../AGENTS.md -->

# laputa/.laputa/sections — Governance Section Schemas

**Generated:** 2026-08-01  
**Purpose:** Detailed schema and governance semantics for all 14 sections

---

## Purpose

This document defines the structure and validation rules for each of the 14 governance sections stored in `.laputa/sections/`:

- **Mandatory fields:** required in all valid sections
- **Section-specific fields:** per-section structure
- **Mutability rules:** explicit vs. append-only
- **Validation:** constraints and invariants

---

## Mandatory Metadata

All 14 sections include:

```json
{
  "_meta": {
    "updated_at": "2026-08-01T12:34:56Z",
    "version": "1.0"
  }
}
```

| Field | Type | Meaning |
|-------|------|---------|
| `_meta.updated_at` | RFC3339 | Timestamp of last mutation (UTC) |
| `_meta.version` | string | Schema version (e.g., "1.0") |

---

## Section Schemas

### 01-identity.json

**Mutability:** explicit

**Schema:**

```json
{
  "_meta": { "updated_at": "...", "version": "1.0" },
  "role": "string",
  "capabilities": ["string"],
  "constraints": ["string"],
  "sop": ["string"]
}
```

**Fields:**
- `role` — agent role description (e.g., "autonomous rhythm reporter")
- `capabilities` — list of capabilities
- `constraints` — list of operational constraints
- `sop` — standard operating procedures

**Example:**
```json
{
  "role": "MemoryOS curator",
  "capabilities": ["recall", "ingest", "classify"],
  "constraints": ["no silent mutation"],
  "sop": ["validate before apply"]
}
```

---

### 02-relationship.json

**Mutability:** explicit

**Schema:**

```json
{
  "_meta": { "updated_at": "...", "version": "1.0" },
  "relationships": [
    {
      "agent_id": "string",
      "type": "string",
      "resonance": {}
    }
  ],
  "resonance": {}
}
```

**Fields:**
- `relationships` — list of agent relationships with resonance signals
- `resonance` — global resonance state

---

### 03-commitment.json

**Mutability:** explicit

**Schema:**

```json
{
  "_meta": { "updated_at": "...", "version": "1.0" },
  "commitments": ["string"],
  "red_lines": ["string"]
}
```

**Fields:**
- `commitments` — list of commitments (what this agent is committed to)
- `red_lines` — hard constraints that cannot be violated

---

### 04-preferences.json

**Mutability:** explicit

**Schema:**

```json
{
  "_meta": { "updated_at": "...", "version": "1.0" },
  "settings": {},
  "mode": "string"
}
```

**Fields:**
- `settings` — arbitrary preference key-value pairs
- `mode` — current operational mode

---

### 05-memory_md.json

**Mutability:** explicit

**Schema:**

```json
{
  "_meta": { "updated_at": "...", "version": "1.0" },
  "summary": "string",
  "ltm_entries": ["string"]
}
```

**Fields:**
- `summary` — distilled long-term memory summary
- `ltm_entries` — list of LTM entries (references to Mentle)

---

### 06-history_md.json

**Mutability:** explicit

**Schema:**

```json
{
  "_meta": { "updated_at": "...", "version": "1.0" },
  "timeline": [
    {
      "event": "string",
      "timestamp": "RFC3339",
      "context": "string"
    }
  ]
}
```

**Fields:**
- `timeline` — historical event log with timestamps

---

### 07-daily.json, 08-weekly.json, 09-monthly.json

**Mutability:** append-only

**Schema:**

```json
{
  "_meta": { "updated_at": "...", "version": "1.0" },
  "reports": [
    {
      "title": "string",
      "summary": "string",
      "highlights": ["string"],
      "open_questions": ["string"],
      "generated_at": "RFC3339"
    }
  ]
}
```

**Fields:**
- `reports` — array of rhythm reports (append-only)
- Each report includes title, summary, highlights, and open questions

---

### 10-journal_reflective.json

**Mutability:** append-only

**Schema:**

```json
{
  "_meta": { "updated_at": "...", "version": "1.0" },
  "entries": [
    {
      "date": "RFC3339",
      "reflection": "string",
      "tags": ["string"]
    }
  ]
}
```

**Fields:**
- `entries` — reflective journal entries (append-only)

---

### 11-proposal_inbox.json

**Mutability:** explicit

**Schema:**

```json
{
  "_meta": { "updated_at": "...", "version": "1.0" },
  "proposals": [
    {
      "id": "string",
      "type": "string",
      "status": "pending|approved|rejected",
      "content": "string",
      "submitted_at": "RFC3339"
    }
  ]
}
```

**Fields:**
- `proposals` — evolution proposals pending review
- `status` — proposal status (pending, approved, rejected)

---

### 12-changelog.json

**Mutability:** append-only

**Schema:**

```json
{
  "_meta": { "updated_at": "...", "version": "1.0" },
  "entries": [
    {
      "timestamp": "RFC3339",
      "section": "string",
      "action": "set|append|delete",
      "authority": "string",
      "summary": "string",
      "prior_value_hash": "string"
    }
  ]
}
```

**Fields:**
- `entries` — audit log of all mutations (append-only)
- `prior_value_hash` — SHA256 hash of prior section state for rollback

---

### 13-report_indexes.json

**Mutability:** explicit

**Schema:**

```json
{
  "_meta": { "updated_at": "...", "version": "1.0" },
  "indexes": [
    {
      "name": "string",
      "section": "string",
      "query": "string"
    }
  ]
}
```

**Fields:**
- `indexes` — search indexes over reports

---

### 14-aaak_summaries.json

**Mutability:** explicit

**Schema:**

```json
{
  "_meta": { "updated_at": "...", "version": "1.0" },
  "summaries": [
    {
      "entity_code": "string",
      "topics": ["string"],
      "emotion_codes": ["string"],
      "flag_codes": ["string"]
    }
  ]
}
```

**Fields:**
- `summaries` — AAAK dialect symbolic summaries

**AAAK Codes:**
- **Entity:** 3-letter uppercase (KAI, MAX, PRI, etc.)
- **Topics:** frequency-based with proper noun boosting
- **Emotions:** vul, joy, fear, trust, grief, wonder, rage, etc.
- **Flags:** DECISION, ORIGIN, CORE, PIVOT, TECHNICAL, etc.

---

## Validation Rules

All sections must:

1. Include `_meta` with `updated_at` and `version`
2. Have `updated_at` in RFC3339 format (UTC)
3. Have `version` as semver string (e.g., "1.0")
4. Be valid JSON

Append-only sections (07-09, 10, 12):
- New entries are appended to array; old entries never modified
- Queries may filter by timestamp, but cannot update history

---

## MANUAL

When updating:

1. Add new section schema to this file
2. Update parent `.laputa/AGENTS.md` with section # and purpose
3. Increment version if schema changes
4. Add migration logic to laputa.Engine if version changes
5. Do not remove sections; archive old schemas to docs/archive/

Parent reference: `../AGENTS.md`
