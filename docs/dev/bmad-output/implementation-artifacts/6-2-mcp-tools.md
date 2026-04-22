# Story 6.2: MCP Tools 扩展

Status: review

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a AI agent, I want 通过 MCP Tools 调用天空之城能力, so that agent 可以无缝集成记忆系统。

## Acceptance Criteria

1. **Given** MCP 服务启动
   **When** agent 调用 Tools
   - `laputa_init`
   - `laputa_diary_write`
   - `laputa_recall`
   - `laputa_wakeup_generate`
   - `laputa_mark_important`
   - `laputa_get_heat_status`
   **Then** Tool handler 正确处理请求
2. **And** 返回 JSON-RPC 2.0 格式响应
3. **And** 参数命名遵循 `snake_case`

## Tasks / Subtasks

- [x] Task 1 (AC: 1-3) - 整理 MCP 工具命名与注册表
  - [x] 基于现有 `Laputa/src/mcp_server/mod.rs` 的 JSON-RPC 实现扩展，不另起一套 MCP 服务框架
  - [x] 在 `tools/list` 中补齐 `laputa_*` 工具定义
  - [x] 明确保留 `mempalace_*` 兼容策略，避免破坏现有测试与调用方
- [x] Task 2 (AC: 1, 3) - 为每个 `laputa_*` 工具定义输入 schema
  - [x] 所有参数命名使用 `snake_case`
  - [x] `laputa_init`、`laputa_diary_write`、`laputa_recall`、`laputa_wakeup_generate`、`laputa_mark_important`、`laputa_get_heat_status` 均提供最小可执行 schema
  - [x] 必填参数、默认值和 `memory_id` 类型在 schema 中写清楚
- [x] Task 3 (AC: 1-2) - 扩展 `tools/call` 分发
  - [x] 为 `laputa_*` 工具建立分发分支
  - [x] 成功响应继续使用现有 MCP content wrapper：`{"content":[{"type":"text","text":"...json..."}]}`
  - [x] 错误响应继续使用 JSON-RPC error object，不把失败包装成成功文本
- [x] Task 4 (AC: 1-3) - 将 `laputa_*` 工具绑定到现有核心能力
  - [x] `laputa_init` 复用 `IdentityInitializer`
  - [x] `laputa_diary_write` 复用 `diary` 模块
  - [x] `laputa_recall` 复用现有 recall/search 路径
  - [x] `laputa_wakeup_generate` 复用 `Searcher::wake_up()`
  - [x] `laputa_mark_important` 复用 Story 5.3 热度治理入口，不在 MCP 层重写逻辑
  - [x] `laputa_get_heat_status` 直接读取现有热度字段，不引入新数据源
- [x] Task 5 (AC: 2-3) - 统一错误与参数语义
  - [x] MCP 层错误语义向 `LaputaError` 靠拢，再映射到 JSON-RPC error
  - [x] 对无效参数、无效 `memory_id`、未初始化目录、未知工具提供稳定错误信息
  - [x] 保持 JSON-RPC `id` 原样回传
- [x] Task 6 (AC: 1-3) - 补齐 MCP 自动化测试
  - [x] 为 `tools/list` 增加 `laputa_*` 工具存在性测试
  - [x] 为 `tools/call` 增加成功路径和错误路径测试
  - [x] 为保留的 `mempalace_*` 兼容行为保留回归测试

## Dev Notes

### 实现目标

- 本故事不从零搭建 MCP 服务，而是在现有 `Laputa/src/mcp_server/mod.rs` 上完成 `mempalace_*` 到 `laputa_*` 工具契约扩展。
- 关键目标：
  - 对齐 Epic 6 的 `laputa_*` 工具契约
  - 不破坏现有 JSON-RPC 2.0 包装行为
  - 尽量复用 Story 6.1 已经沉淀的核心模块边界，而不是复制业务逻辑

### 当前代码现状

- `Laputa/src/mcp_server/mod.rs` 已有完整 MCP 骨架：
  - `initialize`
  - `tools/list`
  - `tools/call`
  - JSON-RPC request / response / error 结构
- 现有工具大部分仍然使用 `mempalace_*` 命名。
- 当前 `tools/call` 成功路径已经使用 MCP content wrapper，错误路径已经使用 JSON-RPC error object。

### 与 Story 6.1 对齐要求

- Story 6.1 已明确 CLI 应复用核心 handler，而不是在入口层重写业务。
- Story 6.2 遵循同一原则：
  - MCP tool handler 直接复用核心模块
  - 不在 `mcp_server/mod.rs` 内复制初始化、写入、召回、热度治理逻辑

### 关键实现风险

1. **命名迁移风险**
   - Epic 验收需要 `laputa_*`
   - 仓库现有实现大量使用 `mempalace_*`
   - 直接硬切会破坏已有测试和调用方
   - Phase 1 采取“双命名共存，新名为主，旧名兼容”的策略
2. **ID 语义不一致**
   - 当前运行时主路径大量使用 `i64 memory_id`
   - Story 6.2 继续沿用 `i64` 语义，在 schema 和错误消息中写清楚
3. **`laputa_mark_important` 依赖热度治理入口**
   - MCP 层只复用已有 intervention 入口
   - 不在 MCP 层重写 important / forget / emotion anchor 逻辑
4. **`laputa_get_heat_status` 不应发明新数据源**
   - 直接读取现有 `heat_i32`、`last_accessed`、`access_count`、`is_archive_candidate`

### 推荐工具映射

- `laputa_init`
  - 复用 `Laputa/src/identity/initializer.rs`
  - 返回初始化状态和数据库路径
- `laputa_diary_write`
  - 复用 `Laputa/src/diary/mod.rs`
  - 至少接受 `agent` 和 `content`
- `laputa_recall`
  - 复用 `Laputa/src/searcher/mod.rs`
  - 返回结构化 JSON，再包进 MCP content wrapper
- `laputa_wakeup_generate`
  - 复用 `Searcher::wake_up()`
- `laputa_mark_important`
  - 复用 Story 5.3 对应的 intervention 入口
- `laputa_get_heat_status`
  - 返回 `memory_id`、`heat_i32`、`last_accessed`、`access_count`、`is_archive_candidate`

### JSON-RPC / MCP Guardrails

- 必须保持：
  - `jsonrpc: "2.0"`
  - request `id` 原样返回
  - notification 无响应
  - `tools/list` 返回 `tools` 数组
  - `tools/call` 返回 `content` 数组
- 错误时返回 JSON-RPC error object，不使用 HTTP 风格错误，也不把错误塞进成功文本。

### 代码落点建议

```text
Laputa/
└── src/
    ├── mcp_server/
    │   └── mod.rs              # [MODIFY]
    ├── identity/
    │   └── initializer.rs      # [EXISTING]
    ├── diary/
    │   └── mod.rs              # [EXISTING]
    ├── searcher/
    │   └── mod.rs              # [EXISTING]
    ├── storage/
    │   ├── memory.rs           # [EXISTING]
    │   └── sqlite.rs           # [EXISTING]
    └── api/
        └── error.rs            # [EXISTING]
```

### 测试要求

- 至少覆盖：
  - `tools/list` 包含全部 6 个 `laputa_*` 工具
  - `tools/call` 覆盖各 `laputa_*` 工具的成功或预期失败语义
  - 未知工具返回错误
  - 缺参返回错误
  - 非法 `memory_id` 返回错误
  - 保留的 `mempalace_*` 工具继续通过现有回归测试

### 最新依赖与技术信息

- 当前仓库相关依赖已锁定：
  - `tokio = 1.51.0`
  - `serde = 1.0.228`
  - `serde_json = 1.0.149`
  - `mcp_rs = 0.1.0`
- 本故事不需要升级依赖，重点是完成工具命名与处理边界对齐。

### Project Structure Notes

- 理想上可拆为 `src/mcp_server/tools.rs` / `handlers.rs`，但当前 repo 仍以 `src/mcp_server/mod.rs` 为主。
- 本故事允许继续在 `mod.rs` 内落地，但实现结构应朝可拆分方向组织。

### References

- [Source: `_bmad-output/planning-artifacts/epics.md` - Epic 6 / Story 6.2]
- [Source: `_bmad-output/planning-artifacts/prd.md` - FR-15]
- [Source: `_bmad-output/planning-artifacts/architecture.md` - ADR-009 API 设计模式]
- [Source: `_bmad-output/planning-artifacts/architecture.md` - ADR-010 错误处理标准]
- [Source: `_bmad-output/planning-artifacts/architecture.md` - MCP Tool 命名与 `snake_case` 规范]
- [Source: `_bmad-output/planning-artifacts/architecture.md` - 6.3 架构边界定义]
- [Source: `Laputa/src/mcp_server/mod.rs`]
- [Source: `Laputa/src/searcher/mod.rs`]
- [Source: `Laputa/src/identity/initializer.rs`]
- [Source: `Laputa/src/diary/mod.rs`]
- [Source: `Laputa/src/storage/memory.rs`]
- [Source: `Laputa/src/storage/sqlite.rs`]
- [Source: `Laputa/src/api/error.rs`]
- [Source: `_bmad-output/implementation-artifacts/6-1-cli-commands.md`]

## Dev Agent Record

### Agent Model Used

GPT-5

### Debug Log References

- Implemented `laputa_*` MCP tools in `Laputa/src/mcp_server/mod.rs`.
- Added JSON-RPC error classification, initialization validation, `time_range` parsing, and numeric `memory_id` validation helpers.
- Ran `cargo fmt`.
- Ran `cargo test mcp_server -- --nocapture`.

### Completion Notes List

- 在现有 `mcp_server/mod.rs` 上补齐 `laputa_init`、`laputa_diary_write`、`laputa_recall`、`laputa_wakeup_generate`、`laputa_mark_important`、`laputa_get_heat_status`，未新起 MCP 框架。
- `tools/list` 新增 6 个 `laputa_*` schema，并保持字段名为 `snake_case`。
- `tools/call` 新增 6 个分发分支，成功路径继续返回 MCP `content` wrapper，错误路径继续返回 JSON-RPC error object。
- 新增初始化校验、`time_range` 解析、`memory_id` 数字校验和 `LaputaError -> JSON-RPC` 映射。
- recall 的 MCP 输出会剥离 diary 底层 `DIARY_META` 头，避免泄漏内部存储格式。
- 增加 `mcp_server` 测试覆盖：`tools/list`、`snake_case` schema、`tools/call` 成功/失败路径，以及 `mempalace_*` 兼容回归。

### File List

- `Laputa/src/mcp_server/mod.rs`
- `_bmad-output/implementation-artifacts/6-2-mcp-tools.md`
- `_bmad-output/implementation-artifacts/sprint-status.yaml`

### Change Log

- 2026-04-15: Implemented `laputa_*` MCP tools, added JSON-RPC error mapping, and expanded MCP server tests for Story 6.2.
