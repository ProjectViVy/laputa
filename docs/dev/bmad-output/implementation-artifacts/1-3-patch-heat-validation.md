# Story 1.3-Patch: 热度边界验证修复

Status: done

## Story

As a 开发者，
I want 修复 `set_heat()` 方法的边界验证，使其符合 AC#3 要求返回 ValidationError，
So that 热度字段的越界输入能被正确拒绝而非静默修正，保证数据完整性。

## Origin

**来源**: Story 1.3 代码审查发现 (2026-04-14)

- **父 Story**: 1-3-memoryrecord-extension
- **发现编号**: F1
- **审查记录**: `[Review][Patch] set_heat() 不返回 ValidationError`

## Acceptance Criteria

1. **Given** `LaputaMemoryRecord` 的 `set_heat()` 方法
   **When** 输入热度值 `v < 0.0` 或 `v > 100.0`
   **Then** 返回 `Err(LaputaError::ValidationError)`，不修改 `heat_i32` 字段

2. **Given** `heat_to_i32()` 转换函数
   **When** 被公共 API 调用
   **Then** 不使用 `clamp()` 静默修正，而是依赖调用方进行边界检查

3. **Given** 输入热度值在合法范围 `[0.0, 100.0]`
   **When** 调用 `set_heat()`
   **Then** 成功存储并返回 `Ok(())`

4. **Given** 新增的 `LaputaError` 枚举
   **When** 定义错误类型
   **Then** 包含 `ValidationError` 变体，可携带描述信息

## Tasks / Subtasks

- [x] Task 1: 定义 LaputaError 枚举
  - [x] 1.1 在 `src/storage/memory.rs` 导入 `crate::api::LaputaError`（已存在于 `src/api/error.rs`）
  - [x] 1.2 确认 `ValidationError(String)` 变体已存在
  - [x] 1.3 确认 `std::error::Error` 和 `Display` trait 已实现

- [x] Task 2: 修改 set_heat() 方法
  - [x] 2.1 添加边界检查：`if v < 0.0 || v > 100.0`
  - [x] 2.2 返回类型改为 `Result<(), LaputaError>`
  - [x] 2.3 合法值时调用 `heat_to_i32()` 并返回 `Ok(())`

- [x] Task 3: 更新 heat_to_i32() 函数
  - [x] 3.1 移除 `clamp()` 静默修正
  - [x] 3.2 添加文档注释说明调用方需做边界检查

- [x] Task 4: 更新测试
  - [x] 4.1 添加越界测试：验证 `set_heat(-1.0)` 返回错误
  - [x] 4.2 添加越界测试：验证 `set_heat(101.0)` 返回错误
  - [x] 4.3 验证合法边界值：`0.0` 和 `100.0` 成功存储

- [x] Task 5: 更新调用方（如有）
  - [x] 5.1 `with_updated_heat()` 保持原签名，直接使用 `heat_to_i32()` 绕过验证（构建器风格）
  - [x] 5.2 测试文件 `test_memory_record.rs` 调用 `set_heat()` 改用 `.unwrap()`

## Dev Notes

### 当前实现问题

```rust
// memory.rs:70-72 (当前)
pub fn set_heat(&mut self, heat: f64) {
    self.heat_i32 = heat_to_i32(heat);  // heat_to_i32 使用 clamp() 静默修正
}

// memory.rs:94-96 (当前)
pub fn heat_to_i32(value: f64) -> i32 {
    (value.clamp(0.0, 100.0) * 100.0).round() as i32  // 静默修正！
}
```

### 目标实现

```rust
// 定义错误类型
#[derive(Debug, Clone)]
pub enum LaputaError {
    ValidationError(String),
}

impl std::fmt::Display for LaputaError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            LaputaError::ValidationError(msg) => write!(f, "Validation error: {}", msg),
        }
    }
}

impl std::error::Error for LaputaError {}

// 修复后的 set_heat
pub fn set_heat(&mut self, v: f64) -> Result<(), LaputaError> {
    if v < 0.0 || v > 100.0 {
        return Err(LaputaError::ValidationError("heat out of range [0.0, 100.0]".into()));
    }
    self.heat_i32 = heat_to_i32(v);
    Ok(())
}
```

### with_updated_heat() 处理策略

`with_updated_heat()` 是不可变更新方法，返回新实例。有两种选择：
1. 保持原签名，内部调用 `set_heat()` 时使用 `unwrap()`（假设调用方已验证）
2. 改为返回 `Result<Self, LaputaError>`

建议：保持原签名，因为它是构建器风格，调用方应自行验证输入。

### References

- [1-3-memoryrecord-extension.md#AC#3] 原始验收标准
- [memory.rs:70-72] 当前实现位置
- [memory.rs:94-96] heat_to_i32 实现
- [architecture.md#L852-L861] 构建器模式规范

## File List

- `src/storage/memory.rs` - 主要修改文件
- `src/error.rs` - 新增错误类型文件（可选）
- `tests/test_memory_record.rs` - 测试更新

## Dev Agent Record

### Agent Model Used

GPT-5 Codex

### Debug Log References

- `cargo test --test test_memory_record` — 8 tests passed
- `cargo test` — 137 unit tests passed (1 pre-existing failure in test_identity.rs unrelated to this patch)

### Completion Notes List

- Task 1: `LaputaError` 已存在于 `src/api/error.rs`，包含 `ValidationError` 变体，直接导入使用
- Task 2: `set_heat()` 改为返回 `Result<(), LaputaError>`，添加边界检查 `[0.0, 100.0]`
- Task 3: `heat_to_i32()` 移除 `clamp()` 静默修正，添加文档注释
- Task 4: 新增 3 个测试：越界负数、越界超限、边界值合法
- Task 5: `with_updated_heat()` 保持原签名，内部直接调用 `heat_to_i32()`

### File List

- `src/storage/memory.rs` — 主要修改：导入 LaputaError、修改 set_heat()、移除 clamp
- `tests/test_memory_record.rs` — 新增 3 个越界测试，适配 Result 类型

## Change Log

- 2026-04-14: Story 创建，追踪代码审查发现的 patch 项
- 2026-04-14: 完成实现，所有 5 个 tasks 完成并通过测试，状态更新为 review

### Review Findings

- [x] [Review][Decision] `with_updated_heat()` 接受越界值无防护，静默写入非法 heat_i32 — 决策：加 debug_assert! 调试构建捕获，保留原签名，已修复 [memory.rs:91-103]
- [x] [Review][Patch] set_heat(f64::NAN) 穿透边界检查，静默存储 heat_i32=0；已加 `heat.is_nan()` 检查，新增 NaN 测试 [memory.rs:74-79]
- [x] [Review][Defer] heat_to_i32() 是 pub 函数，外部可绕过验证 [memory.rs:108-110] — deferred, pre-existing，AC#2 明确设计意图
- [x] [Review][Defer] 越界测试未验证错误消息具体内容 [test_memory_record.rs:55-58,66-68] — deferred, pre-existing，AC 未要求消息内容
- [x] [Review][Defer] set_heat(99.999) 舍入后 get_heat() 返回 100.0，精度不对称 [memory.rs:108-110] — deferred, pre-existing，i32 缩放设计固有特性
- [x] [Review][Defer] 测试未固化「错误后 heat_i32 不变」的实现顺序约束 [test_memory_record.rs:57-58] — deferred, pre-existing，现有断言已部分覆盖
- [x] [Review][Defer] update_emotion 使用 clamp 静默修正与 set_heat 拒绝策略不一致 [memory.rs:84-85] — deferred, pre-existing，Story 1-3 原始设计决策