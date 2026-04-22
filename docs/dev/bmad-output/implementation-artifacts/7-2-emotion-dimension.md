# Story 7.2: 情绪维度记录
**Story ID:** 7.2  
**Story Key:** 7-2-emotion-dimension  
**Status:** done  
**Created:** 2026-04-14  
**Project:** 天空之城 (Laputa)

---

## 用户故事

As a **系统**,
I want **记录记忆的情绪效价和激活程度**,
So that **情感维度可用于检索、排序和后续唤醒摘要生成**。

---

## 验收标准

- **Given** 记忆写入或后续情绪标记发生
- **When** 设置情绪维度
- **Then** `emotion_valence` 落在 `-100..=100`
- **And** `emotion_arousal` 落在 `0..=100`

- **Given** 情绪维度已写入
- **When** 系统读取该记忆
- **Then** 情绪字段可从统一 `MemoryRecord` 读取
- **And** 不需要额外的独立 emotion store

- **Given** 用户或系统提供 emotion 输入
- **When** 写入流程解析 emotion 标记
- **Then** emotion 名称应优先沿用 `dialect::EMOTION_CODES` 的既有词典与映射语义
- **And** valence / arousal 作为独立数值维度落到 `memories` 主表

- **Given** 情绪维度存在
- **When** 执行情绪驱动检索或排序
- **Then** 系统具备可复用的情绪过滤或排序入口
- **And** 后续 search / wakeup / MCP 可以在不重写 SQL 的情况下消费这些能力

- **And** 自动化测试覆盖边界值、越界输入处理、默认值、写入持久化、读取回传，以及基于情绪维度的最小查询能力

扩展约束：
- 本 Story 负责“情绪维度记录与最小可用查询入口”，不负责完整情绪胶囊生成
- 本 Story 不得把 valence / arousal 写进自由文本或 JSON 角落里冒充已实现
- 本 Story 不得重复定义第二套 emotion 字段，因为 `MemoryRecord` 和 SQLite schema 已经具备这两个字段

---

## Epic 上下文

### Epic 7 目标

Epic 7 聚焦“关系与情绪记忆”，覆盖：
- 关系变化与共振度
- 情绪锚定（`valence + arousal`）

`7.2 emotion-dimension` 是 Epic 7 的第二个 Story，职责是把已经出现在数据模型中的 `emotion_valence` / `emotion_arousal` 从“仅有字段”推进为“可被写入、读取、过滤和后续消费的能力”。

### 与前置 Story 的关系

- `1.3 memoryrecord-extension`
  - 已把 `emotion_valence`、`emotion_arousal` 放进统一记忆模型
  - 本 Story 不应重新建模，只应打通这些字段的行为路径
- `2.1 diary-write`
  - Epic 文档要求 `diary.write(content, tags, emotion)` 支持 emotion 输入
  - 当前 repo 的 `Diary::write_entry()` 仍未接入该语义
  - 本 Story 应明确这是后续接入点之一
- `2.3 emotion-anchor`
  - 已定义情绪锚点的业务含义：影响热度并记录 valence/arousal
  - 本 Story 是其底层数据维度支撑，不负责完整锚点业务链路
- `3.3 wakepack-generate`
  - 唤醒包后续会消费情绪维度进行摘要和权重判断
  - 本 Story 需要给 wakeup 留下稳定读取入口
- `7.1 relation-node`
  - 已把 UPSP 融合概念中的 resonance 固化到关系层
  - 本 Story 补齐剩余两项情绪维度：valence / arousal

---

## 现有代码情报

### 已存在且必须复用的能力

1. `Laputa/src/storage/memory.rs`
   - 已定义：
     - `emotion_valence: i32`
     - `emotion_arousal: u32`
   - 已提供 `update_emotion(valence, arousal)`，并带有 clamp 行为：
     - valence 限制到 `-100..=100`
     - arousal 限制到 `0..=100`

2. `Laputa/src/storage/memory.rs::ensure_memory_schema()`
   - 已确保 `memories` 表具备情绪字段
   - 说明 schema 层不需要重新设计，只需要复用

3. `Laputa/src/vector_storage.rs`
   - 当前插入 `memories` 时已写入：
     - `emotion_valence`
     - `emotion_arousal`
   - 但默认都是 `0`
   - 查询路径已经会把两个字段读回到 `MemoryRecord`

4. `Laputa/src/dialect/mod.rs`
   - 已有 `EMOTION_CODES`
   - Epic / 架构明确要求沿用这套情绪编码核心
   - 该模块适合作为 emotion 名称到语义编码的来源，而不是另造字典

5. `Laputa/tests/test_memory_record.rs`
   - 已覆盖默认值和 clamp 行为
   - 可以作为本 Story 持久化与查询测试的基础参照

### 当前缺口与实现风险

1. **只有字段，没有写入语义**
   - 当前模型和 schema 都支持情绪维度
   - 但当前主要写入路径 `Searcher::add_memory()` / `VectorStorage::add_memory()` 仍只写默认 `0/0`
   - 如果不扩展写入接口，本 Story 就会停留在“结构存在但业务不可用”

2. **Diary 输入与 Epic 不一致**
   - Epic 2.1 写的是 `diary.write(content, tags, emotion)`
   - 当前 `Diary::write_entry(agent, content)` 没有 emotion 参数
   - 本 Story 需要明确 emotion 应通过统一记忆写入路径接入，而不是把 diary 表单独升级成另一套情绪存储

3. **缺少最小查询入口**
   - `vector_storage.rs` 现有查询按时间、wing/room 或语义搜索进行
   - 还没有情绪过滤或情绪排序入口
   - 后续 wakeup / MCP / search 如果都自行拼 SQL，会导致重复实现

4. **EMOTION_CODES 与数值维度可能被混淆**
   - `EMOTION_CODES` 是类别编码
   - valence / arousal 是数值维度
   - 本 Story 不能用一个替代另一个；二者应并存而非互斥

---

## 架构与实现约束

### 1. 统一数据模型

- 情绪维度必须继续附着在统一 `MemoryRecord`
- 不要创建独立 `emotion_memories` 表
- 不要在 `DiaryEntry` 或 `KnowledgeGraph` 里偷偷保存另一套主情绪真值

### 2. 字段语义约束

- `emotion_valence`
  - 范围：`-100..=100`
  - 语义：情感效价，负向到正向

- `emotion_arousal`
  - 范围：`0..=100`
  - 语义：激活程度，低唤醒到高唤醒

- 默认值：
  - 无显式情绪时默认为 `0/0`

### 3. EMOTION_CODES 的使用边界

- `dialect::EMOTION_CODES` 用于 emotion 名称或标签的标准化映射
- 但 `emotion_valence` / `emotion_arousal` 必须作为显式数值落库
- 不要把 `EMOTION_CODES` 的缩写写进数值字段

### 4. 写入边界

- 主要情绪写入入口应以现有主存储路径为中心：
  - `VectorStorage`
  - `Searcher`
  - 后续 `mark_emotion_anchor(...)`
- 不要在 CLI、MCP、Diary 三处各自写一套更新逻辑

### 5. 查询边界

- 至少提供一个可复用的情绪查询入口，例如：
  - 按 valence / arousal 过滤
  - 按情绪强度排序
- 后续 search / wakeup / MCP 应调用该入口，而不是内联 SQL

### 6. 错误与验证

- 越界输入策略必须统一
- 当前模型层已经采用 clamp 行为；如果上层选择校验失败，应有明确理由并全链路一致
- 在没有明确反证前，建议沿用现有模型层 clamp 语义，避免多层行为冲突

---

## 推荐实现方案

### 推荐领域接口

优先在 `VectorStorage` 或一个轻量情绪服务层中暴露清晰接口，例如：

```rust
pub fn update_memory_emotion(&self, id: i64, valence: i32, arousal: u32) -> Result<()>;
pub fn list_memories_by_emotion(
    &self,
    min_valence: Option<i32>,
    max_valence: Option<i32>,
    min_arousal: Option<u32>,
    max_arousal: Option<u32>,
    limit: usize,
) -> Result<Vec<MemoryRecord>>;
```

如果团队希望把语义更清晰，可再加：

```rust
pub fn list_emotionally_salient_memories(&self, limit: usize) -> Result<Vec<MemoryRecord>>;
```

### 推荐写入流程

1. 上层接收 emotion 输入：
   - 可是 emotion 名称
   - 可是直接 valence/arousal
2. 若传入 emotion 名称，先通过 `EMOTION_CODES` 做标准化
3. 将数值维度映射为 valence/arousal
4. 通过统一写入入口更新 `memories` 表的两个字段
5. 读取时直接回传到 `MemoryRecord`

### 推荐查询能力

至少落地其中一种：

1. **过滤式查询**
   - 例：`valence >= 50`
   - 例：`arousal >= 70`

2. **排序式查询**
   - 例：按 `ABS(emotion_valence)` 或 `emotion_arousal DESC` 排序

3. **组合式查询**
   - 例：高唤醒且强负向 / 高唤醒且强正向

对本 Story 来说，关键不是一次性做完整情绪搜索 DSL，而是留下一个稳定入口给后续 Story 复用。

### 与 diary / emotion-anchor 的衔接建议

- `Diary` 表仍可保持轻量事件日志定位
- 真正用于检索、排序和唤醒的情绪维度应落在 `memories` 主表
- 后续 `diary.write(..., emotion)`、`mark_emotion_anchor(...)` 最终都应复用本 Story 的统一情绪更新接口

---

## 特别防错说明

### 1. 不要重复定义情绪字段

当前 repo 已经在以下位置有情绪字段：
- `storage/memory.rs`
- `storage/sqlite.rs`
- `vector_storage.rs`

本 Story 的重点是“打通行为”，不是“再加字段”。

### 2. 不要把 emotion category 当作数值维度

- `joy` / `fear` / `love` 这类标签属于分类信号
- valence / arousal 属于数值维度
- 需要同时保留，而不是互相覆盖

### 3. 不要把情绪搜索藏进特定调用方

- 如果只在 MCP 或 wakeup 某个调用里临时实现 SQL，本 Story价值会很低
- 必须沉到底层可复用接口

### 4. 不要破坏现有 clamp 语义

`LaputaMemoryRecord::update_emotion()` 已经提供稳定边界处理。除非团队有明确架构变更，否则 Story 应沿用这一行为，避免模型层和存储层出现两套规则。

---

## 测试要求

至少补齐以下测试：

1. 持久化更新
   - 更新 emotion 后重新读取，值正确落库

2. 边界值
   - `-100`、`100`、`0`
   - `0`、`100`

3. 越界输入
   - 小于 `-100`
   - 大于 `100`
   - 大于 `100` 的 arousal
   - 行为与模型层 clamp 规则一致

4. 默认值
   - 未设置 emotion 时仍为 `0/0`

5. 最小查询能力
   - 可按 valence / arousal 过滤或排序返回结果

6. 不破坏既有检索
   - 现有 `search()` / `get_memory_by_id()` / `get_memories()` 返回结构仍包含情绪字段

推荐测试文件：
- `Laputa/tests/test_emotion_dimension.rs`

可复用现有测试：
- `Laputa/tests/test_memory_record.rs`

---

## 禁止事项

- 不要新增独立 emotion store
- 不要只改 schema 而不打通写入和查询
- 不要把 emotion 信息仅写入自由文本
- 不要在调用方内联 SQL 实现唯一的情绪过滤逻辑
- 不要把 `EMOTION_CODES` 直接塞进 `emotion_valence` / `emotion_arousal`

---

## 实施任务

- [x] 在主存储路径中补齐情绪更新入口
- [x] 在主存储路径中补齐最小情绪查询或排序入口
- [x] 明确 emotion 名称到 `EMOTION_CODES` 与数值维度的映射边界
- [x] 保持 `MemoryRecord` 为统一情绪数据载体
- [x] 为后续 `diary.write(..., emotion)` 与 `mark_emotion_anchor(...)` 留下可复用接口
- [x] 补齐 `Laputa/tests/test_emotion_dimension.rs`
- [x] 运行 `cargo test`
- [x] 运行 `cargo clippy --all-features --tests -- -D warnings`

---

## 完成定义

- [x] 情绪维度可被统一写入 `memories` 主表
- [x] 情绪维度可从统一 `MemoryRecord` 读取
- [x] 至少具备一个稳定的情绪过滤或排序入口
- [x] 越界输入行为与既有模型层规则一致
- [x] 后续 wakeup / search / MCP 能复用该能力
- [x] 自动化测试覆盖边界、持久化与最小查询
- [x] `cargo test` 通过
- [x] `cargo clippy --all-features --tests -- -D warnings` 通过

---

## 参考资料

- `D:\VIVYCORE\newmemory\_bmad-output\planning-artifacts\epics.md`
- `D:\VIVYCORE\newmemory\_bmad-output\planning-artifacts\prd.md`
- `D:\VIVYCORE\newmemory\_bmad-output\planning-artifacts\architecture.md`
- `D:\VIVYCORE\newmemory\_bmad-output\implementation-artifacts\7-1-relation-node.md`
- `D:\VIVYCORE\newmemory\Laputa\src\storage\memory.rs`
- `D:\VIVYCORE\newmemory\Laputa\src\storage\sqlite.rs`
- `D:\VIVYCORE\newmemory\Laputa\src\vector_storage.rs`
- `D:\VIVYCORE\newmemory\Laputa\src\diary\mod.rs`
- `D:\VIVYCORE\newmemory\Laputa\src\dialect\mod.rs`
- `D:\VIVYCORE\newmemory\Laputa\src\searcher\mod.rs`
- `D:\VIVYCORE\newmemory\Laputa\src\wakeup\mod.rs`
- `D:\VIVYCORE\newmemory\Laputa\tests\test_memory_record.rs`

---

## Dev Agent Record

### Agent Model Used

GPT-5

### Debug Log References

- 2026-04-16: 新增 `canonical_emotion_code()`，统一复用 `EMOTION_CODES` 做 emotion 名称标准化。
- 2026-04-16: 在 `VectorStorage` 新增 `update_memory_emotion()` 与 `list_memories_by_emotion()`，并通过 `Searcher` 暴露复用入口。
- 2026-04-16: 新增 `Laputa/tests/test_emotion_dimension.rs`，覆盖默认值、clamp、持久化、过滤与排序。
- 2026-04-16: 为通过仓库级 `clippy`，将 `RelationKind::from_str` 重构为标准 `FromStr` 实现并更新调用点。
- 2026-04-16: 验证完成：`cargo clippy --all-features --tests -- -D warnings`、`cargo test --quiet`。

### Completion Notes List

- 已实现统一情绪写入入口 `VectorStorage::update_memory_emotion()`，复用 `MemoryRecord::update_emotion()` 的 clamp 语义写回 `memories` 主表。
- 已实现稳定的情绪查询入口 `VectorStorage::list_memories_by_emotion()`，支持 valence/arousal 过滤以及按最近、valence、arousal、绝对 valence 排序。
- 已在 `Searcher` 暴露同名接口，给后续 `diary.write(..., emotion)`、`mark_emotion_anchor(...)`、wakeup、search、MCP 复用，避免调用方重复拼 SQL。
- 已抽取 `dialect::canonical_emotion_code()`，让 diary 的 emotion 名称标准化直接复用既有 `EMOTION_CODES` 字典，不把类别编码混入数值维度字段。
- 已新增 `Laputa/tests/test_emotion_dimension.rs`，并确认完整 `cargo test --quiet` 与 `cargo clippy --all-features --tests -- -D warnings` 均通过。

### File List

- `Laputa/src/dialect/mod.rs`
- `Laputa/src/diary/mod.rs`
- `Laputa/src/searcher/mod.rs`
- `Laputa/src/vector_storage.rs`
- `Laputa/src/knowledge_graph/mod.rs`
- `Laputa/src/knowledge_graph/relation.rs`
- `Laputa/tests/test_emotion_dimension.rs`
- `_bmad-output/implementation-artifacts/7-2-emotion-dimension.md`

### Change Log

- 2026-04-16: 新增统一情绪写入/查询接口、emotion 名称标准化 helper、`test_emotion_dimension.rs`，并修复 `RelationKind` 的 `clippy` 阻塞以通过仓库级验证。
- 2026-04-16: 三层代码审查完成（Blind Hunter + Edge Case Hunter + Acceptance Auditor）。所有 AC 验证通过。

## Review Findings

### Deferred (Pre-existing)

- [x] [Review][Defer] update_emotion静默clamp无越界警告 [memory.rs:91-94] — 用户决策：方案A（添加log::warn!()），暂不修改代码，推迟原因：需与Resonance策略统一时同步修改
- [x] [Review][Defer] Drop实现忽略WAL checkpoint错误 [vector_storage.rs:1279-1283] — deferred, 设计决策
- [x] [Review][Defer] 错误被静默吞没(embed_single) [searcher/mod.rs:137-139] — deferred, 优雅降级设计
- [x] [Review][Defer] canonical_emotion_code空字符串处理未文档化 [dialect/mod.rs:9-15] — deferred, 行为正确
- [x] [Review][Defer] limit=0边界处理不一致 [vector_storage.rs:716-719] — deferred, 两套策略都有合理理由
- [x] [Review][Defer] EmotionQuery参数无验证 [vector_storage.rs:70-82] — deferred, 查询参数逻辑验证非必需
- [x] [Review][Defer] 公共字段暴露内部状态(VectorStorage) [vector_storage.rs:127-131] — deferred, 设计决策
- [x] [Review][Defer] Regex未缓存编译 [dialect/mod.rs:334] — deferred, 性能优化项
