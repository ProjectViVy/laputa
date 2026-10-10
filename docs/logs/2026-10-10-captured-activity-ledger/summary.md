# Native captured activity ledger

The real ingestion ledger now projects trusted fresh terminal user source into native bounded Pulse/Recap pairs with durable intent/receipt recovery and a source-watermark fence. The ingest worker recovers transient projection failures without new messages. A host lifecycle boundary waits for the original session projection and archives it before message deletion. Native composition assigns the one existing ACTMEM owner before worker Start.

This closes native plumbing prerequisites. Full App proof is recorded in the VIVY activity increment, separately from these store/service tests. No accepted-row backfill, automatic ACTMEM recall, persona mutation, release, merge or pin change.
