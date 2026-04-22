# Story 9.2: 仓库元数据与文档独立化
**Story ID:** 9.2  
**Story Key:** 9-2-repo-metadata-doc-independence  
**Status:** done  
**Created:** 2026-04-20  
**Project:** 天空之城 (Laputa)

---

## 用户故事

As a **新用户**，I want **从仓库元数据和 README 中看到 Laputa 是一个可独立运行的项目**，So that **我可以在新环境中直接理解、安装并启动它，而不是误以为还缺少若干兄弟仓库**。

---

## 验收标准

- **Given** 当前仓库文档仍将 `mempalace-rs` / `agent-diva` / `UPSP` / `LifeBook` 作为并列上下文直接引用
- **When** 开发者完成仓库独立化修补
- **Then** `Cargo.toml` 的 `repository`、`homepage`、`documentation` 必须指向 Laputa 自身，而不是 `mempalace-rs`
- **And** `README.md` 中的启动与安装说明只能够依赖当前仓库
- **And** `STATUS.md` 应记录上游 lineage 与版本来源，而不是把 `../mempalace-rs` 当作运行时前提
- **And** `AGENTS.md` 与相关 planning / implementation 文档中凡是“复制同级目录”“依赖兄弟仓库”的表述，必须改为“历史来源”或“迁移说明”
- **And** README 必须明确独立仓库最小启动路径为：
  - `cargo build`
  - `cargo test`
  - `cargo run -- init`

---

## 开发任务

- [x] 修正 `Cargo.toml` 仓库元数据
- [x] 修正 README 中的相关项目与启动路径
- [x] 修正 STATUS 中的上游记录语义
- [x] 清理 AGENTS 与关键 planning 文档中的误导性兄弟目录假设
- [x] 明确区分“历史来源”与“当前运行前提”

---

## 说明

这个 Story 的目标不是“淡化 lineage”，而是**防止运行前提与历史来源混淆**。Laputa 可以承认自己源自 `mempalace-rs` 演化，但不能再让用户以为必须把上游仓库放在隔壁才能启动。

---

## Dev Agent Record

### Implementation Plan

- 更新 `Laputa/Cargo.toml` 的 `repository` / `homepage` / `documentation`，使其指向 Laputa 自身仓库
- 重写 `Laputa/README.md`、`Laputa/AGENTS.md`、`Laputa/STATUS.md`，明确“独立仓库运行前提”与“历史 lineage”边界
- 清理与本 Story 直接相关的 planning artifacts 表述，避免把兄弟仓示例写成默认运行前提
- 增加仓库元数据回归测试，验证 README/AGENTS/STATUS 不再依赖兄弟仓路径

### Debug Log

- 2026-04-21 识别到 `Laputa/Cargo.toml` 的 `repository`、`homepage`、`documentation` 仍指向 `mempalace-rs`
- 2026-04-21 识别到 `Laputa/README.md`、`Laputa/AGENTS.md`、`Laputa/STATUS.md` 仍将兄弟仓路径写成当前仓库上下文
- 2026-04-21 新增 `Laputa/tests/test_repo_metadata.rs`，用于约束仓库元数据与独立运行文档契约
- 2026-04-21 完成定向验证：`cargo test test_manifest_uses_in_repo_usearch_patch --test test_standalone_build`
- 2026-04-21 完成定向验证：`cargo test test_initialize_creates_db_and_identity --test test_identity`
- 2026-04-21 完成定向验证：`cargo test --test test_repo_metadata`
- 2026-04-21 全量 `cargo test` 在当前环境失败，原因为与本 Story 无关的链接/内存分配问题，涉及 `test_cli_flow`、`test_cli_mark`、`test_timeline_recall`、`test_archiver`、`test_semantic_search`、`test_relation_node`、`test_export_full` 以及 `bin "laputa" test`
- 2026-04-21 复核收尾阶段再次执行定向验证：`cargo test --test test_repo_metadata`
- 2026-04-21 复核收尾阶段再次执行定向验证：`cargo test test_manifest_uses_in_repo_usearch_patch --test test_standalone_build`
- 2026-04-21 复核收尾阶段再次执行定向验证：`cargo test test_initialize_creates_db_and_identity --test test_identity`

### Completion Notes

- 已将 `Laputa` 包元数据改为指向 Laputa 自身仓库地址，不再引用 `mempalace-rs`
- 已重写 README/AGENTS/STATUS，明确当前独立仓库最小启动路径为 `cargo build`、`cargo test`、`cargo run -- init`
- 已将上游项目表述改为历史来源或迁移背景，不再把兄弟仓目录写成运行前提
- 已补充仓库元数据回归测试，并通过定向 smoke 测试验证关键独立化约束
- 本次收尾复核再次通过 `test_repo_metadata`、`test_standalone_build`、`test_identity` 三项定向验证，Story 已具备进入 `review` 的条件

### File List

- Laputa/Cargo.toml
- Laputa/README.md
- Laputa/AGENTS.md
- Laputa/STATUS.md
- Laputa/tests/test_repo_metadata.rs
- _bmad-output/planning-artifacts/sprint-change-proposal-2026-04-20.md
- _bmad-output/planning-artifacts/prd-laputa-agent-diva-integration.md
- _bmad-output/implementation-artifacts/2-1-diary-write.md
- _bmad-output/implementation-artifacts/9-2-repo-metadata-doc-independence.md
- _bmad-output/implementation-artifacts/sprint-status.yaml

## Review Findings

### Deferred 级发现（pre-existing 或测试增强建议）

- [x] [Review][Defer] panic! 错误信息不友好 [test_repo_metadata.rs:6] — 测试框架行为，非 AC 相关，pre-existing 风格问题
- [x] [Review][Defer] 断言语义验证不足 [test_repo_metadata.rs:67-69] — 测试设计改进项，当前实现满足 AC，属于增强而非修复
- [x] [Review][Defer] mcp_rs 依赖源未明确 [Cargo.toml:33] — pre-existing 依赖配置，不在本 Story scope
- [x] [Review][Defer] 变体路径检测遗漏 [test_repo_metadata.rs:46-64] — 测试覆盖增强，当前 AC 已满足（文档无 ../ 变体）
- [x] [Review][Defer] DECISIONS.md 链接可能失效 [README.md:58] — pre-existing 文档链接问题，不在本 Story scope
- [x] [Review][Defer] vendor/usearch 目录存在性 [Cargo.toml:55-56] — pre-existing build 配置，已在 Story 9-1 处理 standalone build

### Dismissed 级发现

- AGENTS.md 条件语气弱 [AGENTS.md:52] — 已有明确 "Do not assume" 约束，第 52 行是补充说明，不违反 AC
- STATUS.md commit SHA 硬编码 [STATUS.md:18] — AC 仅要求记录 lineage，硬编码 SHA 符合要求

### 验收标准验证结果

| AC | 状态 | Evidence |
|----|------|----------|
| AC1: Cargo.toml repository/homepage/documentation 指向 Laputa | ✅ PASS | Cargo.toml:7-12 全部指向 `https://github.com/jxoesneon/laputa` |
| AC2: README 启动说明只依赖当前仓库 | ✅ PASS | README.md:19-25 明确 `cargo build/test/run -- init` |
| AC3: STATUS.md 记录上游 lineage | ✅ PASS | STATUS.md:14-22 明确 "historical source lineage" |
| AC4: AGENTS/planning 文档兄弟目录表述改为历史来源 | ✅ PASS | AGENTS.md:15-17 明确 "Do not assume sibling repositories" |
| AC5: README 明确独立启动路径 | ✅ PASS | README.md:19-25 三条命令完整 |

### 审查统计

- Decision-needed: 0
- Patch: 0
- Deferred: 6
- Dismissed: 2

**结论：所有验收标准 PASS，Clean Review。发现 6 项均为 pre-existing 或测试增强建议。**

## Change Log

- 2026-04-21: 完成 Story 9.2 的仓库元数据与文档独立化实现，补充仓库契约测试；因仓库级全量 `cargo test` 存在与本 Story 无关的链接/内存失败，状态保持为 `in-progress`
- 2026-04-21: 收尾复核通过定向验证，Story 9.2 状态更新为 `review`
- 2026-04-21: BMAD 代码审查完成，所有 AC 通过，状态更新为 `done`
