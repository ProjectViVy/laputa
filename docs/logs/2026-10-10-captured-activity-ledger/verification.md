# Verification

Task environment; from garden:
- go test -race -json ./internal/ingest ./agentapi -count=1: chain-activity-native-source-final-race.jsonl, 104 pass, zero fail/skip, observed exit0.
- go test -json ./internal/ingest ./internal/runtimecore ./agentapi -count=1: chain-activity-native-final-default.jsonl, 120 pass, zero fail/skip, observed exit0.

REDs preserved: missing ingest API; actual no-new-queue retry timeout; archive boundary API; readable-capsule failure discovered by the native archive test (repaired separately in ce919da). Real canonical OpenCatalogService/native file owner are used, with no SQL-seeded success/fake backend. Tests cover old accepted rows, raw-first gating, replay/original receipts, interrupted original intent before/after effect, capsule recovery, changed-head unknown fence, retry and archive. Controlled interrupted stages are service units, not a claimed six-cut App crash matrix.

git diff --check passes. Full phase CI/conformance/source sealing/native identity/fresh whole-phase review remain pending.
