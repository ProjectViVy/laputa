# Story 5.3: 用户干预接口

**Story ID:** 5.3  
**Story Key:** 5-3-user-intervention  
**Status:** review  
**Created:** 2026-04-14  
**Project:** 天空之城 (Laputa)

---

## 用户故事

As a **用户**,
I want **手动将记忆标记为重要 / 遗忘 / 情绪锚点**,
So that **我可以主动影响记忆生命周期，而不是完全依赖系统自动衰减与归档判断**。

---

## 验收标准

- **Given** 目标记忆已经存在于 SQLite `memories` 表中
- **When** 用户通过 CLI 调用 `laputa mark --id <memory_id> --important`
- **Then** 目标记录的 `heat_i32` 被设置为 `9000`
- **And** 该记录立即进入“锁定/高热”区间语义
- **And** 本次人工干预原因被写入可审计的 `reason` 信息，而不是只修改热度数值

- **Given** 目标记忆已经存在于 SQLite `memories` 表中
- **When** 用户通过 CLI 调用 `laputa mark --id <memory_id> --forget`
- **Then** 目标记录的 `heat_i32` 被设置为 `0`
- **And** `is_archive_candidate` 被设置为 `true`
- **And** 本次人工干预原因被写入可审计的 `reason` 信息

- **Given** 目标记忆已经存在于 SQLite `memories` 表中
- **When** 用户通过 CLI 调用 `laputa mark --id <memory_id> --emotion-anchor --valence <n> --arousal <n>`
- **Then** 目标记录的 `heat_i32` 增加 `2000`，且结果被裁剪到上限 `10000`
- **And** `emotion_valence` / `emotion_arousal` 被更新到合法区间
- **And** 本次人工干预原因被写入可审计的 `reason` 信息

- **Given** 用户提供了不存在的 `memory_id`
- **When** 执行任一 `mark` 子命令
- **Then** 返回明确错误
- **And** 不产生任何副作用写入

- **Given** 用户误传多个互斥标记或未提供任何标记
- **When** 执行 `laputa mark`
- **Then** CLI 参数校验失败
- **And** 返回可读的使用提示

- **And** 自动化测试覆盖重要标记、遗忘标记、情绪锚点、互斥参数校验、缺失记录错误和持久化结果

扩展约束：
- 本 Story 聚焦“用户显式干预接口 + 持久化更新”，不实现完整 HeatService 衰减策略，不实现批量热度刷新，也不实现真实物理归档。
- `reason` 必须可审计。若当前主表中没有现成字段，可通过最小增量方式补充可审计元数据存储，但不要新造第二套记忆主存储。
- CLI 是本 Story 的主交付接口；MCP 对应暴露可作为后续 Story 复用该能力，不要求在本 Story 中一并完成。

---

## Epic 上下文

### Epic 5 目标

Epic 5 负责“热度机制与手动治理”，覆盖：
- `FR-3` 记忆筛选
- `FR-11` 手动治理
- `FR-12` 归档候选

`5.3 user-intervention` 的职责是把用户的显式治理动作落到真实接口与数据更新路径上。它不是热度衰减算法本身，也不是归档执行器，而是用户对现有记忆进行人工纠偏、强化、遗忘和情绪加权的入口。

### 与同 Epic 其他 Story 的关系

- `5.1 heat-service` 负责热度公式、阈值区间和状态机语义，是本 Story 的底层语义来源。
- `5.2 mixed-trigger` 负责读取时 touch 和定时批量衰减，是本 Story 的自动侧更新路径。
- `5.3 user-intervention` 负责人工覆盖和人工强化，是“自动机制”之外的人为治理入口。
- `5.4 archive-candidate` 负责低热度内容的候选标记与查询；本 Story 只在 `--forget` 场景下触发 `is_archive_candidate = true`，不实现归档检查任务本身。

---

## 现有代码情报

### 已存在且必须复用的能力

1. `Laputa/src/storage/memory.rs`
   - 已有 `LaputaMemoryRecord`
   - 已有 `heat_i32`
   - 已有 `emotion_valence` / `emotion_arousal`
   - 已有 `update_emotion(valence, arousal)` 边界裁剪逻辑
   - 已有 `is_archive_candidate`

2. `Laputa/src/vector_storage.rs`
   - 已有 `get_memory_by_id(id)`
   - 已有 SQLite `memories` 读写主路径
   - 已有 `touch_memory(id)`，说明“面向现有记忆做更新”的持久化模式已经存在
   - 是本 Story 最自然的领域落点

3. `Laputa/src/cli/mod.rs`
   - 当前仅有模块注释，尚未实现真实 CLI 子命令
   - Epic 6 会做完整 CLI 命令矩阵，但本 Story 已经要求 `laputa mark` 语义，因此这里至少需要最小可用实现或稳定入口预留

4. `Laputa/src/archiver/mod.rs`
   - 当前为占位模块
   - 说明“归档候选”相关完整逻辑尚未落地
   - 本 Story 不应等待 Archiver 完整实现，只需要把 `forget` 的结果映射到 `is_archive_candidate = true`

5. `Laputa/src/main.rs`
   - 当前仅输出版本号，说明 CLI 实际入口尚未接线
   - 本 Story 需要把 `mark` 命令从“文档语义”推进到可调用入口

6. `Laputa/config/laputa.toml`
   - 已有 `[heat]`、` [archive]`、`[wakeup]` 配置
   - 热度阈值当前为：
     - `hot_threshold = 8000`
     - `warm_threshold = 5000`
     - `cold_threshold = 2000`
   - 这些配置提供了“important / forget / emotion-anchor”行为落点的系统语义背景

### 从前序 Story 继承的结论

来自 `1-3-memoryrecord-extension`：
- 热度字段已经固定为 `heat_i32`，禁止新造 `heat: f64` 持久化列。
- 情绪字段已经进入主模型，后续 Story 必须复用现有 schema。

来自 `2-3-emotion-anchor`：
- 已经明确“情绪锚点”的最合理落点是 `VectorStorage` 级别的现有记录更新接口。
- 应复用 `LaputaMemoryRecord::update_emotion()`，而不是重复写边界裁剪逻辑。
- 对缺失 `memory_id` 必须返回明确错误，不能静默成功。

因此，`5.3` 不应重新设计一套独立的用户治理数据路径，而应在同一套 `VectorStorage + CLI` 路径上扩展“重要 / 遗忘 / 情绪锚点”三类干预。

---

## 架构与实现约束

### 1. 职责边界

本 Story 负责：
- 定义并实现用户可调用的 `mark` 干预入口
- 将用户意图映射为现有记录的热度/情绪/归档候选更新
- 写入可审计原因信息
- 处理互斥参数和无效输入

本 Story 不负责：
- 完整 HeatService 公式计算
- 定时衰减/批量更新
- 自动归档检查任务
- MCP tool 暴露
- 图形界面或更复杂治理面板

### 2. 数据语义约束

- `--important`
  - `heat_i32 = 9000`
  - 不要求新增“永不衰减”实现，但必须保证它立即落入高热/锁定语义区间

- `--forget`
  - `heat_i32 = 0`
  - `is_archive_candidate = true`
  - 不做物理归档删除

- `--emotion-anchor`
  - `heat_i32 = min(current + 2000, 10000)`
  - 复用 `update_emotion(valence, arousal)` 做边界裁剪

### 3. 可审计 reason 约束

- Epic 5 和 PRD 都要求人工治理具备可解释性。
- 当前 `memories` 主表没有现成 `reason` 字段。
- 开发时必须选择一种最小增量、可持续复用的方案保存人工干预原因，例如：
  - 为 `memories` 表补充 `reason` / `last_reason` 字段；或
  - 新增轻量治理日志表，按 `memory_id` 记录动作、原因、时间戳。
- 不允许把 `reason` 只打印到 stdout 或只留在测试里。
- 不允许创建脱离主存储语义的第二套记忆库。

推荐方向：
- 若只需满足当前 Story，优先使用轻量治理日志表，例如 `memory_interventions`，因为它能同时保留动作历史，避免单字段覆盖丢失审计轨迹。
- 若实现成本受限，也可以先落地 `last_reason` + `last_intervention_at` 的主表增量方案，但要保证后续 Story 不会因为该选择被锁死。

### 4. CLI 约束

- `laputa mark` 至少应支持：
  - `--id <memory_id>`
  - `--important`
  - `--forget`
  - `--emotion-anchor`
  - `--valence <i32>`
  - `--arousal <u32>`
  - `--reason <text>` 或等价参数
- `--important` / `--forget` / `--emotion-anchor` 必须互斥。
- `--emotion-anchor` 需要 `valence` 和 `arousal`；其他模式不需要。
- 参数校验失败时必须在 CLI 层尽早失败，避免进入持久化逻辑。

### 5. 代码组织约束

优先采用以下落点：
- `Laputa/src/cli/mod.rs`
  - 定义 `mark` 子命令和参数校验
- `Laputa/src/vector_storage.rs`
  - 落地对现有记忆的持久化更新接口
- `Laputa/src/storage/memory.rs`
  - 如需补充纯模型辅助方法，可放在这里；但不要把数据库逻辑塞进来
- `Laputa/src/main.rs`
  - 把 CLI 真正接线起来，至少让 `mark` 能运行

不要：
- 在 CLI 层直接拼 SQL
- 把治理逻辑埋进 `main.rs`
- 新建脱离 `VectorStorage` 的独立存储通路
- 顺手把 MCP/Archiver/HeatService 全部做完

---

## 推荐实现方案

### 推荐领域接口

优先在 `Laputa/src/vector_storage.rs` 中新增一组显式治理方法，例如：

```rust
pub fn mark_important(&self, memory_id: i64, reason: &str) -> Result<MemoryRecord>
pub fn mark_forget(&self, memory_id: i64, reason: &str) -> Result<MemoryRecord>
pub fn mark_emotion_anchor(
    &self,
    memory_id: i64,
    valence: i32,
    arousal: u32,
    reason: &str,
) -> Result<MemoryRecord>
```

如需避免接口过散，也可定义统一命令对象：

```rust
pub enum UserIntervention {
    Important { reason: String },
    Forget { reason: String },
    EmotionAnchor { valence: i32, arousal: u32, reason: String },
}

pub fn apply_intervention(&self, memory_id: i64, intervention: UserIntervention) -> Result<MemoryRecord>
```

统一命令对象更利于后续 MCP 复用，也更符合架构里统一抽象的方向。

### 推荐流程

1. CLI 解析并校验参数
2. 构造领域命令（important / forget / emotion-anchor）
3. `VectorStorage` 读取目标记录
4. 计算新状态：
   - important → `heat_i32 = 9000`
   - forget → `heat_i32 = 0`, `is_archive_candidate = true`
   - emotion-anchor → `heat_i32 += 2000`，并更新情绪维度
5. 将变更和 `reason` 持久化到主数据路径
6. 重新读取并返回更新后的记录或输出摘要

### 推荐输出行为

CLI 成功时至少返回：
- `memory_id`
- 执行动作
- 更新后的 `heat_i32`
- 是否被标记为 archive candidate
- 如适用，更新后的 `emotion_valence` / `emotion_arousal`

这样既便于用户确认，也便于后续脚本调用。

---

## 测试要求

至少补齐以下测试：

1. `important` 成功路径
   - 已存在记录被标记为重要
   - `heat_i32 == 9000`
   - 审计 reason 被持久化

2. `forget` 成功路径
   - `heat_i32 == 0`
   - `is_archive_candidate == true`
   - 审计 reason 被持久化

3. `emotion-anchor` 成功路径
   - `heat_i32` 增加 `2000`
   - `valence/arousal` 正确裁剪
   - reason 被持久化

4. 上限裁剪
   - 原始高热记录执行 `emotion-anchor` 后不超过 `10000`

5. 缺失记录
   - 任一干预对不存在 `memory_id` 返回明确错误

6. CLI 参数校验
   - 多个互斥标记同时出现时失败
   - 未提供任何标记时失败
   - `emotion-anchor` 缺失 `valence`/`arousal` 时失败

7. 持久化回读
   - 更新后重新读取记录与审计信息，确认数据一致

推荐测试文件：
- `Laputa/tests/test_user_intervention.rs`

如 CLI 测试拆分更清晰，也可增加：
- `Laputa/tests/test_cli_mark.rs`

测试实现应遵循架构文档既有标准：
- 使用 `serial_test` 做文件/数据库隔离
- 复用 `tests/fixtures/with_tempdir.rs` 或其他现有 fixture
- 保持纯 Rust、本地可运行，不引入外部服务依赖

---

## 禁止事项

- 不要在本 Story 中实现完整 HeatService
- 不要实现定时批量衰减
- 不要实现自动归档任务
- 不要把 MCP tool 一起耦合进来
- 不要新造独立的“人工治理数据库”
- 不要把 `reason` 只输出到日志而不持久化
- 不要绕过现有 `VectorStorage` 主存储路径

---

## 实施任务

- [x] 在 `Laputa/src/cli/mod.rs` 中定义 `mark` 子命令和互斥参数
- [x] 在 `Laputa/src/main.rs` 中接入真实 CLI 入口
- [x] 在 `Laputa/src/vector_storage.rs` 中实现用户干预持久化接口
- [x] 复用 `Laputa/src/storage/memory.rs` 中现有热度/情绪模型语义
- [x] 为 `reason` 增加最小可审计持久化方案
- [x] 实现 `important` / `forget` / `emotion-anchor` 三种干预路径
- [x] 为缺失记录与无效参数返回明确错误
- [x] 补齐集成测试与 CLI 参数测试
- [x] 运行 `cargo test`
- [x] 运行 `cargo clippy --all-features --tests -- -D warnings`

---

## 完成定义

- [x] 用户可以通过 `laputa mark` 对单条记忆执行人工治理
- [x] `important`、`forget`、`emotion-anchor` 三种行为都能正确持久化
- [x] `reason` 具备可审计持久化结果
- [x] 错误输入不会产生副作用
- [x] 不引入第二套主存储路径
- [x] 自动化测试通过
- [x] `cargo clippy --all-features --tests -- -D warnings` 通过

---

## 参考资料

- `D:\VIVYCORE\newmemory\_bmad-output\planning-artifacts\epics.md`
- `D:\VIVYCORE\newmemory\_bmad-output\planning-artifacts\architecture.md`
- `D:\VIVYCORE\newmemory\_bmad-output\planning-artifacts\prd.md`
- `D:\VIVYCORE\newmemory\Laputa\AGENTS.md`
- `D:\VIVYCORE\newmemory\Laputa\config\laputa.toml`
- `D:\VIVYCORE\newmemory\Laputa\src\storage\memory.rs`
- `D:\VIVYCORE\newmemory\Laputa\src\vector_storage.rs`
- `D:\VIVYCORE\newmemory\Laputa\src\cli\mod.rs`
- `D:\VIVYCORE\newmemory\Laputa\src\main.rs`
- `D:\VIVYCORE\newmemory\Laputa\src\archiver\mod.rs`
- `D:\VIVYCORE\newmemory\_bmad-output\implementation-artifacts\1-3-memoryrecord-extension.md`
- `D:\VIVYCORE\newmemory\_bmad-output\implementation-artifacts\2-3-emotion-anchor.md`

---

## Dev Agent Record

### Context Notes

- 当前仓库已经具备热度、情绪字段和按 `memory_id` 回读现有记录的能力，但“用户干预入口”尚未落地。
- `5.3` 的关键不是重新设计热度机制，而是把人工治理语义压到真实 CLI 和持久化更新路径上。
- 由于当前 `memories` 主表没有现成 `reason` 字段，本 Story 的一个真实设计点是选择最小可持续的审计落点；实现时必须把这部分做成可复用结构，而不是临时输出。
- 当前工作区不是 git repository，本 Story 无法提供近期提交模式参考。

### Completion Note

已完成 `laputa mark` 最小 CLI 入口，支持 `--important`、`--forget`、`--emotion-anchor` 三种互斥干预，并通过 `--reason` 将人工治理原因持久化到主表 `memories.reason`。
- 在 `VectorStorage` 中新增统一 `UserIntervention` / `apply_intervention` 路径，并补齐 `important` / `forget` / 带审计原因的 `emotion-anchor` 更新逻辑。
- 在 `main.rs` 中接入真实命令行执行，默认复用 `MempalaceConfig` 的 `config_dir/vectors.db` 与 `vectors.usearch` 主路径，不引入第二套存储。
- 新增 `test_user_intervention.rs` 与 `test_cli_mark.rs` 覆盖三类干预、缺失记录、互斥参数、缺少 action、缺少情绪参数等场景。
- 为满足 Story 完成门槛，补了测试夹具的 `dead_code` 标注，并修复 `test_heat.rs` 的 clippy 告警。
- 验证通过：`cargo test`、`cargo clippy --all-features --tests -- -D warnings`。

## File List

- `Laputa/src/cli/mod.rs`
- `Laputa/src/main.rs`
- `Laputa/src/vector_storage.rs`
- `Laputa/tests/test_user_intervention.rs`
- `Laputa/tests/test_cli_mark.rs`
- `Laputa/tests/fixtures/memory_only.rs`
- `Laputa/tests/fixtures/with_tempdir.rs`
- `Laputa/tests/test_heat.rs`

## Change Log

- 2026-04-15: 实现 `laputa mark` CLI 与用户干预持久化路径，新增集成测试与 CLI 参数校验测试，并修复 clippy 阻塞项。
