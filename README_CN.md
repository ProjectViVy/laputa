# LAPUTA

[English](README.md)

面向持续运行 AI Agent 的治理型记忆操作系统。LAPUTA 将个人工作材料连接到可召回的上下文和可复用的能力——同时不把存储、检索或演化输出本身当作权威。

## 核心理念

```text
MemoryOS = 以 Agent 人格与治理记忆为核心，
           能理解、定位、调用个人全部工作信息的记忆操作系统。
```

三个独立的 Go 模块强制执行严格的所有权边界：

| 模块 | 职责 |
|------|------|
| **Laputa** | 身份、权威、生命周期、策略、审计 |
| **Mentle** | 规范材料、证据、检索、分类、知识图谱 |
| **Garden** | 源摄入、召回编排、ContextView 组装、HTTP 网关 |

任何模块不对其他模块持有权威。每个模块均可优雅降级。

## 关键设计决策

- **渐进式召回** — Fast Recall（默认）：零 LLM、确定性、低延迟、可缓存。Deep Recall（显式升级）：独立预算、KG/时间线/图谱扩展、完整追踪。
- **候选 ≠ 证据 ≠ ContextView** — 发现、有界证据读取、最终组装是独立阶段，各有独立预算。
- **禁止静默高影响变更** — 权威变更、技能审批、宿主安装、物理删除始终显式且可审计。
- **治理型演化** — 外部 Evolver 提出能力提案；只有 Laputa 批准并应用权威。

## 架构

```text
┌─────────────────────────────────────────────────────┐
│  宿主适配器 (Hermes / Claude Code / Codex)           │
└──────────────────────┬──────────────────────────────┘
                       │ HTTP
┌──────────────────────▼──────────────────────────────┐
│  Garden — 编排网关                                   │
│  /v2/recall/fast · /v2/recall/deep                  │
│  /v2/activity/*  · /v2/governance/*                 │
│  /v2/evolution/* · /v1/*（兼容层）                   │
└───────┬─────────────────────────────┬───────────────┘
        │                             │
┌───────▼────────┐          ┌─────────▼──────────────┐
│  Laputa        │          │  Mentle                │
│  治理          │          │  材料 + 检索            │
│  权威          │          │  证据 + 图谱            │
│  审计          │          │  混合搜索 (HNSW)        │
└────────────────┘          └────────────────────────┘
```

## 快速开始

```bash
# 前置条件：Go 1.26+，启用 CGO（用于 SQLite）

# 构建所有模块
cd laputa  && go build ./...
cd ../mentle && go build ./...
cd ../garden && go build -o garden.exe .

# 运行服务（默认：http://127.0.0.1:7373）
./garden.exe

# 健康检查
curl -s http://127.0.0.1:7373/health
```

## 配置

| 环境变量 | 说明 | 默认值 |
|----------|------|--------|
| `GARDEN_PIPELINE_CONFIG` | pipelines.yaml 路径 | `~/.garden/pipelines.yaml` |
| `GARDEN_RAG_BASE_URL` | OpenAI 兼容 LLM 端点 | _（禁用）_ |
| `GARDEN_RAG_API_KEY` | LLM planner API 密钥 | _（禁用）_ |
| `GARDEN_RAG_MODEL` | planner 模型名称 | _（禁用）_ |

未配置 LLM 环境变量时，Garden 使用确定性 planner 并报告降级，不会失败。

## 测试

```bash
cd laputa  && GOSUMDB=off go test ./governance/...
cd ../mentle && GOSUMDB=off go test ./facade/...
cd ../garden && GOSUMDB=off go test ./internal/...
GOSUMDB=off go test -tags=e2e ./e2e/...
```

## 仓库结构

```text
laputa/    Go 治理模块 — 权威、身份、生命周期、审计
mentle/    Go 材料与检索模块 — 规范目录、证据、混合搜索、图谱
garden/    Go 应用模块 — HTTP 网关、召回、活动编排
docs/      架构决策、迁移计划、历史归档
```

## 性能目标

| 操作 | 目标 |
|------|------|
| 治理投影（热） | P95 ≤ 5 ms |
| SearchCards | P95 ≤ 80 ms |
| 过滤 / 排序 / 去重 | P95 ≤ 10 ms |
| 有界 ReadEvidence | P95 ≤ 40 ms |
| Fast Recall 总计 | P95 ≤ 150 ms |
| 仅治理降级 | P95 ≤ 30 ms |

## 文档

- [架构计划 (vNext)](docs/architecture/0001-memoryos-vnext-architecture.md)
- [ADR-0002：认知分区决策](docs/architecture/0002-laputa-cognitive-partition-decision.md)
- [ADR-0003：运维控制台设计](docs/architecture/0003-operations-console-design.md)
- [文档索引](docs/README.md)
- [历史归档](docs/archive/2026-08-01-pre-memoryos-redesign/)

## 参考项目

- [MemGPT / Letta](https://github.com/letta-ai/letta) — LLM 记忆管理，虚拟上下文分页
- [Mem0](https://github.com/mem0ai/mem0) — AI Agent 记忆层
- [Zep](https://github.com/getzep/zep) — AI 助手长期记忆服务
- [LangChain Memory](https://github.com/langchain-ai/langchain) — LLM 应用可组合记忆模块
- [LlamaIndex](https://github.com/run-llama/llama_index) — 基于 LLM 的检索数据框架
- [Cognee](https://github.com/topoteretes/cognee) — 基于知识图谱的 AI Agent 记忆管理
- [HNSW (govector)](https://github.com/DotNetAge/govector) — Mentle 使用的 HNSW 向量索引
- [Eino (CloudWeGo)](https://github.com/cloudwego/eino) — Laputa 使用的 LLM 编排框架

## 许可证

[MIT](LICENSE)
