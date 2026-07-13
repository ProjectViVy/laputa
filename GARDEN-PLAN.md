# GARDEN 实施计划 — v3-final (状态冻结版)

> **作者**: 松本(大湿)
> **日期**: 2026-07-14
> **状态**: Phase 0/1/2/3 已完成,Phase 4 待开
> **替代**: ~~GARDEN-PLAN-2026-07-08.md~~(已移至 `docs/archive/`)
> **关联 ADR**: `docs/architecture/0001-garden-merge.md`
> **关联 README**: `~/Desktop/garden/README.md`(设计哲学)
> **范围**: Go 路径(laputa + mentle + garden);Rust 路径完全不动

---

## 0. 一句话架构(终极,不再变)

```
单 garden.exe HTTP :7373
  → CRUD 4 个动作 (write / read / list / forget)
    → key 前缀路由:
       section:*  → laputa/governance   (14 section, 纯文件)
       memory:*   → mentle/facade       (redis + sqlite + govector + sqlite-vec)
```

工作区布局:

```
~/Desktop/garden/                              ← 唯一工作区,跟 ~/Desktop/projects/ 物理隔离
├── README.md                                  ← 设计哲学(土壤/天空/种植)
├── GARDEN-PLAN.md                             ← 本文件(状态冻结版)
├── docs/
│   ├── architecture/0001-garden-merge.md      ← ADR
│   └── archive/                               ← 历史计划
│       └── GARDEN-PLAN-2026-07-08.md          ← v1 已废弃
├── PHASE0-RESULT.md                           ← 3.7 KB
├── PHASE1-RESULT.md                           ← 3.8 KB
├── PHASE2-RESULT.md                           ← 4.3 KB
├── PHASE3-RESULT.md                           ← 4.6 KB
├── laputa/                                    ← 仓库 1:治理层(天空)
│   ├── go.mod            module github.com/dashimaki/laputa
│   ├── governance/       ← 顶层包
│   └── cmd/laputa/       ← deprecated fallback (保 :7373 兼容)
├── mentle/                                    ← 仓库 2:记忆层(土壤,V2 多存储版)
│   ├── go.mod            module github.com/dashimaki/mentle
│   ├── facade/           ← 顶层包(4 CRUD)
│   └── internal/         ← 17 internal 保持原状
└── garden/                                    ← 仓库 3:种植层(CLI/HTTP)
    ├── go.mod            module github.com/dashimaki/garden
    ├── main.go
    └── internal/{crud, router, server, lifecycle, supervision}
```

**模块名 / 仓库名固定不变**:
- `github.com/dashimaki/laputa` (V1 治理)
- `github.com/dashimaki/mentle` (V2 多存储兼容)
- `github.com/dashimaki/garden` (应用层)

---

## 1. 当前完成状态(冻结快照,2026-07-14)

### 1.1 5 个 Phase 完成度

| Phase | 内容 | 状态 | 交付物 | commit |
|---|---|---|---|---|
| **0** | 物理搬入 + mentle 改名 + 抽 governance/facade 顶层包 | ✅ 完成 | `PHASE0-RESULT.md` | `7e16be3` |
| **1** | garden 仓库骨架 + 4 CRUD + key 前缀 router | ✅ 完成 | `PHASE1-RESULT.md` | `48cc0fc` |
| **2** | HTTP server + 4 CRUD 路由 + /health + graceful degradation | ✅ 完成 | `PHASE2-RESULT.md` | `3537c4c` |
| **3** | lifecycle + supervision + signal + 日志双写 | ✅ 完成 | `PHASE3-RESULT.md` | `673c27c` |
| **4** | 4 个独立测试入口(含 e2e build tag) | 🟡 待开 | 暂无 | — |

**主体进度: 80% (4/5 phase)。**

### 1.2 Phase 0 完成事实(2026-07-09)

| 改动 | 路径 |
|---|---|
| `~/Desktop/projects/laputa/` → `~/Desktop/garden/laputa/` | 物理 mv |
| `~/Desktop/projects/mempalace-go-redis/` → `~/Desktop/garden/mentle/` | 物理 mv + 改名 |
| `github.com/dashimaki/mempalace-go-redis` → `github.com/dashimaki/mentle` | go.mod 改名 |
| `laputa.go` → `governance/engine.go` | package = `governance` |
| `internal/{rhythm,scheduler,store,wakeup,web}` → `governance/*` | 5 sub-package |
| 新建 `mentle/facade/{facade.go,crud.go,facade_test.go}` | 顶层包 |

**验收**:
```bash
cd ~/Desktop/garden/laputa && go build ./governance/... && go test ./governance/...
cd ~/Desktop/garden/mentle && go build ./... && go test ./facade/...
```

### 1.3 Phase 1 完成事实(2026-07-09)

| 包 | 路径 | 内容 |
|---|---|---|
| `internal/crud` | `garden/internal/crud/crud.go` | `Handler.Write/Read/List/Forget` |
| `internal/router` | `garden/internal/router/{router,governance,mentle_adapter}.go` | key 前缀分发 + 2 adapter |
| `internal/router` | `garden/internal/router/router_test.go` | 路由表 |

**路由规则** (ADR §3.2):
- `section:*` → governance backend
- `memory:*` → mentle facade
- 其他 → error

**go.mod replace**:
```go
replace (
    github.com/dashimaki/laputa => ../laputa
    github.com/dashimaki/mentle  => ../mentle
)
```

### 1.4 Phase 2 完成事实(2026-07-09)

**HTTP 端点** (5 个):

| Method | Route | Handler |
|---|---|---|
| POST | `/v1/memories` | `crud.Handler.Write` |
| GET | `/v1/memories/{key}` | `crud.Handler.Read` |
| GET | `/v1/memories` | `crud.Handler.List` |
| DELETE | `/v1/memories/{key}` | `crud.Handler.Forget` |
| GET | `/health` | 静态 OK + timestamp |

**Graceful degradation**: mentle 启动失败 → 回退到 governance-only 模式 (section: keys 继续工作,memory: keys 返回 error)。

**环境变量**:
| Variable | Default |
|---|---|
| `GARDEN_ADDR` | `:7373` |
| `GARDEN_GOVERNANCE_DIR` | `~/.laputa/sections` |

**main.go wiring**: `governance.Engine` + `facade.Service` + `crud.Handler` + `server.Server`。

### 1.5 Phase 3 完成事实(2026-07-09)

**架构改进** (超出 plan): `lifecycle` 内部封装 `supervision`,`main.go` 只依赖 `lifecycle`。

**main.go 一行启动**:
```go
lifecycle.Run(ctx, srv)
```

**默认策略** (解决 plan §13 待拍 #3):

| 策略 | 默认值 | 来源 |
|---|---|---|
| Health 检查间隔 | 10s | `supervision.New` |
| Health 失败后停止阈值 | 3 | `supervision.HealthCheckFailLimit` |
| Server crash 重启延迟 | 5s | `supervision.CrashRestartDelay` |
| 最大重试次数 | 3 | `supervision.MaxCrashRestarts` |
| 优雅关闭超时 | 30s | `lifecycle.defaultShutdownTimeout` |

**环境变量新增**:
| Variable | Default |
|---|---|
| `GARDEN_LOG_DIR` | `~/.garden` |

**日志双写**: `~/.garden/garden.log` + stderr。

### 1.6 累计测试覆盖(Phase 1+2+3)

| 包 | 测试 |
|---|---|
| `laputa/governance/...` | Phase 0 已 work |
| `mentle/facade/...` | Phase 0 已 work |
| `garden/internal/crud` | ✅ |
| `garden/internal/router` | ✅ |
| `garden/internal/server` | ✅ 6 handler tests |
| `garden/internal/lifecycle` | ✅ |
| `garden/internal/supervision` | ✅ |

**总数**: 至少 17 个 test 通过 (Phase 3 dispatcher 报告)。

---

## 2. Phase 4 — 待开 (待派,我后面自己开)

### 2.1 目标

完成 4 个**独立** `go test` 入口(用户原话:"分开写测试脚本做测试")。

### 2.2 4 个 test 入口

| 入口 | 路径 | 命令 |
|---|---|---|
| `governance_test` | `laputa/governance/` | `cd laputa && go test ./governance/...` |
| `facade_test` | `mentle/facade/` | `cd mentle && go test ./facade/...` |
| `garden_unit_test` | `garden/internal/` | `cd garden && go test ./internal/...` |
| `garden_e2e_test` | `garden/e2e/` | `cd garden && go test -tags=e2e ./e2e/...` |

**前 3 个已 work** (Phase 3 验证),仅 e2e 待建。

### 2.3 e2e 实现要点

```go
// garden/e2e/e2e_test.go
//go:build e2e

package e2e

func TestGardenEndToEnd(t *testing.T) {
    // 1. go build garden.exe  到 t.TempDir()
    // 2. net.Listen 找空闲端口
    // 3. exec.Command 启动 subprocess
    //    env: GARDEN_ADDR, GARDEN_GOVERNANCE_DIR, GARDEN_LOG_DIR 指向 t.TempDir()
    // 4. poll GET /health 直到 200 或 timeout
    // 5. POST /v1/memories {key: "section:01-identity", value: "..."}
    // 6. GET /v1/memories/{key} 验证 value
    // 7. defer process.Kill()
}
```

**build tag 必须**: `//go:build e2e` 在文件顶。
**不能用 mock**: e2e 必须起 garden 真 HTTP 进程,跑全栈。

### 2.4 验收

```bash
# 1. 普通跑不应触发 e2e
cd ~/Desktop/garden/garden
GOSUMDB=off go test ./...

# 2. 带 tag 跑 e2e
GOSUMDB=off go test -tags=e2e ./e2e/...
```

### 2.5 预计交付

- `garden/e2e/e2e_test.go` (1 个 file,~80 行)
- `PHASE4-RESULT.md` (仿 0-3 模板,3-5 KB)
- 仓库 commit:`test(e2e): end-to-end test entry`

**预估时间**: 1-2 小时 (cursor 7/09 之前实测 ~133 秒 / phase, e2e 更简单)。

---

## 3. 移交清单 / 接手要点

### 3.1 你需要知道的真身歧义

**之前 compaction 里的错误认知已校正:**

| 错误认知 | 真相 |
|---|---|
| "Go laputa 不存在" | **存在**,7/05 initial commit 起就在 `~/Desktop/projects/laputa/`(现在 `~/Desktop/garden/laputa/`),跟 argylelabcoat-mempalace-go 是**两个独立项目** |
| "argylelabcoat 是 laputa-go 真身" | **错**。argylelabcoat-mempalace-go 是 mempalace-flavored fork,dashimaki/laputa 是独立 governance 框架 |
| "`mempalace-go-redis-v2` 是真名" | **错**。真名是 `mempalace-go-redis`(无 -v2) |
| "mentle-go 是项目名" | **错**。本会话期间口头用过,真名是 `mentle`(无 -go) |
| "mentle" = 只有 governance | **错**。laputa=governance,**mentle=mempalace V2 多存储兼容版**(redis + sqlite + govector + sqlite-vec) |

**真身一对**:

| 项目名 | 物理位置 | module | 角色 |
|---|---|---|---|
| **laputa** (V1) | `~/Desktop/garden/laputa/` | `github.com/dashimaki/laputa` | 治理框架 (14 section, 纯文件) |
| **mentle** (V2) | `~/Desktop/garden/mentle/` | `github.com/dashimaki/mentle` | mempalace V2 (redis + sqlite 多 backend) |

### 3.2 已拍板的 5 个决策 (不再反复)

| # | 决策 | 含义 |
|---|---|---|
| Q1 | **b** | CRUD = write / read / list / forget |
| Q2 | **b** | garden = `~/Desktop/garden/` 全新顶层工作区 |
| Q3 | **b+i** | mentle 仓库边界保留 + 物理搬入 garden 内部,garden 内相对 import,**不 vendor** |
| Q4 | **b** | laputa 重整成 `governance` 包,laputa.exe 退役 |
| Q-WORKSPACE | **B** | 工作区顶层 `~/Desktop/garden/`,与 `~/Desktop/projects/` 物理隔离 |

### 3.3 已处理的 4 个开放问题

| # | plan §13 待拍 | 处理结果 |
|---|---|---|
| 1 | mentle module 是否改名 | ✅ 已改 (`mempalace-go-redis` → `mentle`) |
| 2 | garden HTTP 端口 | ✅ `:7373` (继承 laputa) + `GARDEN_ADDR` env 覆盖 |
| 3 | supervision 1 次停 vs 3 次重试 | ✅ 3 次重试后 exit (Phase 3 默认值) |
| 4 | mentle 启动问题修复时间 | ⚠️ 未修复,通过 facade + graceful degradation 隔离,启动失败 garden 回退 governance-only |

### 3.4 不做清单(继承 ADR §3.5)

| 项 | 不做原因 |
|---|---|
| 重写 laputa 业务代码 | governance 重命名已足够 |
| 重写 mentle 17 internal | 抽 facade 已足够 |
| 5 facade 业务方法(7/6 doc) | 4 CRUD 取而代之 |
| Pipeline / WorkflowStep / Interceptor | 7/6 doc Layer 2,Phase 1 后看是否引入 |
| LLM profile routing | 7/6 doc Layer 3 |
| Memory file export | 7/6 doc Layer 5 |
| MCP server (garden) | facade 已收敛 |
| Postgres / 多 agent / 多模态 | 没必要 |
| **Rust 路径任何变更** | 用户明示 "go vs rust 隔离" |
| 修改 laputa module path | "叫 laputa" 不变 |
| mempalace-py 改动 | 已搬到 `morediva/.workspace`,物理隔离 |

### 3.5 接手人守则

1. **go.mod replace 不要改**:
   ```go
   replace (
       github.com/dashimaki/laputa => ../laputa
       github.com/dashimaki/mentle  => ../mentle
   )
   ```
   garden 不动这俩 replace 就能编。

2. **GOSUMDB=off** 是必需的 (本机 Go sumdb 不可达,环境变量加在命令前)。

3. **mentle 启动问题** 不要在 Phase 4 解决,graceful degradation 已隔离,Phase 4 测试 governance-only mode 就能 pass。

4. **Hermes 禁区**: `/c/Users/Administrator/.hermes/`, `~/.claude/` 不可触碰。

5. **不接 proxy**: 7892 FlClashHelperService 在跑,但环境变量不 persist,也不需要。

6. **不接 codex/claude-code/subagent**: 用户偏好 (memory 持久化)。

7. **rust 隔离**: `laputa-work/`、`morediva/`、`olv-rs/`、`new-mentle/memtle` 完全不动。

---

## 4. 关键命令速查

```bash
# === 工作区根 ===
cd ~/Desktop/garden

# === Phase 0 验证 ===
cd laputa && go build ./governance/... && go test ./governance/...
cd ../mentle && go build ./... && go test ./facade/...

# === Phase 1/2/3 验证 ===
cd garden
GOSUMDB=off go build ./...
GOSUMDB=off go build -o garden.exe .
GOSUMDB=off go test ./internal/...

# === 运行 garden ===
./garden.exe &
sleep 2
curl -s http://127.0.0.1:7373/health
curl -s -X POST http://127.0.0.1:7373/v1/memories \
  -H 'Content-Type: application/json' \
  -d '{"key":"section:01-identity","value":"hello garden"}'
curl -s http://127.0.0.1:7373/v1/memories/section:01-identity

# === Phase 3 验证 (signal) ===
PID=$(pgrep garden)
kill -TERM $PID
sleep 2
ps -p $PID || echo "garden exited cleanly"
cat ~/.garden/garden.log

# === Phase 4 (待开) ===
GOSUMDB=off go test ./...                    # 不应触发 e2e
GOSUMDB=off go test -tags=e2e ./e2e/...      # 跑 e2e
```

---

## 5. git 状态

```text
garden (顶层)
├── 673c27c docs: Phase 3 complete — lifecycle + supervision
├── 3537c4c docs: Phase 2 complete — HTTP server with 4 CRUD routes
├── 48cc0fc docs: Phase 1 complete — garden module + 4 CRUD + router
└── 7e16be3 docs: Garden Laputa Phase 0 complete — initialize workspace

garden/garden (子模块)
├── c1cf3d9 feat(lifecycle): signal handling + supervision + graceful shutdown
├── 0b34692 feat(server): HTTP server with 4 CRUD routes + health
└── 67e154b feat: scaffold garden module + 4 CRUD + router

garden/laputa + garden/mentle (Phase 0 各自仓库)
```

---

## 6. 下一步

1. **派自己(用户)** 开 Phase 4 e2e (用户 2026-07-14 已确认 "我待会自己派")
2. e2e 完成后写 `PHASE4-RESULT.md`
3. garden 整体进入"5/5 phase 完成"状态

---

**计划完成**: 2026-07-14 (Phase 0/1/2/3 真实已 work,Phase 4 待开)
**本计划文件**: `C:\Users\Administrator\Desktop\garden\GARDEN-PLAN.md`
**配套 ADR**: `C:\Users\Administrator\Desktop\garden\docs\architecture\0001-garden-merge.md`
**配套 README**: `C:\Users\Administrator\Desktop\garden\README.md`
**历史归档**: `C:\Users\Administrator\Desktop\garden\docs\archive\GARDEN-PLAN-2026-07-08.md`
