# Laputa + agent-diva-nano 架构集成研究总结

> **研究日期**: 2026-04-22
> **研究人员**: John (PM Agent) + Qoder
> **研究目标**: 确认 Laputa 作为 agent-diva-memory 的实现定位，以及 agent-diva-nano 作为试验场的正确集成路径

---

## 一、核心问题澄清

### 1.1 用户原始需求

用户希望确认：
1. Laputa 作为"重要记忆源"合并进入 agent-loop 关键索引
2. 仅做兼容保留原先 agent-diva 的 memory.md
3. agent-diva-memory 由 Laputa 实现（而非独立 crate）
4. agent-diva-nano 作为试验场验证 MemoryProvider 自动注入

### 1.2 发现的架构混淆

当前存在以下理解偏差：

| 概念 | 当前 Story 10.3 设计 | 正确的架构方向 |
|------|---------------------|----------------|
| **Laputa 角色** | Tool 方案（LLM 主动调用） | **MemoryProvider 实现**（自动注入） |
| **agent-diva-memory** | 假设为独立 crate | **不存在**，由 Laputa 直接实现 MemoryProvider |
| **集成方式** | Phase 1 Tool 方案，Phase 2 MemoryProvider | MemoryProvider 是核心路径，Tool 是补充路径 |
| **MEMORY.md** | 未明确处理方式 | Laputa 导出 MEMORY.md 格式作为兼容层 |

---

## 二、关键文档发现

### 2.1 agent-diva-memory 原设计文档

位置：`docs/dev/archive/memory-evolution/2026-03-26-agent-diva-integrated-memory-design.md`

该文档定义了：
- 六层记忆架构：Session → Consolidation → Memory Store → Diary → Retrieval → Soul Evolution
- 新增 `agent-diva-memory` crate 的职责（第 10.2 节）
- 双分区日记机制（理性/感性）
- MemoryRecord 统一结构

**重要发现：agent-diva-memory crate 尚未实现，只有设计文档。**

### 2.2 MemoryProvider Trait 定义

位置：`docs/dev/hermes-integration/00-current-architecture-analysis.md:377`

```rust
#[async_trait]
pub trait MemoryProvider: Send + Sync {
    fn name(&self) -> &str;
    async fn initialize(&self, session_id: &str) -> Result<()>;
    fn is_available(&self) -> bool;
    fn get_tool_schemas(&self) -> Vec<Value>;
    fn system_prompt_block(&self) -> String;           // ← 唤醒包注入点
    async fn prefetch(&self, query: &str, session_id: &str) -> Result<String>;
    async fn sync_turn(&self, user_content: &str, assistant_content: &str) -> Result<()>;
    async fn handle_tool_call(&self, tool_name: &str, args: &Value) -> Result<String>;
    async fn on_session_end(&self, messages: &[Message]) -> Result<()>;
    async fn on_pre_compress(&self, messages: &[Message]) -> Result<String>;
}
```

**重要发现：MemoryProvider trait 只在设计文档中定义，尚未正式编码实现。**

### 2.3 UpspMemoryProvider 适配器示例

位置：`docs/dev/hermes-learning/02-upsp-integration.md:132`

展示了如何为 UPSP 实现 MemoryProvider trait 的适配器模式。

---

## 三、正确的架构定位

### 3.1 Laputa 替代 agent-diva-memory 的映射关系

| MemoryProvider 方法 | Laputa 对应能力 | 功能说明 |
|---------------------|-----------------|----------|
| `system_prompt_block()` | `wakeup.generate()` | 返回唤醒包 (<1200 tokens)，注入 agent context |
| `prefetch()` | `search.semantic()` + `recall.by_time_range()` | 语义检索 + 时间流召回 |
| `sync_turn()` | `diary.write()` | 每轮对话写入日记 |
| `handle_tool_call()` | MCP Tools | 用户主动调用记忆能力 |
| `on_session_end()` | `rhythm.trigger()` | 会话结束触发节律整理 |
| `on_pre_compress()` | (待设计) | 压缩前提取关键信号 |

### 3.2 架构图

```
┌──────────────────────────────────────────────────────────────────┐
│                   agent-diva-nano (试验场)                        │
│                                                                   │
│  ┌─────────────────────────────────────────────────────────────┐ │
│  │                    AgentLoop                                 │ │
│  │  context_assembly()                                          │ │
│  │      │                                                       │ │
│  │      ├─→ MemoryManager.system_prompt_block()                 │ │
│  │      │       │                                               │ │
│  │      │       ├─→ LaputaMemoryProvider.system_prompt_block()  │ │
│  │      │       │       └─→ Laputa.wakeup.generate() (<1200tk) │ │
│  │      │       │                                               │ │
│  │      │       └─→ BuiltinMemoryProvider (MEMORY.md 兼容)       │ │
│  │      │                                                       │ │
│  │      └─→ 构建 system prompt + 唤醒包                         │ │
│  └─────────────────────────────────────────────────────────────┘ │
│                                                                   │
│  ┌─────────────────────────────────────────────────────────────┐ │
│  │                    MemoryProvider trait                      │ │
│  │  (定义位置: agent-diva-core/src/memory/provider.rs 或 nano)  │ │
│  │                                                              │ │
│  │  - system_prompt_block() → 唤醒包注入                       │ │
│  │  - prefetch() → 语义/时间检索                                │ │
│  │  - sync_turn() → diary.write                                 │ │
│  │  - on_session_end() → rhythm 整理触发                        │ │
│  └─────────────────────────────────────────────────────────────┘ │
│                                                                   │
│  ┌─────────────────────────────────────────────────────────────┐ │
│  │              LaputaMemoryProvider (Laputa crate 实现)        │ │
│  │                                                              │ │
│  │  impl MemoryProvider for LaputaMemoryProvider {             │ │
│  │      fn system_prompt_block() → wakeup.generate()           │ │
│  │      async prefetch() → recall + search                     │ │
│  │      async sync_turn() → diary.write                        │ │
│  │      async on_session_end() → rhythm.trigger()              │ │
│  │  }                                                           │ │
│  └─────────────────────────────────────────────────────────────┘ │
│                                                                   │
│  ┌─────────────────────────────────────────────────────────────┐ │
│  │              Tool 方案 (保留作为补充路径)                     │ │
│  │  - LaputaWakeupTool                                          │ │
│  │  - LaputaRecallTool                                          │ │
│  │  - LaputaWriteTool                                           │ │
│  │  → 用户/LLM 主动调用记忆能力                                 │ │
│  └─────────────────────────────────────────────────────────────┘ │
└──────────────────────────────────────────────────────────────────┘

结论:
❌ 不需要: agent-diva-memory crate
✅ Laputa 直接实现 MemoryProvider trait
```

---

## 四、MEMORY.md 兼容层处理

### 4.1 原 MEMORY.md 设计

在原 agent-diva-memory 设计中，MEMORY.md 是：
- 存储长期记忆的 markdown 文件
- 由 BuiltinMemoryProvider 读取
- 直接注入 system prompt

### 4.2 正确的处理方式

在 Laputa 替代方案中：

| 处理方式 | 说明 |
|----------|------|
| **主数据源** | Laputa 的 `laputa.db` SQLite 数据库 |
| **MEMORY.md 生成** | Laputa 可导出 MEMORY.md 格式供其他组件读取 |
| **兼容层** | BuiltinMemoryProvider 改为读取 Laputa 导出的 MEMORY.md |
| **更新机制** | 每次 wakeup 时重新生成 MEMORY.md |

### 4.3 MEMORY.md 导出格式建议

```markdown
# Memory Context (Generated by Laputa)

## Identity
{从 identity.md 提取}

## Recent Events (Last 7 Days)
{按热度排序的近期记忆摘要}

## Weekly Capsule
{最新周级胶囊内容}

## Key Relations
{共振度 > 50 的关键关系}

---
Generated: {timestamp}
Token Count: {count} (target: <1200)
```

---

## 五、Story 10.3 AC 扩展建议

### 5.1 原 AC（仅 Tool 方案）

1. Tool trait 实现（LaputaWakeupTool 等）
2. JSON Schema 参数设计
3. ToolRegistry 注册方案
4. Cargo.toml 依赖配置
5. MCP/HTTP API 标记为 Phase 2

### 5.2 扩展后 AC（Tool + MemoryProvider）

```markdown
## Acceptance Criteria (扩展版)

### Part A: MemoryProvider 实现（核心集成点）
1. **Given** agent-diva-core 定义 MemoryProvider trait
2. **When** 实现 LaputaMemoryProvider
3. **Then** 包含以下内容：
   - MemoryProvider trait 定义（在 agent-diva-nano 或 agent-diva-core）
   - LaputaMemoryProvider struct 实现
   - system_prompt_block() → 调用 Laputa wakeup.generate()
   - prefetch() → 调用 Laputa recall/search
   - sync_turn() → 调用 Laputa diary.write()
4. **And** MemoryManager 能加载 LaputaMemoryProvider
5. **And** AgentLoop context assembly 自动注入唤醒包

### Part B: Tool 方案（辅助调用路径）
6. Tool trait 实现方案（LaputaWakeupTool 等）
7. ToolRegistry 注册方案
8. Cargo.toml 依赖配置（laputa 作为 dependency）

### Part C: 兼容层设计
9. MEMORY.md 导出方案（Laputa 数据 → MEMORY.md 格式）
10. 错误降级策略（Laputa 不可用时 fallback）

### Part D: 文档输出
11. 设计文档更新 `_bmad-output/implementation-artifacts/10-3-laputa-nano-integration-design.md`
12. 明确 MCP/HTTP API 整合为 Phase 2 后续工作
```

---

## 六、实施步骤建议

| 步骤 | 任务 | 产出位置 |
|------|------|----------|
| **Step 1** | 定义 MemoryProvider trait | `agent-diva-nano/src/memory/provider.rs` 或 `agent-diva-core/src/memory/provider.rs` |
| **Step 2** | 实现 LaputaMemoryProvider 适配器 | 调用 Laputa 的 wakeup/recall/diary 能力 |
| **Step 3** | 修改 AgentLoop 集成点 | context assembly 时调用 system_prompt_block() |
| **Step 4** | 实现 MEMORY.md 导出 | Laputa → MEMORY.md 格式生成器 |
| **Step 5** | 实现 Tool 方案（可选） | 作为备选调用路径 |
| **Step 6** | 端到端验证 | `init → diary write → wakeup → recall` 链路测试 |

---

## 七、关键决策确认

| 决策项 | 确认结果 |
|--------|----------|
| Laputa 是否替代 agent-diva-memory | ✅ 是，Laputa 直接实现 MemoryProvider trait |
| agent-diva-memory crate 是否需要 | ❌ 否，不需要单独 crate |
| MemoryProvider 定义位置 | `agent-diva-core/src/memory/provider.rs` 或 nano |
| MEMORY.md 是否保留 | ✅ 是，作为 Laputa 导出的兼容层 |
| Story 10.3 是否需要扩展 | ✅ 是，扩展包含 MemoryProvider 实现 |
| Tool 方案是否保留 | ✅ 是，作为补充调用路径 |

---

## 八、相关文档索引

| 文档 | 位置 | 内容 |
|------|------|------|
| agent-diva-memory 设计 | `docs/dev/archive/memory-evolution/2026-03-26-agent-diva-integrated-memory-design.md` | 六层架构、双分区日记 |
| MemoryProvider trait 定义 | `docs/dev/hermes-integration/00-current-architecture-analysis.md:377` | Trait 方法定义 |
| UpspMemoryProvider 示例 | `docs/dev/hermes-learning/02-upsp-integration.md:132` | 适配器实现参考 |
| Story 10.3 设计 | `_bmad-output/implementation-artifacts/10-3-laputa-nano-integration-design.md` | Tool 方案设计草图 |
| Laputa PRD | `_bmad-output/planning-artifacts/prd-laputa-agent-diva-integration.md` | 集成产品需求 |
| epics.md | `_bmad-output/planning-artifacts/epics.md` | Epic 10 Stories 定义 |
| nano 迁移指南 | `docs/dev/agent-diva-nano-migration-guide.md` | nano 改造指南 |

---

## 九、待办事项

1. [ ] 更新 Story 10.3 文档，扩展 AC 包含 MemoryProvider 实现方案
2. [ ] 创建 MemoryProvider trait 正式定义（编码）
3. [ ] 实现 LaputaMemoryProvider 适配器
4. [ ] 修改 AgentLoop context assembly 集成点
5. [ ] 实现 MEMORY.md 导出功能
6. [ ] 端到端验证链路测试

---

**研究状态**: ✅ 已完成
**下一步**: 等待用户决策更新 Story 10.3 或开始编码实现