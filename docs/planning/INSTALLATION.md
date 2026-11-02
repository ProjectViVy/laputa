# Installation record (public-safe)

Scope: one owner-controlled macOS `default` Hermes profile. This document records what was
installed and what was deliberately left disabled. It contains no persona text, no profile
contents, no credentials and no absolute private paths.

## What was installed

| Item | Detail |
|---|---|
| Host SDK capability | `background_review_persona_stage_version = 1` |
| Pinned host base | `bf0203a794a07aac3ff764972ab9087baa4e4f4e` |
| Host local commit | `d9b0f08349bae0e0554113b502798063bb5f7dd2`, local branch only, not pushed by the installer |
| Host tree check | Staged tree equal to the verified extension tree (`e8236cc2…^{tree}`); nothing else staged |
| Plugin | `garden-persona` 0.1.0, `manifest_version: 2`, `api_version: 1`, standalone |
| Plugin layout | Self-contained directory bundle: unchanged verified package files plus a relative-import entry adapter; no runtime dependencies added |
| Enablement | Hermes plugin enabled; `settings.persona.enabled` **false** |
| Consent flags | `source_access_approved`, `writes_approved`, `model_use_approved` all **false** |
| Isolation | Unchanged native in-process host; no policy relaxation |

## Evidence collected

- Native directory loader parsed the manifest, imported the namespaced module and registered it;
  `hermes plugins doctor --ci` reported discovery, parsing, import and registration OK, with zero
  tools and zero hooks.
- The plugin was exercised from the installed bundle against a synthetic home: a local-surface
  turn produced a deterministic assembled SOUL and a private last-apply receipt; a `qqbot`-surface
  turn was refused; no network connection was attempted; unload removed the registration.
- Source and host regression on the installed tree: 87 component and 218 host cases, 0 failed,
  0 skipped. Fresh SDK source plus patch plus installed wheel reproduced the same counts.
- After the owner restarted the gateway from an external shell, a new process was observed with
  the messaging channel connected. The plugin still loaded, consent flags still read false, and
  both the existing `SOUL.md` digest and unrelated configuration were unchanged.
- A private, permission-restricted backup of the pre-install configuration, `SOUL.md` and the
  affected host files was taken and digest-verified before any write.

## What this record does not claim

- No real role documents exist; `MISSION`/`IDENTITY`/`RELATIONSHIP`/`REDLINE` were not created or
  migrated, and no Mission version was pinned.
- No helper model route was approved and no real provider call was made.
- No automatic source or SOUL write is possible with the recorded configuration.
- No identity change was observed in a live session; a stored-file write would still not prove a
  session reloaded it.
- The plugin runs inside the Hermes process; that is not a sandbox, and the conservative
  authority-wording checks are heuristics, not semantic guarantees.
- Source and SOUL publications are separate atomic replacements, not a two-file transaction; a
  non-cooperating external editor can still race the final drift checks.

## Enabling evolution later

Required, in order and each as its own decision:

1. Approve a role directory `personas/<persona_id>/` under an absolute shared root, with mission
   text authored by the owner.
2. Create the role documents, then approve the exact `MISSION` digest and the managed `SOUL`
   digest as the initial binding.
3. Approve source access and writes.
4. Approve a helper `provider`/`model`, its egress scope and budgets.
5. Rebind after enabling; verify write, readback, requested activation and effective use as
   separate outcomes.

Remote surfaces (messaging platforms, groups, cron, API) are not eligible evidence in this design;
supporting them requires its own trusted-identity and data-boundary acceptance.

## Rollback boundary

Disable the plugin with the native command first; then confirm the recorded host commit, backup
digests and whether later edits exist before deciding anything about the host commit. Do not
hard-reset the repository, overwrite a configuration that has changed since the backup, or delete
session stores, memory data or the private recovery directory.
