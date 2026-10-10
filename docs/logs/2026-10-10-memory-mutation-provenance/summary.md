# Canonical memory mutation provenance

Garden's Mentle adapter discarded evolution Sources; its writer validation also accepted source scopes outside the admitted read union. Map the primary typed reference into the native MemorySource locator/revision, retain the full evolution SourceRef set in the same canonical row's metadata, and reject foreign subject/workspace sources before mutation. Personal sources in an admitted workspace read union remain valid. Locators encode opaque identifiers and scope with net/url; they are provenance data, not bearer permissions or new retrieval APIs.

Mentle's atomic update previously changed only content/version, retaining stale source and inference metadata. Update supplied source/metadata with the body in the same expected-version SQL statement and receipt transaction. Metadata keys merge into the existing row; omitted source/metadata fields preserve prior values. The transactional index outbox uses those new canonical metadata bytes. No schema change, second provenance store, data migration or old-receipt rewriting.

Reference revisions retain their actual supplied values. An ingestion reference with revision 0 remains unspecified; it is not relabeled with a guessed canonical memory version. Existing memory receipts replay their original results, without silently backfilling old records.
