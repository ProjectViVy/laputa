# Summary

Concurrent scope binding no longer races on Garden's shared native adapter cache. Each of 32 workspace scopes retains its own correctly bound adapter. The repair adds one private cache mutex and does not hold it over model calls or ingestion shutdown.

This is a local development increment. Full memory-loop acceptance, same-candidate packaging, Windows and live-model checks remain open. No push, merge or release.
