# Story 7.1: 关系节点记录

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a 系统，
I want 记录关系变化与共振度，
so that 记忆之间的共鸣可量化。

## Acceptance Criteria

1. **Given** 关系变化事件
   **When** 写入 `KnowledgeGraph`
   **Then** 创建/更新关系节点：
   - 关系类型（人-人 / 人-项目 / 人-主体）
   - 共振度（-100~+100）
2. **And** 时间三元组记录变化

## Tasks / Subtasks

- [x] Task 1 (AC: 1, 2) - 在现有 `KnowledgeGraph` 上定义关系语义
  - [x] 明确关系类型模型：`person_person`、`person_project`、`person_self` 或等价枚举
  - [x] 明确共振度的存储位置与读取方式
  - [x] 不把关系语义留成调用方自行拼装的松散 JSON 约定
- [x] Task 2 (AC: 1) - 建立关系写入 / 更新入口
  - [x] 为 `KnowledgeGraph` 增加 Laputa 专属 relation API
  - [x] 支持首次创建关系
  - [x] 支持已有关系的共振度更新和关系状态变更
- [x] Task 3 (AC: 2) - 记录关系时间线
  - [x] 更新关系时关闭旧记录，再插入新记录
  - [x] 保留 `valid_from` / `valid_to` 或等价有效期语义
  - [x] 保证后续 timeline 查询能回放关系演化
- [x] Task 4 (AC: 1) - 建立稳定读取接口
  - [x] 提供当前关系查询入口
  - [x] 提供关系时间线查询入口
  - [x] 返回结构化字段，避免后续 `wakeup` / MCP / CLI 自己猜字段
- [x] Task 5 (AC: 1, 2) - 补齐自动化测试
  - [x] 覆盖创建、更新、失效、timeline、边界共振度
  - [x] 覆盖非法共振度输入
  - [x] 覆盖同一当前关系不重复无限插入

## Dev Notes

### 实现目标

- 本故事的目标不是“把三元组写进图库”这么宽泛，而是让 Laputa 真正具备可消费的主体关系模型。
- 最小交付必须让后续调用方稳定获得三类信息：
  - `relation_type`
  - `resonance`
  - 当前有效关系与历史关系的区分
- 这样 Story 3.3 的 wakeup 才能可靠筛选“关键关系（共振度 > 50）”，后续 Story 7.2 也才能在此基础上补 emotion 维度。

### 当前代码现状

- `Laputa/src/knowledge_graph/mod.rs` 已有：
  - `entities` 表
  - `triples` 表
  - `add_entity()`
  - `add_triple()`
  - `invalidate()`
  - `query_entity()`
  - `stats()`
- 当前实现是通用 triple store，还没有：
  - Laputa 专属关系类型
  - 共振度字段语义
  - “当前关系”与“历史关系”稳定读接口
- `Laputa/src/mcp_server/mod.rs` 已经复用 `KnowledgeGraph` 暴露基础查询/写入/时间线能力，说明 `KnowledgeGraph` 就是本故事的正确扩展点。
- `Laputa/src/palace_graph.rs` 只处理房间和 wing 的空间拓扑，不是主体关系图。

### 关键 Guardrails

1. **只能扩展 `KnowledgeGraph`**
   - 关系节点必须落在 `Laputa/src/knowledge_graph/`
   - 不能改用 `PalaceGraph`
   - 不能新建一套平行 `relation.db` / JSON 文件 / 内存专用图

2. **`PalaceGraph` 明确不在范围内**
   - `PalaceGraph` 的职责是 room / wing 连通关系
   - 本故事处理的是人、项目、主体之间的语义关系
   - 两者混用会直接把领域模型做坏

3. **时间线必须保留**
   - Epic AC 明确要求“时间三元组记录变化”
   - 所以更新关系时不能简单覆盖当前 resonance
   - 必须保留旧状态的结束时间，再写入新状态

4. **共振度是保留的 UPSP 核心概念之一**
   - 当前故事只处理 resonance
   - 不要把 UPSP 其他复杂结构一起带入
   - `valence` / `arousal` 留给 Story 7.2

### 建模建议

- 当前 `triples` 表已有：
  - `subject`
  - `predicate`
  - `object`
  - `valid_from`
  - `valid_to`
  - `confidence`
  - `source_closet`
  - `source_file`
- 对本故事最重要的不是新增多少列，而是保证以下信息可稳定读取：
  - 关系类型
  - 共振度
  - 是否为当前有效关系
- 推荐策略：
  - 关系类型使用显式枚举或等价受控字符串
  - 共振度作为结构化属性存储，而不是散落在自由文本里
  - 读取接口负责把底层 triples 组装成稳定 relation record

### 关系类型约束

- 当前故事至少覆盖：
  - 人-人
  - 人-项目
  - 人-主体
- 推荐引入 `RelationKind` 或等价模型。
- 不要让关系类型在调用方里到处以随意字符串出现，否则后续 wakeup 很难稳定筛选“关键关系”。

### 共振度约束

- 范围以 Epic / PRD / Architecture 为准：`-100..=100`
- 必须明确统一策略：
  - clamp 到边界
  - 或直接报 `ValidationError`
- 二选一即可，但代码、测试、对外接口必须一致。

### 时间线更新建议

- 首次写入关系：
  - 确保实体存在
  - 插入当前有效关系记录
- 更新关系：
  - 找到当前有效关系
  - 关闭旧记录的 `valid_to`
  - 插入新记录，带新的 resonance 和新的 `valid_from`
- 这能同时满足：
  - 当前关系查询
  - 历史 timeline 查询
  - 后续 wakeup 取“当前关键关系”

### 读取接口建议

- 当前 `query_entity()` 返回通用 `serde_json::Value`
- 本故事应补更稳定的接口，例如：
  - `upsert_relation(...)`
  - `get_current_relations(entity)`
  - `get_relation_timeline(entity)`
- 目标不是禁止 `Value`，而是减少后续模块对底层 JSON 字段细节的硬编码依赖。

### 与其他故事的依赖关系

- Story 3.3 依赖这里提供“关键关系（共振度 > 50）”的稳定读路径
- Story 7.2 会在这个关系模型之上补 emotion 维度
- Story 8.2 导出主体数据时也会需要稳定的 relation 结构，而不是临时拼装结果

### 测试要求

- 至少覆盖：
  - 新关系创建
  - 同一关系更新
  - 旧关系失效 / 新关系生效
  - 当前关系查询
  - 关系 timeline 查询
  - resonance 边界值 `-100` / `100`
  - resonance 非法输入
- 继续沿用当前 `knowledge_graph` 的 SQLite 测试风格，不需要引入新框架。

### 代码落点建议

```text
Laputa/
├── src/
│   ├── knowledge_graph/
│   │   ├── mod.rs              # [MODIFY] 核心扩展点
│   │   ├── relation.rs         # [NEW/OPTIONAL] 关系模型与查询封装
│   │   └── resonance.rs        # [NEW/OPTIONAL] 共振度约束逻辑
│   ├── mcp_server/
│   │   └── mod.rs              # [FOLLOW-UP] 后续复用关系查询
│   └── wakeup/
│       └── mod.rs              # [FOLLOW-UP] 后续消费关键关系
└── tests/
    └── test_relation_node.rs   # [NEW] 关系与 timeline 测试
```

### 最新依赖与技术信息

- 当前仓库已具备本故事所需依赖：
  - `rusqlite = 0.32`
  - `serde_json = 1.0.149`
  - `uuid = 1`
- 本故事不需要升级依赖版本；重点是把领域关系模型落在现有图存储上。

### Project Structure Notes

- 必须遵守 `AGENTS.md` 中“扩展 mempalace-rs，而不是另起底座”的原则。
- 关系节点属于 `knowledge_graph` 领域，不属于 `palace_graph` 空间拓扑层。
- 当前 `Laputa` 不是 git 仓库根目录，本次故事以上述源码和规划文档为准。

### References

- [Source: `_bmad-output/planning-artifacts/epics.md` - Epic 7 / Story 7.1]
- [Source: `_bmad-output/planning-artifacts/prd.md` - FR-9]
- [Source: `_bmad-output/planning-artifacts/architecture.md` - ARCH-10 共振度保留]
- [Source: `_bmad-output/planning-artifacts/architecture.md` - 6.2 项目目录结构]
- [Source: `_bmad-output/planning-artifacts/architecture.md` - `knowledge_graph/resonance.rs` / `relation.rs` 规划]
- [Source: `Laputa/src/knowledge_graph/mod.rs`]
- [Source: `Laputa/src/palace_graph.rs`]
- [Source: `Laputa/src/mcp_server/mod.rs`]
- [Source: `Laputa/AGENTS.md`]

## Dev Agent Record

### Agent Model Used

GPT-5

### Debug Log References

- 2026-04-15: 为 `KnowledgeGraph` 新增 `RelationKind`、`RelationRecord`、`Resonance`，并实现 `upsert_relation()`、`get_current_relations()`、`get_relation_timeline()`。
- 2026-04-15: 更新 `top_relations()` / `relation_changes_between()` 以消费结构化关系记录。
- 2026-04-15: 新增 `Laputa/tests/test_relation_node.rs`，并调整 `test_wakepack.rs`、`test_rhythm.rs` 使用显式关系 API。
- 2026-04-15: 验证通过 `cargo test --test test_relation_node`、`cargo test --test test_wakepack`、`cargo test --test test_rhythm`、`cargo fmt --check`、`cargo test`。

### Completion Notes List

- 已将 Laputa 关系语义收敛为 `RelationKind` 枚举，限定 `person_person`、`person_project`、`person_self` 三类关系。
- 已将共振度建模为 `Resonance` 值对象，统一校验范围 `-100..=100`，并以结构化方式落在 `triples.confidence` 中读取/写入。
- 已为 `KnowledgeGraph` 增加 `upsert_relation()`、`get_current_relations()`、`get_relation_timeline()`，调用方无需自行拼装松散 JSON。
- 已实现同一关系对的时间线更新：更新时关闭旧记录 `valid_to`，再写入新记录，支持关系类型切换与共振度变更。
- 已将 wakeup 与 weekly rhythm 的关系消费路径切换到显式关系模型，保证后续 Story 3.3 / 7.2 / 8.2 的稳定读取前提。
- 已补齐自动化测试，覆盖创建、更新、时间线、边界值、非法输入和幂等去重，并通过全量 `cargo test`。

### File List

- `_bmad-output/implementation-artifacts/7-1-relation-node.md`
- `Laputa/src/knowledge_graph/mod.rs`
- `Laputa/src/knowledge_graph/relation.rs`
- `Laputa/src/knowledge_graph/resonance.rs`
- `Laputa/tests/test_relation_node.rs`
- `Laputa/tests/test_rhythm.rs`
- `Laputa/tests/test_wakepack.rs`

## Change Log

- 2026-04-15: 实现结构化关系节点模型、关系写入/时间线 API、关系测试与现有 wakeup/rhythm 适配；故事状态更新为 `review`。
- 2026-04-16: 三层代码审查完成（Blind Hunter + Edge Case Hunter + Acceptance Auditor）。

## Review Findings

### Deferred (Pre-existing)

- [x] [Review][Defer] confidence_to_resonance 边界转换存在歧义 [mod.rs:581-589] — 用户决策：方案A（添加文档注释），暂不修改代码，推迟原因：需跨模块规格文档同步
- [x] [Review][Defer] SQL动态拼接存在注入风险模式 [mod.rs:341-344] — deferred, 模式问题但非当前代码实际漏洞
- [x] [Review][Defer] N+1查询性能隐患(relation_changes_between) [mod.rs:398-488] — deferred, 性能优化项
- [x] [Review][Defer] valid_from=None时历史记录排序 [mod.rs:530] — deferred, 已有 COALESCE 处理
- [x] [Review][Defer] Resonance缺少Default trait [resonance.rs:6-26] — deferred, 非阻塞
