# 天空之城 (Laputa) 核心决策记录

**Author:** 大湿  
**Date:** 2026-04-13  
**Status:** APPROVED

---

## 一、项目定位决策

### D-001：天空之城与 agent-diva 的关系

**决策**：天空之城是为 agent-diva 的记忆模块而开发的，但**必须具备高度可移植性**。

**理由**：
- 可在 openfang 等 Rust agent 项目中使用
- 未来需要 Python API，可接入其他项目
- 不绑定单一宿主，作为通用记忆底座

**影响**：
- 架构设计必须考虑多宿主适配
- 需要设计清晰的 API 边界
- CLI/MCP/Python API 三种接口并存

---

### D-002：双轨记忆（理性/感性）决策

**决策**：**不再采用双轨记忆设计**。

**理由**：
- 改用"热度+主题"自动归档，比理性/感性二元分类更复杂、更精细
- 一切基于 mempalace-rs 改进，而非另起炉灶
- mempalace 的"有温度的房子"理念：热度决定保留，主题决定归类

**替代方案**：
- 情绪编码（沿用 mempalace-rs EMOTION_CODES）
- 热度衰减（艾宾浩斯反向）
- 主题聚合（wing/room/closet 结构）

---

### D-003：主体协议融合决策

**决策**：不保留 UPSP 七文件结构，只**取其一个核心概念：共振度（关系节点）**。

**理由**：
- UPSP 的复杂概念（工化指数、动态六轴、变速轮等）不适合天空之城
- 天空之城的记忆栈和热度机制已覆盖大部分功能
- 只需要共振度这一个概念，用于关系节点

### D-003-A：明确排除的 UPSP 概念

| UPSP 概念 | 排除理由 |
|-----------|----------|
| **工化指数 (workhood_index)** | 大湿明确不喜欢，不采用 |
| 动态六轴中的 focus/mood/humor/safety | UPSP 自创，学术依据不强 |
| 核心六轴 | 静态认知风格不适合动态记忆系统 |
| 变速轮 (speed_wheel) | 256轮触发机制与天空之城节律点设计不同 |
| 模型戳 | 天空之城不追踪模型切换 |
| 七文件结构 | 天空之城已有记忆栈，不需要文件分离 |
| 权重形态 [F]/[S]/[A] | 用热度范围替代（>80锁定，<20归档） |

### D-003-B：保留的 UPSP 概念（共三个）

| UPSP 概念 | 天空之城对应 | 用途 |
|-----------|--------------|------|
| **共振度 (Resonance)** | 关系节点 + 知识图谱 | 记录与交互对象的关系温度 |
| **valence** | 情绪胶囊数值维度 | 情感效价（-100~+100，负面到正面） |
| **arousal** | 情绪胶囊数值维度 | 激活程度（0~100，平静到兴奋） |

**共振度实现**：
- 范围：-100 ~ +100
- 存储位置：knowledge_graph 的 triples 表
- 更新机制：每次交互后按 Δvalence 更新
- 在唤醒包中展示关键关系的共振度

**valence + arousal 实现**：
- 用途：情绪胶囊可视化、情绪搜索、共振度更新
- 存储：记忆条目的 emotion_valence 和 emotion_arousal 字段
- 与 EMOTION_CODES 并存：离散标签 + 连续数值
- 共振度更新公式：`delta_r = Δvalence / resistance`

---

### D-004：RAG/语义检索决策

**决策**：**RAG 在天空之城设计中不可或缺，MVP 必须启用**。

**理由**：
- LifeBook 反 RAG 的观点不代表天空之城
- 语义检索与时间流检索是互补关系，不是对立关系
- 时间流适合"最近发生了什么"，语义检索适合"具体某句话怎么说"

**定位**：
- 时间流为主轴（唤醒包、节律摘要）
- 语义检索为补充（深度检索、考古工具）
- 情绪搜索、节点搜索为第三维度

---

### D-005：归档格式决策

**决策**：**沿用 mempalace-rs 已有的结构**，不重新定义天空之城专用打包格式。

**理由**：
- 保持兼容性
- mempalace-rs 的 SQLite + usearch 结构已验证
- 新增热度字段和归档候选标记即可

---

## 二、架构决策

### D-006：代码基线

**决策**：以 `mempalace-rs` 为代码基线继续开发，而不是从零重写。

**理由**：
- 197 tests 已通过，生产可用
- 核心架构已验证：四层记忆栈、AAAK 压缩、知识图谱、MCP 工具
- 演化而非重构

---

### D-007：记忆栈结构

**决策**：沿用 mempalace-rs L0-L3 结构，新增 L4 归档层。

| 层 | mempalace-rs | 天空之城扩展 |
|---|--------------|-------------|
| L0 | IDENTITY | identity.md + 情绪状态 |
| L1 | ESSENTIAL | 近期事件 + 热度排序 |
| L2 | ON-DEMAND | 语义检索 + 主题聚合 |
| L3 | SEARCH | 原始语义搜索 |
| L4 | (新增) | 归档候选 + 考古区 |

---

### D-008：热度机制

**决策**：热度决定记忆生命周期。

| 热度范围 | 处理 |
|---------|------|
| > 80 | 锁定，不衰减 |
| 50-80 | 正常保留，缓慢衰减 |
| 20-50 | 归档候选，节律整理时考虑打包 |
| < 20 | 自动打包进入归档区 |

---

### D-009：情绪编码

**决策**：沿用 mempalace-rs 的 AAAK EMOTION_CODES。

**核心情绪映射**（部分）：
```
joy → joy
love → love
fear → fear
trust → trust
grief → grief
wonder → wonder
rage → rage
hope → hope
despair → despair
curiosity → curious
```

---

## 三、接口决策

### D-010：多接口并存

**决策**：CLI / MCP / Python API 三种接口并存。

| 接口 | MVP 状态 | 用途 |
|------|---------|------|
| CLI | 必须 | 本地开发、测试、演示 |
| MCP | 必须 | AI agent 集成 |
| Python API | Phase 2 | 跨语言集成 |

---

### D-011：agent-diva 嵌入时机

**决策**：**C** - Phase 1 仅 diary 分区概念。

**理由**：
- 天空之城为 agent-diva 记忆模块开发，但保持可移植性
- Phase 1 先验证核心假设
- agent-diva 特有的 soul 演化等能力在后续阶段融入

---

## 四、项目结构决策

### D-012：项目主目录

**决策**：天空之城项目主目录为 `Laputa`。

**理由**：
- 可移植性强兼容
- 与 mempalace-rs、agent-diva 等项目平级
- 独立仓库，独立演化

---

## 五、Open Questions 最终答案

| Question | 答案 | 关键理由 |
|----------|------|---------|
| Q1 agent-diva 嵌入时机 | **C** Phase 1 仅 diary 分区概念 | 可移植性优先 |
| Q2 主体结构 | 融入设计，不保留文件形式 | UPSP 理念融合 |
| Q3 RAG/语义检索 | **必须启用** | 与时间流互补 |
| Q4 归档格式 | 沿用 mempalace-rs 结构 | 保持兼容 |

---

## 六、mempalace-rs 核心价值继承

### 来自 MISSION.md 的理念

> "MemPalace is not just about storing info in a highly structured way. 
> But also RETRIEVING it in a highly UNSTRUCTURED way!"

**继承**：
- 结构化存储 + 灵活检索
- Zettelkasten 方法：wing/room/closet/drawer
- AAAK 压缩：AI-readable shorthand
- 后台 hooks：降低 token 使用

### 来自代码的核心能力

| 模块 | 核心价值 | 天空之城继承 |
|------|---------|-------------|
| `storage.rs` | Layer0/1 结构 | 扩展为 L0-L4 |
| `dialect.rs` | 情绪编码 | 直接沿用 |
| `knowledge_graph.rs` | 时间三元组 | 关系时间线 |
| `searcher.rs` | 向量检索 | RAG 能力 |
| `mcp_server.rs` | 20 MCP 工具 | 接口基础 |
| `diary.rs` | 日记存储 | diary 分区 |

---

**最后更新**: 2026-04-13