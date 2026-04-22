# Story 10.3: Laputa → agent-diva-nano Tool 整合设计

Status: ready-for-dev

## Story

As a 系统架构师，
I want 完成 Laputa 作为 agent-diva-nano Tool 的整合设计，
So that Agent 可以通过 Tool trait 主动调用 Laputa 记忆能力，实现"有记忆的 AI 伙伴"体验。

## Acceptance Criteria

1. **Given** agent-diva-tools 定义了 `Tool` trait ([base.rs:8-19](agent-diva/agent-diva-tools/src/base.rs:8-19))
2. **When** 完成整合设计文档
3. **Then** 文档包含以下内容：
   - Laputa Tool 实现方案（Tool trait 方法定义）
   - 工具参数 JSON Schema 设计
   - ToolRegistry 注册方案
   - agent-diva-nano Cargo.toml 依赖配置
   - 与现有 Agent::send 的集成点设计
4. **And** 设计文档写入 `_bmad-output/planning-artifacts/laputa-nano-tool-design.md`
5. **And** MCP/HTTP API 整合方案标记为"Phase 2 后续工作"

## Tasks / Subtasks

- [ ] Task 1: 分析 Tool trait 与 Laputa 能力映射 (AC: #1, #3)
  - [ ] 1.1 分析 agent-diva-tools Tool trait 定义
  - [ ] 1.2 分析 Laputa 核心能力（wakeup/recall/write/mark）
  - [ ] 1.3 设计 Laputa Tool 工具集合命名方案
  - [ ] 1.4 确定工具粒度（单一工具 vs 多工具）

- [ ] Task 2: 设计 Laputa Tool 实现 (AC: #3)
  - [ ] 2.1 设计 `LaputaWakeupTool` - 唤醒包生成
  - [ ] 2.2 设计 `LaputaRecallTool` - 时间流检索
  - [ ] 2.3 设计 `LaputaWriteTool` - 日记写入
  - [ ] 2.4 设计 `LaputaMarkTool` - 情绪锚定/重要标记
  - [ ] 2.5 设计 `LaputaSearchTool` - 语义检索
  - [ ] 2.6 设计各工具的 JSON Schema 参数定义

- [ ] Task 3: 设计 ToolRegistry 注册方案 (AC: #3)
  - [ ] 3.1 分析 agent-diva-tools ToolRegistry 实现
  - [ ] 3.2 设计 Laputa 工具注册时机（Agent 初始化时）
  - [ ] 3.3 设计工具启用/禁用的 feature flag 方案

- [ ] Task 4: 设计依赖与 workspace 结构 (AC: #3)
  - [ ] 4.1 规划 Laputa 作为 agent-diva-nano dependency 的路径
  - [ ] 4.2 规划本地开发时的 path dependency 方案
  - [ ] 4.3 规划 feature flag 设计（`memory-laputa` feature）
  - [ ] 4.4 规划最小化构建闭包（不含 Laputa 的 fallback）

- [ ] Task 5: 设计 Agent 集成点 (AC: #3)
  - [ ] 5.1 分析 Agent::send 与 ToolRegistry 的交互方式
  - [ ] 5.2 设计工具调用触发时机（用户主动调用 vs 自动触发）
  - [ ] 5.3 设计错误处理与降级策略

- [ ] Task 6: 撰写设计文档 (AC: #4, #5)
  - [ ] 6.1 撰写完整设计文档
  - [ ] 6.2 明确 MCP/HTTP API 为 Phase 2 后续工作
  - [ ] 6.3 提交 Winston (架构师) 审核（可选）

## Dev Notes

### Tool trait 定义（[base.rs](agent-diva/agent-diva-tools/src/base.rs)）

```rust
#[async_trait]
pub trait Tool: Send + Sync {
    fn name(&self) -> &str;
    fn description(&self) -> &str;
    fn parameters(&self) -> Value;  // JSON Schema
    async fn execute(&self, args: Value) -> Result<String>;
    fn validate_params(&self, params: &Value) -> Vec<String>;
    fn to_schema(&self) -> Value;   // OpenAI function schema
}
```

### Laputa Tool 工具集设计草图

**工具命名：**
- `laputa_wakeup` - 生成唤醒包
- `laputa_recall` - 时间流检索
- `laputa_write` - 日记写入
- `laputa_mark` - 情绪锚定/重要标记
- `laputa_search` - 语义检索

**LaputaWakeupTool 示例：**
```rust
pub struct LaputaWakeupTool {
    laputa: Arc<LaputaService>,
}

#[async_trait]
impl Tool for LaputaWakeupTool {
    fn name(&self) -> &str { "laputa_wakeup" }

    fn description(&self) -> &str {
        "Generate a wakeup context package for session start. \
         Includes identity, recent events, weekly capsule, key relations. \
         Token budget ≤1200."
    }

    fn parameters(&self) -> Value {
        serde_json::json!({
            "type": "object",
            "properties": {
                "include_relations": {
                    "type": "boolean",
                    "description": "Include key relations in wakeup package",
                    "default": true
                },
                "max_tokens": {
                    "type": "integer",
                    "description": "Maximum token budget for wakeup package",
                    "default": 1200,
                    "minimum": 100,
                    "maximum": 2000
                }
            },
            "required": []
        })
    }

    async fn execute(&self, args: Value) -> Result<String> {
        let include_relations = args.get("include_relations")
            .and_then(|v| v.as_bool())
            .unwrap_or(true);
        let max_tokens = args.get("max_tokens")
            .and_then(|v| v.as_u64())
            .unwrap_or(1200) as usize;

        let wakeup = self.laputa.generate_wakeup(max_tokens, include_relations)
            .await
            .map_err(|e| ToolError::ExecutionFailed(e.to_string()))?;

        Ok(wakeup.to_string())
    }
}
```

**LaputaRecallTool 示例：**
```rust
pub struct LaputaRecallTool {
    laputa: Arc<LaputaService>,
}

#[async_trait]
impl Tool for LaputaRecallTool {
    fn name(&self) -> &str { "laputa_recall" }

    fn description(&self) -> &str {
        "Recall memories by time range. Returns temporal-first results \
         sorted by heat score. Supports day/week/month range queries."
    }

    fn parameters(&self) -> Value {
        serde_json::json!({
            "type": "object",
            "properties": {
                "time_range": {
                    "type": "string",
                    "description": "Time range to recall: 'today', 'week', 'month', or specific date range",
                    "examples": ["today", "week", "2026-04-01~2026-04-15"]
                },
                "limit": {
                    "type": "integer",
                    "description": "Maximum number of memories to return",
                    "default": 10,
                    "minimum": 1,
                    "maximum": 100
                },
                "include_content": {
                    "type": "boolean",
                    "description": "Include full content or just summaries",
                    "default": true
                }
            },
            "required": ["time_range"]
        })
    }

    async fn execute(&self, args: Value) -> Result<String> {
        let time_range = args.get("time_range")
            .and_then(|v| v.as_str())
            .unwrap_or("today");
        let limit = args.get("limit")
            .and_then(|v| v.as_u64())
            .unwrap_or(10) as usize;

        let memories = self.laputa.recall_by_time(time_range, limit)
            .await
            .map_err(|e| ToolError::ExecutionFailed(e.to_string()))?;

        Ok(serde_json::to_string_pretty(&memories)
            .unwrap_or_else(|_| "[]".to_string()))
    }
}
```

### ToolRegistry 注册方案

**参考 agent-diva-tools/registry.rs：**

```rust
// agent-diva-nano/src/agent.rs 或 lib.rs
use agent_diva_tools::ToolRegistry;
use laputa_tools::{LaputaWakeupTool, LaputaRecallTool, ...};

impl AgentBuilder {
    pub fn with_laputa_tools(mut self, laputa: Arc<LaputaService>) -> Self {
        self.registry.register(LaputaWakeupTool::new(laputa.clone()));
        self.registry.register(LaputaRecallTool::new(laputa.clone()));
        self.registry.register(LaputaWriteTool::new(laputa.clone()));
        self.registry.register(LaputaMarkTool::new(laputa.clone()));
        self.registry.register(LaputaSearchTool::new(laputa));
        self
    }
}
```

### Feature Flag 设计

```toml
# agent-diva-nano/Cargo.toml
[features]
default = []
memory-laputa = ["laputa"]

[dependencies]
laputa = { version = "0.1", optional = true }
agent-diva-tools = { version = "0.4" }
```

### 依赖路径规划

**方案 A: 本地开发 path dependency**
```toml
# agent-diva-nano/Cargo.toml (开发环境)
[dependencies.laputa]
version = "0.1"
optional = true
path = "../../Laputa"  # 相对路径引用
```

**方案 B: crates.io 发布后**
```toml
# agent-diva-nano/Cargo.toml (正式发布)
[dependencies.laputa]
version = "0.1"
optional = true
```

### Agent 集成点设计

**触发时机：**
- 用户主动调用：LLM 决定何时调用 `laputa_*` 工具
- 不自动注入：区别于 memory_provider 方案，Tool 方案由 Agent 自主决策

**与 Agent::send 的交互：**
```
Agent::send(message)
    │
    ├── 1. 消息发送到 DynamicProvider
    │
    ├── 2. LLM 分析是否需要调用工具
    │       → 可能调用 laputa_recall 查询历史
    │       → 可能调用 laputa_wakeup 获取上下文
    │       → 可能调用 laputa_write 记录内容
    │
    └── 3. 工具执行结果返回给 LLM
    │
    └── 4. LLM 生成最终回复
```

### MCP/HTTP API 整合方案（Phase 2）

**标记为后续工作，不在当前 Story 茆内：**
- `/api/memory/*` HTTP endpoints → Phase 2
- MCP Tools 映射 → Phase 2
- MemoryProvider trait 自动注入方案 → Phase 2

**当前阶段专注于：**
- Tool trait 实现 → Phase 1（当前）
- TUI example 复制 → Story 10.4

### References

- [Source: agent-diva-tools/src/base.rs](agent-diva/agent-diva-tools/src/base.rs) - Tool trait 定义
- [Source: agent-diva-tools/src/lib.rs](agent-diva/agent-diva-tools/src/lib.rs) - 工具导出
- [Source: agent-diva-tools/src/registry.rs](agent-diva/agent-diva-tools/src/registry.rs) - ToolRegistry 实现
- [Source: epics.md#Story 10.3](planning-artifacts/epics.md) - Story 定义与 AC
- [Source: prd.md](planning-artifacts/prd.md) - Laputa 产品需求

## Dev Agent Record

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### File List