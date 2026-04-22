# Story 1.2.1: 代码审查修复补丁

**Story ID:** 1.2.1  
**Story Key:** 1-2-1-code-review-fixes  
**Status:** ready-for-review  
**Created:** 2026-04-14  
**Project:** 天空之城 (Laputa)  
**Parent Story:** 1-2-identity-initialization

---

## 用户故事

As a **开发者**,
I want **修复 Story 1.2 代码审查发现的 7 个问题**,
so that **身份初始化模块达到生产质量，具备完整的数据完整性保障和错误处理**。

---

## 验收标准

1. **Given** SQLite schema 需要 CHECK 约束  
   **When** `create_schema` 执行  
   **Then** `emotion_valence` 限制为 -100~+100，`emotion_arousal` 限制为 0~100。

2. **Given** 初始化检查需要原子性  
   **When** `is_initialized()` 执行  
   **Then** 同时检查 `identity.md` 和 `laputa.db`，两者都存在才认为已初始化。

3. **Given** schema 创建需要事务保障  
   **When** `create_schema` 执行  
   **Then** 使用 `BEGIN; ... COMMIT;` 包裹，失败时自动 ROLLBACK。

4. **Given** `user_name` 输入需要验证  
   **When** `initialize(user_name)` 执行  
   **Then** 空字符串或包含换行符的输入返回 `ValidationError`。

5. **Given** 错误类型命名需要精确  
   **When** 路径无效时  
   **Then** 返回 `InvalidPath` 而非模糊的 `ConfigError`。

6. **Given** 测试需要验证 schema 约束  
   **When** schema 测试执行  
   **Then** 检查 DEFAULT 值和 NOT NULL 约束是否正确。

7. **Given** `emotion_arousal` 类型一致性  
   **When** schema 定义  
   **Then** INTEGER 类型添加 CHECK 约束防止负数存储。

---

## Tasks / Subtasks

- [x] Task 1: 添加 schema CHECK 约束（AC: 1, 7）
  - [x] `emotion_valence >= -100 AND emotion_valence <= 100`
  - [x] `emotion_arousal >= 0 AND emotion_arousal <= 100`
  - [x] 文件: `Laputa/src/storage/sqlite.rs`

- [x] Task 2: 增强初始化检查原子性（AC: 2）
  - [x] `is_initialized()` 同时检查 db 和 identity 文件
  - [x] 文件: `Laputa/src/identity/initializer.rs`

- [x] Task 3: 使用事务保障 schema 创建（AC: 3）
  - [x] 用 `BEGIN; ... COMMIT;` 包裹 DDL
  - [x] 文件: `Laputa/src/storage/sqlite.rs`

- [x] Task 4: 添加 user_name 输入验证（AC: 4）
  - [x] 验证非空且不含换行符
  - [x] 返回 `ValidationError`
  - [x] 文件: `Laputa/src/identity/initializer.rs`

- [x] Task 5: 重命名错误变体（AC: 5）
  - [x] `ConfigError` → `InvalidPath`
  - [x] 文件: `Laputa/src/api/error.rs`, `initializer.rs`

- [x] Task 6: 扩展 schema 测试（AC: 6）
  - [x] 检查 DEFAULT 值正确性
  - [x] 检查 NOT NULL 约束
  - [x] 文件: `Laputa/tests/test_identity.rs`

---

## 开发说明

### 优先级排序

| 任务 | 优先级 | 风险等级 | 说明 |
|------|--------|----------|------|
| Task 3 | P0 | 高 | 事务缺失可能导致数据不一致 |
| Task 1 | P1 | 中 | CHECK 约束保障数据完整性 |
| Task 2 | P1 | 中 | 部分初始化状态检测 |
| Task 4 | P2 | 低 | 输入验证防止格式破坏 |
| Task 6 | P2 | 低 | 测试完整性 |
| Task 5 | P3 | 低 | 命名优化 |

### 实现约束

- 所有修改必须保持现有测试通过
- CHECK 约束添加不影响 Story 1.3 的 MemoryRecord 扩展
- 事务使用 SQLite 原生支持，无需额外依赖
- 错误类型变更需要同步更新调用方

### 与 Story 1.3 的衔接

Story 1.3 扩展 `LaputaMemoryRecord`，本 Story 的 CHECK 约束与 1.3 的字段定义必须一致：
- `emotion_valence`: i32, 范围 -100~+100
- `emotion_arousal`: u32, 范围 0~100

---

## References

- `_bmad-output/implementation-artifacts/1-2-identity-initialization.md` - 父 Story 及审查发现
- `Laputa/src/storage/sqlite.rs` - schema 定义
- `Laputa/src/identity/initializer.rs` - 初始化逻辑
- `Laputa/src/api/error.rs` - 错误定义
- `Laputa/tests/test_identity.rs` - 现有测试

---

## Dev Agent Record

### 实现摘要

已完成 Story 1.2 代码审查发现的 7 个问题修复：

| AC | 修复内容 | 文件 |
|----|---------|------|
| AC 1, 7 | emotion_valence/arousal CHECK 约束 | sqlite.rs |
| AC 2 | is_initialized() 原子性检查 | initializer.rs |
| AC 3 | schema 创建事务包裹 | sqlite.rs |
| AC 4 | user_name 输入验证 | initializer.rs |
| AC 5 | ConfigError → InvalidPath | error.rs, initializer.rs |
| AC 6 | schema DEFAULT/NOT NULL 测试 | test_identity.rs |

### 测试结果

- 全部 153 个测试通过
- 新增 4 个测试：
  - `test_schema_default_values_and_constraints`
  - `test_schema_check_constraints`
  - `test_user_name_validation_empty`
  - `test_user_name_validation_newline`

### 附加修复

修复了 `memory.rs:76` 预先存在的 `format!` 宏语法错误（缺少 `!`）。

---

## File List

| 文件 | 变更类型 |
|------|----------|
| `Laputa/src/storage/sqlite.rs` | 修改 - 事务 + CHECK 约束 |
| `Laputa/src/identity/initializer.rs` | 修改 - 原子性 + 输入验证 |
| `Laputa/src/api/error.rs` | 修改 - ConfigError → InvalidPath |
| `Laputa/src/storage/memory.rs` | 修复 - format! 宏语法 |
| `Laputa/tests/test_identity.rs` | 新增 - 4 个扩展测试 |

---

_故事状态: ready-for-review | 完成时间: 2026-04-14 | 全部测试通过: 153/153_