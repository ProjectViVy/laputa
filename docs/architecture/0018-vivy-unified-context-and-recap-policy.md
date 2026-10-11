# ADR-0018: VIVY Unified Context and Subject Recap Policy

**Status:** accepted owner direction; implementation and engineering details pending
**Date:** 2026-10-11
**Decision owner:** project owner
**Source:** [VIVY issue #58: Unified context pipeline: memory enhancement, Recap-only ACTMEM and editable UI](https://github.com/ProjectViVy/agent-vivy/issues/58), owner clarification in the documentation reconciliation session, and [issue #57: role-only workers](https://github.com/ProjectViVy/agent-vivy/issues/57).
**Refines:** ADR-0012, ADR-0013, ADR-0015, ADR-0016 and ADR-0017 only where their ACTMEM/context-policy assumptions conflict with this decision. ADR-0014 remains the Mentle authority/recovery contract.
**Delivery:** documentation only. No runtime, API, schema, migration or backend switch is delivered by this ADR.

## Decision and ownership

The unified context pipeline is a VIVY foundation. LIFE is its first default-enabled pilot; coding is excluded from phase 1. Implement through the existing runtime, context assembly, Journal and context-policy abstraction. Do not create a LIFE-specific pipeline, second compaction flow, duplicate active memory authority, new Journal, or parallel scheduler.

VIVY determines authorized query scope, candidate admission, context budget and policy selection. A selected backend (BML, Mentle, or another supported provider) owns concrete CRUD/search, canonical storage, indexing and the content it returns. Garden/Laputa retain their existing governed operations, authority boundaries and shared adapter contracts. A backend result is a candidate subject to VIVY admission; neither a backend profile nor storage format defines runtime identity, persona, access permissions or context priority.

BML is the first validation backend for this policy. Its completion capture is a bounded completion projection, not the authoritative raw conversation transcript; retained raw evidence stays with its existing lifecycle. The owner's Mentle-backed DIVA path remains required through the same capability-aware abstract interface. This does not authorize switching an existing default, dual-writing, moving canonical data, publishing a migration, or pretending unsupported provider capabilities exist. Within Mentle, canonical SQLite, transactional receipts and `index_jobs` remain authoritative; vectors/BM25/WAL cannot reconstruct or override canonical memory.

## One persistent subject Recap

All persistent sessions of one VIVY subject share one subject-level ACTMEM Recap. Its active content is free-form, AI-organized prose; it replaces Pulse/Recap/Work as the selected policy's target active shape. It is not one Recap per session, a transcript, a backend record, a persona file, or a second compact checkpoint. Revision/provenance/access metadata must not become a second independently writable authority.

Subject continuity does not grant universal disclosure. The host must enforce authorization before evidence enters consolidation and before a Recap projection reaches a model. How a subject-level free-form Recap preserves restricted-workspace information safely is an engineering question to resolve before implementation; it must not be answered by silently inheriting unrestricted cross-workspace access.

Issue #57 role-only workers do not automatically inherit the subject's persona, Recap, history or permissions. Admission is explicit and bound to the existing host policy. Worker delegation cannot manufacture authority through a role label, shared subject identifier, file path or model-generated permission field.

## Consolidation and context retirement

The next Recap is organized from the previous committed Recap plus the retiring raw conversation segment. Commit the next Recap with revision-aware compare/conflict handling before unloading that segment from raw CONTEXT. Unloading removes raw text from the active prompt; raw evidence remains retained in its existing Journal/backend/archive location. A failed or conflicting update must not retire the only active representation of the segment.

Concurrent session consolidation and authenticated human edits use the same authority and revision-aware mutation boundary. A stale writer cannot overwrite a newer edit or another consolidation. Resolve an explicit conflict against the new revision; blind last-write-wins and replay of stale generated content are prohibited. This ADR does not choose locking, storage grammar, merge implementation or cross-process support.

The selected policy may admit a bounded Recap projection into dynamic context through the existing policy path. It does not add ACTMEM to Frozen Core or permit automatic WORLD loading. Context thresholds, Recap sizes, reserved budgets and refresh cadence are experimental and must be tuned with evidence; old 1600/200/1200/800 limits and fixed fold timers are historical baseline values, not phase-1 requirements.

## Human control and targeted deletion

Provide a dedicated UI where the authenticated human can inspect the full active Recap, edit it and delete it. Full human inspection is distinct from any bounded model projection. Human edits must carry the current revision and surface conflicts; consolidation cannot silently replace them.

Explicit deletion of active ACTMEM through this UI clears the active Recap, commits a new revision and invalidates active projections/caches that refer to the prior revision. This is a targeted operation. Retained raw evidence, older revisions, capsules/archives, Journal history, BML records and Mentle canonical records remain intact unless separately targeted by an explicitly authorized delete tool. No cascade erasure, automatic source deletion, or implied cleanup of every historical occurrence is authorized.

Retained data does not authorize immediately restoring the deleted active head from an old cache or archive. Whether later authorized consolidation may use retained raw evidence is a policy decision to resolve explicitly; a targeted clear must be observable and effective in the active projection lifecycle.

## Agent tools and maintenance boundaries

The agent needs ACTMEM inspect/update/delete tools. They are normally hidden and become available through deferred discovery and explicit activation when needed. Do not install always-on schemas or treat this as authorization for autonomous recurring maintenance. The exact deferred-discovery/activation mechanism and API mapping remain unresolved.

Current Garden `agentapi` ACTMEM methods and authenticated human control actions do not prove that VIVY has model-visible ACTMEM tools. Existing read/query/Work mutation methods and REST shapes are implementation baseline surfaces. This documentation change creates no new endpoint and claims no Recap tool integration has shipped.

## Frozen Core and “must not be forgotten”

A human asking for something that must not be forgotten should be guided to the existing Laputa governed authority workflow appropriate to that content. Use its established human edit/review/history and revision controls. Ordinary Recap editing is not permanent pinning and cannot create a new authority slot or permission grant.

The six established Persona projections (Identity, Relationship, Redline, User, Dream, Dark) remain bounded and frozen for an admitted session. The existing cognitive v2 Mission slot, recorded in ADR-0017 and its contracts, is human-controlled orientation; this decision neither adds a slot nor changes Mission permissions. A later source edit does not mutate an admitted Frozen Core. New sessions capture applicable revisions; existing mission revision admission/effect fences remain in force. WORLD stays tool-only.

Summary/consolidation agents cannot automatically write Mission, Dream or frozen Persona content. Retain the existing distinction between authenticated human edits, reviewed Persona proposals and explicitly authorized conversational-agent tools; consolidation must not disguise itself as the latter. Frozen Core is a session-frozen authority projection, not permanent raw evidence storage.

## Compatibility classification

| Classification | Scope |
| --- | --- |
| Keep | Existing runtime/context/Journal and policy abstraction; governed Laputa authority; Mentle canonical/outbox recovery; authorization, provenance, receipts, UI human controls; WORLD tool-only; supported DIVA memory path |
| Supersede for selected policy | Absolute ACTMEM no-auto-context rule; Pulse/Recap/Work active schema; fixed historical caps/fold cadence; always-visible ACTMEM read-tool assumption |
| Plan compatibility | Existing Work reconcile, Pulse/Recap capture, capsule fold, evolution consumers, REST/MCP/SDK/desktop DTOs and tests; inventory callers before changing any interface |
| Deferred | Free-form Recap storage/metadata, conflict/merge semantics, scope-safe projections, deferred tool discovery, projection invalidation and clear/reconsolidation lifecycle, thresholds and model/provider experiments |
| Drop | Separate LIFE memory pipeline; duplicate ACTMEM/compaction authority; unrestricted worker inheritance; automatic Mission/Dream/frozen Persona writes; permanent pinning through Recap; cascade erasure; default-backend switch, dual write or migration without separate authorization |

Historical accepted ADRs, frozen plans and pass/fail records remain evidence of their snapshots. Their checks do not establish implementation of this policy. Read the [alignment and evidence inventory](../dev/memory-policy-alignment-2026-10-11.md) for current proof and unresolved acceptance gates.
