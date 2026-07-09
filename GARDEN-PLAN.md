# GARDEN 实施计划 — Phase 0 起手版

> **作者**: 松本(大湿)
> **日期**: 2026-07-09
> **关联 ADR**: `~/Desktop/garden/docs/architecture/0001-garden-merge.md`
> **替代**: ~~GARDEN-PLAN-2026-07-08.md~~(vendor 假设,作废)
> **范围**: Go 路径(laputa + mentle + garden);Rust 路径完全不动
> **关联 README**: `~/Desktop/garden/README.md`(设计哲学)

---

## 0. 5 个决策(已拍,本计划基础)

| # | 决策 | 含义 |
|---|---|---|
| Q1 | **b** | CRUD = write / read / list / forget |
| Q2 | **b** | garden = `~/Desktop/garden/` **全新顶层工作区**(不混合 vendor) |
| Q3 | **b+i** | mentle 仓库边界保留 + **物理搬入 garden 内部**,garden 内相对 import,**不 vendor** |
| Q4 | **b** | laputa 重整成 `governance` 包,laputa.exe 退役 |
| Q-WORKSPACE | **B** | 工作区顶层 `~/Desktop/garden/`,与 `~/Desktop/projects/` 物理隔离 |
| **命名** | — | `mempalace-go-redis-v2` 一律称 **mentle**(目录 + module path),无 `-go` 后缀 |
| **README** | — | 项目名仍是 `laputa`,README 第一行 `# Laputa — Garden Laputa`,哲学放 README |

### 0.1 物理布局（搬入后）

```
~/Desktop/garden/                      ← 工作区根
├── README.md                          ← 哲学 + 架构入口
├── docs/architecture/0001-garden-merge.md  ← ADR
├── GARDEN-PLAN.md                     ← 本文件
├── laputa/                            ← 仓库 1：搬入自 ~/Desktop/projects/laputa/
│   ├── go.mod                         module github.com/dashimaki/laputa
│   ├── governance/                    ← Phase 0 新顶层包
│   ├── cmd/laputa/main.go             ← deprecate 一行注释
│   └── ...
├── mentle/                            ← 仓库 2：搬入自 ~/Desktop/projects/mempalace-go-redis-v2/
│   ├── go.mod                         module github.com/dashimaki/mentle   ← 改名!
│   ├── facade/                        ← Phase 0 新顶层包
│   ├── cmd/server/                    ← 内部 import path 全改
│   └── internal/                      ← 17 internal 保持原状
└── garden/                            ← 仓库 3：Phase 1 起新建
    └── go.mod                         module github.com/dashimaki/garden
```

### 0.2 命名替换清单

| 旧 | 新 |
|---|---|
| `~/Desktop/projects/mempalace-go-redis-v2/` | `~/Desktop/garden/mentle/` |
| `github.com/dashimaki/mempalace-go-redis`（module） | `github.com/dashimaki/mentle` |
| 文档里 `mentle-go` | `mentle` |
| 文档里 `mempalace`（指 Go 版仓库的） | `mentle` |
| `mempalace-py`（Python 版，已在 morediva/.workspace） | **不动** |
| `github.com/dashimaki/laputa`（module） | **不动** |
| Rust 路径（memtle / agent-diva-laputa） | **不动** |

---

## 1. 终极架构(一句话)

> **单 `garden` CLI / HTTP** 通过 garden 内相对 import 引用 **`laputa/governance`** 和 **`mentle/facade`**，对外暴露 **4 个 CRUD**：`write` / `read` / `list` / `forget`。`key` 前缀路由：`section:*` → governance；其它 → mentle facade。

---

## 2. 5 个 Phase 工作流

```
Phase 0 (基础设施)    物理搬运 + mentle module path 改名 + governance/facade 顶层包抽出
       ↓
Phase 1 (CRUD)        garden 仓库骨架 + write/read/list/forget 4 个动作
       ↓
Phase 2 (HTTP)        garden HTTP server + 路由分发
       ↓
Phase 3 (运维)        lifecycle + supervision + 日志
       ↓
Phase 4 (测试)        4 个独立测试入口（governance / facade / garden / 集成）
```

---

## 3. Phase 0 — 基础设施重构 (3-5 天)

### 3.1 目标

1. 把 `~/Desktop/projects/laputa/` 搬进 `~/Desktop/garden/laputa/`
2. 把 `~/Desktop/projects/mempalace-go-redis-v2/` 搬进 `~/Desktop/garden/mentle/`
3. mentle 仓库 module path 从 `mempalace-go-redis` 改 `mentle`
4. laputa 暴露顶层 `governance` 包
5. mentle 暴露顶层 `facade` 包
6. 两边 `go build ./...` + `go test ./...` 全绿
7. 写 `PHASE0-RESULT.md`

### 3.2 操作序列（执行清单）

**步骤 A：物理搬运**

```bash
# 0. 预检：确认 ~/Desktop/projects/laputa 和 ~/Desktop/projects/mempalace-go-redis-v2 都在
ls ~/Desktop/projects/laputa/go.mod
ls ~/Desktop/projects/mempalace-go-redis-v2/go.mod

# 1. 搬 laputa
mv ~/Desktop/projects/laputa ~/Desktop/garden/laputa

# 2. 搬 mentle（同时改路径名）
mv ~/Desktop/projects/mempalace-go-redis-v2 ~/Desktop/garden/mentle

# 3. 验证
ls ~/Desktop/garden/
# 应该看到 laputa/ mentle/ README.md docs/ GARDEN-PLAN.md ...
ls ~/Desktop/projects/
# 应该不再有 laputa/ mempalace-go-redis-v2/
```

**步骤 B：mentle module path 改名**

```bash
cd ~/Desktop/garden/mentle

# 1. 改 go.mod 第一行
sed -i 's|module github.com/dashimaki/mempalace-go-redis|module github.com/dashimaki/mentle|' go.mod

# 2. 全文替换所有 .go 文件里的 import path
grep -rl 'github.com/dashimaki/mempalace-go-redis' --include='*.go' . | xargs sed -i 's|github.com/dashimaki/mempalace-go-redis|github.com/dashimaki/mentle|g'

# 3. 重建 go.sum
rm go.sum
go mod tidy

# 4. 验证编译
go build ./...
```

**步骤 C：laputa governance 重命名**

```bash
cd ~/Desktop/garden/laputa

# 1. 建顶层 governance/ 目录
mkdir governance

# 2. 移 laputa.go → governance/engine.go，并改 package
mv laputa.go governance/engine.go
sed -i 's/^package laputa$/package governance/' governance/engine.go

# 3. 移 internal/* 进 governance/*（保留 package 边界，只改物理位置）
mv internal/rhythm    governance/rhythm
mv internal/scheduler governance/scheduler
mv internal/store     governance/store
mv internal/wakeup    governance/wakeup
mv internal/web       governance/web

# 4. deprecate cmd/laputa（保留作 fallback 二进制）
cat > cmd/laputa/main.go << 'EOF'
// Deprecated: laputa.exe is replaced by garden (laputa + mentle facade).
// Kept as fallback binary for hermes plugin HTTP compatibility on :7373.
package main
EOF

# 5. 验证编译
go build ./...
```

### 3.3 mentle facade 抽出

```bash
cd ~/Desktop/garden/mentle

# 1. 新建顶层 facade/ 目录
mkdir facade

# 2. 写 facade.go（Service 聚合 17 internal）
cat > facade/facade.go << 'EOF'
package facade

import (
    "context"
    "github.com/dashimaki/mentle/internal/config"
    "github.com/dashimaki/mentle/internal/diary"
    "github.com/dashimaki/mentle/internal/kg"
)

type Service struct {
    Cfg   *config.Config
    Diary *diary.Diary
    KG    *kg.KnowledgeGraph
}

func (s *Service) Init(ctx context.Context, opts Options) error {
    // 复制 cmd/server/main.go 里的组装逻辑
    return nil
}

func (s *Service) Close() error { return nil }
EOF

# 3. 写 facade/crud.go write/read/list/forget
cat > facade/crud.go << 'EOF'
package facade

import "context"

func (s *Service) Write(ctx context.Context, key, content string, meta map[string]any) (string, error) {
    return "", nil
}
func (s *Service) Read(ctx context.Context, key string) (map[string]any, error) { return nil, nil }
func (s *Service) List(ctx context.Context, prefix string, limit int) ([]map[string]any, error) {
    return nil, nil
}
func (s *Service) Forget(ctx context.Context, key string) (bool, error) { return false, nil }
EOF

# 4. cmd/server 简化为 facade 入口
cat > cmd/server/main.go << 'EOF'
package main

import (
    "context"
    "github.com/dashimaki/mentle/facade"
)

func main() {
    ctx := context.Background()
    svc := &facade.Service{}
    if err := svc.Init(ctx, facade.Options{}); err != nil { panic(err) }
    defer svc.Close()
    // 原 MCP serve 逻辑后续接上
    select {}
}
EOF

# 5. 验证
go build ./...
go test ./facade/...
```

### 3.4 Phase 0 验收

```bash
# laputa governance
cd ~/Desktop/garden/laputa
go build ./governance/... && echo "laputa gov OK"
go test ./governance/... && echo "laputa gov tests OK"

# mentle facade
cd ~/Desktop/garden/mentle
go build ./... && echo "mentle OK"
go test ./facade/... && echo "mentle facade tests OK"

# hermes plugin HTTP（如果 laputa.exe 还能跑）
curl -s -m 3 http://127.0.0.1:7373/healthz
```

### 3.5 Git 状态

| 仓库 | 提交 |
|---|---|
| `~/Desktop/garden/laputa` | 1-2 commits: 搬入（如果需要） + `refactor: consolidate into governance package` |
| `~/Desktop/garden/mentle` | 2 commits: `refactor: rename module to mentle` + `feat(facade): add facade package` |
| `~/Desktop/garden` | 1 commit: `docs: README + updated GARDEN-PLAN + PHASE0-RESULT` |

### 3.6 已知风险与缓解

| 风险 | 概率 | 缓解 |
|---|---|---|
| mentle 17 internal 互相 import 路径全乱 | 高 | `grep -rl` 一次扫完 + 批量 sed + `go mod tidy` 重生 |
| laputa internal/* 物理位置改导致 import path 断裂 | 中 | 移动前先 grep 确认 import path 是相对路径还是 module path；后者才需要 sed |
| mentle cmd/server 启动跑不起来（chromadb 问题） | 中 | Phase 0 只验 `go build` + `go test ./facade/`，不要求 cmd/server 启动 |
| git 历史跨目录移动丢失 | 中 | 用 `git mv` 而非 `mv`，先在仓库内改名再 mv 到 garden |

---

## 4. Phase 1 — garden 仓库 + CRUD 4 个动作 (3-5 天)

### 4.1 目标

在 `~/Desktop/garden/garden/` 建立全新 Go module,实现 4 个 CRUD 函数调用 `laputa/governance` + `mentle/facade`。

### 4.2 仓库初始化

```bash
cd ~/Desktop/garden
mkdir garden
cd garden

# 1. git init
git init
git config user.name "Matsumoto"
git config user.email "matsumoto@dashimaki.local"

# 2. go.mod（本地相对路径，物理上都在 ~/Desktop/garden/）
cat > go.mod << 'EOF'
module github.com/dashimaki/garden

go 1.26.4

require (
    github.com/dashimaki/laputa v0.0.0
    github.com/dashimaki/mentle  v0.0.0
)

require (
    // 由 go mod tidy 自动填入 indirect deps
)

replace (
    github.com/dashimaki/laputa => ../laputa
    github.com/dashimaki/mentle  => ../mentle
)
EOF

# 3. main.go 占位
cat > main.go << 'EOF'
package main

func main() { println("garden v0.0.1") }
EOF

# 4. 验证
go mod tidy
go build ./...
```

### 4.3 目录布局

```
~/Desktop/garden/garden/
├── go.mod
├── go.sum
├── main.go                          cmd/garden 入口
├── internal/
│   ├── crud/
│   │   └── crud.go                  4 个动作
│   ├── router/
│   │   └── router.go                key 前缀分发
│   ├── server/                      (Phase 2 实现)
│   ├── lifecycle/                   (Phase 3 实现)
│   └── supervision/                 (Phase 3 实现)
├── config/
│   └── config.example.yaml
├── e2e/                             (Phase 4)
└── README.md
```

### 4.4 CRUD 4 个动作（核心代码）

```go
// internal/crud/crud.go
package crud

import (
    "context"
    "github.com/dashimaki/garden/internal/router"
    "github.com/dashimaki/laputa/governance"
    "github.com/dashimaki/mentle/facade"
)

type Handler struct {
    Gov    *governance.Engine
    Facade *facade.Service
    Router *router.Router
}

func (h *Handler) Write(ctx context.Context, key, value string, meta map[string]any) (string, error) {
    backend, err := h.Router.Route(key)
    if err != nil { return "", err }
    return backend.Write(ctx, key, value, meta)
}

func (h *Handler) Read(ctx context.Context, key string) (map[string]any, error) {
    backend, err := h.Router.Route(key)
    if err != nil { return nil, err }
    return backend.Read(ctx, key)
}

func (h *Handler) List(ctx context.Context, prefix string, limit int) ([]map[string]any, error) {
    backend, err := h.Router.Route(prefix)
    if err != nil { return nil, err }
    return backend.List(ctx, prefix, limit)
}

func (h *Handler) Forget(ctx context.Context, key string) (bool, error) {
    backend, err := h.Router.Route(key)
    if err != nil { return false, err }
    return backend.Forget(ctx, key)
}
```

```go
// internal/router/router.go
package router

import (
    "context"
    "errors"
    "strings"
)

type Backend interface {
    Write(ctx context.Context, key, value string, meta map[string]any) (string, error)
    Read(ctx context.Context, key string) (map[string]any, error)
    List(ctx context.Context, prefix string, limit int) ([]map[string]any, error)
    Forget(ctx context.Context, key string) (bool, error)
}

type Router struct {
    Governance Backend
    Mentle     Backend
}

func (r *Router) Route(key string) (Backend, error) {
    if strings.HasPrefix(key, "section:") { return r.Governance, nil }
    if strings.HasPrefix(key, "memory:")  { return r.Mentle, nil }
    return nil, errors.New("unknown key prefix")
}
```

### 4.5 Phase 1 验收

```bash
cd ~/Desktop/garden/garden
go build ./...
go test ./internal/crud/... -v
```

### 4.6 Git 状态

| 仓库 | 提交 |
|---|---|
| `~/Desktop/garden/garden` | 2-3 commits: `init: garden module` + `feat(crud): 4 actions` + `test: crud unit` |

---

## 5. Phase 2 — HTTP server + 路由 (2-3 天)

### 5.1 目标

`garden.exe` 起 HTTP server，4 个 CRUD 暴露为 HTTP 路由。

### 5.2 server.go 骨架

```go
// internal/server/server.go
package server

import (
    "encoding/json"
    "net/http"
    "github.com/dashimaki/garden/internal/crud"
)

type Server struct {
    Handler *crud.Handler
    Addr    string
}

func (s *Server) ListenAndServe() error {
    mux := http.NewServeMux()
    mux.HandleFunc("POST /v1/memories",          s.handleWrite)
    mux.HandleFunc("GET /v1/memories/{key}",     s.handleRead)
    mux.HandleFunc("GET /v1/memories",           s.handleList)
    mux.HandleFunc("DELETE /v1/memories/{key}",  s.handleForget)
    mux.HandleFunc("GET /health",                s.handleHealth)
    return http.ListenAndServe(s.Addr, mux)
}

func (s *Server) handleWrite(w http.ResponseWriter, r *http.Request) {
    var body struct {
        Key   string         `json:"key"`
        Value string         `json:"value"`
        Meta  map[string]any `json:"meta,omitempty"`
    }
    if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
        http.Error(w, err.Error(), 400)
        return
    }
    id, err := s.Handler.Write(r.Context(), body.Key, body.Value, body.Meta)
    if err != nil { http.Error(w, err.Error(), 500); return }
    json.NewEncoder(w).Encode(map[string]string{"id": id})
}
```

### 5.3 main.go 串联

```go
// main.go
package main

import (
    "context"
    "github.com/dashimaki/garden/internal/crud"
    "github.com/dashimaki/garden/internal/lifecycle"
    "github.com/dashimaki/garden/internal/router"
    "github.com/dashimaki/garden/internal/server"
    "github.com/dashimaki/laputa/governance"
    "github.com/dashimaki/mentle/facade"
)

func main() {
    ctx := context.Background()
    gov, err := governance.NewEngine(...).Init(ctx, governance.Options{})
    if err != nil { panic(err) }

    mem := &facade.Service{}
    mem.Init(ctx, facade.Options{})

    h := &crud.Handler{
        Gov:    gov,
        Facade: mem,
        Router: &router.Router{Governance: gov, Mentle: mem},
    }

    srv := &server.Server{Handler: h, Addr: ":7373"}
    lifecycle.Run(ctx, srv)
}
```

### 5.4 Phase 2 验收

```bash
cd ~/Desktop/garden/garden
go build -o garden.exe .
./garden.exe &
sleep 2

curl -s -X POST http://127.0.0.1:7373/v1/memories \
  -H 'Content-Type: application/json' \
  -d '{"key":"section:01-identity","value":"{\"agent\":\"matsumoto\"}"}'
curl -s http://127.0.0.1:7373/v1/memories/section:01-identity
curl -s http://127.0.0.1:7373/v1/memories
curl -s -X DELETE http://127.0.0.1:7373/v1/memories/section:01-identity
curl -s http://127.0.0.1:7373/health
```

### 5.5 Git 状态

| 仓库 | 提交 |
|---|---|
| `~/Desktop/garden/garden` | `feat(server): HTTP server with 4 CRUD routes` |

---

## 6. Phase 3 — Lifecycle + Supervision (2-3 天)

### 6.1 目标

启停顺序、优雅关闭、crash 重试、健康检查。

### 6.2 lifecycle.go

```go
// internal/lifecycle/lifecycle.go
package lifecycle

import (
    "context"
    "os"
    "os/signal"
    "syscall"
    "github.com/dashimaki/garden/internal/server"
    "github.com/dashimaki/garden/internal/supervision"
)

func Run(ctx context.Context, srv *server.Server) {
    ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
    defer cancel()

    sup := supervision.New(srv)
    sup.Start(ctx)

    <-ctx.Done()
    sup.Stop(ctx)
}
```

### 6.3 supervision.go

```go
// internal/supervision/supervision.go
package supervision

import (
    "context"
    "time"
)

type Supervisor struct {
    srv         interface{ Shutdown(context.Context) error }
    healthTick  time.Ticker
    crashCount  int
}

func New(srv interface{ Shutdown(context.Context) error }) *Supervisor {
    return &Supervisor{srv: srv}
}

func (s *Supervisor) Start(ctx context.Context) {
    go s.healthCheckLoop(ctx)
}

func (s *Supervisor) Stop(ctx context.Context) {
    _ = s.srv.Shutdown(ctx)
}

func (s *Supervisor) healthCheckLoop(ctx context.Context) {
    s.healthTick = *time.NewTicker(10 * time.Second)
    defer s.healthTick.Stop()
    for {
        select {
        case <-ctx.Done(): return
        case <-s.healthTick.C:
            // 失败累计 3 次 → 全停
        }
    }
}
```

### 6.4 Phase 3 验收

```bash
cd ~/Desktop/garden/garden
./garden.exe &
PID=$!
sleep 3
kill -TERM $PID
sleep 2
ps -p $PID || echo "garden exited cleanly"
cat ~/.garden/garden.log
```

### 6.5 Git 状态

| 仓库 | 提交 |
|---|---|
| `~/Desktop/garden/garden` | `feat(lifecycle): supervision + graceful shutdown` |

---

## 7. Phase 4 — 4 个独立测试入口 (2-3 天)

### 7.1 目标

**用户原话:"分开写测试脚本做测试"**。4 个独立 `go test` 入口,不混在一起。

### 7.2 4 个 test 入口

| 入口 | 路径 | 范围 | 命令 |
|---|---|---|---|
| `governance_test` | `~/Desktop/garden/laputa/governance/` | laputa governance 单测 | `cd laputa && go test ./governance/...` |
| `facade_test` | `~/Desktop/garden/mentle/facade/` | mentle facade 单测 | `cd mentle && go test ./facade/...` |
| `garden_unit_test` | `~/Desktop/garden/garden/internal/` | garden 4 CRUD 单测 | `cd garden && go test ./internal/...` |
| `garden_e2e_test` | `~/Desktop/garden/garden/e2e/` | 起 garden,真 HTTP,查结果 | `cd garden && go test -tags=e2e ./e2e/...` |

### 7.3 e2e test 写法

```go
// e2e/e2e_test.go
//go:build e2e

package e2e

import (
    "os/exec"
    "testing"
    "time"
)

func TestGardenEndToEnd(t *testing.T) {
    cmd := exec.Command("./garden.exe")
    cmd.Start()
    defer cmd.Process.Kill()

    time.Sleep(3 * time.Second)

    // POST /v1/memories
    // GET  /v1/memories/key
    // GET  /v1/memories
    // DELETE /v1/memories/key
    // GET  /health
}
```

### 7.4 Phase 4 验收

```bash
cd ~/Desktop/garden/laputa && go test ./governance/...
cd ~/Desktop/garden/mentle && go test ./facade/...
cd ~/Desktop/garden/garden && go test ./internal/...
cd ~/Desktop/garden/garden && go test -tags=e2e ./e2e/...
```

### 7.5 Git 状态

| 仓库 | 提交 |
|---|---|
| `~/Desktop/garden/laputa` | `test(governance): unit tests` |
| `~/Desktop/garden/mentle` | `test(facade): unit tests for 4 CRUD` |
| `~/Desktop/garden/garden` | `test(garden): unit + e2e` |

---

## 8. 工作量估算

```
Phase 0  基础设施重构         3-5 天
Phase 1  garden + CRUD        3-5 天
Phase 2  HTTP server          2-3 天
Phase 3  lifecycle + super    2-3 天
Phase 4  4 个独立测试         2-3 天
──────────────────────────────────
总计                          12-19 天 (3-4 周)
```

---

## 9. 不做清单

| 项 | 不做原因 |
|---|---|
| 重写 laputa 业务代码 | governance 重命名已足够 |
| 重写 mentle 17 internal | 抽 facade 已足够 |
| 5 facade 业务方法（7/6 doc） | 4 CRUD 取而代之 |
| Pipeline / WorkflowStep / Interceptor | 7/6 doc Layer 2，Phase 1 之后看是否再引入 |
| LLM profile routing | 7/6 doc Layer 3 |
| Memory file export | 7/6 doc Layer 5 |
| MCP server（garden） | facade 已收敛 43 tool |
| Postgres / 多 agent / 多模态 | 没必要 |
| **Rust 路径任何变更** | 用户明示 "go vs rust 隔离" |
| 修改 laputa module path | "叫 laputa" 不变 |
| mempalace-py 改动 | 已搬到 morediva/.workspace，物理隔离 |

---

## 10. 风险与缓解

| 风险 | 概率 | 缓解 |
|---|---|---|
| mentle 启动问题（7/8 实测）在 Phase 0 末段爆发 | 中 | Phase 0 只验 `go build` + `go test ./facade/`，不要求 cmd/server 启动 |
| governance 重命名 breaking hermes plugin | 低 | hermes 只调 :7373 HTTP，cmd/laputa 保留 fallback |
| facade 整合破坏 cmd/server MCP 兼容 | 中 | Phase 0 跑 MCP smoke（可选） |
| 跨 git 仓库开发体验差 | 低 | 物理搬入后都用 garden 顶层，相对路径 |
| 命名混乱（mempalace vs mentle） | 低 | 文档统一用 mentle，本计划已替换完成 |

---

## 11. 关键命令速查

```bash
# Phase 0 完
cd ~/Desktop/garden/laputa && go test ./governance/...
cd ~/Desktop/garden/mentle && go test ./facade/...

# Phase 1 完
cd ~/Desktop/garden/garden && go build ./...

# Phase 2 完
cd ~/Desktop/garden/garden && ./garden.exe &
sleep 2
curl -s http://127.0.0.1:7373/health

# Phase 3 完
kill -TERM $(pgrep garden) && cat ~/.garden/garden.log

# Phase 4 完 — 4 个独立 test
cd ~/Desktop/garden/laputa && go test ./governance/...
cd ~/Desktop/garden/mentle && go test ./facade/...
cd ~/Desktop/garden/garden && go test ./internal/...
cd ~/Desktop/garden/garden && go test -tags=e2e ./e2e/...
```

---

## 12. 时间线

| 日期 | 里程碑 |
|---|---|
| 2026-07-09（今天） | README + 本计划更新完，派 cursor 开 Phase 0 |
| 2026-07-10 ~ 07-13 | Phase 0 |
| 2026-07-14 ~ 07-18 | Phase 1 |
| 2026-07-19 ~ 07-22 | Phase 2 |
| 2026-07-23 ~ 07-26 | Phase 3 |
| 2026-07-27 ~ 07-30 | Phase 4 |

---

## 13. 待拍（写完本计划后的开放问题）

| # | 问题 | 何时拍 |
|---|---|---|
| 1 | mentle module path 改名是否要在 git rename PR 里做（保持历史） | Phase 0 末段 |
| 2 | garden HTTP 端口（7373 / 7374） | Phase 2 |
| 3 | supervision 默认（1 次停 vs 3 次重试） | Phase 3 |
| 4 | mentle 启动问题修复时间窗 | Phase 0 末段 |

---

**计划完成**: 2026-07-09 16:00
**本计划文件**: `C:\Users\Administrator\Desktop\garden\GARDEN-PLAN.md`
**配套 ADR**: `C:\Users\Administrator\Desktop\garden\docs\architecture\0001-garden-merge.md`
**配套 README**: `C:\Users\Administrator\Desktop\garden\README.md`
**下一步**: 派 cursor 开 Phase 0