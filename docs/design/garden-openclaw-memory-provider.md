# Garden × OpenClaw：Memory 接入方案（v2，MCP 范式）

- 状态：**已实施（Phase 1 最小闭环完成并验证）**
- 日期：2026-08-18（v3：Phase 1 已落地）
- 范围：把 Garden（MemoryOS）做成 OpenClaw agent 的记忆后端，让 agent 的记忆存取落到 Garden 的 Persona / Mentle evidence / Recall 之上
- 前置：ADR-0012 / ADR-0013（Laputa clean break，Persona 只读 slice 已上线验证）

---

## 0. v2 修订说明（为什么重写）

v1 文档错误地排除了"外部记忆框架接入"路径，只给了"同步文件 / 替换 memory-core 插件"两条路。经核实：

- **OpenClaw 确实能接外部记忆框架（mem0/Titen/Lithtrix/Honcho 等）**，用户记忆无误。
- 但接入点**不在 `memorySearch` 配置里**（`store.driver` 被锁死为 `sqlite`，`provider` 只是 embedding 后端选择），**而在 `mcp.servers`** —— 外部记忆框架以 **MCP server** 形式挂给 agent。
- 证据：ClawHub 技能市场有 `titen-memory`（用 Titen MCP server 做证据召回/信号记录/检查点）、`lithtrix-memory`（Lithtrix MCP server 跨会话记忆+身份）、`ecloud-long-term-memory`（云记忆服务语义搜索）等——**全部是 MCP 形态**。
- 结论：**Garden 的正确接入形态 = Garden 暴露 MCP server**，openclaw 通过 `mcp.servers["garden"]` 挂载。这不是"文件同步"，是"协议接入"。

---

## 1. 结论先行（TL;DR）

| 项 | 结论 |
|---|---|
| 对接形态 | **Garden 暴露 MCP server**，OpenClaw `mcp.servers` 挂载（stdio 本地进程 或 streamable-http 远程） |
| 记忆工具 | `persona_*`（人格）、`memory_search/get`（Recall）、`evidence_*`（Mentle 证据）、`record_signal`（对齐 titen 语义） |
| 需要新增的 Garden 端 | 一个 **MCP 服务进程**（Go 或 Node），桥到现有 `/v2/*` API；部分检索 API 可能需要补充 |
| 与内置 memory 关系 | **互补**：Garden MCP 管"权威记忆"，openclaw 内置 SQLite 管"会话/本地杂项"；两者并存 |
| 工作量 | 最小闭环（1 MCP server + 2-3 工具）≈ 0.5-1 天；完整（写路径+认证+同步）≈ 3-5 天 |

---

## 2. OpenClaw 记忆接入路径核实（2026-07 版，已逆向确认）

### 2.1 内置 memorySearch —— 不可作为外部记忆后端
- `agents.defaults.memorySearch`：
  - `store.driver` **const `sqlite`** —— 存储驱动写死，无法接外部库
  - `provider` 枚举：`openai/openai-compatible/gemini/voyage/mistral/bedrock/deepinfra/github-copilot/lmstudio/ollama/local` —— **只决定 embedding 后端**
  - `remote.baseUrl/apiKey/headers` —— 只覆盖 embedding API 端点，不是记忆库后端
  - `extraPaths` —— 可把**外部目录/.md 加入索引**（Garden 导出文件可挂载，但这是"数据接入"非"协议接入"）
  - `sync`（watch/onSearch/intervalMinutes）、`store.vector`（sqlite-vec）、`chunking`、`query`
- 结论：**内置引擎锁死 SQLite 文件索引，没有外部记忆库 provider 槽位。**

### 2.2 MCP servers —— 外部记忆框架的标准挂载点 ✅
- 顶层 `mcp.servers`（object，key = server 名），每个 server 可配：
  - `command` + `args` + `env` + `cwd`（**stdio 本地进程**）
  - `url` + `transport`（`stdio` | `sse` | `streamable-http`）（**远程 MCP**）
  - `headers`（HTTP 鉴权头）、`timeout`/`connectTimeout`/`requestTimeoutMs`
  - `supportsParallelToolCalls`、`auth: "oauth"` + `oauth` 配置
- 证据：ClawHub `titen-memory`/`lithtrix-memory`/`ecloud-long-term-memory` 均以 MCP server 接外部记忆；docs 明确"记忆插件可以是 memory-core、Honcho、和 others"。

### 2.3 memory 插件（registerMemoryCapability）—— 深度替换，非首选
- `memory-core` 插件声明 `kind: "memory"` + `contracts.tools: [memory_get, memory_search]`，通过 `api.registerMemoryCapability({promptBuilder, flushPlanResolver, runtime, publicArtifacts})` 挂载。
- Honcho 插件（官方文档提及）未随 ClawX 打包，需 ClawHub 安装 —— 证明 memory 插件是**开放扩展点**，但实现成本高、协议在演进。
- 结论：可作为后续架构目标，但**当前不推荐**（见 §5 风险）。

### 2.4 本机现状
- `~/.openclaw/openclaw.json`：`memorySearch.provider = "none"`；`mcp.servers` 未配置；`tools.web.fetch` 允许私有网络（agent 可直接 fetch Garden）。

---

## 3. Garden 端可暴露的能力（已核对现有 API）

| Garden 领域 | 现有 API | 对应 MCP 工具 | 备注 |
|---|---|---|---|
| Persona 人格 | `GET /v2/persona/status`, `/files/{kind}`, `/history` | `persona_status`, `persona_get`, `persona_list` | 只读；写路径下一 slice |
| Recall 检索 | `GET /v2/recall/fast`, `/deep`, `/bootstrap`, `/traces/*` | `memory_search`, `memory_get` | 语义召回 |
| Memories | `GET /v2/memories`, `/v2/memories/{id}` | `memory_get` | 持久记忆条目 |
| Mentle 证据 | `GET /v2/materials/cards`, `/collections` | `evidence_search`, `evidence_get` | 证据卡片 |
| Activity | `GET /v2/activity/events` | `activity_recent` | 活动流 |
| Reports | `GET /v2/reports`, `/latest` | `report_get` | 报告 |

> 注意：Garden 当前只有 **read** API（Persona 写入路径尚未实现，是下一 slice）。因此本方案**第一阶段为只读接入**；写路径（`record_signal`/checkpoint/反向同步）需等 Garden 写 API 落地。

---

## 4. 推荐方案：Garden MCP Server

### 4.1 架构
```
OpenClaw agent
   │  mcp.servers["garden"]   (stdio 或 streamable-http)
   ▼
garden-mcp（新进程：Go stdlib 或 Node @modelcontextprotocol/sdk）
   ├── persona_status / persona_get / persona_list    → GET /v2/persona/*
   ├── memory_search / memory_get                     → GET /v2/recall/*, /v2/memories/*
   ├── evidence_search / evidence_get                 → GET /v2/materials/*
   └── activity_recent                                → GET /v2/activity/events
   ▼
Garden HTTP API（现有 /v2/* 或新增的 MCP 专用端点）
```

### 4.2 形态选择
| 传输 | 优点 | 缺点 | 适用 |
|---|---|---|---|
| **stdio**（`command: garden-mcp` 本地进程） | 零网络、免认证、最简 | 与 Garden server 分开启动 | ✅ 最小闭环（推荐先做） |
| **streamable-http**（`url` 指向 Garden 自带端点） | agent 与 server 天然同源、可远程 | 需要认证（headers/token） | ✅ 完整方案（后续） |

### 4.3 实现方式（二选一）
- **A. Go 实现**：Garden repo 内新增 `garden/mcp/`（Go `@modelcontextprotocol/go-sdk` 或手写 JSON-RPC），与 Garden server 同仓同源。✅ 推荐（进 repo、可测试）。
- **B. Node 实现**：独立 `garden-mcp/` node 工程，用官方 `@modelcontextprotocol/sdk`，HTTP 调 Garden。✅ 更快（SDK 成熟），但多一个运行时。

### 4.4 工具契约草案（对齐 MCP 规范 + titen 语义）
| 工具 | 参数 | 返回 | 数据源 |
|---|---|---|---|
| `persona_status` | `-` | 7 文件状态 + profile 状态 | `/v2/persona/status` |
| `persona_get` | `kind` | 该文件 Markdown 全文 + rev/hash | `/v2/persona/files/{kind}` |
| `persona_list` | `-` | 文件清单 + 状态 | `/v2/persona/status` |
| `memory_search` | `query, maxResults, corpus` | 召回结果（含来源/相关度） | `/v2/recall/fast` + `/v2/memories` |
| `memory_get` | `id/path, from, lines` | 精确条目 | `/v2/memories/{id}` |
| `evidence_search` | `query, limit` | 证据卡片 | `/v2/materials/cards` |
| `evidence_get` | `id` | 单卡片全文 | `/v2/materials/cards/{id}` |
| `activity_recent` | `limit` | 近期活动事件 | `/v2/activity/events` |
| `record_signal` *(后续)* | `text, tags, source` | 写入确认 | 需 Garden 写 API（Persona/ACTMEM 下一 slice） |

### 4.5 最小闭环（Phase 1，0.5-1 天）
1. 在 Garden repo 建 `garden/mcp/`，实现 **stdio** MCP server，暴露 `persona_status` + `persona_get` + `memory_search` 3 个工具（桥现有 `/v2/*`）
2. `openclaw.json` 加 `mcp.servers.garden = { command: ".../garden-mcp", transport: "stdio" }`
3. 验证：`openclaw agent --message "读一下我的 Garden 人格是什么"` → agent 调 `persona_get` → 返回 IDENTITY.MD 内容
4. 再验证：`memory_search` 能召回 Garden 的 Recall/Memories 结果

### 4.6 完整方案（Phase 2+，3-5 天）
- 全部工具（§4.4）落地
- 认证：Garden MCP 端点加 token（headers 传递），streamable-http 形态
- 写路径：Garden Persona/ACTMEM 写 API 落地后，加 `record_signal`/checkpoint（对齐 titen 的 record/checkpoint 语义）
- 与内置 memory 的关系：Garden MCP 管"权威记忆"；openclaw 内置 SQLite 管"会话/本地杂项"，两者并存不冲突

---

## 5. 备选方案与取舍

### 5.1 备选 A：memorySearch.extraPaths（数据接入）
- 把 Garden 导出的 Persona/Recall 摘要放进 openclaw 索引目录 → 内置引擎检索。
- 优点：零 MCP 开发、纯配置。
- 缺点：单向快照，非"活的 provider"；Garden 侧仍需导出逻辑。
- 适用：作为 MCP 方案的**补充**（Garden 同步一份快照供内置引擎兜底）。

### 5.2 备选 B：registerMemoryCapability 插件（真 provider）
- 写 `kind: "memory"` 插件替换 `memory_search/get` → 桥 Garden。
- 优点：彻底替换内置引擎，最"provider"。
- 缺点：要逆向完整契约（promptBuilder/flushPlanResolver/runtime）；OpenClaw memory 内部在演进（`refactor/database-first`），协议可能变；且**已有 MCP 标准路径不用**，重复造轮子。
- 结论：**不推荐当前做**，除非未来有强需求要完全隐藏内置引擎。

---

## 6. 决策请求（等你拍板）

1. **Phase 1 采用 MCP 方案（§4.5 最小闭环）**？还是先看备选 A/B？
2. **MCP server 用 Go（进 Garden repo）还是 Node（独立工程）**？
3. **传输形态**：先 stdio 本地进程（最简），还是直接上 streamable-http（要认证）？
4. **工具范围**：最小闭环先做 `persona_status/persona_get/memory_search` 3 个，还是直接做全量 8 个？
5. **只读单向**：先只读（推荐，等 Persona 写路径），还是现在就规划写路径 `record_signal`？

---

## 附：Phase 1 实施记录（2026-08-18 完成）

### 已交付
- **代码**：`garden/cmd/garden-mcp/main.go` —— stdio MCP server（Go, mcp-go v0.58.0），暴露 **8 工具**：
  - **只读**：`persona_status` / `persona_get` / `memory_search` / `evidence_search` / `evidence_get` / `evidence_collections` / `activity_recent`
  - **写**：`record_signal`（→ `POST /v2/memories`，source=agent）
- **二进制**：`garden/garden/bin/garden-mcp.exe`（`go build -o bin/garden-mcp.exe ./cmd/garden-mcp/`）
- **OpenClaw 配置**：`~/.openclaw/openclaw.json` → `mcp.servers.garden = { command, cwd, timeout }`（stdio）

### 工具 → API 映射
| MCP 工具 | Garden API | 读写 |
|---|---|---|
| `persona_status` | `GET /v2/persona/status` | 读 |
| `persona_get` | `GET /v2/persona/files/{kind}` | 读 |
| `memory_search` | `POST /v2/recall/fast` + `GET /v2/memories` | 读 |
| `evidence_search` | `GET /v2/materials/cards?query=` | 读 |
| `evidence_get` | `GET /v2/materials/cards/{id}/evidence` | 读 |
| `evidence_collections` | `GET /v2/materials/collections` | 读 |
| `activity_recent` | `GET /v2/activity/sessions/{id}` | 读 |
| `record_signal` | `POST /v2/memories`（content/kind/tags，source=agent） | **写** |

### 验证结果（全部通过）
1. **协议层**：`e2e-tmp/mcp_stdio_test.py`（8 工具全调 + record_signal 写后 memory_search 验证持久化）✅
2. **OpenClaw 连接**：`openclaw mcp probe` → `garden: 8 tools` ✅
3. **端到端（只读）**：agent 调 `evidence_collections`/`evidence_search`/`activity_recent` → 正确返回 ✅
4. **端到端（写）**：agent 调 `record_signal` 写入记忆 → `memory_search` 召回验证（agent 曾尝试 kind=signal 遇 400 后自动 fallback 到 note）✅

### 人格设定（"记忆花园的守护者"）
- 工具：`e2e-tmp/set_persona.py` —— 一次性脚本，把 5 个必需 persona 文件（IDENTITY/RELATIONSHIP/REDLINE/USER/WORLD）从 rev1 更新到 rev2，并同步历史（snapshot + diff + log.jsonl entry）
- 内容主题：沉稳克制的记忆守护者；证据为语言、事实为边界；红线=不伪造/不破坏/不臆测/不泄露；与用户是长期信任的协作伙伴
- 验证：
  - API status 保持 `ready`（rev2，历史一致，无 history_mismatch）
  - history 端点可追溯 rev1→rev2（hash/actor/source/reason/base_revision 完整）
  - **端到端**：openclaw agent 通过 MCP 读取 5 文件，以第一人称精准复述人格（职责/红线/关系）—— 人格被 agent 真正内化 ✅
- 关键契约：修改 persona 文件必须同步更新 `history/<FILE>/log.jsonl`（content_hash 与实际内容 sha256 一致），否则 status 变 `history_mismatch`

### 实现要点 / 坑
- `record_signal` 的 `source` 是**对象**（`{type: agent}`），不能是字符串；合法 type: user/agent/session/import/report_projection
- memory `kind` 合法枚举：note/fact/preference/decision/session_digest/source_artifact/semantic_unit（默认 note）
- mcp-go v0.58 用 `mcp.WithArray("tags", mcp.WithStringItems())` 定义字符串数组（无 `WithStringArray`）
- `recall/fast` 是 **POST**（非 GET），body `{query, budget_chars}`

### 运行依赖
- Garden server 需在 `127.0.0.1:7373` 运行（garden-mcp 是它的 loopback 客户端）
- MCP server 每次被 openclaw spawn（stdio），无常驻进程

### 后续（Phase 2+）
- streamable-http 形态 + token 认证
- Garden Persona/ACTMEM 写路径落地后，`record_signal` 可扩展对齐 titen 的 checkpoint/lease 语义
- 备选：memorySearch.extraPaths 同步快照作兜底

## 附：已核实的技术事实来源（未动任何代码）
- OpenClaw 2026.7.1 (2d2ddc4)：`openclaw --help`、`docs search "mem0"/"honcho"/"mcp servers"/"memory"`、`config schema`
- MCP 配置：`openclaw.json` → `mcp.servers` schema（command/args/env/cwd/url/transport/headers/timeout/supportsParallelToolCalls/auth:oauth）
- memorySearch 配置：`agents.defaults.memorySearch` schema（store.driver=sqlite const、provider 枚举、remote、extraPaths、sync、query、chunking）
- 外部记忆框架证据：ClawHub skills search（`titen-memory`/`lithtrix-memory`/`ecloud-long-term-memory` 均 MCP 形态）；docs `memory-honcho`
- memory-core 插件契约：`dist/extensions/memory-core/{index.js, api.js, openclaw.plugin.json, manager-runtime.js}`
- Garden 路由：`garden/internal/server/*.go` 的 `/v2/*` 清单
- 本机 openclaw 配置：`~/.openclaw/openclaw.json`
