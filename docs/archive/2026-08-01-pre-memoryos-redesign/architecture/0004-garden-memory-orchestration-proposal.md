# ADR-0004: Garden 外挂记忆编排与 Skill 治理提案

> **Status**: proposed — 待松本评审  
> **Date**: 2026-07-15  
> **范围**: session 自动整理、记忆 CRUD、冲突/版本治理、分级查询、报告、记忆注入、高级能力 Skill 化  
> **讨论记录**: [`0004-garden-memory-orchestration-discussion.md`](./0004-garden-memory-orchestration-discussion.md)

---

## 0. 给松本的摘要

本提案建议把 Garden 定义为外挂记忆系统的**唯一对外入口与编排层**：

- Hermes 保留完整记忆增删改查，但不理解 Mentle 的 wing、room、向量、BM25、KG 等内部结构。
- session 结束后由宿主通过 HTTP 提交给 Garden，Garden 自动压缩、提取关键事实、识别冲突并治理写入。
- 普通上下文由 Garden 自动做 basic retrieval；只有复杂问题才由 Hermes 调用一个最小输入的高级查询工具。
- Pipeline 是 Garden 的受控执行引擎；Skill 是可版本化的高级能力包；Laputa 负责能力与写权限治理。
- 日报、周报、月报使用独立 report pipeline，按时间窗口和 source lineage 生成，避免重复压缩。
- Mentle 保存规范记忆；Laputa 保存权威治理规则、压缩投影和报告；Garden 负责最终裁决、审计和对外 API。

核心原则：**保留 CRUD，收口复杂性；保留高级能力，减少 Hermes 工具输入；自动覆盖采用可追溯失效，不做静默物理删除。**

---

## 1. 背景与当前实现

### 1.1 当前模块职责

| 模块 | 当前主要能力 |
|---|---|
| Garden | HTTP、按 key 前缀路由的 CRUD、`agentic_recall_v1` 查询管道 |
| Mentle | 向量/BM25 混合检索、wing/room、KG、timeline、extractor/miner 等内部能力 |
| Laputa | 14 section、写权元数据、rhythm 报告、wakeup 投影 |

Garden 当前公开 CRUD、context resolve 和 pipeline 观测接口，见 [`garden/internal/server/server.go`](../../garden/internal/server/server.go)。记忆写入通过 facade 直接保存调用方给定的 content，见 [`mentle/facade/crud.go`](../../mentle/facade/crud.go)。

### 1.2 已经实现的部分

- Garden 已是 Laputa 与 Mentle 的统一 HTTP 入口。
- `/v1/context/resolve` 已能混合 governance、Mentle hybrid、KG、timeline。
- 查询管道已有 planner、refinement、去重、排序、治理过滤和证据引用。
- Mentle 内部已有 extractor、conversation miner、room detector、entity detector 等可复用零件。
- Laputa 已有 identity、commitment、preferences、memory/history、daily/weekly/monthly 等 section。

### 1.3 主要缺口

- Garden 没有 session ingest API，也不会自动整理 session。
- facade 写入不做摘要、关键事实提取、冲突检测、版本替代或治理审批。
- Garden 缺少正式 `PATCH`，删除只认识精确 key。
- 自动更新没有 `superseded_by`、时间有效性、来源和审计语义。
- 当前 `WriteAuthority` 是 registry 元数据，`SetSection` 仍可直接写 store。
- 报告从完整 Laputa snapshot 生成，会读到旧日报/周报/月报，存在递归压缩与重复内容风险。
- Garden 没有正式区分 basic 与 advanced retrieval。
- wakeup 注入和 Garden context resolve 尚未统一。

---

## 2. 决策目标

1. Hermes 只依赖稳定、少参数的 Garden API/tool，不直接依赖 Mentle MCP 工具。
2. 每个 session 至少形成一条规范 `SessionDigest`，并可携带结构化 claims。
3. 所有记忆 mutation 最终经过 Garden 治理、版本和审计。
4. 保留外挂记忆库应有的完整增删改查。
5. Basic retrieval 默认快速稳定；Agentic RAG 作为按需高级能力保留。
6. Pipeline 与 Skill 分工明确，高级功能可独立版本化、授权、禁用和降级。
7. 报告不把既有压缩结果无边界地再次压缩。
8. 现有 CRUD 和 `context/resolve` 提供向后兼容迁移期。

---

## 3. 目标架构

```text
Hermes / Host
  ├─ lifecycle: POST /v1/sessions
  ├─ CRUD tools: /v1/memories
  └─ garden_search(query): POST /v1/context/resolve
                         │
                         ▼
                    Garden HTTP
                         │
       ┌─────────────────┼───────────────────┐
       ▼                 ▼                   ▼
 session_ingest_v1  memory_mutation_v1  context_resolve_v2
       │                 │              basic / advanced
       └─────────┬───────┘                   │
                 ▼                           │
            Mentle Facade ◄──────────────────┘
      canonical memory / hybrid / KG / timeline
                 │
                 ├─ projection update ─→ Laputa sections
                 └─ event stream ──────→ report_generate_v1

Garden Skill Registry
  ├─ advanced-retrieval
  ├─ conflict-resolution
  ├─ daily/weekly/monthly-report
  └─ personality-projection

Laputa Governance
  └─ skill capability、写权、外部上下文、source/room policy
```

### 3.1 职责边界

| 组件 | 负责 | 不负责 |
|---|---|---|
| Hermes | 发起 CRUD 与高级查询；消费上下文 | room/wing、模型、KG、pipeline 步骤 |
| Garden | HTTP、pipeline、skill registry、冲突裁决、权限、审计、模式选择 | 向量引擎细节 |
| Mentle facade | 规范记忆持久化、索引、检索、KG/timeline、原子 mutation | 决定什么信息应覆盖什么信息 |
| Laputa | 权威规则、人格/记忆投影、报告、Skill capability policy | 原始 session 主存、外挂记忆主索引 |

---

## 4. 对外 API 与 Hermes 工具面

### 4.1 记忆 CRUD 必须保留

```http
POST   /v1/memories
GET    /v1/memories/{id}
PATCH  /v1/memories/{id}
DELETE /v1/memories/{id}
GET    /v1/memories
```

Hermes native tools 使用专用 adapter 发 HTTP，不提供通用 curl/http 工具：

| Tool | 必需输入 | Garden 推断的内容 |
|---|---|---|
| `garden_memory_create` | `content` | source、session、scope、分类、ID |
| `garden_memory_get` | `id` | — |
| `garden_memory_update` | `id`, `content` | version、reason/source |
| `garden_memory_delete` | `id` | 默认 soft delete |
| `garden_memory_list` | 无 | 默认 active、分页上限 |

高级字段可以作为 HTTP API 的可选参数存在，但不进入默认 native tool schema。

### 4.2 Session 自动提交

```http
POST /v1/sessions
```

这由 Hermes 宿主 lifecycle 在 stop/precompact/session-end 调用，不作为模型工具。请求包含 `session_id`、messages/content、workspace 和时间信息。

### 4.3 高级查询保持一个最小工具

Hermes 只看到：

```json
{"query":"为什么 Laputa 最后选择 A？"}
```

`garden_search(query)` adapter 固定通过 HTTP 请求 advanced retrieval。它不接受 wing、room、top_k、planner、model、pipeline 名称等参数。

Basic context 由宿主自动预取，不消耗模型工具选择；高级查询只有在上下文不足或问题复杂时由 Hermes 主动调用。Garden 仍负责最终 capability gate 和 fallback。

### 4.4 Report 读取

```http
GET /v1/reports/latest?cadence=daily|weekly|monthly
```

该接口只返回派生报告，不替代通用 memory query。

---

## 5. 规范记忆模型

建议每个 session 至少保存一条 `SessionDigest`，避免把整段对话拆成大量无意义碎片；同时在同一记录内保留结构化 claims，支持冲突与时间治理。

```json
{
  "id": "mem_01...",
  "kind": "session_digest",
  "content": "本次确定 Laputa 采用 A 架构。",
  "claims": [
    {
      "subject": "laputa",
      "predicate": "selected_architecture",
      "value": "A",
      "scope": "project:garden",
      "confidence": 1.0
    }
  ],
  "status": "active",
  "source": {"type":"session","session_id":"..."},
  "valid_from": "...",
  "valid_to": null,
  "supersedes": ["mem_B", "mem_C"],
  "created_at": "...",
  "updated_at": "..."
}
```

Mentle 的 wing/room 可以继续作为内部检索元数据，但不属于公共记忆契约。

---

## 6. Session Ingest Pipeline

`session_ingest_v1`：

```text
validate envelope
→ normalize transcript
→ summarize into one SessionDigest
→ extract claims/entities
→ classify internal metadata
→ retrieve possible conflicts
→ invoke conflict-resolution skill when needed
→ governance decision
→ atomic facade mutation
→ update KG/timeline
→ update selected Laputa projections
→ audit trace
```

### 6.1 “每条都进 facade”的解释

- 每个被接受的 session 都形成至少一条 facade 记录。
- extractor 可以产生多个 claims，但不要求每个句子都形成一个独立 drawer。
- 原始 transcript 可按隐私/保留策略保存为 source artifact，不默认参与普通检索。
- 自动提取失败时仍保存 `SessionDigest` 或明确记录 rejected/degraded 状态，不能静默丢失。

---

## 7. 增、改、删的治理

### 7.1 自动新增与覆盖

“最新 A 替代旧 B/C”不能简单物理删除。应先判断：

- B/C 是正式旧决策：标记 `superseded`，设置 `valid_to` 和 `superseded_by=A`。
- B/C 只是讨论过的候选：保留为 `alternative`，不视为冲突。
- 新信息置信度不足：进入 proposal/review，不自动覆盖。
- 用户明确确认：以 user authority 执行。

普通查询默认只返回 active；历史/决策演变查询可以返回 superseded。

### 7.2 用户修改

`PATCH` 创建新 version 或 revision，不静默覆盖。需要同步：正文、embedding、BM25、KG/timeline、投影和审计。

### 7.3 用户删除

- 默认 DELETE 是 soft delete/tombstone，立即从普通查询隐藏。
- `purge=true` 仅接受明确用户授权，并清理正文、索引、KG、缓存及派生投影。
- 自动冲突消解不得触发 purge。

### 7.4 是否走管道

所有 mutation 都走 `memory_mutation_v1`，但步骤可按风险缩短：

- 精确 ID 的用户删除/修改：不需要 LLM，仍需权限、原子提交与审计。
- 自动覆盖：需要 conflict skill、governance gate 和可回滚提交。

---

## 8. 分级查询与 Agentic RAG

### 8.1 Basic Retrieval

默认自动注入路径：

```text
selected Laputa projections
→ Mentle hybrid top-k
→ exact-content dedupe
→ governance filter
→ extractive ContextPackage
```

不调用 planner、KG/timeline refinement 或 LLM 总结，目标是低延迟、稳定、可预测。

### 8.2 Advanced Retrieval

由 Hermes 显式调用 `garden_search(query)`：

```text
planner
→ multi-query hybrid retrieval
→ KG/timeline expansion
→ at most one refinement round
→ rerank/dedupe
→ governance filter
→ cited ContextPackage
```

现有 `agentic_recall_v1` 保留为实现基础。OpenAI planner 失败或 governance 禁止 LLM 时退回 RulePlanner/basic，不让查询整体失败。

### 8.3 为什么不暴露 mode/room/top_k

Hermes 的任务是表达问题，不是管理检索基础设施。模式、预算、房间和 capability 由 Garden 根据 Skill manifest、Laputa policy 和系统配置决定；工具 schema 越小，模型误用和耦合越少。

---

## 9. Pipeline 与 Skill 的关系

### 9.1 Pipeline

Pipeline 是 Garden 核心执行机制，负责：

- step 顺序、超时、取消和重试；
- 幂等、部分失败和 fallback；
- trace、审计和运行查询；
- capability gate；
- 原子 mutation 的 prepare/commit/rollback。

### 9.2 Skill

Skill 是高级能力的版本化声明与实现包，例如：

```yaml
name: advanced-retrieval
version: "1"
input_schema: query-only
capabilities: [llm, hybrid, kg, timeline]
pipeline: agentic_recall_v1
fallback: basic-retrieval
output_schema: context-package-v1
```

建议的首批 Skills：

- `advanced-retrieval`
- `conflict-resolution`
- `daily-report` / `weekly-report` / `monthly-report`
- `personality-projection`

### 9.3 安全边界

Skill 可以检索、分析、生成 mutation proposal，但不能绕过 Garden 直接写 Mentle/Laputa。所有写入必须回到 mutation pipeline，通过 authority、审计和原子提交。

Laputa policy 控制 Skill 能否使用 `llm`、`hybrid`、`kg`、`timeline`，能读取哪些 source，以及能提出哪些 section mutation。

---

## 10. Report Pipeline

日报、周报、月报不再读取完整 snapshot 作为唯一输入，而读取时间窗口内的 canonical memory/event：

```text
daily:   last_daily_watermark → now
weekly:  week window canonical events
monthly: month window canonical events
```

每份报告保存：

```json
{
  "cadence": "weekly",
  "window_start": "...",
  "window_end": "...",
  "source_ids": ["mem_1", "mem_2"],
  "source_hash": "...",
  "summary": "...",
  "generated_at": "..."
}
```

规则：

- 相同 `source_hash` 不重复生成。
- 报告默认写入 Laputa daily/weekly/monthly，不重新灌回 Mentle。
- 周报/月报若引用下级报告，只作为 secondary hint，仍以 canonical source IDs 去重。
- report pipeline 只允许 `report_system` authority 写目标 section。

---

## 11. 人格治理与记忆注入

Laputa section 需要明确三类语义：

1. **权威治理规则**：commitment、能力策略、外部上下文规则；不是记忆摘要。
2. **压缩投影**：identity、preferences、memory/history 的派生视图；必须记录 source IDs/revision。
3. **派生报告**：daily/weekly/monthly；不得反向成为无来源的事实。

Garden 统一组装注入：

```text
system governance block（Laputa 权威规则）
+ stable personality projection（Laputa）
+ turn-specific basic context（Mentle + governance）
+ optional advanced search result（Hermes 显式请求）
```

旧 wakeup 行为在迁移期保留，最终由 Garden context/bootstrap API 接管。

---

## 12. 迁移顺序

### Phase A：补齐 facade mutation 基础

- 定义 canonical memory、status、version、source、lineage。
- facade 增加 Update、soft delete、batch mutation、WAL/索引一致性。
- 保持现有 Write/Read/List/Forget 兼容。

### Phase B：补齐 Garden CRUD 与权限

- 新增 `PATCH`。
- 支持 Garden 生成 ID，兼容旧 `memory:` key。
- authority、审计、幂等 key、错误码。

### Phase C：Session ingest

- 新增 `/v1/sessions` 和 `session_ingest_v1`。
- 复用 Mentle extractor/miner 思路，但通过 facade 公开稳定能力。
- 先只生成 digest；冲突 skill 在 feature flag 下灰度。

### Phase D：Basic/Advanced 分流与 Skill Registry

- 抽取 shared retrieval steps。
- basic 自动预取；`garden_search(query)` 固定 advanced。
- 注册 Skill manifest、capability policy、版本和 fallback。

### Phase E：Report 与注入统一

- report 使用 window/source lineage。
- Garden 接管 context bootstrap。
- 旧 Laputa CLI/wakeup/report 路径进入兼容期。

---

## 13. 验收标准

- Hermes 无需传 wing、room、top_k、model 或 pipeline 名称。
- Hermes 能通过 Garden 完成记忆增、查、改、删、列举。
- 每个成功 ingest 的 session 至少形成一条可追溯 digest。
- A 替代 B/C 时，旧正式决策变为 superseded；候选方案不会被误删。
- 用户 delete 默认立即从普通查询消失，purge 需要显式授权。
- basic retrieval 不依赖外部 LLM；advanced 失败可降级。
- advanced 输出包含 trace、evidence 和有效 citation。
- 日/周/月报告不会因相同 source set 重复生成。
- Skill 无法绕过 mutation pipeline 直接写入。
- 现有四组测试入口继续通过，现有 CRUD/context API 在迁移期兼容。

---

## 14. 待松本拍板

1. SessionDigest 是否作为“一 session 一主记录”的强约束，还是允许大 session 拆成少量主题 digest？
2. 原始 transcript 是否保存；保存在哪里、保留多久、是否默认参与检索？
3. 自动 supersede 的最低置信度和哪些 predicate 允许自动处理？
4. 用户明确删除默认 soft delete 是否可接受，purge 是否需要二次确认？
5. Hermes 的 basic context 由每 turn 自动注入，还是只在 session start/明确需要时注入？
6. `garden_search(query)` 是否永远 advanced，还是 Garden 可以在明显简单时降为 basic？
7. Skill manifest 由 Garden 管理，还是由 Laputa section 作为权威 registry？
8. 旧 Mentle MCP tools 和 Laputa wakeup/CLI 的退役时间表。

---

## 15. 非目标

- 本 ADR 不决定具体 LLM provider。
- 本 ADR 不要求 Hermes 直接理解 Mentle taxonomy。
- 本 ADR 不把所有历史数据立即迁移为 claims。
- 本 ADR 不允许自动冲突消解执行不可逆物理删除。
- 本 ADR 不删除现有 Agentic RAG、FallbackPlanner 或 OpenAIPlanner。

