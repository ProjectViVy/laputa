# NEW-LAPUTA — 一体化设计文档

**作者**: 松本 (大湿)
**日期**: 2026-07-06
**状态**: 决策记录(DR) + 设计提案
**范围**: 桌面文档,非工程产物

---

## 目录

- [1. 现状快照](#1-现状快照)
- [2. 一体化真瓶颈](#2-一体化真瓶颈)
- [3. Garden 的最终决定](#3-garden-的最终决定)
- [4. Laputa 现状 vs memU 7 抽象](#4-laputa-现状-vs-memu-7-抽象)
- [5. 候选方案整理](#5-候选方案整理)
- [6. 待办与不做清单](#6-待办与不做清单)
- [7. 关键事实索引](#7-关键事实索引)
- [8. 时间线](#8-时间线)

---

## 1. 现状快照

### 1.1 项目结构(go 这边,2026-07-06 现场)

| 项目 | 路径 | 状态 | 备注 |
|---|---|---|---|
| Laputa (Go) | `~/Desktop/projects/laputa/` | 跑稳,PID 12996,uptime ~3h | 14 section JSON,HTTP :7373 |
| Mempalace (Go) | `~/Desktop/projects/mempalace-go-redis-v2/` | 二进制在,无进程 | MCP stdio,43 tool,data 在 `~/.mempalace/` |
| Diva (Rust) | `~/Desktop/morediva/agent-diva/` | 独立线,**当前不动** | 已经有 `agent-diva-laputa` + `memtle` crate |
| Hermes (Python) | `C:/Users/Administrator/AppData/Local/hermes/` | 当前 agent | 调 laputa :7373,不调 mempalace |

### 1.2 Laputa 14 section 状态表(实测)

| # | Section | 写权 | SchemaOwner | Status | 数据真实状态 |
|---|---|---|---|---|---|
| 1 | `01-identity` | agent_self | laputa | stable | 有数据(`_meta.updated_at: 2026-07-04T21:19:05Z`) |
| 2 | `02-relationship` | agent_self | laputa | stable | 有数据 |
| 3 | `03-commitment` | **user_only** | laputa | stable | 有数据 |
| 4 | `04-preferences` | agent_self | laputa | stable | 有数据 |
| 5 | `05-memory_md` | agent_self | laputa | stable | 有数据 |
| 6 | `06-history_md` | agent_self | laputa | stable | 有数据 |
| 7 | `07-daily` | **report_system** | report_system | stable | 实际是 tbd-like(没 active) |
| 8 | `08-weekly` | **report_system** | report_system | stable | 同上 |
| 9 | `09-monthly` | **report_system** | report_system | stable | 同上 |
| 10 | `10-journal_reflective` | tbd | tbd | **tbd** | 空 |
| 11 | `11-proposal_inbox` | tbd | tbd | **tbd** | 空 |
| 12 | `12-changelog` | tbd | tbd | **tbd** | **已激活** (Phase 2 写入 `{entries: [...]}`,2 个测试 entry) |
| 13 | `13-report_indexes` | tbd | tbd | **tbd** | 空 |
| 14 | `14-aaak_summaries` | tbd | tbd | **tbd** | 空,`summaries: []` |

**统计: 6 stable+agent_self + 1 stable+user_only + 3 stable+report + 5 tbd = 14**

### 1.3 Mempalace 43 tool 分类表(实测,纠正我之前 22 个的错)

| 类别 | 数量 | 工具名 |
|---|---|---|
| **CRUD 核心** | 4 | `add_drawer`, `delete_drawer`, `update_drawer`, `get_drawer` |
| **批量** | 1 | `batch_store` |
| **检索** | 4 | `search`, `hybrid_search`, `layer2_search`, `recall` |
| **KG 操作** | 6 | `kg_add`, `kg_invalidate`, `kg_query`, `kg_stats`, `kg_timeline`, `graph_add`, `graph_link` |
| **graph 遍历** | 3 | `traverse`, `navigate`, `list_rooms`, `list_wings`, `get_taxonomy` |
| **Wake 协议** | 3 | `wake`, `layer0_get`, `layer1_recall` |
| **diary / journal** | 3 | `diary_write`, `diary_read`, `journal` |
| **chunk 工具** | 2 | `split`, `compress` |
| **编码 (AAAK)** | 2 | `entity_encode`, `entity_decode` |
| **检查 / 状态** | 4 | `status`, `stats`, `health`, `check_duplicate` |
| **Mine** | 3 | `mine_project`, `mine_conversation`, `auto_save` |
| **WAL** | 1 | `wal_replay` |
| **系统** | 2 | `log`, `sync` |
| **备份** | 2 | `backup`, `restore` |

> **我之前一直说 "22 tool" — 错的。实际 43 tool(我对话里反复提的 22 是早期错记)。**

### 1.4 Laputa 现有 Go interface(go 这边已经现成)

```go
// SectionStore - 14 section CRUD interface
type SectionStore interface {
    Read(ctx, section SectionName) (map[string]any, error)
    Write(ctx, section SectionName, data map[string]any) error
    Patch(ctx, section SectionName, path string, value any) error
    Delete(ctx, section SectionName, path string) error
    List(ctx) ([]SectionName, error)
    Exists(ctx, section SectionName) (bool, error)
}
```

### 1.5 Mempalace 现有 Go interface(已现成)

| Interface | 文件 | 方法 |
|---|---|---|
| `search.Store` | `internal/search/searcher.go:13` | Search / Add / AddBatch / Delete / ListAll / Close |
| `search.Embedder` | `internal/search/searcher.go:22` | CreateEmbedding / CreateEmbeddings |
| `search.LlamaClient` | `internal/search/searcher.go:27` | CreateEmbedding |
| `hybrid.Store` | `internal/hybrid/searcher.go:18` | embed search.Store + BM25Search / BM25Index / BM25Remove |
| `miner.RoomDetector` | `internal/miner/miner.go:46` | (room 检测) |

> **关键事实: 两个项目都已经有"backend interface"抽象。Garden 不需要从零造 interface,直接复用。**

### 1.6 Hermes 当前怎么用 laputa

- 入口: `C:/Users/Administrator/AppData/Local/hermes/plugins/laputa/__init__.py`
- 协议: HTTP 调 :7373,4 步契约(`system_prompt_block` / `prefetch` / `sync_turn` / `on_session_end`)
- Fallback: builtin mirror `~/.hermes/memories/MEMORY.md` + `USER.md`(当 laputa down 时降级)
- 3 层降级模式: PRIMARY / DEGRADED / UNAVAILABLE
- 不调 mempalace(go 版),不调 garden(还没建)

### 1.7 Laputa 已发现的 bug / 隐患

| 项 | 现状 | 影响 |
|---|---|---|
| `internal/store/redis/` | 写了完整代码,**生产未启用** (-store flag 始终 file) | dead code, 增复杂度 |
| `7/5 早间 rhythm 重构` | 5:55–14:57 崩了 6 次,根因未确认 | rhythm 路径可能还有未发现 bug |
| `laputa-supervisor.cmd` | 用 `start /B` 不重定向 stderr | crash 时无现场 |
| `12-changelog` 写权 = tbd | agent 实际能写(user 协议未强制) | 治理边界被绕过 |
| `5 个 tbd section` | 14/15/... status 写 "tbd" 但 schema 设计 | 长期不激活会让 governance 残缺 |

---

## 2. 一体化真瓶颈

> **来源**: 大湿 2026-07-06 在对话中给出的 2 个真瓶颈 (贴给我看的事实表)

### 瓶颈 1:事实库分叉

```
~/.laputa/        ← Laputa 14 section JSON  (v2 时代, 14 个扁平文件)
~/.mempalace/     ← Mempalace SQLite + vectors.db + WAL  (v1 时代, palace graph)
```

**问题**:
- Laputa 完全**不读** mempalace
- Mempalace 也**不读** laputa
- 谁是 source of truth? 不清
- 同步靠 `laputa-mempalace-bridge` 手工脚本(`bulk_write.py`)
- 修改 `06-history_md` 时,`12-changelog` 不知道 → 数据漂移

**表现**:
- Hermes 通过 `laputa-hermes-memory-provider` plugin 拉 laputa,但**不知道** mempalace 22 tool 暴露了什么
- Mempalace 的 `recall` 返 0 结果(数据空) — 大湿 2026-07-06 实测
- Bridge 写 14 条 → mempalace status 报 "Total drawers: 34" 但预期 42(去重 8 条)

### 瓶颈 2:无编排层

Laputa 当前行为:**API 层直调底层**,没有 step 抽象:

```
HTTP POST /api/sections/{name} → atomic_write_json (laputa.go:388)
                                 → os.WriteFile 覆盖
                                 → 无事务
                                 → 无审计索引
                                 → 无 step 校验
```

**Mempalace 22 tool 同上**:`add_drawer` 调 `palace.AddDrawer` → 写 SQLite → 无 step 抽象。

**后果**:
- "**改 06-history_md 时,12-changelog 不一致就出现**"(大湿原话)
- 改 1 个 section 不联动其他 section
- 没有"先 X 后 Y"或"X 失败回滚"的 step 编排
- Wakeup 4 步契约(system_prompt_block / prefetch / sync_turn / on_session_end)**最像** memU 协议,但下游只有 hermes 一个 client,没人在用 4 步契约做更复杂编排

### 瓶颈不是"代码多",而是"两套事实库 + 没有 step 抽象"

---

## 3. Garden 的最终决定

### 3.1 5 层结构(2026-07-06 最终拍板)

```
┌─────────────────────────────────────────────────────────────────┐
│  Hermes  (上游 client)                                          │
└────────────────────────┬────────────────────────────────────────┘
                         │
┌────────────────────────▼────────────────────────────────────────┐
│  LAYER 1: FACADE  (5 个业务方法,garden 入口)                     │
│    recall_for_intent  /  archive_session  /  propose_milestone  │
│    sync_governance_to_memory  /  sync_memory_to_governance      │
└────────────────────────┬────────────────────────────────────────┘
                         │
┌────────────────────────▼────────────────────────────────────────┐
│  LAYER 2: PIPELINE  (从 memU 抄的 step 编排)                      │
│    ├── WorkflowStep      (requires / produces / capabilities)    │
│    ├── PipelineManager   (同引擎跑多 pipeline)                    │
│    ├── Runner            (串行 step chain)                        │
│    └── Interceptor       (before_step / after_step / on_error)   │
└────────────────────────┬────────────────────────────────────────┘
                         │
┌────────────────────────▼────────────────────────────────────────┐
│  LAYER 3: 治理层  (Database Protocol + LLM Profile routing)       │
│    ├── Database interface (聚合 laputa.SectionStore + mempalace) │
│    ├── 6 repo 协议 (in-memory / sqlite / 未来 redis)             │
│    ├── LLM profile routing (llm_profile / embed_llm_profile)     │
│    └── Composition root (一个 __init__ 装好所有)                 │
└────────────────────────┬────────────────────────────────────────┘
                         │
┌────────────────────────▼────────────────────────────────────────┐
│  LAYER 4: BACKEND  (现成的,直接 import)                           │
│    ├── laputa.SectionStore          (14 section CRUD)            │
│    ├── mempalace.search.Store       (vector search)               │
│    ├── mempalace.hybrid.Store       (BM25 + vector)               │
│    ├── mempalace.Embedder           (ONNX)                        │
│    ├── mempalace.KnowledgeGraph     (KG)                          │
│    └── mempalace.Diary / RoomDetector / ...                       │
└────────────────────────┬────────────────────────────────────────┘
                         │
┌────────────────────────▼────────────────────────────────────────┐
│  LAYER 5: Memory file export  (抄自 memU)                         │
│    INDEX.md / MEMORY.md / memory/ / resource/                     │
│    人类 + agent 都可读                                            │
└─────────────────────────────────────────────────────────────────┘
```

### 3.2 请求流(2026-07-06 最终拍板)

```
hermes 调用
   ↓
facade.recall_for_intent(intent)
   ↓
pipeline.Run("recall_pipeline", ctx)
   ↓
[step1: read identity (laputa.SectionStore)]
   ↓ interceptor 审计
[step2: search (mempalace.search.Store)]
   ↓ interceptor 审计
[step3: filter by capabilities (治理: 哪些 API 启用)]
   ↓
return result
```

### 3.3 5 个业务方法 (facade 入口)

| 方法 | requires | produces | 走 backend |
|---|---|---|---|
| `recall_for_intent(intent, session_id)` | intent, session_id | memory_hits | laputa + mempalace |
| `archive_session(session_log)` | session_log | changelog_entry | laputa 12-changelog |
| `propose_milestone(milestone, source)` | milestone, source | proposal_id | laputa 11-proposal_inbox |
| `sync_governance_to_memory(section_name)` | section_name | drawer | laputa → mempalace |
| `sync_memory_to_governance(drawer_id)` | drawer_id | proposal_id | mempalace → laputa |

### 3.4 Garden 形态的 3 句概括

1. **Garden = facade(5 入口) + pipeline(step 编排) + 治理(Database Protocol + LLM profile routing)**
2. **Backend 复用现成** (laputa.SectionStore + mempalace.{search.Store, Embedder, ...}),不重新造
3. **每个业务方法走 1 个 pipeline, pipeline 串行调 1-N 个 step, step 通过治理 config 决定调哪些 backend**

### 3.5 范围边界(明确做 / 不做)

#### 3.5.1 Garden 范围

| 范畴 | 决策 | 理由 |
|---|---|---|
| **Go 路径独立** | ✅ 只做 Go | Rust 那条线(diva)无清晰思路,Go 先验证 |
| **包装 laputa + mempalace** | ✅ 复用现有 Go 代码 | 已是 library 形态,不改它们的代码 |
| **memU 7 抽象** | ✅ 借鉴, 但不照搬 | memU 是 Python,Go 这边要 idiom 化 |
| **Composition root** | ✅ | "一个 __init__ 装好所有 backend" |
| **5 个 facade 入口** | ✅ | CRUD 4 + governance sync 1 |
| **WorkflowStep 契约** | ✅ | requires / produces / capabilities |
| **PipelineManager 运行时编排** | ✅ | enable_step / disable_step / replace_step |
| **Interceptor 框架** | ✅ | before / after / on_error,审计 + metrics |
| **LLM profile routing** | ⏸️ 第二阶段 | 先不引入,等 step 跑通再加 |
| **Memory file export (INDEX.md 等)** | ⏸️ 第二阶段 | memU 抄,但优先级低 |

#### 3.5.2 不做(明确划线)

| 范畴 | 决策 | 理由 |
|---|---|---|
| 重写 laputa Go 代码 | ❌ | laputa 跑稳了,不破坏 |
| 重写 mempalace 16 个 internal 包 | ❌ | mempalace 跑稳了,只 import 现成 interface |
| 引入 ONNX(用 mempalace 现成的就行) | ❌ | ONNX 已经在 mempalace,garden 调它 |
| Postgres backend | ❌ | 现阶段 file + sqlite 够用 |
| 多 agent 中心 | ❌ | diva 单 agent,garden 也单 agent |
| 多模态(mempalace 的 image/video 工具) | ❌ | diva 是工程 agent,用不上 |
| 取代 LaputaMemoryProvider / HybridMemoryProvider | ⏸️ 第二阶段 | diva 那边的 provider 暂时不动 |
| Garden 起 HTTP server | ⏸️ 第二阶段 | 第一版:CLI subcommand + 5 facade |
| Garden 起 MCP server | ❌ | 跟 mempalace 重复;garden 不是 mempalace 替代品 |

### 3.6 工作量预估

| Phase | 内容 | 工作量 |
|---|---|---|
| Phase 1 | facade 5 方法 + pipeline 骨架 + 1 个 demo pipeline ("recall_for_intent") | 1-2 周 |
| Phase 2 | Interceptor 框架 + PipelineManager 运行时编排 + LLM profile routing | 2 周 |
| Phase 3 | Memory file export + Garden 起 HTTP/MCP server | 2 周 |
| Phase 4 | 集成测试 + 性能基准 + 文档 | 1 周 |
| **总计** | | **6-8 周** |

### 3.7 第一个 demo pipeline 设计 (Phase 1)

`recall_for_intent` 是最常用的 pipeline,第一版做这个:

```yaml
pipeline: recall_for_intent
description: 根据用户意图召回相关记忆
steps:
  - id: read_identity
    role: governance_load
    requires: [session_id]
    produces: [identity]
    capabilities: [laputa]
    handler: laputa.SectionStore.Read(01-identity)
  
  - id: search_palace
    role: semantic_search
    requires: [intent]
    produces: [memory_hits]
    capabilities: [mempalace.search, mempalace.embedder]
    handler: |
      mempalace.search.Store.Search(
        query: embedder.Embed(intent),
        limit: 10,
        filter: {agent: identity.agent_id}
      )
  
  - id: filter
    role: policy_filter
    requires: [identity, memory_hits]
    produces: [filtered_hits]
    capabilities: [governance]
    handler: governance.FilterByPolicy(memory_hits, identity.permissions)
  
  - id: audit
    role: observability
    requires: [session_id, memory_hits]
    produces: []
    capabilities: [audit]
    handler: audit.LogRecall(session_id, memory_hits)
```

**拦截器** (Phase 2 加):
- `before_step`: 校验 step requires 是否被前面 step produces 满足
- `after_step`: metrics 收集(耗时, 错误数)
- `on_error`: 失败重试或降级

---

## 4. Laputa 现状 vs memU 7 抽象

> **来源**: 大湿 2026-07-06 贴给我看的事实表

### 4.1 逐条映射

| memU 抽象 | memU 实现 | Laputa 现状 | 评估 |
|---|---|---|---|
| **WorkflowStep** | `requires / produces` 两套 set, runner 强制校验 | 没有任何 step 抽象, HTTP API 直调底层 | ❌ **完全缺失** |
| **PipelineManager** | 同一套 step 引擎跑 memorize/retrieve/retrieve_workspace/memory_files 4 个 pipeline | 没有 pipeline 概念, 只有 wakeup 4 步契约 | ⚠️ **部分对应** (wakeup 4 步) |
| **Database Protocol** | `Database(Protocol)` + 6 个 repo 协议 | **已有** `SectionStore` interface + `FileStore` 实现 | ✓ **已有, 但太窄** (只 cover 14 section) |
| **MemorizeMixin / RetrieveMixin** | MemoryService 通过 mixin 组合 | 没有 mixin, service.rs 3182 行 monolith | ❌ **缺失** |
| **能力标签 capabilities** | `={"llm","vector","db","io"}` step 显式声明 | 没有能力标签, 都是直接调 | ❌ **缺失** |
| **Profile-based LLM routing** | step config 字段 `llm_profile / embed_llm_profile` | rhythm 子命令单 profile,没有 step 级别路由 | ⚠️ **部分对应** (rhythm) |
| **Composition root 收口** | `MemoryService.__init__` 一个函数装好 | `NewEngine(store SectionStore) *Engine` 单参数,只接受 1 个 backend | ⚠️ **部分对应** |
| **Memory file export** | INDEX.md / MEMORY.md / memory/ / resource/ | 完全没有,只有 JSON section | ❌ **完全缺失** |

### 4.2 结论

| 类别 | 数量 | memU 抽象 |
|---|---|---|
| 完全缺失 | 4 | WorkflowStep / Mixin / capabilities / file export |
| 部分对应 | 3 | PipelineManager / LLM routing / Composition root |
| 已有但需扩展 | 1 | Database Protocol (SectionStore 范围太窄) |

**Garden 实现 = 把 4 个"完全缺失"补上 + 扩展 1 个"已有"** = 5 个新增抽象,落在 Garden Layer 1-3。

---

## 5. 候选方案整理

> **来源**: 2026-07-06 一天 6 轮对话的演变

### 5.1 候选演变时间线

| 时间 | 候选 | 状态 |
|---|---|---|
| 轮 1 (我提) | **B1**: laputa + mempalace 进程共存,加 HTTP↔stdio 桥 | ❌ 否(分体式带来不便) |
| 轮 1 (我提) | **B2**: Go 内置 sqlite + FTS5 + ONNX,单 binary | ❌ 否(Go 不是最终目标) |
| 轮 1 (我提) | **B3**: 重写 SectionStore 合并 14 section + palace graph | ❌ 否(实质不成立) |
| 轮 2 (我提) | **候选 1**: 角色分工(la puta 治理/mempalace 记忆) | ❌ 否(同 B1) |
| 轮 2 (我提) | **候选 2**: laputa 是薄壳(砍 14 section) | ❌ 否(14 section 是核心) |
| 轮 2 (我提) | **候选 3**: laputa + 内置 file/sqlite 后端 (轻量化) | ❌ 否(性能妥协) |
| 轮 3 | 讨论 22 API + 5 个 CRUD | 收窄方向但没定 |
| 轮 3 | 讨论 memtle (Rust 替代 mempalace-go) | 重要洞察,走 rust 路径 |
| 轮 3 | 多 agent 中心 / soul / bridge / partition | ❌ 否(我自己跑偏了) |
| 轮 4 | 11 个候选, codex 评估 9 个 | codex 客观打分 |
| 轮 4 | 我推荐 β (garden 包装 memtle+laputa) | 错 — rust 不是目标 |
| 轮 5 (大湿纠正) | **Go 路径独立, rust 不管** | ✅ 重新定方向 |
| 轮 5 | **Garden = facade**(大湿拍) | ✅ 确认 |
| 轮 5 | **Garden 不只有 facade, 也有 pipeline 总之是治理层** | ✅ 拍板 |
| 轮 5 | **c 选项 (facade 入口 → pipeline 调 → 治理) ** | ✅ 拍板 |
| 轮 6 | memU 7 抽象表 (大湿贴) | ✅ 借鉴清单 |
| **当前** | **Garden = facade + pipeline + 治理层** (3 个并列子系统) | ✅ **最终定** |

### 5.2 Codex 9 候选评估(2026-07-06 派出去)

Codex 实读代码,9 候选打分(从高到低):

| # | 候选 | 分数 | 备注 |
|---|---|---|---|
| α | 扩展 hybrid.rs | 37 | 但 Cargo 依赖方向锁死,治理↔记忆统一架构上做不到 |
| X4 | trait 注入 | 36 | 0 新 dep, 解决"互斥路径合并" |
| β | garden adapter (包装) | 35 | 解两边,新 crate 600-800 行 |
| X6 | garden helper | 35 | 不动 provider 路由,只做工具函数 |
| γ | garden + trait 抽象 | 29 | YAGNI 风险高 |
| C3 | laputa+sqlite(Go 路径) | 23 | out of scope (laputa 内部) |
| X5 | laputa+memtle 直连 | 19 | 违反 LAPUTA.md §5 权威方向 |
| B2 | Go mid-fusion | 9 | "Go 不是目标" 明确否 |
| B3 | Go deep-fusion | 9 | 同 B2 |

**Codex 报告路径**: `C:/Users/Administrator/Desktop/morediva/agent-diva/docs/research/diva-memory-architecture/9-candidate-comparison.md`

### 5.3 关键的认知校正(2026-07-06 我犯的错)

| 我之前说 | 事实 |
|---|---|
| "mempalace 有 22 tool" | 实际 43 tool |
| "laputa 跟 mempalace 是两个独立 Go binary" | ✓ 这点对 |
| "diva 那边的 mentle 是 mempalace 替代" | ✓ 对(都是 mempalace 的 Rust 移植) |
| "garden = 5 个 CRUD facade" | 部分对, 实际 facade + pipeline + 治理层 |
| "garden 是 1-2 周工作量" | 错, 实际 6-8 周(memU 7 抽象要全做) |
| "codex 报告里我推荐 β" | 当时凭印象, 实际打分 α 37 > β 35 |
| "Hermes 不知道 laputa 有 soul.json" | ✓ 对,soul.json 是野生存在的 |
| "SectionStore interface 已现成" | ✓ 对(我之前查到了) |
| "22 tool 包含 5 个 infra + 17 typed wrapper" | 错的,实际 43 tool |

### 5.4 大湿反复纠正我的事(记下来别再犯)

1. "go 和 rust 是两套实现, 彼此隔离" — 我反复把它们混在一起
2. "garden 不只是 facade" — 我反复把它简化成 facade
3. "现在我没想到怎么继续 rust" — 我反复试图把 rust 拉回来
4. "garden 是 facade 也成立" — 上一轮我以为这否了 facade,实际是确认
5. "22 API 不要混,这是 22 个" — 我有时把数字搞错

---

## 6. 待办与不做清单

### 6.1 Phase 1 (1-2 周) - 立即开始

- [ ] **D1.1**: 在 `~/Desktop/projects/laputa/` 下创建 `garden/` Go module
- [ ] **D1.2**: 实现 facade 5 个方法 (空实现, 编译过)
- [ ] **D1.3**: 实现 Pipeline + WorkflowStep + Runner (骨架, 只 1 个 demo pipeline)
- [ ] **D1.4**: 写 `recall_for_intent` demo pipeline (3 步: read identity → search → audit)
- [ ] **D1.5**: 集成测试: facade.recall_for_intent("松本的偏好") → 真正召回 laputa + mempalace
- [ ] **D1.6**: 文档: garden/README.md 写"这是 go-only, 不依赖 rust"

### 6.2 Phase 2 (2 周) - 等 Phase 1 跑通

- [ ] **D2.1**: Interceptor 框架 (before_step / after_step / on_error)
- [ ] **D2.2**: PipelineManager enable/disable/replace_step 运行时编排
- [ ] **D2.3**: LLM profile routing
- [ ] **D2.4**: Database Protocol 完整化 (6 repo 协议: identity / relationships / commitments / preferences / sessions / drawers)

### 6.3 Phase 3 (2 周) - 输出

- [ ] **D3.1**: Memory file export (INDEX.md / MEMORY.md / memory/ / resource/)
- [ ] **D3.2**: Garden HTTP server (供 hermes 调)
- [ ] **D3.3**: 替换 hermes 当前双 plugin 路径 (laputa + mempalace → garden)

### 6.4 Phase 4 (1 周) - 收尾

- [ ] **D4.1**: 性能基准 (recall 1000 次的 P50 / P95)
- [ ] **D4.2**: 集成测试覆盖 5 个 facade 方法
- [ ] **D4.3**: 文档: GARDEN-USER-GUIDE.md

### 6.5 不做 (明确划线)

- ❌ 改 laputa Go 代码(跑稳了)
- ❌ 改 mempalace 16 个 internal 包(跑稳了)
- ❌ Rust 路径(用户没思路, 暂不碰)
- ❌ Postgres backend(file + sqlite 够用)
- ❌ 多 agent 中心(diva 单 agent)
- ❌ 多模态(工程 agent 用不上)
- ❌ Garden 替代 HybridMemoryProvider / LaputaMemoryProvider(Phase 3 才考虑)
- ❌ Garden 替代 mempalace (mempalace 自己跑, garden 包装它)

### 6.6 待定 (用户没拍)

- [ ] Garden 是否起 HTTP server? (Phase 3 决定)
- [ ] Garden 是否起 MCP server? (倾向否)
- [ ] LLM profile routing 是否第一版就要? (倾向第二阶段)
- [ ] Garden 是否也吸收 9-12 tbd section 的 schema? (laputa 14 + mempalace KG 的 混合 schema)

---

## 7. 关键事实索引

### 7.1 文件路径速查

| 项 | 路径 |
|---|---|
| Laputa 项目 | `~/Desktop/projects/laputa/` |
| Laputa.go (核心 3182 行) | `~/Desktop/projects/laputa/laputa.go` |
| Laputa ARCHITECTURE.md | `~/Desktop/projects/laputa/ARCHITECTURE.md` |
| Mempalace 项目 | `~/Desktop/projects/mempalace-go-redis-v2/` |
| Mempalace server 入口 | `~/Desktop/projects/mempalace-go-redis-v2/cmd/server/main.go` (1207 行) |
| Hermes laputa plugin | `C:/Users/Administrator/AppData/Local/hermes/plugins/laputa/__init__.py` |
| Diva workspace | `~/Desktop/morediva/agent-diva/` |
| Codex 9 候选报告 | `~/Desktop/morediva/agent-diva/docs/research/diva-memory-architecture/9-candidate-comparison.md` |
| 架构 HTML (本对话产物) | `~/Desktop/morediva/agent-diva/arch-candidates.html` |
| 之前桌面文档 (参考格式) | `~/Desktop/LAPUTA-plan1-remaining-2026Q3.md` |
| **本文件目标路径** | `~/Desktop/NEW-LAPUTA.md` |

### 7.2 进程与端口

| 进程 | PID | 状态 | 端口 |
|---|---|---|---|
| laputa.exe | 12996 | 跑,uptime 3h+ | 127.0.0.1:7373 |
| mempalace.exe | (无) | 未跑 | (stdio, 无端口) |
| Memurai Redis | 5188 | 跑(系统服务) | 127.0.0.1:6379 |
| Hermes | (live) | 当前 agent | (主进程) |

### 7.3 关键数字

| 数字 | 含义 |
|---|---|
| **14** | laputa section 总数 |
| **5** | 5 stable + agent_self sections |
| **1** | 1 stable + user_only sections (commitment) |
| **3** | 3 stable + report_system sections (daily/weekly/monthly) |
| **5** | 5 tbd sections (journal_reflective / proposal_inbox / changelog / report_indexes / aaak_summaries) |
| **43** | mempalace tool 总数(我之前说 22 是错的) |
| **3182** | laputa.go 行数 |
| **1207** | mempalace cmd/server/main.go 行数 |
| **5** | mempalace 主要 struct: search.Searcher / hybrid.Searcher / kg.KnowledgeGraph / diary.Diary / embedder.Encoder |
| **4** | mempalace 现成 interface: Store / Embedder / LlamaClient / RoomDetector |
| **6** | laputa 14 section 中已激活的 stable section 数 |
| **2** | laputa 已激活但 report path 未运行的 section (daily/weekly) |

### 7.4 Go 这边的 interface 现状(都是现成)

```go
// laputa 已有
type SectionStore interface {
    Read(ctx, section SectionName) (map[string]any, error)
    Write(ctx, section SectionName, data map[string]any) error
    Patch(ctx, section SectionName, path string, value any) error
    Delete(ctx, section SectionName, path string) error
    List(ctx) ([]SectionName, error)
    Exists(ctx, section SectionName) (bool, error)
}

// mempalace 已有
type Store interface {  // search.Store
    Search(query []float32, limit int, filter map[string]any) ([]govector.SearchResult, error)
    Add(id string, vector []float32, payload map[string]any) error
    AddBatch(points []govector.Point) error
    Delete(id string) error
    ListAll(limit int) ([]govector.SearchResult, error)
    Close() error
}

type Embedder interface {
    CreateEmbedding(ctx, text) ([]float32, error)
    CreateEmbeddings(ctx, texts) ([][]float32, error)
}
```

---

## 8. 时间线

### 2026-07-06 关键事件

| 时间 | 事件 |
|---|---|
| ~08:00 | Hermes 调 laputa 跑稳 (PID 12996) |
| 上午 | 大湿 11 轮对话讨论 11 个候选方案 |
| 中午 | 大湿 8 轮讨论 memtle / diva 集成 |
| 14:00 | codex 9 候选评估报告落地 |
| 14:30 | 大湿贴 memU 7 抽象 + Laputa 7 缺陷表 |
| 15:00 | 大湿拍板: Garden = facade + pipeline + 治理层 |
| 15:10 | 大湿拍板: c 选项(facade 入口 → pipeline 调 → 治理) |
| 15:20 | 大湿拍板: 桌面文档 `NEW-LAPUTA.md` |
| 15:30 | **当前** - 整理文档中 |

### 后续时间表(预估)

| 周 | 计划 |
|---|---|
| 2026-07-13 ~ 2026-07-19 | Phase 1: garden/ module 骨架 + 1 个 demo pipeline |
| 2026-07-20 ~ 2026-07-31 | Phase 2: Interceptor + PipelineManager + LLM profile |
| 2026-08-01 ~ 2026-08-14 | Phase 3: file export + HTTP server + 替换 hermes plugin |
| 2026-08-15 ~ 2026-08-21 | Phase 4: benchmark + 集成测试 + 文档 |

---

## 附录 A: 关键命令速查

### 验证 laputa 活着

```bash
curl -s -m 3 http://127.0.0.1:7373/healthz
# 或
curl -s -m 3 http://127.0.0.1:7373/api/sections/01-identity | python -m json.tool | head -20
```

### 验证 mempalace 可启动

```bash
cd ~/Desktop/projects/mempalace-go-redis-v2
# 单次启动测试(ONNX 加载 ~30s)
(sleep 2 && echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"t","version":"1"}}}' && sleep 1 && echo '{"jsonrpc":"2.0","method":"notifications/initialized","params":{}}' && sleep 1 && echo '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"mempalace_status","arguments":{}}}') | timeout 10 ./mempalace.exe server
```

### 验证 garden(等 Phase 1 实现后)

```bash
cd ~/Desktop/projects/laputa/garden
go test ./...
./garden recall --intent "松本的偏好"
```

---

## 附录 B: 术语表

| 术语 | 含义 |
|---|---|
| **Laputa** | Go 实现的 governance 引擎,14 section 治理,跑 :7373 |
| **Mempalace** | Go 实现的 memory palace,43 MCP tool,走 stdio |
| **Garden** | 新建 Go 项目,façade + pipeline + 治理层,包装 Laputa + Mempalace |
| **MemU** | Python 21,260 行的 memory 系统,作为 Garden 抽象借鉴来源(不是直接用户) |
| **Memtle** | Rust 0.1.2 crate, Mempalace 的 Rust 移植(diva 用,Go 不用) |
| **Diva** | Rust 实现的 agent 系统(16 crate),单 agent,使用 memtle + agent-diva-laputa |
| **Hermes** | Python TUI agent, 当前 laputa plugin 调 :7373 |
| **Section** | Laputa 的 14 个 JSON 治理单元(01-identity 到 14-aaak-summaries) |
| **Step** | Garden pipeline 的单个原子操作,有 requires/produces/capabilities 契约 |
| **Pipeline** | 串行执行的 step 集合,调 1 个或多个 backend |
| **Backend** | 现成的 Laputa SectionStore + Mempalace 22 tool 集合 |
| **Composition root** | Garden __init__ 一个函数装好所有 backend,没有"先启 A 再启 B" |
| **写权 (WriteAuth)** | Laputa section 的写权限: agent_self / user_only / report_system / tbd |

---

## 附录 C: 我之前对话里反复犯的错(留给以后的松本)

1. ❌ "mempalace 有 22 tool" — 实际 43
2. ❌ "garden 是 facade" — 实际 facade + pipeline + 治理
3. ❌ "Go + Rust 是一套" — 实际两套隔离
4. ❌ "1-2 周工作量" — 实际 6-8 周
5. ❌ "codex 9 候选里我推荐 β" — 实际 α 37 分最高(但不解两边)
6. ❌ "soul.json 是治理层 schema 残留" — 实际野生存在,未编入 14 section
7. ❌ "Hermes 不知道 mempalace" — 实际 Hermes 调 laputa :7373,没 mempalace 路径

---

**文档完成时间**: 2026-07-06 15:30
**文件**: `C:\Users\Administrator\Desktop\NEW-LAPUTA.md`
**下一步**: 等大湿 review,然后开 Phase 1
