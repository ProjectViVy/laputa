# Verification

- Mentle CLI: initial four dependency/build failures -> four passing tests after bounded module graph closure.
- Garden `go test -tags=e2e -json ./e2e -count=1`: 2 pass / 0 skip / exit 0 with explicit real models. One test uses an existing domain double; the other starts the real Garden process/backend.
- Mentle full suite before config repair: 530 pass, 5 MCP skip; exit 0 was insufficient.
- MCP config admission: `TestMCPProtocolLineFraming` observed RED (unsupported config-dir flag / no initialize response), then all 7 MCP protocol tests GREEN with real local ONNX/backend.
- Mentle `go test -json ./... -count=1` after repair: 535 pass, 0 skip, 0 fail, exit 0.
- Independent Laputa suite after task-local pinned INOFY layout: 63 pass, 0 skip, exit 0.

Raw logs are retained in the isolated memory-loop workspace and collected by the DIVA continuation record. Native Windows/live-model acceptance is separate.
