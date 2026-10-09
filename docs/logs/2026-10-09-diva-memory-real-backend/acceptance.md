# Acceptance

From garden, run go test -json ./agentapi -run ^TestMemoryLoopRealBackend -count=1 with the Go compiler on PATH. Confirm the actual pinned mentle/models assets exist. Run the complete ./agentapi suite and ./... after building Console dist. Fix independent-module INOFY/sum and e2e model configuration before claiming baseline green.
