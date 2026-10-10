# Verification

`TestUnresolvedWorkReceiptStopsLaterReflection` unknown/rejected cases: RED executed two model calls and two effects; GREEN executes only the reconcile model call and original WorkPatch effect, retains its receipt and returns the appropriate owner outcome.

Full Laputa module: 67 pass, zero skip/fail, exit 0 (`chain-work-receipt-regression.jsonl`). Full Garden module: 480 pass, zero skip/fail, exit 0 (`chain-garden-work-receipt-regression.jsonl`). The library boundary unit uses explicit scripted Domain outcomes; it is not real-backend positive acceptance. Paired VIVY real rejected-update composition and durable settlement regression are recorded in the DIVA v0.3 checkpoint.

Final new-source conformance/candidate sealing and the complete crash recovery matrix remain pending. No old conformance hash is reused for this revision.
