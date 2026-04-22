# Story 2.3: 情绪锚定记录

**Story ID:** 2.3  
**Story Key:** 2-3-emotion-anchor  
**Status:** done  
**Created:** 2026-04-14  
**Project:** 天空之城 (Laputa)

---

## 用户故事

As a **用户**,
I want **为已存在的记忆条目设置情绪锚点**,
So that **重要情感记忆获得额外保留权重，并在后续检索、唤醒和生命周期判断中更稳定地被保留与召回**。

---

## 验收标准

- **Given** 目标记忆已经存在于 SQLite `memories` 表中
- **When** 调用 `mark_emotion_anchor(memory_id, valence, arousal)`
- **Then** 目标记录的 `heat_i32` 增加 `2000`，并且结果被裁剪到上限 `10000`
- **And** `emotion_valence` 被设置为指定值，范围限制为 `-100..100`
- **And** `emotion_arousal` 被设置为指定值，范围限制为 `0..100`
- **And** 本次变更被持久化到当前主存储路径，而不是只停留在内存对象中
- **And** 当 `memory_id` 不存在时，返回明确错误，不产生任何写入副作用
- **And** 自动化测试覆盖热度提升、上限裁剪、情绪边界裁剪、持久化结果和缺失记录错误路径

扩展约束：

- 本 Story 聚焦“对已存在记忆做人工情绪强化”，不重新实现写入入口、MemoryGate、HeatService 或 CLI/MCP 暴露
- “保鲜 7 天”在架构文档中是热度策略语义提示；本 Story 只需要落地显式锚定效果 `heat_i32 += 2000`
- 必须复用现有 `LaputaMemoryRecord` / `VectorStorage` 能力，禁止平行造一套新的 emotion-anchor 存储路径

---

## Epic 上下文

### Epic 2 目标

Epic 2 负责“日记与记忆输入”，覆盖：

- `FR-2` 日记写入
- `FR-3` 记忆筛选
- `FR-10` 情绪锚定

`2.3 emotion-anchor` 位于写入与筛选之后，作用是对“已经保留下来的记忆”进行显式情绪强化。它不是初始写入的一部分，也不是筛选逻辑的一部分，而是用户对既有记忆施加的二次保留权重。

### 与相邻 Story 的关系

- `2.1 diary-write` 负责把输入写入 `MemoryRecord` 主路径，并建立默认 `heat_i32 = 5000` 与 emotion 基础字段
- `2.2 memory-filter-merge` 负责写入期筛选、重复合并和 `reason` 等解释性信息
- `2.3 emotion-anchor` 只对“已经存在且可定位的记录”做人工强化，不反向耦合 `2.2` 的筛选职责

---

## 现有代码情报

### 必须复用的现有能力

1. `Laputa/src/storage/memory.rs`
   - 已有 `LaputaMemoryRecord`
   - 已有 `heat_i32`
   - 已有 `emotion_valence` / `emotion_arousal`
   - 已有 `update_emotion(valence, arousal)` 边界裁剪逻辑
   - 已有 `with_updated_heat()`，体现“不要直接散落式修改热度语义”的方向

2. `Laputa/src/vector_storage.rs`
   - 已有 `get_memory_by_id(id)`
   - 已有 `touch_memory(id)`
   - 已有 `update_memory_summary(id, new_summary)`
   - 已有完整 `memories` 表读写与 row mapping
   - 是本 Story 最自然、最小改动的落点

3. `Laputa/tests/test_memory_record.rs`
   - 已验证 emotion 边界裁剪
   - 已验证 heat 转换与默认值
   - 可直接沿用测试风格和 fixture 模式

### 从前序 Story 继承的结论

来自 `1-3-memoryrecord-extension`：

- 热度字段已经固定为 `heat_i32`，禁止引入新的 `heat: f64` 持久化字段
- schema 已经具备 `emotion_valence`、`emotion_arousal`、`heat_i32`
- 后续 Story 应在现有 schema 和 `VectorStorage` 基础上扩展行为，而不是再定义第二份记录模型

---

## 架构与实现约束

### 1. 职责边界

本 Story 只负责“对已存在 memory 的情绪强化更新”，不负责：

- 新记忆创建
- 过滤、合并、丢弃判断
- 周期性热度衰减
- Archive candidate 计算
- CLI `--emotion-anchor`
- MCP tools 暴露

如果在实现时发现需要这些能力，应该只做最小可复用接口预留，不要在本 Story 中提前落地完整功能。

### 2. 热度语义

- 增量规则固定为 `heat_i32 += 2000`
- 上限必须裁剪到 `10000`
- 不需要在本 Story 中实现“7 天衰减减半”的完整机制
- 不要改动当前默认热度 `5000` 的既有语义

推荐计算方式：

```rust
let new_heat = (record.heat_i32 + 2000).min(10_000);
```

### 3. 情绪边界语义

- `valence` 范围固定为 `-100..100`
- `arousal` 范围固定为 `0..100`
- 不要在多个位置重复写边界裁剪逻辑；优先复用 `LaputaMemoryRecord::update_emotion()`

### 4. 错误处理

- 缺失记录必须返回明确错误，且调用方能区分“记录不存在”和“数据库执行失败”
- 若继续沿用 `anyhow::Result`，错误消息必须具备可判别性
- 不允许“更新 0 行但仍返回成功”

### 5. 持久化要求

- 变更必须写回 `memories` 表
- 更新后重新读取记录时，`heat_i32` / `emotion_valence` / `emotion_arousal` 必须一致
- 不允许只更新内存对象、但忘记写回 SQLite

---

## 推荐实现方案

### 推荐落点

优先在 `Laputa/src/vector_storage.rs` 为 `VectorStorage` 增加面向现有记录的更新方法，例如：

```rust
pub fn mark_emotion_anchor(
    &self,
    memory_id: i64,
    valence: i32,
    arousal: u32,
) -> Result<MemoryRecord>
```

### 推荐流程

1. 调用 `get_memory_by_id(memory_id)` 读取目标记录
2. 若记录不存在，立即返回错误
3. 在内存中基于现有记录计算：
   - `new_heat = min(record.heat_i32 + 2000, 10000)`
   - 复用 `update_emotion()` 得到裁剪后的 `valence/arousal`
4. 用单条 `UPDATE` 或等价事务完成持久化
5. 重新读取记录并返回，便于测试和调用方使用

推荐 SQL 形态：

```sql
UPDATE memories
SET heat_i32 = ?1,
    emotion_valence = ?2,
    emotion_arousal = ?3
WHERE id = ?4
```

### 为什么不建议放在别处

- 放在 `storage/memory.rs`：适合模型辅助方法，不适合数据库写入
- 放在 `cli/` 或 `mcp_server/`：会把领域能力绑死到接口层
- 新建独立 `emotion_anchor.rs`：除非后续要扩展成完整服务，否则当前会制造过早抽象

---

## 测试要求

至少补齐以下测试：

1. 成功路径
   - 对存在记录执行 emotion anchor
   - `heat_i32` 增加 `2000`
   - `emotion_valence` / `emotion_arousal` 被写入

2. 上限裁剪
   - 当原始热度接近上限时，结果不超过 `10000`

3. 边界裁剪
   - `valence < -100` 被裁剪到 `-100`
   - `valence > 100` 被裁剪到 `100`
   - `arousal > 100` 被裁剪到 `100`

4. 缺失记录
   - 不存在的 `memory_id` 返回错误
   - 数据库中其他记录不受影响

5. 持久化验证
   - 更新后重新查询，字段值保持一致

推荐测试文件：

- `Laputa/tests/test_emotion_anchor.rs`

也可在必要时扩展现有测试，但不要把本 Story 的测试语义全部塞进 `test_memory_record.rs`，否则会混淆模型测试与存储行为测试边界。

---

## 禁止事项

- 不要引入新的浮点热度持久化列
- 不要在本 Story 中实现完整 `HeatService`
- 不要顺手实现 CLI `mark --emotion-anchor`
- 不要顺手实现 MCP tool handler
- 不要复制一份新的 emotion 边界裁剪逻辑
- 不要对不存在的记录静默成功

---

## 实施任务

- [x] 在 `Laputa/src/vector_storage.rs` 中新增 emotion anchor 更新入口
- [x] 复用 `LaputaMemoryRecord` 的 emotion 边界裁剪语义
- [x] 按 `heat_i32 += 2000` 实现热度强化并裁剪上限
- [x] 持久化更新 `heat_i32` / `emotion_valence` / `emotion_arousal`
- [x] 对缺失记录返回明确错误
- [x] 补齐成功、边界、持久化和错误路径测试
- [x] 运行 `cargo test`
- [x] 运行 `cargo clippy --all-features --tests -- -D warnings`

---

## 完成定义

- [x] 已存在记录可被显式标记 emotion anchor
- [x] `heat_i32` 正确增加 `2000` 且不超过 `10000`
- [x] `emotion_valence` / `emotion_arousal` 正确裁剪并持久化
- [x] 缺失记录返回明确错误
- [x] 不引入新持久化热度模型或重复实现路径
- [x] 自动化测试通过
- [x] `cargo clippy --all-features --tests -- -D warnings` 通过

---

## 参考资料

- `D:\VIVYCORE\newmemory\_bmad-output\planning-artifacts\epics.md`
- `D:\VIVYCORE\newmemory\_bmad-output\planning-artifacts\architecture.md`
- `D:\VIVYCORE\newmemory\_bmad-output\planning-artifacts\prd.md`
- `D:\VIVYCORE\newmemory\Laputa\DECISIONS.md`
- `D:\VIVYCORE\newmemory\Laputa\AGENTS.md`
- `D:\VIVYCORE\newmemory\Laputa\src\storage\memory.rs`
- `D:\VIVYCORE\newmemory\Laputa\src\vector_storage.rs`
- `D:\VIVYCORE\newmemory\Laputa\tests\test_memory_record.rs`
- `D:\VIVYCORE\newmemory\_bmad-output\implementation-artifacts\1-3-memoryrecord-extension.md`
- `D:\VIVYCORE\newmemory\_bmad-output\implementation-artifacts\2-1-diary-write.md`
- `D:\VIVYCORE\newmemory\_bmad-output\implementation-artifacts\2-2-memory-filter-merge.md`

---

## Dev Agent Record

### Context Notes

- 当前代码已经具备 emotion 字段和热度字段，缺口主要在“对已存在记录的显式更新能力”
- 相邻 Story 已把写入期和筛选期职责拆开，本 Story 不能把边界重新揉乱
- 当前最合理的落点是 `VectorStorage`，不是接口层，也不是新的抽象层

### Debug Log

- `cargo test --test test_emotion_anchor`
- `cargo test`
- `cargo clippy --all-features --tests -- -D warnings`

### Completion Notes

- 在 `Laputa/src/vector_storage.rs` 中新增 `VectorStorage::mark_emotion_anchor(...)`，先读取目标记录，再基于现有记录计算 `heat_i32 + 2000` 并裁剪到 `10000`
- 复用 `LaputaMemoryRecord::update_emotion(...)` 完成 `valence` 与 `arousal` 的边界裁剪，避免在存储层复制情绪裁剪语义
- 通过单条 `UPDATE memories` 持久化 `heat_i32`、`emotion_valence`、`emotion_arousal`，并在写回后重新读取记录返回，保证测试与调用方都拿到持久化结果
- 缺失 `memory_id` 时返回带 `id` 的明确错误信息，不产生任何写入副作用
- 新增 `Laputa/tests/test_emotion_anchor.rs`，覆盖成功路径、热度上限裁剪、正负边界裁剪、重开存储后的持久化验证以及缺失记录错误路径

## File List

- `Laputa/src/vector_storage.rs`
- `Laputa/tests/test_emotion_anchor.rs`
- `_bmad-output/implementation-artifacts/2-3-emotion-anchor.md`
- `_bmad-output/implementation-artifacts/sprint-status.yaml`

## Review Findings

### 2026-04-16 Code Review

**结论: ✅ PASS** — 所有AC通过，无阻塞问题

#### patch (已存档)

- [x] [Review][Patch] 边界值测试不完整：补充valence/arousal/heat精确边界测试 [test_emotion_anchor.rs] — 存档deferred-work

#### dismissed

- 4项低严重度发现（TOCTOU竞态、rows_affected检查、reason未暴露、min而非clamp）

## Change Log

- 2026-04-16: Epic 2 代码审查完成，Story 2-3 PASS
- 2026-04-15: 实现 Story 2.3 emotion anchor 持久化更新接口与专项集成测试，并完成 `cargo test` 与 `cargo clippy --all-features --tests -- -D warnings` 验证
