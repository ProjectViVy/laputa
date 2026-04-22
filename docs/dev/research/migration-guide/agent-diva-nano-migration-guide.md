# agent-diva-nano 独立运行时迁移改造指南

> 本文档面向接手 agent-diva-nano 移植改造工作的同事，提供完整的背景、任务清单、注意事项和参考资料。

---

## 1. 项目背景

### 1.1 agent-diva-nano 是什么？

`agent-diva-nano` 是 Agent Diva 的 **精简版运行时**，定位为：

- **独立 HTTP Gateway + 控制平面**：提供 REST API 管理接口
- **核心 Agent 运行时**：包含完整的 AgentLoop 能力
- **轻量化部署方案**：适合嵌入式/边缘场景，无需完整 channels 集成
- **架构验证平台**：用于验证核心模块的解耦程度

### 1.2 当前位置

```
d:\VIVYCORE\newmemory\
├── agent-diva\               # 主仓库 (完整版)
│   ├── agent-diva-core\
│   ├── agent-diva-agent\
│   ├── agent-diva-providers\
│   ├── agent-diva-channels\
│   ├── agent-diva-tools\
│   └── ...
│
└── nano-workspace\           # 独立 workspace (精简版)
    └── agent-diva-nano\
        └── src\
            ├── runtime.rs    # 核心运行时入口
            ├── manager.rs    # API 管理器
            ├── handlers.rs   # HTTP 处理器
            └── ...
```

### 1.3 当前状态

根据 `sprint-status.yaml`：
- **Epic 10** 已创建，状态为 `backlog`
- Story `10-3-laputa-nano-integration-design` 等待启动
- 这是 MVP 验收阶段的最后一环

---

## 2. 你需要做什么（任务清单）

### Phase 1: 立即修复（阻塞问题）

#### 任务 1.1: 修复路径依赖配置

**问题描述**：当前 `nano-workspace/agent-diva-nano/Cargo.toml` 的路径依赖配置错误。

**当前错误配置**：
```toml
agent-diva-core = { path = "../../agent-diva-core", version = "0.4.0" }
```

**正确配置应为**：
```toml
agent-diva-core = { path = "../../agent-diva/agent-diva-core", version = "0.2.0" }
```

**修复清单**：
```toml
# nano-workspace/agent-diva-nano/Cargo.toml 需修改的依赖
agent-diva-core = { path = "../../agent-diva/agent-diva-core", version = "0.2.0" }
agent-diva-agent = { path = "../../agent-diva/agent-diva-agent", version = "0.2.0" }
agent-diva-providers = { path = "../../agent-diva/agent-diva-providers", version = "0.2.0" }
agent-diva-channels = { path = "../../agent-diva/agent-diva-channels", version = "0.2.0" }
agent-diva-tools = { path = "../../agent-diva/agent-diva-tools", version = "0.2.0" }
```

**验证命令**：
```bash
cd d:\VIVYCORE\newmemory\nano-workspace
cargo check
```

#### 任务 1.2: 统一版本号

| 模块 | 当前 nano 版本 | 实际主仓库版本 | 修改为 |
|-----|---------------|---------------|-------|
| agent-diva-core | 0.4.0 | 0.2.0 | 0.2.0 |
| agent-diva-agent | 0.4.0 | 0.2.0 | 0.2.0 |
| agent-diva-providers | 0.4.0 | 0.2.0 | 0.2.0 |
| agent-diva-channels | 0.4.0 | 0.2.0 | 0.2.0 |
| agent-diva-tools | 0.4.0 | 0.2.0 | 0.2.0 |

---

### Phase 2: 独立化改造

#### 任务 2.1: 实现核心 crate 发布

将以下核心模块发布到 crates.io（或保持 git 依赖）：

**发布顺序**（按依赖层级）：

```
L1 (无依赖) → L2 (依赖 L1) → L3 (依赖 L1+L2)

Phase 2.1: agent-diva-core (L1)
Phase 2.2: agent-diva-providers, agent-diva-memory (L2)
Phase 2.3: agent-diva-tools, agent-diva-agent (L3)
```

**发布准备工作**：
1. 检查 `Cargo.toml` 的 `publish` 字段
2. 确保 `license` 和 `repository` 字段正确
3. 运行 `cargo package --list` 检查打包内容
4. 执行 `cargo publish` 发布

#### 任务 2.2: 创建 git 依赖替代方案

如果暂不发布 crates.io，修改为 git 依赖：

```toml
# nano-workspace/agent-diva-nano/Cargo.toml
[dependencies]
agent-diva-core = { git = "https://github.com/ProjectViVy/agent-diva", branch = "main" }
agent-diva-agent = { git = "https://github.com/ProjectViVy/agent-diva", branch = "main" }
# ... 其他同理
```

---

### Phase 3: 功能裁剪与特性开关

#### 任务 3.1: 为 tools 模块添加 features

```toml
# agent-diva/agent-diva-tools/Cargo.toml
[features]
default = ["core"]
core = []
mcp = ["rust-mcp-sdk"]
full = ["core", "mcp"]

[dependencies]
rust-mcp-sdk = { version = "0.9", optional = true }
```

#### 任务 3.2: 为 channels 模块拆分

建议拆分为：
- `agent-diva-channels-core`: 通道抽象 trait
- `agent-diva-channels-telegram`: Telegram 通道（可选）
- `agent-diva-channels-discord`: Discord 通道（可选）
- `agent-diva-channels-slack`: Slack 通道（可选）
- ... 其他通道独立

**nano 版可仅依赖 channels-core**

---

### Phase 4: 创建极简运行时

#### 任务 4.1: 实现 minimal gateway

在 `nano-workspace/agent-diva-nano/src/` 创建：

```rust
// minimal_runtime.rs
pub fn run_minimal_gateway(config: MinimalConfig) -> Result<()> {
    // 仅包含:
    // 1. HTTP API server (handlers + server)
    // 2. AgentLoop (agent-diva-agent)
    // 3. DynamicProvider (agent-diva-providers)
    // 排除: channels, cron, mcp, memory
}
```

#### 任务 4.2: 双版本验证（重要决策）

根据团队决策 `模块集成双版本验证标准`，必须同时完成：

| 版本 | 描述 | 验证内容 |
|-----|------|---------|
| **缝合版** | 直接改造 agent-diva 的 memory 模块 | 完整 TUI 体验验证 |
| **独立版** | 新建最小化项目引入 Laputa crate | 轻量 TUI + 核心能力验证 |

---

## 3. 注意事项（关键风险点）

### 3.1 路径依赖陷阱

**风险**：路径依赖在不同操作系统上行为可能不一致

**规避**：
- Windows: 使用 `\` 或 `/` 均可，Cargo 自动处理
- 确保相对路径从 `nano-workspace` 出发计算正确

**路径计算示例**：
```
nano-workspace/               # 起点
  ├── agent-diva-nano/
  │   └── Cargo.toml          # 这里配置 ../../agent-diva/agent-diva-core
  │                           # 解析: nano-workspace/../agent-diva/agent-diva-core
  │                           # 实际: newmemory/agent-diva/agent-diva-core ✓
```

### 3.2 版本同步风险

**风险**：主仓库版本更新后，nano 依赖未同步

**规避**：
- 建议使用 git 依赖 + commit hash 锁定
- 或发布后使用 crates.io 版本锁定

### 3.3 Channels 模块编译风险

**风险**：channels 包含大量第三方 SDK，编译时间长

**规避**：
- nano 版默认禁用所有 channels
- 添加条件编译开关

### 3.4 Memory 模块数据库风险

**风险**：`rusqlite` bundled 编译耗时，且可能跨平台问题

**规避**：
- nano 版可使用内存存储替代
- 或提供 `no-sqlite` feature

### 3.5 MCP SDK 依赖风险

**风险**：`rust-mcp-sdk` 版本敏感，可能与主项目冲突

**规避**：
- 锁定版本号
- 添加 optional feature

---

## 4. 参考资料

### 4.1 项目文件索引

| 文件 | 位置 | 内容 |
|-----|------|------|
| 主仓库 Cargo.toml | `agent-diva/Cargo.toml` | workspace 配置 |
| nano Cargo.toml | `nano-workspace/agent-diva-nano/Cargo.toml` | **需修改** |
| 核心运行时 | `nano-workspace/agent-diva-nano/src/runtime.rs` | AgentLoop 启动逻辑 |
| 管理器 | `nano-workspace/agent-diva-nano/src/manager.rs` | API 命令处理 |
| HTTP 处理器 | `nano-workspace/agent-diva-nano/src/handlers.rs` | REST API 实现 |
| Sprint 状态 | `_bmad-output/implementation-artifacts/sprint-status.yaml` | Epic 10 任务状态 |

### 4.2 依赖树结构图

```
agent-diva-nano (当前依赖)
│
├── agent-diva-core ──────────────────────────── [L1] 无依赖
│   ├── bus (消息总线)
│   ├── config (配置加载)
│   ├── cron (定时任务)
│   ├── session (会话管理)
│   └── soul (人格系统)
│
├── agent-diva-providers ─────────────────────── [L2] 依赖 core
│   ├── reqwest (HTTP)
│   └── DynamicProvider (热切换)
│
├── agent-diva-memory ────────────────────────── [L2] 依赖 core
│   └── rusqlite (bundled) ⚠️ 编译耗时
│
├── agent-diva-tools ─────────────────────────── [L2] 依赖 core+memory
│   └── rust-mcp-sdk (MCP 协议)
│
├── agent-diva-agent ─────────────────────────── [L3] 依赖 core+memory+providers+tools
│   └── AgentLoop (核心运行时)
│
└── agent-diva-channels ──────────────────────── [L3] 依赖 core+providers
    ├── teloxide (Telegram)
    ├── slack-morphism (Slack)
    ├── tokio-tungstenite (WebSocket)
    └── lettre/imap (Email)
    ⚠️ 12+ 第三方 SDK，编译时间长
```

### 4.3 极简版建议依赖图

```
agent-diva-nano-minimal (建议)
│
├── agent-diva-core ✓ 必需
├── agent-diva-providers ✓ 必需
├── agent-diva-agent ✓ 必需
│
├── agent-diva-tools-core ✓ 可选 (无 MCP)
├── agent-diva-memory ⚠️ 可选 (或用内存存储)
│
└── agent-diva-channels ✗ 排除
└── agent-diva-tools-mcp ✗ 排除
└── cron ✗ 排除
```

### 4.4 相关决策记忆

| 决策 | 关键词 | 内容摘要 |
|-----|--------|---------|
| 双版本验证标准 | 模块集成、双版本验证 | 必须同时完成缝合版+独立版验证 |
| 版本跟踪机制 | Laputa、mempalace-rs | Laputa 需跟踪 mempalace-rs 版本 |

---

## 5. 执行检查清单

### 5.1 Phase 1 完成标准

```bash
# 在 nano-workspace 目录执行
cd d:\VIVYCORE\newmemory\nano-workspace

# 1. 路径修复后
cargo check          # 应通过无错误

# 2. 编译测试
cargo build          # 应成功生成 target/debug/

# 3. 运行测试
cargo test           # 应通过所有单元测试
```

### 5.2 Phase 2 完成标准

- 核心模块可独立编译（脱离主仓库路径）
- 可通过 git 依赖或 crates.io 依赖引入

### 5.3 Phase 3 完成标准

- features 开关工作正常
- `cargo build --no-default-features` 可编译精简版

### 5.4 Phase 4 完成标准

- 双版本验证均通过
- minimal gateway 可启动 HTTP 服务
- AgentLoop 可响应请求

---

## 6. 快速上手命令

```bash
# 进入 nano workspace
cd d:\VIVYCORE\newmemory\nano-workspace

# 查看依赖树
cargo tree

# 检查编译
cargo check

# 完整构建
cargo build --release

# 运行测试
cargo test

# 查看 Cargo.lock 中的实际依赖版本
cat Cargo.lock | grep "agent-diva"
```

---

## 附录：问题排查指南

### Q1: cargo check 报路径找不到

**检查步骤**：
1. 确认 `agent-diva` 目录存在于 `newmemory/` 下
2. 确认路径配置为 `../../agent-diva/agent-diva-*`
3. 运行 `ls ../../agent-diva` 验证路径

### Q2: 编译超时或卡住

**可能原因**：
- `rusqlite bundled` 编译 sqlite3
- channels 的第三方 SDK 编译

**解决**：
- 先禁用 channels 和 memory，验证核心模块
- 分阶段引入依赖

### Q3: 版本冲突

**检查**：
```bash
cargo tree -d  # 显示重复依赖
```

**解决**：
- 统一所有 `agent-diva-*` 版本为 `0.2.0`
- 确保 workspace.dependencies 版本一致

---

**文档版本**: v1.0
**创建日期**: 2026-04-21
**维护者**: Qoder Agent
**相关 Epic**: Epic 10 - MVP验收与agent-diva-nano整合准备