---
stepsCompleted:
  - step-01-init
  - step-02-context-analysis
  - advanced-elicitation-architecture-decision-records
  - step-03-heat-mechanism-design
  - step-04-core-adr
  - step-05-implementation-patterns
  - step-06-structure-boundaries
  - step-07-validation
workflowType: 'architecture'
lastStep: 8
status: 'complete'
completedAt: '2026-04-13'
inputDocuments:
  - D:/VIVYCORE/newmemory/_bmad-output/planning-artifacts/prd.md
  - D:/VIVYCORE/newmemory/Laputa/DECISIONS.md
  - D:/VIVYCORE/newmemory/Laputa/AGENTS.md
  - D:/VIVYCORE/newmemory/brain-memory-system-design.md
  - D:/VIVYCORE/newmemory/mempalace-rs/README.md
workflowType: 'architecture'
project_name: '天空之城 (Laputa)'
user_name: '大湿'
date: '2026-04-13'
---

# Architecture Decision Document - 天空之城 (Laputa)

_This document builds collaboratively through step-by-step discovery. Sections are appended as we work through each architectural decision together._

---

## 初始化完成

**文档已创建**: `_bmad-output/planning-artifacts/architecture.md`

**已发现的输入文档**:

| 文档 | 状态 | 说明 |
|------|------|------|
| prd.md | ✅ 已加载 | Product Requirements Document，311 行 |
| DECISIONS.md | ✅ 已加载 | 12 个核心决策，248 行 |
| AGENTS.md | ✅ 已加载 | 项目规范，244 行 |
| brain-memory-system-design.md | ✅ 已加载 | 原始设计文档，1340 行 |
| mempalace-rs/README.md | ✅ 已加载 | 代码基线参考 |

**PRD 验证**: ✅ 已确认存在

**项目上下文**:
- 项目名称: 天空之城 (Laputa)
- 代码基线: mempalace-rs (纯 Rust，197 tests 通过)
- 核心理念: 像人一样记忆——会判断、会遗忘、会情绪标记、会睡眠整理
- 目标宿主: agent-diva (但必须可移植)

---

## Step 2: 项目上下文分析 (Advanced Elicitation 强化版)

_通过 Party Mode 多代理协作完成架构决策分析。参与者：Winston (架构师)、John (PM)、Amelia (开发者)、Murat (测试架构师)。_

---

### 2.1 MVP 功能优先级排序

**来源**: John (PM) 分析

| 优先级 | 功能 | 理由 |
|--------|------|------|
| **P0** | 主体初始化 | 没有用户就没有体验，基础中的基础 |
| **P0** | 日记输入层 | 没有输入=没有数据=没有价值验证 |
| **P1** | 检索层（时间流+语义） | **核心验证点**。找回那一刻是"啊哈时刻" |
| **P1** | 唤醒上下文层 (<1200 tokens) | 检索的"交付质量"，决定用户是否觉得"懂我" |
| **P2** | 节律整理层（周级胶囊） | 有价值但可简化，MVP 用最基础的时间分桶 |
| **P3** | 生命周期治理层（热度、归档） | 优化层，非核心体验 |

**核心验证点结论**: 检索层是 MVP 成败的关键。用户旅程：输入 → 存储 → **找回**。找回那一刻证明系统价值。

---

### 2.2 mempalace-rs 模块复用/扩展/重构边界

**来源**: Winston (架构师) + Amelia (开发者) 分析

| 模块 | 决策 | 理由 | 风险等级 |
|------|------|------|----------|
| `storage.rs` Layer0/1 | **复用，只读扩展** | 核心数据路径，197 tests 覆盖，稳定基线 | 🟢 低 |
| `dialect.rs` EMOTION_CODES | **直接沿用** | 情绪编码核心，无需修改 | 🟢 低 |
| `knowledge_graph.rs` | **扩展点** | 热度衰减和睡眠整理集成点 | 🟡 中 |
| `searcher.rs` | **扩展，加热度排序因子** | 加可选参数 `sort_by`，默认行为不变 | 🟡 中 |
| `mcp_server.rs` | **扩展边界** | MCP 协议本身就是扩展点，新 tool 暴露 L4 | 🟢 低 |
| `diary.rs` | **复用+事件钩子** | 可能需验证热度排序对日记内容选择的影响 | 🟡 中 |

**架构建议 (Winston)**:

```
┌─────────────────────────────────────────┐
│  外部接口层                              │
│  CLI | MCP | Python API                 │
├─────────────────────────────────────────┤
│  统一抽象层                              │
│  MemoryOperation trait                  │
│  - store() / recall() / archive()       │
│  - with_heat_sorting() / with_l4()      │
├─────────────────────────────────────────┤
│  核心引擎 (mempalace-rs)                │
│  storage | graph | search | diary       │
└─────────────────────────────────────────┘
```

**关键原则 (Amelia)**:
- 新 Layer 作为独立 struct 而非 enum variant，隔离存储路径
- 新增 `#[cfg(feature = "laputa")]` feature flag 隔离天空之城专属代码
- 现有测试不动，新增测试文件 `test_heat.rs`, `test_layer4.rs`, `test_wakepack.rs`

---

### 2.3 L4 归档层设计决策

**来源**: Winston (架构师) 分析

**风险结论**: 有风险，但可控。

| 风险点 | 描述 | 解决方案 |
|--------|------|----------|
| 层级边界定义 | L3→L4 迁移逻辑位置 | 独立 `Archiver` 组件，不侵入存储层核心 |
| 访问透明性 | L4 数据是否透明召回 | 惰性加载模式，默认查询只走 L0-L3 |
| 知识图谱一致性 | 记忆归档后图节点引用 | 保留节点，加 `location: Archive` 标记 |

**架构决策**: L4 作为独立 store，与 L0-L3 共享知识图谱。记忆项迁移时图节点保留，只改 location 标记。

---

### 2.4 热度机制实现路径

**来源**: Amelia (开发者) 分析

**热度字段扩展**:
```rust
MemoryRecord → 添加 heat: f64 字段
             → 添加 last_accessed: DateTime<Utc>
             → 添加 access_count: u32
```

**衰减公式**: `heat = base_score * e^(-decay_rate * days_since_access) * log(access_count + 1)`

**热度计算策略 (Winston)**: 混合模式
- 高频访问记忆（最近24小时）→ 实时更新
- 低频记忆 → 批量更新

**风险警告 (Murat)**: 热度变化涉及**状态机转换**，不是简单 if-else。需用状态机测试框架覆盖所有转换路径。

---

### 2.5 唤醒包 <1200 tokens 约束

**来源**: Amelia (开发者) + John (PM) 分析

**技术实现 (Amelia)**:
```rust
struct WakePack {
    memories: Vec<MemorySummary>,  // 按热度 Top-K
    token_count: usize,             // 实时累计
}

// token 估算 (中文 ~1.5 chars/token)
fn estimate_tokens(&self) -> usize {
    (self.content.chars().count() as f64 / 1.5).ceil() as usize
}
```

**待澄清问题 (John)**: <1200 tokens 这个数字从哪来的？是技术约束还是产品假设？需大湿明确。

---

### 2.6 共振度/Valence/Arousal 优先级

**来源**: John (PM) 分析

| 概念 | 优先级 | WHY |
|------|--------|-----|
| **共振度 (Resonance)** | P0 | "像人一样记忆"的核心——记忆之间会共鸣。没有它就是普通数据库 |
| **Valence** | P1 | 情感正负决定"好记忆还是坏记忆"，影响检索排序 |
| **Arousal** | P2 | MVP 数据不足，可简化为二元（高/低） |

---

### 2.7 测试架构风险评估

**来源**: Murat (测试架构师) 分析

**🚨 最高风险**: 测试隔离性破坏（风险等级 8/10）

| 风险 | 描述 | 影响 |
|------|------|------|
| 共享状态污染 | 新增 L4 测试可能破坏隔离 | CI 不稳定，Flaky tests |
| Fixture 耦合 | 197 tests fixture 隐式依赖特定状态 | 新测试失败难以定位 |
| 并发安全 | cargo test 默认并行，归档涉及文件锁 | 随机失败 |

**缓解策略**:
```rust
// 显式隔离策略
#[cfg(test)]
mod tests {
    #[serial]  // serial_test 宏强制串行
    #[test]
    fn test_archive_flow() { ... }
}
```

**L4 归档层 Fixture 影响**（破坏性风险 7/10）:
- Schema 迁移 → 所有 `setup_test_db()` fixture 需改动
- 生命周期假设 → 现有测试可能假设"数据永存"，需 mock 时间流逝

**建议 Fixture 架构**:
```rust
trait TestFixture {
    fn memory_only() -> Self;      // 纯内存，快速，适合单元测试
    fn with_tempdir() -> Self;     // 真实文件系统，适合归档测试
    fn with_archive_layer() -> Self; // 包含 L4 配置
}

struct TimeMachine {
    fn advance_days(days: u64);  // 模拟热度衰减
    fn freeze_time();            // 固定时间点测试
}
```

---

### 2.8 MVP 端到端链路测试覆盖

**来源**: Murat (测试架构师) 分析

**黄金路径**: `daily diary → weekly capsule → wakeup context → answer`

**必测场景矩阵**:

| 场景 | 正向 | 边界 | 故障 | 优先级 |
|------|------|------|------|--------|
| Daily Diary 写入 | ✓ | 空内容、超长内容 | 存储满 | P0 |
| Weekly Capsule 生成 | ✓ | 无数据周、数据溢出 | 合并失败 | P0 |
| Wakeup Context 构建 | ✓ | 冷启动无历史 | 索引损坏 | P0 |
| Answer 检索 | ✓ | 无匹配、多匹配 | 超时 | P0 |

---

### 2.9 关键分歧点与待决策项

**Party Mode 发现的争议**:

| 争议点 | 观点 A | 观点 B | 需大湿决策 |
|--------|--------|--------|------------|
| **热度机制 MVP 状态** | John: 可推迟到 Phase 2 | Winston/Murat: 核心决策逻辑，需状态机测试 | 🔴 高 |
| **L4 架构边界** | Winston: 独立 Archiver 组件 | Amelia: feature flag 隔离 | 🟡 中 |
| **<1200 tokens 约束来源** | John 追问：技术约束？产品假设？ | - | 🟡 中 |
| **生命周期治理层是否砍掉** | John: 建议砍，第100天的事 | Winston: 热度影响决策逻辑 | 🔴 高 |

---

### 2.10 风险优先级汇总

```
风险                        │ 影响 │ 概率 │ 优先处理
────────────────────────────┼──────┼──────┼──────────
测试隔离性破坏               │ 高   │ 高   │ 🔴 P0
热度边界状态转换             │ 高   │ 中   │ 🔴 P0
L4 Fixture 重构成本          │ 中   │ 高   │ 🟡 P1
<1200 tokens 约束来源不明    │ 中   │ 低   │ 🟡 P1
CI 并发安全                  │ 中   │ 中   │ 🟢 P2
```

---

## 待决策项清单

| ID | 决策项 | 来源 | 状态 |
|----|--------|------|------|
| AD-001 | 热度机制 MVP 是否必须？ | John vs Winston/Murat | ✅ **必须** - Phase 1 必须上 |
| AD-002 | <1200 tokens 约束来源 | John 追问 | ✅ **产品设计约束**（经验值，可调整） |
| AD-003 | 生命周期治理层是否砍掉 | John 建议 | ✅ **保留** - 热度衰减+睡眠整理+用户干预+归档候选 |
| AD-004 | L4 架构边界 | Winston vs Amelia | ✅ **独立 Archiver 组件** |
| HEAT-01 | 热度存储格式 | Murat 建议 | ✅ **i32 整数**（放大100倍，消除浮点精度） |
| HEAT-02 | Feature Flag 方式 | Amelia 建议 | ✅ **运行时开关**（ABI兼容性） |
| HEAT-03 | 热度机制终极目标 | John 追问 | ✅ **Phase 1 优先帮记住**，归档只标记不执行 |
| HEAT-04 | 归档候选通知方式 | John 建议 | ✅ **用户主动查询**（Phase 1 归档不执行，无需通知） |

---

## Step 3: 热度机制架构设计（已决策）

_基于 Party Mode 多代理协作讨论结果，大湿按最佳实践决策。_

---

### 3.1 热度存储格式

**决策**: `i32` 整数存储（放大100倍）

```rust
// 存储格式
pub struct MemoryRecord {
    pub heat_i32: i32,  // 0-10000 对应 0.00-100.00
}

// 读写转换
fn heat_from_i32(v: i32) -> f64 { v as f64 / 100.0 }
fn heat_to_i32(v: f64) -> i32 { (v * 100.0).round() as i32 }
```

**理由**:
- 消除浮点精度问题，边界测试更稳定
- 状态机转换不会因精度误差产生幽灵状态
- 100倍精度足够（80.00 vs 80.01 的边界差异）

---

### 3.2 热度计算触发策略

**决策**: 混合模式（Winston 建议）

| 场景 | 触发方式 | 操作 |
|------|---------|------|
| **读取时** | 同步更新 | `access_count += 1`, `last_accessed = now()` |
| **写入时** | 同步计算 | 重新计算 heat 值 |
| **后台定时** | 批量衰减 | 每小时扫描低频记忆，批量更新 |

**数据流**:
```
用户查询 → touch() 更新 access_count + last_accessed
         → 返回结果（不计算 heat）

定时任务（每小时）→ 筛选 access_count > 0 的记忆
                  → 重新计算 heat
                  → 批量写入

归档检查（每日）→ 对 heat < 20 的记忆标记归档候选（Phase 1 只标记）
```

---

### 3.3 热度模块架构

**决策**: 独立 HeatService 模块

```
src/
├── heat/
│   ├── mod.rs           # 模块入口
│   ├── service.rs       # HeatService - 核心计算逻辑
│   ├── index.rs         # HeatIndex - 阈值判断 + 热度缓存
│   └── decay.rs         # 衰减公式实现
│   └── config.rs        # HeatConfig - 配置化阈值
└── storage/
    └── storage.rs       # 调用 HeatService，不内嵌逻辑
```

**接口设计**:
```rust
pub struct HeatService {
    config: HeatConfig,
}

impl HeatService {
    fn calculate(&self, record: &MemoryRecord) -> i32;
    fn calculate_batch(&self, records: &mut [MemoryRecord]);
    fn should_archive(&self, heat: i32) -> bool;
}

pub struct HeatConfig {
    pub hot_threshold: i32,      // 默认 8000 (80.00)
    pub warm_threshold: i32,     // 默认 5000 (50.00)
    pub cold_threshold: i32,     // 默认 2000 (20.00)
    pub decay_rate: f64,         // 衰减系数
    pub enabled: bool,           // 运行时开关
}
```

---

### 3.4 热度阈值与状态

**决策**: 四区间状态机

| 热度范围 | 状态 | 处理 |
|---------|------|------|
| > 8000 (80) | 🔥 锁定 | 不衰减，高亮/置顶 |
| 5000-8000 (50-80) | 📌 正常 | 缓慢衰减 |
| 2000-5000 (20-50) | 💤 归档候选 | 标记但不执行归档（Phase 1） |
| < 2000 (20) | 📦 自动打包候选 | 标记但不执行（Phase 1） |

**Phase 1 约束**: 归档只标记，不自动执行。Phase 2 再实现完整归档流程。

---

### 3.5 用户干预接口

**决策**: CLI/API 命令

| 命令 | 热度影响 | 说明 |
|------|----------|------|
| `--important` | heat = 9000 并锁定 | 用户明确"很重要" |
| `--forget` | heat = 0，标记归档候选 | 用户主动遗忘（但保留可撤销） |
| `--emotion anchor` | heat += 2000，衰减率减半7天 | 情感锚点"保鲜" |

---

### 3.6 用户感知方式

**决策**: 不显示数值，只显示状态图标

| 状态 | 图标 | UX 表现 |
|------|------|--------|
| 锁定（>80） | 🔥 | 高亮/置顶 |
| 正常（50-80） | 📌 | 正常展示 |
| 归档候选（20-50） | 💤 | 灰显/折叠 |
| 自动打包候选（<20） | 📦 | 底部提示区 |

**Phase 1 不提供周期性报告**。用户可通过 CLI/API 查询归档候选列表。

---

### 3.7 热度测试策略

**决策**: 状态机测试优先（Murat 建议）

**必测场景（P0）**:

| 场景ID | 描述 | 验证点 |
|--------|------|--------|
| SM-01 | 锁定态衰减至解锁 | 状态转换正确 |
| SM-02 | 边界穿越：8000 → 7999 | 锁定→正常 |
| SM-03 | 边界穿越：5000 → 4999 | 正常→归档候选 |
| SM-04 | 边界穿越：2000 → 1999 | 归档候选→自动打包 |
| SM-08 | 快速连续状态转换 | 无幽灵状态 |

**并发测试**:
```rust
#[test]
fn test_concurrent_heat_increment() {
    // 100线程同时增加热度
    // 验证：最终热度正确，无丢失更新
}
```

**TimeMachine Fixture**:
```rust
struct TimeMachine {
    fn advance(&mut self, seconds: u64);  // 模拟热度衰减
    fn freeze(&mut self);                 // 精确边界测试
}
```

---

## Step 4: 核心架构决策记录 (ADR)

_基于前置讨论，汇总所有架构决策并补充剩余关键类别。_

---

### 4.1 决策优先级分析

**已决策项（Critical - 阻塞实现）**:

| ADR ID | 决策 | 状态 | 影响 |
|--------|------|------|------|
| ADR-001 | 热度机制 Phase 1 必须上 | ✅ 已决策 | 核心功能 |
| ADR-002 | 热度存储 i32 整数 | ✅ 已决策 | 数据模型 |
| ADR-003 | HeatService 独立模块 | ✅ 已决策 | 模块架构 |
| ADR-004 | 混合触发策略 | ✅ 已决策 | 性能 |
| ADR-005 | L4 独立 Archiver 组件 | ✅ 已决策 | 归档架构 |
| ADR-006 | mempalace-rs 继承边界 | ✅ 已决策 | 代码基线 |
| ADR-007 | MVP 功能优先级 | ✅ 已决策 | 实现顺序 |

**待补充项（Important - 影响架构）**:

| ADR ID | 决策 | 状态 | 影响 |
|--------|------|------|------|
| ADR-008 | 数据存储架构 | ⏳ 待确认 | SQLite + usearch |
| ADR-009 | API 设计模式 | ⏳ 待确认 | CLI/MCP/Python |
| ADR-010 | 错误处理标准 | ⏳ 待确认 | 统一错误枚举 |
| ADR-011 | 配置管理策略 | ⏳ 待确认 | config.toml |
| ADR-012 | 测试架构策略 | ⏳ 待确认 | serial_test + Fixture |

---

### 4.2 数据架构决策

**ADR-008: 数据存储架构**

| 决策项 | 选择 | 理由 | 版本 |
|--------|------|------|------|
| **主存储** | SQLite | mempalace-rs 已验证，轻量级，无服务依赖 | 3.x |
| **向量索引** | usearch | 高性能向量检索，Rust native | 2.x |
| **知识图谱** | 内存 + triples 表 | 关系时间线，低延迟查询 | N/A |
| **归档格式** | SQLite dump | 沿用 mempalace-rs 结构 | D-005 |

**数据模型扩展**:
```rust
pub struct MemoryRecord {
    // 继承字段
    pub id: Uuid,
    pub content: String,
    pub layer: Layer,       // L0-L3 (新增 L4)
    pub emotion: EmotionCode,  // 沿用 dialect.rs
    pub created_at: DateTime<Utc>,
    
    // 新增字段
    pub heat_i32: i32,         // 热度（100倍精度）
    pub last_accessed: DateTime<Utc>,
    pub access_count: u32,
    pub is_archive_candidate: bool,  // 归档候选标记
    pub emotion_valence: i32,        // -100~+100（UPSP 融合）
    pub emotion_arousal: u32,        // 0~100（UPSP 融合）
}
```

---

### 4.3 API & 接口决策

**ADR-009: API 设计模式**

| 接口 | MVP 状态 | 设计模式 | 用途 |
|------|---------|---------|------|
| **CLI** | P0 必须 | 命令行参数 + 子命令 | 本地开发/测试 |
| **MCP** | P0 必须 | JSON-RPC 2.0 + Tools | AI agent 集成 |
| **Python API** | Phase 2 | PyO3 绑定 | 跨语言集成 |

**统一抽象层 (Winston 建议)**:
```rust
pub trait MemoryOperation {
    fn store(&mut self, record: MemoryRecord) -> Result<Uuid>;
    fn recall(&self, query: RecallQuery) -> Result<Vec<MemoryRecord>>;
    fn archive(&mut self, id: Uuid) -> Result<()>;  // Phase 2
    
    // 扩展方法
    fn with_heat_sorting(self, enabled: bool) -> Self;
    fn with_l4_query(self, include_archive: bool) -> Self;
}
```

**ADR-010: 错误处理标准**

```rust
pub enum MemoryError {
    StorageError(String),
    NotFound(Uuid),
    ValidationError(String),
    HeatThresholdError(i32),  // 状态机边界错误
    ArchiveError(String),    // L4 归档错误
    ConfigError(String),
}

// CLI: exit code + stderr
// MCP: JSON-RPC error object
// Python: Exception subclass
```

---

### 4.4 配置管理决策

**ADR-011: 配置管理策略**

```toml
# config.toml 结构
[heat]
enabled = true
hot_threshold = 8000      # 80.00
warm_threshold = 5000     # 50.00
cold_threshold = 2000     # 20.00
decay_rate = 0.1          # 衰减系数
update_interval_hours = 1 # 热度批量更新间隔

[archive]
enabled = false           # Phase 1 禁用自动归档
archive_threshold = 2000  # < 20 自动标记候选
check_interval_days = 1   # 归档检查间隔

[storage]
db_path = "./laputa.db"
vector_dim = 384          # embedding 维度
usearch_path = "./laputa.usearch"

[wakeup]
max_tokens = 1200         # 唤醒包 token 上限
include_identity = true
include_recent_events = true
include_resonance = true
```

---

### 4.5 测试架构决策

**ADR-012: 测试架构策略**

| 策略 | 选择 | 理由 |
|------|------|------|
| **隔离模式** | serial_test crate | 避免 Flaky tests |
| **Fixture 分层** | TestFixture trait | memory_only / with_tempdir / with_archive |
| **时间模拟** | TimeMachine fixture | 模拟热度衰减 |
| **并发测试** | Arc + 多线程 | 验证热度更新原子性 |
| **覆盖率目标** | 197+ tests 继承 + 新增 | 不破坏现有测试 |

**测试文件结构**:
```
tests/
├── test_heat.rs           # 热度机制测试（状态机）
├── test_layer4.rs         # L4 归档层测试
├── test_wakepack.rs       # 唤醒包生成测试
├── test_integration.rs    # E2E 链路测试
└── fixtures/
    ├── memory_only.rs     # 纯内存 fixture
    ├── with_tempdir.rs    # 真实文件系统
    └── time_machine.rs    # 时间模拟工具
```

---

### 4.6 决策影响分析

**实现顺序（基于 MVP 优先级）**:

```
Phase 1 实现顺序:
├── 1. 主体初始化 (P0)
│   └── ADR-001: HeatService 初始化
├── 2. 日记输入层 (P0)
│   └── ADR-006: diary.rs 扩展 + heat 字段
├── 3. 检索层 (P1 - 核心验证点)
│   └── ADR-003: searcher.rs 加热度排序
├── 4. 唤醒上下文层 (P1)
│   └── ADR-004: WakePack + HeatIndex Top-K
├── 5. 节律整理层 (P2)
│   └── Weekly Capsule 生成
└── 6. 生命周期治理层 (P3 - Phase 1 只标记)
    └── ADR-002: 归档候选标记
```

**跨组件依赖图**:

```
HeatService (核心)
    │
    ├──→ Storage (heat_i32 字段写入)
    ├──→ Searcher (热度排序因子)
    ├──→ WakePack (Top-K 选择)
    └──→ KnowledgeGraph (共振度更新)
        │
        └──→ Archiver (Phase 2 - L4 迁移)
```

---

### 4.7 技术栈版本锁定

| 技术 | 版本 | 来源 | 状态 |
|------|------|------|------|
| Rust | 1.75+ | mempalace-rs | ✅ 已验证 |
| SQLite | 3.x | mempalace-rs | ✅ 已验证 |
| usearch | 2.x | mempalace-rs | ✅ 已验证 |
| tokio | 1.x | mempalace-rs | ✅ 已验证 |
| serde | 1.x | mempalace-rs | ✅ 已验证 |
| chrono | 0.4 | mempalace-rs | ✅ 已验证 |
| serial_test | 2.x | 新增 | ⏳ 待引入 |
| uuid | 1.x | mempalace-rs | ✅ 已验证 |

---

## 决策汇总表

| ADR ID | 决策项 | 选择 | 状态 | Phase |
|--------|--------|------|------|-------|
| ADR-001 | 热度机制 MVP | 必须上 | ✅ 已决策 | Phase 1 |
| ADR-002 | 热度存储格式 | i32 整数 | ✅ 已决策 | Phase 1 |
| ADR-003 | 热度模块架构 | HeatService 独立 | ✅ 已决策 | Phase 1 |
| ADR-004 | 热度触发策略 | 混合模式 | ✅ 已决策 | Phase 1 |
| ADR-005 | L4 架构边界 | 独立 Archiver | ✅ 已决策 | Phase 2 |
| ADR-006 | mempalace-rs 继承 | 扩展而非重写 | ✅ 已决策 | Phase 1 |
| ADR-007 | MVP 功能优先级 | 检索层核心 | ✅ 已决策 | Phase 1 |
| ADR-008 | 数据存储 | SQLite + usearch | ✅ 已确认 | Phase 1 |
| ADR-009 | API 设计 | CLI + MCP + Python | ✅ 已确认 | Phase 1-2 |
| ADR-010 | 错误处理 | MemoryError 统一 | ✅ 已确认 | Phase 1 |
| ADR-011 | 配置管理 | config.toml | ✅ 已确认 | Phase 1 |
| ADR-012 | 测试架构 | serial_test + Fixture | ✅ 已确认 | Phase 1 |

---

## Step 5: 实现模式与一致性规则

_确保多个 AI Agent 编写兼容、一致的代码。_

---

### 5.1 命名模式

**Rust 代码命名**:

| 类型 | 规范 | 示例 |
|------|------|------|
| 模块 | snake_case | `heat_service`, `archiver` |
| Struct | PascalCase | `HeatService`, `MemoryRecord` |
| 函数 | snake_case | `calculate_heat()`, `should_archive()` |
| 字段 | snake_case | `heat_i32`, `last_accessed` |
| 常量 | UPPER_SNAKE | `HOT_THRESHOLD`, `MAX_WAKEUP_TOKENS` |
| Enum variant | PascalCase | `HeatState::Locked`, `LaputaError::StorageError` |

**数据库命名（SQLite）**:

| 类型 | 规范 | 示例 |
|------|------|------|
| 表名 | snake_case | `memory_records`, `heat_index` |
| 列名 | snake_case | `heat_i32`, `last_accessed`, `archive_candidate` |
| 索引 | idx_表名_列名 | `idx_memory_records_heat` |
| 外键 | fk_表名_列名 | `fk_triples_memory_id` |

**MCP Tool 命名**:

```json
// 前缀 laputa_ + snake_case
"laputa_store_memory"
"laputa_recall"
"laputa_get_heat_status"
"laputa_mark_important"
"laputa_generate_wakeup_context"

// 参数名：snake_case
{"memory_id": "uuid", "heat_threshold": 5000}
```

---

### 5.2 结构模式

**项目目录结构**:

```
Laputa/
├── src/
│   ├── heat/              # 独立热度模块
│   │   ├── mod.rs
│   │   ├── service.rs     # HeatService 核心逻辑
│   │   ├── index.rs       # HeatIndex 阈值判断
│   │   ├── decay.rs       # 衰减公式
│   │   └── config.rs      # HeatConfig
│   ├── archiver/          # 独立归档模块（Phase 1 只标记）
│   │   ├── mod.rs
│   │   └── marker.rs      # 归档候选标记
│   ├── storage/           # 继承 mempalace-rs，扩展 L4
│   ├── searcher/          # 继承 mempalace-rs，加热度排序
│   ├── knowledge_graph/   # 继承 mempalace-rs，共振度
│   ├── dialect/           # 继承 mempalace-rs（EMOTION_CODES）
│   ├── diary/             # 继承 mempalace-rs
│   ├── mcp_server/        # 扩展 mempalace-rs（新增 Tools）
│   ├── cli/               # 新增 CLI 入口
│   ├── lib.rs
│   └── main.rs            # CLI entry point
├── tests/
│   ├── test_heat.rs       # 热度机制测试
│   ├── test_archiver.rs   # 归档标记测试
│   ├── test_wakepack.rs   # 唤醒包测试
│   ├── integration/       # E2E 测试
│   │   └── test_mvp_flow.rs
│   └── fixtures/
│       ├── memory_only.rs     # 纯内存 fixture
│       ├── with_tempdir.rs    # 真实文件系统
│       └── time_machine.rs    # 时间模拟
├── config/
│   └── laputa.toml        # 配置文件
├── Cargo.toml
├── Cargo.lock
└── README.md
```

**文件命名规则**:

| 文件类型 | 规范 | 示例 |
|----------|------|------|
| Rust 模块 | snake_case.rs | `heat_service.rs` |
| 测试文件 | test_前缀.rs | `test_heat.rs` |
| 配置文件 | snake_case.toml | `laputa.toml` |
| 文档文件 | snake_case.md | `heat_mechanism.md` |

---

### 5.3 格式模式

**API 响应格式（统一三接口）**:

```rust
// 成功响应
pub struct ApiResponse<T> {
    pub data: T,
    pub meta: ResponseMeta,
}

pub struct ResponseMeta {
    pub timestamp: DateTime<Utc>,  // ISO 8601
    pub version: String,           // "v1.0.0"
}

// 错误响应
pub struct ApiError {
    pub error: ErrorDetail,
}

pub struct ErrorDetail {
    pub code: String,              // "HEAT_THRESHOLD_ERROR"
    pub message: String,           // 用户友好消息
    pub detail: Option<String>,    // 技术细节（可选）
}
```

**日期时间格式**:

| 场景 | 格式 | 示例 |
|------|------|------|
| 存储 | DateTime<Utc> | Rust chrono 类型 |
| 序列化 | ISO 8601 | `"2026-04-13T12:34:56Z"` |
| 日志 | ISO 8601 | `"2026-04-13T12:34:56Z"` |
| 用户展示 | 本地化 | "2026年4月13日"（可选） |

**JSON 字段命名**:

| 场景 | 规范 | 示例 |
|------|------|------|
| API 响应 | snake_case | `{"heat_i32": 8000}` |
| MCP 参数 | snake_case | `{"memory_id": "uuid"}` |
| 配置文件 | snake_case | `heat_i32 = 8000` |

---

### 5.4 通信模式

**内部事件命名**:

```rust
// 前缀 laputa_ + snake_case + 过去分词
enum LaputaEvent {
    HeatThresholdCrossed { memory_id: Uuid, old_state: HeatState, new_state: HeatState },
    ArchiveCandidateMarked { memory_id: Uuid },
    WakeupContextGenerated { token_count: usize },
    WeeklyCapsuleCreated { week: u32 },
}
```

**状态更新模式**:

```rust
// 不可变更新（Rust 风格）
impl MemoryRecord {
    pub fn with_updated_heat(&self, new_heat: i32) -> Self {
        Self {
            heat_i32: new_heat,
            ..self.clone()
        }
    }
}

// 而非直接修改（避免并发问题）
// record.heat_i32 = new_heat; // ❌ 禁止
```

---

### 5.5 进程模式

**错误处理模式**:

```rust
// 统一错误枚举
pub enum LaputaError {
    StorageError(String),
    HeatThresholdError(i32),      // 状态机边界错误
    ArchiveError(String),        // L4 归档错误
    WakepackSizeExceeded(usize), // 唤醒包超限
    ConfigError(String),
    NotFound(Uuid),
    ValidationError(String),
}

// 错误转换链
impl From<sqlite::Error> for LaputaError {
    fn from(e: sqlite::Error) -> Self { LaputaError::StorageError(e.to_string()) }
}

// 公共函数签名
pub fn store(&mut self, record: MemoryRecord) -> Result<Uuid, LaputaError>;
```

**日志格式**:

```rust
// 结构化日志
#[derive(Serialize)]
struct LogEntry {
    timestamp: DateTime<Utc>,  // ISO 8601
    level: LogLevel,           // INFO/WARN/ERROR
    module: String,            // "heat::service"
    message: String,
    context: Option<serde_json::Value>,
}

// 示例输出
{"timestamp":"2026-04-13T12:34:56Z","level":"INFO","module":"heat::service","message":"Heat threshold crossed","context":{"memory_id":"abc","new_state":"Normal"}}
```

**并发安全模式**:

```rust
// 热度更新：Arc + Mutex 或 RwLock
use std::sync::{Arc, RwLock};

pub struct HeatIndex {
    cache: Arc<RwLock<HashMap<Uuid, i32>>>,
}

// 读取：允许多线程
fn get_heat(&self, id: Uuid) -> i32 {
    self.cache.read().unwrap().get(&id).copied().unwrap_or(0)
}

// 写入：独占锁
fn update_heat(&self, id: Uuid, new_heat: i32) {
    self.cache.write().unwrap().insert(id, new_heat);
}
```

---

### 5.6 强制规则

**所有 AI Agent 必须遵守**:

| ID | 规则 | 违反后果 |
|----|------|----------|
| **R-001** | 命名一致性：snake_case（模块/函数/字段）+ PascalCase（struct/enum） | 代码 review 拒绝 |
| **R-002** | 继承优先：扩展 mempalace-rs 时不修改现有 public API 签名 | 破坏 197 tests |
| **R-003** | 热度字段：所有 MemoryRecord 必须包含 `heat_i32`（放大 100 倍） | 数据模型不完整 |
| **R-004** | 错误处理：所有公共函数返回 `Result<T, LaputaError>` | API 不一致 |
| **R-005** | 测试隔离：归档/热度测试必须用 `#[serial]` + `tempdir` | Flaky tests |
| **R-006** | 日志格式：结构化 JSON，包含 timestamp/level/module/message | 日志不可解析 |
| **R-007** | 文档注释：所有 public API 必须有 `///` 文档注释 | 无法生成 docs |

**模式验证**:

```bash
# CI 检查脚本（建议）
cargo clippy --all-features  # lint 检查
cargo test --all            # 197+ tests 必须
cargo doc --no-deps          # 文档生成
```

---

### 5.7 示例与反例

**正确示例**:

```rust
// ✅ 正确：遵循命名规范
pub struct HeatService {
    config: HeatConfig,
}

impl HeatService {
    pub fn calculate(&self, record: &MemoryRecord) -> i32 {
        let days = (Utc::now() - record.last_accessed).num_days() as f64;
        let decayed = record.heat_i32 as f64 * (-self.config.decay_rate * days).exp();
        (decayed * 100.0).round() as i32
    }
}

// ✅ 正确：不可变更新
let updated = record.with_updated_heat(new_heat);

// ✅ 正确：测试隔离
#[serial]
#[test]
fn test_heat_threshold_cross() {
    let tempdir = tempfile::tempdir().unwrap();
    // ...
}
```

**错误反例**:

```rust
// ❌ 错误：PascalCase 函数名
pub fn CalculateHeat() { }  // 应为 calculate_heat()

// ❌ 错误：直接修改字段
record.heat_i32 = new_heat;  // 应用 with_updated_heat()

// ❌ 错误：浮点热度
pub heat: f64;  // 应为 heat_i32: i32

// ❌ 错误：缺少错误处理
pub fn store(record: MemoryRecord) -> Uuid;  // 应返回 Result<Uuid, LaputaError>

// ❌ 错误：并行测试无隔离
#[test]
fn test_archive() {  // 应加 #[serial]
    // 涉及文件系统操作
}
```

---

## Step 6: 项目结构与边界

_定义完整项目结构和架构边界，映射需求到具体实现位置。_

---

### 6.1 需求到组件映射

**MVP 功能 → 架构组件**:

| MVP 功能 | 源码位置 | 关键文件 | 依赖 |
|---------|---------|---------|------|
| P0: 主体初始化 | `src/identity/` | `initializer.rs`, `state.rs` | storage, heat |
| P0: 日记输入层 | `src/diary/` (继承) | `diary.rs` (扩展) | storage, heat, dialect |
| P1: 检索层 | `src/searcher/` (继承) | `searcher.rs` (扩展) | storage, heat, usearch |
| P1: 唤醒上下文层 | `src/wakeup/` (新增) | `wakepack.rs`, `generator.rs` | searcher, heat, identity |
| P2: 节律整理层 | `src/rhythm/` (新增) | `capsule.rs`, `weekly.rs` | diary, heat |
| P3: 生命周期治理 | `src/heat/` + `src/archiver/` | `service.rs`, `marker.rs` | storage, knowledge_graph |

---

### 6.2 完整项目目录结构

```
Laputa/
├── Cargo.toml                    # Rust 配置
├── Cargo.lock                     # 依赖锁定
├── .gitignore                     # Git 忽略
├── README.md                      # 项目介绍
├── LICENSE                        # MIT/Apache 双许可
│
├── config/
│   ├── laputa.toml                # 主配置文件
│   └── laputa.toml.example        # 配置示例
│
├── src/
│   ├── lib.rs                    # 库入口
│   ├── main.rs                   # CLI 入口
│   │
│   ├── heat/                     # 热度模块 (ADR-003)
│   │   ├── mod.rs                # 模块导出
│   │   ├── service.rs            # HeatService 核心计算
│   │   ├── index.rs              # HeatIndex 阈值判断
│   │   ├── decay.rs              # 衰减公式实现
│   │   ├── config.rs             # HeatConfig 配置结构
│   │   └── state.rs              # HeatState 状态机
│   │
│   ├── archiver/                 # 归档模块 (ADR-005)
│   │   ├── mod.rs                # 模块导出
│   │   ├── marker.rs             # 归档候选标记 (Phase 1)
│   │   ├── packer.rs             # 打包器 (Phase 2)
│   │   └── digger.rs             # 考古工具 (Phase 2)
│   │
│   ├── wakeup/                   # 唤醒模块 (新增)
│   │   ├── mod.rs                # 模块导出
│   │   ├── wakepack.rs           # WakePack 结构
│   │   ├── generator.rs          # 唤醒包生成器
│   │   └── token_estimator.rs    # Token 估算
│   │
│   ├── rhythm/                   # 节律整理模块 (新增)
│   │   ├── mod.rs                # 模块导出
│   │   ├── capsule.rs            # 摘要胶囊
│   │   ├── weekly.rs             # 周级整理
│   │   └── scheduler.rs          # 定时调度
│   │
│   ├── identity/                 # 主体初始化模块 (新增)
│   │   ├── mod.rs                # 模块导出
│   │   ├── initializer.rs        # 初始化器
│   │   └── state.rs              # 主体状态
│   │
│   ├── storage/                  # 继承 mempalace-rs，扩展
│   │   ├── mod.rs                # 模块导出 (扩展 L4)
│   │   ├── memory.rs             # MemoryRecord (扩展字段)
│   │   ├── layer.rs              # L0-L4 层定义
│   │   └── sqlite.rs             # SQLite 操作
│   │
│   ├── searcher/                 # 继承 mempalace-rs，扩展
│   │   ├── mod.rs                # 模块导出
│   │   ├── timeline.rs           # 时间流检索
│   │   ├── semantic.rs           # 语义检索 (RAG)
│   │   └── hybrid.rs             # 混合排序 (加热度因子)
│   │
│   ├── knowledge_graph/          # 继承 mempalace-rs，扩展
│   │   ├── mod.rs                # 模块导出
│   │   ├── triples.rs            # 时间三元组
│   │   ├── resonance.rs          # 共振度计算 (UPSP 融合)
│   │   └── relation.rs           # 关系节点
│   │
│   ├── dialect/                  # 继承 mempalace-rs (直接沿用)
│   │   ├── mod.rs                # 模块导出
│   │   ├── emotion.rs            # EMOTION_CODES
│   │   └── aaak.rs                # AAAK 压缩 V:3.2
│   │
│   ├── diary/                    # 继承 mempalace-rs，扩展
│   │   ├── mod.rs                # 模块导出
│   │   ├── writer.rs             # 日记写入
│   │   └── reader.rs             # 日记读取
│   │
│   ├── mcp_server/               # 继承 mempalace-rs，扩展
│   │   ├── mod.rs                # 模块导出
│   │   ├── server.rs             # MCP 服务器
│   │   ├── tools.rs              # 20+ MCP Tools (扩展)
│   │   └── handlers.rs           # Tool 处理器
│   │
│   ├── cli/                      # CLI 接口 (新增)
│   │   ├── mod.rs                # 模块导出
│   │   ├── commands.rs           # 子命令定义
│   │   ├── handlers.rs           # 命令处理器
│   │   └── output.rs             # 输出格式化
│   │
│   ├── api/                      # 统一抽象层 (ADR-009)
│   │   ├── mod.rs                # 模块导出
│   │   ├── operation.rs          # MemoryOperation trait
│   │   ├── response.rs           # ApiResponse/ApiError
│   │   └── error.rs               # LaputaError 枚举
│   │
│   └── utils/                    # 工具函数
│   │       ├── mod.rs            # 模块导出
│   │       ├── time.rs           # 时间工具
│   │       └── logging.rs        # 结构化日志
│
├── tests/
│   ├── test_heat.rs              # 热度机制测试 (SM-01~08)
│   ├── test_archiver.rs          # 归档标记测试
│   ├── test_wakepack.rs          # 唤醒包测试
│   ├── test_rhythm.rs            # 节律整理测试
│   ├── test_identity.rs          # 主体初始化测试
│   ├── test_integration.rs       # 集成测试
│   │
│   ├── fixtures/
│   │   ├── mod.rs                # Fixture 导出
│   │   ├── memory_only.rs        # 纯内存 fixture
│   │   ├── with_tempdir.rs       # 真实文件系统
│   │   ├── with_archive.rs       # L4 fixture
│   │   └── time_machine.rs       # 时间模拟工具
│   │
│   └── integration/
│       ├── test_mvp_flow.rs      # E2E: diary → capsule → wakeup → answer
│       └── test_cli_flow.rs       # CLI 端到端测试
│
├── benches/                      # 性能基准 (可选)
│   └── bench_heat.rs             # 热度计算基准
│
├── docs/                         # 设计文档
│   ├── design-memory-stack.md    # 记忆栈设计
│   ├── design-heat-mechanism.md  # 热度机制设计
│   ├── design-rhythm-organize.md # 节律整理设计
│   └── design-wakeup-context.md  # 唤醒包设计
│
├── examples/                     # 使用示例
│   ├── basic_usage.rs            # 基础使用示例
│   └── mcp_client.rs             # MCP 客户端示例
│
└── .github/                      # CI/CD
    └── workflows/
        ├── ci.yml                # 持续集成
        └── release.yml           # 发布流程
```

---

### 6.3 架构边界定义

**API 边界**:

| 边界 | 入口 | 协议 | 跨边界通信 |
|------|------|------|----------|
| CLI → Core | `src/cli/` | Rust 函数调用 | 直接调用 MemoryOperation trait |
| MCP → Core | `src/mcp_server/` | JSON-RPC 2.0 | Tool handlers 调用 Core |
| Python → Core | Phase 2 | PyO3 绑定 | Python API 包装 Core |
| Core → Storage | `src/storage/` | Rust trait | Storage trait 抽象 |
| Core → Searcher | `src/searcher/` | Rust trait | Searcher trait 抽象 |

**组件边界图**:

```
┌─────────────────────────────────────────────────────────────┐
│  External Interfaces                                         │
│  ┌─────────┐  ┌─────────┐  ┌─────────────┐                   │
│  │   CLI   │  │   MCP   │  │ Python API  │ (Phase 2)         │
│  └─────────┘  └─────────┘  └─────────────┘                   │
│       │            │              │                           │
│       └────────────┼──────────────┘                           │
│                    ↓                                         │
├─────────────────────────────────────────────────────────────┤
│  API Abstraction Layer                                       │
│  ┌────────────────────────────────────────────┐             │
│  │ MemoryOperation trait                       │             │
│  │ - store() / recall() / archive()            │             │
│  │ LaputaError unified error handling          │             │
│  └────────────────────────────────────────────┘             │
│                    ↓                                         │
├─────────────────────────────────────────────────────────────┤
│  Core Modules                                                │
│  ┌────────┐ ┌────────┐ ┌────────┐ ┌────────┐ ┌────────┐      │
│  │  heat  │ │ wakeup │ │ rhythm │ │identity│ │archiver│      │
│  │(新增)  │ │(新增)  │ │(新增)  │ │(新增)  │ │(新增)  │      │
│  └────────┘ └────────┘ └────────┘ └────────┘ └────────┘      │
│       │          │          │          │          │          │
│       └──────────┴──────────┴──────────┴──────────┘          │
│                    ↓                                         │
├─────────────────────────────────────────────────────────────┤
│  Inherited from mempalace-rs                                 │
│  ┌────────┐ ┌────────┐ ┌────────┐ ┌────────┐ ┌────────┐      │
│  │storage │ │searcher│ │ graph  │ │dialect │ │ diary  │      │
│  │(扩展)  │ │(扩展)  │ │(扩展)  │ │(沿用)  │ │(扩展)  │      │
│  └────────┘ └────────┘ └────────┘ └────────┘ └────────┘      │
│                    ↓                                         │
├─────────────────────────────────────────────────────────────┤
│  Data Layer                                                  │
│  ┌─────────────────────┐  ┌─────────────────────┐           │
│  │ SQLite (laputa.db)  │  │ usearch (向量索引)   │           │
│  └─────────────────────┘  └─────────────────────┘           │
└─────────────────────────────────────────────────────────────┘
```

---

### 6.4 需求到结构映射

**功能 → 目录映射**:

| 功能 | 源码位置 | 测试位置 | 配置位置 |
|------|---------|---------|---------|
| 热度机制 | `src/heat/` | `tests/test_heat.rs` | `config/laputa.toml [heat]` |
| 唤醒包生成 | `src/wakeup/` | `tests/test_wakepack.rs` | `config/laputa.toml [wakeup]` |
| 节律整理 | `src/rhythm/` | `tests/test_rhythm.rs` | `config/laputa.toml [rhythm]` |
| 归档标记 | `src/archiver/marker.rs` | `tests/test_archiver.rs` | `config/laputa.toml [archive]` |
| 主体初始化 | `src/identity/` | `tests/test_identity.rs` | `config/laputa.toml [identity]` |
| 记忆检索 | `src/searcher/` | 继承 mempalace-rs tests | - |
| 知识图谱 | `src/knowledge_graph/` | 继承 mempalace-rs tests | - |
| MCP Tools | `src/mcp_server/tools.rs` | MCP 客户端测试 | - |

**跨切面关注点**:

| 关注点 | 位置 | 影响模块 |
|--------|------|---------|
| 错误处理 | `src/api/error.rs` | 所有模块 |
| 日志格式 | `src/utils/logging.rs` | 所有模块 |
| 配置加载 | `src/lib.rs` | 所有模块 |
| 测试隔离 | `tests/fixtures/` | 所有测试 |

---

### 6.5 集成点定义

**内部通信**:

| 通信方式 | 示例 | 约束 |
|---------|------|------|
| Trait 抽象 | `MemoryOperation::store()` | 统一接口 |
| 事件驱动 | `LaputaEvent::HeatThresholdCrossed` | 异步可选 |
| 数据流 | `heat → searcher → wakeup` | 单向依赖 |

**外部集成**:

| 成成点 | 协议 | 状态 |
|--------|------|------|
| AI Agent (MCP) | JSON-RPC 2.0 | Phase 1 必须 |
| agent-diva 嵌入 | Rust API | Phase 3 |
| openfang 集成 | Rust API | Phase 2+ |
| Python 客户端 | PyO3 绑定 | Phase 2 |

**数据流图**:

```
用户输入
    │
    ↓
MemoryGate (筛选)
    │
    ↓ (store)
Storage Layer (L0-L3 + heat_i32)
    │
    ├──────────────────────┐
    ↓                      ↓
HeatService           KnowledgeGraph
(热度计算)              (共振度更新)
    │                      │
    ↓                      ↓
定时批量更新            关系节点维护
    │
    ↓ (recall)
Searcher (时间流 + 语义 + 热度排序)
    │
    ↓
WakePack Generator (Top-K + <1200 tokens)
    │
    ↓
唤醒上下文注入 AI Agent
```

---

## Step 7: 架构验证结果

_验证架构一致性、需求覆盖和实现就绪度。_

---

### 7.1 一致性验证

| 检查项 | 结果 | 说明 |
|--------|------|------|
| **技术兼容性** | ✅ 通过 | Rust + SQLite + usearch + tokio 全部兼容，mempalace-rs 已验证 |
| **决策一致性** | ✅ 通过 | 12 个 ADR 无冲突，继承边界明确 |
| **模式一致性** | ✅ 通过 | snake_case/PascalCase 统一，错误处理统一 |
| **结构对齐** | ✅ 通过 | 目录结构支持所有 ADR，边界清晰 |

**技术栈版本兼容矩阵**:

| 技术 | 版本 | 兼容依赖 |
|------|------|---------|
| Rust | 1.75+ | mempalace-rs 基线 |
| SQLite | bundled | rusqlite 0.30 |
| usearch | 0.25 | 向量检索 |
| tokio | 1.x | 异步运行时 |
| serial_test | 2.x | 新增测试隔离 |

---

### 7.2 需求覆盖验证

| PRD 需求 | 架构支持 | ADR 引用 |
|----------|---------|---------|
| FR-01 主体初始化 | ✅ `src/identity/` | ADR-007 (P0) |
| FR-02 日记输入 | ✅ `src/diary/` (继承) | ADR-006 |
| FR-03 节律整理 | ✅ `src/rhythm/` | ADR-007 (P2) |
| FR-04 唤醒上下文 | ✅ `src/wakeup/` | ADR-004, HEAT-03 |
| FR-05 检索层 | ✅ `src/searcher/` (继承) | ADR-007 (P1 核心) |
| FR-06 生命周期治理 | ✅ `src/heat/` + `src/archiver/` | ADR-001~005 |
| FR-07 RAG 必须启用 | ✅ `src/searcher/semantic.rs` | D-004 |
| FR-08 共振度 | ✅ `src/knowledge_graph/resonance.rs` | D-003-B |
| FR-09 可移植性 | ✅ CLI/MCP/Python API | ADR-009 |

**覆盖率**: 9/9 FR = **100%**

---

### 7.3 实现就绪验证

| 检查项 | 结果 | 说明 |
|--------|------|------|
| **决策完整性** | ✅ | 12 ADR 全部有版本和理由 |
| **模式完整性** | ✅ | 命名/结构/格式/通信/进程 5 类模式完整 |
| **结构完整性** | ✅ | 目录结构完整，边界明确，集成点定义 |
| **示例完整性** | ✅ | 正确示例 + 错误示例 |
| **继承边界明确** | ✅ | mempalace-rs 继承 vs Laputa 新增标注清晰 |
| **L4 分类层明确** | ✅ | ADR-005 定义归档层，Phase 1 只标记 |

---

### 7.4 差距分析

| 优先级 | 差距 | 状态 | 处理建议 |
|--------|------|------|----------|
| 🔴 Critical | 无 | - | 架构完整 |
| 🟡 Important | Python API 详细设计 | Phase 2 | 待后续 PRD 补充 |
| 🟡 Important | agent-diva 嵌入时机 | Phase 3 | D-003-A 已记录 |
| 🟢 Nice-to-Have | 性能基准测试 | 预留 | benches/ 已规划 |
| 🟢 Nice-to-Have | openfang 集成 | Phase 2+ | 待需求明确 |

---

### 7.5 核心约束确认

| 大湿强调的约束 | 文档位置 | 状态 |
|----------------|---------|------|
| **基于 mempalace-rs 修改** | D-006 (DECISIONS.md), ADR-006, 项目结构 746-751 行 | ✅ 已确保 |
| **新增 L4 分类层** | D-007, ADR-005, 项目结构 746 行注释 | ✅ 已确保 |
| **继承 vs 新增标注** | 每个模块注释 | ✅ 已确保 |

---

### 7.6 架构完整性检查清单

**✅ 需求分析**

- [x] 项目上下文彻底分析
- [x] 规模和复杂度评估
- [x] 技术约束识别
- [x] 跨切面关注点映射

**✅ 架构决策**

- [x] 关键决策已版本化文档
- [x] 技术栈完整定义
- [x] 集成模式定义
- [x] 性能考量已处理

**✅ 实现模式**

- [x] 命名约定建立
- [x] 结构模式定义
- [x] 通信模式定义
- [x] 进程模式文档化

**✅ 项目结构**

- [x] 完整目录结构定义
- [x] 组件边界建立
- [x] 集成点映射
- [x] 需求到结构映射完成

**✅ mempalace-rs 继承边界**

- [x] 7 个必须继承模块明确
- [x] 继承 vs 扩展 vs 新增分类
- [x] L4 归档层独立 Archiver 组件
- [x] 测试隔离策略（serial_test + Fixture）

---

### 7.7 架构就绪评估

**整体状态**: ✅ **READY FOR IMPLEMENTATION**

**信心等级**: **高** (基于完整验证结果)

**核心优势**:

1. **代码基线坚实** - mempalace-rs 197 tests 已验证，演化而非重写
2. **继承边界清晰** - 每个 module 明确标注继承/扩展/新增
3. **热度机制完整** - i32 存储 + HeatService + 状态机 + 阈值体系
4. **L4 归档层独立** - Archiver 组件不侵入存储核心，Phase 1 只标记
5. **测试架构成熟** - serial_test + TimeMachine + 分层 Fixture

**未来增强方向**:

- Phase 2: Python API (PyO3 绑定)
- Phase 2: 完整归档流程 (packer + digger)
- Phase 3: agent-diva 嵌入

---

### 7.8 实现交接

**AI Agent 指导**:

- 遵循所有架构决策，严格按文档执行
- 统一使用实现模式，跨组件一致
- 尊重项目结构和边界
- 架构问题参考此文档

**首要实现优先级**:

```bash
# Step 1: 创建 Laputa 目录结构
mkdir -p Laputa/src/{heat,archiver,wakeup,rhythm,identity,cli,api,utils}
mkdir -p Laputa/tests/fixtures
mkdir -p Laputa/config

# Step 2: 复制 mempalace-rs 继承模块
cp -r mempalace-rs/src/storage Laputa/src/
cp -r mempalace-rs/src/searcher Laputa/src/
cp -r mempalace-rs/src/knowledge_graph Laputa/src/
cp -r mempalace-rs/src/dialect Laputa/src/
cp -r mempalace-rs/src/diary Laputa/src/
cp -r mempalace-rs/src/mcp_server Laputa/src/

# Step 3: 扩展 storage/memory.rs 添加 heat_i32 字段
# Step 4: 实现 HeatService 核心逻辑
```

---

_架构验证完成，准备进入实现阶段_