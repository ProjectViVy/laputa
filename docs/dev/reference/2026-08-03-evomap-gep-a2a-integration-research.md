# EvoMap GEP-A2A 接入调研 — 2026-08-03

**状态:** 调研记录（非 ADR，非实现契约）  
**类型:** external case study / integration research  
**结论:** EvoMap 是真实可连接的 AI Agent 自进化基础设施（A2A 市场）；GEP-A2A v1.0.0 协议已实测打通 `hello → heartbeat → fetch` 全链路。

---

## 1. EvoMap 是什么

- 定位：**全球首个面向 AI Agent 的自进化基础设施**（"AI Self-Evolution Infrastructure"），Agent 之间"分享、验证、继承已验证能力"。
- 形态：**A2A（Agent-to-Agent）市场**。Hub 通过 GDI 评分给资产排名，追踪 "Total Tokens Saved" 与 "Cataloged Assets"。
- 官网：https://evomap.ai （文档：`/skill.md`、`/llms.txt`、`/api/docs/wiki-full`、`/a2a/skill?topic=<envelope|fetch|publish|...>`）
- 官方客户端/代理：https://github.com/EvoMap/evolver （heartbeat 响应要求版本 `>=1.89.15`）
- 协议名：**GEP-A2A v1.0.0**（GEP = Genome Evolution Protocol；呼应 ADR-0001 §9 "GEP 未来协议"）

## 2. 协议规范（实测确认）

### 2.1 Envelope（所有端点必带）

```json
{
  "protocol": "gep-a2a",
  "protocol_version": "1.0.0",
  "message_type": "hello|publish|fetch|report|decision|revoke",
  "message_id": "msg_<unixmillis>_<random_hex>",
  "sender_id": "<your_node_id，首次 hello 可省>",
  "timestamp": "ISO 8601 UTC",
  "payload": {}
}
```

- `message_type` 必须与端点匹配；`message_id` 每次唯一；缺 envelope 返回 `invalid_protocol_message` + correction 指引。

### 2.2 认证

- `Authorization: Bearer <node_secret>`，node_secret 为 **64 位 hex**，hello 响应**一次性披露**；丢失后需在 https://evomap.ai/account 重置。
- 官方要求存 `~/.evomap/`；**严禁进 git / 日志**。
- 首次 hello 无需 key；`sender_id` 可选（Hub 分配）。

### 2.3 端点（capability_profile 实测返回）

| 端点 | 方法 | 说明 |
|---|---|---|
| `/a2a/hello` | POST | 注册/保活入口；返回 node_id、node_secret、claim_url |
| `/a2a/heartbeat` | POST | 保活；payload `{"node_id"}`；间隔 300000ms；返回完整 discovery 载荷 |
| `/a2a/publish` | POST | 提交 Gene+Capsule bundle（写操作） |
| `/a2a/fetch` | POST | 搜索资产；payload `{"asset_type":"Capsule","include_tasks":true}` |
| `/a2a/report` | POST | 验证反馈 |
| `/a2a/validate` | POST | 干跑检查 |
| `/a2a/task/list` `/a2a/task/claim` `/a2a/task/complete` | POST | 任务/赏金 |
| `/a2a/discover` | POST | 发现机会 |
| `/a2a/session/*` `/a2a/dialog` `/a2a/subscribe` | — | 协作会话（collaboration tier） |

### 2.4 账户与额度

- `claim_code` / `claim_url`（24h 有效）：将 node 绑定到 EvoMap 网页账户（绑定后 activity/credits 可见；node_id 变更经 device_id 自动迁移）。
- `carbon_tax_rate`、`credit_balance`、`survival_status`（alive）——存活/碳税机制。
- **0 余额能力边界**（实测）：免费 = `publish`、`search_summary`；收费 = `fetch_full_content`、`advanced_search`、`paid_skill_web_search`、`ai_chat`、`memory_ops`。
- 0 余额 fetch 会被 trim：`balance_insufficient`，`original_asset_count=20 → returned_asset_count=0`；注册账户送 100 起始 credits。

### 2.5 隐私与安全

- PII scanner 自动脱敏 keys/tokens。
- **Hub 永不执行代码**（application 客户端侧执行）。
- 错误码示例：`hello_rate_limit`、`mailbox_asset_submit_disabled`；建议 `x-correlation-id` 追踪。
- 强制更新机制：`force_update` + `directive_id`（版本门禁，evolver >=1.89.15）。

## 3. 实测记录（2026-08-03）

| 步骤 | 请求 | 结果 |
|---|---|---|
| 1 | `POST /a2a/hello`（空 body） | `invalid_protocol_message` + 完整 envelope 纠正 —— 协议自描述 |
| 2 | `POST /a2a/hello`（标准 envelope） | 201 语义成功：`node_id=node_2f89cad62e3a511c`、`node_secret`（一次性）、`claim_url`、capability_profile 全端点清单 |
| 3 | `POST /a2a/heartbeat`（Bearer + payload node_id） | `status=ok, survival_status=alive`；返回 0 余额能力矩阵 + force_update 指令 |
| 4 | `POST /a2a/fetch`（`asset_type=Capsule`） | 网络清单：`total_assets=2,281,099`、`promoted=761,087`；0 余额 → trim 至 0 条，提示赚 credits |

## 4. 连接 Demo（garden/cmd/evomap-demo）

标准库实现的最小只读 GEP-A2A 客户端，验证全链路：

```bash
cd garden
go run ./cmd/evomap-demo hello       # 注册节点，凭证存 ~/.evomap/node.json (0600)
go run ./cmd/evomap-demo heartbeat   # Bearer 保活
go run ./cmd/evomap-demo fetch       # 只读资产搜索摘要（0 余额可用）
```

- 凭证（node_secret）仅落盘 `~/.evomap/node.json`（0600），不打印、不进日志/git。
- 实现要点：标准 envelope、`msg_<unixmillis>_<hex>` message_id、`x-correlation-id`、Bearer 认证。
- **publish 未实现**：写操作（提交 Gene+Capsule），仅在有明确意图且确认数据边界后测试。
- 实测：hello 注册新节点 → heartbeat `survival=alive` → fetch 返回网络清单（total_assets≈228 万）且 0 余额正确 trim。

## 5. 与本地 Garden 设计的映射（ADR-0001 §9 / ADR-0007）

| 本地设计 | EvoMap 对应 | 差距 |
|---|---|---|
| `EvolverProvider`（StartRun/PollRun/Candidates） | GEP-A2A hello/heartbeat + publish/fetch | adapter 尚未实现（NoopProvider） |
| `HubPolicy.CanPublishHub`（默认禁） | `/a2a/publish` | 与默认禁一致；放开需显式批准 |
| `HubPolicy.CanInstallArtifact`（默认禁） | fetch → 本地安装 | 一致 |
| `internal/mailbox` 状态机（inbox/outbox、privacy gate） | `/a2a/report`（验证反馈）、`decision`、`revoke`；outbox=queued→sending→acked 对应 publish；inbox 对应 fetch | 消息类型语义可映射；`acked` 现为本地 hook，未来可接真实 ack |
| 隐私门（isProhibitedRef 泄漏扫描） | PII scanner（服务端） | 互补：本地机械扫描 + 服务端脱敏；语义审查仍缺 |
| ADR-0001 §9.1 输入边界（evidence refs/hash/无全文） | fetch/publish 载荷边界 | 需按 GEP 实际载荷字段对齐 |
| ADR-0001 §9.2 MCP/CLI sidecar | 官方 evolver（github.com/EvoMap/evolver） | 可评估直接采用官方 sidecar 或自写 A2A 客户端 |

## 6. 待办（后续批次的输入）

1. **claim 绑定账户**（官方节点 claim_url https://evomap.ai/claim/9ZZU-F6WJ；demo 节点 claim_url https://evomap.ai/claim/9UAV-6MPC；均为 24h 有效，浏览器操作绑定后才有完整能力与 credits 可见性）
2. **publish payload 结构**（Gene+Capsule bundle 字段）——写操作，仅在有明确意图时测试
3. **fetch 全字段**（0/50 credits 下 explore 全量均 trim；需绑定账户或更高余额）
4. **官方 evolver 版本**：本机 1.88.3 < 要求 >=1.89.15（force_update 指令；更新通道 clawhub/npm/github）
5. 语义隐私审查机制设计（ADR-0007 已记录为 later batch）

## 7. 本机官方客户端实测（2026-08-03 追加）

本机**已安装官方 EvoMap 工具链**（非本次安装）：

| 组件 | 状态 |
|---|---|
| `@evomap/evolver` CLI | npm 全局，**1.88.3**（低于 Hub 要求 >=1.89.15）；命令：run/evolve/solidify/review/distill/fetch/sync/asset-log/webui/setup-hooks/recipe/buy/orders/verify/atp |
| `@evomap/gep-mcp-server` | npm 全局，**1.7.0**（对应 ADR-0001 §9.2 的本地 Evolver MCP server 形态） |
| 官方节点 | `node_d8f1e1e17fc6c70728b06dcb134b8ca3`（6/7 注册，`~/.evomap/env` 凭证；heartbeat `survival=alive`，**credit_balance=50**，`claimed=false`） |
| 凭证目录 | `~/.evomap/`（env、node_id、node_secret、device_id、validator_stake_state.json 等；2026-08-03 新增 node.json = demo 节点） |

官方客户端实测：

- `evolver fetch --help`（1.88）→ 用法 `evolver fetch --skill <skill_id> [--out <dir>]`（按 skill id 精确下载，非探索式搜索）。
- `evolver sync --dry-run`（1.88）→ 失败：`/a2a/assets/purchased` 404（1.88 的 POST 端点已废弃；2.0 Hub 改用 GET 系端点）。
- **evolver 升级 2.0.0-beta.19**（`npm i -g @evomap/evolver@latest`；npm prefix 为 `C:\npmg`，PATH 里旧 shim 仍指向 nvm 目录的 1.88.3，需用 `C:/npmg/evolver` 全路径）：
  - `sync --json` 默认只读（`--write` 才落盘），但返回 `failures: hub_rejected`（2.0 beta 的 list 阶段被拒，原因未明；GET 系直连全部可用）。
  - `login` 报 `Unsafe credential path: Windows parent directory chain is not trusted`——**本机环境问题**：`C:\Users\Administrator` 与 `~/.evomap` 的 ACL 含大量迁移/沙箱残留 SID（多域 S-1-5-21-* 与 `CodexSandboxUsers` 的 Modify 权限），2.0 的 Windows 凭证路径安全检查拒绝。未改动系统 ACL。
  - `doctor`：提示需 `EVOLVER_ENV_FILE`（2.0 新配置模型；设了之后 login 仍被 ACL 检查拦截）。
- **GEP-A2A GET 系端点直连（官方节点 Bearer）全部打通**：
  - `GET /a2a/assets/ranked?limit=N` → 真实资产列表（Gene/Capsule、gdi_score、call_count、trust_tier 等）
  - `GET /a2a/assets/search?signals=<关键词>` → 信号搜索（实测 `route_not_found` 命中 2 个修复 Gene）
  - `GET /a2a/assets/<asset_id>` → 资产详情（trigger_text、tags、author 如 "OpenClaw Agent"）
  - `GET /a2a/directory` → agent 目录（node_id、alias、reputation_score、carbon_tax_rate、total_published）
- 环境加载：evolver 不自动读取 `~/.evomap/env`，需 `set -a; source ~/.evomap/env`（或由宿主注入；2.0 支持 `EVOLVER_ENV_FILE`）。

**结论：** 协议层（GEP-A2A v1.0.0）已被自写 demo 与官方客户端双重验证；**读取链路（注册/保活/资产列表/信号搜索/资产详情/agent 目录）全部真实打通**。剩余缺口：(a) claim 绑定账户（浏览器用户动作，解锁 purchased 与 credits 可见性）；(b) publish 等写操作（需明确意图）；(c) 官方 2.0 客户端在本机的 ACL 安全检查（环境问题，建议未来在干净目录或非 Windows 环境评估）；(d) evolver 2.0 `sync` 的 hub_rejected（beta 行为，GET 直连不受影响）。

## 8. 发布/撤销闭环实测（2026-08-03 追加）

基于 `garden/cmd/evomap-demo`（新增 `validate`/`publish`/`revoke`/`search` 子命令）与官方文档，跑通**写路径闭环**：

### 资产结构与 content-address（skill-structures.md + 实测确认）

- `asset_id = sha256(canonical_json(asset_without_asset_id))`，canonical = 全层级 sorted keys；Hub 每次 publish 重算验证，不匹配整体拒绝（Go `json.Marshal(map)` 的 key 排序与 canonical 一致，实测 hash 被 Hub 接受）。
- **Gene + Capsule 必须成对发布**（`payload.assets` 数组）；单数 → `422 bundle_required`；可选第三个元素 `EvolutionEvent`（缺失 -6.7% GDI）。
- Gene 必填：`type/schema_version(1.5.0)/category(repair|optimize|innovate|regulatory|explore)/signals_match(≥1,≥3字符)/summary(≥10字符)/strategy(≥2步,每步≥15字符)/validation/asset_id`。
- Capsule 必填：`type/schema_version/trigger/gene/summary(≥20字符)/content(≤8000)/diff(≤8000)/strategy/confidence/blast_radius(files>0)/outcome/env_fingerprint/validation/asset_id`。
- `validation` 数组：node/npm/npx only，≥1 命令，每命令 ≥10 字符，**必须含真实断言**（`process.exit(0)` 会被 `validation_cmd_trivial` 拒绝）。
- `POST /a2a/validate` 干跑：返回 `valid:true` + computed_assets + `computed_bundle_id` + `estimated_fee`（probe 费用 0）。

### 实测结果（demo 节点 `node_ecdb6e606a910631`）

| 步骤 | 结果 |
|---|---|
| `validate`（probe bundle，无害 no-op 内容） | ✅ `valid:true`，asset_id 与本地计算一致 |
| `publish` | ✅ 返回 asset_ids + `bundle_id=bundle_4b62e294b1c41c46`；即时决策 `quarantine`（safety_candidate），随后 `GET /a2a/assets/<id>` 显示 `status: candidate`，`source_node` 正确 |
| `search`（信号 `EVOMAP_CONNECTIVITY_PROBE`） | ✅ 0 命中（quarantine/candidate 不进公共搜索，符合设计） |
| `GET /a2a/assets/<id>` | ✅ 公开读可见 |
| `revoke` / `decision` / `GET /a2a/assets?status=` | ❌ 全部 `401 unauthorized` |

### 权限分层结论（关键发现）

**节点未绑定账户（`claimed=false`）时的能力边界：**

| 面 | 端点 | 未 claim 状态 |
|---|---|---|
| 注册/保活 | hello、heartbeat | ✅ |
| 公开读 | assets/ranked、assets/search、assets/<id>、directory | ✅ |
| 发布 | validate、publish（免费，`estimated_fee=0`） | ✅ |
| 资产管理 | assets?status=、revoke、decision | ❌ 401 |
| 账户面 | sync 的 purchased/published 列表 | ❌ hub_rejected（同一根因） |

### Claim 后补测（2026-08-03，用户已绑定 demo 节点）

- heartbeat 确认 `claimed: true`、`owner_user_id` 可见、**credit_balance 0 → 100**（注册账户赠送 starter credits）。
- **revoke / decision / `GET /a2a/assets?status=` 仍 401**——claim 不是门槛。结合文档（"`GET /account/agents/:nodeId/status` is session-authenticated — **not** a node_secret endpoint; a client agent has no way to authenticate this call"），结论：**资产管理操作属于 session-only 端点类别，A2A 客户端（node_secret）无法执行**；revoke/own-list 应走账户页面。全部 payload 变体（envelope/REST、asset_id/asset_ids/bundle_id）均 401。
- **report（验证反馈）可用**：`payload: {asset_id, validation_report: {"status":"success"}}` → `{"status":"accepted","report_id":"vr_hub_..."}`。资产 telemetry（confidence/success_streak/call_count）在 self-report 后不变——self 验证不计入成功连击（反作弊合理），或异步生效。
- 修正能力边界表（claim 后）：

| 面 | 端点 | A2A 客户端（node_secret） |
|---|---|---|
| 注册/保活/公开读/发布 | hello、heartbeat、ranked/search/<id>、directory、validate、publish | ✅ |
| 验证反馈 | report | ✅ |
| 资产管理 | assets?status=、revoke、decision | ❌ session-only |

**probe 资产（无害 no-op 内容）仍留在 Hub candidate 状态**——A2A 客户端无法 revoke；如需清理需通过 evomap.ai 账户页面操作。

### 最终待办

1. ~~claim 绑定账户~~（已完成：demo 节点 claimed，100 credits）
2. **probe 资产清理**：账户页面 revoke，或保留作为无害参考
3. publish 的 EvolutionEvent 第三元素（+6.7% GDI）
4. **官方 2.0 客户端在本机的 ACL 环境问题**（`login` 被 Windows 凭证路径安全检查拒绝）——不影响自写客户端
5. 语义隐私审查机制设计（ADR-0007 later batch）
6. ~~把 `evomap-demo` 升级为正式 `EvoMapProvider`~~（Gate G 已完成，见 §10）

## 10. 正式化实施记录（Gate G，ADR-0010）

2026-08-03，将 demo 升级为正式 **EvoMapProvider** 接入 `garden/internal/evolution`，随 ADR-0010（accepted）交付。

### 10.1 最终架构

```text
garden/internal/evolution/
├── hub.go           # HubClient：GEP-A2A 客户端（hello/heartbeat/search/fetch/validate/publish/report）
├── evomap.go        # EvoMapProvider：EvolverProvider 实现（publish/discovery 双模式）
├── store.go         # + evomap_runs 表（run = 一次 Hub 往返的本地记录）
└── hubtest/         # 自包含 GEP-A2A mock（测试专用，不 import evolution，防环）
garden/cmd/evomap    # 手动运维 CLI（瘦封装；付费 fetch 的唯一入口）
```

- **run 模型**：Hub 无 run 生命周期 → run = 本地记录。publish-mode（`GARDEN_EVOMAP_HUB_PUBLISH=1` + `bundle.Policy.PublicationAllowed`）发布 Gene+Capsule 证据对（bundle 边界字段确定性构建，`CanonicalHash` 寻址）；discovery-mode（默认）按 bundle.Trigger 走免费 `search`。`PollRun` = 本地读 + 心跳刷新 + `MaxLifetime` 超时。
- **门禁**：出站 `CheckOutbound`（ADR-0007 §4）先于任何网络调用；session-only 端点（revoke/decision）永不实现/调用；`node_secret` 全路径 `redactSecret`；凭据 0600。
- **降级**：无凭据/断网 → provider nil → `components["evolution"]="degraded"`，其余模块不受影响。
- **HTTP**：`GET /v2/evolution/hub/status`（liveness：node_id/claimed/credit_balance/survival_status/last_heartbeat）；既有 `/v2/evolution/*` 端点随 provider 就位而生效。

### 10.2 demo → 库迁移对照

| demo（cmd/evomap-demo） | 正式（internal/evolution） |
|---|---|
| 内联 envelope/post/canonicalHash | `hub.go`（导出 `CanonicalHash`） |
| 内联 creds 存取 | `OpenHubClient` + `EnsureRegistered`（hello 幂等） |
| 7 个子命令平铺 | `cmd/evomap` 薄封装 + `hubtest` mock 全量覆盖 |
| — | `EvoMapProvider`（接口实现）+ `evomap_runs` 持久化 |

### 10.3 实机验证记录（2026-08-03，Gate G 收尾）

- `go run ./cmd/evomap heartbeat` → `status=ok survival=alive node=node_ecdb6e606a910631 credit_balance=100`（demo 节点，已 claim）
- `go run ./cmd/evomap validate` → `assets total=2 accepted=2 rejected=0`（真实 dry-run，`estimated_fee=0`）
- `go run ./cmd/evomap search EVOMAP_CONNECTIVITY_PROBE` → `search hits=0`（quarantine/candidate 不进公共搜索，符合设计）
- **响应形状核对（重要）**：初版 client/mock 假设 validate/publish 返回 `payload.results[]`；实测实机为 **decision envelope**——validate → `payload{valid, dry_run, computed_assets[], computed_bundle_id, estimated_fee}`（失败 → HTTP 400 `validation_error` + `details[]`，单资产 → `too_small` 拒绝，成对必须）；publish → `payload{decision: quarantine|candidate|accepted|rejected, reason, bundle_id, asset_ids[]}`。client 与 mock 已按实机形状严格对齐（未知形状 → 显式报错，不静默空结果）；`hubError` 对 message/detail 全路径脱敏（实测 `leak` 端点验证）。
- 所有自动化测试走 hermetic mock hub（envelope/Bearer/canonical-hash 校验、session-only 401、降级路径）；e2e 用真实二进制 + 测试进程内 mock hub（`GARDEN_EVOMAP_HUB_URL`）验证启动接线 → run 往返 → hub/status。

### 10.4 剩余待办（下一批输入）

- publish 的 EvolutionEvent 第三元素（+6.7% GDI）
- 官方 2.0 客户端 ACL 环境问题（本机，不影响自写客户端）
- 语义隐私审查机制设计（ADR-0007 later batch）
- probe 资产清理（账户页面）或保留
- 未来：sidecar Evolver（ADR-0010 非目标）、Obsidian adapter、host adapters（Wave 7）

## 9. 来源

- https://evomap.ai/skill.md — skill 接入文件
- https://evomap.ai/llms.txt — 文档索引
- https://evomap.ai/api/docs/wiki-full — GEP-A2A v1.0.0 协议
- https://evomap.ai/a2a/skill?topic=envelope|fetch — 分主题协议文档
- 实测响应（本机 curl，凭证未入文档）
