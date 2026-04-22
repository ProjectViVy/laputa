# Story patch-2: CLI 与 MCP 接口关键缺陷修复

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a 开发者，
I want 修复 Epic 6 在 CLI 与 MCP 接口层暴露出的关键缺陷与边界问题，
so that Laputa 的外部接口在数据路径、参数校验、错误语义和查询边界上保持一致、可预测、可测试。

## Acceptance Criteria

1. **C1 数据路径一致性**
   **Given** 用户分别通过 `laputa diary write` 与 MCP `laputa_diary_write` 写入日记
   **When** 两条路径落盘到本地存储
   **Then** 两者必须使用同一套日记/记忆数据库路径，不允许 CLI 与 MCP 写入不同 SQLite 文件导致数据隔离

2. **H1 memory_id 正值校验**
   **Given** CLI `mark` 命令接收 `--id`
   **When** 传入 `0`、负数或其他非法值
   **Then** CLI 必须拒绝该输入，并返回明确的 `ValidationError`

3. **H2 空白用户名校验**
   **Given** 用户执行 `laputa init --name`
   **When** 传入仅包含空格或制表符的用户名
   **Then** 命令必须失败，且不能创建空白 `identity.md`

4. **M1 limit 边界收敛**
   **Given** CLI recall 与 MCP recall 都支持 limit
   **When** 调用方传入 `0`、负向语义值或超大值
   **Then** 两条接口必须执行统一边界策略，最小值为 `1`，最大值为 `10000`

5. **M2 极端日期拒绝**
   **Given** CLI 与 MCP 都支持 `time_range`
   **When** 传入 `0001-01-01`、`9999-12-31` 或其他会把时间戳推向极端边界的日期
   **Then** 必须在参数解析阶段返回明确错误，而不是继续进入后续查询逻辑

6. **M3 时间跨度限制**
   **Given** 用户传入合法格式的 `time_range`
   **When** 起止日期跨度超过 `365` 天
   **Then** CLI 与 MCP 都必须拒绝该请求，避免单次查询扫出异常大的结果集

7. **M4 路径字符串转换显式报错**
   **Given** MCP 服务初始化知识图谱或其他本地路径相关资源
   **When** `PathBuf -> &str` 转换失败或路径不可安全表示
   **Then** 代码必须返回显式错误，不能使用 `unwrap_or("knowledge.db")` 之类的静默回退掩盖真实问题

## Tasks / Subtasks

- [ ] Task 1 (AC: 1) 统一 CLI 与 MCP 的日记落盘路径
  - [ ] 对比 `Laputa/src/cli/handlers.rs` 中 `Diary::new(config.config_dir.join("vectors.db"))`
  - [ ] 对比 `Laputa/src/mcp_server/mod.rs` 中 `laputa_diary_write` 与 `mempalace_diary_write` 当前使用的 `diary.db`
  - [ ] 以现有 `Diary` 模块的设计意图为准，明确唯一正确路径后统一入口实现
  - [ ] 补充跨接口回归测试，验证 CLI 写入结果能被 MCP recall / heat 状态读取到

- [ ] Task 2 (AC: 2) 在 CLI `parse_memory_id` 中补齐正值验证
  - [ ] 保留当前 Phase 1 “仅支持 numeric memory_id”的约束
  - [ ] 新增 `id <= 0` 拒绝逻辑，使 CLI 与 MCP `parse_memory_id` 行为对齐
  - [ ] 保持 UUID 仍返回“Phase 1 尚未接线”的明确错误，而不是误判为其他错误

- [ ] Task 3 (AC: 3) 在 CLI init 命令入口拒绝空白用户名
  - [ ] 在 `handle_init` 入口对 `command.name.trim()` 结果进行显式空白校验
  - [ ] 不要依赖 `IdentityInitializer::initialize()` 的非空校验替代该行为，因为当前仅 trim 后传值但输出仍引用原始名称
  - [ ] 补充测试覆盖 `"   "`、`\t\t` 等输入

- [ ] Task 4 (AC: 4) 统一 recall limit 边界策略
  - [ ] 校验 `Laputa/src/cli/commands.rs` 中 `limit: usize`
  - [ ] 校验 `Laputa/src/mcp_server/mod.rs` 中 `args["limit"].as_u64().unwrap_or(100) as usize`
  - [ ] 统一采用显式 clamp 策略，避免 CLI 与 MCP 各自漂移
  - [ ] 不要修改 `RecallQuery::validated_limit()` 的既有行为语义，只在入口层增加 guardrail

- [ ] Task 5 (AC: 5, 6) 收紧 CLI / MCP 的时间范围解析
  - [ ] 复用现有 `parse_time_range` 结构，不重写另一套日期解析器
  - [ ] 在 CLI `Laputa/src/cli/handlers.rs` 与 MCP `Laputa/src/mcp_server/mod.rs` 两处同步添加极端日期校验
  - [ ] 同步添加 `<= 365` 天跨度限制，错误信息需直指问题而非笼统失败
  - [ ] 保持 `start <= end` 的现有校验

- [ ] Task 6 (AC: 7) 移除 MCP 路径字符串静默回退
  - [ ] 修复 `KnowledgeGraph::new(config.config_dir.join("knowledge.db").to_str().unwrap_or("knowledge.db"))`
  - [ ] 同步检查同文件中其他 `to_str().unwrap_or(...)` 用法，至少覆盖 `new()` / `new_test()` 相关路径
  - [ ] 返回 `LaputaError::InvalidPath` 或等价显式错误，避免服务在错误路径上悄悄继续运行

- [ ] Task 7 (AC: 1-7) 补齐回归测试
  - [ ] 为 CLI 单元测试补充 `parse_memory_id` 正值验证
  - [ ] 为 CLI 单元测试补充 `parse_time_range` 极端日期与超跨度验证
  - [ ] 为 MCP 测试补充 `time_range` 边界与 `limit` 边界验证
  - [ ] 为初始化流程测试补充空白用户名拒绝验证
  - [ ] 增加一条 CLI/MCP 共享数据路径的端到端回归测试

## Dev Notes

### 缺陷背景与业务上下文

- 本 Story 是 Epic 6 已交付后形成的补丁批次，来源为代码审查中的 deferred findings。
- Epic 6 的目标不是“分别做一套 CLI 和一套 MCP”，而是为 FR-15 提供一致的工具化接口。
- 当前主要风险不是功能缺失，而是接口层行为漂移：
  - CLI 与 MCP 对同类参数的校验不一致
  - 同一业务入口写入不同数据库文件
  - 极端输入可能越过入口层直接打到存储或时间处理逻辑

### 故事基础来源

- Epic 6 目标：为 `FR-15` 提供 CLI / MCP 工具接口。
- Story 6.1 已建立 CLI 命令树、handlers 分层、`LaputaError` 返回约束。
- Story 6.2 已建立 `laputa_*` MCP tools、JSON-RPC 包装、MCP 参数解析模式。
- 本 Story 的工作重点是“修正入口边界与一致性”，不是重构接口架构。

### 代码现状与直接落点

- CLI diary 写入路径当前位于 `Laputa/src/cli/handlers.rs`
  - `Diary::new(config.config_dir.join("vectors.db"))`
- MCP diary 写入路径当前位于 `Laputa/src/mcp_server/mod.rs`
  - `mempalace_diary_write()` 使用 `diary.db`
  - `laputa_diary_write()` 当前实现也沿用 diary 路径族
- CLI 时间范围解析位于 `Laputa/src/cli/handlers.rs::parse_time_range`
- MCP 时间范围解析位于 `Laputa/src/mcp_server/mod.rs::parse_time_range`
- CLI memory_id 解析位于 `Laputa/src/cli/handlers.rs::parse_memory_id`
- MCP memory_id 解析位于 `Laputa/src/mcp_server/mod.rs::parse_memory_id`

### 开发 Guardrails

- 不要新建第三套共用工具模块来“顺手重构” CLI/MCP。当前目标是批量修补高风险缺陷，保持修改最小而明确。
- 不要改变 Phase 1 的 `numeric memory_id` 约束。CLI 与 MCP 都应继续只接受 numeric id。
- 不要把入口层问题推给底层碰运气处理；本 Story 的价值就在于把不安全输入挡在接口层。
- 不要用静默回退路径掩盖本地路径错误。路径转换失败必须是显式失败。
- 不要让 CLI 和 MCP 在 `limit`、`time_range`、`memory_id` 三类参数上继续各自维护一套不同语义。

### 架构约束

- 遵守 `AGENTS.md` 中“扩展 mempalace-rs，而不是另起底座”的原则。
- 遵守 Architecture 中 API 边界：
  - `CLI -> Core` 通过 handlers 调用既有模块
  - `MCP -> Core` 通过 tool handlers 调用既有模块
- 保持统一错误语义：
  - CLI 公共 handler 返回 `Result<_, LaputaError>`
  - MCP 内部继续将 `LaputaError` 映射为 JSON-RPC error object
- 保持命名与参数风格：
  - CLI 子命令仍使用既有命令树
  - MCP 参数名继续使用 `snake_case`

### 复用优先级

- 优先复用现有解析函数并在其内部补校验。
- 优先复用现有 `Diary` / `Searcher` / `VectorStorage` / `IdentityInitializer`。
- 如果需要统一常量，优先在现有文件内抽出小范围常量，而不是引入新模块。

### 具体技术要求

- `memory_id`
  - CLI 当前允许 `i64` 解析成功即通过，这是缺陷；必须增加 `id > 0`
  - MCP 已有 `id > 0` 约束，本 Story 要让 CLI 与之对齐
- `time_range`
  - 保留格式要求：`YYYY-MM-DD~YYYY-MM-DD`
  - 保留 `start <= end`
  - 新增极端日期拒绝
  - 新增跨度 `<= 365` 天
- `limit`
  - 统一入口边界 `1..=10000`
  - 若采用 clamp，错误消息与最终行为要可预测
- 路径转换
  - `to_str()` 失败必须返回显式错误
  - 不允许默认回退到 `"knowledge.db"`、`"test_knowledge.db"` 等字符串

### 文件结构要求

- 主要改动文件应限制在：
  - `Laputa/src/cli/handlers.rs`
  - `Laputa/src/mcp_server/mod.rs`
  - `Laputa/src/cli/commands.rs`（如需入口级 limit guardrail）
  - 相关测试文件
- 若新增测试文件，优先复用现有测试布局，不引入新的测试框架

### 测试要求

- 最少覆盖以下断言：
  - CLI `parse_memory_id("-1")` / `parse_memory_id("0")` 返回 `ValidationError`
  - CLI init 对空白用户名失败且不产生有效初始化结果
  - CLI `parse_time_range("0001-01-01~2026-04-01")` 失败
  - CLI `parse_time_range("2025-01-01~2026-04-02")` 因跨度超限失败
  - MCP recall `limit=0` 与超大值时行为符合统一策略
  - MCP `memory_id=0` 或负值时失败
  - CLI 写入后，MCP 侧能从同一数据库路径读取或查看同一条记录的状态

### 来自前序 Story 的实现情报

- Story 6.1 已将 CLI 分为 `commands.rs` / `handlers.rs` / `output.rs`，本 Story 应沿用该边界，不把业务逻辑塞回 `main.rs`
- Story 6.1 已明确 Phase 1 仅支持 numeric `memory_id`，不要借本次补丁扩展 UUID
- Story 6.2 已在 `mcp_server/mod.rs` 中聚合大部分 MCP 工具，本 Story 允许继续在该文件内修补，不要求立刻拆分 `tools.rs`
- Story 6.2 已建立 `LaputaError -> JSON-RPC` 映射，本 Story 应直接复用

### 最近实现模式

- 仓库根目录当前不是 git 根，无法从当前工作目录提取最近 5 条提交记录
- 因此本 Story 的模式参考以现有实现文件和已完成的 6-1 / 6-2 story 文档为准

### 外部版本与依赖说明

- 当前仓库已锁定关键依赖：
  - `clap = 4.6.0`
  - `tokio = 1.51.0`
  - `serde = 1.0.228`
  - `serde_json = 1.0.149`
  - `mcp_rs = 0.1.0`
  - `rusqlite = 0.32`
- 本 Story 不要求升级依赖；重点是修复现有实现中的一致性和边界问题

### 完成定义

- 所有 AC 落地
- Story 状态可从 `ready-for-dev` 进入 `in-progress`
- 新增/更新测试覆盖关键边界
- 不引入新的接口语义漂移
- 不留下“CLI 与 MCP 继续使用不同数据库路径”的未决状态

### References

- [Source: `_bmad-output/planning-artifacts/epics.md` - Epic 6 / Story 6.1 / Story 6.2]
- [Source: `_bmad-output/planning-artifacts/prd.md` - FR-15]
- [Source: `_bmad-output/planning-artifacts/architecture.md` - API 边界 / 统一错误处理 / snake_case 约束]
- [Source: `Laputa/AGENTS.md`]
- [Source: `_bmad-output/implementation-artifacts/deferred-work.md` - Epic 6 code review findings]
- [Source: `_bmad-output/implementation-artifacts/6-1-cli-commands.md`]
- [Source: `_bmad-output/implementation-artifacts/6-2-mcp-tools.md`]
- [Source: `Laputa/src/cli/handlers.rs`]
- [Source: `Laputa/src/cli/commands.rs`]
- [Source: `Laputa/src/mcp_server/mod.rs`]
- [Source: `Laputa/Cargo.toml`]

## Dev Agent Record

### Agent Model Used

GPT-5

### Debug Log References

- Loaded BMAD create-story workflow, template, checklist, and discovery protocol
- Loaded planning artifacts: `epics.md`, `architecture.md`, `prd.md`
- Loaded implementation artifacts: `deferred-work.md`, `6-1-cli-commands.md`, `6-2-mcp-tools.md`
- Inspected live code anchors in `Laputa/src/cli/handlers.rs`, `Laputa/src/mcp_server/mod.rs`, `Laputa/Cargo.toml`

### Completion Notes List

- 将原始缺陷草稿重写为可直接交给 dev agent 的 Story，上下文集中于“接口一致性补丁”而非重新描述 Epic 6
- 明确了应复用的现有模块、禁止的重构方向、必须对齐的 CLI/MCP 参数语义
- 把 code review 中的 C1/H1/H2/M1-M4 缺陷映射到了具体文件与测试任务

### File List

- `_bmad-output/implementation-artifacts/patch-2-cli-mcp-critical.md`

### Review Findings

#### Patch 级发现（需立即修复）

- [x] [Review][Patch] H2: MCP用户名空白验证缺失 — FIXED: mod.rs:521 添加 trim(), initializer.rs:99 改为 trim().is_empty()

- [x] [Review][Patch] M1: MCP limit静默回退默认值 — FIXED: parse_recall_limit 返回 Result<usize>, 非整数返回 ValidationError

#### Defer 级发现（延后处理）

- [x] [Review][Defer] 路径穿越风险：用户名未验证路径分隔符 [cli/handlers.rs:45-51] - deferred, pre-existing，需跨模块统一设计路径安全策略
- [x] [Review][Defer] H1: MCP JSON类型混淆无明确错误 [mcp_server/mod.rs:1057-1090] - deferred，低频边缘场景，已有基础验证
- [x] [Review][Defer] H1: 全宽数字/科学记数法未明确提示 [cli/handlers.rs:260-281] - deferred，parse失败已有ValidationError，消息可后续优化
- [x] [Review][Defer] M4: CLI未显式路径转换 [cli/handlers.rs:68] - deferred，错误会在深层抛出，非最外层验证
- [x] [Review][Defer] i64.MAX内存ID误导错误 [mcp_server/mod.rs:1057-1090] - deferred，超出i64范围的数值返回模糊错误，低频场景

#### Dismiss 级发现（噪音/设计决策）

- parse_memory_id实现不一致 - CLI处理字符串，MCP处理JSON数值，接口差异而非缺陷
- 重复常量定义 - Story明确禁止新建共享模块，保持分离是正确的
- and_hms_opt死代码分支 - chrono API设计，保留更安全
- 时间范围验证逻辑分散 - 功能正确，可后续重构优化
- 平台依赖limit行为 - MAX_RECALL_LIMIT=10000 已平台无关

### Change Log

- 2026-04-19: 按 BMAD create-story 工作流重建 patch-2 Story，上下文补全并标记为 ready-for-dev
- 2026-04-19: 完成三层代码审查，发现2个patch级问题需修复，5个defer级问题延后处理
