# Verification

Initial missing-kernel API compile RED and a fixture enum typo are retained in chain-activity-native-pair-initial-red.jsonl; corrected API RED is separate. First kernel execution exposed empty-head rendering after final fold (separate committed repair), retained in chain-activity-native-pair-first.jsonl.

Final four file-backed kernel cases repeated three times under -race pass 12 named executions, zero skips, exit0 (chain-activity-native-pair-final-race.jsonl). Before the final bounded-head-only fresh-path adjustment, full native module regression passes74/0skip/exit0 and pair race12/0skip/exit0. The final focused tests cover the adjusted path and explicit archive recovery.

```bash
source /workspace/work/memory-loop/tools/environment.sh
# From laputa:
go test -race -json ./actmem -run '^TestCapturedPair' -count=3
go test -json ./... -count=1
```

Real file kernel proof, not real primary-turn continuity, ONNX, complete crash cuts or final candidate attestation. Exact source ledger receipts, unknown outcomes and original-operation reconciliation remain to be integrated.
