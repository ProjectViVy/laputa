<!-- Parent: ../AGENTS.md -->

# garden/internal/ingest — Material Ingestion & Normalization

**Generated:** 2026-08-01  
**Purpose:** Normalize and ingest materials (code, docs, conversations) into Mentle

---

## Purpose

The `ingest/` package handles material ingestion from diverse sources:

- **Source normalization** — standardize code files, docs, chat logs, etc.
- **Metadata extraction** — source URI, revision, timestamp, author
- **Fragment creation** — break large materials into indexable chunks
- **Mentle integration** — write to material store via facade
- **Idempotency** — dedup by content hash

---

## Structure

```
ingest/
├── service.go       # Ingest service
└── service_test.go
```

---

## Key Concepts

### Ingest Request

```go
type IngestRequest struct {
    SourceType string              // "code", "doc", "chat", "memory"
    SourceURI  string              // file path or identifier
    Content    string              // raw material
    Metadata   map[string]any      // author, timestamp, etc.
    Scope      string              // session or project scope
}
```

### Material Fragment

After ingest, material is split into fragments:
- Fragment size: configurable (default 1024 chars)
- Overlap: configurable (default 128 chars)
- Metadata preserved per fragment
- Content hash for dedup

### Service Operations

**Ingest(ctx, request):**
- Normalize source type
- Extract metadata
- Create fragments
- Write to Mentle via facade
- Return count of fragments stored

**IngestBatch(ctx, requests):**
- Process multiple materials
- Deduplicate across batch
- Write in parallel
- Report per-source results

---

## Testing

```bash
cd garden
GOSUMDB=off go test -v ./internal/ingest/...
```

**Behavioral tests:**

- Code files are normalized correctly
- Metadata is preserved
- Large materials are chunked
- Dedup prevents duplicate storage
- Ingest is idempotent on content hash
- Scope is preserved

---

## Conventions

- All sources are treated as read-only
- Content hash is computed once per ingest
- Fragment size never exceeds Mentle limits
- Metadata is user-readable

---

## MANUAL

Keep ingest focused on normalization. Mining logic (extraction of entities, decisions) goes to Mentle.

Parent reference: ../AGENTS.md
