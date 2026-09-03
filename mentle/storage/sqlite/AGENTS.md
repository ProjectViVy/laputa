<!-- Parent: ../../AGENTS.md -->

# mentle/storage/sqlite — SQLite Helpers

SQLite helpers serve explicitly owned Mentle data. The canonical memory schema and transactional `index_jobs` boundary belong to `facade` under ADR-0014.

- `canonical.sqlite3` is the sole authority for logical memory identity, content, metadata, version, status, idempotency and audit state.
- Schema changes require safe forward migration and fault tests; optimistic concurrency is enforced in SQL at commit.
- JSONL WAL, vector stores and BM25 cannot reconstruct or override canonical rows.
- Repair/reindex may read canonical state consistently but may replace only derived artifacts; canonical SQLite is never renamed, overwritten or deleted.
- Knowledge-graph SQLite remains a separate provenance-bearing subsystem and does not become Persona/ACTMEM authority.

Run the owning package tests and `GOSUMDB=off go test ./facade/...` after changes.

Parent reference: `../AGENTS.md`
