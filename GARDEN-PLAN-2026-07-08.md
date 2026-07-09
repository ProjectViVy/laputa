# GARDEN Plan — 2026-07-08 定稿（已废弃）

> **⚠️ DEPRECATED 2026-07-09 ⚠️**
>
> 本计划已被 [`GARDEN-PLAN.md`](./GARDEN-PLAN.md)（2026-07-09 更新版）**完全取代**。
>
> **取代原因**：
> 1. 物理布局升级：laputa 和 mempalace-go-redis-v2 整库搬入 `~/Desktop/garden/` 内部（不再跨仓库引用）
> 2. 命名变更：`mempalace-go-redis-v2` → `mentle`（去 `-go` 后缀）
> 3. 项目名仍是 `laputa`，README 副标题 = "Garden Laputa"
> 4. 设计哲学（土壤/天空/种植）写入 README.md
>
> 本文档保留作为**历史快照**，便于追溯决策演变。**不要按本计划执行**。

---

> **作者**: 松本(大湿)
> **日期**: 2026-07-08
> **状态**: 4 个 Q 全部 = b,定稿
> **范围**: Go 路径(laputa + mempalace-go-redis-v2,Rust 不动)
> **基础**: 7/6 `NEW-LAPUTA.md` + 7/8 用户 4 轮细化

---

## 0. 4 个决策（已拍,不再讨论）

| # | 决策 | 含义 |
|---|---|---|
| **Q1** | CRUD = **write / read / list / forget** | memory 风格,不是数据库/治理风格 |
| **Q2** | garden 放 **`~/Desktop/projects/laputa/garden/`** | laputa 仓库顶层子目录,独立 Go package |
| **Q3** | mempalace **保留 mempalace-go-redis-v2 仓库**,17 internal 整合成 1 facade 包 | 中创:仓库边界保留,内部重构 |
| **Q4** | laputa **整成 governance 包**,laputa.exe 退役,只留 garden.exe | 中创:二进制合一 |

---

## 1. 跟 7/6 `NEW-LAPUTA.md` 拍板的差异

| 7/6 拍板 | 7/8 拍板 | 影响 |
|---|---|---|
| garden = 新独立仓库 `~/Desktop/projects/garden/` | garden 在 laputa 仓库内顶层子目录 | 仓库 3 个 → 形式上 2 个(garden=laputa 子目录不增仓库) |
| Phase 0 先库化 laputa | laputa.go + 5 internal 直接整合成 governance 包 | 跳过"包化"步骤,直接重整 |
| mempalace 抽 Assemble 函数 | mempalace 17 internal 整合成 1 facade 包 | 目标更完整:不只抽 facade,内部整合 |
| 5 个 facade 业务方法(7-6 §3.3) | **CRUD 4 个:write/read/list/forget** | 范围更窄,4 个不是 5 个,且是粗粒度 |
| 7/6 doc §3.1 "5 层架构"(facade+ pipeline+ 治理+ backend+ file export) | **单 exe + 单 governance 包 + 单 memory facade**,无 5 层 | 砍掉 7/6 doc 设计的 80% |
| Phase 4 集成测试 | "分开写测试脚本做测试" | 每个层/包独立测试,不合并 |

**7/6 doc 哪些留下来:**

- 数据库/Database Protocol 协议思想 ✓ 进 governance 包设计
- "不复造 backend,只 import 现成" 原则 ✓
- "不起 MCP server"(§3.5.2) ✓ 进 garden server 设计
- "文件 export"(§3.5.2 不做清单) ✗ 本次也不做(优先级低)
- "LLM profile routing"(§3.5.2 不做清单) ✗ 本次也不做
- "pipeline + Interceptor"(§3.5.1 ⏸️) ✗ 本次也不做

---

## 2. 终态架构（一图）

```
┌─────────────────────────────────────────────────┐
│         garden.exe (单二进制,唯一入口)          │
│         ~/Desktop/projects/laputa/garden/main.go│
└─────────────────────┬───────────────────────────┘
                      │
      ┌───────────────┼───────────────┐
      ▼               ▼               ▼
┌──────────┐   ┌──────────────┐  ┌──────────────┐
│ CRUD 4 个│   │ garden server│  │  治理编排    │
│ write/   │   │ (HTTP/stdiomcp│  │ (统一事务    │
│ read/    │   │  二选一,待定) │  │  step chain) │
│ list/    │   │              │  │              │
│ forget   │   │              │  │              │
└──────────┘   └──────────────┘  └──────────────┘
      │               │               │
      └───────────────┼───────────────┘
                      ▼
┌─────────────────────────────────────────────────┐
│       laputa 仓库内 (~/Desktop/projects/laputa/) │
│  ┌──────────────────────────────────────────┐   │
│  │ governance/  (重整 lap uta 职能,Q4=b)    │   │
│  │  ├── engine.go       (lap uta.go 重整)   │   │
│  │  ├── sections.go     (14 section CRUD)  │   │
│  │  ├── rhythm.go       (从 internal/rhythm)│  │
│  │  ├── scheduler.go    (从 internal/scheduler)│ │
│  │  ├── store.go        (从 internal/store)  │ │
│  │  ├── wakeup.go       (从 internal/wakeup) │ │
│  │  └── web/            (从 internal/web)   │   │
│  │  └─ 旧 package laputa 退役                │   │
│  └──────────────────────────────────────────┘   │
│  ┌──────────────────────────────────────────┐   │
│  │ garden/        (Q2=b,顶层子目录)         │   │
│  │  ├── main.go    (单 exe 入口)            │   │
│  │  ├── crud.go    (write/read/list/forget) │   │
│  │  ├── server.go  (HTTP 或 stdio 协议)     │   │
│  │  ├── lifecycle.go (启动/关闭/重启)       │   │
│  │  └── supervision.go (crash 重试)        │   │
│  └──────────────────────────────────────────┘   │
│  ┌──────────────────────────────────────────┐   │
│  │ mempalace/       (Q3=b,本地 vendored 子模块,│   │
│  │                  来自 mempalace-go-redis-v2)│ │
│  │  └── facade/      (公开 1 个 facade 包)  │   │
│  │       ├── facade.go  (单一入口,组合 17   │   │
│  │       │               internal)          │   │
│  │       ├── diary.go    (从 internal/diary)│   │
│  │       ├── kg.go       (从 internal/kg)   │   │
│  │       ├── search.go   (从 internal/search)│  │
│  │       ├── hybrid.go   (从 internal/hybrid)│  │
│  │       ├── embedder.go (从 internal/embedder)│ │
│  │       └── ... (其他 internal 整合)       │   │
│  └──────────────────────────────────────────┘   │
│  cmd/laputa/         → 退役(被 garden/ 替代)   │
│  internal/           → 退役(被 governance/ 替代)│
└─────────────────────────────────────────────────┘
```

**注意(已定,i)**:Q3=b + garden 单 exe build = **vendor mempalace 进 laputa 工作区作为子目录**。具体做法:`mempalace-go-redis-v2` 仓库的源码作为 `~/Desktop/projects/laputa/mempalace/` 子目录(Git subtree 或软链,详见 Phase 2.0),build 时单一 Go module(只用 `~/Desktop/projects/laputa/go.mod`)。

---

## 3. CRUD 4 个 API 的"粗粒度"

基于 Q1=b:

| API | 输入 | 输出 | 背后操作 |
|---|---|---|---|
| **write** | `(key, value, metadata?)` | `record_id` | 写一处:laputa section 或 mempalace drawer (按 key 路由) |
| **read** | `(key)` | `value + metadata` | 读一处(同上规则) |
| **list** | `(filter?)` | `[]record` | 列所有匹配的(过滤由 key 前缀/metadata 完成) |
| **forget** | `(key)` | `bool` | 删一处(同上规则) |

**路由规则**(governance/garden 内部):
- `key` 以 `section:` 开头 → laputa governance section
- `key` 以 `memory:` 开头 → mempalace drawer
- 其它 → 报错(防止混乱)

**为什么 user 要"几个简单 API"**:这套 API 是暴露给 hermes agent 用的粗粒度入口,**不需要让上层知道 laputa vs mempalace**。上层只用 4 个动词搞定一切。

> ⚠️ **绝不混**:diva 是 Rust 路径(`agent-diva` + `agent-diva-laputa` + `memtle`),跟本计划无关。diva 调 hermes 的记忆面走它自己的 `mentle` crate,不走 garden。本计划只在 **Go 路径**内做。

---

## 4. 4 个 Phase（按依赖排序）

### Phase 1: governance 包整理 (3-5 天)

**目标**: laputa 重整为单一 governance 包,laputa.exe 退役

#### 1.1 目录调整

```
~/Desktop/projects/laputa/
├── laputa.go               → governance/engine.go       (重命名, 内部 package governance)
├── laputa_test.go          → governance/engine_test.go
├── internal/rhythm/        → governance/rhythm/
├── internal/scheduler/     → governance/scheduler/
├── internal/store/redis/   → governance/store/
├── internal/wakeup/        → governance/wakeup/
├── internal/web/           → governance/web/
├── cmd/laputa/             → 退役(被 garden 替代)
└── governance/             ← 新顶层 package
    ├── engine.go           (原 laputa.go,改 package governance)
    ├── sections.go         (14 section schema 定义,从 laputa.go 抽出)
    ├── rhythm.go           (从 internal/rhythm/)
    ├── scheduler.go
    ├── store.go            (从 internal/store/redis/)
    ├── wakeup.go
    └── web/                (从 internal/web/)
```

#### 1.2 关键约束(不改的)

- `package governance` 必须保留**原始 `package laputa` 的所有 exported 符号**
- `NewEngine()` / `Engine` / 14 section 常量 / wakeup 协议 — 这些是 hermes plugin 已经依赖的 public API
- `NewEngine` 现在签名是 `(store SectionStore)` — 保留,**不**因 "Q4=b" 改成别的(防止 breaking hermes plugin)

#### 1.3 验证

```bash
cd ~/Desktop/projects/laputa
go build ./governance/...       # 新 package 能编译
go test ./governance/...        # laputa_test.go 测试通过(改包名后)
go build ./cmd/laputa/...        # 老 laputa.exe 仍能编译(预备退役但先能用)
```

#### 1.4 git

- 1 个 commit: `refactor(laputa): consolidate into governance package, deprecated cmd/laputa`

---

### Phase 2: mempalace facade 整合 + 进入 laputa 工作区 (4-7 天)

**目标**: mempalace-go-redis-v2 仓库内 17 internal 整合成 1 facade 包, **vendor 进 laputa 工作区**

#### 2.0 mempalace 进 laputa 工作区 (0.5-1 天) — **Q3=i 决定**

```
~/Desktop/projects/laputa/
├── mempalace/                  ← Git subtree / 软链 (从 mempalace-go-redis-v2)
│   ├── facade/
│   ├── internal/               (17 个内部包)
│   ├── cmd/                    (mempalace.exe 还会在,但 laputa 工作区内的)
│   └── go.mod                  (子目录的 go.mod, 用于独立调试 facade)
├── governance/
├── garden/
└── go.mod                      (主 build 用, replace 指向 ./mempalace)
```

**操作方式**(2 选 1,你拍):

| 选项 | 怎么做 | 利 | 弊 |
|---|---|---|---|
| **(α) Git subtree** | `git subtree add --prefix=mempalace mempalace-go-redis-v2 main` | 历史完整,VSCode 看得到 | 后续 mempalace 改了要 `subtree pull` |
| **(β) 软链 / symlink** | `ln -s ../mempalace-go-redis-v2 mempalace` | 一改两边同步 | Git 不跟踪软链,Windows 软链需要管理员权限 |

推荐 **(α) subtree** 因为 Git 友好 + Windows 友好。但 subtree 加进来的代码后续双向同步麻烦 — 这点要算进 Q3 长期成本。

#### 2.1 目录调整(在 mempalace-go-redis-v2 仓库内)

```
~/Desktop/projects/mempalace-go-redis-v2/
├── main.go                  → 简化,只调 facade.Init(ctx, opts)
├── cmd/
│   ├── server/main.go       → 简化,只调 facade.RunMCP(ctx, svc)
│   └── cli/main.go
├── internal/                → 全部 keep,作为 facade 的实现细节
│   ├── bm25/
│   ├── config/              → 变为 config 包
│   ├── dialect/
│   ├── diary/
│   ├── embedder/
│   ├── entity/
│   ├── extractor/
│   ├── hybrid/
│   ├── instructions/
│   ├── kg/
│   ├── layers/
│   ├── miner/
│   ├── palace/
│   ├── registry/
│   ├── room/
│   ├── sanitizer/
│   └── search/
├── facade/                  ← 新增顶层 package(不再是 internal)
│   ├── facade.go            (Service struct + Init/Close)
│   ├── service.go           (组合 17 internal 装配逻辑)
│   ├── crud.go              (write/read/list/forget 实施:把原 4 类工具合并成 4 个入口)
│   ├── diary.go             (从 internal/diary)
│   ├── kg.go
│   ├── search.go
│   ├── hybrid.go
│   ├── embedder.go
│   └── mcp.go               (暴露 MCP server 给 cmd/server)
├── go.mod                   (mempalace-go-redis 路径不变)
└── README.md
```

**关键**: facade 包跟 internal 平级(`mempalace/facade` 作为公开路径),不放在 internal/ 下,这样外部 import 时不受 internal/ 私有性限制。

#### 2.2 facade.Service 结构

```go
// facade/facade.go
type Service struct {
    Config    *config.Config
    Search    *search.Searcher
    Hybrid    *hybrid.Searcher
    KG        *kg.KnowledgeGraph
    Embedder  *embedder.Embedder
    Diary     *diary.Diary
    // ... 其他 12 个 internal
    // 17 internal 的实例化在 facade.Service.Init() 里完成
}

func (s *Service) Init(ctx context.Context, opts Options) error
func (s *Service) Close() error

// crud.go
func (s *Service) Write(ctx context.Context, key, content string, meta map[string]any) (string, error)
func (s *Service) Read(ctx context.Context, key string) (map[string]any, error)
func (s *Service) List(ctx context.Context, prefix string, limit int) ([]map[string]any, error)
func (s *Service) Forget(ctx context.Context, key string) (bool, error)
```

#### 2.3 验证

```bash
cd ~/Desktop/projects/mempalace-go-redis-v2
go test ./...
./mempalace.exe server &
sleep 2
(echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"t","version":"1"}}}' ; sleep 1 ; echo '{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}' ; sleep 1) | ./mempalace.exe server 2>&1 | head -20
# 行为必须保持不变

# 验证 facade package 能被外部 import
mkdir /tmp/test-import
cd /tmp/test-import
cat > test.go << 'EOF'
package main
import "github.com/dashimaki/mempalace-go-redis/facade"
func main() { _ = facade.Service{} }
EOF
go mod init test
go mod edit -replace github.com/dashimaki/mempalace-go-redis=/c/Users/Administrator/Desktop/projects/mempalace-go-redis-v2
go mod tidy
go build ./...
EOF
```

#### 2.4 git

- 在 mempalace-go-redis-v2 仓库 2-3 个 commit:
  - `feat(facade): add facade package consolidating 17 internal packages`
  - `refactor(cmd): simplify cmd/server to use facade.Service`
  - 测试/文档

---

### Phase 3: garden 包 + 单 exe (5-7 天)

**目标**: `~/Desktop/projects/laputa/garden/` 实现单二进制 garden.exe

#### 3.1 目录

```
~/Desktop/projects/laputa/garden/
├── main.go                  (cobra root + lifecycle)
├── server.go                (HTTP server, 选 HTTP 因为有现成 :7373 概念)
├── crud.go                  (write/read/list/forget, 路由到 governance 或 mempalace/facade)
├── lifecycle.go             (启动 governance + mempalace, 关闭顺序)
├── supervision.go           (crash 重试,健康检查)
└── garden_test.go
```

#### 3.2 main.go 骨架

```go
package main

import (
    "context"
    "github.com/dashimaki/laputa/governance"
    "github.com/dashimaki/laputa/garden"
)

func main() {
    ctx := context.Background()
    cfg := garden.LoadConfig()
    
    g, err := garden.New(ctx, cfg)
    if err != nil { panic(err) }
    
    if err := g.Start(ctx); err != nil { panic(err) }
    defer g.Stop(ctx)
    
    g.Wait(ctx)  // 阻塞,信号来了才退
}
```

#### 3.3 lifecycle.go 设计

```go
type Garden struct {
    Gov     *governance.Engine
    Memory  *facade.Service     // mempalace facade 包
    Config  *Config
    Server  *http.Server
    log     *slog.Logger
}

func New(ctx context.Context, cfg *Config) (*Garden, error)
func (g *Garden) Start(ctx context.Context) error   // 启 governance + memory + server
func (g *Garden) Stop(ctx context.Context) error    // 逆序 stop
func (g *Garden) Health(ctx context.Context) error
func (g *Garden) Wait(ctx context.Context) error    // 阻塞直到 ctx 取消
```

**启动顺序**: governance → memory facade (Init) → HTTP server
**关闭顺序**: HTTP server → memory.Close → governance.Engine.Close
**健康检查**: 每 10s 打 `/health` + 调 facade.Service.Health
**崩溃重试**: server 崩溃 → 5s 后重试 → 连续 3 次失败 → 整体退出 + 报警

#### 3.4 server.go 路由

```go
// 暴露 4 CRUD + 几个 meta endpoint
mux := http.NewServeMux()

// CRUD 4 个 (Q1=b)
mux.HandleFunc("POST /v1/memories",        g.handleWrite)     // write
mux.HandleFunc("GET /v1/memories/{key}",   g.handleRead)      // read
mux.HandleFunc("GET /v1/memories",         g.handleList)      // list
mux.HandleFunc("DELETE /v1/memories/{key}",g.handleForget)    // forget

// Meta
mux.HandleFunc("GET /health", g.handleHealth)
mux.HandleFunc("GET /v1/governance/sections", g.handleGovernanceList)
```

**默认端口**: 7373(跟现在 laputa 一致,切换时无感)

#### 3.5 验证

```bash
cd ~/Desktop/projects/laputa
# 编译
go build -o garden.exe ./garden

# 跑(假设 ~/.garden/config.yaml 已存在)
./garden.exe &
GARDEN_PID=$!
sleep 3

# CRUD 4 个能调
curl -s -X POST http://127.0.0.1:7373/v1/memories -d '{"key":"section:01","content":"test"}'
curl -s http://127.0.0.1:7373/v1/memories/section:01
curl -s http://127.0.0.1:7373/v1/memories
curl -s -X DELETE http://127.0.0.1:7373/v1/memories/section:01

# 健康检查
curl -s http://127.0.0.1:7373/health

# 干净退
kill -INT $GARDEN_PID
```

#### 3.6 git

- 2-3 个 commit:
  - `feat(garden): initial garden package with CRUD 4 routes`
  - `chore(garden): add lifecycle + supervision`
  - 文档/示例

---

### Phase 4: 分开测试 (2-3 天)

**目标**: 用户原话"分开写测试脚本做测试"

#### 4.1 测试结构(4 个测试入口,独立可跑)

```
~/Desktop/projects/laputa/
├── governance/
│   └── governance_test.go        (单元测试,只测试 governance)
├── garden/
│   ├── garden_test.go            (单元测试,只测试 garden)
│   └── integration_test.go       (起 garden,跑 CRUD 4 个真调)
└── e2e/
    └── scenario_test.go          (端到端:起 garden,写一段,读出,验证)

~/Desktop/projects/mempalace-go-redis-v2/
├── facade/
│   ├── facade_test.go            (单元)
│   └── crud_test.go              (facade CRUD 4 个测试)
└── integration/
    └── mcp_compat_test.go        (确认 cmd/server MCP 协议兼容性不变)
```

#### 4.2 验证

```bash
# 独立跑 4 个测试
cd ~/Desktop/projects/laputa && go test ./governance/...
cd ~/Desktop/projects/laputa && go test ./garden/...
cd ~/Desktop/projects/laputa/garden && go test -tags=integration ./...
cd ~/Desktop/projects/mempalace-go-redis-v2 && go test ./facade/...
cd ~/Desktop/projects/mempalace-go-redis-v2 && go test ./integration/...
```

每个测试独立 go test,**不**有一个 cmd/test.go 全跑的入口。

#### 4.3 git

- 各仓库每个测试 1 commit(细粒度)

---

## 5. 总结

```
Phase 1 (governance 整理)     3-5 天
Phase 2 (mempalace facade)    3-5 天
Phase 3 (garden 单 exe)        5-7 天
Phase 4 (分开测试)              2-3 天
─────────────────────────────────────
总计                            13-20 天 (3-4 周)
```

**单一目标检验**(对齐你的本意):

| 你的话 | 本计划如何满足 |
|---|---|
| "API 在现有基础上额外加 CRUD" | governance + facade 既存;garden 在它们之上加 4 个粗粒度 CRUD |
| "改 laputa 现有工作区" | 全在 `~/Desktop/projects/laputa/` 仓库内,不新建 |
| "现有网关 + MCP 全部整理到 garden" | governance.web/ + mempalace/cmd/server 都通过 facade.Service 收入 garden |
| "laputa 那边的职能整理成专门包" | governance/ |
| "mempalace 也变包" | mempalace/facade |
| "分开写测试脚本" | 4 个独立 `go test` 入口 |

---

## 6. 不做清单(沿用 7/6 + 本次新增)

> ⚠️ **路径隔离铁律**:本计划仅在 Go 路径内。Rust 路径(`agent-diva` + `agent-diva-laputa` + `memtle`)独立线,本次完全不动。diva 那边的 `LaputaMemoryProvider` / `HybridMemoryProvider` 本计划既不调用也不修改,等 diva 自己拍 Rust 路径再续。

| 项 | 不做原因 |
|---|---|
| 起 2 个独立的 HTTP / MCP server | 单 exe 只能 1 个端口 |
| Garden 起 MCP server | 不暴露 MCP,因为 facade 已收敛所有 43 tool 为 4 CRUD |
| Pipeline + WorkflowStep + Interceptor | 7/6 doc Layer 2,本次砍 |
| LLM profile routing | 7/6 doc Layer 3,本次砍 |
| Memory file export | 7/6 doc Layer 5,本次砍 |
| 替换 Herme s/diva provider | 本次只准备 garden,切换等下次 |
| Postgres backend | file + sqlite + chroma + redis 够用 |
| 多 agent 中心 | 单 agent |
| Rust 路径 | 用户明示"现在没想好" |

---

## 7. 风险

| 风险 | 概率 | 缓解 |
|---|---|---|
| Phase 1 governance 重整 breaking hermes plugin | 中 | hermes plugin 只 import 几个 exported 符号,优先 audit |
| Phase 2 facade 重整 breaking MCP 协议 | 中 | 必须跑 cmd/server smoke test |
| Phase 3 garden 端口跟现 :7373 冲突 | 低 | 启动前先停老的(在 supervision 做) |
| mempalace 一直跑不起来(7/8 实测) | 高 | Phase 2 facade 部署时优先修:chromadb import 顺序,chromadb_rust_bindings 缺失 |
| 旧 laputa.exe / mempalace.exe 路径被下游依赖 | 中 | Phase 1-2 期间保留.cmd/* 不删,但 cmd/laputa 内部标 `Deprecated` |

---

## 8. 关键坑(经验)

### 8.1 mempalace 必须先解决"跑不起来"

7/8 实测 QQ 通道错误链:
1. MCP server 没启(:8765 不通)
2. hermes venv 里 `from mempalace.service import run_diary_write` 触发 `from .mcp_server import tool_diary_write` → `from chromadb.errors import NotFoundError` → **ModuleNotFoundError**
3. 强制先 `import chromadb` 后 `from mempalace.service import run_diary_write` → `chromadb.PersistentClient(path=...)` 失败: `chromadb.api.rust → import chromadb_rust_bindings` 失败
4. chromadb_rust_bindings 包**是装着的**(`ls $VENV/Lib/site-packages/chromadb_rust_bindings/` 有 .pyd),但 `chromadb 1.5.9` 在 Windows venv 上有命名空间包冲突

**Phase 2 之前必须修 1 + 3 + 4**。否则 facade 起来后 backend 还是开不了。

### 8.2 hermes plugin 不能 broken

Phase 1 改 governance 包时 hermes plugin 在 `C:/Users/Administrator/AppData/Local/hermes/plugins/laputa/__init__.py` 调 :7373 — 它直接走 HTTP,**不** import Go 代码,所以 governance 重整不影响它。但 Phase 3 之后 garden 接管 :7373,hermes 不动也照旧能跑。

### 8.3 supervision 红题:stderr 重定向

7/6 doc 已提:旧的 `laputa-supervisor.cmd` 用 `start /B` stderr 没重定向 → crash 无现场。**Phase 3 supervision.go 必须**用同步文件日志 `~/.garden/garden.log`,不能用 `&` 启动。

---

## 9. 验证命令速查

### 9.1 各阶段 smoke

```bash
# Phase 1 完:governance package 编译 + 测试
cd ~/Desktop/projects/laputa
go build ./governance/...
go test ./governance/...

# Phase 2 完:mempalace facade + cmd/server 兼容
cd ~/Desktop/projects/mempalace-go-redis-v2
go test ./...
./mempalace.exe server &
sleep 2
(echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"t","version":"1"}}}' ; sleep 1 ; echo '{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}' ; sleep 1) | ./mempalace.exe server 2>&1 | head -20

# Phase 3 完:garden 单 exe
cd ~/Desktop/projects/laputa
go build -o garden.exe ./garden
./garden.exe &
GARDEN_PID=$!
sleep 3
curl -s http://127.0.0.1:7373/health
curl -s -X POST http://127.0.0.1:7373/v1/memories -d '{"key":"section:01","content":"test"}'
curl -s http://127.0.0.1:7373/v1/memories/section:01
curl -s -X DELETE http://127.0.0.1:7373/v1/memories/section:01
kill -INT $GARDEN_PID

# Phase 4 完:4 个独立测试
cd ~/Desktop/projects/laputa && go test ./governance/...
cd ~/Desktop/projects/laputa && go test ./garden/...
cd ~/Desktop/projects/laputa/garden && go test -tags=integration ./...
cd ~/Desktop/projects/mempalace-go-redis-v2 && go test ./facade/...
cd ~/Desktop/projects/mempalace-go-redis-v2 && go test ./integration/...
```

---

## 10. 下一步

**等你一句"开 Phase 1"我就动 governance 重整**(预计先改 `package laputa` → `package governance`,做完业务代码 + 测试不引入任何 breaking change,然后才动内部子包)。

如果你 Q3 的"mempalace 也变包"实际上不是 Q3=b (我在第 2 节有标注"如果你不同意这层,Q3 重谈")— 也请提醒一下。
