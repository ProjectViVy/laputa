# Accepted capture receipt recovery

Garden exposes a read-only capture receipt lookup from a host-bound session and terminal Run/EventSeq provenance. The service performs the existing principal/binding policy checks and derives the full event key; ingest reads only the existing session/event row. Changed-content submission still rejects its old hash conflict. No new database table, receipt authority or mutation path is introduced.

The Mentle MCP fixture filters inherited MEMPALACE configuration overrides from its child environment so the isolated config cannot redirect it into a production palace. Unrelated environment settings are preserved.

No release: local paired repair branch only.
