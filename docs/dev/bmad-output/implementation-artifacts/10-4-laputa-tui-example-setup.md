# Story 10.4: Laputa TUI Example 复制与设置

Status: ready-for-dev

## Story

As a 开发者，
I want 将 agent-diva-nano TUI example 复制到 Laputa 项目中，
So that Laputa 项目可以展示一个完整的 TUI demo，作为 agent-diva-nano 的参考实现。

## Acceptance Criteria

1. **Given** agent-diva-nano TUI example 位于 `nano-workspace/agent-diva-nano/examples/tui/`
2. **When** 完成 TUI example 复制与设置
3. **Then** Laputa 项目包含以下内容：
   - `Laputa/examples/tui/` 目录结构完成
   - 所有 TUI 源文件复制到位（main.rs, app.rs, ui.rs 等）
   - `Cargo.toml` example 配置正确
   - 独立运行 `cargo run --example tui` 可正常启动
4. **And** 复制操作不影响原 agent-diva-nano TUI example
5. **And** 复制后的 TUI example 保留原 agent-diva-nano 依赖关系（暂不集成 Laputa）

## Tasks / Subtasks

- [ ] Task 1: 创建目标目录结构 (AC: #3)
  - [ ] 1.1 创建 `Laputa/examples/` 目录（如不存在）
  - [ ] 1.2 创建 `Laputa/examples/tui/` 目录

- [ ] Task 2: 复制 TUI 源文件 (AC: #3)
  - [ ] 2.1 复制 `main.rs` → `Laputa/examples/tui/main.rs`
  - [ ] 2.2 复制 `app.rs` → `Laputa/examples/tui/app.rs`
  - [ ] 2.3 复制 `ui.rs` → `Laputa/examples/tui/ui.rs`
  - [ ] 2.4 复制 `commands.rs` → `Laputa/examples/tui/commands.rs`
  - [ ] 2.5 复制 `config.rs` → `Laputa/examples/tui/config.rs`
  - [ ] 2.6 复制 `manager.rs` → `Laputa/examples/tui/manager.rs`
  - [ ] 2.7 复制 `provider.rs` → `Laputa/examples/tui/provider.rs`
  - [ ] 2.8 复制 `wizard.rs` → `Laputa/examples/tui/wizard.rs`

- [ ] Task 3: 配置 Cargo.toml example (AC: #3)
  - [ ] 3.1 添加 `[[example]]` 配置段
  - [ ] 3.2 配置 example 名称 `name = "tui"`
  - [ ] 3.3 配置 example 路径 `path = "examples/tui/main.rs"`
  - [ ] 3.4 添加 example 依赖（ratatui, crossterm, agent-diva-nano）

- [ ] Task 4: 验证独立运行 (AC: #3)
  - [ ] 4.1 执行 `cargo build --example tui` 验证编译
  - [ ] 4.2 执行 `cargo run --example tui` 验证启动
  - [ ] 4.3 验证 TUI 基本交互功能

- [ ] Task 5: 文档记录 (AC: #4)
  - [ ] 5.1 在复制后的文件顶部添加来源注释
  - [ ] 5.2 记录复制时间与来源路径

## Dev Notes

### 原始 TUI Example 文件清单

**来源路径：** `nano-workspace/agent-diva-nano/examples/tui/`

| 文件 | 大小 | 功能 |
|------|------|------|
| `main.rs` | 10.2 KB | 主入口，事件循环 |
| `app.rs` | 6.9 KB | App 状态管理 |
| `ui.rs` | 2.8 KB | UI 渲染逻辑 |
| `commands.rs` | 861 B | 命令解析（/quit, /clear 等） |
| `config.rs` | 2.1 KB | 配置加载/保存 |
| `manager.rs` | 1.1 KB | Agent 管理器 |
| `provider.rs` | 421 B | Provider 名称解析 |
| `wizard.rs` | 4.2 KB | 配置向导步骤 |

**总计：** 8 个文件，约 28 KB

### TUI 依赖清单

**需要在 Laputa/Cargo.toml 的 `[dev-dependencies]` 或 example 中配置：**

```toml
[dev-dependencies]
agent-diva-nano = { path = "../nano-workspace/agent-diva-nano" }
ratatui = "0.29"
crossterm = "0.28"
tokio = { version = "1", features = ["rt-multi-thread", "macros"] }
chrono = "0.4"
serde_json = "1.0"
```

**或使用 [[example]] 配置：**

```toml
[[example]]
name = "tui"
path = "examples/tui/main.rs"

[example.dependencies]
agent-diva-nano = { path = "../nano-workspace/agent-diva-nano" }
ratatui = "0.29"
crossterm = "0.28"
tokio = { version = "1", features = ["rt-multi-thread", "macros"] }
chrono = "0.4"
serde_json = "1.0"
```

### 复制注意事项

1. **保留原依赖关系**：复制后的 TUI example 暂时仍使用 agent-diva-nano 作为依赖
2. **来源注释**：在每个文件顶部添加来源说明
   ```rust
   //! TUI example - copied from agent-diva-nano
   //! Source: nano-workspace/agent-diva-nano/examples/tui/
   //! Copied: 2026-04-22
   //! Purpose: Reference implementation for Laputa integration
   ```
3. **路径调整**：如果 Cargo.toml 的相对路径不同，需调整 `agent-diva-nano` 的 path dependency

### 验收检查点

- `cargo build --example tui` 无编译错误
- `cargo run --example tui` 启动成功，显示 TUI 界面
- `/quit` 命令可正常退出
- 基本聊天功能可用（需配置 API Key）

### 后续演进方向（不在本 Story 茆内）

本 Story 只完成"复制"操作，后续演进方向：
- **Story 11.x**: 将 TUI example 改造为使用 Laputa 作为记忆后端
- **Story 12.x**: TUI 集成 Laputa Tool，展示记忆功能

### References

- [Source: nano-workspace/agent-diva-nano/examples/tui/](nano-workspace/agent-diva-nano/examples/tui/) - 原 TUI example
- [Source: agent-diva-nano TUI main.rs](nano-workspace/agent-diva-nano/examples/tui/main.rs) - 主入口代码

## Dev Agent Record

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### File List