# Story 3.2: 语义检索（RAG）

**Story ID:** 3.2  
**Story Key:** 3-2-semantic-search  
**Status:** review  
**Created:** 2026-04-14  
**Project:** 天空之城 (Laputa)

---

## 用户故事

As a **AI agent**,  
I want **发起深度语义检索获取相关记忆**,  
So that **我可以补充当前对话的历史上下文**。

---

## 验收标准

- **Given** usearch 向量索引已建立
- **When** agent 调用 `search.semantic(query, top_k)`
- **Then** 返回语义最相关的记忆列表
- **And** 每条结果包含相似度分数
- **And** RAG 能力启用（D-004 决策）

扩展约束：
- 语义检索必须基于 usearch HNSW 索引，不能绕向量检索
- 相似度分数必须反映 cosine similarity，范围 [-1, 1] 或 [0, 1]
- 返回结构应为 `Vec<MemoryRecord>` 或等价类型，便于上层消费
- 默认不返回 `discard_candidate = true` 的记录
- 结果应加热度二次排序因子（可选参数，默认不启用）

---

## Epic 上下文

### Epic 3 目标

Epic 3 是产品 MVP 的核心验证点（P1），负责：
- `FR-6` 唤醒包生成
- `FR-7` 时间流召回
- `FR-8` 深度检索

本 Story 是 Epic 3 的语义检索层，与 Story 3.1（时间流检索）共同构成检索基础设施，最终由 Story 3.4（混合检索）融合。

---

## 现有代码与差距

### 已有能力

1. `Laputa/src/searcher/mod.rs`
   - 已有 `Searcher` 结构体
   - 已有 `search()` / `search_memories()` 方法
   - 已有 `format_search_results()` 格式化输出
   - 已有 `wake_up()` 唤醒接口

2. `Laputa/src/vector_storage.rs`
   - 已有 `VectorStorage` 结构体
   - 已有 fastembed + usearch + SQLite 集成
   - 已有 `get_memories(...)` 查询接口
   - 已有 `VECTOR_DIMS = 384`，HNSW 配置

3. `Laputa/src/storage/memory.rs`
   - 已有 `LaputaMemoryRecord` 结构
   - 已有 `heat_i32`、`last_accessed`、`access_count` 字段

### 当前缺口

- `Searcher::search()` 当前偏向通用搜索，缺少明确的语义检索入口 `search.semantic(query, top_k)`
- 缺少返回结果携带相似度分数的能力
- 缺少默认排除 `discard_candidate` 的过滤逻辑
- 缺少热度二次排序因子（`sort_by_heat` 选项）

---

## 架构约束

### 1. 不重写检索栈

架构已明确：
- `src/searcher/` 是检索层
- `src/vector_storage.rs` 是 SQLite/usearch 数据访问层
- 语义检索增强应在这两层扩展

禁止：
- 新建独立语义检索存储
- 在 `mcp_server` 中直接拼接 SQL 绕过 `searcher`
- 把语义检索硬塞进 `Diary` 或其他模块

### 2. 向量索引是硬依赖

Story 3.2 的核心是 usearch HNSW 索引：
- 向量维度固定 `384`（fastembed AllMiniLML6V2）
- 距离度量 `Cosine similarity`
- 不支持离线环境下的语义检索（需预先建立索引）

### 3. 返回结构必须携带相似度分数

验收标准要求"每条结果包含相似度分数"。因此返回结构应扩展为：
```rust
pub struct SearchResult {
    pub record: MemoryRecord,
    pub similarity: f32,  // cosine similarity
}
```

或返回 `Vec<(MemoryRecord, f32)>`。

### 4. 默认排除 discard_candidate

延续 Story 2.2 规则：
- `discard_candidate = true` 默认不进入语义检索
- SQL 层过滤 `discard_candidate = 0`
- 可选参数 `include_discarded = true` 时才返回

### 5. 热度二次排序（可选）

架构文档 D-004 要求 RAG 能力启用。建议：
- 默认按相似度排序
- 可选参数 `sort_by_heat = true` 时，按 `heat_i32 DESC` 二次排序

---

## 推荐实现方案

### Searcher 层 API

建议新增：
```rust
pub async fn semantic_search(
    &self,
    query: &str,
    top_k: usize,
    options: SemanticSearchOptions,
) -> Result<Vec<SearchResult>>
```

其中：
```rust
pub struct SemanticSearchOptions {
    pub wing: Option<String>,
    pub room: Option<String>,
    pub include_discarded: bool,
    pub sort_by_heat: bool,
}
```

### VectorStorage 层 API

建议新增或扩展：
```rust
pub fn semantic_search(
    &self,
    query_vector: &[f32],
    top_k: usize,
    filter: Option<&SearchFilter>,
) -> Result<Vec<(MemoryRecord, f32)>>
```

### SearchResult 结构

建议定义：
```rust
pub struct SearchResult {
    pub record: MemoryRecord,
    pub similarity: f32,
    pub rank: usize,
}
```

---

## 文件建议

优先考虑以下改动点：

- `Laputa/src/searcher/mod.rs`
  - 新增 `semantic_search()` 入口
  - 新增 `SearchResult` 结构
  - 新增 `SemanticSearchOptions` 参数结构
- `Laputa/src/searcher/result.rs`
  - SearchResult 结构定义（可选独立文件）
- `Laputa/src/vector_storage.rs`
  - 扩展或新增语义检索查询方法
  - 支持相似度分数返回
- `Laputa/src/mcp_server/mod.rs`
  - MCP Tool `laputa_semantic_search` 暴露（可选）
- `Laputa/tests/test_semantic_search.rs`
  - 单元测试与集成测试

---

## 实现细节要求

### 1. 向量嵌入生成

- 使用已有 `fastembed::TextEmbedding`
- 模型固定 `AllMiniLML6V2`，384 维
- 输入文本长度建议 clamp（超过 512 tokens 截断）

### 2. usearch 索引查询

- 使用 `index.search(query_vector, top_k)` 获取候选
- 返回 `(label, distance)` 列表
- 将 label 映射到 SQLite `memories` 表记录

### 3. 相似度分数转换

usearch cosine distance 转换：
- `distance = 0` → similarity = 1（完全相似）
- `distance = 1` → similarity = 0（无相似）
- `distance = 2` → similarity = -1（相反）

转换公式：`similarity = 1 - distance`

### 4. discard 过滤

SQL 层过滤：
```sql
WHERE is_archive_candidate = 0 AND discard_candidate = 0
```

或在 VectorStorage 层后处理过滤。

### 5. 热度二次排序

若 `sort_by_heat = true`：
- 先获取相似度 Top-K 候选
- 再按 `heat_i32 DESC` 重排序
- 最终裁剪到 `top_k`

---

## 测试要求

至少补齐以下测试：

1. 语义检索返回结果测试
   - 输入 query 能返回相关记忆
   - 结果包含相似度分数

2. Top-K 裁剪测试
   - 传入 top_k=5 返回 5 条
   - top_k 大于实际记录数时不报错

3. discard 过滤测试
   - `discard_candidate = true` 默认不返回
   - `include_discarded = true` 时返回

4. 热度二次排序测试
   - `sort_by_heat = true` 时结果按热度排序

5. 空库/冷启动测试
   - 无索引数据时返回空结果而非报错

6. 长文本截断测试
   - 超长 query 不导致嵌入失败

推荐测试文件：
- `Laputa/tests/test_semantic_search.rs`

---

## 禁止事项

- 不要绕过 usearch 索引直接用 SQLite LIKE 查询
- 不要在语义检索中引入新的向量维度或模型
- 不要忽略 `discard_candidate` 过滤
- 不要在检索路径实时计算热度衰减
- 不要返回无相似度分数的结果

---

## 实施任务

- [x] 在 `Searcher` 新增 `semantic_search(query, top_k, options)` 入口
- [x] 定义 `SearchResult` 结构，携带 `record` + `similarity` + `rank`
- [x] 定义 `SemanticSearchOptions` 参数结构
- [x] 在 `VectorStorage` 扩展语义检索查询方法，返回 `(MemoryRecord, f32)`
- [x] 实现 usearch 查询 + SQLite 映射 + 相似度分数转换
- [x] 默认排除 `discard_candidate` 和 `is_archive_candidate`
- [x] 可选热度二次排序 `sort_by_heat`
- [x] 补齐语义检索、Top-K、过滤、排序、空库测试
- [x] 可选 MCP Tool `laputa_semantic_search` 暴露

---

## 完成定义

- [x] 能按 query 返回语义相关记忆
- [x] 每条结果包含相似度分数
- [x] 默认返回 top_k 条（默认 10）
- [x] 默认不返回 discard_candidate
- [x] 不破坏现有 `searcher` 时间检索能力
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
- `D:\VIVYCORE\newmemory\_bmad-output\implementation-artifacts\3-1-timeline-recall.md`
- `D:\VIVYCORE\newmemory\_bmad-output\implementation-artifacts\3-4-hybrid-search.md`

---

## Dev Agent Record

### Context Notes

- Story 3.2 是 Epic 3 语义检索层，依赖 usearch HNSW 索引
- 当前 Searcher 已有通用 search 入口，但缺少明确的语义检索 API
- VectorStorage 已集成 fastembed + usearch，可直接复用
- 需要返回结构携带相似度分数
- 需要默认排除 discard_candidate

### Completion Note

Implemented a typed semantic-search path with Searcher::semantic_search(...), SemanticSearchOptions, and ranked SearchResult output.
Extended VectorStorage with filtered usearch retrieval that maps SQLite rows back to cosine similarity scores, excludes discarded and archive candidates by default, and optionally reranks top-k candidates by heat.
Added MCP exposure via laputa_semantic_search and kept the existing formatted search path aligned with the new semantic-search results.
Added regression coverage for top-k truncation, similarity payloads, discard/archive filtering, optional heat reranking, empty-index behavior, JSON formatting, and graceful fallback when storage or embeddings are unavailable.

### File List

- Laputa/src/searcher/mod.rs
- Laputa/src/vector_storage.rs
- Laputa/src/mcp_server/mod.rs
- Laputa/tests/test_semantic_search.rs
- _bmad-output/implementation-artifacts/sprint-status.yaml

### Change Log

- 2026-04-15: Implemented Story 3.2 semantic search, added MCP exposure, added regression tests, and moved the story to review.

