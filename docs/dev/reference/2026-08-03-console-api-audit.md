# Console API 综合审查（Gate H）

**日期:** 2026-08-03
**范围:** `garden/console`（React SPA）× `garden/internal/server`（51 条注册路由）全量交叉审查
**目标:** 确保前端完全可用——调用面零缺口、契约匹配、已知缺陷修复、Gate F/G 成果可见

---

## 1. 审查矩阵（前端实际调用 × 后端契约）

前端实际发起的 14 个端点，全部存在且契约匹配：

| 前端端点 | 调用点 | 后端 handler | 结论 |
|---|---|---|---|
| `GET /health` | TopBar.tsx（10s 轮询） | handleHealth | ✅ |
| `GET /v2/admin/overview` | Overview / GovernanceMap / Operations / SafetyBanner | handleAdminOverview | ✅（双轮询见 §4） |
| `GET /v2/admin/audit?limit=` | Overview / Operations | handleAdminAudit | ✅ |
| `GET /v2/admin/components` | Operations | handleAdminComponents | ✅ |
| `GET /v2/admin/spool` | Operations | handleAdminSpool | ✅ |
| `GET /v2/pipelines` | Operations | handlePipelines | ✅ |
| `GET /v2/recall/traces/{id}` | RecallTrace | handleRecallTrace | ✅（404 路径实测） |
| `GET /v2/materials/collections` | MaterialsPage | handleMaterialsCollections | ✅ |
| `GET /v2/materials/cards?query=` | MaterialsPage | handleMaterialsCards | ✅（仅非空查询才发请求，与后端 query 必填不冲突） |
| `GET /v2/materials/cards/{id}/evidence` | MaterialsPage | handleMaterialsEvidence | ✅（形状实测） |
| `GET /v2/reports?cadence=&limit=` | ReportsPage | handleReportsList | ✅ |
| `GET /v2/reports/modules?kind=&status=` | ModulesPage | handleModulesList | ✅ |
| `POST /v2/reports/modules` | ModulesPage | handleModuleCreate | ✅ |
| `PATCH /v2/reports/modules/{id}` | ModulesPage | handleModuleUpdate | ⚠️→✅（created_at bug，见 §2.3） |

基础设施：SPA fallback 正常（`/evolution` 深链刷新返回 index.html，不 404）；同源 embed serve 无需 CORS（后端无 CORS 头，dev 走 vite proxy）；前端无内嵌密钥、无 `dangerouslySetInnerHTML`。

## 2. 发现的缺陷与修复

### 2.1 ModulesPage dismiss/reactivate 按钮竞态（已修）

`ModulesPage.tsx` 的 create/save 按钮有 `disabled={busy}` 防护，dismiss/reactivate 没有——快速双击穿透 stale `busy` 闭包发重复 PATCH。补齐 `disabled={busy}`。

### 2.2 client.ts 无条件 JSON 解析（已修）

`send()` 成功路径无条件 `res.json()`；204/空 body 端点会抛解析错。加 `status===204 || content-length==="0"` 防护返回 `undefined`。

### 2.3 module PATCH 丢失 created_at（实机发现，已修）

实机 PATCH 后响应 `created_at:"0001-01-01T00:00:00Z"`。根因：`report/modules.go` `UpdateModule` 读取行时 scan 了 created_at 字符串但从未解析进 `m.CreatedAt`。修复 + 回归测试（`modules_test.go` 断言 update 保留 CreatedAt）。DB 数据无损，仅响应错误；前端列表重载走 ListModules（解析正确）故界面侥幸未坏，但直接 PATCH 响应契约已破。

### 2.4 未定义的 CSS token `--ink-1`（已修）

`pages.css` 6 处引用 `var(--ink-1)` 但 tokens.css 从未定义（静默回落继承色）。tokens.css 补 `--ink-1: var(--ink)`。

## 3. 覆盖缺口与本批新增

### 3.1 Evolution 面板（新增）

Gate F/G 成果此前在 console 完全不可见：`/v2/evolution/*` 全部 8 条路由零前端调用。新增 `/evolution` 页面（EvolutionPage.tsx）：

- **Hub 状态卡**：`GET /v2/evolution/hub/status`（node_id / claimed / claim 链接 / credits / survival / heartbeat）；503 → 降级横幅
- **Runs**：新 `GET /v2/evolution/runs` 列表（8s 轮询）；start-run 表单（trigger 必填，provider 降级时禁用）；行展开 → 详情（4s 轮询）+ candidates → 生成提案
- **Proposals**：新 `GET /v2/evolution/proposals` 列表；详情展示 leakage_report；pending 提案 approve/reject（两步确认防误触）

### 3.2 后端补 list 端点（新增）

面板无法枚举 runs/proposals（原只有按 id 读），补两条只读路由（读现有 SQLite 表，`{items[], count}`，limit 默认 20 / 1-100 / 越界 400）：

- `GET /v2/evolution/runs`（与 `GET /v2/evolution/runs/{run_id}` 共存，Go 1.22 精确模式优先，实测无冲突）
- `GET /v2/evolution/proposals`

纯增量只读、无权威语义，不单立 ADR（记录于本文档与各级 AGENTS.md 端点表）。

## 4. 审查过但本批不修（记录在案）

| 项 | 说明 | 定性 |
|---|---|---|
| `/v2/admin/overview` 双轮询 | SafetyBanner（8s，常驻）+ 当前页各自轮询，Overview 页上 2 并发同端点 | 性能微瑕，将来提共享 context |
| `useApi` 首帧 `loading=false` | useEffect 内才置 true，首帧闪一下空态 | 外观 |
| 无 AbortController | 靠 cancelled/alive 标志丢弃陈旧响应 | 功能正确，请求仍跑完 |
| 非 loopback 无鉴权 | `GARDEN_ADDR` 非 loopback 仅打警告日志，mutation 端点无认证 | **安全待办**——单主机本地假设内可接受；暴露公网前必须加 token/拒绝启动 |
| dist 不入库 | `console/.gitignore` 忽略 `dist/*`（仅 `.gitkeep`），embed 依赖本地构建 | 仓库既有惯例；全新 clone 需先 `npm run build` 才能 `go build` |

## 5. 仍无 UI 的后端面（候选后续批次）

`/v2/memories*`（CRUD）、`/v2/mailbox/*`（收件箱审批）、`/v2/governance/*`（projection/mutations/audit，GovernanceMap 目前只作文本展示）、`/v2/cognitive/world`、`/v2/pipelines/{name}/runs*`、`/v2/admin/context-manifest/{id}`、recall fast/deep 交互面。

## 6. 实机验证记录（2026-08-03，真实二进制 + 本机 EvoMap 凭据）

| 项 | 结果 |
|---|---|
| 13 个 GET 端点巡检 | 全 200，形状正确 |
| `hub/status` | ✅ node_ecdb… 已 claim，100 credits，alive |
| discovery run | ✅ POST 202 → completed（约 10s）→ 出现在 list（0 candidates，quarantine 设计使然） |
| 降级实例（无凭据 + AUTO_REGISTER=0） | ✅ hub/status 与 POST runs 均 503；runs list / admin overview 仍 200 |
| modules PATCH created_at | ⚠️ 发现 bug → 修复 → 复验通过 |
| SPA fallback | ✅ `/` 与 `/evolution` 均返回 index.html |
| 线上 bundle | ✅ 含 Evolution 页面与中英文案 |
| materials evidence | ✅ 空库返回 `{fragments:[],source:"live"}` 200 |
| memories 写路径 | ✅ POST 201 |

单元/集成/e2e：`go test ./internal/...` 全绿（含新增 list 端点 6 个 server 测试、3 个 store 测试、created_at 回归）；`-tags=e2e` 10.9s 绿；`tsc -b` 干净。

**验证方式诚实声明**：本环境无浏览器自动化工具，前端行为经「tsc 类型检查 + bundle 内容断言 + curl 全端点形状验证」兜底，未做真实浏览器点验。

## 7. 文档偏差修正

`garden/AGENTS.md` 曾列 `POST /v2/governance/proposals`——代码从未实现该路由（提案实际在 `/v2/evolution/proposals`），本批已从文档移除。
