# Durable native capture projection

Reuse the existing raw-first ingestion ledger for typed activity intent and operation receipts. ACTMEM Markdown remains the sole activity authority. Fresh in-process captures opt in through JSON-excluded metadata containing admitted, already-redacted user content; native terminal phase and profile/session/workspace/source sequence come from the bound host. Previously accepted rows remain not_requested, with no backfill.

One existing ingest worker owns ordered projection. Persist applying plus the original opaque head fingerprint before one atomic native Pulse/Recap append; persist applied with real entry IDs and original revision afterwards and clear the temporary recap source. On interrupted applying/unknown, rejoin exact retained entries/capsules; absent entries plus an unchanged original pre-write head prove no effect. Changed/partial/conflicting state remains fenced. Later watermarks cannot cross the first unresolved requested projection in their bound scope.

A service-owned one-second recovery timer runs only after projection failure, stops on Close and retries original work without needing a new queue delivery. ArchiveCapturedSession is an in-process lifecycle boundary, not an HTTP maintenance action: after the host seals/drains producers, wait for the session's original projections under the existing worker lock and fold through the native owner. Failure never deletes host messages.

Keep existing source receipt IDs/content, public transport fields, Work CAS and ACTMEM grammar. No new model, backend, runtime, source-authority store or dependency pin. S05/S06/S10/S11 formal gates and complete crash matrices remain open.
