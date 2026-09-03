# Garden Authority & Recovery — Operator Runbook

**Scope:** local Garden/Mentle deployments after the Waves 0–3 clean-break cutover.

This runbook treats `canonical.sqlite3` as the only recoverable memory authority. `vectors.db`, the in-memory BM25 projection, and the legacy JSONL WAL are disposable or diagnostic artifacts. None can reconstruct, override, or roll back canonical memory.

## 1. Before maintenance

1. Record the profile, Mentle palace path, current `/health`, and `/v2/admin/index-health` response.
2. Stop Garden and any Mentle writer/import process before taking a file-level backup or restoring an artifact. Do not copy a live SQLite file with `Copy-Item` or `cp` as a substitute for SQLite's backup API.
3. Keep the capability token out of shell history and logs. `GARDEN_CAPABILITY_OPERATOR_TOKEN` is for explicit repair operations; the read token is sufficient for health probes.

The clean-break service deliberately has no HTTP rebuild endpoint. `GET /v2/admin/index-health` is read-only. Derived repair is an operator CLI operation.

## 2. Back up canonical memory

Use the SQLite backup API while the service is stopped (or through an equivalent SQLite backup-capable tool):

```text
sqlite3 <palace>/canonical.sqlite3 ".backup '<backup-dir>/canonical-YYYYMMDD-HHMMSS.sqlite3'"
```

Verify that the backup opens and contains the expected `memories` and `index_jobs` tables before considering it usable. If the `sqlite3` utility is unavailable, use a small maintenance program that calls SQLite's online backup API; do not replace it with a raw file copy of a live database.

Back up `vectors.db` separately only as a disposable rollback convenience. Do not treat the WAL directory, vector payloads, BM25 state, or any JSON export as a canonical backup.

## 3. Rebuild derived indexes

From the Mentle module, run the operator repair command against the target palace:

```powershell
cd mentle
$env:GOSUMDB = 'off'
go run . --palace '<palace>' repair
```

The command reads an active/current snapshot from `canonical.sqlite3`, checks embedding identity, prepares and verifies derived vector state, then replaces only the derived artifact. BM25 is rebuilt from the same canonical snapshot. A same-volume staging manifest records the snapshot contract. Canonical SQLite is not moved, renamed, overwritten, or reconstructed. Pending `index_jobs` remain replayable work and are drained by the service lifecycle.

If the command fails, preserve the failure output and inspect `last_rebuild.error_code`; do not delete or edit `canonical.sqlite3` to make health green.

## 4. Verify live health

```text
GET /health
GET /v2/admin/index-health
Authorization: Bearer <read-or-operator-capability-token>
```

`/health` and `/v2/admin/overview` aggregate the same live Mentle probe. A healthy response must be based on current probes, not a startup label or a missing endpoint. `status=degraded` is valid and must be investigated using `reasons`, including:

- `index_jobs_pending` or `index_jobs_failed`;
- `vector_count_diverged`, `bm25_count_diverged`, or `tombstone_pressure`;
- `embedding_identity_missing` or `embedding_identity_mismatch`;
- `rebuild_running` or `last_rebuild_failed`.

`503 index_health_unavailable` means a minimum probe could not produce a trustworthy report. The error details expose only public reason codes.

## 5. Rollback and incident boundaries

- A failed derived rebuild is expected to retain the previous usable derived index. If an artifact rollback is necessary, stop writers, restore only the separately backed-up `vectors.db`, restart, and let BM25 regenerate from canonical SQLite. Recheck live health and tombstone/count divergence.
- A canonical rollback is a separate data-recovery incident: stop all writers, preserve the current database, and restore only from a verified SQLite backup using SQLite tooling and an explicit owner decision. Rebuild derived indexes afterward.
- Never restore `vectors.db`, BM25 output, JSONL WAL, or a Console export over `canonical.sqlite3`. Never replay the WAL into canonical memory.
- If the embedding identity changes, do not mix vectors. Resolve the identity mismatch through an explicit staged rebuild and verify the result before resuming normal writes.

Record the operator, time, backup path/checksum, command output, health response, reason codes, and whether canonical data was changed. Routine index repair must leave canonical row IDs, versions, statuses and `index_jobs` untouched.
