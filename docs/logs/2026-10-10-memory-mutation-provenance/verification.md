# Verification

RED: adapter creation lost the source set; foreign subject and foreign workspace references were accepted. After adapter correction, update RED showed new content with old source/inference metadata. A separate native facade test independently reproduced the stale-source update.

GREEN: full Garden486 named tests/subtests, zero skip/fail, exit0; full Mentle537, zero skip/fail, exit0. Facade baseline69 and final70 pass. Adapter tests use the supported real OpenCatalogService canonical/BM25 development seam, not an injected memory Backend. The native update/outbox unit disables only its disposable Hybrid projection to observe the real committed pending job; no SQL data is seeded. An earlier assertion incorrectly expected an already-consumed outbox row; that fixture failure is preserved, then corrected to observe the explicit derived-unavailable case.

Paired actual DIVA App automatic/large-source/reflection-process-reopen composition:13 pass, zero skip/fail, exit0. It joins canonical SourceRefs to the actual source provider, ingestion sequence and bound scope and preserves memory/receipt/provenance across a new OS PID. Raw logs are in the DIVA v0.5 canonical-provenance checkpoint.

Final frozen-source complete CI, conformance reproduction, regenerated candidate/Inspect/native identity and fresh review remain pending for the remaining local chain.
