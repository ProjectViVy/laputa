# Mission effect boundary fence

Native EvolutionPorts can pin the Mission revision of each persisted run while reusing the one existing domain/ledger. Zero is an actual unassigned revision. Each pinned effect or recovery Lookup checks that pin under the same owner mutation gate used by HumanClient Persona writes/review decisions. Model inference holds no mutation gate. Runtime owner lifetime is held during domain operations so Close cannot race a checked effect.

This repairs actual DIVA stale Mission settlement: changing Mission after reflect request admission previously allowed watermark1. Native owner now prevents subsequent effect/recovery under that stale binding. Ordinary human results/receipts remain readable through their existing controls; no unknown run is reset or retried.

Scope/destination, native authorities, fixed effect vocabulary, policy controls and engine pin stay intact. No new ledger/runtime, backfill, push or release.
