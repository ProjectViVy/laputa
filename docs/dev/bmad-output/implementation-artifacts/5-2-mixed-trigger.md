# Story 5.2: 混合触发策略

Status: review

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a 系统，
I want 实现读取时 `touch` + 定时批量衰减，
so that 热度计算高效且准确。

## Acceptance Criteria

1. **Given** 用户读取记忆
   **When** `access_count += 1` 且 `last_accessed = now()`
   **Then** touch 更新即时执行
2. **And Given** 热度机制已启用
   **When** 后台衰减任务运行
   **Then** 批量衰减每小时执行一次
3. **And Given** 读取与后台衰减可能并发发生
   **When** 同一条记忆被并发访问和更新
   **Then** 并发测试验证无丢失更新

## Tasks / Subtasks

- [x] Task 1 (AC: 1) - 补齐 Heat 模块的最小可运行骨架
  - [x] 在 `Laputa/src/heat/` 下新增 `service.rs`、`config.rs`、`decay.rs`、`state.rs`
  - [x] 在 `Laputa/src/heat/mod.rs` 导出 `HeatService`、`HeatConfig`、`HeatState`
  - [x] 将配置字段与 `Laputa/config/laputa.toml` 的 `[heat]` 段对齐：`enabled`、`hot_threshold`、`warm_threshold`、`cold_threshold`、`decay_rate`、`update_interval_hours`
- [x] Task 2 (AC: 1) - 把读取时 touch 接入真实读取路径
  - [x] 复用现有 `Laputa/src/vector_storage.rs::touch_memory(id)`，不要重写一套访问计数逻辑
  - [x] 在会返回 `MemoryRecord` 的读取路径接入 touch：优先检查 `search()`、`search_room()`、`get_memory_by_id()`、`get_memories()` 以及上层 recall/search/wakeup 调用链
  - [x] 明确规则：只在“真正读取成功并向上返回结果”后做 touch；空结果、不存在记录、失败路径不更新
- [x] Task 3 (AC: 2) - 实现后台批量衰减流程
  - [x] 由 `HeatService` 基于 `heat = base * e^(-decay * days) * log(count+1)` 重新计算热度
  - [x] 为批量衰减增加“候选筛选”查询，只处理需要衰减的记录，避免每次全表重算
  - [x] 将批量任务周期与 `update_interval_hours = 1` 对齐，并为后续调度器或 CLI/MCP 手动触发预留调用入口
  - [x] 衰减后同步更新四区间状态判断，为 Story 5.4 的归档候选标记保留兼容接口
- [x] Task 4 (AC: 2-3) - 处理并发与写入一致性
  - [x] 确保批量衰减不会覆盖并发 touch 刚写入的 `access_count` / `last_accessed`
  - [x] 优先使用 SQL 原子更新、事务或 compare-and-update 风格，避免“先读后写”导致丢失更新
  - [x] 定义并记录冲突策略：touch 永远保留最新 `last_accessed`，衰减任务不得把访问时间回写为旧值
- [x] Task 5 (AC: 1-3) - 增加自动化测试
  - [x] 新建 `Laputa/tests/test_heat.rs`
  - [x] 为 touch 行为增加单元/集成测试：成功读取后 `access_count` 递增、`last_accessed` 更新
  - [x] 为批量衰减增加 TimeMachine 驱动测试：模拟时间推进后热度下降、边界跨越符合状态机定义
  - [x] 为并发场景增加串行隔离测试：多线程/多任务 touch 与批量衰减并发执行后，无丢失更新

## Dev Notes

### Epic Context

- Epic 5 的目标不是单纯“算出一个热度值”，而是建立生命周期治理链路：`heat_i32 + HeatService + 状态机 + 归档候选标记`。
- Story 5.2 是 Epic 5 的执行中枢：Story 5.1 定义计算核心，Story 5.2 负责把热度真正挂到读取路径和后台任务上，Story 5.3/5.4 再消费这些状态。
- 当前仓库里 `5-1-heat-service.md` 尚未创建，但代码基座已经具备实现 5.2 所需的关键前提：`heat_i32` 字段、schema、`touch_memory()`、`TimeMachine` fixture、`[heat]` 配置段。

### 现有代码与复用点

- `Laputa/src/vector_storage.rs`
  - 已有 `touch_memory(id)`，会执行 `access_count = access_count + 1, last_accessed = ?` 的原子 SQL 更新。
  - `search()`、`search_room()`、`get_memories()`、`get_memory_by_id()` 都会返回带 `heat_i32 / last_accessed / access_count` 的 `MemoryRecord`。
  - 当前 `row_to_memory_record()` 已经把 `importance_score + last_accessed + access_count` 组合成 `importance`，说明读取路径已经携带“访问衰减”语义，Story 5.2 必须避免和这套逻辑冲突。
- `Laputa/src/storage/memory.rs`
  - 已定义 `LaputaMemoryRecord` 与 `heat_from_i32()` / `heat_to_i32()`。
  - 已建立 `idx_heat` 索引，适合批量衰减后查询高热度/低热度记录。
- `Laputa/config/laputa.toml`
  - 已存在 `[heat]` 配置，可直接作为 `HeatConfig` 的运行时来源。
- `Laputa/tests/fixtures/time_machine.rs`
  - 已有时间推进工具，适合做衰减边界和批量任务测试。

### 架构约束

1. **热度存储格式必须继续使用 `i32` 放大 100 倍**
   - 不要把持久化字段改回 `f32/f64`
   - 计算阶段可用浮点，落库前必须转换回 `i32`

2. **混合触发策略是 ADR 级决策，不是实现偏好**
   - 读取时只做轻量 touch
   - 后台任务负责批量重算热度
   - 不要在每次读取时执行全量热度重算，否则违背“高效且准确”的目标

3. **状态机边界必须与架构文档一致**
   - `> 8000`: 锁定
   - `5000-8000`: 正常
   - `2000-5000`: 归档候选
   - `< 2000`: 打包候选

4. **Phase 1 只标记，不执行真实归档**
   - Story 5.2 可以暴露“是否应进入候选态”的判定
   - 不要在本故事中实现打包、迁移或物理归档

5. **不要绕开现有模块边界**
   - 热度逻辑放在 `src/heat/`
   - 存储层只负责持久化与调用，不把衰减公式散落到 `vector_storage.rs`、`storage/mod.rs`、`diary/mod.rs`

### 实现建议

- 建议在 `HeatService` 中明确拆分三个能力：
  - `touch(id)` 或 `touch_record(...)`：封装读取后访问更新
  - `calculate(record)`：单条重算
  - `calculate_batch(...)` / `run_decay_pass(...)`：批量衰减
- 如果需要在 `VectorStorage` 中执行批量衰减，优先新增一个明确的方法，如：
  - `list_decay_candidates(...)`
  - `update_heat_fields(...)`
  - `run_heat_decay_batch(...)`
- touch 接入点以“对用户可见的读取”为准，避免内部辅助查询、索引修复、后台扫描也增加访问次数。

### 并发与一致性护栏

- 当前 `touch_memory()` 已经用 SQL 原子递增 `access_count`，这是正确方向，批量衰减应复用同一数据库连接模型或事务策略。
- 批量任务若采用“先查全部记录 -> Rust 重算 -> UPDATE 全字段回写”，有很高概率覆盖并发 touch 的最新时间戳与计数；必须规避。
- 推荐至少满足以下一致性规则：
  - `access_count` 只能单调递增，衰减任务不能减少它
  - `last_accessed` 只能前进，不能被旧时间覆盖
  - `heat_i32` 更新应基于最新可见的访问数据

### 测试要求

- 测试文件放在 `Laputa/tests/test_heat.rs`
- 使用 `serial_test` 隔离 SQLite/时间流相关测试
- 至少覆盖：
  - touch 成功路径
  - 空结果/失败路径不 touch
  - 每小时批量衰减触发
  - `8000 -> 7999`、`5000 -> 4999`、`2000 -> 1999` 边界穿越
  - touch 与 decay 并发执行时无丢失更新

### 版本与最新技术核验

- 当前仓库依赖与近期官方文档仍兼容：
  - `tokio = 1.51.0`，docs.rs 最近可见版本为 `1.51.1`
  - `rusqlite = 0.32`，docs.rs 最近可见版本为 `0.38.0`
  - `usearch = 2`，docs.rs 最近可见版本为 `2.24.0`
  - `chrono = 0.4.44`，docs.rs 当前为 `0.4.44`
  - `serial_test = 2`，docs.rs 可见 `2.0.0`
- 结论：本故事不要求升级依赖；优先在现有锁定版本上完成实现，避免把功能故事扩张为依赖升级故事。

### Project Structure Notes

- 目标代码位置以当前仓库真实结构为准，而不是架构文档中的理想结构：
  - 已存在：`Laputa/src/heat/mod.rs`
  - 已存在：`Laputa/src/vector_storage.rs`
  - 已存在：`Laputa/src/storage/memory.rs`
  - 已存在：`Laputa/tests/fixtures/time_machine.rs`
  - 待新增：`Laputa/src/heat/service.rs`
  - 待新增：`Laputa/src/heat/config.rs`
  - 待新增：`Laputa/src/heat/decay.rs`
  - 待新增：`Laputa/src/heat/state.rs`
  - 待新增：`Laputa/tests/test_heat.rs`
- `Laputa/src/config.rs` 目前仍是 mempalace 风格配置代码；本故事若要读取 `config/laputa.toml`，需要谨慎新增 Laputa 专属配置入口，不要破坏现有 mempalace 兼容行为。

### Dependencies / Sequencing

- 强依赖：
  - Story 1.3 已完成，提供 `heat_i32` / `last_accessed` / `access_count` / `is_archive_candidate`
  - Story 5.1 尚未创建文档，但其核心公式与状态机规则已在架构文档中明确，本故事可按既定 ADR 落地
- 后续受益：
  - Story 5.3 会复用 `HeatService` 的状态变更入口
  - Story 5.4 会复用低热度候选判定与批量任务基础设施

### References

- `_bmad-output/planning-artifacts/epics.md`
  - Epic 5 目标
  - Story 5.2 验收标准
- `_bmad-output/planning-artifacts/prd.md`
  - FR-11 手动治理
  - FR-12 归档候选
  - NFR-2 离线可用性
  - NFR-10 可测试性
- `_bmad-output/planning-artifacts/architecture.md`
  - Step 3.1 热度存储格式
  - Step 3.2 热度计算触发策略
  - Step 3.3 热度模块架构
  - Step 3.4 四区间状态机
  - Step 3.7 热度测试策略
  - Step 6.2/6.4 目录结构与测试位置
- `Laputa/src/vector_storage.rs`
  - `touch_memory()`
  - `search()` / `search_room()` / `get_memories()` / `get_memory_by_id()`
- `Laputa/src/storage/memory.rs`
  - `LaputaMemoryRecord`
  - schema 与 `idx_heat`
- `Laputa/config/laputa.toml`
  - `[heat]`、`[archive]`
- `Laputa/tests/fixtures/time_machine.rs`
  - 时间模拟夹具
- docs.rs
  - Tokio: https://docs.rs/crate/tokio
  - rusqlite: https://docs.rs/rusqlite/
  - usearch: https://docs.rs/usearch
  - chrono: https://docs.rs/crate/chrono/latest
  - serial_test: https://docs.rs/crate/serial_test/2.0.0/source/

## Dev Agent Record

### Agent Model Used

GPT-5 Codex

### Debug Log References

- create-story workflow
- 2026-04-15: verified Story 5.2 context, existing `HeatService`, and current read-path touch behavior before implementation.
- 2026-04-15: added user-visible touch integration in `Searcher` timeline/semantic reads and `WakePackGenerator` recent-state loading.
- 2026-04-15: added `VectorStorage` decay candidate query, compare-and-update heat persistence, and hourly decay pass orchestration.
- 2026-04-15: ran `cargo fmt --all`, targeted tests (`test_heat`, `test_timeline_recall`, `test_wakepack`), and full `cargo test`.

### Completion Notes List

- 已完成 Story 5.2 的混合触发落地：保留读取侧轻量 `touch`，把热度重算放到独立的批量衰减流程。
- 已在 `Searcher` 的时间召回与语义检索路径，以及 `WakePackGenerator` 的 recent-state 加载路径上接入成功读取后的 `touch`，避免内部辅助查询误增访问计数。
- 已在 `VectorStorage` 中新增衰减候选筛选、compare-and-update 写回与 `run_heat_decay_pass(_at)` 入口；并发冲突时以最新 `touch` 的 `last_accessed/access_count` 为准，不回写旧值。
- 已补充 Story 5.2 所需测试覆盖：读取触摸、批量衰减、归档候选标记，以及 compare-and-update 并发护栏；`cargo test` 全量通过。

### File List

- `_bmad-output/implementation-artifacts/5-2-mixed-trigger.md`
- `Laputa/src/searcher/mod.rs`
- `Laputa/src/vector_storage.rs`
- `Laputa/src/wakeup/mod.rs`
- `Laputa/tests/test_heat.rs`
- `Laputa/tests/test_timeline_recall.rs`
- `Laputa/tests/test_wakepack.rs`

## Change Log

- 2026-04-15: implemented mixed-trigger heat updates, added batch decay and concurrency guards, extended read-path touch integration, and expanded regression coverage; story moved to review.
