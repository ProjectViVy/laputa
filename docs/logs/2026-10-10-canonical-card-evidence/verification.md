# Verification

Collection RED: chain-recall-card-collection-red.jsonl. Canonical facade regression after both fixes: 71 named tests/subtests pass, zero skips, exit 0 (chain-recall-canonical-read-regression.jsonl). Expansion initial compile error and one uninstrumented green are retained; the race-instrumented RED repeats ten mixed snapshots. Identical concurrent probe after repair passes ten repeated executions, zero skips, exit 0 (chain-recall-expansion-revision-green.jsonl).

Garden native backend, agentapi and internal regression: 461 named tests/subtests pass, zero skips, exit 0 (chain-recall-garden-canonical-regression.jsonl).

```bash
source /workspace/work/memory-loop/tools/environment.sh
# From mentle:
go test -json ./facade -count=1
# From garden:
go test -race -json ./backends/mentle -run '^TestConcurrentExpansionKeepsOneCanonicalRevision$' -count=10
go test -json ./backends/mentle ./agentapi ./internal/... -count=1
```

New concurrent probe uses supported real canonical/BM25 mutations and reads, 200 record versions and 1500 expansion attempts per execution. It is native contract proof, not full ONNX/App acceptance. Existing facade regression includes its original test doubles. Source freeze, full CI, sealed pack and review remain due.
