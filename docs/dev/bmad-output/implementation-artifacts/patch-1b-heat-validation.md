# Story: MemoryInsert heat_i32 边界验证

**Story ID:** patch-1b  
**Story Key:** patch-1b-heat-validation  
**Status:** done  
**Created:** 2026-04-16  
**Updated:** 2026-04-19  
**Project:** 天空之城 (Laputa)

**Origin:** `deferred-work.md` Epic 1 P5

---

## 用户故事

As a **开发�?*�? 
I want **�?`VectorStorage::add_memory_record()` 入口验证 `MemoryInsert.heat_i32` 的取值范�?*�? 
So that **外部调用方无法把越界热度写入 SQLite，后续热度状态机、检索排序和导出逻辑都能基于有效数据运行**�?
---

## 验收标准

1. **Given** `VectorStorage::add_memory_record()` 接收 `MemoryInsert` 参数  
   **When** `heat_i32` 不在 `[0, 10000]` 范围  
   **Then** 返回 `LaputaError::ValidationError`，拒绝写入数据库与向量索�?
2. **Given** `VectorStorage::add_memory_record()` 接收边界�?`heat_i32=0` �?`heat_i32=10000`  
   **When** 调用写入  
   **Then** 写入成功，读取结果保持原值，不发生静�?clamp

3. **Given** `VectorStorage::add_memory_record()` 接收越界�?`heat_i32=-1` �?`heat_i32=10001`  
   **When** 调用写入  
   **Then** 返回错误前不写入任何 `memories` 行，也不�?usearch 索引添加节点

---

## 上下文与业务价�?
- �?patch 来自 Epic 1 综合代码审查�?P5 发现，属于输入校验缺口，不是热度算法改造�? 
- `heat_i32` 是全系统共享字段，架构已固定�?`0..10000` 对应 `0.00..100.00`。如果入口允许脏值写入，会破坏：
  - HeatService 的阈值判断与状态机
  - 检索排序与归档候选判�?  - 导出/展示中对 `heat_i32` 的语义假�?- 该修复应在最靠近写入的位置阻断非法输入，而不是依赖下游模块兜底�?
---

## 缺陷清单

| ID | 问题 | 文件位置 | 影响 |
|----|------|----------|------|
| **P5** | `MemoryInsert` 缺少 `heat_i32` 边界验证 | `Laputa/src/vector_storage.rs:220` | 外部调用可写入越界热度，污染数据库与向量索引 |

---

## 实施任务

- [x] Task 1: 在写入入口添�?heat 范围校验 (AC: 1, 2, 3)
  - [x] �?`Laputa/src/vector_storage.rs` �?`add_memory_record()` 开头、`embed_single()` 之前验证 `insert.heat_i32`
  - [x] 复用 `Laputa/src/storage/memory.rs` 中现有常�?`MIN_HEAT_I32` / `MAX_HEAT_I32`，不要在 `vector_storage.rs` 重新定义同名常量
  - [x] 若越界，返回 `anyhow::Error` 包装�?`LaputaError::ValidationError`

- [x] Task 2: 确保失败路径无副作用 (AC: 1, 3)
  - [x] 校验必须发生在任何数据库写入之前
  - [x] 校验必须发生在任�?usearch `reserve()` / `add()` 之前
  - [x] 保持现有成功路径不变，不改动 `row_id` 获取、索引扩容、向量写入顺�?
- [x] Task 3: 补充自动化测�?(AC: 2, 3)
  - [x] 在合适的测试文件中新�?`add_memory_record()` 边界测试，覆�?`-1 / 0 / 10000 / 10001`
  - [x] 断言越界时返�?`ValidationError`
  - [x] 断言越界时数据库中未新增记录
  - [x] 如测试使用真�?`VectorStorage`，同时断言索引 `size()` 未增�?
---

## 开发约束与防错护栏

### 必须复用的现有实�?
- `Laputa/src/storage/memory.rs` 已定义：
  - `MIN_HEAT_I32: i32 = 0`
  - `MAX_HEAT_I32: i32 = 10_000`
- `LaputaError::ValidationError(String)` 已存在于 `Laputa/src/api/error.rs`
- `LaputaMemoryRecord::set_heat()` 已采用“越界直接报错，不静默修正”的策略；本 patch 要与该行为保持一致，不要引入新的 clamp 语义

### 明确禁止

- 不要�?`vector_storage.rs` 新建重复�?heat 常量
- 不要把越界值自�?`clamp()` 到合法范�?- 不要只在读取侧修复；必须在写入入口拒绝非法输�?- 不要为了这个 patch 改动 `HeatService`、`state.rs`、`decay.rs` �?schema
- 不要把错误改�?`panic!` / `assert!`；这是运行时输入校验，不是调试断言

### 推荐实现方式

- �?`vector_storage.rs` 增加�?`MIN_HEAT_I32` 的导入，与现�?`MAX_HEAT_I32` 一起使�?- 使用区间判断�?
```rust
if !(MIN_HEAT_I32..=MAX_HEAT_I32).contains(&insert.heat_i32) {
    return Err(LaputaError::ValidationError(
        format!("heat_i32 out of range [{MIN_HEAT_I32}, {MAX_HEAT_I32}]: {}", insert.heat_i32),
    ).into());
}
```

- 错误消息允许与上例略有不同，但必须明确包含越界事实与合法区间

---

## 相关代码情报

### 主要修改�?
- `Laputa/src/vector_storage.rs`
  - `MemoryInsert` 已暴�?`heat_i32: i32`
  - `add_memory()` 默认传入 `DEFAULT_HEAT_I32`，本 patch 不应改变默认值逻辑
  - `add_memory_record()` 当前流程�?`embed_single()` �?`INSERT` �?`last_insert_rowid()` �?`index.reserve()` �?`index.add()`
  - �?patch 唯一需要插入的新行为是“在上述流程最前面做合法性校验�?
### 相邻既有模式

- `Laputa/src/storage/memory.rs`
  - `set_heat()` 已明确拒�?NaN 和区间外 `f64`
  - `with_updated_heat()` 仅保�?`debug_assert!`，说明调用入口仍需显式校验
- `Laputa/src/heat/state.rs`
  - 已复�?`MIN_HEAT_I32` / `MAX_HEAT_I32` 做范围判断，说明这些常量是系统级单一真相来源

### 测试落点建议

- 优先考虑�?`Laputa/tests/` 中新增或扩展针对 `VectorStorage` 写入入口的测试文件，而不是塞�?`test_memory_record.rs`
- `test_memory_record.rs` 当前覆盖的是 `LaputaMemoryRecord` 值对象行为，不是 SQLite/usearch 写入路径
- 如仓库已�?`VectorStorage` 写入相关测试，可在同文件延续�?fixture 和断言风格

---

## 测试要求

- `heat_i32 = -1` 返回 `ValidationError`
- `heat_i32 = 0` 写入成功，读取结果等�?`0`
- `heat_i32 = 10000` 写入成功，读取结果等�?`10000`
- `heat_i32 = 10001` 返回 `ValidationError`
- 任一失败用例都要验证数据库未新增记录
- 若测试可�?usearch 索引尺寸，任一失败用例都要验证索引未增�?
---

## 架构一致性要�?
- 保持 `heat_i32` 的统一语义：`0..10000`�?00 倍精度整数存�?- 保持错误处理模式：领域校验失败走 `ValidationError`
- 保持项目结构：只修改写入入口与对应测试，不新增无关模�?- 保持 MVP 约束：这�?patch 修复，不扩展功能范围

---

## 参考资�?
- `D:\VIVYCORE\newmemory\_bmad-output\implementation-artifacts\deferred-work.md`
- `D:\VIVYCORE\newmemory\_bmad-output\implementation-artifacts\epic-1-patch-security-validation.md`
- `D:\VIVYCORE\newmemory\_bmad-output\planning-artifacts\architecture.md`
- `D:\VIVYCORE\newmemory\Laputa\src\vector_storage.rs`
- `D:\VIVYCORE\newmemory\Laputa\src\storage\memory.rs`
- `D:\VIVYCORE\newmemory\Laputa\src\api\error.rs`
- `D:\VIVYCORE\newmemory\Laputa\src\heat\state.rs`

---

## 完成状�?
- Story 状�? `review`
- Sprint 状态更新要�? `patch-1b-heat-validation` �?`backlog` 更新�?`review`
- 说明: 已补全为开发可执行故事，包含实现边界、复用约束、测试要求与防回归护�?
---


## Dev Agent Record

### Implementation Plan
- Reused the existing `VectorStorage::add_memory_record()` entrypoint validation and kept `validate_heat_i32(insert.heat_i32)?` before `embed_single()`, SQLite writes, and usearch reserve/add.
- Reused `MIN_HEAT_I32` / `MAX_HEAT_I32` from `Laputa/src/storage/memory.rs`; no duplicate constants or clamp behavior were introduced.
- Added deterministic unit coverage for successful boundary writes (`0`, `10000`) and retained integration coverage for out-of-range rejection (`-1`, `10001`) with no DB/index side effects.

### Debug Log
- `cargo test --lib test_add_memory_record_accepts_heat_boundaries_without_clamping` passed.
- `cargo test --test test_vector_storage_validation` passed.
- `cargo test --test test_memory_record --test test_heat --test test_vector_storage_validation` passed.
- `cargo test` did not complete in this environment: Windows reported OS error 1455 (page file too small) while compiling `test_semantic_search`, followed by an `ort_sys` rlib-format build error.

### Completion Notes
- `heat_i32` values outside `[0, 10000]` are rejected as `LaputaError::ValidationError` before embedding, SQLite insertion, or usearch mutation.
- Boundary values `0` and `10000` write successfully and round-trip unchanged through `get_memory_by_id()` in unit coverage.
- Existing successful write ordering remains unchanged after validation.

### File List
- `Laputa/src/vector_storage.rs`
- `Laputa/tests/test_vector_storage_validation.rs`
- `_bmad-output/implementation-artifacts/patch-1b-heat-validation.md`
- `_bmad-output/implementation-artifacts/sprint-status.yaml`
---

## Change Log

- 2026-04-16: 创建初始 patch story，占位记录 Epic 1 P5
- 2026-04-17: 由 create-story 工作流补全为 ready-for-dev 开发故事
- 2026-04-19: 代码审查通过，状态更新为 done

---

### Review Findings (2026-04-19)

#### Deferred 级发现

- [x] [Review][Defer] 缺失极端负值/大正值测试 — 测试覆盖建议，当前覆盖 -1/0/10000/10001，未覆盖极端值
- [x] [Review][Defer] heat_to_i32 潜在溢出风险 [memory.rs:122-124] — pre-existing，本 patch 未涉及

#### 验收标准验证结果

| AC | 状态 |
|----|------|
| AC1: 越界返回 ValidationError | PASSED |
| AC2: 边界值写入成功无静默 clamp | PASSED |
| AC3: 越界不写入 DB/索引 | PASSED |
| 约束：复用现有常量 | PASSED |
| 约束：不改动 HeatService | PASSED |
| 约束：无 panic/assert | PASSED |

**审查通过，状态已更新为 done。**

- 2026-04-16: 创建初始 patch story，占位记�?Epic 1 P5
- 2026-04-17: �?create-story 工作流补全为 ready-for-dev 开发故�?