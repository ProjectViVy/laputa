# Verification

Observed RED: chain-activity-fold-provenance-red.jsonl, both TestFoldCapturedProvenanceRemainsReadable and TestFoldOversizeMetadataRefusesBeforeRemovingSources failed. Initial compact serialization alone still refused the bounded long recap; preserve chain-activity-fold-provenance-first.jsonl.

Final command (task environment): cd laputa && go test -race -json ./actmem -count=1. chain-activity-fold-provenance-final-green.jsonl: 27 pass, zero fail/skip; observed command exit 0. Includes native pair replay, archive recovery, scope, capacity, read budgets and the two new regressions. git diff --check passed. Full phase CI/source conformance, artifact pack, native identity and fresh whole-phase review remain pending.
