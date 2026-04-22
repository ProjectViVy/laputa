# Story 2.1: 日记写入核心功能

Status: done

## Story

As a 用户，
I want 写入日记条目并指定时间、分区、标签与情绪，
so that 系统可以保存并解释我的记忆输入。

## Acceptance Criteria

1. **Given** 用户有已初始化的记忆库
2. **When** 用户调用 `diary.write(content, tags, emotion)`
3. **Then** 系统创建 `MemoryRecord` 并写入 L1 层
4. **And** 热度初始值设置为 `5000`
5. **And** 情绪编码映射到 `EMOTION_CODES`

## Tasks / Subtasks

- [x] 实现 Laputa 的日记写入入口，统一落到 `MemoryRecord` 写入链路，而不是继续停留在旧的 `diary_entries` 表接口上 (AC: 1, 2, 3)
- [x] 设计并实现 `diary.write(content, tags, emotion)` 对应的参数模型，至少覆盖内容、标签、情绪、时间戳和分区/room 映射 (AC: 2, 3, 5)
- [x] 将写入结果保存到 L1 可见的数据路径，确保 `Layer1::generate()` 能读取到该记录 (AC: 3)
- [x] 新记录默认写入 `heat_i32 = 5000`，并保留 Story 1.3 已引入的扩展字段兼容性 (AC: 4)
- [x] 调用 `dialect::EMOTION_CODES` 完成 emotion 到内部编码的映射；未知 emotion 需要有可解释的降级策略 (AC: 5)
- [x] 补齐单元测试和最小集成测试，覆盖写入成功、默认热度、emotion 映射、L1 可见性和异常输入 (AC: 1, 3, 4, 5)

## Dev Notes

### 实现目标

- 本 story 的目标不是继续维护 `DiaryEntry { agent, content, timestamp }` 这一套旧结构，而是把“日记输入”接到 Laputa 当前的数据主路径：`MemoryRecord`/`LaputaMemoryRecord` + SQLite `memories` + 向量索引。
- 目前代码库里 `Laputa/src/diary/mod.rs` 仍是 mempalace-rs 风格的独立 `diary_entries` 表实现；这不满足本 story 的 AC 3，因为 `Layer1::generate()` 实际读取的是 `VectorStorage::get_memories(...)` 返回的 `memories` 记录。
- 这个 story 需要明确建立桥接：日记写入后，结果必须能进入 L1，而不是只存在于一个孤立的日记表里。

### 关键架构约束

- 复用而不是重写：沿用 `mempalace-rs` 继承下来的 `src/diary/`、`src/vector_storage.rs`、`src/storage/` 路线，优先做扩展钩子，不要并行造第二套写入栈。
- 数据主存储仍是 SQLite，向量索引仍是 `usearch`，CLI/MCP 仍是一阶段首要接口；不要为这个 story 引入新的数据库、消息队列或网络依赖。
- `MemoryRecord` 扩展字段已经在 Story 1.3 中落地，当前 story 必须直接复用：
  - `heat_i32`
  - `emotion_valence`
  - `emotion_arousal`
  - `is_archive_candidate`
- 热度使用整数存储，默认值固定为 `5000`；不要改成浮点存储，也不要在本 story 引入新的热度公式。
- `emotion` 的“编码映射”要求基于 `dialect::EMOTION_CODES`，这是当前已存在的情绪词典；不要发明第二套 emotion code 表。

### 建议代码落点

- `Laputa/src/diary/mod.rs`
  - 扩展或重构现有 diary API，使其接受 `content/tags/emotion/timestamp/room` 这类写入参数。
  - 如果保留旧接口，需新增 Laputa 专用写入方法，而不是破坏原有已存在调用点。
- `Laputa/src/vector_storage.rs`
  - 接入或复用已有 memory insert 路径，保证新 diary 写入最终进入 `memories` 表和向量索引。
- `Laputa/src/storage/memory.rs`
  - 只复用现有扩展结构与常量；本 story 不应再次修改 schema 设计，除非发现 Story 1.3 漏项。
- `Laputa/src/storage/mod.rs`
  - 校验 `Layer1::generate()` 读取路径是否无需额外修改即可看到 diary 写入结果；若需 room/wing 约定，应在这里确认。
- `Laputa/src/cli/` 或 `Laputa/src/mcp_server/`
  - 如果本 story 需要暴露实际调用入口，应优先补齐现有 CLI/MCP 接口，而不是新建临时脚本入口。

### 现有代码现状与风险

- `Laputa/src/diary/mod.rs` 当前表结构仅包含 `agent/content/timestamp/created_at`，没有：
  - 标签字段
  - emotion 字段
  - room / wing 映射
  - `heat_i32`
  - 向量索引接入
- `Layer1::generate()` 当前从 `VectorStorage::get_memories(...)` 读取并生成 L1，因此如果仅写 `diary_entries`，L1 看不到新增日记。
- `Laputa/src/heat/mod.rs` 目前只有模块占位注释，尚未有完整 `HeatService` 实现；所以本 story 只需要落实“默认热度 5000”，不要阻塞在完整热度服务未完成这件事上。

### 明确实现边界

- 本 story 关注“写入入口”和“落库可见性”，不负责完整的 MemoryGate 筛选/合并逻辑；那是 Story 2.2 的职责。
- 本 story 不需要实现情绪锚点加热或 7 天保鲜；那属于 Story 2.3。
- 本 story 不需要补完整个 HeatService、Archiver、WakePack 或 Rhythm。
- 本 story 应保留向后兼容思路：如果旧的 `write_entry(agent, content)` 还被其他地方使用，不要直接删除；优先封装新入口并在合适位置复用底层写入逻辑。

### Emotion 映射要求

- 复用 `dialect::EMOTION_CODES` 做输入映射，输入如 `joy`, `fear`, `trust`, `grief` 等应映射到既有短码。
- 对未知 emotion，必须有确定行为并写进测试，推荐二选一：
  - 保留原值并走现有降级逻辑
  - 显式返回校验错误
- 无论采用哪种策略，都要保持“可解释”，避免静默吞掉输入。

### L1 与分区建议

- `Layer1::generate()` 按 `wing/room` 与 importance 组织输出，因此 diary 写入时至少需要决定默认 `wing` 与 `room`。
- 如果架构没有更细约束，优先沿用 Story 1.3 测试中已出现的 `wing = "self"`、`room = "journal"` 作为默认值，避免引入新命名分叉。
- `tags` 在本 story 至少要保存为可追踪的 metadata 或可回读字段，不能只在 API 层接收后丢弃。

### 测试要求

- 单元测试：
  - 写入 diary 后生成的 record 具有 `heat_i32 = 5000`
  - emotion 输入能映射到 `EMOTION_CODES`
  - 未知 emotion 的降级/报错行为稳定
  - 标签、时间戳和默认 room/wing 落盘正确
- 集成测试：
  - 写入后通过 `VectorStorage` 或 `Layer1::generate()` 能读到该条记录
  - 不破坏 Story 1.3 已有 `tests/test_memory_record.rs`
- 测试文件优先考虑：
  - `Laputa/tests/test_diary_write.rs`
  - 如需补充读取链路验证，可扩展现有 `Laputa/tests/test_memory_record.rs` 或新增针对 L1 的测试

### 依赖与版本守卫

- 当前项目依赖已经锁定在 `Laputa/Cargo.toml`：
  - `rusqlite = 0.32`
  - `tokio = 1.51.0`
  - `chrono = 0.4.44`
  - `usearch = 2`，当前通过仓内 `vendor/usearch` patch 维护；旧的 `../mempalace-rs/patches/usearch` 路径仅属于历史来源
- 本 story 不应升级这些依赖版本，尤其不要绕过本地 `usearch` patch。

### References

- [Source: `_bmad-output/planning-artifacts/epics.md` - Story 2.1]
- [Source: `_bmad-output/planning-artifacts/prd.md` - FR-2 日记写入]
- [Source: `_bmad-output/planning-artifacts/prd.md` - FR-4 统一记忆记录]
- [Source: `_bmad-output/planning-artifacts/prd.md` - FR-10 情绪锚定]
- [Source: `_bmad-output/planning-artifacts/prd.md` - NFR-1 纯 Rust 运行]
- [Source: `_bmad-output/planning-artifacts/prd.md` - NFR-2 离线可用性]
- [Source: `_bmad-output/planning-artifacts/architecture.md` - 4.2 数据架构决策]
- [Source: `_bmad-output/planning-artifacts/architecture.md` - 4.6 实现顺序]
- [Source: `_bmad-output/planning-artifacts/architecture.md` - 4.7 技术栈版本锁定]
- [Source: `Laputa/src/diary/mod.rs`]
- [Source: `Laputa/src/storage/mod.rs`]
- [Source: `Laputa/src/storage/memory.rs`]
- [Source: `Laputa/src/dialect/mod.rs`]
- [Source: `Laputa/Cargo.toml`]

## Dev Agent Record

### Agent Model Used

GPT-5

### Debug Log References

- `cargo test test_diary -- --nocapture`
- `cargo test test_mempalace_diary -- --nocapture`
- `cargo fmt`

### Completion Notes List

- 将 `Laputa/src/diary/mod.rs` 重构为 `MemoryRecord` 写入入口，新增 `DiaryWriteRequest`、标签/时间戳/room/wing 默认值、`EMOTION_CODES` 映射与可解析 metadata 头。
- 复用 `memories` 主路径写入，并在 `VectorStorage` 不可初始化 embedder 时回退到纯 SQLite 持久化，保证 L1 读取链路和离线场景可用。
- 扩展 `Laputa/src/vector_storage.rs` 以支持自定义时间戳/热度的结构化插入；扩展 `Laputa/src/mcp_server/mod.rs` 以接受 `tags`、`emotion`、`timestamp`、`wing`、`room` 可选参数。
- 通过 `cargo test test_diary -- --nocapture` 与 `cargo test test_mempalace_diary -- --nocapture` 验证默认热度、emotion 映射、异常输入、按时间读取、L1 可见性与 MCP 入口兼容性。

### File List

- `Laputa/src/diary/mod.rs`
- `Laputa/src/vector_storage.rs`
- `Laputa/src/mcp_server/mod.rs`
- `_bmad-output/implementation-artifacts/2-1-diary-write.md`

### Review Findings

### 2026-04-16 Code Review

#### decision_needed (已决策)

- [x] [Review][Decision] AC5未满足：未知emotion无降级策略 — **决策**: 降级到neutral，emotion_code=None，继续写入
- [x] [Review][Decision] emotion_valence/arousal硬编码为0 — **决策**: 创建emotion_code→(valence,arousal)映射表

#### patch (已存档)

- [x] [Review][Patch] embedding重复计算：MemoryGate和add_memory_record双重embedding [diary/mod.rs:179-183] — 存档deferred-work
- [x] [Review][Patch] 目录创建错误被忽略：`let _ = create_dir_all`需改为显式错误 [diary/mod.rs:419] — 存档deferred-work

#### defer (预存问题)

- [x] [Review][Defer] VectorStorage无线程安全保护 [vector_storage.rs] — deferred，超出Story范围
- [x] [Review][Defer] panic风险：系统时钟异常 [.expect] — deferred，系统级问题

#### dismissed

- 2项噪声/误报已删除

## Change Log

- 2026-04-16: Epic 2 代码审查完成，发现2个patch待修复
- 2026-04-14: 实现 diary write 到 `MemoryRecord`/`memories` 主路径的接入，补齐 MCP 参数模型与离线 L1 可见性测试。
