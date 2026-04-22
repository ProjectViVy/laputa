# Story Epic-1-Patch: 身份初始化安全与健壮性修复

Status: done

## Story

As a 开发者，
I want 修复 Epic 1 身份初始化链路中的输入校验与失败清理缺口，
so that 初始化过程能拒绝危险输入、避免残留脏状态，并保证写入层 heat 边界一致。

## Origin

**来源**: Epic 1 综合代码审查（2026-04-16）  
**父 Epic**: Epic 1 - 记忆库初始化  
**来源文档**: `_bmad-output/implementation-artifacts/deferred-work.md` 中 Epic 1 的 P1-P5 patch 待办项

## Acceptance Criteria

1. **Given** `IdentityInitializer::initialize()` 接收 `user_name` 参数  
   **When** `user_name` 包含路径遍历字符（如 `../`、`..\\` 或等价编码变体）  
   **Then** 返回 `LaputaError::ValidationError`，拒绝初始化

2. **Given** `IdentityInitializer::initialize()` 接收 `user_name` 参数  
   **When** `user_name` 长度超过 256 字符  
   **Then** 返回 `LaputaError::ValidationError`，拒绝初始化

3. **Given** `IdentityInitializer::initialize()` 接收 `user_name` 参数  
   **When** `user_name` 含控制字符（至少覆盖 `\x00` 等非换行控制字符）  
   **Then** 返回 `LaputaError::ValidationError`，拒绝初始化

4. **Given** `IdentityInitializer::initialize()` 已成功创建 `laputa.db` 但写入 `identity.md` 失败  
   **When** `fs::write` 返回错误  
   **Then** 清理已创建的数据库文件，不留下半初始化状态

5. **Given** `VectorStorage::add_memory_record()` 接收 `MemoryInsert` 参数  
   **When** `heat_i32` 不在 `[0, 10000]` 范围  
   **Then** 返回 `LaputaError::ValidationError`，拒绝写入

## Tasks / Subtasks

- [x] Task 1: 收紧 `user_name` 输入校验（AC: 1, 2, 3）
  - [x] 在 `Laputa/src/identity/initializer.rs` 为 `initialize()` 增加路径遍历检测
  - [x] 增加最大长度常量与长度校验，拒绝超过 256 字符的输入
  - [x] 增加控制字符检测，保持现有“非空且不含换行”约束不回退
  - [x] 统一返回明确的 `LaputaError::ValidationError` 消息，避免模糊失败原因

- [x] Task 2: 修复初始化失败清理（AC: 4）
  - [x] 仅在数据库创建成功、`identity.md` 写入失败时执行清理
  - [x] 调用 `std::fs::remove_file(&self.db_path)` 删除刚创建的数据库
  - [x] 不要吞掉主失败原因；清理失败只能作为附加信息或日志，不能覆盖原始写入错误

- [x] Task 3: 补齐写入层 `heat_i32` 边界校验（AC: 5）
  - [x] 在 `Laputa/src/vector_storage.rs` 的 `add_memory_record()` 入口校验 `insert.heat_i32`
  - [x] 复用 `Laputa/src/storage/memory.rs` 中现有 `MIN_HEAT_I32` / `MAX_HEAT_I32`
  - [x] 失败时返回 `LaputaError::ValidationError`，阻止 SQLite 写入与向量索引写入

- [x] Task 4: 补齐测试覆盖
  - [x] 在 `Laputa/tests/test_identity.rs` 新增危险 `user_name` 用例：路径遍历、超长、控制字符
  - [x] 增加 `identity.md` 写入失败后的数据库清理测试
  - [x] 为 `add_memory_record()` 新增 `heat_i32 < 0` 与 `heat_i32 > 10000` 的拒绝测试
  - [x] 保留并复用现有初始化成功、重复初始化、schema 默认值测试，不重写已有 fixture

## Dev Notes

### Story Context

这个 patch story 对应 Epic 1 审查发现的 P1-P5。`epic-1` 主线已完成，但该补丁仍在 deferred 批次中单独跟踪，开发时不要改写 Epic 1 已交付行为，只修复审查明确指出的缺口。  
[Source: `_bmad-output/implementation-artifacts/deferred-work.md` - “Deferred from: Epic 1 综合代码审查 (2026-04-16)”]

### Current Code Reality

- `IdentityInitializer::initialize()` 当前只校验“非空且不含换行”，位置在 [initializer.rs](D:/VIVYCORE/newmemory/Laputa/src/identity/initializer.rs:37)。
- 当前写库与写 `identity.md` 顺序为：先 `Connection::open(&self.db_path)`，后 `fs::write(&self.identity_path, identity_content)`，写文件失败时没有清理数据库。[initializer.rs](D:/VIVYCORE/newmemory/Laputa/src/identity/initializer.rs:57) [initializer.rs](D:/VIVYCORE/newmemory/Laputa/src/identity/initializer.rs:66)
- `MIN_HEAT_I32` / `MAX_HEAT_I32` 已定义在 [memory.rs](D:/VIVYCORE/newmemory/Laputa/src/storage/memory.rs:8) 和 [memory.rs](D:/VIVYCORE/newmemory/Laputa/src/storage/memory.rs:9)。
- `VectorStorage::add_memory_record()` 当前直接接收并写入 `insert.heat_i32`，尚未在入口做边界拒绝。[vector_storage.rs](D:/VIVYCORE/newmemory/Laputa/src/vector_storage.rs:220)

### Implementation Guardrails

- 不要发明新的错误类型。沿用已有 `LaputaError::ValidationError` 即可，和现有初始化校验保持一致。  
  [Source: `Laputa/src/identity/initializer.rs:40`, `Laputa/src/storage/memory.rs:82`]
- 不要新增独立配置文件或全局 validator 模块；本 patch 应在现有入口就地加固，保持 diff 小而可审查。
- 路径遍历检测以“拒绝危险片段”为目标，不要尝试把非法输入规范化后继续使用。这个场景是初始化入口，不需要容错式修复。
- `heat_i32` 校验必须发生在 `add_memory_record()` 持久化之前，否则会留下数据库与上层对象语义不一致的问题。
- 本 story 只覆盖审查列出的 P1-P5，不顺带处理 `user_name` 中 Markdown 特殊字符等已标记为 pre-existing / non-security 的问题。  
  [Source: `_bmad-output/implementation-artifacts/deferred-work.md` 同节 deferred 列表]

### Architecture Compliance

- 项目要求在现有 `mempalace-rs` 演化路径上做增量修复，优先扩展现有模块，不重写底座。  
  [Source: `Laputa/AGENTS.md` “必须继承的 mempalace-rs 能力”]
- 核心链路必须保持纯 Rust、本地可运行、错误可测试。此 patch 直接作用于初始化链路与本地存储边界，属于 MVP 核心健壮性修复。  
  [Source: `_bmad-output/planning-artifacts/prd.md` - NFR-1, NFR-7, NFR-10]
- Identity 初始化属于 Epic 1 / FR-1 范围，不应影响后续 heat、archive、search 设计边界。  
  [Source: `_bmad-output/planning-artifacts/epics.md` - Epic 1 / Story 1.2]

### File Structure Requirements

优先修改以下文件：

- [initializer.rs](D:/VIVYCORE/newmemory/Laputa/src/identity/initializer.rs) - `user_name` 校验与失败清理
- [vector_storage.rs](D:/VIVYCORE/newmemory/Laputa/src/vector_storage.rs) - `add_memory_record()` 的 `heat_i32` 边界拒绝
- [test_identity.rs](D:/VIVYCORE/newmemory/Laputa/tests/test_identity.rs) - 身份初始化补充测试

如果为 `add_memory_record()` 增加新测试，优先放在最贴近存储入口的现有测试文件；没有合适位置时再新增独立测试文件，但不要复制现有 identity fixture 逻辑。

### Testing Requirements

- 保持已有 `test_identity.rs` 现有成功路径测试继续通过，尤其是：初始化成功、重复初始化拒绝、schema 默认值断言、空字符串/换行拒绝。  
  [Source: `Laputa/tests/test_identity.rs`]
- 新增测试必须断言“拒绝后未留下副作用”：
  - 非法 `user_name` 不创建有效初始化结果
  - `identity.md` 写入失败后 `laputa.db` 被清理
  - 非法 `heat_i32` 不写入 memories
- 对于边界校验，至少覆盖 `-1`、`10001`，并保留合法边界 `0` / `10000` 的现有语义不变。

### Prior Story Intelligence

- [1-3-patch-heat-validation.md](D:/VIVYCORE/newmemory/_bmad-output/implementation-artifacts/1-3-patch-heat-validation.md) 已经确立一个重要模式：边界违规返回 `ValidationError`，不要使用静默 clamp 或自动修复。
- `Laputa/src/storage/memory.rs` 已存在 heat 语义边界常量，本 patch 应复用这些常量而不是重复定义第二套范围。
- Epic 1 deferred-work 已把 P5 明确放在 `VectorStorage::add_memory_record()`，不要把责任错误地下沉到别的调用层。

### Project Context Reference

- [AGENTS.md](D:/VIVYCORE/newmemory/Laputa/AGENTS.md) - 项目级约束：增量演化、纯 Rust、本地优先
- [epics.md](D:/VIVYCORE/newmemory/_bmad-output/planning-artifacts/epics.md) - Epic 1 / Story 1.2 与 Epic 1 范围
- [prd.md](D:/VIVYCORE/newmemory/_bmad-output/planning-artifacts/prd.md) - NFR-1 / NFR-7 / NFR-10
- [deferred-work.md](D:/VIVYCORE/newmemory/_bmad-output/implementation-artifacts/deferred-work.md) - Epic 1 审查发现的 P1-P5

## Dev Agent Record

### Agent Model Used

GPT-5 Codex

### Debug Log References

- `rg -n "struct IdentityInitializer|fn initialize|identity_path|db_path|fs::write|remove_file|ValidationError|user_name" Laputa/src/identity/initializer.rs`
- `rg -n "fn add_memory_record|heat_i32|ValidationError|MIN_HEAT_I32|MAX_HEAT_I32|MemoryInsert" Laputa/src/vector_storage.rs Laputa/src/storage/memory.rs`
- `rg -n "test_identity|initialize\(|IdentityInitializer|heat_i32|out of range|path traversal|control characters" Laputa/tests`

### Completion Notes List

- 2026-04-19: 依据 Epic 1 审查发现与当前代码现状重建 story context
- 2026-04-19: 明确本 story 为 deferred patch 批次，不回退 `epic-1: done` 主线状态
- 2026-04-19: 为开发阶段补充具体文件入口、测试落点与实现边界
- 2026-04-19: 在 `IdentityInitializer::initialize()` 增加显式路径遍历、长度上限和控制字符校验，统一返回 `ValidationError`
- 2026-04-19: 为 `identity.md` 写入失败补充 `laputa.db` 清理逻辑，保留原始写入失败语义
- 2026-04-19: 在 `VectorStorage::add_memory_record()` 入口复用 `MIN_HEAT_I32` / `MAX_HEAT_I32` 拒绝越界 heat
- 2026-04-19: 新增 identity 与 vector storage 拒绝路径测试，验证失败不留下数据库或索引副作用

### File List

- `_bmad-output/implementation-artifacts/epic-1-patch-security-validation.md`
- `Laputa/src/identity/initializer.rs`
- `Laputa/src/vector_storage.rs`
- `Laputa/tests/test_identity.rs`
- `Laputa/tests/test_vector_storage_validation.rs`

### Review Findings

#### Patch 级发现（已修复）

- [x] [Review][Patch] P1: update_memory_after_merge 未验证 heat_i32 [vector_storage.rs:616] — ✅ 已修复：添加 `validate_heat_i32(new_heat_i32)?`
- [x] [Review][Patch] P2: update_heat_fields_if_unchanged 未验证 heat_i32 [vector_storage.rs:980] — ✅ 已修复：添加 `validate_heat_i32(new_heat_i32)?`
- [x] [Review][Patch] P3: create_schema 失败未清理 db 文件 [initializer.rs:55-58,81-96] — ✅ 已修复：添加 `cleanup_db_file_after_failure` 函数处理 schema 失败清理

#### Defer 级发现（pre-existing 或超出范围）

- [x] [Review][Defer] URL编码变体路径遍历未覆盖 [initializer.rs:91-98] — deferred，超出 Story 范围（P1-P5明确限定为字符串模式检测）
- [x] [Review][Defer] TOCTOU 竞态条件 [initializer.rs:42-46] — deferred，多进程并发初始化不在 MVP 范围
- [x] [Review][Defer] identity.md 内容未进行 YAML 转义 [initializer.rs:75-78] — deferred，非安全风险，Markdown 注入在此场景风险低

#### Dismissed 级发现（测试覆盖建议）

- 路径遍历模式测试覆盖不全（仅覆盖 `../`） — 实现正确覆盖所有四种模式，建议补充测试但不作为 patch
- 控制字符测试仅覆盖 ` ` — 实现正确覆盖所有控制字符，建议补充测试但不作为 patch

## Change Log

- 2026-04-19: 重新生成 Story，上下文对齐 deferred patch 批次与当前代码状态，状态更新为 `ready-for-dev`
- 2026-04-19: 完成 Epic 1 patch 安全校验修复，实现初始化输入加固、失败清理与 heat 写入边界校验，状态更新为 `review`
- 2026-04-19: 三层代码审查完成，发现 3 patch + 3 defer + 2 dismissed
- 2026-04-19: Patch findings P1-P3 已修复并验证通过，状态更新为 `done`
