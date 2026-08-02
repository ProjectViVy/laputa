# ADR-0002: Laputa Cognitive Partition and Report-System Decision

> **Status:** accepted
> **Date:** 2026-08-02
> **Decision owner:** project owner
> **Supersedes:** the active-plan LTM / `LONGMEM.MD` lifecycle model; the intended business semantics of legacy sections `06-history_md`, `10-journal_reflective` through `14-aaak_summaries`.
> **Does not yet change:** Go registry names, persisted legacy JSON files, runtime APIs, or host adapters. Those require a separately approved migration plan.

---

## 1. Context

The historical 14-section registry mixed five different concerns in one flat list:

- frozen personality and user-bound constraints;
- short-term working state;
- a duplicate long-term-memory concept;
- human-facing periodic reports;
- future governance queues, logs, indexes and compression artifacts.

That flat registry encouraged two incorrect directions:

1. treating Laputa as a second long-term material store beside Mentle; and
2. treating internal implementation artifacts as ordinary authority/content files.

The MemoryOS has now established the following durable boundary:

```text
Mentle = Material Universe / Evidence Lake
         raw sessions, documents, code, images, video, audio, historical
         versions, rejected ideas and retrievable source evidence.

Laputa = Agent cognitive governance and stable operating surfaces.

Garden = ingestion, policy-controlled recall and disposable ContextView
         assembly for a particular host, task and budget.
```

Therefore a separate LTM body or `LONGMEM.MD` authority index is unnecessary. Mentle preserves all material; no second long-term store may duplicate it.

---

## 2. Decision

Laputa is partitioned by cognitive role rather than by the legacy flat 14-section numbering.

```text
A. Frozen Core Prompt
   01 identity
   02 relationship
   03 commitment
   04 preferences

B. Active Work
   05 MEMORY.MD / STM checkpoint

C. Cognitive Governance (new named files; initial human-operated)
   MEMRULES.MD
   WORLD.MD

D. Human-facing Report System
   07 daily
   08 weekly
   09 monthly
   10 AMBITION
   11 USER SUGGESTIONS

E. Infrastructure, not Laputa content files
   append-only change/audit log
   EvoMap mailbox (separate future design)
   Laputa transient spool (Mentle-outage raw-event outbox)

F. Removed conceptual sections
   06 history_md / LTM
   13 report_indexes
   14 aaak_summaries
```

The legacy Go implementation still has 14 registry constants. It is an implementation compatibility surface, **not** the target cognitive architecture. No new feature may use removed section semantics merely because their constants still exist.

---

## 3. Partition responsibilities

### 3.1 A — Frozen Core Prompt: 01–04

These files establish how the Agent must act, not what historical material happens to be retrievable.

| Section | Responsibility | Context rule | Change rule |
|---|---|---|---|
| `01 identity` | Agent identity, role, voice and stable boundaries | session bootstrap; frozen for that session | governed revision; later session takes effect |
| `02 relationship` | stable agent–user collaboration relationship | session bootstrap; frozen for that session | governed revision |
| `03 commitment` | user red lines and non-negotiable constraints | session bootstrap; frozen for that session | user-controlled |
| `04 preferences` | confirmed stable collaboration preferences | session bootstrap; frozen for that session | governed update; do not infer a transient preference as stable |

The core must stay compact. It is not a user dossier, material corpus or dynamic recall result.

### 3.2 B — Active Work: 05 `MEMORY.MD`

`MEMORY.MD` is the STM authority checkpoint and working projection:

```text
current goal / active scope / constraints / decisions in flight
open loops / next actions / material and evidence references
```

It is not raw source storage, a report or LTM. Agent edits remain lightweight: new projection revision, `base_revision`, `updated_at`, actor and atomic write result. Historical source material remains append-only in Mentle or, during outage, in the transient spool.

### 3.3 C — Cognitive Governance: `MEMRULES.MD` and `WORLD.MD`

#### `MEMRULES.MD`

`MEMRULES.MD` defines how Garden and Laputa form, use, revise and retire memory/world understanding. It is the cognitive-governance rulebook, not a memory payload.

It must cover, at minimum:

```text
- Mentle raw material and evidence remain primary.
- Confirmed fact, observation, inference and hypothesis are distinct.
- New contradictory evidence does not silently overwrite prior understanding.
- User-confirmed information outranks agent inference.
- Scope, time, confidence, provenance and visibility constrain use.
- Entry into WORLD requires action relevance and a bounded, reviewable claim.
- WORLD is not copied wholesale into an Agent context.
```

Initial operating policy:

```text
- it is not injected into an Agent ContextView;
- Garden reads/enforces it on ingest, recall and cognitive-write paths;
- no Agent write interface is exposed initially;
- a human manually edits and debugs it until its governance semantics are proven.
```

#### `WORLD.MD`

`WORLD.MD` is a compact, revisable description of the Agent's actionable world. It contains neither original evidence nor an unbounded history.

Examples:

```text
- current place, physical/environmental facts permitted by the user;
- operating system, machine, installed tools, paths and reachable services;
- projects, repositories, modules, dependencies and operational state;
- people, organizations and relationships when in scope and permitted;
- future multimodal understanding of places, objects, devices, entities,
  relations and observed change.
```

Write policy is intentionally more permissive than Frozen Core:

```text
- user may edit WORLD directly;
- AutoDream may add/revise observations, hypotheses and summaries;
- normal interactive Agent writes are not exposed in the initial slice;
- a user-confirmed assertion may not be silently replaced by AutoDream;
- each claim must remain distinguishable as confirmed, observed, inferred,
  hypothesis or stale/disputed, with a Mentle evidence reference when one exists.
```

`WORLD.MD` is **not** default context. Garden may project only a scope- and task-relevant, budgeted slice; evidence comes from Mentle on demand. This prevents context bloat and unnecessary disclosure as the world model grows, especially after multimodal ingestion.

### 3.4 D — Human-facing Report System: 07–11

This partition is primarily for human reading. It is not a fourth memory tier, an authority source, or a default prompt payload.

| Item | Role | Cadence / context rule |
|---|---|---|
| `07 daily` | readable daily continuity report | scheduled/on-demand; not default context |
| `08 weekly` | readable weekly continuity report | scheduled/on-demand; not default context |
| `09 monthly` | monthly review and orientation report | scheduled/on-demand; may host optional modules |
| `10 AMBITION` | Agent's open-ended wishes: what it hopes for the world, the user and its future self | entertainment/report module; injected only during monthly review |
| `11 USER SUGGESTIONS` | non-binding suggestions to the user based on observed work/opportunity | entertainment/report module; injected only during monthly review |

`AMBITION` is not a task list, commitment, policy or automatic self-modification authority. `USER SUGGESTIONS` is not a proposal queue, commitment or automatically-created task. The user decides whether a suggestion becomes work.

The primary reader is human. During a Mentle outage, a matching latest human report may provide bounded orientation after Frozen Core and STM, but it never replaces raw transient-event recovery.

### 3.5 E — Infrastructure outside content partitions

#### Change/audit log

Legacy section `12-changelog` becomes an append-only operational/governance log, not an authority file or normal ContextView content. It records consequential events such as:

```text
- manual MEMRULES revisions;
- significant WORLD revisions;
- EvoMap mailbox state changes;
- approved capability/host installation/reversal;
- report generation revisions.
```

Routine STM edits and every raw Mentle ingestion must not create heavy governance-log noise.

#### EvoMap mailbox

The historic `11-proposal_inbox` is not retained as the EvoMap proposal system. Evolver requires a separately designed mailbox with explicit inbox/outbox, state machine, evidence references, privacy gate, evaluation, approval/rejection, retry and dead-letter semantics. It remains deferred.

#### Transient spool

The Laputa transient spool remains a bounded append-only emergency outbox for raw activity while Mentle is unavailable. It is not an authority section, report, world model or material lake. Entries stay `pending_mentle` until receipt-bound idempotent drain by `event_id + content_hash`.

### 3.6 F — Removed concepts

| Legacy item | Decision | Reason |
|---|---|---|
| `06-history_md` / LTM / `LONGMEM.MD` | remove from target architecture | Mentle already owns historical materials, discovery and evidence. A second LTM store causes duplicate authority and stale-current-state problems. |
| `13-report_indexes` | remove as a Laputa concept | report catalog/indexing is report-subsystem metadata, not cognition or authority. |
| `14-aaak_summaries` | remove as a Laputa concept | its low-cost summary role is superseded by STM, human reports, Mentle discovery and Garden ContextView. |

The useful AAAK lesson survives only as a Mentle ingestion principle: future Obsidian content should become small, provenance-preserving semantic units for indexing—not a Laputa summary file.

```text
Obsidian page
  -> heading/block/semantic unit
  -> source ref + heading path + offsets + content hash + scope/tags
  -> Mentle indexing and evidence read
```

---

## 4. Context Plane

```text
Layer 0  Frozen Core Prompt (01–04), frozen within the session
Layer 1  STM bootstrap (05 MEMORY.MD + current working set)
Layer 2  Garden Context Facade: task/scope/budget controlled Mentle evidence
         and selective WORLD projection

Never default-inject:
- MEMRULES.MD
- complete WORLD.MD
- complete human reports
- raw Mentle materials
- operational/audit log
```

---

## 5. Consequences

### Positive

- eliminates the duplicate LTM material/index architecture;
- separates rules governing cognition from the current understanding of reality;
- gives WORLD room to grow into a multimodal digital-companion world model without making it default prompt baggage;
- makes reports explicitly human-facing and keeps AI ambition/suggestions non-binding;
- removes legacy report-index and summary-file concepts from Laputa;
- prevents an external Evolver mailbox from being confused with a user-facing report feature.

### Costs and constraints

- existing Go constants/files remain a legacy migration surface until a later implementation plan changes them;
- no implementation may claim that `MEMRULES.MD` or `WORLD.MD` exists until its storage, projection and validation contract is separately implemented;
- the `MEMRULES.MD` human-edit-only policy intentionally slows automation at first;
- WORLD needs bounded claim/provenance conventions before AutoDream writes can be enabled safely.

---

## 6. Required follow-up before implementation

1. Design the physical representation and validation of `MEMRULES.MD` and `WORLD.MD`.
2. Define the compatibility migration from legacy section constants/files without data loss.
3. Design the EvoMap mailbox separately; do not reuse `proposal_inbox` semantics.
4. Extend the report subsystem only after the report data model can support monthly optional modules (`AMBITION`, `USER SUGGESTIONS`).
5. Specify Obsidian source adapter and AAAK-style semantic-unit ingestion for Mentle; do not create `aaak_summaries` in Laputa.
6. Update tests and host projections after the physical migration plan is accepted.
