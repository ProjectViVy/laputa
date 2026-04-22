---
stepsCompleted:
  - step-01-validate-prerequisites
  - step-02-design-epics
  - step-03-create-stories
  - step-04-final-validation
  - epic-10-append
workflowType: 'epics-and-stories'
lastStep: 4
status: 'in-progress'
completedAt: '2026-04-13'
lastUpdated: '2026-04-21'
inputDocuments:
  - D:/VIVYCORE/newmemory/_bmad-output/planning-artifacts/prd.md
  - D:/VIVYCORE/newmemory/_bmad-output/planning-artifacts/architecture.md
  - D:/VIVYCORE/newmemory/_bmad-output/implementation-artifacts/deferred-work.md
---

# 天空之城 (Laputa) - Epic Breakdown

## Overview

本文档提供天空之城项目的完整 Epic 和 Story 分解，将 PRD、架构决策分解为可实现的 Story。

## Requirements Inventory

### Functional Requirements

**FR-1**: 主体初始化 - 用户可以创建一个新的天空之城记忆库，并生成最小可运行的身份、状态与记忆布局

**FR-2**: 日记写入 - 用户可以写入新的日记条目，并指定或推断时间、分区、标签、来源与情绪锚定信息

**FR-3**: 记忆筛选 - 系统可以对新增内容执行可解释的保留判断，包括存储、丢弃、合并或加热既有条目

**FR-4**: 统一记忆记录 - 系统可以将事件、关系、状态、摘要胶囊与主体信号保存为统一结构的记忆记录，并支持稳定 ID

**FR-5**: 节律整理 - 系统可以在节律点对新增素材进行整理，并生成至少周级摘要胶囊

**FR-6**: 唤醒包生成 - 系统可以输出一个用于 AI 启动时注入的唤醒包，默认覆盖身份、近期状态、最近周期摘要、关键关系和关键任务

**FR-7**: 时间流召回 - 用户或上层 agent 可以按时间窗口、周期层级或最近阶段召回相关记忆

**FR-8**: 深度检索 - 用户或上层 agent 可以对记忆库发起按需深检索，以获取与当前问题最相关的历史条目

**FR-9**: 关系记忆 - 系统可以记录并召回人与人、人与项目、人与主体之间的最小关系变化与重要互动

**FR-10**: 情绪锚定 - 系统可以记录记忆条目的情绪权重，并使其影响保留、强化或召回顺序

**FR-11**: 手动治理 - 用户可以对记忆条目执行至少以下干预：标记重要、标记遗忘、设置情绪锚点、手动回看

**FR-12**: 归档候选 - 系统可以识别低热度、久未调用内容，并将其标记为归档候选

**FR-13**: 归档导出 - 系统可以将归档候选打包导出，并保留用户后续解包与考古的能力

**FR-14**: 数据导出与迁移 - 用户可以导出其主体定义、摘要胶囊与核心记忆记录，用于迁移到新设备、新模型或新宿主

**FR-15**: 工具化接口 - 系统可以通过 CLI 与 MCP 暴露初始化、写入、整理、唤醒、召回、检索、标记与归档等能力

### NonFunctional Requirements

**NFR-1**: 纯 Rust 运行 - 系统应在无 Python 运行时依赖的前提下完成核心安装、初始化、写入、整理、召回、检索与导出流程

**NFR-2**: 离线可用性 - 系统应在无外部网络条件下完成全部 MVP 核心能力，且本地数据读写与召回能力不依赖远程服务

**NFR-3**: 唤醒成本控制 - 默认唤醒包应控制在 1200 tokens 以内，且 95% 的普通唤醒请求不需要全库扫描

**NFR-4**: 本地响应性能 - 在 100,000 条以内记忆记录规模下，普通唤醒请求应在 500ms 内返回，深检索请求应在 2s 内返回首批结果

**NFR-5**: 数据可解释性 - 对任一条被保留、强化、遗忘或归档的记录，系统应保留可审查的原因字段或等价解释元数据

**NFR-6**: 可迁移性 - 用户可通过文档化格式导出核心主体、关系、摘要与记忆数据，并在新实例中恢复，不依赖专有云服务

**NFR-7**: 稳定性 - 核心记忆存储需支持异常恢复，单次异常退出后不得造成已确认写入记录的结构性损坏

**NFR-8**: 扩展性 - MVP 的数据模型应允许后续增加月/季/年节律、更多主体分区与更丰富归档层，而无需重写全部基础结构

**NFR-9**: 隐私默认值 - 系统默认本地保存，不主动上传、不默认同步、不默认将原始记忆内容发送至第三方服务

**NFR-10**: 可测试性 - MVP 每一条核心链路均应具备自动化测试：初始化、写入、节律整理、唤醒包生成、召回、检索、归档导出

### Additional Requirements (Architecture)

**ARCH-1**: 基于 mempalace-rs 代码基线演化，继承 7 个必须模块 (D-006, ADR-006)

**ARCH-2**: 新增 L4 归档层，独立 Archiver 组件 (D-007, ADR-005)

**ARCH-3**: 热度机制 Phase 1 必须上线：i32 存储 + HeatService + 状态机 (ADR-001~003)

**ARCH-4**: 混合触发策略：读取时 touch + 定时批量衰减 (ADR-004)

**ARCH-5**: 四区间状态机：锁定(>80)/正常(50-80)/归档候选(20-50)/打包候选(<20) (Step 3.4)

**ARCH-6**: 统一抽象层 MemoryOperation trait (CLI/MCP/Python) (ADR-009)

**ARCH-7**: 统一错误处理 LaputaError 枚举 (ADR-010)

**ARCH-8**: 配置管理 config.toml (heat/archive/storage/wakeup) (ADR-011)

**ARCH-9**: 测试隔离策略：serial_test + TimeMachine + 分层 Fixture (ADR-012)

**ARCH-10**: 共振度 + valence + arousal (UPSP 融合概念) (D-003-B)

### UX Design Requirements

(无 - CLI/MCP 项目)

### FR Coverage Map

| FR | Epic | 状态 |
|----|------|------|
| FR-1 | Epic 1 | ✅ 覆盖 |
| FR-2 | Epic 2 | ✅ 覆盖 |
| FR-3 | Epic 2, Epic 5 | ✅ 覆盖 |
| FR-4 | Epic 1, Epic 2 | ✅ 覆盖 |
| FR-5 | Epic 4 | ✅ 覆盖 |
| FR-6 | Epic 3 | ✅ 覆盖 |
| FR-7 | Epic 3 | ✅ 覆盖 |
| FR-8 | Epic 3 | ✅ 覆盖 |
| FR-9 | Epic 7 | ✅ 覆盖 |
| FR-10 | Epic 2, Epic 7 | ✅ 覆盖 |
| FR-11 | Epic 5 | ✅ 覆盖 |
| FR-12 | Epic 5 | ✅ 覆盖 |
| FR-13 | Epic 8 | ✅ 覆盖 |
| FR-14 | Epic 8 | ✅ 覆盖 |
| FR-15 | Epic 6 | ✅ 覆盖 |

**覆盖率**: 15/15 FR = **100%**

## Epic List

### Epic 1: 记忆库初始化

**Epic 目标**: 用户可以创建一个新的天空之城记忆库，建立最小可运行的身份、状态与记忆布局。基于 mempalace-rs 代码基线，继承 7 个必须模块。

**完成价值**: 用户拥有可运行的记忆库基底。

**FR 覆盖**: FR-1, FR-4

---

### Epic 2: 日记与记忆输入

**Epic 目标**: 用户可以写入日记条目（时间、分区、标签、情绪），系统执行可解释的记忆筛选判断（存储/丢弃/合并）。记录情绪锚点。

**完成价值**: 用户可以持续记录记忆。

**FR 覆盖**: FR-2, FR-3, FR-10

---

### Epic 3: 记忆检索与唤醒 ⭐ 核心验证点

**Epic 目标**: 用户可以按时间流检索记忆。AI agent 可以发起深度语义检索。系统生成唤醒包（身份+近期状态+摘要+关系）。MVP 成败的关键。

**完成价值**: 用户可以"找回那一刻"——证明产品价值。

**FR 覆盖**: FR-6, FR-7, FR-8

**MVP 优先级**: P1 核心验证点

---

### Epic 4: 节律整理与摘要

**Epic 目标**: 系统在节律点自动整理新增素材。生成周级摘要胶囊。

**完成价值**: 系统自动压缩记忆。

**FR 覆盖**: FR-5

---

### Epic 5: 热度机制与手动治理

**Epic 目标**: 热度机制（i32 + HeatService + 状态机）。用户可标记重要/遗忘/情绪锚点。归档候选标记（Phase 1 只标记）。

**完成价值**: 记忆有生命周期治理。

**FR 覆盖**: FR-3, FR-11, FR-12

**技术关键**: ADR-001~005 决策

---

### Epic 6: CLI与MCP工具接口

**Epic 目标**: CLI 子命令：init/write/recall/wakeup。MCP Tools：20+ 工具暴露全部能力。

**完成价值**: 用户有完整工具接口。

**FR 覆盖**: FR-15

---

### Epic 7: 关系与情绪记忆

**Epic 目标**: 记录关系变化与共振度。情绪锚定（valence + arousal）。

**完成价值**: 记忆有情感维度。

**FR 覆盖**: FR-9, FR-10

---

### Epic 8: 归档与数据迁移

**Epic 目标**: 归档候选打包导出。主体/摘要/记忆导出与迁移。

**完成价值**: 用户可迁移记忆库。

**FR 覆盖**: FR-13, FR-14

---

## Epic 1 Stories

### Story 1.1: 项目结构建立与模块继承

As a 开发者，
I want 建立 Laputa 项目结构并继承 mempalace-rs 模块，
So that 代码基线继承完成，可以开始开发。

**Acceptance Criteria:**

- **Given** mempalace-rs 代码库已验证
- **When** 执行项目初始化脚本
- **Then** Laputa 目录结构完成：
  - `src/heat/`, `src/archiver/`, `src/wakeup/`, `src/rhythm/`, `src/identity/`, `src/cli/`, `src/api/`, `src/utils/`
  - 继承模块：`src/storage/`, `src/searcher/`, `src/knowledge_graph/`, `src/dialect/`, `src/diary/`, `src/mcp_server/`
- **And** Cargo.toml 配置继承 mempalace-rs 依赖版本

### Story 1.2: 主体身份初始化

As a 用户，
I want 创建一个新的天空之城记忆库实例，并定义基础身份，
So that 可以开始记录个人记忆。

**Acceptance Criteria:**

- **Given** Laputa 项目已初始化
- **When** 用户执行初始化命令
- **Then** 创建 `laputa.db` SQLite 数据库
- **And** L0 层写入 identity.md 包含：
  - user_name: "大湿"
  - user_type: "个人记忆助手"
  - created_at: 当前时间
- **And** 初始化成功返回数据库路径

### Story 1.3: MemoryRecord 数据结构扩展

As a 开发者，
I want 扩展 MemoryRecord 结构添加热度/情绪字段，
So that 生命周期治理和情绪维度可用。

**Acceptance Criteria:**

- **Given** mempalace-rs 的 `storage/memory.rs` 已继承
- **When** 扩展 MemoryRecord 结构
- **Then** 新增字段：
  - `heat_i32: i32` (热度，放大100倍)
  - `last_accessed: DateTime<Utc>`
  - `access_count: u32`
  - `emotion_valence: i32` (-100~+100)
  - `emotion_arousal: u32` (0~100)
  - `is_archive_candidate: bool`
- **And** 单元测试验证字段读写

---

## Epic 2 Stories

### Story 2.1: 日记写入核心功能

As a 用户，
I want 写入日记条目并指定时间、分区、标签与情绪，
So that 系统可以保存并解释我的记忆输入。

**Acceptance Criteria:**

- **Given** 用户有已初始化的记忆库
- **When** 用户调用 `diary.write(content, tags, emotion)`
- **Then** 系统创建 MemoryRecord 并写入 L1 层
- **And** 热度初始值设置为 5000（正常区间）
- **And** 情绪编码映射到 EMOTION_CODES

### Story 2.2: 记忆筛选与合并逻辑

As a 系统，
I want 对新增内容执行可解释的保留判断，
So that 低价值内容被过滤或合并，高价值内容被强化。

**Acceptance Criteria:**

- **Given** 新增日记条目
- **When** MemoryGate 筛选逻辑执行
- **Then** 重复内容合并到既有条目（热度 +500）
- **And** 低价值内容标记为 discard_candidate
- **And** 筛选原因写入 reason 字段

### Story 2.3: 情绪锚定记录

As a 用户，
I want 为记忆条目设置情绪锚点，
So that 重要情感记忆获得特殊保留权重。

**Acceptance Criteria:**

- **Given** 记忆条目已存在
- **When** 用户调用 `mark_emotion_anchor(memory_id, valence, arousal)`
- **Then** heat_i32 += 2000（保鲜7天）
- **And** emotion_valence 设置为指定值（-100~+100）
- **And** emotion_arousal 设置为指定值（0~100）

---

## Epic 3 Stories

### Story 3.1: 时间流检索

As a 用户，
I want 按时间窗口或周期层级检索记忆，
So that 我可以回顾特定阶段的经历。

**Acceptance Criteria:**

- **Given** 记忆库有 L0-L3 数据
- **When** 用户调用 `recall.by_time_range(start, end)`
- **Then** 返回时间范围内的记忆列表
- **And** 按热度排序（高热度优先）
- **And** 结果限制在合理数量（默认100条）

### Story 3.2: 语义检索（RAG）

As a AI agent，
I want 发起深度语义检索获取相关记忆，
So that 我可以补充当前对话的历史上下文。

**Acceptance Criteria:**

- **Given** usearch 向量索引已建立
- **When** agent 调用 `search.semantic(query, top_k)`
- **Then** 返回语义最相关的记忆列表
- **And** 每条结果包含相似度分数
- **And** RAG 能力启用（D-004 决策）

### Story 3.3: 唤醒包生成

As a AI agent，
I want 在会话启动时获取唤醒上下文包，
So that 我可以注入用户身份、近期状态、关键关系与摘要。

**Acceptance Criteria:**

- **Given** 记忆库有身份定义和近期记忆
- **When** agent 调用 `wakeup.generate()`
- **Then** 输出 WakePack 包含：
  - 身份定义（来自 identity.md）
  - 近期状态（最近7天高热度记忆）
  - 周级摘要胶囊（来自 rhythm）
  - 关键关系（共振度 > 50）
- **And** token_count < 1200（NFR-3）
- **And** 响应时间 < 500ms（NFR-4）

### Story 3.4: 混合检索排序

As a 系统，
I want 融合时间流与语义检索结果并加热度排序，
So that 用户获得最相关且有价值的记忆。

**Acceptance Criteria:**

- **Given** 时间流和语义检索结果
- **When** hybrid_search 执行
- **Then** 结果按热度因子加权排序
- **And** 重复结果去重
- **And** 最终结果限制在 top_k

---

## Epic 4 Stories

### Story 4.1: 周级摘要胶囊生成

As a 系统，
I want 在每周节律点自动整理记忆素材，
So that 生成的摘要胶囊可用于后续唤醒。

**Acceptance Criteria:**

- **Given** 本周有足够日记素材（>7条）
- **When** scheduler 触发 weekly_capsule
- **Then** 生成 SummaryCapsule 包含：
  - 本周关键词提取
  - 高热度事件摘要
  - 关系变化记录
- **And** 胶囊写入 L2 层
- **And** AAAK 压缩（~30x）

### Story 4.2: 节律调度器

As a 系统，
I want 定时触发整理任务，
So that 节律整理自动化执行。

**Acceptance Criteria:**

- **Given** scheduler 配置生效
- **When** 达到节律点（每周一凌晨）
- **Then** 触发 weekly_capsule 任务
- **And** 任务状态记录到日志
- **And** 异常情况写入 error 字段

---

## Epic 5 Stories

### Story 5.1: HeatService 核心计算

As a 系统，
I want 计算记忆热度并维护热度状态机，
So that 记忆生命周期可管理。

**Acceptance Criteria:**

- **Given** MemoryRecord 有 heat_i32 字段
- **When** HeatService.calculate(record) 执行
- **Then** 热度按衰减公式计算：
  - `heat = base * e^(-decay * days) * log(count+1)`
- **And** 状态转换符合四区间定义
- **And** 边界测试覆盖 SM-01~08

### Story 5.2: 混合触发策略

As a 系统，
I want 实现读取时 touch + 定时批量衰减，
So that 热度计算高效且准确。

**Acceptance Criteria:**

- **Given** 用户读取记忆
- **When** access_count += 1, last_accessed = now()
- **Then** touch 更新即时执行
- **And** 批量衰减每小时执行一次
- **And** 并发测试验证无丢失更新

### Story 5.3: 用户干预接口

As a 用户，
I want 手动标记记忆为重要/遗忘/情绪锚点，
So that 我可以主动影响记忆生命周期。

**Acceptance Criteria:**

- **Given** 记忆条目已存在
- **When** 用户调用 CLI 命令：
  - `--important`: heat = 9000 并锁定
  - `--forget`: heat = 0，标记归档候选
  - `--emotion-anchor`: heat += 2000，衰减率减半7天
- **Then** 状态立即更新
- **And** 原因写入 reason 字段

### Story 5.4: 归档候选标记（Phase 1）

As a 系统，
I want 识别低热度记忆并标记为归档候选，
So that Phase 1 只标记不执行归档。

**Acceptance Criteria:**

- **Given** 记忆热度 < 2000（打包候选）
- **When** 归档检查每日执行
- **Then** 记忆标记 is_archive_candidate = true
- **And** 不执行物理归档（Phase 1）
- **And** 用户可查询归档候选列表

---

## Epic 6 Stories

### Story 6.1: CLI 子命令实现

As a 用户，
I want 通过 CLI 子命令操作记忆库，
So that 本地开发和测试便捷。

**Acceptance Criteria:**

- **Given** CLI 入口 `laputa`
- **When** 用户执行子命令：
  - `laputa init --name "大湿"`
  - `laputa diary write --content "..." --tags "work"`
  - `laputa recall --time-range "2026-04-01~2026-04-13"`
  - `laputa wakeup`
  - `laputa mark --id <uuid> --important`
- **Then** 命令正确执行并返回结果
- **And** 错误情况返回 LaputaError

### Story 6.2: MCP Tools 扩展

As a AI agent，
I want 通过 MCP Tools 调用天空之城能力，
So that agent 可以无缝集成记忆系统。

**Acceptance Criteria:**

- **Given** MCP 服务器启动
- **When** agent 调用 Tools：
  - `laputa_init`
  - `laputa_diary_write`
  - `laputa_recall`
  - `laputa_wakeup_generate`
  - `laputa_mark_important`
  - `laputa_get_heat_status`
- **Then** Tool handler 正确处理请求
- **And** 返回 JSON-RPC 2.0 格式响应
- **And** 参数命名遵循 snake_case

---

## Epic 7 Stories

### Story 7.1: 关系节点记录

As a 系统，
I want 记录关系变化与共振度，
So that 记忆之间的共鸣可量化。

**Acceptance Criteria:**

- **Given** 关系变化事件
- **When** 写入 KnowledgeGraph
- **Then** 创建/更新关系节点：
  - 关系类型（人-人/人-项目/人-主体）
  - 共振度（-100~+100）
- **And** 时间三元组记录变化

### Story 7.2: 情绪维度记录

As a 系统，
I want 记录记忆的情绪效价和激活程度，
So that 情感维度可用于检索和排序。

**Acceptance Criteria:**

- **Given** 记忆写入
- **When** 设置情绪标记
- **Then** emotion_valence（-100~+100）
- **And** emotion_arousal（0~100）
- **And** 可用于情绪驱动检索

---

## Epic 8 Stories

### Story 8.1: 归档候选导出

As a 系统，
I want 打包导出归档候选，
So that 低热度记忆可离线保存。

**Acceptance Criteria:**

- **Given** 归档候选已标记
- **When** 用户调用 `archive.export()`
- **Then** 生成 SQLite dump 文件
- **And** 沿用 mempalace-rs 格式（D-005）
- **And** 导出路径记录到配置

### Story 8.2: 主体数据导出

As a 用户，
I want 导出主体定义和核心记忆，
So that 我可以迁移到新设备或新宿主。

**Acceptance Criteria:**

- **Given** 用户请求导出
- **When** 调用 `export.full()`
- **Then** 导出内容包含：
  - identity.md
  - relation.md
  - 近期摘要胶囊
  - 高热度记忆记录（>5000）
- **And** 导出格式为文档化结构（NFR-6）
- **And** 可在新实例中恢复

---

### Epic 9: 独立仓库化与迁移阻断修补

**Epic 目标**: 让 Laputa 脱离当前父工作区与兄弟目录依赖，成为可独立 clone、独立构建、独立测试、独立运行的 Rust 项目，为新工作服务器迁移和后续 crates.io 发布做好前置准备。

**完成价值**: 项目可从当前机器安全迁移到新服务器继续开发与运行，发布链具备真实基础。

**FR 覆盖**: NFR-1, NFR-2, NFR-6（补强），发布阻断修补

**MVP 优先级**: P0 迁移阻断修补

---

### Epic 10: MVP 验收与 agent-diva-nano 整合准备

**Epic 目标**: 完成最后一公里验收（端到端链路），修复已识别的关键缺陷（C1 数据库路径），同时为 agent-diva-nano 整合奠定架构基础。

**完成价值**: MVP 可正式发布，整合路径有清晰蓝图。

**FR 覆盖**: PRD Success Criterion #7，Deferred C1 修复

**MVP 优先级**: P0 验收阻断修补

---

## Epic 9 Stories

### Story 9.1: 构建链路去同级路径依赖

As a 维护者，
I want 移除 Laputa 对父工作区和兄弟目录的构建期依赖，
So that 项目被单独 clone 到新工作服务器后也能直接编译与测试。

**Acceptance Criteria:**

- **Given** 当前 `Laputa` 项目仍存在 `path = "../..."` 或等价的兄弟目录构建依赖
- **When** 开发者执行独立仓库化修补
- **Then** `Cargo.toml` 中不得再存在指向父工作区/兄弟目录的构建期依赖或 patch
- **And** 如果 `usearch` patch 属于必须保留的修复，则该 patch 必须以本仓可维护形式内置，或切换到可公开获取的稳定来源
- **And** 仅复制 `Laputa/` 目录到全新路径后执行 `cargo build` 必须成功
- **And** 仅复制 `Laputa/` 目录到全新路径后执行核心测试集必须成功
- **And** 不允许通过 README 或开发说明要求用户额外 clone `mempalace-rs`、`agent-diva` 或其他兄弟仓来满足构建前提

### Story 9.2: 仓库元数据与文档独立化

As a 新用户，
I want 从仓库元数据与 README 中看到 Laputa 是一个可独立运行的项目，
So that 我可以在新环境中直接理解、安装并启动它，而不是猜测缺失了哪些同级仓库。

**Acceptance Criteria:**

- **Given** 当前仓库文档仍将 `mempalace-rs` / `agent-diva` / `UPSP` / `LifeBook` 作为平级上下文直接引用
- **When** 开发者完成仓库独立化修补
- **Then** `Cargo.toml` 的 `repository`、`homepage`、`documentation` 必须指向 Laputa 自身，而不是 `mempalace-rs`
- **And** `README.md` 中的启动与安装说明只能依赖当前仓库
- **And** `STATUS.md` 应记录上游 lineage 与版本来源，而不是把 `../mempalace-rs` 当作运行时前提
- **And** `AGENTS.md` 与相关 planning / implementation 文档中凡是“复制同级目录”“依赖兄弟仓库”的表述，必须改为“历史来源”或“迁移说明”
- **And** README 必须明确独立仓库最小启动路径：`cargo build`、`cargo test`、`cargo run -- init`

### Story 9.3: 新服务器独立运行验收

As a 维护者，
I want 在不含旧工作区的干净环境中验证 Laputa 的最小运行链路，
So that 我可以确认这次修补真正解决了迁移阻断问题。

**Acceptance Criteria:**

- **Given** Story 9.1 与 9.2 已完成
- **When** 开发者将 `Laputa/` 单独放置到一个不包含旧兄弟仓库的干净目录
- **Then** `cargo build` 必须成功
- **And** `cargo test` 至少通过核心 smoke tests
- **And** `cargo run -- init` 必须成功
- **And** 必须验证一条最小 CLI 链路：`init` → `diary write` → `wakeup`
- **And** 必须产出迁移验收记录，明确环境条件、执行命令、结果和阻断原因（若失败）

---

## Epic 10 Stories

### Story 10.1: 端到端链路验收脚本

As a 产品验收者，
I want 有一条自动化脚本验证完整的 MVP 链路，
So that 我可以证明 PRD Success Criterion #7 已满足。

**Acceptance Criteria:**

- **Given** Laputa 项目已编译完成
- **When** 执行验收脚本
- **Then** 以下 CLI 链路必须全部通过：
  - `laputa init --name "测试用户"` → 成功创建 laputa.db
  - `laputa diary write --content "测试日记" --tags "test"` → 成功写入 L1 层
  - `laputa wakeup` → 返回唤醒包，token_count < 1200
  - `laputa recall --time-range "today"` → 返回今日记忆
- **And** 验收报告写入 `_bmad-output/implementation-artifacts/mvp-acceptance-report.md`
- **And** 验收报告包含：执行时间、环境信息、各步骤结果、阻断原因（若失败）

### Story 10.2: CLI/MCP 数据库路径统一

As a 开发者，
I want CLI 与 MCP 使用统一的数据库文件，
So that 数据不会因接口不同而隔离。

**Acceptance Criteria:**

- **Given** 当前 CLI 使用 `vectors.db`，MCP 使用 `diary.db`
- **When** 执行数据库路径统一修复
- **Then** CLI 和 MCP 统一使用 `laputa.db`
- **And** 所有相关代码路径（CLI handlers、MCP handlers、config）同步修改
- **And** 单元测试验证路径一致性
- **And** 修复后端到端链路仍能正常通过

### Story 10.3: Laputa → agent-diva-nano Tool 整合设计

As a 系统架构师，
I want 完成 Laputa 作为 agent-diva-nano Tool 的整合设计，
So that Agent 可以通过 Tool trait 主动调用 Laputa 记忆能力。

**Acceptance Criteria:**

- **Given** agent-diva-tools 定义了 `Tool` trait
- **When** 完成整合设计文档
- **Then** 文档包含以下内容：
  - Laputa Tool 实现方案（Tool trait 方法定义）
  - 工具参数 JSON Schema 设计
  - ToolRegistry 注册方案
  - agent-diva-nano Cargo.toml 依赖配置
  - 与现有 Agent::send 的集成点设计
- **And** 设计文档写入 `_bmad-output/implementation-artifacts/10-3-laputa-nano-tool-integration-design.md`
- **And** MCP/HTTP API 整合方案标记为"Phase 2 后续工作"

### Story 10.4: Laputa TUI Example 复制与设置

As a 开发者，
I want 将 agent-diva-nano TUI example 复制到 Laputa 项目中，
So that Laputa 项目可以展示一个完整的 TUI demo，作为参考实现。

**Acceptance Criteria:**

- **Given** agent-diva-nano TUI example 位于 `nano-workspace/agent-diva-nano/examples/tui/`
- **When** 完成 TUI 复制与设置
- **Then** Laputa 项目包含：
  - `Laputa/examples/tui/` 目录结构
  - 所有 TUI 源文件复制到位（main.rs, app.rs, ui.rs 等）
  - `Cargo.toml` example 配置正确
  - 独立运行 `cargo run --example tui` 可正常启动
- **And** 复制后的 TUI example 保持原 agent-diva-nano 依赖关系（暂不集成 Laputa）
- **And** Story 文件写入 `_bmad-output/implementation-artifacts/10-4-laputa-tui-example-setup.md`

---

## Stories Summary

| Epic | Stories 数量 | FR 覆盖 |
|------|------------|---------|
| Epic 1 | 3 | FR-1, FR-4 |
| Epic 2 | 3 | FR-2, FR-3, FR-10 |
| Epic 3 | 4 | FR-6, FR-7, FR-8 |
| Epic 4 | 2 | FR-5 |
| Epic 5 | 4 | FR-3, FR-11, FR-12 |
| Epic 6 | 2 | FR-15 |
| Epic 7 | 2 | FR-9, FR-10 |
| Epic 8 | 2 | FR-13, FR-14 |
| Epic 9 | 3 | NFR-1, NFR-2, NFR-6（补强），发布阻断修补 |
| Epic 10 | 4 | Success Criterion #7，C1修复，Tool整合，TUI example |

**总 Stories**: 29

**总 FR 覆盖**: 15/15 = **100%**

**MVP 验收状态**: Epic 10 待执行
