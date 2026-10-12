# Garden Codex CLI Persona adapter

Opt-in native command hooks deliver additive developer context through `garden-host-context`. Qualified on **Linux x86_64, Codex0.162.1**: installed native app-server persistent root turns and a native `codex exec` startup/prompt boundary. Hooks remain experimental. [Exact qualification and limits](../../../docs/research/garden-cli-native-qualification-2026-10-12.md); [authoritative Story index](../../../docs/superpowers/plans/garden-standalone-v1/index.md).

From the existing selected workspace, use the qualified native local installer:

```sh
export GARDEN_HOST_CONTEXT_BIN=/absolute/bin/garden-host-context
export GARDEN_HOST_CONTEXT_CONFIG=/absolute/private/bridge.json
codex plugin marketplace add /absolute/laputa/garden/adapters/codex-cli --json
codex plugin add garden@garden-local --json
```

The native installer records registration in the selected Codex home. Review and trust both native hook hashes through Codex's hook controls before enabling contributions; an updated handler needs fresh trust. Qualification used native `hooks/list` to inspect each exact `currentHash` and explicitly trusted only those handlers in the isolated owner configuration.

## Owner configuration

Build `garden/cmd/garden-host-context` and select its absolute executable path. Supply a Garden-owned JSON file with mode0600 inside an owner directory with mode0700. Required fields: `endpoint`, `binding_id`, `credential_file`, `host_kind`, `host_version`, `host_executable` (absolute actual native binary), `host_home` (canonical selected native home), `host_instance`, `scope_fingerprint` (SHA256 of canonical existing cwd) and `cwd`. `credential_file` references a separate private owner file; never embed the token in settings or commands. `endpoint` must be numeric loopback HTTP.

Installation/loading is an explicit native owner action. This package has no installation script, shared-settings merger, instruction/history writer or credential reader during discovery. Separately configure the matching Garden binding, enable PersonaSync, preview and confirm its current revision. Installation alone does not authorize Persona. Existing instructions, workspace and history remain.

## Delivery boundary

SessionStart (`startup|resume|clear|compact`) and UserPromptSubmit invoke the same quoted command/config paths without interpolating event content. Native timeout2s encloses the bridge's1.5s deadline and1s HTTP limit. Garden bounds output to6000 Unicode characters. The bridge verifies actual binary version, native home, canonical OS/event cwd, persistent transcript-path parent, private file ownership and authenticated S15 admission. It never reads the transcript. Only five metadata fields reach Garden.

Wrong scope/version, identified workers/fork-start, lock/pending, offline or revoke omit new context without stale fallback. Ordinary later root-fork prompts expose no fork metadata and remain root turns. Already admitted Garden can remain in history. First turns can contain both event contributions. Status is **offered**; there is no file-written or loaded receipt. TUI frontends, other versions/OS and automatic memory lifecycle remain unqualified.

Set `host_kind` to `codex-cli`, `host_version` to `0.162.1` and `host_home` to the canonical selected CODEX_HOME. Codex null transcript omits ephemeral events. Actual fresh native children receive no Garden; fork children can retain earlier parent Garden in copied history. No Persona-free fork history is claimed. Startup/resume/clear/compaction and N→N1 were observed in actual native requests.

Both handlers set `additionalContextLimit:0`, preventing Codex's default2500-token spill/truncated preview while Garden's own bound remains. Actual full-byte delivery of a large Unicode projection is qualified for both handlers.

## Separate explicit memory

The existing `garden-mcp` server remains independent of PersonaSync. Enable only desired Recall/Capture controls and add this entry through owner-native MCP configuration, preserving other servers:

```toml
[mcp_servers.garden]
command = "/absolute/bin/garden-mcp"
[mcp_servers.garden.env]
GARDEN_HOST_CREDENTIAL_FILE = "/absolute/private/credential"
GARDEN_HOST_SESSION_ID = "owner-selected-memory-session"
GARDEN_MCP_BASE_URL = "http://127.0.0.1:7373"
```

Keep the explicit memory session namespace stable for retries; it is not an automatic host-conversation continuity mapping. The existing12 tool names remain. Selected `record_signal` accepts `event_id` for stable receipts and optional `shared_space_id`; shared writes require current read_write membership. Changed payload under the same selected event conflicts. Native MCP RPC qualified write/replay/search/expansion, unrelated-server preservation and revoke denial with PersonaSync off. No transcript capture, Recap policy or S09 native memory-provider completion follows.
