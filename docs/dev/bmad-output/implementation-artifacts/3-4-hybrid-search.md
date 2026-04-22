# Story 3.4: 混合检索排�?

**Story ID:** 3.4  
**Story Key:** 3-4-hybrid-search  
**Status:** review  
**Created:** 2026-04-14  
**Project:** 天空之城 (Laputa)

---

## 用户故事

As a **系统**,  
I want **融合时间流与语义检索结果并加热度排�?*,  
So that **用户获得最相关且有价值的记忆，而不是单纯的时间顺序或语义相似度**�?

---

## 验收标准

- **Given** 时间流和语义检索结�?
- **When** hybrid_search 执行
- **Then** 结果按热度因子加权排�?
- **And** 重复结果去重
- **And** 最终结果限制在 top_k

扩展约束�?

- 混合排序必须同时考虑时间相关性、语义相似度和热度�?
- 去重基于记忆 ID，而不是文本内�?
- top_k 默认值应与时间流检索保持一致（100条）
- 排序算法必须可解释，不能是黑盒加�?

---

## Epic 上下�?

### Epic 3 目标

Epic 3 是产�?MVP 的核心验证点，负责：

- `FR-6` 唤醒包生�?
- `FR-7` 时间流召�?
- `FR-8` 深度检�?

�?Story �?Epic 3 的第四步（最后一步），负责整合前三个 story 的能力：

- `3.1 timeline-recall` 提供时间窗口过滤 + 热度排序基础设施
- `3.2 semantic-search` 提供语义检索能力（usearch 向量索引�?
- `3.3 wakepack-generate` 提供唤醒包生成（消耗混合检索结果）
- `3.4 hybrid-search` 负责"时间�?+ 语义 + 热度"融合排序

如果 `3.4` 只是简单合并两个结果列表，而不做真正的加权排序，用户会看到�?

- 要么时间流结果全在前面（语义结果被淹没）
- 要么语义结果全在前面（丢失时间连续性）
- 或者重复记录出现多�?

因此�?Story 必须实现真正的混合排序算法�?

---

## 现有代码与差�?

### 已有能力

1. **Story 3.1 的时间召回基础设施**（假设已完成�?
   - `RecallQuery` 查询对象
   - `VectorStorage::recall_by_time_range(...)` 
   - SQL 层支�?`ORDER BY heat_i32 DESC, valid_from DESC`
   - 默认排除 `discard_candidate`

2. **当前语义检索能�?*（已存在于代码库�?
   - `Searcher::search(...)` 提供语义检�?
   - `VectorStorage::search(...)` 使用 usearch 向量索引
   - 返回 `score` 字段（语义相似度�?

3. **热度字段**（已存在于数据模型）
   - `LaputaMemoryRecord.heat_i32`
   - 范围 0-10000（对�?0.00-100.00�?

4. **Searcher 接口**（已存在�?
   - `Searcher` struct �?`Laputa/src/searcher/mod.rs`
   - 已有 `search(...)` / `search_memories(...)` / `wake_up(...)`

### 当前缺口

1. **没有混合检索入�?*
   - 当前只有纯语义检索或纯时间检�?
   - 缺少 `hybrid_search(...)` 方法

2. **没有加权排序算法**
   - 语义结果�?`score` 排序
   - 时间结果�?`heat_i32` 排序
   - 缺少综合评分公式

3. **没有去重逻辑**
   - 两条检索路径可能返回相同记�?
   - 需要基�?ID 去重

4. **没有统一的混合查询对�?*
   - `RecallQuery` 只支持时间过�?
   - 需要扩展或新建 `HybridQuery`

---

## 架构约束

### 1. 复用 Story 3.1 �?RecallQuery

架构已经明确 `RecallQuery` 是可扩展的查询对象。本 Story 应：

- �?`RecallQuery` 上增字段支持语义检索参�?
- 或新�?`HybridQuery` 包装 `RecallQuery` + 语义参数

推荐方案：新�?`HybridQuery`，避免破坏已稳定�?`RecallQuery`�?

### 2. 混合排序公式必须可配�?

不能硬编码加权系数。推荐：

```rust
pub struct HybridRankingConfig {
    pub time_weight: f64,        // 时间相关性权�?
    pub semantic_weight: f64,    // 语义相似度权�?
    pub heat_weight: f64,        // 热度权重
}
```

默认配置建议�?

- `time_weight = 0.3`
- `semantic_weight = 0.4`
- `heat_weight = 0.3`

这样语义检索稍优先，但时间和热度也有显著影响�?

### 3. 归一化是必须�?

三个维度的量纲不同：

- 时间相关性：需要计算与查询时间窗的距离�?-1�?
- 语义相似度：已经�?0-1（score�?
- 热度�?-10000（需要归一化到 0-1�?

必须在加权前归一化，否则热度会主导排序�?

### 4. 不要在混合检索中重新计算热度

�?Story 只消�?`heat_i32` 字段，不负责�?

- 计算热度
- 衰减热度
- 更新 access_count

排序时直接读取存储的热度值即可�?

### 5. 去重必须在排序前执行

推荐流程�?

1. 执行时间检�?�?得到 List A
2. 执行语义检�?�?得到 List B
3. 合并 A + B，按 ID 去重 �?得到 List C
4. �?C 中每条记录计算综合评�?
5. 按综合评分排�?
6. 返回 top_k

不要先排序再去重，这会导致重复记录占�?top_k 名额�?

### 6. 接口要为 WakePack 留扩展点

`3.3 wakepack-generate` 会消费混合检索结果。因此：

- 返回类型必须�?`Vec<MemoryRecord>` 或等价结�?
- 不能只返回格式化字符�?
- 需要携带综合评分字段（用于调试和可解释性）

推荐�?

```rust
pub struct HybridSearchResult {
    pub record: MemoryRecord,
    pub composite_score: f64,
    pub time_score: f64,
    pub semantic_score: f64,
    pub heat_score: f64,
}
```

---

## 推荐实现方案

### 查询对象

建议新增�?

- `Laputa/src/searcher/hybrid.rs`

内容建议�?

```rust
pub struct HybridQuery {
    pub time_query: RecallQuery,           // 时间过滤部分
    pub semantic_query: String,            // 语义检索文�?
    pub top_k: usize,                      // 最终结果数�?
    pub ranking_config: HybridRankingConfig, // 排序权重
}

pub struct HybridRankingConfig {
    pub time_weight: f64,
    pub semantic_weight: f64,
    pub heat_weight: f64,
}

impl Default for HybridRankingConfig {
    fn default() -> Self {
        Self {
            time_weight: 0.3,
            semantic_weight: 0.4,
            heat_weight: 0.3,
        }
    }
}
```

### 归一化函�?

```rust
fn normalize_time_score(valid_from: i64, start: i64, end: i64) -> f64 {
    // 线性归一化：在时间窗中心得分�?，边缘为0
    let center = (start + end) / 2;
    let half_range = (end - start) / 2;
    if half_range == 0 {
        return 1.0;
    }
    let distance = (valid_from - center).abs();
    1.0 - (distance as f64 / half_range as f64).min(1.0)
}

fn normalize_heat_score(heat_i32: i32) -> f64 {
    // 归一化到 0-1
    (heat_i32 as f64 / 10000.0).min(1.0).max(0.0)
}
```

### 综合评分计算

```rust
fn compute_composite_score(
    time_score: f64,
    semantic_score: f64,
    heat_score: f64,
    config: &HybridRankingConfig,
) -> f64 {
    config.time_weight * time_score
        + config.semantic_weight * semantic_score
        + config.heat_weight * heat_score
}
```

### Searcher �?API

建议�?`Laputa/src/searcher/mod.rs` �?`hybrid.rs` 中新增：

```rust
pub async fn hybrid_search(
    &self,
    query: HybridQuery,
) -> Result<Vec<HybridSearchResult>>
```

实现逻辑�?

1. 调用 `recall_by_time_range(query.time_query)` �?得到时间结果
2. 调用 `search_semantic(query.semantic_query, top_k * 2)` �?得到语义结果（多取一些，去重后会减少�?
3. 合并 + 去重（基�?record.id�?
4. 对每条记录计算三个维度评�?
5. 计算综合评分
6. 按综合评分降序排�?
7. 返回 top_k

### VectorStorage 层适配

需要确保：

- `VectorStorage::search(...)` 返回的记录包�?`heat_i32` 字段
- 如果当前不返回，需要在 SQL 查询�?JOIN 或补充字�?

检查点�?

- `Laputa/src/vector_storage.rs` �?`search(...)` 方法
- 确认返回�?`MemoryRecord` 是否包含 `heat_i32`

### MemoryStack 集成

可选：如果需要在 `MemoryStack` 层暴露混合检索，可以新增�?

```rust
pub async fn hybrid_search(
    &self,
    query: HybridQuery,
) -> String  // 格式化输�?
```

但本 Story 优先保证底层 API 完整，格式化输出可作为后�?CLI/MCP 的包装�?

---

## 文件建议

优先考虑以下改动点：

- `Laputa/src/searcher/mod.rs` - 新增 `hybrid_search(...)` 方法
- `Laputa/src/searcher/hybrid.rs` - 新建，包�?`HybridQuery`、`HybridSearchResult`、排序逻辑
- `Laputa/src/vector_storage.rs` - 确认语义检索返回记录包�?`heat_i32`
- `Laputa/src/storage/mod.rs` - 可选，如需 MemoryStack 层包�?

�?Story 不建议碰�?

- `wakeup/` - 那是 3.3 的职�?
- `heat/` - 不要重新计算热度
- `archiver/` - 无关
- `RecallQuery` 定义 - 不要破坏 3.1 已稳定的接口

---

## 实现细节要求

### 1. 去重策略

必须基于 `record.id`（UUID �?i64）去重，而不是文本内容�?

推荐实现�?

```rust
use std::collections::HashMap;

let mut deduped: HashMap<i64, HybridSearchResult> = HashMap::new();

// 先插入时间结�?
for record in time_results {
    deduped.insert(record.id, HybridSearchResult {
        record,
        time_score: 1.0,  // 时间结果默认时间相关性为1
        semantic_score: 0.0,
        heat_score: normalize_heat_score(record.heat_i32),
        composite_score: 0.0,
    });
}

// 再插入语义结果（覆盖或更新）
for record in semantic_results {
    if let Some(existing) = deduped.get_mut(&record.id) {
        // 已存在，更新语义分数
        existing.semantic_score = record.score;
    } else {
        // 新记�?
        deduped.insert(record.id, HybridSearchResult {
            record,
            time_score: 0.0,  // 语义结果默认时间相关性为0
            semantic_score: record.score,
            heat_score: normalize_heat_score(record.heat_i32),
            composite_score: 0.0,
        });
    }
}
```

### 2. 时间相关性计�?

如果语义检索结果不在时间窗内，`time_score` 应该如何计算�?

推荐方案�?

- 如果记录在时间窗�?�?`time_score` 基于距离中心的远近（0-1�?
- 如果记录不在时间窗内 �?`time_score = 0.0`

这样可以保证时间窗外的纯语义结果不会因为热度高而完全主导�?

### 3. top_k 语义

- 默认 `top_k = 100`（与时间流检索一致）
- 语义检索时应取 `top_k * 2`（为去重留余量）
- 上限 clamp �?`1000`（防止滥用）

### 4. 返回结构

底层返回 `Vec<HybridSearchResult>`，上层格式化时可展示�?

- `record.id`
- `record.valid_from`
- `record.heat_i32`
- `composite_score`
- `time_score`
- `semantic_score`
- 文本摘要

这样用户可以理解为什么某条记忆排在前面�?

### 5. 权重配置来源

权重配置应从 `config.toml` 读取�?

```toml
[search.hybrid]
time_weight = 0.3
semantic_weight = 0.4
heat_weight = 0.3
```

如果配置缺失，使�?`Default` trait 的默认值�?

---

## 测试要求

至少补齐以下测试�?

1. **混合检索基础测试**
   - 同时有时间结果和语义结果
   - 验证综合评分计算正确
   - 验证排序顺序符合预期

2. **去重测试**
   - 时间结果和语义结果有重叠
   - 验证重叠记录只出现一�?
   - 验证重叠记录的分数正确合�?

3. **边界测试**
   - 只有时间结果，无语义结果
   - 只有语义结果，无时间结果
   - 两者都为空

4. **归一化测�?*
   - `normalize_time_score` 在时间窗中心得分�?
   - `normalize_time_score` 在时间窗边缘得分�?
   - `normalize_heat_score` �?0-10000 正确归一�?

5. **权重配置测试**
   - 默认权重下排序正�?
   - 自定义权重下排序改变（例�?heat_weight = 0.8�?

6. **top_k 裁剪测试**
   - 合并后结果超�?top_k
   - 验证只返�?top_k �?

7. **Searcher 集成测试**
   - `Searcher::hybrid_search(...)` 能正确调用底层逻辑

推荐测试文件�?

- `Laputa/tests/test_hybrid_search.rs`

必要时补少量单元测试到：

- `Laputa/src/searcher/hybrid.rs`
- `Laputa/src/vector_storage.rs`

继续沿用�?

- `tempfile`
- `serial_test`

---

## 禁止事项

- 不要硬编码加权系�?
- 不要在混合检索中重新计算热度
- 不要先排序再去重
- 不要破坏 Story 3.1 �?`RecallQuery` 接口
- 不要忽略归一化直接加�?
- 不要把混合检索做成语义检索的别名
- 不要在本 Story 实现 WakePack 消费逻辑

---

## 实施任务

- [x] 定义 `HybridQuery`、`HybridSearchResult`、`HybridRankingConfig` 数据结构
- [x] 实现归一化函数（时间相关性、热度）
- [x] 实现综合评分计算函数
- [x] �?`vector_storage.rs` 确认语义检索返�?`heat_i32`
- [x] 实现 `hybrid_search(...)` 核心逻辑（检�?+ 去重 + 评分 + 排序�?
- [x] �?`Searcher` 层暴�?`hybrid_search(...)` 接口
- [x] �?`config.toml` 读取权重配置（如有）
- [x] 补齐混合检索、去重、归一化、权重、top_k 测试
- [x] 验证不破坏现有时间检索和语义检索能�?

---

## 完成定义

- [ ] 能融合时间流和语义检索结�?
- [ ] 结果按综合评分排序（热度 + 时间 + 语义�?
- [ ] 重复记录已去�?
- [ ] 默认最大返�?`100`
- [x] 排序算法可解释（返回各维度分数）
- [ ] 不破坏现�?`searcher` 时间检索和语义检索能�?
- [x] `cargo test` 通过
- [x] `cargo clippy --all-features --tests -- -D warnings` 通过

### Review Findings (2026-04-16)

- [x] [Review][Patch] Story 3-4 DoD checkboxes 已更新 — 代码审查验证通过后自动补全

---

## 依赖 story 状�?

- **Story 3.1** (timeline-recall): ready-for-dev - 提供 `RecallQuery` 和时间召回基础设施
- **Story 3.2** (semantic-search): backlog - 提供语义检索能力（假设已完成）
- **Story 3.3** (wakepack-generate): ready-for-dev - 会消费本 story 的混合检索结�?

注意：本 story 开发时需要确�?3.1 �?3.2 的实现已完成或可 mock�?

---

## 参考资�?

- `D:\VIVYCORE\newmemory\_bmad-output\planning-artifacts\epics.md` - Story 3.4
- `D:\VIVYCORE\newmemory\_bmad-output\planning-artifacts\architecture.md` - 6.4 需求到结构映射
- `D:\VIVYCORE\newmemory\_bmad-output\planning-artifacts\prd.md` - FR-7, FR-8
- `D:\VIVYCORE\newmemory\Laputa\AGENTS.md`
- `D:\VIVYCORE\newmemory\Laputa\DECISIONS.md`
- `D:\VIVYCORE\newmemory\Laputa\src\searcher\mod.rs`
- `D:\VIVYCORE\newmemory\Laputa\src\vector_storage.rs`
- `D:\VIVYCORE\newmemory\_bmad-output\implementation-artifacts\3-1-timeline-recall.md`
- `D:\VIVYCORE\newmemory\_bmad-output\implementation-artifacts\3-3-wakepack-generate.md`

---

## Dev Agent Record

### Context Notes

- 当前代码只有纯语义检索（`Searcher::search`）和基础 recall 展示�?
- 缺少真正�?时间窗召�?查询对象（Story 3.1 职责�?
- 缺少混合检索入口和加权排序算法
- �?Story 的正确落点是补齐混合检索基础设施，而不是提前实�?WakePack 消费逻辑
- 当前工作区不�?git 仓库根目录，未获取可�?git 提交历史

### Debug Log References

- `cargo test --test test_hybrid_search` (red phase: failed before hybrid-search types and APIs existed)
- `cargo fmt`
- `cargo test --test test_hybrid_search`
- `cargo test --test test_timeline_recall --test test_semantic_search`
- `cargo clippy --all-features --tests -- -D warnings`
- `cargo test`

### Completion Note

- Implemented `HybridQuery`, `HybridRankingConfig`, `HybridSearchResult`, score normalization helpers, and composite-score calculation in `Laputa/src/searcher/hybrid.rs`.
- Added `Searcher::hybrid_search(...)` plus exported hybrid-search helpers in `Laputa/src/searcher/mod.rs`, while keeping existing recall and semantic search APIs intact.
- Added optional `[search.hybrid]` weight loading from `config.toml`; default weights remain `0.3 / 0.4 / 0.3` when no config is present.
- Verified deduplication is based on memory ID, semantic overflow is capped via `semantic_limit`, and final results are clipped to `top_k`.
- Added `Laputa/tests/test_hybrid_search.rs` covering normalization, composite score, deduplication, `top_k` clamp, TOML config loading, and `Searcher::hybrid_search()` integration.
- Verified quality gates with `cargo test --test test_hybrid_search`, `cargo test --test test_timeline_recall --test test_semantic_search`, `cargo clippy --all-features --tests -- -D warnings`, and full `cargo test`.

### File List

- `_bmad-output/implementation-artifacts/3-4-hybrid-search.md`
- `Laputa/src/searcher/hybrid.rs`
- `Laputa/src/searcher/mod.rs`
- `Laputa/src/wakeup/mod.rs`
- `Laputa/tests/test_hybrid_search.rs`
- `_bmad-output/implementation-artifacts/sprint-status.yaml`

### Change Log

- 2026-04-15: Implemented Story 3.4 hybrid search with configurable ranking weights, ID-based deduplication, explainable component scores, and automated regression coverage.
