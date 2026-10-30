# ADR-0016: Laputa Protocol and Native Agent Governance

**Status:** accepted direction; detailed protocol and implementation pending
**Date:** 2026-10-31
**Decision owner:** project owner
**Refines:** ADR-0012, ADR-0013, ADR-0014 and ADR-0015
**Precedence:** this decision controls the new cross-agent architecture. Earlier
ADRs remain contracts for the existing Go/Garden implementation where compatible;
their storage, context, review and transport choices are not universal adoption
requirements.

## 1. Intent

Laputa becomes a specification/protocol for persona and memory governance, with
small implementations adapted to the consuming agent. Garden becomes a reusable
governance workflow and an optional management application. Apply More, Fast,
Good, Frugal by using the agent's existing authority files, memory engine,
working-context engine and evolution facilities.

The Laputa-Garden separation remains useful. Requiring every agent to run the
complete Go stack or establish another persona/memory authority does not. The
new Vivy already has a full Python-native MemPalace integration; its Laputa
integration should concentrate on persona. This records direction, not a rewrite.

## 2. Responsibilities

| Part | New responsibility | Boundary |
| --- | --- | --- |
| Laputa specification/protocol | Define persona ownership, reads/updates, context delivery and relations to memory/Recap providers | Independent of one language, file layout, server or transport; exact schemas remain open. |
| Host persona implementation | Maintain one authoritative source and supply persona context | Prefer native files or supported APIs; do not create a competing Laputa copy. |
| Long-term memory engine | Own canonical content, retrieval and supported corrections/deletions | Backend is replaceable; engine-specific storage and recovery remain internal. |
| Agent/harness Recap engine | Own working context, compaction, persistence and context admission | Reuse native behavior; no mandatory Garden ACTMEM engine/file. |
| Garden | Collect material, invoke a selected agent, apply updates and verify effective results | Optional reusable governance workflow; reuse native evolution where it exists. |
| Host runtime | Own model calls, execution, tools, cancellation, sessions/tasks and prompt assembly | The protocol and Garden do not become a second agent runtime. |

Laputa data belongs in the persona layer of model context. In the new Vivy this
is the root persona block in application-level top-level prompt composition,
preferably supplied by a small Rust component. Other hosts need not adopt Rust.
Dedicated workflows, including Vivy's persona-free Dream prompt, retain their
selected context policy.

Native file names, fields, body limits and refresh boundaries are adapter/profile
choices. Seven Markdown files and Frozen Core remain a valid existing Go profile,
not a schema imposed on every host. This does not revive retired Go JSON
authority data, descriptor migration or fallback paths.

## 3. Memory and Recap implementations

- **Vivy now:** retain full MemPalace through the existing Python workbench.
  MemPalace L0 does not become another persona source.
- **Vivy later:** permit another long-term backend, such as Mem0, without
  rewriting persona governance or host lifecycle. No Mem0 adapter, automatic
  interchange or data migration is selected here.
- **Existing Garden:** Mentle remains an existing backend. ADR-0014 still controls
  canonical writes and recovery inside Mentle; it does not make Mentle SQLite
  mandatory for every Laputa consumer.
- **Recap:** the host owns its implementation and context policy. Expose applicable
  operations without copying Recap into a parallel Garden file/checkpoint store.

Choose the smallest backend interface needed by actual consumers. Capabilities
need not be identical; report unsupported behavior instead of silently equating
search/update/deletion semantics. A corpus/evolution workspace holds source
materials and outputs without becoming another canonical long-term memory engine.

## 4. Native authority first

Hermes and OpenClaw are owner-named integration examples. Their exact current
file layouts, write APIs, evolution hooks and reload behavior require adapter
research; no compatibility is certified by this ADR.

Garden should update a host's actual persona/governance files through its
supported write boundary. Do not require another Laputa directory and synchronize
competing copies. Reuse or coordinate an existing native evolution path rather
than adding an independent writer/scheduler over the same material.

Write policy belongs to the host/profile. The existing Garden Persona review
policy is not universal. Existing protected inputs retain their selected policy.
Vivy AutoDream retains its accepted automatic adoption without routine human
approval of each candidate.

## 5. Required closed loop

```text
collect corpus/evidence into an explicit workspace
    -> invoke a selected agent or native evolution path
    -> produce candidate updates
    -> check and apply through the owning host's write policy
    -> verify persisted changes and their effective context boundary
    -> retain outcomes and useful evidence for a later run
```

Targets may include native persona files, an existing memory backend or another
explicitly supported host artifact. Candidate generation, persisted application
and effective use are separate outcomes. A successful file write alone does not
prove that a running agent reloaded or used a new persona revision.

Detailed design must settle collection scope, invocation/cancellation, no-change
outcomes, conflicts, source association, duplicate work, failures/retry,
history/rollback and activation boundaries. Reuse host mechanisms where available.
These needs do not select a universal workflow engine, approval platform,
scheduler, SDK or conformance platform.

Vivy's AutoDream may implement its own closed loop; Garden is not a deployment
prerequisite. Extract shared code only when actual consumers justify it.

## 6. Management and transport

A unified Garden Web console and management entry point are possible later
extensions. Managing Vivy's internal Laputa from Garden is explicitly left for
further discussion. No remote-management transport or network exposure is selected.

ADR-0015's `laputa-agent/1`, canonical Garden REST, MCP facade and Go SDK remain
an earlier service-specific access profile and implementation record. They are
not mandatory transports or lifecycle machinery for the new Vivy or native-file
adapters. Read their plans with this refinement before reusing their assumptions.

## 7. Keep, Cut, Deferred, Drop

| Classification | Decision |
| --- | --- |
| Keep | Persona/memory/working-context separation; readable authority; host write policies; useful history/evidence; collection-to-effective-update loop; existing Go behavior within its profile. |
| Cut | Mandatory independent Laputa library/server, universal seven-file/ACTMEM layout, hardwired memory/Recap engine and REST-only adoption assumptions. |
| Deferred | Exact protocol/schema, minimal Rust component, native adapters, backend interfaces, workflow packaging, Web console and Garden control of Vivy's internal Laputa. |
| Drop | Parallel persona/Recap authorities solely for integration; full-memory mirroring; unnecessary replacement of native evolution; unverified backend-parity/compatibility claims. |

## 8. Scope and evidence

This decision authorizes documentation synchronization. It does not change code,
migrate personal data, install adapters, replace Mentle or certify host
compatibility. Existing Go gates remain evidence for that implementation, not
acceptance of this architecture. Other worktrees' source changes remain outside
this record.

The consuming experiment is `C:/Users/Administrator/Desktop/vivy-test`, whose
`docs/DESIGN.md` records the same direction. Recipe-based Rust composition,
deferred real-provider acceptance, durable sequencing and the later inbox
schedule remain unchanged. No original Vivy checkout or private data is needed.
