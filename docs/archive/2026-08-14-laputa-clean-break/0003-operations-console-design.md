# ADR-0003: MemoryOS Operations Console Design

> **Status:** accepted
> **Date:** 2026-08-02
> **Decision owner:** project owner
> **Resolved:** 2026-08-02 — product identity, frontend iteration mode, admin API shape, i18n, scope default, and preview-mock policy (§10).
> **Relates to:** [0001 MemoryOS vNext Architecture](./0001-memoryos-vnext-architecture.md), [ADR-0002 Laputa Cognitive Partition](./0002-laputa-cognitive-partition-decision.md)
> **Scope:** front-end console product/UX design, admin API contract additions, MVP phasing. Acceptance of this design does **not** authorize runtime migration of Laputa/Mentle/Garden Go code or the legacy 14-section registry; implementation requires a separately approved plan.

---

## 1. Background and positioning

The MemoryOS has three Go modules and a growing set of HTTP surfaces, but no unified way for the owner to see, understand, review, and control the system:

```text
Laputa  = cognitive governance, identity, authority, lifecycle, audit
Mentle  = material universe, evidence lake, retrieval, taxonomy, graph
Garden  = ingestion, recall orchestration, ContextView assembly
Evolver = external evolution engine through a governed adapter (future)
Hosts   = Hermes / Claude Code / Codex / OpenClaw integrations (future)
```

The owner needs an **operations console** (运营台): a local-first admin UI with a standard admin layout (sidebar, top bar, main content) that explains the system's layered architecture, shows component flow status, and lets the owner inspect work context, materials, recall traces, reports, and architecture documents.

### Positioning statement

> The operations console is the **MemoryOS Control Plane**: a place to see, understand, review, and explicitly control a continuous Agent's memory and governance system. It is not a generic CRUD admin, and it is not a second memory tier.

### Core boundaries

```text
Observability ≠ editability      running status ≠ governance authority
Material ≠ cognition             candidates ≠ facts
Graph ≠ decoration               every visual must trace to a real object
```

---

## 2. Goals and non-goals

### Goals

- **Workbench-first:** the console's primary identity is the MemoryOS workbench (context, materials, persona, reports) and aims to be as comprehensive as practical; observability/ops remains a first-class second track, not a separate product.
- Provide a single local console covering Laputa / Garden / Mentle status, flows, and artifacts.
- Visualize the layered architecture (Laputa → Garden → Mentle → hosts) with **per-component flow status** and per-layer state (runtime / data / governance / roadmap).
- Make every recall trace explainable: why candidates were selected or rejected, what evidence was read, and the resulting ContextView manifest.
- Show document hierarchy by role (vision → decisions → target architecture → runtime contracts → implementation evidence → historical archive), not just a file tree.
- Default to **read-only observation**; edits are explicit, scoped, and governed.
- Label legacy compatibility surfaces (14-section registry, sections 06/13/14) so the UI never presents them as target architecture.

### Non-goals (explicit)

- Not a generic CRUD admin for arbitrary keys.
- Not a full editor for every Laputa/Mentle file (no IDE-style backend).
- No automatic WORLD writing, no EvoMap mailbox UI in first phases, no host-side auto-install/publish.
- No cloud sync, no cross-device dashboards, no real-time collaboration.
- No "green health" simulation for capabilities that exist only as accepted design (`MEMRULES.MD`, `WORLD.MD`, EvoMap mailbox, AMBITION/USER SUGGESTIONS reports).

---

## 3. Users and scenarios

| User | Primary scenario |
|---|---|
| Project owner (大湿) | Daily check: is the system healthy, what is the current working context, what needs review? |
| The owner as auditor | Trace a recall: why was this context built? What evidence was used? What was rejected? |
| The owner as operator | See ingestion/spool/pipeline status, pending recovery, degraded components, and operational log. |
| Future host agents (read-only) | Get a stable, bounded status/architecture projection through the same admin API. |

Primary flows:

```text
Morning check   -> Overview (runtime/cognitive/governance/data health)
Investigate     -> Governance Map -> click component -> Inspector -> linked trace/report
Audit a recall  -> Recall Trace -> ContextView manifest -> evidence read
Browse material -> Materials & Evidence -> SearchCards -> Evidence read (bounded)
Find a decision -> Architecture Library -> L0..L5 document hierarchy -> related ADR
```

---

## 4. Information architecture

### 4.1 Global layout

```text
┌───────────────────────────────────────────────────────────────┐
│ Top bar: scope selector · global evidence search · safety banner │
├──────────────┬────────────────────────────────────────────────┤
│ Sidebar      │ Main content                                    │
│              │                                                 │
│ 概览          │                                                 │
│ 治理图谱       │                                                 │
│ 工作上下文     │                                                 │
│ 材料与证据     │                                                 │
│ Recall Trace  │                                                 │
│ 报告系统       │                                                 │
│ 运行与审计     │                                                 │
│ 文档与架构     │                                                 │
│ 设置          │                                                 │
└──────────────┴────────────────────────────────────────────────┘
```

### 4.2 Top bar (cross-page capabilities)

1. **Scope selector** — current host / project / session / collection / time window. All pages observe the same scope; it is the console's "lens". Default is single-host local-first; the selector keeps the shape for future multi-host projections.
2. **Language switcher (i18n)** — UI default is **English**; the top bar exposes a language toggle (English / 中文) that switches the entire UI. All copy is externalized; no hard-coded user-facing strings in components. Chinese translations are required for every shipped string.
3. **Global evidence search** — searches Mentle **cards** (discovery results), not full raw content; opening a card explicitly enters Evidence Read. Full text is never dumped in the search results pane.
4. **Safety banner** — a persistent, context-aware strip:

```text
Local-only · Mentle degraded · N pending_mentle recovery ·
governance policy mode · legacy-compatibility mode · design-accepted-not-implemented
```

The banner is derived from real component state, never mocked.

---

## 5. Page specifications

### 5.1 概览 / Overview

Answers: *"Is the agent's identity, working memory, material library, and runtime currently healthy? What needs my attention now?"*

Upper half — **system pulse** (four distinct health axes):

```text
Runtime health     Laputa / Garden / Mentle / pipeline / planner
                   -> ok / degraded / offline (real component state)
Cognitive health   STM freshness · working-set growth · WORLD candidate conflicts
Governance health  Frozen Core completeness · rules version · pending review count
Data health        pending_mentle spool · indexing backlog · failed sources · unreadable evidence
```

Lower half — **current work surface** (a "what is happening now" workbench, not business metric cards):

```text
current host / session
current goal + STM checkpoint
recent activity + open loops
latest Fast Recall trace
pending risks / pending review items
```

**Overview never renders full material bodies or full personality files.**

### 5.2 治理图谱 / Governance Map

The product's core page. An interactive **layered system graph** with a flow overlay, not a static architecture poster:

```text
                 ┌──────────────────────────┐
                 │ Hosts (Hermes/CC/Codex…) │
                 └────────────┬─────────────┘
                              │ context / events
        ┌─────────────────────▼──────────────────────┐
        │ Garden · orchestration / ingest / recall   │
        └───────┬─────────────────────────┬──────────┘
                │                         │
     ┌──────────▼───────────┐   ┌─────────▼────────────┐
     │ Laputa               │   │ Mentle               │
     │ identity/governance  │   │ materials/evidence   │
     │ STM/rules/world      │   │ catalog/search/KG    │
     └──────────┬───────────┘   └─────────┬────────────┘
                │                         │
                └────────────┬────────────┘
                             ▼
                    ContextView / trace
```

Each node carries **four state layers** (not one green dot):

| Layer | Meaning | Example |
|---|---|---|
| Runtime | service online, latency, failures, degradation | garden: ok; mentle: degraded |
| Data | backlog, staleness, pending recovery, failed sources | 3 pending_mentle |
| Governance | policy mode, review queue, compatibility mode | legacy compat; rules v2 |
| Roadmap | implemented / experimental / accepted-design / deferred | WORLD.MD: accepted, not implemented |

Clicking a node opens the **Inspector drawer** (right side):

```text
职责 (current responsibility)
运行状态 (runtime)
数据边界 (data boundary)
输入 / 输出 (inputs/outputs)
关联 API / trace
治理规则 (governance rules)
已知限制 (known limitations)
架构阶段 (architecture phase: existing / compat / accepted-deferred)
相关文档与 ADR
```

Switchable **flow modes** (tabs, not one universal line graph):

1. **Context Flow** — Host → Garden → STM / scoped WORLD / Mentle evidence → ContextView.
2. **Raw-first Ingestion Flow** — Event → Mentle SourceArtifact → ACK → STM; Mentle unavailable → transient spool → receipt-bound drain.
3. **Recall Flow** — Intent → Fast Recall → Cards → Evidence Read → ContextView → trace.
4. **Governance / Change Flow** — human edit → policy validation → protected write → audit log (explicitly not an ordinary agent auto-write chain).

### 5.3 工作上下文 / Active Work

Corresponds to Laputa Frozen Core, STM, and the future WORLD projection (safe view only).

Sub-pages:

```text
Core Prompt       (Frozen Core 01–04: identity/relationship/commitment/preferences)
STM / MEMORY.MD   (checkpoint + revision history)
Working Set
WORLD projection  (bounded, scope-filtered; full file never default)
MEMRULES status   (existence/version/enforcement status, not content dump)
```

Every ContextView shown here carries a **Context Manifest**:

```text
Frozen Core refs
STM revision
WORLD projection refs
Mentle evidence refs
token/budget allocation
policy decision
trace_id
```

The page explains *what this session received, why, from where, and what can be edited* — with impact analysis (which hosts/sessions a change would affect). It is the page that most distinguishes Garden from an ordinary "knowledge-base chat".

### 5.4 材料与证据 / Materials & Evidence

Mentle's main surface, respecting the two-layer model:

```text
Card discovery  ->  Evidence read
```

Layout:

```text
Left:   Collections / Sources / Taxonomy / Projects
Center: Search cards / timeline / semantic units
Right:  Evidence Inspector
```

Card fields (minimum):

```text
title + summary
source / path / time / content hash
status: candidate / verified / disputed / superseded / retracted
scope / tags / project
used by: STM, WORLD claims, reports, recall traces
```

Evidence Inspector opens the original fragment **bounded by budget**, with location info. Future Obsidian AAAK-style semantic-unit ingestion lands here naturally:

```text
vault page -> heading / block / semantic unit
           -> source path + heading path + offsets + hash
           -> card + evidence-addressable fragment
```

### 5.5 Recall Trace / ContextView

Garden's explainability page. Every Fast/Deep recall can be replayed:

```text
request: task / host / scope / budget
candidates: why recalled
evidence: what was actually read
rejected: what policy/scope/budget refused
context: final ContextView composition
performance: latency, degradation, token/budget
```

Vertical trace timeline (not a log table):

```text
Intent
  ↓
Policy gate
  ↓
Candidate discovery
  ↓
Evidence read
  ↓
WORLD projection
  ↓
ContextView assembly
  ↓
Host delivery
```

"Rejected" is as visible as "selected". This page is the primary audit entry point for recall quality.

### 5.6 报告系统 / Human Continuity

Strictly follows ADR-0002:

```text
07 Daily
08 Weekly
09 Monthly
10 AMBITION          (monthly, human module)
11 USER SUGGESTIONS  (monthly, human module)
```

The UI is a **continuous work journal**, not a second chat history:

- daily: what happened, results, open loops;
- weekly: trends, blockers, decisions needed;
- monthly: review + optional `AMBITION` + `USER SUGGESTIONS`;
- degraded orientation: report can act as bounded fallback when Mentle is down, but never replaces raw recovery.

`AMBITION` / `USER SUGGESTIONS` carry explicit non-binding tags:

```text
人类阅读模块 · 非任务 · 非承诺 · 不会自动创建 EvoMap mailbox 条目
```

### 5.7 运行与审计 / Operations

Engineering control surface:

```text
Component health
Pipeline runs
Ingestion jobs
Transient spool & recovery
Indexing backlog
Operational / governance log
```

First real data sources (already exist):

```text
GET  /health
POST /v1/sessions
GET  /v1/ingestions/{id}
GET  /v1/pipelines
GET  /v1/pipelines/{name}
GET  /v1/pipelines/{name}/runs
GET  /v1/pipelines/{name}/runs/{trace_id}
POST /v2/recall/fast
POST /v2/activity/events
GET  /v2/activity/sessions/{session_id}
```

Garden already maintains `ok/degraded` component state (mentle, pipeline, planner) at startup — this is the first real, non-mock data feed.

### 5.8 文档与架构 / Architecture Library

Document hierarchy **by role**, not directory tree:

```text
L0  Product Vision         MemoryOS definition, goals, boundaries
L1  Accepted Decisions     ADR-0002 + future accepted ADRs
L2  Target Architecture    0001 master plan: modules, interfaces, flows, migration waves
L3  Runtime Contracts      API, schema, state machines, host adapter contracts
L4  Implementation Evidence tests, traces, benchmarks, migration verification
L5  Historical Evidence    archive, read-only, explicitly not current contract
```

Each document shows:

```text
status: proposed / accepted / implemented / superseded / archived
impacted modules: Laputa / Garden / Mentle / Hosts
related pages / API / trace / test
supersedes / superseded-by chain
```

The console becomes the reverse entry point: from a decision, into its runtime proof.

### 5.9 设置 / Settings

Minimal, non-critical:

```text
scope persistence (default host/project/session)
display preferences (density, dark/light)
language preference (default English; toggle also in top bar)
admin API connection info (read-only display)
```

No destructive settings in first versions.

---

## 6. Data sources and API contract additions

### 6.1 Existing real surfaces (verified by source survey)

| Surface | Module | Viable for UI |
|---|---|---|
| `GET /health`, memories CRUD | Garden | overview, operations |
| `POST /v1/context/resolve`, `/v1/context/bootstrap` | Garden | context manifest |
| `POST /v2/recall/fast` | Garden | recall trace |
| `GET /v1/pipelines*`, `/v1/ingestions/{id}`, `/v2/activity/*` | Garden | operations |
| `GET /v1/reports/latest` | Garden | report system |
| snapshot / sections / health (per module survey) | Laputa | governance matrix (read-only) |
| canonical catalog, SearchCards, ReadEvidence, KG, WAL, audit log | Mentle | materials & evidence (no HTTP admin API yet) |

### 6.2 Gaps and required additions (design only, no implementation yet)

```text
Garden 聚合只读 admin API (proposed):
  GET /v2/admin/overview
  GET /v2/admin/components
  GET /v2/admin/context-manifest/{trace_id}
  GET /v2/admin/spool
  GET /v2/admin/audit?since=...

Laputa read-only snapshot adapter (reuse existing snapshot surface, no write)
Mentle read-only HTTP adapter:
  GET /v2/materials/cards?query=…           (card metadata, no body)
  GET /v2/materials/cards/{id}/evidence     (bounded fragment)
  GET /v2/materials/collections
  GET /v2/materials/kg?scope=…

Trace observability supplement:
  persist recall trace records for replay (design only)
```

**Every admin API response must carry** `source: live | compat | accepted-design` so the UI can never be accused of faking a running capability.

**Admin API shape (decision):** Garden aggregates a single admin surface under `/v2/admin/*` (proposed endpoints above). The frontend talks only to Garden; it does not call Laputa or Mentle HTTP/MCP surfaces directly. This keeps one policy boundary, one trace namespace, and one place to enforce loopback binding.

---

### 6.3 Frontend development mode (decision)

Iterate with a **dev server** during development:

```text
dev:   Vite (or equivalent) dev server on 127.0.0.1:5173
       proxy /v2/* and /v1/* -> Garden 127.0.0.1:7373
       hot reload, i18n catalogs, component isolation

prod:  Garden embeds the built static bundle (Go embed) or serves
       a versioned static directory; no CDN, no external assets
```

Dev-server iteration is the default until the console stabilizes; the build target remains Garden-hosted local static.

---

## 7. MVP phasing

### MVP-0 — Read-only console (first slice)

```text
Overview            (real component health + spool + recent activity)
Governance Map      (layered graph + 4-layer state + Inspector)
Component Inspector
Pipeline / ingestion / recall trace (read)
Architecture Library (L0–L5)
```

Rules:

- only real data from existing/verified endpoints;
- no mock fields for MEMRULES/WORLD/EvoMap/reports redesign;
- no high-risk editors;
- validates the console's **object model** before anything is drawn fully.

### MVP-1 — Controlled work-context workbench

```text
STM viewer / revision history
ContextView manifest
WORLD projection viewer (bounded)
Card → Evidence inspector
Report browsing (07–11 as-designed, where runtime exists)
```

Still read-only / explainable-first.

### MVP-2 — Explicit governed editing

Only after Gate A physical-migration plan is approved:

```text
MEMRULES human editor
WORLD claim review / user edit
protected change preview + impact analysis
audit entry + rollback / recovery
```

### Explicitly deferred

```text
Automatic WORLD writing
EvoMap mailbox UI
Evolution candidate installation
Host-side auto publish / install
Full Deep Recall visual workflow
"Edit every file" IDE-style console
```

---

## 8. Safety and governance constraints

- Default to **minimal exposure**: summaries, hashes, references, redacted previews — never full original text, prompts, WAL contents, or complete governance JSON.
- **Why rejected** is as visible as **why selected**.
- Deletion is replaced by isolate / invalidate / archive / revoke-reference; no hard deletes from the console.
- Legacy sections (06/13/14) are marked **compatibility-only** with an ADR-0002 mapping; they never appear as target architecture.
- High-risk areas default read-only; any write is explicit, scoped, and audited.
- No green-health simulation for `MEMRULES.MD`, `WORLD.MD`, EvoMap mailbox, AMBITION/USER SUGGESTIONS until their runtime exists.
- The console is **local-only**; the admin API binds to loopback and requires explicit opt-in for non-loopback (mirroring Garden's existing HIGH RISK warning behavior).
- **Design-preview mode (decision):** a visual review mode is allowed before runtime exists, but only with an explicit global toggle and every preview card/status labeled `source: accepted-design`. Production mode is strictly zero-mock: any unlabeled placeholder is a release-blocking defect.

---

## 9. Acceptance criteria

1. Every status value on every page traces to a real API field or a clearly labeled `compat` / `accepted-design` source; zero mock fields.
2. Legacy 14-section registry and sections 06/13/14 are visually marked compatibility-only with ADR-0002 mapping.
3. Recall Trace shows selected, read, and rejected paths for a real trace id.
4. Materials & Evidence uses card→evidence two-layer model; no full-text dump in search results.
5. The Governance Map renders the layered graph with 4-layer state per node and opens an Inspector.
6. Overview shows the four health axes from real component state; safety banner reflects real degraded state.
7. Architecture Library groups documents by L0–L5 role and shows status/supersession metadata.
8. No write path exists in MVP-0; MVP-1 writes are limited to explicit, audited, low-risk operations.

---

## 10. Resolved decisions (2026-08-02)

| # | Question | Decision |
|---|---|---|
| 1 | Primary identity | **MemoryOS workbench first**, aiming to be as comprehensive as practical; observability/ops is a first-class second track (§2). |
| 2 | Frontend iteration | **Dev server iteration** (Vite-class) with proxy to Garden; prod target is Garden-hosted static bundle (§6.3). |
| 3 | Admin API shape | **Garden aggregates `/v2/admin/*`** as the single console entry; frontend never calls Laputa/Mentle directly (§6.2). |
| 4 | UI language | **English default** with top-bar language toggle; full i18n, Chinese required for every shipped string (§4.1). |
| 5 | Scope default | **Single-host local-first**, selector shaped for future multi-host projection (§4.1). |
| 6 | Preview-mock policy | **Design-preview mode allowed** with explicit toggle + `source: accepted-design` labels; production strictly zero-mock (§8). |

---

## 11. Document governance

- This document is **accepted** (2026-08-02); decisions in §10 are binding for the console's product/UX/API direction.
- It does not modify Go code, APIs, or the legacy registry.
- Implementation requires a separate approved plan; TODOLIST Gate entries must be added before implementation begins.
- Registered in `docs/architecture/AGENTS.md` and linked from `docs/README.md`.
