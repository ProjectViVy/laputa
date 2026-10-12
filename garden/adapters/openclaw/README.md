# Garden OpenClaw adapter

One package supplies the opt-in native Persona hook. Native memory remains S09; no second plugin or memory-slot replacement is registered here.

Target: OpenClaw **2026.10.2**, source **b8952d477f0c932d8ddc6da5c89cab5a169b6ff1**, Linux x86_64 embedded runner. Build/unit checks alone do not qualify the native runner. Actual model-request qualification is recorded in [the experiment](../../../docs/research/garden-openclaw-native-qualification-2026-10-12.md); [the authoritative index](../../../docs/superpowers/plans/garden-standalone-v1/index.md) owns status.

```sh
npm ci --ignore-scripts
npm test
openclaw plugins install --link --no-enable /absolute/path/to/this/package
```

`--no-enable` leaves a new registration inactive and preserves existing enablement, allowlists and denylists. Inspect the existing entry before an update; this flag does not disable an already-enabled plugin. Passive package discovery, disabled entries, CLI metadata, setup and tool scans perform no request or credential read. The pinned runtime uses both full and scoped discovery registries for explicitly enabled plugins; these capture a lazy callback, and native permissions still govern its registration. Activation requires the owner's existing native configuration, explicit `plugins enable garden`, `plugins.entries.garden.hooks.allowConversationAccess=true`, prompt injection allowed, and a separately enabled Garden PersonaSync binding. Installer confirmation options are host-native; never overwrite an existing package without owner authorization.

Configure `plugins.entries.garden.config` with `endpoint` (loopback HTTP Garden origin), `bindingID`, `credentialFile`, `hostInstance`, `agentID`, and the existing `workspaceDir`. Keep the credential in an owner-only directory/file (0700/0600 on the qualified platform), never in instructions or plugin JSON. The owner configures the matching native scope in Garden as SHA-256 hex of agent ID, newline, canonical workspace; previews while off, enables PersonaSync explicitly, then confirms that preview. Recall/Capture remain independent.

Native SDK registration uses `openclaw/plugin-sdk/plugin-entry`. Only `before_prompt_build` and `appendSystemContext` are used. Runtime version comes from `api.runtime.version`. The current native agent/workspace/session are checked; the public SDK's current model/runtime policy and session metadata restrict contribution to the embedded OpenClaw harness. Subagents, parent-owned sessions, other scopes and unqualified harnesses omit contribution. The adapter ignores prompt/messages, reads the credential privately at invocation, sends only the bounded native request to Garden, and never writes SOUL, AGENTS, native profile/history or memory files.

HTTP timeout is one second. Disabled/locked/offline/revoked responses omit new Garden content with a bounded diagnostic; there is no cached-body fallback. An edit block leaves durable pending in Garden. Save/cancel does not clear it: the owner reads a new preview, confirms it, then the next eligible host prompt pulls the current matching bytes. Already admitted/history context may remain. Garden's offered status is weaker than loaded; loaded remains unknown without a qualified observer.

Pinned CLI help identifies actual embedded invocation:

```sh
openclaw agent --local --agent main --session-key agent:main:main --message 'synthetic qualification' --json --timeout 30
```

Qualification runs use an isolated `OPENCLAW_STATE_DIR` and `OPENCLAW_CONFIG_PATH`, a synthetic local provider/input capture, and the same native profile/workspace/history across turns. Never apply that fixture configuration to an existing user home. Actual native qualification must show N→N+1 on the next prompt, unchanged instruction files, preserved base prompt, lock/pending/fresh retry, permissions, offline/revoke and mismatched scopes before support is marked complete.
