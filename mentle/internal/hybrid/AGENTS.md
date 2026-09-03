<!-- Parent: ../../AGENTS.md -->

# mentle/internal/hybrid — Derived Hybrid Retrieval

Hybrid retrieval combines lexical BM25 and vector candidates. Both indexes are rebuildable projections subordinate to canonical SQLite.

- Search/rerank may apply query, scope, status and budget filters; it does not apply Persona authority or read WORLD/ACTMEM.
- Include only active current canonical revisions. Tombstoned and old physical versions cannot crowd out valid candidates.
- BM25 rebuild is sourced from complete active canonical content or an equivalent uncapped canonical stream, never a capped vector listing.
- Results are deterministic for identical inputs and preserve source/provenance references.
- Empty, degraded and unavailable states remain distinguishable to callers.

```bash
cd mentle
GOSUMDB=off go test ./internal/hybrid/... ./facade/...
```

Parent reference: `../AGENTS.md`
