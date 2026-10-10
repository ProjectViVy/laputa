# Verification

Direct real model RED: chain-recall-embedder-lifetime-red.jsonl. Final single/batch closed-owner assertions and three repeated native deadline/Close race executions: chain-recall-embedder-lifetime-final.jsonl, three pass, zero skips, exit 0. Full embedder and facade regression: 77 named tests/subtests pass, zero skips, exit 0 (chain-recall-embedder-facade-regression.jsonl).

```bash
source /workspace/work/memory-loop/tools/environment.sh
# From mentle:
go test -race -json ./internal/embedder -run '^TestCanceledNativeEmbeddingCanCloseSafely$' -count=3
go test -json ./internal/embedder ./facade -count=1
```

The local bundled ONNX model is loaded explicitly; no fake engine, model download or SQL seeding. The fixture has no exact native forward-start barrier, so repeated race reproduction and the source-level pinned Hugot cancellation behavior establish the root cause. Final full App race/degradation and whole-candidate checks remain due.
