# Laputa — Garden Laputa

> **一个花园，三件事**：在肥沃的土壤里挖呀挖呀挖，在碧蓝的天空下晒呀晒呀晒，
> 而你——在与 AI 对话的过程里种下一棵记忆。

---

## 1. 设计哲学

Laputa 的名字取自宫崎骏《天空之城》。但 Garden Laputa 不只是一座飘在云端的城堡——
它是一座**长出来的花园**。人格和记忆从来不是被设计出来的，
它们是在合适的条件下，**缓慢地、自然地**从土壤里生长出来的。

```
                ☁  天空  sky          ← governance（治理）
            晴 / 雨 / 风 / 霾           外界环境决定花园能不能长大
               ／    ＼
              ／      ＼
             ／   🌱    ＼
            ／  garden   ＼
           ／   laputa    ＼
          ／                ＼
        ~~~~~~~~~~~~~~~~~~~~~~~~~~~~
                  土壤  soil          ← mentle（记忆）
            化石 / 根系 / 矿藏          蕴藏人类历史一切信息的地方
```

### 三件事

| 角色 | 隐喻 | 工程映射 | 它做什么 |
|---|---|---|---|
| **土壤** | 蕴藏人类历史一切信息的地方。可能有化石，但全数包裹。 | **`mentle`**（原 mempalace-go） | 持久存储、向量检索、知识图谱、日记 |
| **天空** | 外界环境因素。空气质量、温度、日照、风雨。决定 garden 长得好不好。 | **`governance`**（原 laputa 内部包） | 14 section 治理、写权规则、step 编排、审计 |
| **种植** | 人与 AI 对话的过程。每一句话是浇的水、施的肥。 | **`garden` CLI / HTTP**（Phase 1+） | 4 CRUD（write/read/list/forget）+ 调度 |

### 挖呀挖呀挖

记忆的检索不是查询，是**发掘**。在小小的花园里挖——找到的可能是种子（一条事实），
可能是化石（一段旧关系），可能是一块从未见过的东西（一段没说出口的偏好）。
每一次挖掘都在改变土壤的形状。

### 花园长得好不好

取决于**空气质量**——也就是 governance：
写权是否混乱？section 之间是否联动？审计是否完整？step 是否原子？
这些不是技术细节，它们是**花园能呼吸的空气**。

---

## 2. 仓库结构

```
~/Desktop/garden/                  ← 工作区根（本仓库）
│
├── README.md                      ← 本文件：哲学 + 架构入口
├── GARDEN-PLAN.md                 ← 实施计划（5 个 Phase）
├── docs/architecture/
│   └── 0001-garden-merge.md       ← ADR：为什么选 Garden 顶层 + 单 exe
│
├── laputa/                        ← 仓库 1：治理层（天空）
│   ├── go.mod                     module github.com/dashimaki/laputa
│   ├── governance/                ← 顶层包（原 laputa.go 拆分入此）
│   ├── cmd/laputa/                ← 已 deprecate，保留作 fallback
│   └── ...
│
├── mentle/                        ← 仓库 2：记忆层（土壤）
│   ├── go.mod                     module github.com/dashimaki/mentle
│   ├── facade/                    ← 顶层包（4 CRUD + Service）
│   ├── cmd/server/                ← stdio MCP（被 facade 替代中）
│   └── internal/                  ← 17 个 internal 包保持原状
│
└── garden/                        ← 仓库 3：种植层（CLI/HTTP）   [Phase 1+]
    └── go.mod                     module github.com/dashimaki/garden
```

### 三层职责

| 层 | 仓库 | package 顶层 | 何时引入 |
|---|---|---|---|
| **记忆（土壤）** | `mentle/` | `facade` | **Phase 0 进行中** |
| **治理（天空）** | `laputa/` | `governance` | **Phase 0 进行中** |
| **种植（入口）** | `garden/` | `crud` / `server` | Phase 1 起 |

---

## 3. 当前状态：Phase 4 待开

| Phase | 内容 | 状态 | commit |
|---|---|---|---|
| **0** | 仓库物理搬迁 + module path 重命名 + 抽顶层 governance/facade 包 | ✅ 完成 | `7e16be3` |
| **1** | garden 仓库骨架 + 4 CRUD（write/read/list/forget） | ✅ 完成 | `48cc0fc` |
| **2** | garden HTTP server + 路由分发 | ✅ 完成 | `3537c4c` |
| **3** | lifecycle + supervision + 日志 | ✅ 完成 | `673c27c` |
| **4** | 4 个独立测试入口（governance / facade / garden / 集成） | 🟡 待开 | — |

**主体进度: 80% (4/5 phase)。** 完整状态冻结见 [`GARDEN-PLAN.md`](./GARDEN-PLAN.md)。
架构决策记录见 [`docs/architecture/0001-garden-merge.md`](./docs/architecture/0001-garden-merge.md)。
设计哲学与历史背景见 [`NEW-LAPUTA.md`](./NEW-LAPUTA.md)（2026-07-06 决策快照）。

---

## 4. 引用

### 4.1 项目内
- 实施计划：[`GARDEN-PLAN.md`](./GARDEN-PLAN.md)
- ADR-0001：[`docs/architecture/0001-garden-merge.md`](./docs/architecture/0001-garden-merge.md)
- 历史快照：[`NEW-LAPUTA.md`](./NEW-LAPUTA.md)
- 旧版计划（已废弃）：[`GARDEN-PLAN-2026-07-08.md`](./GARDEN-PLAN-2026-07-08.md)

### 4.2 命名对照

| 旧名 | 新名 | 含义 |
|---|---|---|
| `mempalace-go-redis-v2/` | `mentle/` | 土壤。去掉 -go 后缀，因 Go 已不是区分标志 |
| `mempalace-go-redis`（module path） | `mentle`（module path） | 同上 |
| `laputa.go`（monolith） | `governance/engine.go` + 5 sub-package | 拆包，仍叫 laputa 仓库 |
| `garden`（项目名） | `laputa`（项目名） + README 副标题 Garden Laputa | 项目仍叫 laputa |
| `garden.exe`（未来二进制） | 待定 | Phase 1 决定 |

### 4.3 不在本仓库
- **`mempalace-py`**（Python 版）已在 `~/Desktop/morediva/.workspace/mempalace-py/`，**不动**
- **Rust 路径**（memtle crate、agent-diva-laputa）**完全不动**

---

## 5. 一句话架构

> **单 `garden` CLI / HTTP** 调用 **`laputa`（governance）** 和 **`mentle`（facade）**，
> 对外暴露 **4 个 CRUD**：`write` / `read` / `list` / `forget`。
> `key` 前缀路由：`section:*` → governance；其它 → mentle facade。

---

*作者：松本（大湿）*
*日期：2026-07-09*
*状态：Phase 0 进行中*