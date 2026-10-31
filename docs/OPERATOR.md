# Garden persona v1 — operator and developer boundary

## Not installed

This deliverable is source, a Python package, a passive native plugin bundle and a separately reviewed native SDK patch. No installer, source migration, live prompt capture or automatic real-data bootstrap has been executed. Do not point the running launcher at the development copy.

The SDK patch targets only its pinned base; its baseline and upstream license accompany it. Hosted plugin isolation is unsupported: retain that boundary rather than silently changing the profile's security model. Installation and real pilot remain separate approvals.

## Source and authority

Use an explicitly approved absolute shared root and home. Role directory is `personas/<persona_id>/`; there is no default-role fallback or sibling scan. Required documents: MISSION.MD, IDENTITY.MD, RELATIONSHIP.MD, REDLINE.MD. DREAM.MD and DARK.MD are optional and may be created by an approved replacement. Source bodies remain opaque plain text, including arbitrary human headings, lists and code fences.

MISSION must be nonblank and match `approved_mission_digest`. To change it, the authenticated owner must authorize the exact role/version/change, edit the source, approve the new pin and rebind deliberately. Editing MISSION alone is not a grant. The automatic path does not implement Mission approval or expose a model-facing administrative API. REDLINE is also automatically read-only.

Do not use generated SOUL as a second authoring source. Native edits create drift and halt adoption; no reverse import occurs. USER.MD/WORLD.MD/notes are not read by Garden. Optional `<home>/plugin-data/garden/HOST.md` is human-owned local assembly context, not a tool/admin grant and not helper-model input. All actual transport/tool/privacy permissions remain native host responsibilities.

First release supports one fixed binding, one runtime/writer, trusted local same-account editing, and POSIX file operations as exercised on macOS. Other OS/filesystem behavior, independent writers, hot persona switching and role-management UI are not certified. Keep permissions stable; do not make sources/world scopes writable to everyone.

## Explicit configuration

The native plugin ID is `garden-persona`; settings live under `plugins.entries.garden-persona.settings.persona`. Import, unsupported SDK and disabled configuration are passive.

A binding requires explicit `enabled`, `source_access_approved`, `writes_approved`, boolean `model_use_approved`, and `scope`. Scope fields are `shared_root`, `home`, `persona_id`, `binding_revision`, `consent_ref`, `surface`, approved Mission hash, adopted SOUL hash, source-byte/output-character budgets, optional `include_host`. Paths must be absolute; hashes are exact lowercase SHA-256 values. A grant reference is not itself an authentication credential.

All enabling/approval flags default to absent/false. Example (inert, deliberately not deployable):

```json
{
  "enabled": false,
  "source_access_approved": false,
  "writes_approved": false,
  "model_use_approved": false,
  "scope": {
    "shared_root": "/OWNER_APPROVED/SHARED",
    "home": "/OWNER_APPROVED/HOME",
    "persona_id": "songbai",
    "binding_revision": "OWNER_APPROVED_REVISION",
    "consent_ref": "OWNER_APPROVED_GRANT_REFERENCE",
    "surface": "cli",
    "approved_mission_digest": "OWNER_APPROVED_MISSION_SHA256",
    "managed_soul_digest": "OWNER_APPROVED_SOUL_SHA256",
    "max_source_bytes": 20000,
    "max_soul_chars": 10000,
    "include_host": false
  }
}
```

No provider/model is selected by this example. Model use additionally requires approved `provider` and `model` settings and matching native per-plugin LLM override grants/allowlists. Credentials remain host-owned. Use the native canonical configuration lifecycle only after installation/data authorization, not copied real secrets or body-supplied origin/grant JSON.

Only trusted `cli`/`local` evidence is eligible. QQ, groups, cron, API, unknown/remote surfaces are not enabled; no nickname/body-supplied owner identity is trusted. Configuration changes invalidate the old registration: stop/unload, inspect pending work and rebind. Old recovery cannot authorize a new binding; preserve/resolve it explicitly instead of deleting it automatically.

## Inference, costs and native triggers

Garden runs only when native memory/skill review is eligible and completed. It does not create another schedule or force a review on every user message. The helper facade gets a narrow approved visible batch and editable file bodies, not the native fork's full snapshot. Nonpersona results are routing hints; Garden never writes native facts/memory/skills. Independent deterministic synchronization of human-edited sources may still rebuild SOUL on a no-change review.

One logical helper invocation at most; empty/duplicate evidence does not call it. Request defaults are 60 seconds and 4096 output tokens; validation rejects invalid/excessive limits, and input context is capped. Native physical retries/fallback can multiply physical requests or elapsed time. Explicit route arguments still depend on native routing/fallback policy, not a cryptographic egress pin. Approve that policy, budgets and managed-local resource contention before real model use. Cancellation fences late adoption; it does not promise network abort, GPU release or a hard cost/wall-clock ceiling.

Authority-sensitive wording is conservatively escalated, but keyword triage is a heuristic, not semantic proof or a tool sandbox. Candidate text cannot edit configuration or supply its own grant. Model/user content remains untrusted.

## Commit, inspection and rollback

Complete source/SOUL bytes and private recovery materials are prepared before admission. At the short commit boundary, source/host/Mission/SOUL versions, binding and native cancellation lease are rechecked. At most one approved source is published, then SOUL, with bounded no-follow readback. Never call the pair ACID; a noncooperating editor can still race the final checks.

Private last recovery is `<home>/plugin-data/garden/last-apply/<persona_id>/APPLY.md`, with the referenced operation's `before/` and `after/` Markdown bodies under its UUID directory. This is the last required recovery material, not a database, event ledger, durable retry queue or long-term corpus. Normal cleanup preserves uncertain referenced material; interrupted filesystem cleanup may leave private temporary material for inspection.

- `persisted_verified`: exact expected source/SOUL bytes were read back; **not** evidence a session reloaded.
- `source_saved_host_pending`: a recorded durable source update with unchanged SOUL. `resume_host` may rebuild only the host after exact version/binding checks and fresh commit admission; no model/source rewrite.
- `unknown_outcome`, `observed_partial_unconfirmed`, `applied_bytes_unconfirmed`: inspection, not automatic retry/adoption. No model replay or overwrite of unresolved last recovery.
- `rollback`: trusted administrative Python entry only. Caller must authenticate fresh owner approval for the exact operation; an operation ID is not a credential. Both targets and all other assembly inputs must still match the expected after versions. Later edits are preserved, not force-overwritten.

Regular files only; no symlink/hardlink aliases or world-writable input files/scopes. FIFO opens are nonblocking and then refused. Backups are private; full SOUL/HOST/private bodies never enter public acceptance reports.

## Activation observation

`observe_identity` is a pure opt-in comparison helper, not an installed all-traffic observer or a refresh command. Supply transport-owned home/session/overlay metadata and actual outbound request data. Missing/active overlay, wrong home, duplicate/incomplete identity or unsupported wire form stays unverified. Only status/hashes leave the helper.

The exercised transport is normalized OpenAI-compatible chat. Other transport observers, including a live Codex/Anthropic route, are not certified by these fixtures. Observed bytes do not prove a model's behavior. Existing Agent caches and genuine DB/history resume may retain the old identity; a genuinely new session loads the assembled file. Reusing an ID without native history is not a genuine resume. No repair command, forced compression, DB surgery, active QQ reset or retroactive refresh is performed.

## Reproduce the development gate

Run `tools/verify.py --help`, then the README command with an isolated patched SDK and explicit Python environments. The verifier copies synthetic test assets into its scratch root, uses a whitelist environment and emits safe test counts. The alternative exists because the pinned native test wrapper hardcodes `/var/tmp`; do not silently describe this harness as that canonical wrapper.

Wheel verification additionally creates a fresh native source worktree at the base, applies the distributed patch, creates a separate environment, installs the wheel and reruns with `--installed`. Import origin must be inside that new environment; project-source shadowing is rejected. This is still synthetic acceptance, not real Songbai/private-provider deployment.
