# Story 1.2: 主体身份初始化

**Story ID:** 1.2  
**Story Key:** 1-2-identity-initialization  
**Status:** done  
**Created:** 2026-04-13  
**Updated:** 2026-04-14  
**Project:** 天空之城 (Laputa)

---

## 用户故事

As a **用户**,
I want **创建一个新的天空之城记忆库实例，并写入最小可运行的主体身份**,
so that **我可以开始记录个人记忆，并为后续写入、检索、唤醒与热度机制提供初始化基线**。

---

## 验收标准

1. **Given** Story 1.1 已建立 `Laputa/` 项目骨架、`Cargo.toml`、`src/api/`、`src/identity/`、`src/storage/` 和测试目录  
   **When** 用户在配置目录上执行初始化逻辑（`IdentityInitializer::initialize(user_name)` 或等价 CLI `laputa init --name <name>`）  
   **Then** 在目标目录创建 `laputa.db` SQLite 数据库文件。

2. **Given** 初始化成功创建数据库  
   **When** `create_schema(&Connection)` 被调用  
   **Then** `memories` 表包含以下列及默认值：
   - `id` (TEXT PRIMARY KEY)
   - `text_content` (TEXT NOT NULL)
   - `wing`, `room` (TEXT NOT NULL DEFAULT '')
   - `source_file` (TEXT, nullable)
   - `valid_from` (INTEGER NOT NULL)
   - `valid_to` (INTEGER, nullable)
   - `heat_i32` (INTEGER NOT NULL DEFAULT 5000) — 对应 50.00 正常区间
   - `last_accessed` (INTEGER NOT NULL)
   - `access_count` (INTEGER NOT NULL DEFAULT 0)
   - `is_archive_candidate` (INTEGER NOT NULL DEFAULT 0) — SQLite 用 0/1 表示 bool
   - `emotion_valence` (INTEGER NOT NULL DEFAULT 0) — 范围 -100~+100
   - `emotion_arousal` (INTEGER NOT NULL DEFAULT 0) — 范围 0~100
   **And** 创建索引：`idx_memories_valid_from`、`idx_memories_heat`、`idx_memories_wing`。

3. **Given** 初始化逻辑需要写入 L0 身份文件  
   **When** `IdentityInitializer::initialize("大湿")` 成功执行  
   **Then** 与数据库同目录生成 `identity.md`  
   **And** 文件内容遵守以下协议：
   - 第一行是 `## L0 — IDENTITY`（注意使用 Unicode EM DASH `—` U+2014）
   - 包含 `user_name: 大湿`
   - 包含 `user_type: 个人记忆助手`
   - 包含 UTC RFC3339 格式的 `created_at`（示例：`2026-04-13T12:34:56Z`）

4. **Given** 同一目录已经存在 `identity.md`  
   **When** 再次执行初始化  
   **Then** 不覆盖现有身份文件  
   **And** 返回 `LaputaError::AlreadyInitialized(String)`，参数为已存在的 identity.md 文件路径。

5. **Given** 初始化是后续 Story 2.x/3.x/5.x 的前置能力  
   **When** 初始化函数返回成功  
   **Then** 返回值中包含实际数据库路径字符串，供调用方和后续组件复用。

6. **Given** `Laputa/tests/test_identity.rs` 存在回归测试  
   **When** 运行身份初始化测试  
   **Then** 至少覆盖以下场景：
   - 成功创建数据库与身份文件
   - 重复初始化返回 `AlreadyInitialized`
   - 返回路径存在且以 `laputa.db` 结尾
   - schema 包含全部必需列且初始化后 `memories` 表为空

---

## 当前实现对应关系

本 Story 已在代码库中实现，当前文档为后续 `dev-story` / `code-review` / 回归修订提供精确上下文。

**实际实现位置：**

| 文件 | 职责 |
|------|------|
| `Laputa/src/identity/initializer.rs` | IdentityInitializer 主体逻辑 |
| `Laputa/src/identity/mod.rs` | 模块导出 |
| `Laputa/src/storage/sqlite.rs` | create_schema + MemoryRecord 结构 |
| `Laputa/src/api/error.rs` | LaputaError 定义 |
| `Laputa/tests/test_identity.rs` | 4 个回归测试 |

**当前状态判断：**
- sprint-status 已将 `1-2-identity-initialization` 标记为 `review`
- Story 1.1 已完成，本 Story 复用其建立的 `LaputaError` 模块
- Story 1.3 已落地扩展 `LaputaMemoryRecord`，本 Story 只约束初始化最小 schema

---

## Tasks / Subtasks

- [x] Task 1: 在 `src/identity/initializer.rs` 实现 `IdentityInitializer`（AC: 1, 3, 4, 5）
  - [x] 基于配置目录推导 `laputa.db` 与 `identity.md` 路径
  - [x] 通过 `is_initialized()` 检查重复初始化
  - [x] 初始化时先创建父目录，再打开 SQLite 连接
  - [x] 成功后返回数据库路径字符串

- [x] Task 2: 在 `src/storage/sqlite.rs` 提供最小 schema 初始化（AC: 2）
  - [x] 创建 `MemoryRecord` 结构体用于 Story 1.2 的初始化语义
  - [x] 创建 `memories` 表及三个索引
  - [x] 将 rusqlite 错误统一映射为 `LaputaError`

- [x] Task 3: 写入 L0 身份文件协议（AC: 3）
  - [x] 生成 `## L0 - IDENTITY` 标题
  - [x] 固定写入 `user_type: 个人记忆助手`
  - [x] 使用 `Utc::now().to_rfc3339_opts(SecondsFormat::Secs, true)` 写入 `created_at`

- [x] Task 4: 统一错误语义（AC: 4）
  - [x] 在 `src/api/error.rs` 定义 `LaputaError::AlreadyInitialized`
  - [x] 实现 `From<rusqlite::Error>` 与 `From<std::io::Error>`

- [x] Task 5: 补齐身份初始化回归测试（AC: 6）
  - [x] 使用 `tempfile::tempdir()` 隔离目录
  - [x] 使用 `#[serial]` 避免文件系统测试互相污染
  - [x] 用 `PRAGMA table_info(memories)` 验证 schema 列

### Review Findings

**Code Review: 2026-04-14**

**审查发现已转移到 Story 1-2-1-code-review-fixes.md 进行后续开发。**
- [x] [Review][Defer] MemoryRecord Serde derive 未使用 [sqlite.rs:8] — deferred, pre-existing
- [x] [Review][Defer] rusqlite 错误映射过于笼统 [error.rs:38-42] — deferred, pre-existing
- [x] [Review][Defer] uuid 格式无验证约束 [sqlite.rs:5,30] — deferred, pre-existing

---

## 开发说明

### Story 目标边界

这不是完整记忆栈初始化。

本 Story 只负责：
- 创建数据库文件
- 创建最小 `memories` schema
- 写入 `identity.md`
- 明确重复初始化错误

本 Story 不负责：
- 日记写入
- 语义索引构建
- HeatService 计算
- 唤醒包生成
- CLI 参数解析全量实现

这些职责分别在后续 Story 中展开。

### 与 Story 1.1 的衔接

Story 1.1 已经建立了 Laputa 项目骨架、`LaputaError` 基础模块、配置文件骨架和测试目录。  
因此本 Story 不能再要求开发者“如果 Cargo.toml 不存在则先创建最小工程”；那是过期前提。

本 Story 必须复用：
- `Laputa/src/api/error.rs`
- `Laputa/src/identity/mod.rs`
- `Laputa/src/storage/mod.rs`
- `Laputa/tests/`

### 与 Story 1.3 的衔接

**字段边界对比：**

| Story 1.2 最小 schema (sqlite.rs) | Story 1.3 扩展 (memory.rs) |
|-----------------------------------|----------------------------|
| id, text_content, wing, room | 继承 |
| source_file, valid_from, valid_to | 继承 |
| heat_i32, last_accessed, access_count | 继承 + 迁移测试 |
| is_archive_candidate | 继承 |
| emotion_valence, emotion_arousal | 继承 |
| — | `LaputaMemoryRecord` 完整结构 |

**约束：** 本 Story schema 为初始化最小列集合，Story 1.3 在此基础上扩展，不反向修改 sqlite.rs。

### 实现约束

- 初始化入口使用 `IdentityInitializer`
- 错误类型统一走 `LaputaError`
- 不覆盖已有 `identity.md`
- `identity.md` 与 `laputa.db` 必须同目录
- 时间写入必须是 UTC RFC3339 字符串
- schema 初始化逻辑必须可被测试直接调用，不要绑死 CLI

### 真实代码特征

当前实现中：
- `IdentityInitializer::new(config_dir)` 负责推导路径
- `IdentityInitializer::initialize(user_name)` 负责目录创建、schema 创建和身份文件写入
- `create_schema(&Connection)` 位于 `Laputa/src/storage/sqlite.rs`
- `LaputaError` 没有使用 `thiserror`，而是手写 `Display` + `Error`

后续开发不要假设：
- `LaputaError` 基于 derive 宏
- `identity.md` 使用 `.txt`
- 初始化阶段已经构建向量索引

### 测试要求

**测试依赖：**
- `serial_test = "2"` — 文件系统测试隔离，避免并发污染
- `tempfile = "3.27.0"` — 临时目录 fixture，隔离测试数据

**必须保留并通过：**
- `test_initialize_creates_db_and_identity`
- `test_reinitialize_returns_error`
- `test_initialize_returns_db_path`
- `test_schema_created_with_required_columns`

**后续修改验证：**
- 身份文件格式未破坏唤醒包的 L0 读取预期
- schema 变更不会破坏 1.3 的 memory migration 测试
- 文件系统路径仍可在 `tempdir()` 中稳定运行

---

## 前序 Story 情报

### Story 1.1 Learnings

从 `1-1-project-structure-setup.md` 可提取的当前有效上下文：
- Laputa 已按目录模块结构组织，不再是 mempalace-rs 的单文件布局
- `api/mod.rs` / `api/error.rs` 已存在，后续 Story 应复用统一错误模型
- 项目测试已大量依赖 `serial_test` 与 `tempfile` 风格的隔离模式
- 当前工作区不是 git 仓库根目录，不能依赖 git log 获取历史；Story 文档需要直接引用文件事实

### 当前仓库现实

代码库中已实际存在：
- `Laputa/src/identity/initializer.rs`
- `Laputa/src/storage/sqlite.rs`
- `Laputa/src/storage/memory.rs`
- `Laputa/tests/test_identity.rs`
- `Laputa/tests/test_memory_record.rs`

这意味着后续修订本 Story 时，应优先考虑“与现有实现对齐”，而不是“重写一个理论上的初始化方案”。

---

## 项目结构说明

与本 Story 直接相关的目录：

```text
Laputa/
├─ src/
│  ├─ api/
│  │  ├─ mod.rs
│  │  └─ error.rs
│  ├─ identity/
│  │  ├─ mod.rs
│  │  └─ initializer.rs
│  └─ storage/
│     ├─ mod.rs
│     ├─ sqlite.rs
│     └─ memory.rs
└─ tests/
   ├─ test_identity.rs
   └─ test_memory_record.rs
```

注意：`storage/sqlite.rs` 与 `storage/memory.rs` 同时存在。  
`sqlite.rs` 服务于 Story 1.2 的最小初始化语义；`memory.rs` 是 Story 1.3 的扩展记录模型。后续改动必须避免让两个文件的职责重新混乱。

---

## References

- `_bmad-output/planning-artifacts/epics.md` - Epic 1 / Story 1.2 原始故事与验收标准
- `_bmad-output/planning-artifacts/prd.md` - FR-1、FR-4、NFR-7、NFR-10
- `_bmad-output/planning-artifacts/architecture.md` - identity 初始化、统一错误处理、测试隔离约束
- `Laputa/AGENTS.md` - mempalace-rs 继承边界与项目规则
- `_bmad-output/implementation-artifacts/1-1-project-structure-setup.md` - 前序 Story 产物与约束
- `_bmad-output/implementation-artifacts/1-3-memoryrecord-extension.md` - 后续/并行 Story 对 MemoryRecord 的实际扩展
- `Laputa/src/identity/initializer.rs` - 当前实现真值来源
- `Laputa/src/storage/sqlite.rs` - 当前初始化 schema 真值来源
- `Laputa/src/api/error.rs` - 当前错误模型真值来源
- `Laputa/tests/test_identity.rs` - 当前测试覆盖真值来源

---

## Dev Agent Record

### Agent Model Used

codex/gpt-5

### Debug Log References

- 2026-04-14: 刷新故事文档，移除“Story 1.1 未完成”的过期前提
- 2026-04-14: 依据 `Laputa/src/identity/initializer.rs`、`Laputa/src/storage/sqlite.rs`、`Laputa/tests/test_identity.rs` 重新对齐验收标准与任务
- 2026-04-14: 检查到当前工作区不是 git 仓库，未纳入 git 历史分析

### Completion Notes List

- 本 Story 文件已按当前代码现实重写
- 保留故事状态为 `review`，避免错误回退 sprint 状态
- 任务与说明已聚焦于初始化职责边界，并显式声明与 1.1 / 1.3 的衔接关系

### File List

- `Laputa/src/api/error.rs`
- `Laputa/src/identity/initializer.rs`
- `Laputa/src/identity/mod.rs`
- `Laputa/src/storage/sqlite.rs`
- `Laputa/tests/test_identity.rs`
- `_bmad-output/implementation-artifacts/1-2-identity-initialization.md`

---

_故事状态: done | 最后刷新时间: 2026-04-14 | 代码审查发现已转移到 Story 1-2-1_
