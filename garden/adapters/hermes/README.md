# Garden Hermes adapter

One opt-in standalone plugin supplies Persona at persistent native turn boundaries. S09 extends this same package for native recall/capture; this package registers no memory provider or frozen system section.

Target: Hermes **0.21.7**, source **71d34bad1045610252eda02ee79549436896e784**, Linux x86_64. Source and unit checks alone do not qualify the native runner; the [authoritative index](../../../docs/superpowers/plans/garden-standalone-v1/index.md) owns acceptance status. The adapter uses only the Python standard library.

```sh
python3 -m unittest discover -s garden/adapters/hermes -p 'test_*.py'
```

Install the directory as the owner's native `garden` plugin in the existing selected Hermes home, initially disabled. Import/discovery is inert; native standalone plugins are opt-in. Use the pinned native plugin activation commands after inspecting actual help, rather than merging a new profile or overwriting config. Installation and qualification commands are recorded only after native execution.

Native `plugins.entries.garden.settings` contains `endpoint` (loopback HTTP Garden origin), `binding_id`, `credential_file`, `host_instance`, and the canonical existing `hermes_home`. Native secret file references are private 0700 directory/0600 regular file; never put the credential in settings, SOUL, messages or diagnostics. The configured home must match the native manager's registration scope. Runtime version comes from `hermes_cli.version_info.get_version_info()`, not a settings claim. Garden native scope is SHA-256 of the canonical selected home. Preview while PersonaSync is off, enable it separately, then explicitly confirm a fresh preview. Recall/Capture stay independent.

The exact native `pre_llm_call` callback accepts current/future keyword metadata and sends only adapter/version/instance/session/scope plus authentication. It ignores user_message and conversation_history. Registration captures the approved home; later ambient HERMES_HOME changes cannot retarget the contribution. Parent/worker/empty sessions are refused; the pinned native turn assembler skips `_persist_disabled` before invoking plugins. Credential reading and a single direct loopback HTTP request occur only at invocation. Environment proxies are bypassed; numeric loopback candidates avoid name-service waits. Connection, headers and body share one one-second deadline; the owned socket closes synchronously with no background request. Redirects, oversize/mismatched output, locked/revoked/unavailable responses omit new context with static diagnostics; there is no cached-body fallback or file writer.

The returned `context` joins the current user message, below the native SOUL/system priority. Original SOUL/profile/history remains authoritative to the host. Native persisted `api_content` can replay an older contribution; disabling or revoking prevents new contributions and cannot erase prior history. Edit blocks retain Garden pending after Save/Cancel until the owner previews/confirms a fresh retry. Status is offered only; written is inapplicable and loaded stays unknown without a qualified observer.

Native acceptance must capture actual persistent CLI/gateway model requests for N→human Save N+1, guarded edit/pending/fresh retry, offline/revoke and excluded ephemeral/worker paths in the same profile/history. It must preserve existing SOUL/configuration and report the user-context priority and retained history. Unit callback success is insufficient.
