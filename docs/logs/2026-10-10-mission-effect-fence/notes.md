# Decisions

Reuse one native domain/ledger and derive a private per-run wrapper from owned EvolutionPorts.WithMissionRevision. Never open another writer to pin a run. Client lifetime read lock precedes authority mutation lock on both effect and human paths, avoiding opposing lock order during Close.

The native gate covers host-exposed Persona mutation and pinned effects/recovery Lookup; supported authority directory still has one owner process. Generic host Mission checks alone cannot atomically serialize a native authority mutation, so the owner performs the same check inside its commit gate.

Old-run receipt queries through human results do not change native engine state. Stale-pin workflow recovery remains refused; no migration, permissive wildcard or invented completed projection.
