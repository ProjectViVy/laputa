# ADR-0006: Laputa Lifecycle and Mentle Index Contract

> **Status:** proposed - 待松本评审
> **Date:** 2026-07-31
> **Scope:** 明确 MemoryOS 以 Agent 人格与记忆为核心，同时管理个人工作信息；定义 Laputa authority files、STM working set 与 Mentle 索引职责。
> **Does not decide:** 自动提炼、自动 STM->LTM 提升、Deep Recall、向量模型、KG 策略、Autodream 的具体实现。

---

## 0. Decision

Laputa 定义 Agent 的人格、记忆和治理边界；Mentle 保存和组织可检索材料；Garden 编排来源、召回和一次性上下文。MemoryOS 面向个人全部工作信息，但它的中心不是资料库，而是一个具有连续人格、受治理记忆和工作上下文的 Agent。

关键决定：

```text
LONGMEM.MD 只保留 LTM 的权威索引，不保存长期记忆正文。
```

长期记忆正文、来源、版本、有效期、证据片段和检索索引都留在 Mentle。Laputa 的 LTM index 只保留足以治理和人工审计的最小条目，并通过稳定 ID 指向 Mentle。

这个边界也适用于外部工作数据。未来 Mentle 可以接入 Obsidian 等外部记事本，对 vault 内容做全量解析、分块、分类和索引。此时 Laputa 的目标不是只记住“对话中发生过什么”，而是成为能够定位一个人工作领域内任意信息的个人记忆框架：对话、笔记、代码、文档、项目材料、报告和派生关系都可以成为可检索对象。

```text
外部来源保持原文权威
  -> source adapter / watcher
  -> Mentle material snapshot + chunks + taxonomy + index
  -> Laputa governance projection / scope / policy
  -> bounded recall and explicit evidence read
```

外部来源接入至少要保留 `source_uri`、`source_revision`、`content_hash`、`observed_at` 和同步状态。Mentle 保存的是可追溯的材料快照与索引，不取得 Obsidian 等外部系统的原文写权限。

```text
Laputa
  MEMORY.MD              STM 的权威检查点 / working-set summary
  LONGMEM.MD             LTM 的权威索引
  SOUL / THOUGHTS        反思域
  USERPROFILE / IDENTITY / TASK
                         Agent 的权威上下文域

STM Runtime Cache
  hot_context            当前轮次与当前 session 的极短期上下文
  working_set            当前项目/任务的活跃材料引用与摘要
  recent_activity        最近 session/event 的可重建记录
  compressed_segments    从活动记录压缩出的 STM segment

Mentle
  memory records         受索引的正文、来源、版本、有效期、证据
  cards                  非全文候选
  evidence               按需全文或范围材料
```

`MEMORY.MD` 不是 STM 的全部内容，而是 STM 的**权威检查点**：它体积小、可审计、跨进程可恢复。STM Runtime Cache 则是性能和连续性的运行时层，允许 STM 同时存在多个窗口、多个 scope 和不同 TTL。

这不是让 Mentle 取代 Laputa 的 LTM 治理。Mentle 可以保存一条 LTM 的完整材料，但它不能自行决定一条材料具有 LTM 身份；该身份必须存在于 Laputa 的 `LONGMEM.MD` 索引中。

---

## 1. Authority File Lifecycle Map

| Laputa file | 生命周期定位 | Mentle 是否索引 | 默认上下文 | 正文位置 | 写入者 |
|---|---|---:|---|---|---|
| `MEMORY.MD` | STM 的权威检查点；近期活动和 working-set 摘要 | 是，按 segment 索引 | 当前 scope 内可用，严格预算 | Laputa 保存可恢复摘要；Runtime Cache 保存热数据；Mentle 留可检索副本/证据 | Agent + Autodream |
| `LONGMEM.MD` | LTM 的权威索引，不是正文库 | 索引条目本身可检索；正文在 Mentle | 不默认整段注入 | Mentle | Agent，经治理路径 |
| `SOUL.MD` | 长期反思人格域 | 否，第一版禁止普通检索 | 仅 wakeup / 显式受权投影 | Laputa | Autodream，用户审计 |
| `THOUGHTS.MD` | 反思草稿、假设、失败记录 | 否，第一版不进普通召回 | 否 | Laputa | Autodream |
| `USERPROFILE.MD` | 用户权威期望 | 否 | 按 scope 投影 | Laputa | User；Agent 可提案 |
| `USER.MD` | Agent 对用户的观察 | 否，先不自动索引 | 按 policy 投影 | Laputa | Agent |
| `IDENTITY.MD` | Agent 身份与角色 | 否 | wakeup / 最小治理投影 | Laputa | User 初始化，随后 Agent |
| `TASK.MD` | 当前与长期任务 | 可选，第二阶段 | 当前项目或显式任务 scope | Laputa | User + Agent |
| 外部工作资料（如 Obsidian） | 外部来源的原文与版本 | 是，经 adapter 全量索引 | 按 query、scope、source policy 按需召回 | 外部来源；Mentle 保存 canonical copy / chunks / index | 外部来源 owner；adapter 只读同步 |

`MEMORY.MD`、`LONGMEM.MD` 和已接入的外部工作资料是第一阶段进入 Mentle 检索链的来源。其他 Laputa 文件先保持权威投影或反思域，避免把人格、用户约束和探索笔记误变成一般语义材料。

---

## 2. STM Runtime Cache

```text
STM Runtime Cache 不是另一个权威记忆库，而是 MEMORY.MD 的可恢复运行时投影：

hot_context
  当前轮次、当前调用链，TTL 最短，进程崩溃可丢

working_set[scope]
  当前项目/任务的活跃卡片、证据引用、未完成线索

recent_activity[session]
  最近 session/event 的有序活动摘要，可重建 working_set

compressed_segments
  session-end / curator 产生的稳定 STM segment，回写 MEMORY.MD 并索引到 Mentle
```

写入顺序建议：

```text
activity event
  -> append-only recent_activity
  -> update working_set / hot_context
  -> bounded session compression
  -> atomic MEMORY.MD checkpoint
  -> Mentle indexing
```

只有 `MEMORY.MD checkpoint` 是跨进程恢复时的权威最小状态。Cache 丢失时，可以从 checkpoint 和 Mentle material 重建；Mentle 索引延迟或失败时，不应阻塞 Agent 当前轮次，但必须保留待重试状态。

STM Cache 的每个条目至少需要：`id`、`scope`、`session_id`、`kind`、`content_ref`、`summary`、`created_at`、`expires_at`、`priority`、`source_ref` 和 `checkpoint_state`。其中 `content_ref` 可以指向 Mentle material，不应把同一份长正文复制到多个缓存层。

因此 STM 不是单一文件，也不是无边界的向量检索结果，而是：

```text
STM = checkpoint + runtime cache + recent activity + Mentle searchable segments
```


## 3. STM -> LTM Flow

```text
session / activity
  -> recent_activity + working_set
  -> compressed working summary
  -> MEMORY.MD checkpoint          STM authority
  -> Mentle STM material index
  -> explicit promotion proposal
  -> LONGMEM.MD index entry        LTM authority
  -> Mentle LTM material record    full content + evidence
```

第一阶段不实现自动 promotion。STM 何时可成为 LTM 是治理判断，不是相似度、热度或使用次数的函数。

至少要满足一个显式来源才允许创建 `LONGMEM.MD` 索引：

- 用户确认；
- Agent 发起的可审计 promotion proposal 被接受；
- 受限的 report/Autodream 流程产出并经过其 authority gate。

---

## 4. LONGMEM.MD Index Contract

每条索引只表达“这是被治理认可的长期记忆，以及如何定位其材料”，不重复正文。

```yaml
- id: ltm_01J...
  title: Garden Fast Recall uses cards before evidence
  kind: decision
  scope: project:garden
  mentle_ref: mem_01J...
  source_ref: session:<session-id>#<event-id>
  status: active
  valid_from: 2026-07-31
  supersedes: [ltm_01H...]
  updated_at: 2026-07-31T00:00:00Z
```

字段规则：

| 字段 | 作用 | 不允许承载 |
|---|---|---|
| `id` | Laputa 稳定的治理 ID | 物理存储路径 |
| `title` | 人可读、短标题 | 长摘要或正文 |
| `kind` | `decision` / `fact` / `preference` / `knowledge` | 任意自由分类 |
| `scope` | 适用工作域 | Mentle wing/room 实现细节 |
| `mentle_ref` | Mentle canonical material ID | 搜索参数或 embedding 信息 |
| `source_ref` | 可追溯来源指针 | 原始 transcript 副本 |
| `status` | `active` / `superseded` / `deleted` | STM 临时状态 |
| `valid_from` | 生效点 | 无界 history |
| `supersedes` | Laputa LTM 语义谱系 | 全部证据关系 |

`LONGMEM.MD` 的每项都必须有有效的 `mentle_ref`。反过来，Mentle 的普通材料不自动获得 LTM 地位；没有对应 active index 的材料只能是 STM、artifact 或候选材料。

---

## 4. Mentle: Categorized Memory Substrate

Mentle 的准确定位不是“LTM 正文仓库”，而是**受 Laputa 治理的、分门别类保存记忆材料并提供检索基础设施的土壤层**。

它回答的是：材料放在哪一类、与什么主题相邻、来自何处、何时有效、如何被找到、正文如何被按需读取。它不回答：材料是否是用户承诺、是否具有 LTM 身份、是否可以自动升级、是否应该写入人格文件。

三个正交坐标必须分开：

```text
Lifecycle: stm | ltm | artifact | reflective
Authority:  Laputa file / actor / policy decision
Taxonomy:   collection -> wing -> room -> optional hall
```

- `Lifecycle` 决定默认召回、TTL、promotion 和保留策略。
- `Authority` 决定是否允许创建、更新、提升或投影。
- `Taxonomy` 只服务于分类、局部检索、管理浏览和 Deep Recall 扩展；它不决定 LTM 身份，也不替代 scope/access filter。

建议把 Mentle 的 palace taxonomy 由自由 wing/room 输入收口为 Garden 写入时生成的稳定分类：

| Collection | 内容 | 默认角色 | 示例 wing / room |
|---|---|---|---|
| `working` | `MEMORY.MD` 的 session/活动片段 | STM | `project/garden`、`conversation/current` |
| `knowledge` | 被 `LONGMEM.MD` index 引用的长期材料 | LTM | `project/architecture`、`domain/memoryos` |
| `artifacts` | transcript、文档片段、代码证据、网页快照 | 非默认召回的来源材料 | `project/garden`、`source/code` |
| `reports` | daily/weekly/monthly 的派生报告 | report material | `rhythm/daily`、`rhythm/weekly` |
| `reflection` | Autodream 的受控中间材料 | 默认隔离 | `autodream/staging`、`autodream/review` |

`collection` 是 Garden/Laputa 的稳定语义；`wing/room/hall` 是 Mentle 内部分类轴。第一版至多要求 collection + wing + room，`hall` 只有在时间桶或来源批次能带来明确检索收益时才启用。

Mentle 应最大化利用已有能力，但按层分工：

| Mentle 特性 | 应用方式 | 不应用于 |
|---|---|---|
| wing / room taxonomy | 分类、局部 scope、可视化浏览、候选预筛选 | STM/LTM 判定、权限判定 |
| vector + BM25 + RRF | `SearchCards` 的混合候选发现 | 直接把全文交给宿主 |
| canonical catalog | material ID、source、revision、validity、状态、lineage | 取代 Laputa 的 LTM authority index |
| WAL / index jobs | material 写入、索引重建、故障恢复 | 代替 Garden 的跨层 transition audit |
| temporal KG | Deep Recall 的实体关系和 as-of 追溯 | 默认 Fast Recall 或自动事实确认 |
| palace graph / tunnels | 深挖时发现跨分类主题关联 | 普通 query 的无边界扩展 |
| extractor / miner | 产出 STM 候选、artifact metadata、分类建议 | 直接写入 `LONGMEM.MD` 或 `SOUL.MD` |
| diary / AAAK | Autodream / specialist 的反思材料 | 一般事实库或默认 prompt 注入 |

Mentle 不再以旧 L0-L3 作为顶层模型。其保留价值是：canonical records、分类 taxonomy、向量/BM25 hybrid search、WAL、来源、版本、有效期、KG 和按需正文读取。

第一阶段只需要两个公开读接口：

```go
type MemoryCard struct {
    ID        string
    Tier      string // stm | ltm
    Kind      string
    Title     string
    Preview   string // deterministic; bounded
    Scope     string
    Status    string
    Version   int
    SourceRef string
    Score     float64
}

SearchCards(query, scope, tier, limit) ([]MemoryCard, error)
ReadEvidence(ids, maxItems, maxChars) ([]EvidenceFragment, error)
```

约束：

- `SearchCards` 不返回全文。
- `ReadEvidence` 是唯一正文跨 Mentle facade 边界的接口。
- `LTM card` 必须能反查 `LONGMEM.MD` 的 active index。
- `STM card` 必须能反查 `MEMORY.MD` 的 source summary 或 session source。
- 每张 card 都带由 Garden 填入的 `collection`；宿主不传 Mentle 的 wing/room/hall。

Mentle 的旧 L0/L1/L2/L3 可在内部保留或逐步退役：它们是老的读取深度/表现层，不拥有 Laputa 的 authority file 语义。

---

## 6. Memory Activation (Heat) Policy

借鉴 CogniCore 的“衰减 + 共鸣 + 巩固”直觉，MemoryOS 保留**热度（activation / heat）**，但把它严格限制为可解释的召回和维护信号，而不是价值、真实性或治理权的替代品。

```text
Heat answers:      当前是否值得优先拿来复用？
Value answers:     这份材料是否有长期价值？
Truth answers:     这是否仍然有效、是否与新事实冲突？
Authority answers: 谁能把它声明为 LTM、修改或删除？
```

这四个问题必须由不同字段和流程回答。Heat 不得直接创建、提升、覆写、删除 `LONGMEM.MD` 条目，也不得写入 `SOUL.MD`、`USERPROFILE.MD`、`IDENTITY.MD` 或 `commitment`。

建议每个 Mentle material/card 独立维护一个**派生、可重建且有上限**的 activation record：

```text
base_heat          初始可见度；由 source class / explicit pin 决定
last_activated_at  上次有效使用时刻
heat_score         经过时间衰减后的短期活跃度
activation_events  user_confirmed_use | successful_task_use | explicit_pin
                   | source_updated | agent_retrieval_only
```

排序中可使用一个有界的 heat bonus：

```text
final_recall_score = semantic_relevance
                   + scope_match
                   + authority/status/validity gates
                   + bounded_heat_bonus
```

其中 gate 先于 rank：无效、超 scope、已 supersede、无权限或被 source policy 排除的材料，不能被 heat 拉回默认结果。

### 6.1 Heat event rules

| 事件 | Heat 影响 | 说明 |
|---|---|---|
| 用户确认/引用某条材料 | 强 | 有明确的人类信号 |
| 材料支持的工作结果被采纳 | 中 | 记录 outcome/source ref |
| 外部来源更新 | 中 | 表示需要重新注意，不等于事实确认 |
| Agent 仅检索到材料 | 极弱或零 | 防止自我强化回声室 |
| 自动重复召回 | 不累计 | 同一 session/window 去重 |
| 用户否定、source 删除、事实 supersede | Heat 清零/冻结 | 真值和 authority 优先 |

### 6.2 Decay and retention

- `hot_context`、`working_set`、`recent_activity` 可使用较快 decay 与 TTL；到期只影响 cache，不物理删除源材料。
- STM segment 可因低 heat 降低默认召回优先级，仍由 `MEMORY.MD` checkpoint、Mentle material 和 retention policy 决定是否保留。
- active LTM 的 heat 只影响同等相关候选的排序，不影响其 `LONGMEM.MD` authority status；低热 LTM 仍可在精确查询、Deep Recall 或显式浏览中找到。
- Obsidian 等外部 source 的低 heat 绝不触发删除。原文保留权归外部来源 owner；Mentle snapshot 的清理遵从 source retention / sync policy。

### 6.3 Heat-assisted lifecycle placement

Heat 不只是排序信号；它也是维持上下文健康的**生命周期调度信号**。它可以建议材料留在 STM、晋升为候选 LTM，或从默认召回面退出到 archive，但不能单独完成治理状态变更。

```text
hot / repeatedly useful / current scope
  -> STM working set

persistently useful across independent contexts
  -> LTM promotion candidate
  -> authority review
  -> LONGMEM.MD active index (only if accepted)

low heat / no active scope / still source-valid
  -> archive / cold Mentle material
  -> excluded from Fast Recall, retained for Deep Recall
```

这实现“人未必立刻想起罕见信息，但在多轮查找后可以重新找到”的能力：冷材料不等于错误或应删除。Fast Recall 只看热区和受限 active LTM；Deep Recall 可以逐轮扩大 source、time、taxonomy 和 archive 边界，直到找到可验证的 evidence 或明确说明未找到。

因此 lifecycle 采用 **heat + utility + authority + retention policy**，不是单纯 heat：

| 决策 | Heat 的作用 | 其他必要条件 |
|---|---|---|
| 保留在 STM | 高 heat 或当前 scope 有效 | TTL 未到、working_set 仍相关 |
| 生成 LTM candidate | 多个独立 session/project 中持续有效 | evidence 完整、无 conflict、scope 明确 |
| 创建 active LTM | 提供支持证据 | 用户确认或可审计 authority approval |
| 转入 archive | 低 heat、脱离 active scope | source 保留策略允许；不得抹掉 authority/source lineage |
| 物理删除 | 不由 heat 决定 | source 删除、retention expiry、用户删除或合规规则 |

Heat 可以触发一个**只读候选信号**，例如“该 STM material 在多个独立工作结果中被有效使用”。它最多进入 proposal inbox；只有用户确认或可审计 authority flow 才能创建/更新 `LONGMEM.MD` index。

### 6.4 Logical arbiter: temporal truth and relationship resolution

逻辑裁判不能是“新文本相似度更高，所以覆盖旧文本”。它必须判定**命题、主体、关系、时间区间、证据和状态**，把记忆从一条模糊文本变成可追溯的时态断言。

```text
assertion = subject + predicate + object + valid_from + valid_to
          + confidence + evidence_refs + status

status = active | superseded | retracted | contextual | proposed
```

对“喜欢小梅，但那只是某段时间的独木桥效应；后来对小王一见钟情并准备结婚”的正确结果不是删除小梅，也不是让两条关系同时作为当前偏好出现：

```text
A1: self --romantic_interest--> 小梅
    valid_from: T1
    valid_to: T2
    status: contextual / superseded
    context: 独木桥效应；非稳定偏好
    evidence: ...

A2: self --romantic_commitment--> 小王
    valid_from: T2
    valid_to: null
    status: active
    confidence: high
    evidence: 用户明确陈述 / 婚约计划
```

默认人格与关系投影只读取 `active` 且在当前时间有效的断言，因此不会再把小梅当作“当前喜欢的人”。但 A1 仍作为历史/反思证据留在 Mentle archive，只有在用户追问历史、关系演变或 Deep Recall 时才会被按预算读取。

裁判的输出只能是 proposal 或受审计 mutation plan，至少包括：冲突类型（replace / refine / coexist / retract）、受影响 assertion、有效期变更、supersedes 边、证据和置信度。对 `USERPROFILE.MD`、`commitment`、`SOUL.MD`、`IDENTITY.MD` 等权威域，裁判无权自动落盘；必须经对应 actor/authority gate。

这保留 Humanoid_memory_bank 中有价值的“时间衰减、重复巩固、冲突显式处理”方向，同时拒绝其直接以访问回写、热度衰减物理删除、LLM 自动覆写长期记忆的做法。

---

## 7. Progressive Recall Policy

MemoryOS 采用渐进召回，而不是一次性把 Mentle 搜索结果和外部资料拼入上下文：

```text
Fast Recall (default)
  1. 最小 Laputa governance projection
  2. hot_context + matching working_set
  3. active LONGMEM.MD 对应的 Mentle LTM cards
  4. 同 scope、未过期的 STM cards
  5. deterministic filter / rank / dedupe
  6. bounded ReadEvidence
  7. disposable ContextView

Deep Recall (explicit escalation)
  1. 保留 Fast Recall 结果与不足原因
  2. 扩大允许的 source / time / taxonomy 边界
  3. 受预算地调用 Mentle KG、timeline、palace graph 与跨来源 retrieval
  4. 对候选再执行 ReadEvidence
  5. 产出 ContextView + RecallTrace
```

Fast Recall 必须满足：无 LLM、低延迟、可缓存、可降级、结果确定；默认不触发 KG、timeline、palace graph 或无边界的外部资料扫描。

Deep Recall 必须由 Agent 或宿主显式升级，携带独立的时间、token 和 source 预算。它不能隐式写 Laputa、不能自动 promotion STM->LTM；输出必须保留 `RecallTrace`，至少记录 query、scope、触发原因、来源集合、过滤条件、候选 ID、已读 evidence、预算消耗和降级/失败状态。

这承接 UPSP 的可取设计：**候选发现 != 正文读取 != 最终上下文**。不引入其 heavyweight `ContextAssembler`、每轮 settlement、常驻大上下文 cache 或热度驱动的 LTM promotion。

---

## 8. Default Recall Policy

```text
1. 解析最小 Laputa governance projection
2. 查 active LONGMEM.MD 对应的 Mentle LTM cards
3. 只有同 scope continuation 才加入 MEMORY.MD 对应的 STM cards
4. filter: status, validity, scope, authority
5. rank/dedupe
6. bounded ReadEvidence
7. 生成可丢弃 ContextView
```

默认行为：

- LTM 优先。
- STM 仅限当前 scope、有效期内、明确允许的 continuation。
- `SOUL.MD` / `THOUGHTS.MD` 不进入普通检索。
- 不注入完整 `MEMORY.MD` 或完整 `LONGMEM.MD`。
- 不触发 LLM、KG 或 timeline；这些属于显式 Deep Recall。

---

## 9. MVP Acceptance

```text
- session-end 只更新 MEMORY.MD 与其 Mentle STM material，不产生 LTM index。
- 创建 LTM 时必须同时创建 LONGMEM.MD index + Mentle material reference；任一失败不得提交。
- 删除或 supersede LTM index 后，对应 Mentle material 不再出现在默认 LTM recall。
- SearchCards 的公共结果没有正文。
- ReadEvidence 受 item 和 character budget 限制。
- Fast Recall 只使用 hot_context、working_set、active LTM cards 和受限 STM cards；不得自动触发 KG、timeline、palace graph 或全量外部索引扫描。
- Heat 只能作为 bounded ranking/maintenance/lifecycle-placement signal；无效、超 scope、superseded 或无权限材料不能被 heat 重新激活。
- archive material 在 Fast Recall 不可见，但必须可由有预算的 Deep Recall 经 source/scope policy 重新发现。
- Logical arbiter 的关系/事实冲突输出必须包含时态、status、evidence 和 supersedes lineage；不得以热度或新文本直接覆写 authority files。
- Deep Recall 必须显式指定预算并产出 RecallTrace。
```

---

## 10. Deferred

- 自动将 `MEMORY.MD` 片段提升到 `LONGMEM.MD`；
- `SOUL.MD` / `THOUGHTS.MD` 的检索策略；
- LTM index 的 YAML/Markdown 精确语法和 migration；
- 多个 Mentle evidence material 指向同一个 LTM index；
- KG 和 timeline 如何为 LTM 提供派生证据；
- 旧 Mentle L0-L3 API 的兼容期与退役时间表。
