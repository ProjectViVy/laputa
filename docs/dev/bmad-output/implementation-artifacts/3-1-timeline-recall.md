# Story 3.1: 时间流检索

**Story ID:** 3.1  
**Story Key:** 3-1-timeline-recall  
**Status:** review  
**Created:** 2026-04-14  
**Project:** 天空之城 (Laputa)

---

## 用户故事

As a **用户**,  
I want **按时间窗口或周期层级检索记忆**,  
So that **我可以回顾特定阶段的经历，并优先看到更有价值的记忆**。

---

## 验收标准

- **Given** 记忆库中已有 L0-L3 数据
- **When** 用户调用 `recall.by_time_range(start, end)`
- **Then** 返回时间范围内的记忆列表
- **And** 结果按热度排序，热度高者优先
- **And** 结果数量限制在合理范围内，默认 `100`

扩展约束：

- 默认实现必须支持精确时间窗口 `start/end`
- 第一版不要求实现周/月/季/年聚合摘要，但接口设计不能阻断后续周期层级扩展
- timeline recall 是“时间检索”，不是语义检索，不应依赖 embedding 相似度排序
- 默认不返回 `discard_candidate = true` 的记录

---

## Epic 上下文

### Epic 3 目标

Epic 3 是产品 MVP 的核心验证点，负责：

- `FR-6` 唤醒包生成
- `FR-7` 时间流召回
- `FR-8` 深度检索

本 Story 是 Epic 3 的第一步，为后续两个能力铺路：

- `3.2 semantic-search` 负责“按问题检索”
- `3.3 wakepack-generate` 负责“对话启动注入”
- `3.4 hybrid-search` 负责“时间流 + 语义 + 热度”融合排序

如果 `3.1` 只做成一个简单 SQL 查询，后续 `3.4` 会被迫重写检索基础设施，因此本 Story 必须把“时间过滤 + 热度排序 + 可扩展接口”一起打稳。

---

## 现有代码与差距

### 已有能力

1. `Laputa/src/searcher/mod.rs`
   - 已有 `Searcher`
   - 已有 `search(...)` / `search_memories(...)`
   - 当前更偏语义检索包装层

2. `Laputa/src/vector_storage.rs`
   - 已有 `get_memories(...)`
   - 已有 `get_memory_by_id(...)`
   - 已有 SQLite `memories` 表访问
   - 已有 `heat_i32` 字段和行映射

3. `Laputa/src/storage/memory.rs`
   - 已有 `LaputaMemoryRecord`
   - 已有 `heat_i32`、`last_accessed`、`access_count`
   - 已有 WAL / FK / schema 初始化

4. `Laputa/src/storage/mod.rs`
   - 已有 `MemoryStack::recall(...)`
   - 当前只是 `Layer2.retrieve(...)` 的包装
   - 尚未支持明确的时间范围过滤对象

### 当前缺口

当前 `VectorStorage::get_memories(...)` 仍是：

- 只支持 `wing/room`
- SQL 默认 `ORDER BY valid_from DESC`
- 没有 `start/end` 过滤
- 没有“热度优先，时间范围过滤”的查询路径

这和 Story 3.1 的要求有两处核心不一致：

1. 缺少时间窗口过滤
2. 排序维度不对，需求要求先看 `heat_i32`，不是仅看最新时间

---

## 架构约束

### 1. 不重写检索栈

架构已经明确：

- `src/searcher/` 是检索层
- `src/vector_storage.rs` 是 SQLite/usearch 数据访问层
- 检索相关增强应在这两层扩展，而不是另起一个 `timeline_db` 或单独服务

禁止：

- 新建独立时间检索存储
- 在 `mcp_server` 中直接拼接 SQL 绕过 `searcher`
- 把时间检索硬塞进 `Diary`

### 2. 热度排序是硬要求

Epic 3 的 AC 明确要求“按热度排序（高热度优先）”。因此本 Story 的 timeline recall 查询必须至少支持：

```sql
ORDER BY heat_i32 DESC, valid_from DESC
```

推荐次排序：

- 第一关键字：`heat_i32 DESC`
- 第二关键字：`valid_from DESC`

这样可以保证：

- 先看重要记忆
- 同热度下再看较新的记录

### 3. 接口要为 3.4 留扩展点

本 Story 不是 Hybrid Search，但接口要能被 `3.4` 复用。建议引入查询对象，而不是把参数散落在多个函数签名里。

推荐：

```rust
pub struct RecallQuery {
    pub start: i64,
    pub end: i64,
    pub wing: Option<String>,
    pub room: Option<String>,
    pub limit: usize,
    pub include_discarded: bool,
}
```

如果后续要扩展：

- 周期层级
- 热度阈值
- 是否混入语义结果

可以继续在 `RecallQuery` 上增字段，而不是反复改公共 API。

### 4. 默认排除 discard_candidate

Story 2.2 已定义：

- `discard_candidate` 默认不进入普通 recall/search

因此 Story 3.1 必须延续这一规则：

- 默认 `include_discarded = false`
- SQL 层过滤 `discard_candidate = 0`

不要把过滤放到返回后再做内存裁剪。

### 5. 不引入 HeatService 依赖

本 Story 只消费已存在的 `heat_i32` 字段，不负责：

- 计算热度
- 衰减热度
- 状态机切换

也就是说：

- 可以排序 `heat_i32`
- 不要在 recall 里补算热度

---

## 推荐实现方案

### 查询对象

建议新增：

- `Laputa/src/searcher/recall.rs`

内容建议：

```rust
pub struct RecallQuery {
    pub start: i64,
    pub end: i64,
    pub wing: Option<String>,
    pub room: Option<String>,
    pub limit: usize,
    pub include_discarded: bool,
}
```

以及：

```rust
impl RecallQuery {
    pub fn by_time_range(start: i64, end: i64) -> Self { ... }
}
```

### Searcher 层 API

建议在 `Laputa/src/searcher/mod.rs` 或 `recall.rs` 中补齐：

```rust
pub async fn recall_by_time_range(
    &self,
    query: RecallQuery,
) -> Result<Vec<MemoryRecord>>
```

或：

```rust
pub async fn recall(&self, query: RecallQuery) -> Result<Vec<MemoryRecord>>
```

这比直接返回格式化字符串更适合：

- CLI
- MCP
- 后续 wakepack / hybrid-search

格式化输出可以作为上层包装，不应成为底层唯一返回类型。

### VectorStorage 层 API

建议新增：

```rust
pub fn recall_by_time_range(
    &self,
    query: &RecallQuery,
) -> Result<Vec<MemoryRecord>>
```

SQL 应支持：

- `valid_from >= start`
- `valid_from <= end`
- 可选 `wing/room`
- 默认 `discard_candidate = 0`
- `ORDER BY heat_i32 DESC, valid_from DESC`
- `LIMIT ?`

### MemoryStack 集成

当前 `MemoryStack::recall(...)` 仍是：

- `wing`
- `room`
- `n_results`

建议本 Story 做最小重构：

- 保留旧接口，避免破坏现有调用
- 额外新增 timeline recall 入口

例如：

```rust
pub async fn recall_by_time_range(
    &self,
    query: RecallQuery,
) -> String
```

这样可以先满足 Story 3.1，同时不阻断后续 `MemoryOperation trait` 统一化。

---

## 文件建议

优先考虑以下改动点：

- `Laputa/src/searcher/mod.rs`
- `Laputa/src/searcher/recall.rs` 或 `Laputa/src/searcher/query.rs`
- `Laputa/src/vector_storage.rs`
- `Laputa/src/storage/mod.rs`
- `Laputa/src/mcp_server/mod.rs` 仅在需要暴露新工具时做薄包装
- `Laputa/src/cli/` 仅在需要补 CLI 子命令时做薄包装

本 Story 不建议碰：

- `wakeup/`
- `heat/`
- `archiver/`

---

## 实现细节要求

### 1. 时间字段选型

当前数据模型使用：

- `valid_from: i64`
- `valid_to: Option<i64>`

因此 Story 3.1 直接沿用 Unix timestamp，不要在持久化层切换成 RFC3339 字符串。

### 2. 时间窗语义

推荐采用闭区间：

- `valid_from >= start`
- `valid_from <= end`

如果 `start > end`：

- 返回 `LaputaError::ValidationError`

### 3. limit 语义

验收标准要求默认 100 条，因此：

- `limit` 缺省值为 `100`
- 同时允许调用方显式传更小值
- 为防止滥用，建议上限 clamp 到 `1000`

### 4. 结果排序

排序固定为：

1. `heat_i32 DESC`
2. `valid_from DESC`

不要在 Story 3.1 引入多种排序模式；如果要留扩展点，可在 `RecallQuery` 中先预留字段但不启用。

### 5. 返回结构

底层返回 `Vec<MemoryRecord>` 或 `Vec<LaputaMemoryRecord>` 即可。  
上层格式化输出时，建议展示：

- `id`
- `valid_from`
- `heat_i32`
- `wing`
- `room`
- 文本摘要

---

## 测试要求

至少补齐以下测试：

1. 时间窗过滤测试
   - 只返回 `start/end` 之间的数据
   - 边界值包含在结果内

2. 热度排序测试
   - 同一时间窗内，热度更高的记录排前
   - 同热度时较新的 `valid_from` 排前

3. discard 过滤测试
   - `discard_candidate = true` 默认不返回
   - `include_discarded = true` 时才返回

4. limit 测试
   - 默认 `100`
   - 显式传入更小 limit 时结果被正确裁剪

5. 参数校验测试
   - `start > end` 返回错误

6. Searcher 集成测试
   - `Searcher` 能调用到底层时间召回接口

7. MemoryStack 或上层封装测试
   - timeline recall 的展示层输出不为空

推荐测试文件：

- `Laputa/tests/test_timeline_recall.rs`

必要时补少量单元测试到：

- `Laputa/src/searcher/mod.rs`
- `Laputa/src/vector_storage.rs`

继续沿用：

- `tempfile`
- `serial_test`

---

## 禁止事项

- 不要把时间检索做成语义检索的别名
- 不要按 `valid_from DESC` 直接交差，需求要求热度优先
- 不要忽略 `discard_candidate`
- 不要在 recall 路径实时计算热度
- 不要为这个 Story 提前实现 WakePack
- 不要把 3.1 和 3.4 一起做成混合排序

---

## 实施任务

- [x] 为 timeline recall 引入统一查询对象 `RecallQuery`
- [x] 在 `vector_storage.rs` 增加按时间范围召回的查询方法
- [x] SQL 层实现 `start/end` 过滤、默认排除 `discard_candidate`
- [x] 按 `heat_i32 DESC, valid_from DESC` 排序
- [x] 默认 limit 设为 `100`，并做参数校验
- [x] 在 `searcher` 层暴露 `recall.by_time_range(...)` 对应接口
- [x] 视需要在 `MemoryStack`、CLI 或 MCP 做薄包装接入
- [x] 补齐时间窗、排序、过滤、limit、参数错误测试

---

## 完成定义

- [x] 能按 `start/end` 返回时间窗内记忆
- [x] 结果按热度优先排序
- [x] 默认最大返回 `100`
- [x] 默认不返回 `discard_candidate`
- [x] 不破坏现有 `searcher` 语义检索能力
- [x] `cargo test` 通过
- [x] `cargo clippy --all-features --tests -- -D warnings` 通过

---

## 参考资料

- `D:\VIVYCORE\newmemory\_bmad-output\planning-artifacts\epics.md`
- `D:\VIVYCORE\newmemory\_bmad-output\planning-artifacts\architecture.md`
- `D:\VIVYCORE\newmemory\_bmad-output\planning-artifacts\prd.md`
- `D:\VIVYCORE\newmemory\Laputa\AGENTS.md`
- `D:\VIVYCORE\newmemory\Laputa\DECISIONS.md`
- `D:\VIVYCORE\newmemory\Laputa\src\searcher\mod.rs`
- `D:\VIVYCORE\newmemory\Laputa\src\vector_storage.rs`
- `D:\VIVYCORE\newmemory\Laputa\src\storage\mod.rs`
- `D:\VIVYCORE\newmemory\_bmad-output\implementation-artifacts\2-2-memory-filter-merge.md`

---

## Dev Agent Record

### Context Notes

- 当前代码已经有语义检索和基础 recall 展示层，但缺少真正的“时间窗召回”查询对象
- 当前 `get_memories(...)` 仍按 `valid_from DESC`，与 Story 3.1 的热度优先要求不符
- 本 Story 的正确落点是补齐 timeline recall 基础设施，而不是提前实现 semantic-search / wakepack / hybrid-search
- 当前工作区不是 git 仓库根目录，未获取可用 git 提交历史

### Completion Note

Implemented timeline recall as a dedicated retrieval path with a reusable `RecallQuery` object.
Added `VectorStorage::recall_by_time_range(...)` with start/end filtering, discard filtering by default, heat-first ordering, and limit validation/clamping.
Exposed the recall path through `Searcher::recall_by_time_range(...)` and `MemoryStack::recall_by_time_range(...)` without changing existing semantic search behavior.
Added targeted regression coverage for query defaults, time-window filtering, ordering, discard inclusion, limit handling, validation errors, Searcher integration, and formatted MemoryStack output.

### File List

- Laputa/src/searcher/recall.rs
- Laputa/src/searcher/mod.rs
- Laputa/src/vector_storage.rs
- Laputa/src/storage/mod.rs
- Laputa/tests/test_timeline_recall.rs
- _bmad-output/implementation-artifacts/sprint-status.yaml

### Change Log

- 2026-04-15: Implemented Story 3.1 timeline recall, added regression tests, and moved story to review.
