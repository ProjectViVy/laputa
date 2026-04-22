# Story 8.2: 主体数据导出
**Story ID:** 8.2  
**Story Key:** 8-2-data-export  
**Status:** done  
**Created:** 2026-04-14  
**Project:** 天空之城 (Laputa)

---

## 用户故事

As a **用户**,
I want **导出主体定义和核心记忆**,
So that **我可以迁移到新设备、新模型或新宿主，同时保留连续性的基础结构**。

---

## 验收标准

- **Given** 用户请求导出
- **When** 调用 `export.full()`
- **Then** 导出内容包含：
  - `identity.md`
  - `relation.md`
  - 近期摘要胶囊
  - 高热度记忆记录（`heat_i32 > 5000`）

- **And** 导出格式为文档化结构，满足 `NFR-6` 的“可迁移、可审阅、不依赖专有云服务”
- **And** 导出产物可在新实例中恢复或至少被新实例稳定消费

- **Given** 当前 repo 中部分导出内容并非原生文件形式持久化
- **When** 执行 `export.full()`
- **Then** 系统应为导出包生成稳定的文档视图，而不是要求这些文件事先已经存在
- **And** `relation.md` 必须从当前知识图谱状态导出
- **And** 近期摘要胶囊必须从 rhythm 层现有数据或稳定 fallback 视图导出

- **Given** 主体初始化已生成 `identity.md`
- **When** 执行完整导出
- **Then** 导出的主体定义以 `identity.md` 为主来源
- **And** 不得因为当前仓库中某些旧路径仍引用 `identity.txt` 而导出错误内容

- **And** 自动化测试覆盖最小可用导出、缺失关系数据、缺失 capsule 数据、仅高热记忆筛选、导出结构完整性，以及“导出不改写主库”的约束

扩展约束：
- 本 Story 负责“主体/关系/摘要/高热记忆”的迁移导出，不负责归档候选包；归档候选导出属于 `8.1 archive-export`
- 本 Story 不要求一次性实现完整导入器，但导出结构必须足够稳定，使后续 restore/import Story 能直接消费
- 本 Story 不得把所有 SQLite 内部表原样 dump 成迁移包并宣称完成，因为验收要求是“文档化结构”，不是内部数据库快照

---

## Epic 上下文

### Epic 8 目标

Epic 8 覆盖：
- `FR-13` 归档导出
- `FR-14` 数据导出与迁移

`8.2 data-export` 是 Epic 8 的第二个 Story，职责是把 Laputa 当前最重要、最可迁移的主体数据整理为一套可审阅、可搬运、可在新实例中恢复的导出结构。它解决的是“如何把主体连续性导出来”，而不是“如何做内部备份”。

### 与前置 Story 的关系

- `1.2 identity-initialization`
  - 已明确主体初始化写入 `identity.md`
  - 本 Story 必须以 `identity.md` 为主体定义的权威来源
- `3.3 wakepack-generate`
  - 明确唤醒过程依赖身份、关系与近期摘要
  - 本 Story 导出的结构应与这些消费路径兼容，而不是另外发明一套“只供导出使用”的专有格式
- `4.1 weekly-capsule`
  - 定义了近期摘要胶囊的语义与输出方向
  - 即使当前 `rhythm` 实现尚未完整落地，本 Story 也必须为摘要导出设计稳定接口或 fallback
- `7.1 relation-node`
  - 明确关系数据当前真实落点是 `knowledge_graph`
  - 因此 `relation.md` 在导出时应被生成，而不是假设仓库已有原生文件
- `8.1 archive-export`
  - 已把归档候选导出单独拆分出去
  - 本 Story 不应混入低热归档候选；目标是主体迁移，不是考古包

---

## 现有代码情报

### 已存在且必须复用的能力

1. `Laputa/src/identity/initializer.rs`
   - 已在 `config_dir` 下创建 `identity.md`
   - `Laputa/tests/test_identity.rs` 已验证该文件存在与格式

2. `Laputa/src/vector_storage.rs`
   - 当前持有核心记忆主库 `vectors.db` 的 SQLite 连接
   - 已可读取 `MemoryRecord`
   - 已包含 `heat_i32`、`last_accessed`、`access_count`、`is_archive_candidate`
   - 是高热记忆筛选导出的自然落点

3. `Laputa/src/knowledge_graph/mod.rs`
   - 当前关系数据真实存于 `entities` / `triples`
   - 尚不存在 `relation.md` 文件化持久层
   - 因此需要在导出阶段生成 `relation.md` 文档视图

4. `Laputa/src/rhythm/mod.rs`
   - 仍是占位模块
   - `4.1` 故事要求近期摘要胶囊最终来自 rhythm / L2
   - 本 Story 需要为“有真实 capsule”和“尚无真实 capsule”两条路径都留出明确处理策略

5. `Laputa/src/config.rs`
   - 真实运行时配置路径是 `config_dir/config.json`
   - 没有现成的 TOML 运行时接入链路
   - 导出元数据记录应继续遵循这个路径体系

### 当前缺口与实现风险

1. **主体文件路径不一致**
   - `IdentityInitializer` 写的是 `identity.md`
   - `storage/mod.rs` 的 `Layer0` / `MemoryStack` 仍有 `identity.txt` 旧路径
   - 如果开发者直接复用旧路径，`export.full()` 可能导出错文件或空内容

2. **关系没有现成文件**
   - Epic AC 指定了 `relation.md`
   - 当前 repo 实际上只有 knowledge graph SQLite 数据
   - 说明 `relation.md` 应是导出视图，而非现成存量文件

3. **capsule 数据尚未稳定落地**
   - `rhythm` 仍为空壳
   - 近期摘要胶囊的导出不能假装已有完善持久化
   - 需要设计稳定 fallback：要么导出空 section 并带 manifest 标注，要么从现有最近摘要来源生成临时文档视图

4. **迁移包边界不清风险**
   - `FR-14` 要求的是“主体定义、摘要胶囊、核心记忆记录”迁移
   - 如果开发者直接复制整个工作目录或整个 SQLite 数据库，会把不属于迁移合同的内部实现细节一起导出

---

## 架构与实现约束

### 1. 导出目标必须是文档化结构

- 导出结果应是人可审阅的目录结构
- 目录中应包含命名稳定的文档和数据文件
- 不要把“单个 SQLite 文件 + 若干内部表”当作 `export.full()` 的最终交付

推荐结构：

```text
export/
├── manifest.json
├── identity.md
├── relation.md
├── capsules/
│   └── recent-*.md
└── memories/
    └── core-memories.jsonl
```

可接受的变体：
- 高热记忆也可以导出为 `core-memories.md` 或 `core-memories.json`
- 但必须是文档化、可恢复、可审查的稳定格式

### 2. 导出内容边界

- `identity.md`：
  - 必须优先使用真实存在的 `config_dir/identity.md`
  - 若代码路径仍有 `identity.txt` 残留，只能作为兼容 fallback，不能覆盖 `identity.md` 主来源

- `relation.md`：
  - 必须从 `knowledge_graph` 当前有效关系生成
  - 推荐导出“当前关系 + 最近变化摘要”
  - 不要简单 dump 全量 triples 原始行

- `近期摘要胶囊`：
  - 优先从 rhythm 持久层读取
  - 如果 rhythm 尚未落地，必须定义稳定 fallback，并在 manifest 中注明摘要来源状态

- `高热记忆`：
  - 明确筛选条件为 `heat_i32 > 5000`
  - 不得混入低热归档候选
  - 导出时应保留稳定 ID 与关键元数据

### 3. 模块边界

- `export.full()` 应有单独导出模块或门面
- 不要把完整导出逻辑散落到 CLI / MCP / identity / knowledge_graph 各处

推荐组织方式：

```text
Laputa/src/export/
├── mod.rs
├── full.rs
├── manifest.rs
└── render.rs
```

如果团队希望避免新增顶级模块，也可落在：

```text
Laputa/src/archiver/
Laputa/src/identity/
Laputa/src/rhythm/
Laputa/src/knowledge_graph/
```

但必须有单一协调入口。

### 4. 运行时配置与状态

- 最近一次 full export 的路径、时间、导出计数等元数据，应记录在现有 `config_dir/config.json` 管理体系或同目录受其管理的状态文件中
- 不要为本 Story 引入新的并行配置系统

### 5. 数据安全约束

- 完整导出不得修改主库数据
- 不得重写 `heat_i32`
- 不得改变关系有效期
- 不得生成“导出声称成功，但结构缺关键文件”的假成功结果

### 6. 公共接口与错误语义

- 公共导出入口返回 `Result<T, LaputaError>`
- public API 需带 `///` 文档注释
- 命名遵守 `snake_case`

---

## 推荐实现方案

### 推荐领域接口

```rust
pub struct FullExporter { ... }

impl FullExporter {
    pub fn export_full(&self, output_dir: Option<PathBuf>) -> Result<FullExportResult, LaputaError>;
}
```

建议结果对象：

```rust
pub struct FullExportResult {
    pub export_dir: PathBuf,
    pub identity_path: PathBuf,
    pub relation_path: PathBuf,
    pub capsule_count: usize,
    pub exported_memory_count: usize,
    pub exported_at: i64,
}
```

### 推荐执行流程

1. 解析默认导出目录
2. 创建导出根目录
3. 复制或渲染 `identity.md`
4. 从 `knowledge_graph` 生成 `relation.md`
5. 从 rhythm 层导出最近 capsule，或生成稳定 fallback
6. 从 `vector_storage` 查询 `heat_i32 > 5000` 的核心记忆
7. 将核心记忆导出为稳定文档化文件
8. 生成 `manifest.json`，记录：
   - 导出时间
   - 导出来源路径
   - 记忆条数
   - capsule 条数
   - 是否使用 capsule fallback
   - 版本信息
9. 仅在结构完整落盘后记录最近一次导出路径/状态

### 关于 `relation.md` 的明确要求

当前 repo 没有原生 `relation.md` 文件，因此推荐把它定义为导出视图，例如：

```markdown
# Relations

## Current
- Alice -> project_x | collaborates_with | resonance: 72

## Recent Changes
- 2026-04-10: Alice -> project_x resonance 60 -> 72
```

关键点：
- 这是一种稳定的人类可读导出表示
- 它应来自当前有效关系与最近变化，而不是随手 dump JSON

### 关于近期摘要胶囊的 fallback

如果当前项目还没有真实持久化 capsule，本 Story 允许：
- 生成 `capsules/recent-placeholder.md`
- 或在 `manifest.json` 中明确 `capsule_export_status: "not_available"`

但不允许：
- 静默缺失 capsule 且仍宣称导出结构完整
- 直接把 WakePack 当成 recent capsule 替代物

### 关于高热记忆导出格式

高热记忆建议最少保留：
- `id`
- `text_content`
- `wing`
- `room`
- `valid_from`
- `valid_to`
- `heat_i32`
- `last_accessed`
- `access_count`
- `emotion_valence`
- `emotion_arousal`

推荐导出格式之一：

```json
{"id":1,"wing":"memory","room":"weekly","heat_i32":7800,"text_content":"..."}
```

使用 `jsonl` 的好处：
- 人可查看
- 机器可增量导入
- 与“文档化结构”兼容

---

## 特别防错说明

### 1. `identity.md` vs `identity.txt`

- `1.2` 以及测试明确规定主体初始化写入 `identity.md`
- 当前 `storage/mod.rs` 仍有 `identity.txt` 遗留读取逻辑
- 对 `8.2` 来说，主体导出必须以 `identity.md` 为准
- 如实现者顺手修复该路径不一致，这是正向改进；但即使不修复全仓，也必须保证 full export 不读错文件

### 2. 不要假设 `relation.md` 已存在

- 它当前不是存量文件，而是应由导出器生成

### 3. 不要把 `8.1` 与 `8.2` 混成一个包

- `8.1` 面向低热归档候选
- `8.2` 面向主体迁移
- 这两者的内容和目的不同

---

## 测试要求

至少补齐以下测试：

1. 最小完整导出
   - identity 存在
   - 高热记忆存在
   - 导出目录结构完整

2. identity 主来源正确
   - `identity.md` 存在时必须优先导出它
   - 不得错误读取 `identity.txt`

3. 关系导出
   - knowledge graph 有数据时生成 `relation.md`
   - 内容包含当前关系或最近变化

4. capsule 缺失 fallback
   - 没有真实 capsule 时，导出结果仍可解释
   - manifest 明确记录 fallback 状态

5. 高热记忆筛选
   - 仅 `heat_i32 > 5000` 的记录被导出
   - 低热记录不混入 full export 的核心记忆集合

6. 主库不变
   - 导出后原始 SQLite 与 knowledge graph 数据不被改写

7. 导出元数据记录
   - 成功导出后，路径/时间被记录到现有运行时配置路径体系
   - 失败时不记录假路径

推荐测试文件：
- `Laputa/tests/test_export_full.rs`

可复用：
- `Laputa/tests/test_identity.rs`
- 未来的 `test_rhythm.rs`
- 未来的关系测试 fixture

---

## 禁止事项

- 不要把整个 SQLite 数据库原样打包当作 `export.full()`
- 不要假设 `relation.md` 是现成文件
- 不要把 WakePack 直接当成摘要胶囊导出
- 不要读取错误的主体文件来源
- 不要引入与 `config_dir/config.json` 并行的新配置系统
- 不要在导出流程中修改主库内容

---

## 实施任务

- [x] 建立 `export.full()` 的单一协调入口
- [x] 导出 `identity.md`，并明确以 `config_dir/identity.md` 为主体定义权威来源
- [x] 从 `knowledge_graph` 生成 `relation.md`
- [x] 接入 rhythm 层最近 capsule 导出；若尚未落地则实现稳定 fallback
- [x] 从 `vector_storage` 导出 `heat_i32 > 5000` 的核心记忆
- [x] 生成 `manifest.json` 记录导出元数据与缺省状态
- [x] 在现有运行时配置路径体系中记录最近一次 full export
- [x] 补齐 `Laputa/tests/test_export_full.rs`
- [x] 运行 `cargo test`
- [x] 运行 `cargo clippy --all-features --tests -- -D warnings`

---

## 完成定义

- [x] `export.full()` 生成可审阅的文档化导出目录
- [x] 导出目录包含 `identity.md`、`relation.md`、近期摘要胶囊或明确 fallback、以及高热记忆数据
- [x] 高热记忆筛选条件符合 `heat_i32 > 5000`
- [x] 导出结果足够稳定，后续可被 restore/import 功能消费
- [x] full export 不改写主库
- [x] 自动化测试覆盖黄金路径与缺省路径
- [x] `cargo test` 通过
- [x] `cargo clippy --all-features --tests -- -D warnings` 通过

---

## 参考资料

- `D:\VIVYCORE\newmemory\_bmad-output\planning-artifacts\epics.md`
- `D:\VIVYCORE\newmemory\_bmad-output\planning-artifacts\prd.md`
- `D:\VIVYCORE\newmemory\_bmad-output\planning-artifacts\architecture.md`
- `D:\VIVYCORE\newmemory\_bmad-output\implementation-artifacts\1-2-identity-initialization.md`
- `D:\VIVYCORE\newmemory\_bmad-output\implementation-artifacts\3-3-wakepack-generate.md`
- `D:\VIVYCORE\newmemory\_bmad-output\implementation-artifacts\4-1-weekly-capsule.md`
- `D:\VIVYCORE\newmemory\_bmad-output\implementation-artifacts\7-1-relation-node.md`
- `D:\VIVYCORE\newmemory\_bmad-output\implementation-artifacts\8-1-archive-export.md`
- `D:\VIVYCORE\newmemory\Laputa\src\identity\initializer.rs`
- `D:\VIVYCORE\newmemory\Laputa\src\storage\mod.rs`
- `D:\VIVYCORE\newmemory\Laputa\src\vector_storage.rs`
- `D:\VIVYCORE\newmemory\Laputa\src\knowledge_graph\mod.rs`
- `D:\VIVYCORE\newmemory\Laputa\src\rhythm\mod.rs`
- `D:\VIVYCORE\newmemory\Laputa\tests\test_identity.rs`

---

## Dev Agent Record

### Agent Model Used

GPT-5

### Debug Log References

- Story creation only; no implementation commands executed.
- `cargo test --test test_export_full`
- `cargo fmt --all`
- `cargo clippy --all-features --tests -- -D warnings`
- `cargo test -- --nocapture`

### Completion Notes List

- 新增 `export` 模块与 `FullExporter` 协调入口，产出文档化目录结构，包括 `manifest.json`、`identity.md`、`relation.md`、`capsules/` 与 `memories/core-memories.jsonl`。
- full export 严格以 `config_dir/identity.md` 为主体来源，不读取旧的 `identity.txt` 作为主来源，避免导出错误主体定义。
- `relation.md` 由 `knowledge_graph` 当前有效关系与近期关系变化动态渲染；当关系库不存在时导出可解释的空视图，而不是伪造现成文件。
- capsule 导出优先复用 `rhythm::load_latest_capsule()`；缺失时通过 `manifest.json` 标记 `capsule_export_status = not_available`，保持结构稳定且可解释。
- 扩展 `MempalaceConfig` 持久化 `full_export_state`，并修复 `VectorStorage::get_memories()` 的超大 `LIMIT` 边界问题，避免导出时触发 SQLite `datatype mismatch`。
- 明确了 `8.2` 的目标是“文档化迁移导出”，不是内部数据库备份，因此要求生成稳定目录结构与 manifest。
- 明确了 `relation.md` 在当前 repo 中不是存量文件，而应从 `knowledge_graph` 生成导出视图。
- 明确了 `identity.md` 与 `identity.txt` 的现有路径不一致，要求 full export 以 `identity.md` 为权威来源，避免导出错误主体内容。
- 明确了近期摘要胶囊当前仍可能缺失真实持久层，因此故事要求实现可解释的 fallback，而不是假装 capsule 已经存在。
- 当前工作区不是 git repository，无法提供最近提交模式参考；本 Story 以现有源码与规划文档为准。

### File List

- `_bmad-output/implementation-artifacts/8-2-data-export.md`
- `_bmad-output/implementation-artifacts/sprint-status.yaml`
- `Laputa/src/config.rs`
- `Laputa/src/export/mod.rs`
- `Laputa/src/export/full.rs`
- `Laputa/src/lib.rs`
- `Laputa/src/vector_storage.rs`
- `Laputa/tests/test_export_full.rs`

### Change Log

- 2026-04-16 13:54:13 +08:00: 实现 full export 文档化导出目录、关系与 capsule 渲染、核心记忆 JSONL 导出、full export 运行时状态持久化，以及对应自动化测试。
- 2026-04-16 代码审查完成：全部8项验收标准通过，18项发现均归入defer。

---

## Review Findings

### Deferred 级发现

- [x] [Review][Defer] KnowledgeGraph连接未关闭 [full.rs:110-111] — deferred, 低频导出场景
- [x] [Review][Defer] VectorStorage重复初始化开销 [full.rs:177-181] — deferred, 非当前Story范围
- [x] [Review][Defer] 空JSONL语义模糊 [full.rs:228-233] — deferred, 空文件合法
- [x] [Review][Defer] capsule导出仅支持单个 [full.rs:164-175] — deferred, Story规范允许多个但实现单个可接受
- [x] [Review][Defer] relation导出无ID字段 [full.rs:125-129] — deferred, 恢复流程是后续Story

### 验收标准验证

| AC | 状态 |
|----|----|
| AC1: 所有组件完整导出 | ✅ PASSED |
| AC2: 文档化结构格式 | ✅ PASSED |
| AC3: identity.md优先读取 | ✅ PASSED |
| AC4: 从KG生成relation.md | ✅ PASSED |
| AC5: capsule fallback状态记录 | ✅ PASSED |
| AC6: 高热阈值过滤正确 | ✅ PASSED |
| AC7: 主库只读不改写 | ✅ PASSED |
| AC8: 失败不记录假路径 | ✅ PASSED |
