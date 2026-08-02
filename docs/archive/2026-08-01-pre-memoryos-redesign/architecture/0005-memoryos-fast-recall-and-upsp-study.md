# ADR-0005：MemoryOS 快速召回内核与 UPSP 调研结论

> **Status**：proposed — 待松本评审  
> **Date**：2026-07-29  
> **范围**：Garden / Laputa / Mentle 的记忆召回架构、默认快速路径、可选 Deep Recall（Agentic RAG）及从 UPSP 获得的边界经验。  
> **不在范围**：Diva 内嵌轻量记忆实现、完整记忆摄取/冲突治理落地、LLM provider 选择、重写 Mentle 的底层向量或图谱引擎。

---

## 0. 决策摘要

Garden 的定位是 **MemoryOS**，而非 Diva 的轻量记忆 Provider，也不是一个以 Agentic RAG 为中心的工作流引擎。

本 ADR 提议确立下列架构顺序：

```text
Fast Recall（默认、零 LLM、确定性、低延迟）
    ↓ 仅在宿主明确要求研究/追溯/复杂推理时升级
Deep Recall（可选 Agentic RAG、带预算与 trace）
```

默认召回必须遵守硬边界：

```text
治理真源（Laputa section）
  ≠ GovernanceProjection（小型只读治理投影）

候选卡（MemoryCard）
  ≠ 已读正文（EvidenceFragment）
  ≠ 最终模型上下文（ContextView / ContextPackage）
```

因此，Garden 不应继续沿着“更多 planner、更多多轮查询、更多全量 context 组装”的路径生长。应优先建设一个可缓存、可测试、可降级的 Fast Recall Core：

```text
query + scope + budget
  → GovernanceProjection
  → Mentle SearchCards
  → policy / status / scope filter
  → deterministic rank
  → bounded ReadEvidence
  → ContextView
```

---

## 1. 本日研究基线与约束

### 1.1 项目优先级与名称边界

- `C:/Users/Administrator/Desktop/garden/` 是当前最高优先级工作区。
- 本文中的 **Garden** 一律指 **Laputa-Garden / MemoryOS**，不是 agent-diva 内部的轻量记忆实现。
- Garden 由三个独立仓库构成：

```text
Garden   = 对外入口、API、编排、召回视图
Laputa   = governance（天空）：权限、section、承诺、策略
Mentle   = memory soil（土壤）：规范记忆、索引、hybrid、KG、timeline
```

### 1.2 当前已验证的 Garden 能力

截至本文基线，Garden 已有：

- 统一 CRUD / HTTP 入口；
- Laputa 14 section 治理；
- Mentle hybrid retrieval、KG 与 timeline 接口；
- `agentic_recall_v1` pipeline；
- `basic / advanced / auto` context resolve 模式；
- 异步 session ingest 队列；
- 现有 Garden 内部测试实际通过：`go test ./internal/...`。

主要代码证据：

| 能力 | 当前文件 |
|---|---|
| RAG service 与 basic/advanced 分流 | `garden/internal/rag/service.go` |
| Context DTO | `garden/internal/rag/types.go` |
| Mentle retrieval facade | `mentle/facade/retrieval.go` |
| 异步 session ingest | `garden/internal/ingest/service.go` |
| Laputa section 与写权 | `laputa/governance/engine.go` |

### 1.3 当前关键缺口

现有 Basic 路径虽然不调用 LLM，但仍不是合格的 Fast Recall Core：

1. `rag.Service` 承载了 fast 与 agentic 两种路径，概念边界不清。
2. Mentle `Retrieve()` 在候选阶段直接返回 `Content` 全文；Garden 事后才裁剪为 excerpt。
3. Basic 路径默认将 `01-identity`、`03-commitment`、`04-preferences`、`05-memory_md`、`06-history_md` 的完整 section JSON 作为候选。后两者增长后会成为隐性常驻上下文池。
4. Garden 尚无 `MemoryCard`、`ReadEvidence`、`GovernanceProjection` 的正式 API 契约。
5. session ingest 当前将输入压平并截断后写为 `session_digest`；它保证投递与幂等，但尚不是语义 digest / claim extraction。

---

## 2. UPSP 调研：应借什么，不应复制什么

### 2.1 UPSP 是什么

UPSP 是独立的本地 Agent 应用，不是 Hermes/Diva 的插件或外置工具。其公开 Alpha Runtime 由 Python 运行时、TypeScript GUI、WinForms 壳和本地文件型状态/记忆系统组成。

它的模型是严格串行：

```text
Setup → Reaction（0..N tool frames）→ Cleanup → Round settlement
```

它将实际 provider request、工具回执、记忆写入和 round close 都视为需要证明的事件。

### 2.2 UPSP 的真正亮点：渐进展开

UPSP 的召回不是直接把所有记忆正文灌入模型。它先投影：

- STM / LTM 热度索引；
- STM / LTM / Skills 倒排索引；
- association / relation 索引；
- container 与工作台焦点。

随后才通过明确读取或规则条件把少量正文装入 CONTENT。其关键经验为：

```text
索引候选 ≠ 正文读取 ≠ 当前模型可见证据
```

这正适合 Garden 的 MemoryOS 召回层。

### 2.3 UPSP 不适合复制的部分

UPSP 的架构将人格、关系、记忆、上下文、工具、审计、cleanup 与 settlement 紧密绑定。其核心模块体积大，且每一轮有重型事务语义。它还维护大量上下文缓存与协议层。

Garden 不应复制：

- 巨型 `ContextAssembler`；
- 每次普通召回都进入 Round / Frame / settlement；
- STM 热度成为记忆生命周期真源；
- “完整主体状态”默认常驻模型上下文；
- 将单一位格/人格数据模型绑定为 MemoryOS 的顶层边界。

UPSP 对 Garden 的价值是：**候选、阅读、证据、上下文分离**；不是其重型 Runtime 或主体化协议。

---

## 3. 目标架构：先快后深

```text
Host / Agent
      │
      ▼
Garden Recall Gateway
      │
      ├── POST /v1/recall             Fast Recall：默认
      │       ├── Laputa GovernanceProjection
      │       ├── Mentle SearchCards
      │       ├── deterministic filter/rank
      │       └── bounded ReadEvidence
      │
      └── POST /v1/recall/deep        Deep Recall：显式可选
              ├── 复用 Fast Recall cards/evidence
              ├── query refinement
              ├── KG / timeline expansion
              ├── 可选 LLM planner
              └── trace / citations / degradation
```

### 3.1 Fast Recall：MemoryOS 内核路径

Fast Recall 的目标是“快速拿到最小可信材料”，而不是“自动写出复杂研究答案”。它必须：

- **零 LLM 依赖**；
- 单次、确定性执行；
- 支持缓存；
- 对 Mentle 故障有治理投影降级；
- 通过 scope、权限、有效性和状态过滤；
- 在读取正文前完成候选选择；
- 严格限制返回的 cards、evidence 条数与字符数。

它不应：

- 跑 planner；
- 拆解多 query；
- 扩展 KG / timeline；
- 创建持久任务或 session 写入；
- 自动摘要；
- 因召回失败阻塞宿主 turn。

### 3.2 Deep Recall：可选研究工具

现有 `agentic_recall_v1` 应保留，但从默认核心路径降格为显式高级能力。它适用于：

- 决策追溯；
- 跨时间线问题；
- 多来源冲突核查；
- 深研究；
- 需要图谱遍历或多轮证据 refinement 的请求。

Deep Recall 必须从 Fast Recall 产物开始，而不是重新对全库盲搜：

```text
Fast Recall cards
  → 用户/宿主请求深挖
  → 精读候选正文 / 扩图 / 时间线
  → 可选 planner refinement
  → 深度 evidence pack
```

---

## 4. 三个核心数据层

### 4.1 GovernanceProjection：Laputa 的小型只读治理视图

Laputa section 是治理真源；但 Fast Recall 不应默认序列化完整 section。

```go
type GovernanceProjection struct {
    Version         string
    HardConstraints []ProjectedRule
    Preferences     []ProjectedPreference
    Identity        []ProjectedIdentity
    Access          AccessProjection
    Sources         []Citation
}
```

投影只包含：

- 不可违反的 commitments；
- 与当前 scope 相关的 preferences；
- 必要的身份/行为边界；
- 读取权限与 capability。

`05-memory_md` 与 `06-history_md` 不得作为默认全段投影。它们属于记忆素材，应走 Mentle 的 cards/evidence 路径。

该投影应在 section 更新时增量编译并版本化缓存，而不是每次 Fast Recall 临时 `json.Marshal` 全量 section。

### 4.2 MemoryCard：候选发现层

Mentle 需要提供不含原始全文的候选卡：

```go
type MemoryCard struct {
    ID          string
    Kind        string // decision | preference | fact | artifact | session_digest
    Title       string
    Summary     string // 严格长度上限的预计算预览
    Scope       string
    Status      string // active | superseded | deleted
    Revision    int
    ValidFrom   string
    ValidTo     string
    Provenance  Provenance
    Score       float64
    WhyMatched  []string
    Access      string
}
```

`SearchCards()` 是 Fast Recall 的候选发现接口。它可使用 Mentle 的 BM25、vector、RRF、taxonomy 等内部能力，但不能把其内部 wing/room、provider 参数和模型选择泄漏给宿主。

### 4.3 EvidenceFragment：被批准读取的最小正文

只有被选中后，Garden 才通过 `ReadEvidence()` 读取正文或范围片段：

```go
type ReadEvidenceRequest struct {
    IDs       []string
    Ranges    []TextRange
    Purpose   string // context
    MaxChars  int
}

type EvidenceFragment struct {
    MemoryID    string
    Revision    int
    Locator     string
    Text        string
    StartOffset int
    EndOffset   int
    Citation    string
    Freshness   string
    Provenance  Provenance
}
```

`EvidenceFragment` 才是模型可依据的材料。它只绑定当前请求/活动，随 ContextView 丢弃或重建；不会反向改变记忆真源。

---

## 5. Fast Recall 执行流程与排序

### 5.1 处理流程

```text
FastRecall(query, scope, budget)
  1. 获取 versioned GovernanceProjection（优先缓存）
  2. Mentle SearchCards(query, scope, limit)
  3. 按 access / scope / status / validity 过滤
  4. 运行确定性去重与排序
  5. 选 0..N cards
  6. 对 0..M 选中 cards 执行 bounded ReadEvidence
  7. 渲染 ContextView / ContextPackage
```

### 5.2 确定性排序

```text
FastRecallScore =
    hybrid_score
  + scope_match_bonus
  + kind_match_bonus
  + recency_or_validity_bonus
  + authority_bonus
  + explicit_pin_bonus
  - superseded_penalty
  - stale_penalty
  - duplicate_penalty
```

原则：

- authority / scope / validity 的优先级高于“历史被使用次数”；
- `superseded`、`deleted`、无权限、过期内容默认不参加普通召回；
- usage 如未来记录，只是低权重、衰减型 feedback，不得成为类似 UPSP heat 的自我强化真源；
- 同一 source lineage 只保留当前有效 revision，避免旧版本复活。

### 5.3 预算契约

Fast Recall API 应显式接收或拥有服务端默认预算：

```text
card_limit
candidate_limit
max_evidence_items
max_evidence_chars
max_governance_chars
```

预算不是 UI 提示，而是 storage read、排序和最终 ContextView 都必须执行的硬上限。

---

## 6. API 与模块边界

### 6.1 公开 API

默认路径：

```http
POST /v1/recall
```

```json
{
  "query": "Garden 对快速记忆召回的架构决策是什么？",
  "scope": {"project": "garden", "workspace": "C:/Users/Administrator/Desktop/garden"},
  "budget": {"cards": 8, "evidence_chars": 4000}
}
```

高级路径：

```http
POST /v1/recall/deep
```

Deep API 必须显式表达深挖意图或由宿主工具调用；不得在普通 `POST /v1/recall` 内静默触发 LLM planner。

### 6.2 Garden 模块建议

```text
garden/internal/recall/
├── fast.go            # 默认 Fast Recall orchestration
├── types.go           # Card / Evidence / Projection / ContextView DTO
├── rank.go            # deterministic rank + dedupe
├── policy.go          # scope/access/status/validity filter
├── projection.go      # Laputa projection adapter/cache
├── evidence.go        # bounded evidence reader
├── cache.go           # projection/card cache policy
└── deep_adapter.go    # 适配现有 internal/rag
```

当前 `garden/internal/rag/` 保留为 Deep Recall 实现，不继续承担 Fast Core 的主语义。

### 6.3 Mentle facade 演进

保留当前 `Retrieve()` 作为迁移兼容接口；新增：

```go
SearchCards(ctx, SearchCardsRequest) ([]MemoryCard, error)
ReadEvidence(ctx, ReadEvidenceRequest) ([]EvidenceFragment, error)
```

`SearchCards()` 只读取索引字段与短预览；`ReadEvidence()` 才读取正文。访问控制应在全文跨越 Mentle facade 边界前完成。

---

## 7. Session ingest 的边界

现有 ingest 的强项是：WAL、幂等、异步 worker、失败记录与 session/event hash。应保留。

但当前行为：

```text
session content → 空白压平 → 最多保留末尾 16KiB → session_digest
```

只能被定义为可靠的归档候选，不应被误称为已完成的语义记忆。

建议拆为：

```text
Phase 1：SessionSourceArtifact
- 原始材料 / hash / workspace / session / occurred_at / retention
- 默认不参加普通检索

Phase 2：DerivedMemory
- digest、claims、decisions、preferences、task/artifact links
- 每项保留 provenance → SourceArtifact
- 只有验证后的派生项进入默认 SearchCards
```

自动提取失败时，原始来源仍可追溯；提取成功时，规范记忆可被版本、冲突和 supersede 机制治理。

---

## 8. 非目标与明确拒绝项

本 ADR 不做：

- UPSP 式 Round / Frame / cleanup settlement；
- UPSP 式 STM/LTM heat 作为召回真源；
- 每次 Fast Recall 创建持久 `RecallActivity`；
- 让 Agentic RAG 成为普通检索默认路径；
- 把完整 Laputa snapshot 或 `memory_md/history_md` 全段注入上下文；
- 由 LLM 决定基础读取权限、scope 或硬治理约束；
- 让 Garden API 暴露 Mentle 的 room、wing、top_k、embedding provider 或 pipeline 名称。

`RecallActivity` 仍可在未来存在，但只适用于跨步骤取证、报告、深研究或高风险操作；Fast Recall 应保持函数式、无状态、低开销。

---

## 9. 分期实施顺序

### P0：基准与边界冻结

1. 记录现有 `/v1/context/resolve` 的输入、输出、延迟和召回质量基线。
2. 定义 Fast Recall 的 fixture corpus：scope、superseded、deleted、版本、权限、长文本、空命中。
3. 明确所有 SLO 是目标而非现状宣称。

### P1：Mentle Card / Evidence 分离

1. 设计并实现 `MemoryCard`、`EvidenceFragment`、`SearchCards`、`ReadEvidence`。
2. 保留旧 `Retrieve()` 兼容层。
3. 对 version / status / validity / provenance 做 retrieval-time 过滤。

### P2：Garden Fast Recall Core

1. 新增 `internal/recall`。
2. 实现 `GovernanceProjection` 与缓存失效。
3. 实现确定性 filter/rank/dedupe 与 bounded evidence read。
4. 增加 `POST /v1/recall`。
5. 让旧 `/v1/context/resolve?mode=basic` 迁移性地转发到 Fast Recall。

### P3：Deep Recall 适配

1. 将 `agentic_recall_v1` 明确为 `/v1/recall/deep`。
2. Deep 路径接收 Fast Recall 的 candidate/evidence seed。
3. planner、KG、timeline 只在明确 capability / budget 下运行。
4. Deep 失败自动退回 Fast Recall，并标注 degraded/warnings。

### P4：Session 语义摄取与治理

1. 引入 SourceArtifact。
2. 逐步增加 digest / claim / decision 提取。
3. 引入 superseded、effective validity、lineage 与审计。
4. 再讨论自动冲突处理阈值；不得先做物理删除。

---

## 10. 目标 SLO 与验收

以下是目标，必须通过真实 benchmark 验证后才能对外宣称：

| 指标 | 目标 |
|---|---:|
| warm GovernanceProjection | P95 ≤ 5ms |
| Mentle SearchCards | P95 ≤ 80ms |
| Garden filter/rank | P95 ≤ 10ms |
| bounded ReadEvidence | P95 ≤ 40ms |
| Fast Recall total | P95 ≤ 150ms（不含宿主网络传输） |
| Mentle 不可用时治理降级 | P95 ≤ 30ms |
| Deep Recall | 独立预算，不与 Fast SLA 共用 |

至少新增以下测试类型：

```text
- Fast Recall warm/cold benchmark
- 10k / 100k cards 下的 P50/P95/P99
- scope / access / status / revision / validity 过滤
- superseded / deleted 不复活
- 长正文不会在 SearchCards 阶段读取
- evidence char/item budget 强制生效
- Mentle unavailable 时仅 governance 的 graceful degradation
- Deep Recall 失败回退 Fast Recall
- citation 指向 memory id + revision + locator
```

---

## 11. 最终原则

> **Garden 是先快后深的 MemoryOS。**
>
> 默认路径用缓存、索引和确定性策略快速返回最小可信记忆；只有宿主明确请求研究、追溯或复杂推理时，才升级到 Agentic RAG、图谱和多步证据扩展。
>
> Laputa 决定什么可信、可读、可写；Mentle 保存和发现记忆；Garden 负责把治理、候选与最小证据组织成可丢弃、可重建的当前上下文视图。
