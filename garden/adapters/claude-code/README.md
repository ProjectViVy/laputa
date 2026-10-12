# Garden Claude Code Persona adapter

Opt-in native command hooks deliver additive conversation context through `garden-host-context`. Qualified on **Linux x86_64, Claude Code2.1.296**, using persistent print/stream-json CLI. Hooks remain experimental. [Exact qualification and limits](../../../docs/research/garden-cli-native-qualification-2026-10-12.md); [authoritative Story index](../../../docs/superpowers/plans/garden-standalone-v1/index.md).

From the existing selected workspace, activate the qualified local package:

```sh
export GARDEN_HOST_CONTEXT_BIN=/absolute/bin/garden-host-context
export GARDEN_HOST_CONTEXT_CONFIG=/absolute/private/bridge.json
claude -p "Your request" --plugin-dir /absolute/laputa/garden/adapters/claude-code \
  --settings '{"enabledPlugins":{"garden@inline":true}}'
```

This is the qualified local loading recipe; no marketplace publication is required. The manifest defaults to disabled. The inline name is `garden@inline`; a bare `garden` key did not activate the pinned runtime.

## Owner configuration

Build `garden/cmd/garden-host-context` and select its absolute executable path. Supply a Garden-owned JSON file with mode0600 inside an owner directory with mode0700. Required fields: `endpoint`, `binding_id`, `credential_file`, `host_kind`, `host_version`, `host_executable` (absolute actual native binary), `host_home` (canonical selected native home), `host_instance`, `scope_fingerprint` (SHA256 of canonical existing cwd) and `cwd`. `credential_file` references a separate private owner file; never embed the token in settings or commands. `endpoint` must be numeric loopback HTTP.

Installation/loading is an explicit native owner action. This package has no installation script, shared-settings merger, instruction/history writer or credential reader during discovery. Separately configure the matching Garden binding, enable PersonaSync, preview and confirm its current revision. Installation alone does not authorize Persona. Existing instructions, workspace and history remain.

## Delivery boundary

SessionStart (`startup|resume|clear|compact`) and UserPromptSubmit invoke the same quoted command/config paths without interpolating event content. Native timeout2s encloses the bridge's1.5s deadline and1s HTTP limit. Garden bounds output to6000 Unicode characters. The bridge verifies actual binary version, native home, canonical OS/event cwd, persistent transcript-path parent, private file ownership and authenticated S15 admission. It never reads the transcript. Only five metadata fields reach Garden.

Wrong scope/version, identified workers/fork-start, lock/pending, offline or revoke omit new context without stale fallback. Ordinary later root-fork prompts expose no fork metadata and remain root turns. Already admitted Garden can remain in history. First turns can contain both event contributions. Status is **offered**; there is no file-written or loaded receipt. TUI frontends, other versions/OS and automatic memory lifecycle remain unqualified.

Set `host_kind` to `claude-code`, `host_version` to `2.1.296`, `host_executable` to the actual Claude ELF binary and `host_home` to the canonical selected CLAUDE_CONFIG_DIR. A normal transcript path is also supplied by fresh no-persistence mode, so this pin additionally requires matching native CLAUDE_CODE_SESSION_ID/CLAUDE_PID, a bounded same-UID ancestor chain, exact configured ELF and bounded native cmdline. `--no-session-persistence` or unknown process ownership omits before credential/HTTP. These checks qualify Linux only.

Actual fresh/resumed no-persistence requests receive no new Garden. A real general-purpose child receives none. The native `fork` Agent type was unavailable in this tested session, so fork-worker execution is unqualified. Startup/resume/clear/compaction and N→N1 were observed in actual native requests. Context sits below native instruction priority; the package does not replace CLAUDE.md or the system prompt.

## Separate explicit memory

The existing `garden-mcp` server remains independent of PersonaSync. Enable only desired Recall/Capture controls. An owner-selected private MCP file can reference the credential without embedding it:

```json
{"mcpServers":{"garden":{"command":"/absolute/bin/garden-mcp","env":{"GARDEN_HOST_CREDENTIAL_FILE":"/absolute/private/credential","GARDEN_HOST_SESSION_ID":"owner-selected-memory-session","GARDEN_MCP_BASE_URL":"http://127.0.0.1:7373"}}}}
```

The pinned native CLI accepts `--mcp-config /absolute/existing.json /absolute/private/garden-mcp.json`; retain other entries and approve only intended tools using native permission controls. Keep the explicit memory session namespace stable for retries; it is not an automatic host-conversation continuity mapping. The existing12 tool names remain. Selected `record_signal` accepts `event_id` for stable receipts and optional `shared_space_id`; shared writes require current read_write membership. Changed payload under the same selected event conflicts. Actual model→native MCP→Garden pipelines qualified write/replay/search/expansion, unrelated-server preservation and revoke denial with PersonaSync off. No transcript capture, Recap policy or S09 native memory-provider completion follows.
