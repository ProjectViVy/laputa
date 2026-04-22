# Story 1.1: 项目结构建立与模块继承

Status: done

## Story

As a 开发者，
I want 建立 Laputa 项目结构并从 mempalace-rs 继承必要模块，
so that 代码基线完整，后续所有 Story 都有可工作的项目骨架。

## Acceptance Criteria

1. **Given** `d:\VIVYCORE\newmemory\mempalace-rs` 已验证可用（197 tests 通过）
   **When** 开发者完成 Laputa 项目初始化
   **Then** `d:\VIVYCORE\newmemory\Laputa/src/` 包含如下完整目录结构：
   - 新增模块：`heat/`, `archiver/`, `wakeup/`, `rhythm/`, `identity/`, `cli/`, `api/`, `utils/`
   - 继承模块（从 mempalace-rs 复制）：`storage/`, `searcher/`, `knowledge_graph/`, `dialect/`, `diary/`, `mcp_server/`

2. **Given** Laputa 目录结构建立完成
   **When** 检查 `Laputa/Cargo.toml`
   **Then** 依赖版本与 mempalace-rs 保持一致，所有必须依赖均已声明（见 Dev Notes 版本清单）

3. **Given** 继承模块已复制
   **When** 执行 `cargo build` （在 Laputa/ 目录下）
   **Then** 编译通过，无 error（warning 允许存在但需注释说明）

4. **Given** Laputa/Cargo.toml 配置正确
   **When** 执行 `cargo test` （在 Laputa/ 目录下）
   **Then** 继承的 mempalace-rs 测试逻辑可以运行，测试不因目录重构而失败

5. **Given** 新增的模块目录已创建
   **When** 检查每个新增模块
   **Then** 每个新增模块目录至少包含 `mod.rs` 占位文件，文件中有模块说明注释

## Tasks / Subtasks

> 🚨 **任务执行顺序强制规定**：Task 2（搬运 mempalace-rs 源文件）必须在 Task 3 之前完成。
> **后续所有 Epic（2~8）的 Story 均依赖这批文件已存在于 Laputa/src/ 中。**
> 如果 Task 2 跳过或只建空目录，后续 Story 将在没有代码基线的情况下开发，导致灾难性返工。

- [x] Task 1: 创建 Laputa Cargo.toml（AC: #2）
  - [x] 以 mempalace-rs/Cargo.toml 为基础，修改 package 名为 `laputa`
  - [x] 新增依赖：`serial_test = "2.x"` (dev-dependencies)
  - [x] 新增依赖：`uuid = { version = "1.x", features = ["v4"] }` (若 mempalace-rs 未引入则添加)
  - [x] 保留所有 mempalace-rs 原有依赖版本（clap 4.6.0, serde 1.0.228, rusqlite 0.32, tokio 1.51.0 等）
  - [x] 保留 `[patch.crates-io]` usearch 补丁（Windows MAP_FAILED fix）

- [x] Task 2: 🔴【第一优先级 · 阻塞后续所有工作】将 mempalace-rs 源文件物理搬运到 Laputa/src/（AC: #1）
  > **这是本 Story 最重要的任务。必须真实复制文件内容，不得只创建空目录或空 mod.rs 占位。**
  > **没有这批代码，后续 Story 1.2、1.3 以及 Epic 2~8 的所有开发都无从下手。**
  >
  > 源目录：`d:\VIVYCORE\newmemory\mempalace-rs\src\`
  > 目标目录：`d:\VIVYCORE\newmemory\Laputa\src\`
  >
  > ⚠️ mempalace-rs 用**单文件**（`xxx.rs`），Laputa 用**子目录模块**（`xxx/mod.rs`），复制时必须重命名路径：
  - [x] 创建目录 `Laputa/src/storage/`，将 `mempalace-rs/src/storage.rs` 完整内容复制为 `Laputa/src/storage/mod.rs`
  - [x] 创建目录 `Laputa/src/searcher/`，将 `mempalace-rs/src/searcher.rs` 完整内容复制为 `Laputa/src/searcher/mod.rs`
  - [x] 创建目录 `Laputa/src/knowledge_graph/`，将 `mempalace-rs/src/knowledge_graph.rs` 完整内容复制为 `Laputa/src/knowledge_graph/mod.rs`
  - [x] 创建目录 `Laputa/src/dialect/`，将 `mempalace-rs/src/dialect.rs`（51.2KB，含 EMOTION_CODES）完整内容复制为 `Laputa/src/dialect/mod.rs`，**不修改任何内容**
  - [x] 创建目录 `Laputa/src/diary/`，将 `mempalace-rs/src/diary.rs` 完整内容复制为 `Laputa/src/diary/mod.rs`
  - [x] 创建目录 `Laputa/src/mcp_server/`，将 `mempalace-rs/src/mcp_server.rs`（57.3KB，含 20+ MCP Tools）完整内容复制为 `Laputa/src/mcp_server/mod.rs`
  - [x] 将 `mempalace-rs/src/models.rs` 完整内容复制为 `Laputa/src/models.rs`（Wing struct 定义）
  - [x] 将 `mempalace-rs/src/config.rs` 完整内容复制为 `Laputa/src/config.rs`（MempalaceConfig，后续 Story 扩展）
  - [x] 将 `mempalace-rs/src/vector_storage.rs` 完整内容复制为 `Laputa/src/vector_storage.rs`（usearch 向量索引封装）
  - [x] **验收**：确认以上 9 个文件/目录在 `Laputa/src/` 中真实存在且有完整代码内容（非空文件）

- [x] Task 3: 创建新增模块目录及 mod.rs 占位（AC: #1, #5）
  - [x] `Laputa/src/heat/mod.rs` — 模块注释: "热度机制模块 (ADR-003)，包含 HeatService/HeatIndex/decay/config/state"
  - [x] `Laputa/src/archiver/mod.rs` — 模块注释: "归档模块 (ADR-005)，Phase 1 只标记，Phase 2 实现 packer/digger"
  - [x] `Laputa/src/wakeup/mod.rs` — 模块注释: "唤醒包生成模块，<1200 tokens 限制 (NFR-3)"
  - [x] `Laputa/src/rhythm/mod.rs` — 模块注释: "节律整理模块，生成周级摘要胶囊"
  - [x] `Laputa/src/identity/mod.rs` — 模块注释: "主体身份初始化模块，对应 P0 优先级"
  - [x] `Laputa/src/cli/mod.rs` — 模块注释: "CLI 接口模块，子命令 init/write/recall/wakeup/mark"
  - [x] `Laputa/src/api/mod.rs` — 模块注释: "统一抽象层 (ADR-009)，MemoryOperation trait + LaputaError"
  - [x] `Laputa/src/utils/mod.rs` — 模块注释: "工具函数：时间工具 + 结构化日志"

- [x] Task 4: 创建 Laputa/src/lib.rs 和 main.rs（AC: #3）
  - [x] `lib.rs`：声明所有模块（`pub mod heat; pub mod archiver; ...` 等），引入依赖
  - [x] `main.rs`：CLI 入口桩，暂用 `fn main() { println!("Laputa v0.1.0"); }`

- [x] Task 5: 创建配置文件骨架（AC: #2）
  - [x] `Laputa/config/laputa.toml` — 包含 [heat]/[archive]/[storage]/[wakeup] 段落（见 Dev Notes 完整模板）
  - [x] `Laputa/config/laputa.toml.example` — 同上作为示例

- [x] Task 6: 创建测试骨架（AC: #4）
  - [x] `Laputa/tests/fixtures/mod.rs` — 导出 fixture 模块
  - [x] `Laputa/tests/fixtures/time_machine.rs` — TimeMachine 结构体占位（见 Dev Notes 模板）
  - [x] `Laputa/tests/fixtures/memory_only.rs` — 纯内存 fixture 占位
  - [x] `Laputa/tests/fixtures/with_tempdir.rs` — tempdir fixture 占位

- [x] Task 7: 验证编译和测试（AC: #3, #4）
  - [x] 在 `Laputa/` 目录执行 `cargo build`，修复所有 error
  - [x] 执行 `cargo test`，确认继承模块测试通过
  - [x] 执行 `cargo clippy --all-features`，lint 通过

## Dev Notes

### 🚨 最重要警告：必须真实搬运 mempalace-rs 源文件

**这不是可选项。这是整个天空之城项目的代码基线。**

- `mempalace-rs` 是已验证可用的代码库（197 tests 通过）
- Laputa **不是从零开始写**，是基于 mempalace-rs **演化**
- 如果只建空目录而不搬运文件内容，则：
  - Story 1.2 `identity` 没有 `storage.rs` 就无法创建数据库
  - Story 1.3 `MemoryRecord` 扩展没有基础结构可用
  - 所有 Epic 2~8 都将在空屑上开工

**必须通过 Task 2 完成以下 9 个文件的真实复制：**

| 源文件 | 目标路径 | 说明 |
|--------|---------|------|
| `mempalace-rs/src/storage.rs` | `Laputa/src/storage/mod.rs` | 存储层核心 |
| `mempalace-rs/src/searcher.rs` | `Laputa/src/searcher/mod.rs` | 向量检索/RAG |
| `mempalace-rs/src/knowledge_graph.rs` | `Laputa/src/knowledge_graph/mod.rs` | 时间三元组 |
| `mempalace-rs/src/dialect.rs` | `Laputa/src/dialect/mod.rs` | EMOTION_CODES 不得修改 |
| `mempalace-rs/src/diary.rs` | `Laputa/src/diary/mod.rs` | 日记存储 |
| `mempalace-rs/src/mcp_server.rs` | `Laputa/src/mcp_server/mod.rs` | 20+ MCP Tools |
| `mempalace-rs/src/models.rs` | `Laputa/src/models.rs` | Wing struct |
| `mempalace-rs/src/config.rs` | `Laputa/src/config.rs` | MempalaceConfig |
| `mempalace-rs/src/vector_storage.rs` | `Laputa/src/vector_storage.rs` | usearch 封装 |

---

### 关键约束（必须遵守）

| 规则 | 说明 | 来源 |
|------|------|------|
| **R-002** | 继承 mempalace-rs 时不修改现有 public API 签名 | 防止破坏测试 |
| **R-004** | 所有公共函数返回 `Result<T, LaputaError>` | API 统一 |
| **R-007** | 所有 public API 必须有 `///` 文档注释 | 文档生成 |
| **ADR-006** | 扩展而非重写 mempalace-rs 模块 | 架构决策 |

### ⚠️ 灾难预防：mempalace-rs 是单文件结构

mempalace-rs 的源码使用**单文件模式**（`storage.rs`, `searcher.rs` 等），不是子目录模块。
Laputa 使用**子目录模块**（`storage/mod.rs` 等）。

复制时必须注意：
- `mempalace-rs/src/storage.rs` → `Laputa/src/storage/mod.rs`（不是直接复制路径）
- 同理适用于：`searcher`, `knowledge_graph`, `dialect`, `diary`, `mcp_server`
- 依赖文件 `models.rs`, `config.rs`, `vector_storage.rs` 直接放在 `Laputa/src/` 根目录

### 依赖版本清单（从 mempalace-rs/Cargo.toml 继承）

```toml
[package]
name = "laputa"
version = "0.1.0"
edition = "2021"

[dependencies]
clap = { version = "4.6.0", features = ["derive"] }
serde = { version = "1.0.228", features = ["derive"] }
serde_json = "1.0.149"
anyhow = "1.0.102"
async-trait = "0.1"
rand = "0.9"
rusqlite = { version = "0.32", features = ["bundled"] }
tokio = { version = "1.51.0", features = ["full"] }
mcp_rs = { version = "0.1.0" }
futures = "0.3.32"
walkdir = "2.5.0"
regex = "1.12.3"
chrono = "0.4.44"
symspell = "0.3.0"
reqwest = { version = "0.12", features = ["json"] }
rayon = "1.10"
dialoguer = "0.11"
tempfile = "3.27.0"
lazy_static = "1.5.0"
md5 = "0.7.0"
hex = "0.4.3"
fastembed = "4"
usearch = "2"
ignore = "0.4.25"
uuid = { version = "1", features = ["v4"] }

[dev-dependencies]
criterion = { version = "0.5", features = ["html_reports"] }
serial_test = "2"          # 新增：测试隔离

[patch.crates-io]
usearch = { path = "../mempalace-rs/patches/usearch" }   # Windows MAP_FAILED fix
```

> ⚠️ `[patch.crates-io]` 中的路径需根据相对位置调整（Laputa 在 newmemory/Laputa/，mempalace-rs 在 newmemory/mempalace-rs/）

### config/laputa.toml 完整模板

```toml
[heat]
enabled = true
hot_threshold = 8000        # 80.00 — 锁定区间
warm_threshold = 5000       # 50.00 — 正常区间
cold_threshold = 2000       # 20.00 — 归档候选区间
decay_rate = 0.1            # 衰减系数
update_interval_hours = 1   # 批量衰减间隔

[archive]
enabled = false             # Phase 1 禁用自动归档
archive_threshold = 2000    # 低于此值标记为打包候选
check_interval_days = 1     # 归档检查间隔

[storage]
db_path = "./laputa.db"
vector_dim = 384            # fastembed 嵌入维度
usearch_path = "./laputa.usearch"

[wakeup]
max_tokens = 1200           # 唤醒包 token 上限 (NFR-3)
include_identity = true
include_recent_events = true
include_resonance = true
```

### TimeMachine 占位模板（tests/fixtures/time_machine.rs）

```rust
//! 时间模拟工具，用于热度衰减测试
//! 详见架构文档 ADR-012 (测试架构策略)

pub struct TimeMachine {
    /// 当前模拟时间（相对偏移秒数）
    offset_seconds: u64,
}

impl TimeMachine {
    pub fn new() -> Self {
        Self { offset_seconds: 0 }
    }

    /// 模拟时间流逝（推进 N 天）
    pub fn advance_days(&mut self, days: u64) {
        self.offset_seconds += days * 86400;
    }

    /// 固定时间，用于精确边界测试
    pub fn freeze(&self) -> u64 {
        self.offset_seconds
    }
}
```

### 项目目录结构参考

来自架构文档 `architecture.md` Section 6.2：

```
Laputa/
├── Cargo.toml                    # package: laputa
├── Cargo.lock
├── .gitignore
├── config/
│   ├── laputa.toml
│   └── laputa.toml.example
├── src/
│   ├── lib.rs
│   ├── main.rs
│   ├── models.rs                  # 继承自 mempalace-rs
│   ├── config.rs                  # 继承自 mempalace-rs（后续扩展）
│   ├── vector_storage.rs          # 继承自 mempalace-rs
│   ├── heat/mod.rs                # 新增
│   ├── archiver/mod.rs            # 新增
│   ├── wakeup/mod.rs              # 新增
│   ├── rhythm/mod.rs              # 新增
│   ├── identity/mod.rs            # 新增
│   ├── cli/mod.rs                 # 新增
│   ├── api/mod.rs                 # 新增 (LaputaError 定义位置)
│   ├── utils/mod.rs               # 新增
│   ├── storage/mod.rs             # 继承自 mempalace-rs/src/storage.rs
│   ├── searcher/mod.rs            # 继承自 mempalace-rs/src/searcher.rs
│   ├── knowledge_graph/mod.rs     # 继承自 mempalace-rs/src/knowledge_graph.rs
│   ├── dialect/mod.rs             # 继承自 mempalace-rs/src/dialect.rs
│   ├── diary/mod.rs               # 继承自 mempalace-rs/src/diary.rs
│   └── mcp_server/mod.rs          # 继承自 mempalace-rs/src/mcp_server.rs
├── tests/
│   └── fixtures/
│       ├── mod.rs
│       ├── time_machine.rs
│       ├── memory_only.rs
│       └── with_tempdir.rs
└── docs/
```

### LaputaError 骨架（api/mod.rs）

Story 1.1 需要在 `src/api/mod.rs` 提前定义 `LaputaError`，供后续 Story 使用：

```rust
//! 统一抽象层 (ADR-009)
//! MemoryOperation trait + LaputaError 统一错误处理 (ADR-010)

use uuid::Uuid;

/// 统一错误枚举，所有公共函数返回 Result<T, LaputaError>
#[derive(Debug, thiserror::Error)]
pub enum LaputaError {
    #[error("Storage error: {0}")]
    StorageError(String),
    #[error("Not found: {0}")]
    NotFound(Uuid),
    #[error("Validation error: {0}")]
    ValidationError(String),
    #[error("Heat threshold error: heat={0}")]
    HeatThresholdError(i32),
    #[error("Archive error: {0}")]
    ArchiveError(String),
    #[error("Wakepack size exceeded: {0} tokens")]
    WakepackSizeExceeded(usize),
    #[error("Config error: {0}")]
    ConfigError(String),
}

impl From<rusqlite::Error> for LaputaError {
    fn from(e: rusqlite::Error) -> Self {
        LaputaError::StorageError(e.to_string())
    }
}
```

> 注意：若 `thiserror` 不在 mempalace-rs 依赖中，可改用手动 `impl std::fmt::Display for LaputaError`，
> 或直接使用 `anyhow::Error`（mempalace-rs 已引入 anyhow 1.0.102）。

### 继承模块注意事项

| 模块 | 原文件 | 目标路径 | 需要注意 |
|------|--------|---------|---------|
| storage | `storage.rs` (1152行) | `storage/mod.rs` | 包含 Storage struct、Layer0/Layer1、Wing CRUD |
| searcher | `searcher.rs` (17.9KB) | `searcher/mod.rs` | 包含向量检索、RAG 能力 (D-004 必须启用) |
| knowledge_graph | `knowledge_graph.rs` | `knowledge_graph/mod.rs` | 时间三元组，共振度扩展点 |
| dialect | `dialect.rs` (51.2KB，最大) | `dialect/mod.rs` | EMOTION_CODES 直接沿用，不修改 |
| diary | `diary.rs` | `diary/mod.rs` | 日记存储，后续 Story 2.1 扩展 |
| mcp_server | `mcp_server.rs` (57.3KB) | `mcp_server/mod.rs` | 20+ MCP Tools，后续 Story 6.2 扩展 |
| models | `models.rs` | `src/models.rs` | Wing struct 定义 |
| config | `config.rs` | `src/config.rs` | MempalaceConfig 定义（后续扩展为 laputa 配置） |
| vector_storage | `vector_storage.rs` | `src/vector_storage.rs` | usearch 向量索引封装 |

### 测试策略

本 Story 为项目骨架搭建，测试目标：
1. `cargo build` 编译通过（零 error）
2. `cargo test` 不因模块重组而失败（继承的测试路径需正确）
3. 新增的占位 `mod.rs` 文件编译无误

> ⚠️ 不需要在本 Story 写功能性测试，功能测试在对应功能 Story 中完成

### Project Structure Notes

- 本 Story 建立的目录结构是后续所有 Epic 的地基
- `heat/`, `archiver/`, `wakeup/`, `rhythm/`, `identity/` 在本 Story 只需 mod.rs 占位
- `api/mod.rs` 需提前定义 `LaputaError` 枚举（供所有模块使用）
- 继承模块必须以**子目录 + mod.rs** 形式组织（与 mempalace-rs 的单文件不同）

### References

- [架构文档 6.2 完整项目目录结构](d:/VIVYCORE/newmemory/_bmad-output/planning-artifacts/architecture.md#6.2)
- [架构文档 5.6 强制规则 R-001~R-007](d:/VIVYCORE/newmemory/_bmad-output/planning-artifacts/architecture.md#5.6)
- [架构文档 ADR-006 mempalace-rs 继承边界](d:/VIVYCORE/newmemory/_bmad-output/planning-artifacts/architecture.md#4.1)
- [架构文档 ADR-012 测试架构策略](d:/VIVYCORE/newmemory/_bmad-output/planning-artifacts/architecture.md#4.5)
- [AGENTS.md 二、代码基线与继承](d:/VIVYCORE/newmemory/Laputa/AGENTS.md#二)
- [mempalace-rs Cargo.toml 依赖版本](d:/VIVYCORE/newmemory/mempalace-rs/Cargo.toml)
- [mempalace-rs src/ 目录结构](d:/VIVYCORE/newmemory/mempalace-rs/src/)
- [Epic 1 Story 1.1 验收标准](d:/VIVYCORE/newmemory/_bmad-output/planning-artifacts/epics.md#Story-1.1)

## Dev Agent Record

### Agent Model Used

codex/gpt-5.3-codex

### Debug Log References

- cargo build 初次失败：`Cargo.toml` 中继承的 `[[bench]]` 指向不存在路径，已移除该 stanza。
- 规范缺口记录：`mcp_server/mod.rs` 依赖 `crate::palace_graph::PalaceGraph`，`dialect/mod.rs` 依赖 `crate::extractor`；为满足 AC 编译与测试，最小额外继承了 `src/palace_graph.rs` 与 `src/extractor.rs`。
- 离线降级修正：保留读取类接口的 graceful degradation，但写入/删除路径改为显式报错，避免未持久化却返回成功。
- 环境约束记录：`lsp_diagnostics` 无法执行（系统缺少 `rust-analyzer.exe`；Biome LSP 也未安装）。

### Completion Notes List

- 按任务顺序完成：Task 1 → Task 2 → Task 3/4/5/6 → Task 7。
- 已完成单文件到目录模块重塑复制（`storage/searcher/knowledge_graph/dialect/diary/mcp_server` → `*/mod.rs`），并创建了 Story 要求的新模块占位、配置骨架、测试夹具。
- 为通过 `cargo test`（离线环境）做了最小兼容修复：读取类路径在向量存储不可用时保持 graceful degradation；写入/删除路径改为显式失败，避免 silent success；同时修复跨平台路径断言与测试数据库残留导致的用例不稳定。

### File List

- Laputa/Cargo.toml
- Laputa/src/lib.rs
- Laputa/src/main.rs
- Laputa/src/models.rs
- Laputa/src/config.rs
- Laputa/src/vector_storage.rs
- Laputa/src/storage/mod.rs
- Laputa/src/searcher/mod.rs
- Laputa/src/knowledge_graph/mod.rs
- Laputa/src/dialect/mod.rs
- Laputa/src/diary/mod.rs
- Laputa/src/mcp_server/mod.rs
- Laputa/src/palace_graph.rs
- Laputa/src/extractor.rs
- Laputa/src/heat/mod.rs
- Laputa/src/archiver/mod.rs
- Laputa/src/wakeup/mod.rs
- Laputa/src/rhythm/mod.rs
- Laputa/src/identity/mod.rs
- Laputa/src/cli/mod.rs
- Laputa/src/api/mod.rs
- Laputa/src/utils/mod.rs
- Laputa/config/laputa.toml
- Laputa/config/laputa.toml.example
- Laputa/tests/fixtures/mod.rs
- Laputa/tests/fixtures/time_machine.rs
- Laputa/tests/fixtures/memory_only.rs
- Laputa/tests/fixtures/with_tempdir.rs
- _bmad-output/implementation-artifacts/sprint-status.yaml

### Change Log

- 2026-04-13: 完成 Story 1.1 实现，状态更新为 review，sprint-status 对应条目更新为 review。
- 2026-04-13: 根据复核结果修正 MCP 身份/AAAK 元数据，并消除写入/删除路径的 silent-success 离线降级语义。
