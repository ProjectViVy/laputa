# Verification

RED: chain-recall-backend-cache-red.jsonl records the actual race and fatal map access. GREEN: the identical probe under -race -count=3 passes three named executions, zero skips, exit 0. Garden ./internal/... regression passes 383 named tests/subtests, zero skips, exit 0.

```bash
source /workspace/work/memory-loop/tools/environment.sh
# From garden:
go test -race -json ./internal/runtimecore -run '^TestBackendForConcurrentScopeBindings$' -count=3
go test -json ./internal/...
```

Raw logs are retained in the task checkpoint. The fixture's lexical native facade does not attest ONNX/App concurrency or direct unsynchronized Garden.Close use. Native client lifetime guards remain the owner of shutdown admission.
