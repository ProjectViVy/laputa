# Memory test environment repair

Mentle standalone CLI builds required the already selected indirect `github.com/google/renameio v1.0.1` dependency and its verified sums. Garden's real process e2e now configures the pinned local model directory explicitly. Mentle MCP now accepts `server --config-dir <directory>` and forwards it to the existing facade ConfigDir option, allowing isolated palace/model configuration without changing HOME. MCP tests build into their own temporary root, fail instead of skip when the server does not respond, enforce a process deadline and kill only their owned process. Broad name-based cleanup is removed.

The default server configuration behavior is preserved. No module upgrade or alternate authority/backend was introduced. Release is not applicable: local verification branch only.
