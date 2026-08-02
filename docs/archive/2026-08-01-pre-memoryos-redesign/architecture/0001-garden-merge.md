# ADR-0001: Garden 单 exe 合并 laputa + mentle

> **状态**: Proposed
> **日期**: 2026-07-09（基于 2026-07-08 初版更新）
> **作者**: 松本(大湿)
> **范围**: Go 路径(laputa + mentle,Rust 路径完全不动)
> **替代**: ~~GARDEN-PLAN-2026-07-08.md~~(已废弃)
> **关联 README**: `~/Desktop/garden/README.md`(设计哲学)
> **关联 PLAN**: `~/Desktop/garden/GARDEN-PLAN.md`(实施计划)

---

## 0. 设计哲学（Garden Laputa）

> **人格和记忆不是设计出来的，是长出来的。**

```
        ☁  天空  sky          ← governance（治理）
      晴 / 雨 / 风 / 霾          外界环境决定花园能不能长大
         ／    ＼
        ／   🌱  ＼
       ／ garden  ＼
      ／  laputa   ＼
     ／              ＼
   ~~~~~~~~~~~~~~~~~~~~~~~~~~~~
          土壤  soil         ← mentle（记忆）
    化石 / 根系 / 矿藏          蕴藏人类历史一切信息的地方
```

| 角色 | 隐喻 | 工程映射 |
|---|---|---|
| **土壤** | 蕴藏人类历史一切信息的地方。可能有化石,但全数包裹。 | `mentle`(原 mempalace-go) |
| **天空** | 外界环境因素。空气质量、温度、日照、风雨。决定 garden 长得好不好。 | `governance`(原 laputa 内部包) |
| **种植** | 人与 AI 对话的过程。每一句话是浇的水、施的肥。 | `garden` CLI / HTTP（Phase 1+） |

**项目名** = `laputa`(不变),**README 副标题** = "Garden Laputa"。

---

## 1. 上下文(Context)

### 1.1 当前事实(2026-07-08 实地验证)

| 项目 | 路径 | 形态 | 角色 |
|---|---|---|---|
| **laputa** | `~/Desktop/projects/laputa/` | Go module,`package laputa` + 5 internal,起 `laputa.exe` + HTTP :7373 | 14 section 治理 |
| **laputa** | `~/Desktop/garden/laputa/` | Go module,`package governance` + 5 sub-package,起 `laputa.exe`（deprecated） + HTTP :7373 | 14 section 治理 |
| **mentle** | `~/Desktop/garden/mentle/` | Go module,17 internal 包 inline 组装在 `cmd/server/main.go` (1207 行),没有公开 facade | 17 类记忆工具 |
| **garden** | `~/Desktop/garden/garden/` | 本次新仓库（Phase 1 引入） | 单 exe 入口 |

> 注:`mempalace-py`(Python 版) 已搬到 `~/Desktop/morediva/.workspace/mempalace-py/`,**与本计划无关**,**不再访问**。

### 1.2 真问题

| # | 问题 | 现象 |
|---|---|---|
| 1 | **mentle 起不来** | 2026-07-08 QQ 通道实测:hermes 通过 `laputa-py/src/laputa/bridge/palace_bridge.py → run_diary_write` 走 mermaid 服务时,本地 chromadb 1.5.9 在 venv 下 `chromadb.api.rust → import chromadb_rust_bindings` 失败,跟 mentle (Go) 无关,但暴露出 Go 版缺失 facade 启动隔离 |
| 2 | **CRUD 入口不收敛** | mentle 暴露 43 个 MCP 工具,laputa 暴露 14 section JSON HTTP;上层 agent 必须知道两套接口的差异 |
| 3 | **没有 step 编排层** | 改 1 个 section 不联动其他 section;改 history_md 不一致变 changelog |
| 4 | **三 binary 运维成本** | laputa.exe / mentle.exe / 未来 garden.exe,三个 supervisor 三个日志 |

---

## 2. 选项(Options Considered)

### Option A: 维持现状,修 mentle 启动(已尝试失败)

- 修 chromadb_rust_bindings 缺失问题
- 不能解决 #2 #3 #4
- **否决**:只解决 1 个症状,不解决 4 个问题

### Option B: laputa + mentle 起两个 binary + stdio 桥(7/6 拍板 1.0)

- 之前讨论过,已否决
- **否决**:桥不稳,两个 supervisor

### Option C: 合并成单 binary(laputa 内化 mentle)

- 取消仓库边界,两个项目合并
- **否决**:丢 Git 历史;mentle 独立演进可能丢失

### Option D: Garden 顶层 + laputa/mentle 作为库(本次决策)

- garden 是新的 Go 仓库,顶层 crate,作为单二进制
- laputa 重整为单一 `governance` 包(package 边界保留,内部 5 个 sub-package)
- mentle 17 internal 整合成 1 个 `facade` 包(package 边界保留)
- garden 通过 `go.mod` `replace` 指令引用 `../laputa` 和 `../mentle`（物理都在 `~/Desktop/garden/` 内）
- **本计划选 D**

---

## 3. 决定(Decision)

### 3.1 工作区布局（物理搬入 garden 内部）

```
~/Desktop/garden/                  ← 工作区根（独立仓库，无 .git 待 init）
├── docs/
│   └── architecture/
│       └── 0001-garden-merge.md   ← 本文件
├── GARDEN-PLAN.md                 ← 实施计划
├── README.md                      ← 设计哲学 + 架构入口
├── laputa/                        ← 仓库 1（搬入自 ~/Desktop/projects/laputa/）
│   ├── go.mod                     module github.com/dashimaki/laputa
│   ├── governance/                ← 新顶层包（package governance）
│   │   ├── engine.go              ← 原 laputa.go
│   │   ├── rhythm/                从 internal/rhythm/
│   │   ├── scheduler/             从 internal/scheduler/
│   │   ├── store/                 从 internal/store/redis/
│   │   ├── wakeup/                从 internal/wakeup/
│   │   └── web/                   从 internal/web/
│   ├── cmd/laputa/                ← 退役（deprecated,留 binary 兜底 :7373）
│   └── .git/                      ← 原 git 仓库，搬入后保持
│
├── mentle/                        ← 仓库 2（搬入自 ~/Desktop/projects/mempalace-go-redis-v2/）
│   ├── go.mod                     module github.com/dashimaki/mentle   ← 改名!
│   ├── facade/                    ← 新顶层包，公开 Service + 4 CRUD
│   │   ├── facade.go              Service struct + Init/Close
│   │   ├── crud.go                write/read/list/forget 实现
│   │   └── ...
│   ├── internal/                  ← 17 个 internal 包保留为实现
│   ├── cmd/server/                ← 简化为调 facade
│   └── .git/                      ← 原 git 仓库，搬入后保持
│
└── garden/                        ← 仓库 3（Phase 1 起新建）
    ├── go.mod                     module github.com/dashimaki/garden
    ├── main.go                    cmd/garden 入口
    ├── internal/
    │   ├── server/                HTTP server 路由层（Phase 2）
    │   ├── crud/                  write/read/list/forget 4 个动作
    │   ├── router/                key 前缀 → governance/facade 分发
    │   ├── lifecycle/             启停顺序（Phase 3）
    │   └── supervision/           crash 重试（Phase 3）
    └── config/
```

### 3.2 CRUD 4 个 API(Q1=b)

| API | HTTP 路由 | 输入 | 输出 | 路由规则 |
|---|---|---|---|---|
| **write** | POST /v1/memories | `(key, value, metadata?)` | `record_id` | `key` 以 `section:` 开头 → governance,其它 → mentle facade |
| **read** | GET /v1/memories/{key} | `(key)` | `(value, metadata)` | 同上 |
| **list** | GET /v1/memories | `(prefix?, limit?)` | `[]record` | `prefix` 路由到对应 backend |
| **forget** | DELETE /v1/memories/{key} | `(key)` | `bool` | 同上 |

**路由规则在 garden 层做,不在 facade / governance 层**:
```go
// internal/router/router.go
func Route(ctx context.Context, key string) (Backend, error) {
    if strings.HasPrefix(key, "section:") { return governance, nil }
    if strings.HasPrefix(key, "memory:")  { return mentleFacade, nil }
    return nil, fmt.Errorf("unknown key prefix: %s", key)
}
```

### 3.3 包重构原则

- **laputa**: 仅 package 名变更(`laputa` → `governance`),其他保留;`NewEngine(store SectionStore)` 签名不变 (避免 breaking hermes plugin)
- **mentle**: 仅新增顶层 `facade` 包,17 internal 全部保留;`facade.Service.Init(ctx, opts)` 模拟原 cmd/server 启动逻辑
- **garden**: 全新代码,从 0 写

### 3.4 命名规则

- Go module 名:
  - `github.com/dashimaki/laputa` **不变**
  - `github.com/dashimaki/mempalace-go-redis` **改名为** `github.com/dashimaki/mentle`（去 `-go` 后缀）
- 物理目录: `~/Desktop/projects/laputa` → `~/Desktop/garden/laputa`；`~/Desktop/projects/mempalace-go-redis-v2` → `~/Desktop/garden/mentle`
- Go package 名: `governance` / `facade` / `crud` 等小写单数
- 项目名: `laputa`(不变),README 副标题 = "Garden Laputa"
- 二进制名: `garden.exe`(Phase 1+ 唯一二进制)

### 3.5 范围边界

| 做 | 不做 |
|---|---|
| garden 单 exe | 重写 laputa 业务代码 |
| governance/facade 顶层包 | 重写 mentle-go 17 internal |
| 4 CRUD API | 5 facade 业务方法(7/6 doc §3.3) |
| laputa.exe 退役为 deprecated | Pipeline / WorkflowStep / Interceptor(7/6 §5) |
| mentle-go cmd/server 简化为 facade | LLM profile routing |
| garden HTTP server | Memory file export (memU 抄) |
| 4 个独立 test 入口 | MCP server(garden 取而代之) |
| | Postgres / 多 agent / 多模态 |
| | **Rust 路径完全不动** |

---

## 4. 后果(Consequences)

### 4.1 正面

| # | 后果 |
|---|---|
| 1 | 4 个 CRUD 收敛上层接口,hermes plugin 不用知道 14 section vs 43 tool 之分 |
| 2 | mentle 启动问题被 facade 隔离 — 启动失败时,garden 可降级为只治理 mode |
| 3 | 单一 supervisor + 单一日志(`~/.garden/garden.log`) |
| 4 | laputa.exe / mentle.exe 二进制退役,但 source 仓库保留,可独立演进 |
| 5 | 工作区 `~/Desktop/garden/` 干净,不混 Python 版 mempalace |
| 6 | 命名清晰:laputa / mentle / garden 三层,各自负责 |

### 4.2 负面

| # | 后果 | 缓解 |
|---|---|---|
| 1 | `mempalace-go-redis` module 名 → `mentle` 改名影响外部 import | 本次只有 garden import,改 go.mod path 即可 |
| 2 | laputa.go package 名 laputa → governance 是 breaking change | hermes plugin 只调 HTTP,不 import Go,不受影响;写 deprecation 注释提醒 |
| 3 | garden 依赖 laputa + mentle **两个外部 go module**,跨仓库调试需要 replace | go.mod 用 replace 指向 `../laputa` 和 `../mentle`（物理同在 garden 工作区） |
| 4 | 全新工作区 `~/Desktop/garden/` 之前意外混入 Python 版 mempalace | 7/8 已搬到 `~/Desktop/morediva/.workspace/mempalace-py/`,**物理隔离完成** |

### 4.3 风险

| 风险 | 概率 | 缓解 |
|---|---|---|
| governance 重命名 breaking hermes plugin | 低 | hermes 只调 :7373 HTTP |
| facade 整合改变 cmd/server 行为 | 中 | Phase 2 必须跑 MCP smoke test |
| 命名混乱（mempalace ≠ mentle） | 低 | 本计划统一用 mentle，文档已替换完成 |
| 物理搬运丢失 git 历史 | 中 | 用 `mv` 而非 `cp`，git 目录完整带过去 |
| mentle 启动问题（7/8 实测）在 facade 部署时再爆发 | 高 | Phase 2 facade 之前先修复 chromadb 1.5.9 兼容 |

---

## 5. 决策时间线

| 时间 | 事件 |
|---|---|
| 2026-07-06 | `NEW-LAPUTA.md` 拍板"Garden = facade + pipeline + 治理层" (6-8 周 scope) |
| 2026-07-08 上午 | 用户说"mempalace go v2 改造成库 + 上层 garden 包,启一套稳定运行" |
| 2026-07-08 中午 | 4 Q 拍板:Q1=b CRUD=write/read/list/forget; Q2=b garden 在 laputa 仓库内顶层; Q3=b+i 仓库边界保留+vendor;i=subtree/软链 二选一 |
| 2026-07-08 下午 | 用户提醒"不要 go 跟 rust 搞混" |
| 2026-07-08 下午 | 用户提出"全新工作区,避免再搞混" → 选 B = `~/Desktop/garden/` 顶层隔离 |
| 2026-07-08 下午 | 用户说"移动到 morediva/.workspace,不再看 mopalace-py" → 完成 |
| 2026-07-08 下午 | 用户命名 Go 版 mempalace = **mentle-go**(临时命名,本计划沿用) |
| 2026-07-08 18:00 | **本文档**（ADR-0001 初版）写完 |
| 2026-07-09 | **物理布局升级**：mempalace-go-redis-v2 和 laputa 整库搬入 `~/Desktop/garden/` 内部；module path `mempalace-go-redis` → `mentle`（去 `-go` 后缀）；项目名仍是 `laputa`，README 副标题 = "Garden Laputa"；设计哲学（土壤/天空/种植）写入 README.md |

---

## 6. 关联

| 文档 | 路径 | 关系 |
|---|---|---|
| 7/6 拍板文档 | `~/Desktop/garden/NEW-LAPUTA.md` | 上游,本 ADR 取代其 §3 Garden 部分 |
| 7/8 旧计划 | `~/Desktop/garden/GARDEN-PLAN-2026-07-08.md` | 取代,作为'废弃参考'保留 |
| 7/8 新计划 | `~/Desktop/garden/GARDEN-PLAN.md` | **本文档实施版** |
| morediva AGENTS.md | `~/Desktop/morediva/AGENTS.md` | 参考其 §3 DOC_GOVERNANCE 风格 |

---

## 7. 未决议题(本 ADR 写完之后)

| # | 待定 | 何时拍 |
|---|---|---|
| 1 | mentle module path 是否真的从 `mempalace-go-redis` 改名为 `mentle`?（影响 GitHub 仓库 rename） | Phase 0 内（物理搬运同时改） |
| 2 | Phase 2 mentle vendor 方式：`git subtree` vs `软链` | 已废除 — 物理搬入 garden 内部后无需 vendor |
| 3 | garden HTTP 端口:7373(继承 laputa)还是 7374(自占)? | Phase 2 |
| 4 | supervision 行为:1 次失败停一切 vs 3 次重试? | Phase 3 |
| 5 | mentle 启动问题（7/8 实测）修复时间窗：Phase 0/1/2? | Phase 2 验收前必须修 |
