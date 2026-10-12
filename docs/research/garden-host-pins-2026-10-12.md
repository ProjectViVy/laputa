# Garden host automation — pinned source findings

**Date:** 2026-10-12. **Status:** source-qualified architecture input, not installed-host conformance.
**Laputa baseline:** 3c4cfffe33d92ad1105ad31dec259c901cc24a4e.
**Authority:** [current specification §7](../superpowers/specs/2026-10-12-garden-standalone-v1-design.md#7-controlled-editing-and-automatic-host-projection).
**Prior report:** [controlled-editing research](garden-standalone-v1-controlled-editing-research-2026-10-12.md), preserved as its earlier evidence snapshot.

## Corrections to the earlier research

The earlier report found no documented same-profile Hermes persona hook and suggested separate profiles/workspaces as possible alternatives. Pinned source now establishes native additive hooks in Hermes and Codex. Those alternatives are **not selected**; this design preserves the user's existing profile, history, workspace and instruction discovery. Only explicit plugin activation is required. No user host was modified.

OpenClaw's expectedHash file RPC is best-effort against independent shell/editor writes. A native RPC name does not establish filesystem CAS. The earlier specification's file-precondition wording was too strong and is superseded.

Hermes documentation says pre_llm_call context is ephemeral, but this pin's request assembly also replays api_content sidecars. Do not promise that past injected text disappears from history. Current delivery is a new versioned contribution; already admitted text remains subject to the host lifecycle.

## Exact pins and source evidence

| Host | Selected qualification target | Evidence and usable seam |
| --- | --- | --- |
| OpenClaw | tag v2026.10.2, commit b8952d477f0c932d8ddc6da5c89cab5a169b6ff1; package version 2026.10.2 | [hook result types](https://github.com/openclaw/openclaw/blob/b8952d477f0c932d8ddc6da5c89cab5a169b6ff1/src/plugins/hook-before-agent-start.types.ts#L20-L48), [context identities](https://github.com/openclaw/openclaw/blob/b8952d477f0c932d8ddc6da5c89cab5a169b6ff1/src/plugins/hook-agent-context.types.ts#L26-L70), [runner applies additive system context](https://github.com/openclaw/openclaw/blob/b8952d477f0c932d8ddc6da5c89cab5a169b6ff1/src/agents/embedded-agent-runner/run/attempt-prompt-build.ts#L280-L315). api.on("before_prompt_build", handler) returns appendSystemContext; do not return systemPrompt (whole prompt replacement). Qualify the embedded runner first, not all OpenClaw harnesses. |
| Hermes | tag v0.21.7, commit 71d34bad1045610252eda02ee79549436896e784 | [plugin registration and hooks](https://github.com/NousResearch/hermes-agent/blob/71d34bad1045610252eda02ee79549436896e784/hermes_cli/plugins.py#L934-L980), [actual per-turn collection](https://github.com/NousResearch/hermes-agent/blob/71d34bad1045610252eda02ee79549436896e784/agent/turn_context.py#L793-L848), [request-copy injection and historical sidecar replay](https://github.com/NousResearch/hermes-agent/blob/71d34bad1045610252eda02ee79549436896e784/agent/turn_context.py#L1250-L1305). ctx.register_hook("pre_llm_call", callback); return {"context": markdown}. Fires once per persistent turn; _persist_disabled paths skip it. This is user-context priority, not SOUL replacement. |
| Codex CLI | tag rust-v0.162.1, commit 092d3acd6bec3e3a14bdc7e7a2810ab628ab759d | [UserPromptSubmit input/output implementation](https://github.com/openai/codex/blob/092d3acd6bec3e3a14bdc7e7a2810ab628ab759d/codex-rs/hooks/src/events/user_prompt_submit.rs#L23-L141), [hook discovery](https://github.com/openai/codex/blob/092d3acd6bec3e3a14bdc7e7a2810ab628ab759d/codex-rs/hooks/src/engine/discovery.rs#L122-L155), [plugin hook sources](https://github.com/openai/codex/blob/092d3acd6bec3e3a14bdc7e7a2810ab628ab759d/codex-rs/hooks/src/declarations.rs). Native UserPromptSubmit/SessionStart output supplies additionalContext; [official semantics](https://developers.openai.com/codex/hooks) identify developer context. It preserves AGENTS.md discovery; no nested workspace is needed. |
| Claude Code | release v2.1.296, public release commit 2301018b1f61073c501a8e7a4813ef48c239163b; binary qualification pending | [immutable release](https://github.com/anthropics/claude-code/releases/tag/v2.1.296), [official hooks contract](https://code.claude.com/docs/en/hooks), [plugin hook packaging](https://code.claude.com/docs/en/plugins-reference). UserPromptSubmit/SessionStart command hooks can return hookSpecificOutput with hookEventName and additionalContext. This is additive conversation context. Closed runtime source prevents static proof that live docs exactly match that binary; S10 must qualify the pinned binary before marking supported. |

Tags were resolved with git ls-remote; OpenClaw, Hermes and Codex were shallow-cloned into task scratch directories and their commits inspected. No product dependencies, model calls, real-host installation or host-runtime tests occurred in this planning session. These pins are exact initial targets, not a claim that every newer version is compatible.

## Native activation and limits

OpenClaw's non-bundled prompt hooks require explicit allowConversationAccess and prompt injection must not be disabled: [pinned hook permissions](https://github.com/openclaw/openclaw/blob/b8952d477f0c932d8ddc6da5c89cab5a169b6ff1/docs/plugins/hooks.md#L118-L145). Activation is an owner-reviewed host-native operation. A denied permission means unavailable, not a fallback writer.

Hermes exposes register_system_prompt_section as well, but its [frozen/rebuild implementation](https://github.com/NousResearch/hermes-agent/blob/71d34bad1045610252eda02ee79549436896e784/agent/system_prompt.py#L113-L135) and [invalidation](https://github.com/NousResearch/hermes-agent/blob/71d34bad1045610252eda02ee79549436896e784/agent/system_prompt.py#L811-L830) have session/cache semantics. The selected per-turn hook avoids pretending the native SOUL or cached system prompt was edited. Do not install both a frozen Garden section and a changing Garden per-turn copy. MemoryProvider remains a separate recall/capture concern.

Both CLIs use an inert Garden plugin package with its own hooks/hooks.json and one small shared Garden command bridge. The user activates it in the same existing host scope. Garden does not merge or replace existing shared settings files. Exact plugin-loader installation syntax belongs to S10's pinned qualification task; until verified, that activation step remains unavailable. Missing hook capability is visible; explicit MCP remains usable but never counts as automatic Persona delivery.

## File alternatives and the R4 boundary

[OpenClaw agents.files.set](https://github.com/openclaw/openclaw/blob/b8952d477f0c932d8ddc6da5c89cab5a169b6ff1/src/gateway/server-methods/agents-files.ts#L380-L489) checks expectedHash in its own queue and then writes. Its source explicitly says shell writers remain independent. This is not external-editor CAS; do not select it for shared SOUL updates.

[OpenClaw config RPC](https://docs.openclaw.ai/gateway/configuration/config-rpc) exposes baseHash for JSON configuration, while [config include](https://docs.openclaw.ai/gateway/config-secrets-env) isolates JSON objects. Neither proves atomic conditional replacement of Markdown against VS Code. They are possible one-time native activation mechanisms, not the runtime Persona sync path.

[Claude memory files](https://code.claude.com/docs/en/memory) support a dedicated rules file and @path imports. [Codex AGENTS discovery](https://developers.openai.com/codex/guides/agents-md) is hierarchical and does not provide an additive sibling-file include. The selected native hooks avoid both limitations and do not narrow Codex's directory scope.

An explicitly Garden-owned derived file can be atomically replaced in the trusted local single-writer support model, provided no other writer owns that path. That is a supported ownership contract, **not a lock against VS Code**. Detected drift stops further writes; an undetected concurrent edit can still race. Because native hooks meet the present requirement without this ambiguity, this increment does not add a second derived-file synchronizer. Keep S08 no-clobber/manual behavior for external shared files and separate cleanup.

## Qualification still required

- OpenClaw: pinned embedded runner emits latest versioned Garden block on the next invocation, preserves native prompt, honors denied hooks/revoke and records unknown loaded without a qualified observer.
- Hermes: pinned persistent CLI/gateway turn includes current additive block without changing SOUL/profile/history; ephemeral/worker paths are denied unless explicitly qualified; retained history is not represented as erased.
- Claude/Codex: exact binary/plugin loader and command output semantics in isolated homes, same workspace, unchanged native instruction files, new turn/restart/compact behavior. No user transcript scraping.
- Reuse a bounded prompt-capture fixture or official host runner where available. An echoing model answer alone is not evidence of the exact prompt; a stub that only tests our callback is not host conformance.

## Subsequent installed-host evidence

The source-finding snapshot above is preserved. Later owner-authorized exact Linux runtime qualification is recorded separately: [OpenClaw embedded CLI](garden-openclaw-native-qualification-2026-10-12.md), [Hermes persistent CLI](garden-hermes-native-qualification-2026-10-12.md), and [Codex/Claude native CLI](garden-cli-native-qualification-2026-10-12.md). These reports scope actual Persona paths and separate explicit memory evidence; they do not qualify every frontend/build, automatic memory lifecycle or S09 native memory mappings. The [index](../superpowers/plans/garden-standalone-v1/index.md) alone owns current Story disposition.
