# Story 9.1: 构建链路去同级路径依赖

**Story ID:** 9.1  
**Story Key:** 9-1-standalone-build-decoupling  
**Status:** done  
**Created:** 2026-04-20  
**Project:** 天空之城 (Laputa)

---

## 用户故事

As a **维护者**，
I want **移除 Laputa 对父工作区和兄弟目录的构建期依赖**，
So that **项目被单独 clone 到新工作服务器后也能直接编译与测试**。

---

## 验收标准

- **Given** 当前 `Laputa` 项目仍存在 `path = "../..."` 或等价的兄弟目录构建依赖
- **When** 开发者执行独立仓库化修补
- **Then** `Cargo.toml` 中不得再存在指向父工作区/兄弟目录的构建期依赖或 patch

- **And** 如果 `usearch` patch 属于必须保留的修复：
  - 必须将 patch 以本仓可维护形式内置，或
  - 切换到可公开获取的稳定来源

- **And** 仅复制 `Laputa/` 目录到全新路径后执行 `cargo build` 必须成功

- **And** 仅复制 `Laputa/` 目录到全新路径后执行核心测试集必须成功

- **And** 不允许通过 README 或开发说明要求用户额外 clone `mempalace-rs`、`agent-diva` 或其他兄弟仓来满足构建前提

---

## 开发任务

- [x] 识别所有构建期路径依赖、patch 和隐式兄弟目录引用
- [x] 修复 `Cargo.toml` 中的本地路径 patch / dependency
- [x] 如需保留 patch，确定本仓内置方案或公开来源替代方案
- [x] 在脱离父工作区的目录中验证 `cargo build`
- [x] 在脱离父工作区的目录中验证核心测试集
- [x] 记录所有被移除的上游路径假设

---

## 说明

这是当前迁移阻断链路中的 **P0 story**。如果本 Story 未完成，后续文档修订和新服务器迁移验证都没有意义。

---

## Dev Agent Record

### Implementation Plan

- 将 `usearch` patch 从 `../mempalace-rs/patches/usearch` 内置到 `Laputa/vendor/usearch`
- 修改 `Laputa/Cargo.toml` 仅引用仓内路径，消除构建期父目录依赖
- 增加 manifest 防回归测试，防止后续重新引入 `../` 构建路径
- 在只复制 `Laputa/` 的隔离目录中执行 `cargo build` 与核心测试验证

### Debug Log

- 2026-04-20 23:36:44 +08:00 识别到唯一构建期父目录依赖为 `Laputa/Cargo.toml` 中的 `[patch.crates-io] usearch = { path = "../mempalace-rs/patches/usearch" }`
- 2026-04-20 23:36:44 +08:00 已将 `mempalace-rs/patches/usearch` 复制到 `Laputa/vendor/usearch` 作为仓内可维护 patch 基线
- 2026-04-20 23:53:06 +08:00 已将 `Laputa/Cargo.toml` 的 `usearch` patch 改为仓内路径 `vendor/usearch`
- 2026-04-20 23:53:06 +08:00 已新增 `Laputa/tests/test_standalone_build.rs`，断言 manifest 不得重新引入 `../` 构建路径
- 2026-04-20 23:53:06 +08:00 在隔离目录 `C:\Users\com01\AppData\Local\Temp\laputa-standalone-eaa4b8d0-6089-4d93-9fc2-ed4c755e10e6\Laputa` 执行 `cargo build` 成功
- 2026-04-20 23:53:06 +08:00 在隔离目录验证 `cargo test test_manifest_uses_in_repo_usearch_patch --test test_standalone_build`、`cargo test test_initialize_creates_db_and_identity --test test_identity`、`cargo test test_cli_init_diary_recall_wakeup_and_mark_flow --test test_cli_flow` 全部通过

### Completion Notes

- 将构建期 `usearch` patch 从父目录 `../mempalace-rs/patches/usearch` 内置到 `Laputa/vendor/usearch`，消除了独立 clone 时的兄弟仓前置条件
- 更新 `Laputa/Cargo.toml` 仅引用仓内 patch，并新增 manifest 防回归测试，确保后续不再引入父目录依赖
- 记录被移除的上游路径假设：当前唯一构建期父目录假设是 `Cargo.toml` 的 `usearch` patch；README/AGENTS/STATUS 中的兄弟仓引用不再作为构建前提，文档层修订留给 Story 9.2
- 在只复制 `Laputa/` 的隔离目录中完成 `cargo build` 与核心测试链路验证，满足 Story 9.1 的独立构建验收

### File List

- Laputa/Cargo.toml
- Laputa/vendor/usearch
- Laputa/tests/test_standalone_build.rs
- _bmad-output/implementation-artifacts/9-1-standalone-build-decoupling.md
- _bmad-output/implementation-artifacts/sprint-status.yaml

## Change Log

- 2026-04-20: Story 9.1 开始实施，已完成构建期路径依赖盘点并建立仓内 `usearch` patch 基线
- 2026-04-20: 完成 `usearch` 仓内内置、manifest 防回归测试以及隔离目录独立构建验证，故事状态更新为 `review`

---

## Review Findings (2026-04-21)

### Decision-needed

- [x] **[Review][Decision] AC3/AC4 需执行实际验证确认独立构建能力** - RESOLVED: 2026-04-21 验证通过，隔离目录 cargo build (48.32s) + 核心测试集全部通过

### Patch

- [x] **[Review][Patch] `../` 检测过于宽泛导致假阳性** [test_standalone_build.rs:14-15] - FIXED: 改用逐行检查 `path = "..."` 中是否包含 `../`
- [x] **[Review][Patch] 精确字符串匹配脆弱** [test_standalone_build.rs:10] - FIXED: 改用双重检查 `usearch` + `vendor/usearch` + 精确路径验证

### Defer (vendor/usearch 上游库问题)

- [x] **[Review][Defer] `unwrap()` on `CARGO_CFG_TARGET_OS`** [vendor/usearch/build.rs:61] - deferred, pre-existing upstream issue
- [x] **[Review][Defer] Windows `/sdl-` 禁用安全检查** [vendor/usearch/build.rs:73] - deferred, pre-existing upstream issue
- [x] **[Review][Defer] MSVC 兼容性定义抑制类型安全** [vendor/usearch/build.rs:74-75] - deferred, pre-existing upstream issue
- [x] **[Review][Defer] 编译重试循环非理想错误处理** [vendor/usearch/build.rs:100] - deferred, pre-existing upstream issue
- [x] **[Review][Defer] 未处理 OS 目标缺少编译标志** [vendor/usearch/build.rs:63-90] - deferred, pre-existing upstream issue
- [x] **[Review][Defer] 未知架构默认 x86 SIMD 目标** [vendor/usearch/build.rs:27] - deferred, pre-existing upstream issue
- [x] **[Review][Defer] `flag_if_supported` 静默忽略** [vendor/usearch/build.rs:65-89] - deferred, pre-existing upstream issue
- [x] **[Review][Defer] b1x8 二进制向量维度不匹配** [vendor/usearch/rust/lib.cpp] - deferred, pre-existing upstream issue
- [x] **[Review][Defer] 错误消息误导** [vendor/usearch/rust/lib.cpp:121-125] - deferred, pre-existing upstream issue
- [x] **[Review][Defer] 临时字符串 c_str() 生命周期风险** [vendor/usearch/rust/lib.cpp:175-179] - deferred, pre-existing upstream issue
- [x] **[Review][Defer] 缓冲区操作无边界检查** [vendor/usearch/rust/lib.cpp:185-195] - deferred, pre-existing upstream issue
- [x] **[Review][Defer] unsafe 函数指针 cast** [vendor/usearch/rust/lib.cpp:85-91] - deferred, pre-existing upstream issue
- [x] **[Review][Defer] Drop 实现潜在 double-free** [vendor/usearch/rust/lib.rs:536-558] - deferred, pre-existing upstream issue
- [x] **[Review][Defer] unsafe Send/Sync 绕过线程安全验证** [vendor/usearch/rust/lib.rs:533-534] - deferred, pre-existing upstream issue
- [x] **[Review][Defer] filtered_search 闭包生命周期风险** [vendor/usearch/rust/lib.rs:729-740] - deferred, pre-existing upstream issue
- [x] **[Review][Defer] change_metric panic 非优雅终止** [vendor/usearch/rust/lib.rs:764] - deferred, pre-existing upstream issue
- [x] **[Review][Defer] MetricFunction 双重指针间接** [vendor/usearch/rust/lib.rs:485-490] - deferred, pre-existing upstream issue

### Dismissed

- 测试失败后无清理 [test_standalone_build.rs:7] - 非实际问题
- patch section 无条件应用 [Cargo.toml:55-56] - Acceptance Auditor 确认 AC 符合
