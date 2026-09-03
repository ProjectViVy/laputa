<!-- Parent: ../AGENTS.md -->

# mentle/pkg — Protocol and Utility Packages

Reusable packages under `pkg/` support adapters and diagnostics. They do not own logical memory authority.

## MCP

- MCP memory mutations call `facade.Service`; they never mutate Searcher/vector storage directly.
- Search returns query-matched cards/results, and evidence content remains an explicit bounded read.
- Protocol errors preserve stable domain codes and do not leak paths or credentials.

## Legacy JSONL WAL

The `pkg/wal` code may be retained temporarily for non-authoritative diagnostics while Epic 3 audits its callers. It is not canonical recovery and cannot reconstruct or override `canonical.sqlite3`.

- Canonical memory commits and transactional `index_jobs` implement durability/recovery under ADR-0014.
- Do not add dual writes, WAL replay into canonical memory, format migration requirements or new business mutation producers.
- Remove unused WAL seams and close retained resources during E03-S09.

## Verification

```bash
cd mentle
GOSUMDB=off go test ./pkg/... ./facade/...
```

Parent reference: `../AGENTS.md`
