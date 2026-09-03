# Garden MemoryOS vNext Architecture Plan

> **Status:** proposed - 待松本评审
> **Date:** 2026-08-01
> **Owner:** Garden / Laputa / Mentle architecture
> **Scope:** Garden 工作区下一轮 MemoryOS 改造的总体架构、模块边界、公共契约、迁移顺序和验收标准。
> **Implementation state:** 仅完成文档设计；尚未修改 Go 运行代码。
> **Historical archive:** `docs/archive/2026-08-01-pre-memoryos-redesign/`

---

## 0. Executive Decision

Garden 的下一阶段不是继续堆叠 CRUD、Agentic RAG planner 或重型 pipeline，而是把当前系统收敛为一个有明确 authority、材料、召回、活动和演化边界的 MemoryOS。

```text
Laputa = Agent identity / authority / lifecycle / policy / audit
Mentle = categorized material substrate / evidence / retrieval / graph
Garden = source ingestion / runtime orchestration / recall / context assembly
Evolver = external evolution engine, accessed through a governed adapter
Host adapters = Hermes / Claude Code / Codex / OpenClaw artifact integration
```

产品定义：

```text
MemoryOS = 以 Agent 人格与治理记忆为核心，
           能理解、定位、调用个人全部工作信息的记忆操作系统
```

核心决策：

1. `MEMORY.MD` 是 STM authority checkpoint，不是全部 STM；Agent 可直接维护其 working projection。
2. Mentle 是原文优先的 Material Universe / Evidence Lake：正常路径先持久化原始 SourceArtifact，再派生卡片、证据、摘要和检索索引；Mentle 不取得 authority。
3. Laputa 不再拥有 LTM / `LONGMEM.MD` 长期内容层；历史材料、版本和证据由 Mentle 保留，避免双写和双真源。
4. `MEMRULES.MD` 是认知治理规则：Garden 在 ingest、recall 和认知写路径执行；初期只允许人工维护，不注入 Agent ContextView。
5. `WORLD.MD` 是可修正的行动世界理解：用户可修改、AutoDream 可参与；仅按 scope/task/budget 投影，绝不默认全量注入。
6. Fast Recall 是默认路径：零 LLM、确定性、低延迟、可缓存、可降级。
7. Deep Recall 是显式升级：独立预算、独立 trace、可调用 KG/timeline/graph。
8. `candidate discovery != evidence read != final ContextView`。
9. Heat 只能影响排序和生命周期建议，不能单独裁决真值、权限或认知状态。
10. Garden 不内置自主进化引擎，直接通过外部 adapter 使用 EvoMap/Evolver；EvoMap mailbox 另行设计，不复用旧 `proposal_inbox`。
11. Evolver 的 Gene/Capsule/SkillDraft 都是 candidate，不能越过 Laputa authority spine。
12. 旧 API 采用兼容迁移；新语义不得继续扩大旧 `memory:*` / `context/resolve` 的隐式职责。
13. Laputa Cognitive Partition 的具体决定以 ADR-0002 为准。

---

## 1. Scope Fence

### 1.1 本方案覆盖

- Garden vNext 的总体组件和数据流；
- Laputa、Mentle、Garden 的明确 ownership；
- STM、`MEMRULES.MD`、`WORLD.MD`、canonical material、external source、evidence、heat 的对象模型；
- Fast Recall、Deep Recall、session ingest、report 和 evolution 的运行边界；
- Garden HTTP API 的新版本契约；
- Go 包结构、接口落点和兼容层；
- 迁移波次、每波输出、测试和性能验收；
- 旧文档归档后的文档治理规则。

### 1.2 本方案不覆盖

- 重写 Laputa 14-section 的具体业务 schema；
- 重写 Mentle 的 vector、BM25、ONNX、KG 或 palace graph 内核；
- 自研第二套 Gene/Capsule/evolution engine；
- EvoMap/Evolver 内部算法或 GEP 未来协议；
- 自动安装或自动发布 host skill；
- 公共 EvoMap Hub 的默认接入；
- Rust agent-diva 的实现迁移；
- Hermes、Claude Code、Codex 的完整宿主插件实现；
- 许可证最终法律意见。

---

## 2. Current Baseline

本节是实施前必须重新验证的事实基线。历史文档已归档，代码和本节的真实状态优先于旧计划中的“Phase complete”声明。

### 2.1 Garden composition root

`garden/main.go` 当前已经组装：

```text
governance.NewFileStore
  -> governance.NewEngine
  -> facade.Service (可降级)
  -> crud.Handler
  -> pipeline.Manager
  -> rag.Service
  -> ingest.Service (SQLite state DB)
  -> report.Service
  -> server.Server
  -> lifecycle.Run
```

对应当前文件：

- `garden/main.go:21-123`：composition root、Mentle degradation、pipeline、ingest、report、server wiring；
- `garden/internal/server/server.go:42-60`：HTTP route registration；
- `garden/internal/server/server.go:71-241`：legacy/canonical CRUD；
- `garden/internal/server/server.go:244-360`：session ingest、bootstrap、context resolve；
- `garden/internal/server/server.go:363-414`：pipeline read-only inspection。

当前已有 route：

```text
POST   /v1/memories
GET    /v1/memories/{key}
GET    /v1/memories
PATCH  /v1/memories/{key}
DELETE /v1/memories/{key}
POST   /v1/sessions
GET    /v1/ingestions/{id}
POST   /v1/context/resolve
POST   /v1/context/bootstrap
GET    /v1/reports/latest
GET    /v1/pipelines...
GET    /health
```

### 2.2 Current retrieval gap

`garden/internal/rag/types.go:49-55` 的 `Candidate` 直接带 `Content`；`mentle/facade/retrieval.go:17-24` 的 `RetrievalHit` 也直接带 `Content`；`mentle/facade/retrieval.go:42-74` 在 retrieval 阶段从 physical vector revision 回填 canonical 正文。

这违反 vNext 的候选/证据分离要求。第一波核心迁移必须先增加 card-only API，再让 Fast Recall 使用新 API。

### 2.3 Current canonical model

`mentle/facade/canonical.go:23-45` 已有可复用字段：

```text
id / kind / content / status / version / scope / tags
source / valid_from / valid_to / supersedes / superseded_by
timestamps / metadata
```

`canonical.go:93-112` 已有 SQLite catalog、idempotency、index_jobs、audit_log；`canonical.go:124-188` 已有 transactional create；`canonical.go:202-250` 已有 versioned update；`canonical.go:253-290` 已有 soft delete。

因此 vNext 应增量扩展 canonical/catalog，不另起第二个 Memory 表。

### 2.4 Current authority gap

`laputa/governance/engine.go:53-87` 已有 14 sections 和 authority metadata；`engine.go:89-108` 的 `SectionStore` 提供 CRUD，但接口本身不带 actor、policy decision、proposal 或 audit context；`engine.go:196-232` 的 `FileStore.Write` 直接替换 section，并只写 `_meta.updated_at/version`。

vNext 必须增加 authority-aware application service 或 mutation gateway，不能让 HTTP handler 或普通 adapter 直接获得 authority file 写权。

### 2.5 Current compatibility facts

旧 `memory:*` / `section:*` CRUD 和旧 `context/resolve` 已被测试或宿主契约使用，不能在第一波删除。兼容层只负责翻译请求，不能继续成为新架构的内部主 API。

`garden/go.mod` 的 sibling module `replace` 路径、Go workspace 布局和三个仓库边界保留，除非后续独立迁移 ADR 明确改变。

---

## 3. Target Architecture

```text
Host: Hermes / Claude Code / Codex / OpenClaw
        │
        │ stable host contract
        ▼
Garden HTTP / local adapter gateway
        │
        ├── Recall Gateway
        │     ├── Fast Recall
        │     └── Deep Recall
        │
        ├── Activity & Source Gateway
        │     ├── session ingest
        │     ├── external source sync
        │     └── report input
        │
        ├── Evolution Adapter
        │     ├── Evidence Bundle sanitizer
        │     ├── local Evolver MCP/CLI sidecar
        │     ├── candidate normalization
        │     └── proposal handoff
        │
        ├── Authority Application Service
        │     ├── Laputa projection
        │     ├── proposal/review
        │     ├── audit
        │     └── rollback
        │
        └── Compatibility Layer
              ├── legacy CRUD
              ├── legacy context resolve
              └── old pipeline inspection
        │
        ├── Laputa governance / authority files
        └── Mentle canonical catalog / material / index / KG
```

### 3.1 Ownership rule

```text
Garden may orchestrate.
Garden may not become authority.
Mentle may store and retrieve.
Mentle may not promote authority.
Evolver may propose capability.
Evolver may not persist authority.
Laputa may approve and apply authority.
```

### 3.2 Runtime modes

| Mode | Default | LLM | KG/graph | Writes | Trace |
|---|---:|---:|---:|---:|---:|
| Fast Recall | yes | no | no | no | compact request trace |
| Deep Recall | explicit | optional | explicit | no | required `RecallTrace` |
| Session Ingest | lifecycle | optional later | no default | activity/material | ingestion trace |
| Evolution Run | explicit/background | external Evolver may use | evidence scope only | proposal only | evolution event chain |
| Authority Apply | explicit governed action | no default | no default | Laputa/Mentle coordinated | audit + rollback ref |

---

## 4. Object Model

### 4.1 GovernanceProjection

A small read-only projection assembled from Laputa. It is not a copy of authority files.

```go
type GovernanceProjection struct {
    IdentityRef       string
    Scope             string
    AllowedSources    []string
    DeniedSources     []string
    AllowedKinds      []string
    WorldProjectionRef string
    WorkingSetRefs    []string
    PolicyRevision    string
    ProjectionVersion string
}
```

Rules:

- never expose full `SOUL.MD`, `USERPROFILE.MD`, or `THOUGHTS.MD` through default recall;
- `03-commitment` remains user-only;
- projection carries references and policy, not unrestricted content;
- cache invalidation is driven by Laputa revision, not by retrieval heat.

### 4.2 MemoryCard

Card-only candidate object. It must not contain full正文.

```go
type MemoryCard struct {
    ID             string
    Kind           string
    Collection     string
    Scope          string
    Title          string
    Summary        string
    SourceRef      string
    Revision       int
    Status         string
    ValidFrom      time.Time
    ValidTo        *time.Time
    SupersededBy   *string
    Tags           []string
    HeatScore      float64
    LastActivated  *time.Time
    CandidateScore float64
}
```

`Summary` is bounded and safe for candidate discovery. `Content` is deliberately absent.

### 4.3 EvidenceFragment

Evidence is read only after a card passes policy and ranking.

```go
type EvidenceFragment struct {
    CardID       string
    MaterialRef  string
    SourceURI    string
    SourceRev    string
    Excerpt      string
    StartOffset  int
    EndOffset    int
    ContentHash  string
    Validity     string
    EvidenceRefs []string
}
```

Every evidence read must enforce item and character budgets.

### 4.4 ContextView

Disposable final context assembled for one request.

```go
type ContextView struct {
    TraceID       string
    Scope         string
    Mode          string
    Governance    GovernanceProjection
    Cards         []MemoryCard
    Evidence      []EvidenceFragment
    Context       string
    BudgetChars   int
    Degraded      bool
    Warnings      []string
    RecallTraceID *string
}
```

A `ContextView` is not written to Laputa authority and is not automatically promoted to memory.

### 4.5 Canonical material extensions

Extend Mentle canonical metadata with orthogonal axes:

```text
Lifecycle: stm | artifact | reflective | report | world_candidate
Collection: working | knowledge | artifacts | reports | reflection
CognitiveRef: WORLD claim / mailbox / authority reference, nullable
SourceRef: external source or session artifact
EvidenceStatus: candidate | verified | disputed | superseded | retracted
Heat: base_heat, last_activated_at, activation_events
```

Do not use `wing` or `room` as lifecycle or authority. They remain Mentle taxonomy:

```text
collection -> wing -> room -> optional hall
```

### 4.6 Evolution objects

Garden-side metadata must distinguish:

```text
EvolutionEvidenceBundle
GeneCandidate
CapsuleCandidate
SkillDraft
EvolutionProposal
PortableSkill
HostArtifact
EvolutionEvent
```

No one of these objects is authority merely because it exists. Only an approved proposal can produce a durable capability index or host installation record.

---

## 5. Module Layout

The following is the target layout. Names are proposed and may be adjusted during implementation without changing ownership.

### 5.1 Garden application module

```text
garden/internal/
  authority/
    projection.go       GovernanceProjection assembly
    proposal.go         EvolutionProposal / mutation plan
    audit.go            cross-component audit events
    service.go          governed apply boundary

  recall/
    cards.go            MemoryCard contracts
    fast.go             deterministic Fast Recall
    deep.go             explicit Deep Recall adapter
    evidence.go         bounded ReadEvidence
    rank.go             policy-first filter/rank/dedupe
    trace.go            RecallTrace

  activity/
    event.go            normalized activity event
    working_set.go      STM runtime cache
    checkpoint.go       MEMORY.MD checkpoint orchestration
    transient.go        Laputa fallback spool and drain state

  source/
    artifact.go         SourceArtifact contract
    adapter.go          read-only external source interface
    sync.go             source revision/hash/sync state

  evolution/
    bundle.go           Evolution Evidence Bundle
    policy.go           privacy/scope/publication policy
    adapter.go          EvolverProvider interface
    normalize.go        Gene/Capsule/SkillDraft normalization
    host.go             PortableSkill/HostArtifact metadata
    events.go           evolution audit events

  compatibility/
    legacy_memory.go    old CRUD translation
    legacy_context.go   old context/resolve translation

  server/
    recall_handlers.go
    evolution_handlers.go
    authority_handlers.go
```

### 5.2 Mentle facade module

Add public APIs without exposing internal vector payloads:

```go
type CardQuery struct {
    Text       string
    Scope      string
    Collection string
    Status     []string
    Limit      int
    Cursor     string
}

func (s *Service) SearchCards(ctx context.Context, q CardQuery) (CardPage, error)
func (s *Service) ReadEvidence(ctx context.Context, q EvidenceQuery) ([]EvidenceFragment, error)
func (s *Service) GetCanonical(ctx context.Context, id string) (Memory, error)
func (s *Service) RecordActivation(ctx context.Context, event ActivationEvent) error
```

Existing `Retrieve()` remains during migration but becomes an internal compatibility implementation. It must not be used by new Fast Recall code after the card API lands.

### 5.3 Laputa governance module

Add a governed application boundary above raw `SectionStore`:

```go
type MutationRequest struct {
    Actor       string
    Section     SectionName
    Operation   string
    Payload     map[string]any
    Reason      string
    RequestID   string
    ProposalID  string
}

type MutationDecision struct {
    Allowed     bool
    PolicyRef   string
    AuditRef    string
    RollbackRef string
}

func (e *Engine) EvaluateMutation(ctx context.Context, req MutationRequest) (MutationDecision, error)
func (e *Engine) ApplyMutation(ctx context.Context, req MutationRequest) (MutationDecision, error)
```

Raw `SectionStore` remains a storage interface. Public Garden routes must use the governed application boundary for authority-affecting changes.

---

## 6. Recall Architecture

### 6.1 Fast Recall algorithm

```text
1. Validate query, scope and character budget.
2. Build or load GovernanceProjection.
3. Load hot_context and working_set for the scope.
4. Select a bounded task-relevant WORLD projection when policy permits; never load all WORLD.
5. Ask Mentle SearchCards for bounded candidates.
6. Apply policy/status/scope/validity gates.
7. Apply deterministic score: semantic relevance + scope + recency + bounded heat bonus.
8. Dedupe by canonical ID and revision.
9. Read bounded evidence for selected cards only.
10. Assemble ContextView.
11. Emit compact trace and return degraded warnings if a backend failed.
```

Hard constraints:

- no LLM;
- no KG/timeline/palace graph;
- no full external source scan;
- no full `MEMORY.MD`, `WORLD.MD` or report injection;
- `MEMRULES.MD` is enforced by Garden but never rendered as recall content;
- no writes;
- no automatic evolution;
- denied or superseded material is filtered before ranking.

### 6.2 Deep Recall algorithm

```text
Fast Recall seed
  -> explicit capability + budget check
  -> wider source/time/taxonomy scope
  -> optional KG/timeline/palace graph
  -> evidence expansion
  -> optional planner
  -> ContextView + RecallTrace
```

Deep Recall must be separately addressable, separately budgeted and separately observable. A failure returns Fast Recall seed plus `degraded=true`; it must not silently execute deep behavior on the default path.

### 6.3 Heat policy

```text
Heat     = priority for reuse
Value    = long-term utility
Truth    = validity / conflict state
Authority= who may promote, edit, delete
```

Events:

- user confirmation, explicit citation and successful task use increase heat;
- mere retrieval is zero or negligible heat;
- repeated automatic reads within one session do not accumulate heat;
- user denial, source deletion or supersede freezes/clears heat;
- low heat may move an item out of Fast Recall but never proves it false;
- physical deletion requires source delete, retention expiry, explicit user delete or compliance policy.

---

## 7. STM, Cognitive Governance and Activity Flow

### 7.1 STM runtime cache

```text
Raw SourceArtifact         immutable original material, normal durable path: Mentle
hot_context[session]       current turn / call chain, shortest TTL
recent_activity[session]   ordered reconstructible event references
working_set[scope]         Agent-editable active cards, evidence refs and pending leads
compressed_segments        optional bounded STM projection derived from raw material
MEMORY.MD                  atomically written authority checkpoint of the working set
Laputa transient spool     bounded raw-event fallback only when Mentle is unavailable
```

The raw material and STM projection are deliberately different things:

```text
raw material      = what happened, preserved with source/provenance
STM projection    = what the Agent is working on now, directly editable and rebuildable
WORLD projection  = a bounded, task-relevant slice of actionable world understanding
MEMRULES          = Garden-enforced cognitive governance, never default context
```

Normal write path:

```text
activity event
  -> canonicalize event_id + content_hash + scope
  -> durable raw SourceArtifact in Mentle
  -> acknowledge durable write
  -> recent_activity append (reference only)
  -> working_set / hot_context update
  -> [optional asynchronous] bounded LLM compression
  -> atomic MEMORY.MD checkpoint
```

LLM compression is an optimization, never the durability gate. If it is unavailable, raw ingest, direct Agent STM edits and the last valid checkpoint continue; only a new derived summary is delayed.

Agent STM edits are normal operations and require no proposal, approval workflow or heavy audit. They may add/remove/reorder active references, change a working summary, annotate a pending lead, or checkpoint the current working set. An STM edit must never overwrite original source material: it creates a new STM projection revision while retaining `source_ref` / `material_ref`.

Each checkpoint and working-set write keeps only minimal operational metadata: `revision`, `updated_at`, `actor`, `base_revision`, and an atomic write result. This is for concurrency/recovery, not a second audit subsystem. Cache loss must be recoverable from `MEMORY.MD` plus Mentle material.

### 7.2 Mentle-unavailable fallback

When Mentle cannot accept the raw event, Garden writes the original event to a bounded Laputa transient spool and acknowledges the write as degraded durable storage:

```text
activity event
  -> canonicalize event_id + content_hash + scope
  -> append-only Laputa transient spool (pending_mentle)
  -> Agent may bounded-read raw events by session/scope/cursor
  -> Mentle health recovery
  -> idempotent drain by event_id + content_hash
  -> verify Mentle receipt
  -> mark drained and expire the transient copy
```

The spool is not a second Mentle and not an authority section. It is an emergency, local, append-only runtime area with retention, quota, cursor-based bounded reads and retry state. It never becomes LTM merely because it was retained. `MEMORY.MD` records pending transient references but does not copy their full content.

**Recovery is mandatory, not best effort.** Every degraded write has a durable outbox record and remains `pending_mentle` until Mentle returns a receipt for the same `event_id + content_hash`. Recovery workers process records in stable event order, retry idempotently, and never remove a transient entry before that receipt is committed. This covers every fragmented raw activity item, including small user messages, tool results, command output, manually added notes and source-ingest fragments; no LLM compression is required to rehydrate it.

A direct Agent STM edit normally changes only the working projection and is already protected by the atomic `MEMORY.MD` revision. If that edit introduces new user-authored or Agent-authored material with no existing `material_ref`, Garden must append that material to the same transient outbox during Mentle outage. On recovery it is materialized in Mentle first, then its STM projection keeps/rebinds the resulting `material_ref`. Thus a transient cache edit cannot leave an orphaned piece of durable content.

The required native read contract is intentionally small:

```go
type TransientReadRequest struct {
    SessionID    string
    Scope        string
    AfterEventID string
    LimitEvents  int
    BudgetChars  int
    IncludeKinds []string
}
```

Default reads are limited to the current session/scope and declared event/character budgets. They perform no LLM call, vector search or full automatic context injection.

### 7.3 Cognitive governance: MEMRULES and WORLD

The former STM→LTM promotion path is removed. A session event never creates a second long-term memory store. Mentle preserves material, historical versions and evidence; `MEMRULES.MD` governs how Garden forms and uses cognitive understanding; `WORLD.MD` holds only compact, actionable world claims.

```text
material / observation / user edit / AutoDream output
  -> Mentle evidence or explicit source reference
  -> MEMRULES scope/provenance/conflict/visibility gate
  -> bounded WORLD claim or revision
  -> optional scoped ContextView projection
```

Initial constraints:

- no Agent write API for `MEMRULES.MD`; human maintenance only;
- user may edit `WORLD.MD`; AutoDream participation is allowed only after claim provenance and protected-user-assertion rules are implemented;
- `WORLD.MD` must never become a default full-context payload;
- a WORLD claim is not raw evidence and must retain its status/authority/confidence distinction plus source reference where available;
- legacy `06-history_md`/`LONGMEM.MD` must not receive new LTM writes.

The full partition and migration constraints are in ADR-0002.

### 7.4 Human reports and emergency orientation

Daily, weekly and monthly reports are **human-facing continuity artifacts**, not a fourth memory tier, a mandatory summarization pipeline, or a long-term-memory promotion mechanism. `AMBITION` and `USER SUGGESTIONS` are optional monthly human modules under ADR-0002. Their primary reader is the user; their secondary use is bounded orientation when Mentle is unavailable.

```text
Report answers:  "这一时间窗口做了什么、结果怎样、现在卡在哪里？"
Mentle answers:  "当时原始材料、证据和细节是什么？"
WORLD answers:   "当前可行动世界如何理解，且与本任务相关的部分是什么？"
STM answers:     "这一会话/当前工作下一步做什么？"
```

Each report is a compact, human-readable snapshot with explicit scope and source references:

```text
cadence + window + scope
goals / completed / validated outcomes
active decisions and changed constraints
open loops / risks / next actions
source_refs + report revision + generated_at
```

The canonical durable human-facing copy belongs under Laputa's report-system-owned sections (`07-daily`, `08-weekly`, `09-monthly`) or their linked local artifacts. A Mentle copy is optional and useful for normal search, but reports must remain readable when Mentle is down. Reports contain concise orientation, never the only copy of raw activity, tool output or source evidence.

Report generation is explicitly scheduled or user-requested, scoped and idempotent. It may use deterministic event/checkpoint data; LLM generation is optional. A failed or skipped report never blocks ingestion, STM checkpointing, transient-spool recovery or WORLD/MEMRULES governance.

When Mentle is unavailable, bootstrap order is:

```text
Laputa Core Prompt
  -> MEMORY.MD checkpoint / working_set
  -> latest matching human report (only if needed for scope orientation)
  -> bounded Laputa transient read for missing recent detail
```

The report is not automatically injected into every session and cannot stand in for the mandatory transient outbox drain. After Mentle recovery, transient raw data is still rehydrated by `event_id + content_hash`; reports are not replayed as source material.

---

## 8. Source and Session Ingestion

### 8.1 SourceArtifact

Every external or session source is represented with:

```text
source_id
source_uri
source_type
source_revision
content_hash
observed_at
sync_state
read_only=true
```

Original source remains authoritative. Mentle stores snapshot/chunks/index and never writes back by default.

### 8.2 Session ingest

Keep the current lifecycle endpoint and idempotency behavior, but change the semantic destination:

```text
POST /v1/sessions
  -> validate session_id/event_id/phase
  -> canonicalize and persist full raw SourceArtifact in Mentle
  -> acknowledge durable raw write
  -> append activity reference and update recent_activity
  -> queue optional bounded compression
  -> optionally update MEMORY.MD checkpoint
```

If Mentle is unavailable, persist the full original event in Laputa transient spool with `pending_mentle` state, return a degraded durability warning, and drain idempotently after recovery. Do not replace the original with a normalized tail digest, and do not clear the only durable raw copy after generating a summary.

A session event must not automatically create a WORLD claim, authority mutation, skill, or EvoMap upload. LLM compression failure must not fail the session event.

### 8.3 External source adapters

First supported adapters are read-only and bounded:

- Obsidian vault;
- local Git repository/worktree;
- Markdown/document directory;
- report directory.

Adapter contract:

```go
type SourceAdapter interface {
    Scan(ctx context.Context, scope SourceScope) ([]SourceArtifact, error)
    Read(ctx context.Context, ref SourceRef, budget ReadBudget) (EvidenceFragment, error)
    Revision(ctx context.Context, ref SourceRef) (SourceRevision, error)
}
```

Full indexing means discoverable, locatable and traceable. It does not mean full injection or automatic WORLD claim creation.

---

## 9. EvoMap/Evolver Integration

Garden does not implement evolution internally. It uses a bounded external adapter.

### 9.1 Input boundary

Garden submits only an `EvolutionEvidenceBundle`:

```text
trigger / failure / correction / successful pattern
outcome and validation result
execution trace reference
blast-radius declaration
evidence refs
source revision and content hashes
privacy/publication policy
```

No full Mentle corpus, raw personality files, tokens, private paths or unrestricted transcript is sent.

### 9.2 Process boundary

Preferred vNext integration:

```text
Garden Go process
  -> local Evolver MCP server or bounded CLI process
  -> restricted local asset store
  -> normalized candidate response
```

The sidecar must have explicit limits for:

- working directory;
- readable evidence refs;
- environment variables;
- network access;
- Hub access;
- timeout and output size;
- process lifetime.

No in-process/bundled Evolver dependency before license and supply-chain review.

### 9.3 Output boundary

```text
Evolver Gene/Capsule/SkillDraft
  -> Garden normalization and leakage/policy report
  -> Laputa EvolutionProposal
  -> user review
  -> authority apply
  -> PortableSkill
  -> HostArtifact
  -> host validation
  -> explicit install/enable
```

No candidate can directly write Laputa, install a host artifact or publish to Hub.

### 9.4 Hub policy

```text
local evolution = policy-allowed
host export     = explicit approval
hub publish     = disabled by default
hub fetch       = explicit, scoped, audited
```

Mechanical leakage audit is necessary but not sufficient; semantic privacy review remains Garden/Laputa responsibility.

---

## 10. HTTP Contract vNext

Use new endpoints for new semantics. Preserve old endpoints as compatibility routes.

### 10.1 Recall

```http
POST /v2/recall/fast
POST /v2/recall/deep
GET  /v2/recall/traces/{trace_id}
```

Fast request:

```json
{
  "query": "当前 Garden 记忆架构的约束是什么？",
  "scope": "project:garden",
  "budget_chars": 6000,
  "session_id": "sess_..."
}
```

Response must expose cards/evidence/context separately:

```json
{
  "trace_id": "recall_...",
  "mode": "fast",
  "cards": [],
  "evidence": [],
  "context": "...",
  "degraded": false,
  "warnings": []
}
```

### 10.2 Sources and activity

```http
POST /v2/activity/events
GET  /v2/activity/sessions/{session_id}
GET  /v2/activity/transient
GET  /v2/activity/transient/status
POST /v2/sources/{source_id}/sync
GET  /v2/sources/{source_id}/status
```

`/v2/activity/transient` is a bounded native read route for pending Laputa fallback events. It requires session or scope plus explicit event/character budgets. It is not a general authority-file or filesystem read endpoint.

Existing `/v1/sessions` remains an adapter route until all hosts migrate.

### 10.3 Evolution

```http
POST /v2/evolution/runs
GET  /v2/evolution/runs/{run_id}
GET  /v2/evolution/candidates/{candidate_id}
POST /v2/evolution/proposals
GET  /v2/evolution/proposals/{proposal_id}
POST /v2/evolution/proposals/{proposal_id}/review
GET  /v2/evolution/events/{event_id}
```

Mutation endpoints require explicit actor, request ID, policy context and proposal linkage. There is no public “auto-apply” endpoint.

### 10.4 Authority

```http
GET  /v2/governance/projection
GET  /v2/governance/proposals/{proposal_id}
POST /v2/governance/proposals/{proposal_id}/apply
POST /v2/governance/proposals/{proposal_id}/rollback
```

These routes call the governed Laputa application boundary, not raw `SectionStore`.

### 10.5 Compatibility routes

```text
/v1/memories                -> legacy CRUD translator
/v1/context/resolve         -> Fast or Deep adapter based on explicit mode
/v1/context/bootstrap       -> Fast Recall bootstrap
/v1/pipelines               -> read-only compatibility inspection
```

The translator must add deprecation metadata and must not expose new internal fields accidentally.

---

## 11. Migration Waves

### Wave 0: Document and baseline freeze

**Goal:** establish one active architecture source and an executable baseline.

Actions:

- archive pre-vNext documents;
- record current Git status and submodule revisions;
- run governance, facade, Garden unit and E2E tests;
- snapshot current route list and health behavior;
- confirm no production runtime code was changed during documentation cleanup.

Exit gate:

```text
archive manifest complete
baseline test output recorded
working tree scope understood
```

### Wave 1: Card/Evidence API

**Goal:** remove full正文 from candidate discovery.

Files:

- `mentle/facade/retrieval.go`
- `mentle/facade/canonical.go`
- new `mentle/facade/cards.go`
- new `mentle/facade/evidence.go`
- `garden/internal/rag/types.go`
- focused facade and RAG tests

Actions:

1. Add `SearchCards` and bounded `ReadEvidence`.
2. Keep `Retrieve` for compatibility.
3. Make new Fast Recall depend only on card API.
4. Add tests proving card responses contain no full content.
5. Add tests for status, validity, version and supersede filtering.

Exit gate: card search never reads long正文; evidence read is budgeted.

### Wave 2: Fast Recall Core

**Goal:** implement deterministic default recall.

Files:

- new `garden/internal/recall/`
- `garden/internal/authority/`
- `garden/internal/server/recall_handlers.go`
- `garden/main.go`
- integration tests

Actions:

1. Build GovernanceProjection.
2. Load hot_context/working_set and, only when task/scope policy allows, a bounded WORLD projection.
3. Search cards, filter policy, rank and dedupe.
4. Read bounded evidence and assemble ContextView.
5. Add `/v2/recall/fast`.
6. Route `/v1/context/bootstrap` through Fast Recall.

Exit gate: Fast Recall is usable with planner disabled and Mentle degraded.

### Wave 3: STM Runtime and session semantics

**Goal:** make session activity reconstructible and checkpointed.

Files:

- `garden/internal/ingest/service.go`
- new `garden/internal/activity/`
- new Mentle source/canonical metadata migration
- `laputa/governance` checkpoint application boundary

Actions:

1. Normalize `event_id`, `content_hash`, scope and source provenance without replacing original content.
2. Persist original activity as a Mentle `SourceArtifact` before acknowledgment; remove the current tail-only digest/clear-source path.
3. Add Agent-editable `working_set` and atomically revisioned `MEMORY.MD` checkpoint state.
4. Add bounded append-only Laputa transient spool, native cursor/budget read and `pending_mentle` drain state for Mentle outages.
5. Make compression asynchronous and optional; prove LLM unavailability never loses raw material or rejects a durable event.
6. Ensure session end and Agent STM edits do not create duplicate long-term material, WORLD changes or authority mutations automatically.

Exit gate: raw event durability is Mentle-or-transient before acknowledgement; every `pending_mentle` raw item and new-material STM fragment drains in stable order exactly once by event ID/hash after recovery; cache loss can be reconstructed from checkpoint and material refs.

### Wave 4: Governance application boundary

**Goal:** stop direct authority mutation through generic CRUD.

Files:

- `laputa/governance/engine.go`
- new authority mutation/proposal files
- `garden/internal/authority/`
- `garden/internal/server/authority_handlers.go`
- authority tests

Actions:

1. Add actor-aware mutation request.
2. Enforce section authority for new routes.
3. Add proposal/audit/rollback references.
4. Keep raw SectionStore private to the governed service.
5. Mark legacy section CRUD as compatibility-only.

Exit gate: a non-authorized actor cannot mutate authority through v2 or compatibility paths.

### Wave 5: Deep Recall and logical arbiter

**Goal:** isolate expensive reasoning and temporal conflict handling.

Files:

- `garden/internal/recall/deep.go`
- `garden/internal/recall/trace.go`
- new logical arbiter package or authority integration
- `mentle/facade` KG DTO extensions

Actions:

1. Accept Fast Recall seed.
2. Require explicit budget/capabilities.
3. Add temporal assertion output:

```text
subject + predicate + object + valid_from + valid_to
+ confidence + evidence_refs + status
```

4. Return proposal/mutation plan, not direct authority write.
5. Add fallback to Fast Recall with degraded trace.

Exit gate: no default request can trigger KG/timeline/graph.

### Wave 6: EvoMap/Evolver adapter

**Goal:** use external evolution without giving it authority.

Files:

- new `garden/internal/evolution/`
- new `garden/internal/server/evolution_handlers.go`
- local sidecar configuration and policy schema
- evidence/privacy tests

Actions:

1. Define Evidence Bundle and policy gate.
2. Implement bounded MCP/CLI provider interface.
3. Normalize candidate results.
4. Persist candidate exchange only through the separately designed EvoMap mailbox; do not reuse legacy `proposal_inbox`.
5. Keep Hub disabled by default.
6. Do not install or publish automatically.

Exit gate: Evolver failure degrades only evolution, never Fast Recall or authority safety.

### Wave 7: Host adapters and release hardening

**Goal:** export approved PortableSkill candidates per host.

Actions:

- Hermes adapter;
- Claude Code adapter;
- Codex adapter;
- per-host validation and rollback metadata;
- benchmark, security and operational documentation.

Exit gate: every HostArtifact has source skill version, renderer version, validation result and rollback reference.

---

## 12. Testing and Verification

### 12.1 Required test layers

| Layer | Command / method | Required proof |
|---|---|---|
| Laputa | `cd laputa && GOSUMDB=off go test ./governance/...` | section, store, authority behavior |
| Mentle | `cd mentle && GOSUMDB=off go test ./facade/...` | canonical, cards, evidence, lifecycle |
| Garden unit | `cd garden && GOSUMDB=off go test ./internal/...` | recall, activity, server, compatibility |
| Garden E2E | `cd garden && GOSUMDB=off go test -tags=e2e ./e2e/...` | real process, HTTP, degradation |
| Security | focused policy tests | denied source/scope/actor cannot leak or mutate |
| Evolution | sidecar fixture | timeout, malformed result, leakage, Hub denial |
| Benchmark | deterministic harness | P50/P95/P99 under declared dataset |

### 12.2 Mandatory behavioral tests

- SearchCards does not return full `Content`.
- ReadEvidence enforces per-item and total character budget.
- superseded, deleted, expired and out-of-scope cards do not reappear.
- Fast Recall does not call Planner, KG, timeline or graph.
- Deep Recall always emits `RecallTrace`.
- Deep Recall failure returns Fast Recall seed and degraded warning.
- Mentle unavailable leaves governance projection and safe Fast Recall available.
- session end is idempotent on `session_id + event_id`.
- raw activity is preserved before a session event is acknowledged; compression failure cannot remove it.
- Agent may edit STM working_set/checkpoint without a governance proposal; the edit produces a new projection revision and retains source refs.
- `MEMRULES.MD` has no Agent write API in the initial slice; WORLD projection is scoped/budgeted and never default-injected.
- Laputa transient read is bounded by session/scope/cursor and character budget.
- Mentle recovery drains every fragmented `pending_mentle` raw item and new-material STM fragment once, in stable event order, by `event_id + content_hash`; it does not duplicate material.
- transient entries are retained until their Mentle receipt is committed; a restart resumes durable pending drain state.
- daily/weekly/monthly reports are readable from report-system storage while Mentle is unavailable, but never substitute for raw-event drain; AMBITION and USER SUGGESTIONS remain monthly-only human modules.
- a report failure never blocks activity acknowledgement, checkpointing or recovery.
- session ingest does not auto-create WORLD claims or a second long-term material store.
- heat alone cannot establish a WORLD claim, truth, authority or user preference.
- unauthorized authority mutation is rejected and audited.
- Evolver cannot read outside the submitted evidence refs.
- no real trace means no successful Capsule claim.
- Hub publish is impossible without explicit policy and user approval.
- HostArtifact cannot install without approval.
- rollback restores prior host state or reports a bounded failure.

### 12.3 Performance targets

Targets are hypotheses until measured:

| Operation | Target |
|---|---:|
| warm GovernanceProjection | P95 <= 5 ms |
| SearchCards | P95 <= 80 ms |
| filter/rank/dedupe | P95 <= 10 ms |
| bounded ReadEvidence | P95 <= 40 ms |
| Fast Recall total | P95 <= 150 ms, excluding host transport |
| governance-only degradation | P95 <= 30 ms |
| Deep Recall | separate budget and SLO |

---

## 13. Security, Privacy and Failure Policy

### 13.1 Data minimization

- Fast Recall receives only cards and bounded evidence.
- Evolution receives only sanitized evidence bundles.
- Host adapters receive only approved PortableSkill data.
- Hub receives nothing by default.

### 13.2 Failure containment

| Failure | Required behavior |
|---|---|
| Mentle unavailable | append raw activity to bounded Laputa transient spool; governance projection + degraded Fast Recall; use latest scope-matching human report only for orientation when STM is insufficient; native bounded transient read; drain on recovery |
| report generation unavailable | no effect on raw ingest, STM, WORLD or recovery; expose report staleness to the human reader |
| LLM compression unavailable | retain raw material and current STM/cache; skip only derived compression/checkpoint refresh until available |
| Planner unavailable | deterministic path; no automatic Deep escalation |
| Evolver unavailable | evolution run degraded; normal recall unaffected |
| index job failed | durable retry record; current turn not blocked |
| authority apply failed | no partial durable claim; audit failure |
| host validation failed | artifact rejected; no install |
| rollback failed | retain old audit state and require operator intervention |

### 13.3 No silent durable mutation

The following are always explicit and keep an auditable minimal change record:

```text
manual MEMRULES revision
protected/user-confirmed WORLD revision or significant world-model correction
Frozen Core authority mutation
skill approval
host installation
evolution proposal acceptance
Hub fetch/publish
physical deletion
```

Normal STM working-set and checkpoint edits are excluded: they are direct Agent operations with atomic revision metadata, not proposal/audit workflows. They cannot overwrite raw source material, silently alter protected WORLD claims, or create a second long-term material store.

---

## 14. Document Governance

### 14.1 Canonical active documents

The active design set is intentionally small:

- `docs/README.md` — navigation and archive policy;
- `docs/architecture/0001-memoryos-vnext-architecture.md` — this master plan;
- `docs/architecture/0002-laputa-cognitive-partition-decision.md` — accepted cognitive partition that supersedes its former LTM/legacy-section semantics;
- later numbered ADRs — only for decisions that materially change this plan;
- implementation plans — under `docs/plans/` and linked from this document.

### 14.2 Archive policy

The pre-vNext documents are preserved under:

```text
docs/archive/2026-08-01-pre-memoryos-redesign/
  root/
  architecture/
```

Archive documents are evidence/history. They do not override this plan. A new decision must update this plan or create a numbered ADR with an explicit supersession statement.

### 14.3 Required status vocabulary

Every active design document uses one of:

```text
proposed
accepted
implemented
superseded
archived
```

“Phase complete” in historical material is not implementation proof for vNext. Every vNext wave requires executable verification output.

---

## 15. First Implementation Slice

The first code slice is deliberately narrow:

```text
Mentle SearchCards + ReadEvidence
  -> Garden Fast Recall
  -> /v2/recall/fast
  -> /v1/context/bootstrap compatibility translation
```

It must not include:

- EvoMap integration;
- LTM / `LONGMEM.MD` restoration or any second long-term material store;
- Deep Recall planner rewrite;
- authority mutation redesign;
- host installation;
- external source sync.

Reason: candidate/evidence separation is the foundational contract. Until it exists, all later recall, ingestion and evolution work risks preserving the current full-content leakage.

---

## 16. Approval Checklist

This architecture is ready for implementation only after the following are answered:

- [ ] Card and evidence DTO names accepted.
- [ ] `collection` vocabulary accepted.
- [ ] v2 route names accepted.
- [ ] legacy route compatibility period defined.
- [ ] Laputa authority application boundary ownership confirmed.
- [ ] STM checkpoint persistence location confirmed.
- [ ] raw SourceArtifact persistence-before-acknowledgement contract accepted.
- [ ] Laputa transient spool retention, quota, native-read budget and drain policy accepted.
- [ ] Agent STM direct-edit/revision semantics accepted.
- [ ] report artifact schema, scope selection and Laputa report-system storage contract accepted.
- [ ] report fallback injection rule accepted: orientation-only, never raw-data replacement.
- [ ] external source first adapter selected.
- [ ] Evolver sidecar/MCP process policy accepted.
- [ ] license and supply-chain review owner assigned.
- [ ] first implementation slice test fixtures identified.
- [ ] benchmark dataset and SLO measurement method defined.

Until these are checked, no production Garden/Laputa/Mentle runtime code should be modified under this plan.

---

## 17. References

- `docs/archive/2026-08-01-pre-memoryos-redesign/architecture/0004-api-contract.md`
- `docs/archive/2026-08-01-pre-memoryos-redesign/architecture/0005-memoryos-fast-recall-and-upsp-study.md`
- `docs/archive/2026-08-01-pre-memoryos-redesign/architecture/0006-laputa-lifecycle-and-mentle-index-contract.md`
- `docs/archive/2026-08-01-pre-memoryos-redesign/architecture/0007-garden-evomap-evolution-integration.md`
- `garden/main.go:21-123`
- `garden/internal/server/server.go:24-60,71-360`
- `garden/internal/rag/types.go:5-68`
- `mentle/facade/canonical.go:23-112,124-188,202-290`
- `mentle/facade/retrieval.go:10-74`
- `laputa/governance/engine.go:15-108,110-232`
- `C:/Users/Administrator/.workspace/evolver-survey/package.json`
- `C:/Users/Administrator/.workspace/evolver-survey/src/gep/schemas/gene.js`
- `C:/Users/Administrator/.workspace/evolver-survey/src/gep/skill2gep.js`
- `C:/Users/Administrator/.workspace/evolver-survey/src/gep/skill2gepAudit.js`

