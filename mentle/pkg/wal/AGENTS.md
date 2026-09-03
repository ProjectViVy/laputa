<!-- Parent: ../../AGENTS.md -->

# mentle/pkg/wal — Non-Authority Legacy Utility

Accepted ADR-0014 explicitly rejects JSONL WAL as canonical memory recovery. `canonical.sqlite3` is the sole logical authority and transactional `index_jobs` is the derived-index recovery log.

- Do not replay this WAL into canonical memory or derived indexes.
- Do not add new memory mutation producers or dual-write coordination.
- Existing append/checkpoint tests may verify the utility's local file behavior, but they are not authority durability evidence.
- E03-S09 removes unused business seams or renames/scopes any retained use as diagnostics only, with clean resource shutdown.

Parent reference: `../AGENTS.md`
