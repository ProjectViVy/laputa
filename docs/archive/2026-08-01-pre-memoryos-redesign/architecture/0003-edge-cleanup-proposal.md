# ADR-0003: 边角料治理提案

> **Status**: proposed (松本/2026-07-15)
> **范围**: 仓库内未交付承诺 / deprecated 残骸 / 空骨架 / 空文件
> **不动**: 主架构 (laputa/governance + mentle/facade + garden 三层 + 4 CRUD + 9-step agentic_recall_v1)
> **目标**: 把"为通用性买了没用的保险"全部清理，不引入新 feature

---

## 0. 范围声明

本文档**只处理"边角料"**——下列 3 类问题：

  1. **deprecated / fallback binary** — 老路径残留，新版不再走
  2. **空骨架** — 文件声明了但内容空，未承载承诺
  3. **未跟踪历史草稿** — `git status` 长期 untracked，不被 docs/ 收口

**本文档不处理** (留给后续 ADR/讨论):

  - 主架构变更
  - ingest pipeline / conflict resolution / tiered retrieval (松本已提出的 7 件事)
  - laputa 14 section 中 10–14 tbd 的 schema / owner 拍板 (本质治理决策)

---

## 1. 现状清单 (实地验证 2026-07-15)

### 1.1 deprecated binary 残留

| 路径 | 大小 | 说明 |
|---|---|---|
| `laputa/cmd/laputa/main.go` | 6738 B / ~140 行 | Phase 0 起就标 deprecated，留 :7373 HTTP fallback |
| `laputa/laputa.exe` | 30 MB | 已编译 binary，2026-07-05 |
| `laputa/laputa.exe~` | 30 MB | 备份 binary (emacs-style 备份命名) |

**实情**: Phase 0 后所有 laputa 业务代码都已迁入 `laputa/governance/`，garden.exe 是唯一对外 HTTP。`cmd/laputa/main.go` 注释里写 "deprecated, fallback :7373"——保留理论动机是"garden 起不来时临时替"，但有 3 个问题：

  (a) 同一 :7373 端口冲突——garden.exe 跑时它**起不来**，fallback 名存实亡
  (b) 文件名冲突——`laputa.exe` (binary) ≠ `laputa/` (repo dir)，运行 `laputa` 在 PATH 命中的是 binary 而不是 repo
  (c) `laputa.exe~` 是 7/5 5:45 编译的，跟 7/5 5:46 当前 binary 一字之差，没保留价值

### 1.2 config.example.yaml 空骨架

`garden/config/config.example.yaml`:

```yaml
# Garden config example (Phase 2+)
server:
  addr: ":7373"
governance:
  store_dir: "~/.laputa/sections"
mentle:
  config_dir: "~/.mentle"
```

但 main.go 实际读的 env 是 `GARDEN_ADDR` / `GARDEN_GOVERNANCE_DIR` / `GARDEN_MENTLE_CONFIG_DIR`——**没有任何代码 load 这个 yaml**。

含义: 文件承诺一个 YAML 配置体系，README 也没真提，**实际所有配置走环境变量**。零配置使用没问题，但文件留存在仓库里是文档噪音。

### 1.3 probe 空文件

`probe` (0 字节) 在 garden/ 顶层，git untracked。推测: 早期调试残留。删除无影响。

### 1.4 git untracked 历史草稿 6 个

`git status` (untracked only):

```
LAPUTA-plan1-remaining-2026Q3.md          35 KB   2026-06-30
MATSUMOTO-laputa-eino-rhythm-autodream.md 20 KB   2026-06-28
MATSUMOTO-overall-refactor-2026Q3.md     79 KB   2026-06-28
MATSUMOTO-refactor-onepager.md            5 KB   2026-06-28
SESSION-REPORT-2026-07-08.md             13 KB   2026-07-08
laputa-galaxyos-structural-borrow.md      6 KB   2026-06-30
laputa-mempalace-batch-extensions-2026Q3.md 21 KB  2026-07-04
docs/architecture/garden-laputa-architecture.html 14 KB (本次新增)
probe                                       0 B
```

每个文件性质:

  - **LAPUTA-plan1-remaining-2026Q3.md** — 6/30 排期草稿，phase 0 已是它落地验证后的 superset
  - **MATSUMOTO-laputa-eino-rhythm-autodream.md** — 6/28 autoDream 早期设计，对应 ARCHITECTURE.md "rhythm" 段
  - **MATSUMOTO-overall-refactor-2026Q3.md** — 6/28 重构笔记，已被 ADR-0001 取代
  - **MATSUMOTO-refactor-onepager.md** — 上面那个的 one-pager
  - **laputa-galaxyos-structural-borrow.md** — 6/30 galaxyOS 借鉴探索，无后续 ADR
  - **laputa-mempalace-batch-extensions-2026Q3.md** — 7/4 增量扩展草案，已被 ADR-0002 实现
  - **SESSION-REPORT-2026-07-08.md** — 7/8 会话历史，不是治理文档

docs/archive/ 现状:

```
docs/archive/
└── GARDEN-PLAN-2026-07-08.md (25 KB)  ← v1 计划已归档
```

意味着 archive 目录**已存在 + 已用过一次**（v1→v3 计划替换），模式清晰。

### 1.5 14 section 中 5 个 tbd (本文不解决, 仅记录)

`laputa/governance/engine.go` SectionRegistry:

```go
SectionJournalReflective: {WriteAuth: AuthorityTBD,  SchemaOwner: "tbd",        Status: "tbd"},
SectionProposalInbox:     {WriteAuth: AuthorityTBD,  SchemaOwner: "tbd",        Status: "tbd"},
SectionChangelog:         {WriteAuth: AuthorityTBD,  SchemaOwner: "tbd",        Status: "tbd"},
SectionReportIndexes:     {WriteAuth: AuthorityTBD,  SchemaOwner: "tbd",        Status: "tbd"},
SectionAAAKSummaries:     {WriteAuth: AuthorityTBD,  SchemaOwner: "tbd",        Status: "tbd"},
```

意思是 14 个 section 已有声明 + 在 FileStore 落位 (上次实测 list 14 个全在), 但 5 个 owner/schema 完全没拍。

**本文不解决**——拍这 5 个的 owner 是治理决策 (谁写、读权给谁、status 怎么从 tbd→stable)，需要 松本单独提案。提议放到 ADR-0004。

---

## 2. 处理提案 (4 项, 编工作序号)

### P1: 删 `laputa/cmd/laputa/` + 两个 binary

  - 删 `laputa/cmd/laputa/main.go` (整个目录, 140 行)
  - 删 `laputa/laputa.exe` (30 MB binary)
  - 删 `laputa/laputa.exe~` (30 MB 备份)
  - 影响: `garden.exe` 仍跑 :7373; garden 起不来时本机没有任何 laputa 兜底——靠 go rebuild garden
  - 不影响: governance 业务代码 (在 `governance/engine.go` 里, 完全独立)
  - 验收: `git grep "deprecated" laputa/` 应无命中 + `find laputa -name '*.exe*'` 返回空

### P3: 删空骨架 / 空文件

  - 删 `garden/config/config.example.yaml` (142 B)
  - 删 `probe` (0 B)
  - 影响: README 里如有提及 `config.example.yaml` 需同步删; 实际查 README 当前没提, 安全
  - 验收: `ls garden/config/probe` 均 NotFound

### P4: 6 个历史草稿 → `docs/archive/historical/`

  - 不删内容 (历史信息保留), 只挪位置
  - 新建 `docs/archive/historical/` 子目录
  - 移 7 个文件进入 (含 SESSION-REPORT-2026-07-08.md)
  - 同步在每个文件第 1 行加 deprecation 戳:

```
DEPRECATED YYYY-MM-DD — superseded by ADR-NNNN (link)
```

  - 或: 在 `docs/archive/historical/INDEX.md` 列归档清单 + 各文件 ADR 归属

  - 影响: git 工作树变更; untracked 转 tracked under docs/archive/historical/
  - 不影响: 任何运行时

### P5 (附加): 同步 garden/main.go 注释 / README

  - main.go 头部改 `// Phase 5 governs pipeline + agentic RAG context. 详见 ADR-0001 / ADR-0002`
  - 删所有"Phase 1+ 决定" / "待定" / "see ADR-0001 for the merged plan" 这类 forward-pointer 注释
  - README 任何指向 `cmd/laputa/` 的引用删

---

## 3. 实施顺序 + 时间盒

  - P3 → P1 → P5 → P4 (按依赖顺序)
  - **预计**: P3 ≈ 3 min | P1 ≈ 10 min (just delete) | P5 ≈ 15 min | P4 ≈ 10 min
  - 总计 ≤ 40 min
  - 每步后跑: `cd garden && GOSUMDB=off go build ./...` 和 `go test ./...`

---

## 4. 不在本提案的事

  - **松本提出的 7 件事** (auto-extract / conflict resolution / 接口不下放 / report 通道 / tiered retrieval / 人格权重 / memory injection) → 留待 ADR-0004+ 单独提案
  - **14 section 中 5 个 tbd 的 owner/schema** → 留待 ADR-0005 (本质治理决策)
  - **mempalace-py 物理隔离完成**, 不动
  - **Rust 路径** 完全不动

---

## 5. 风险评估

  - **低风险**:
    - P3 (删空文件/空 yaml) — 零运行时影响
    - P4 (挪历史文件到 archive/historical/) — 静态文件, 不影响构建
  - **中风险**:
    - P1 (删 fallback binary) — 失去"garden 起不来时 hax"路径; 但 hax 这事本就不该依赖一个老 binary
  - **校验**: 每步删完跑 `cd garden && go build ./... && go test ./...`, 4 个独立 test entrypoint 全过

---

## 6. 验收清单

整体 commit 之后:

```bash
# P1
find laputa -name '*.exe*' -o -name 'laputa/main.go' | grep . && echo FAIL || echo P1-OK

# P3
test -f garden/config/config.example.yaml && echo FAIL-P3-YAML
test -f probe && echo FAIL-P3-PROBE
echo P3-OK-if-no-output

# P4
ls docs/archive/historical/ | wc -l   # 应 ≥ 6
test -f docs/archive/historical/INDEX.md && echo P4-INDEX-OK

# 总验证
cd laputa && go test ./governance/...
cd ../mentle && go test ./facade/...
cd ../garden && GOSUMDB=off go test ./... && GOSUMDB=off go test -tags=e2e ./e2e/...

# 运行时
./garden.exe &
curl -s http://127.0.0.1:7373/health
curl -s http://127.0.0.1:7373/v1/context/resolve -X POST -d '{"intent":"x","session_id":"y"}' -H 'Content-Type: application/json'
```

全部 ✓ → commit + push

---

## 7. 关联

  - **ADR-0001** Garden 单 exe 合并 (paya support) — P1 删 cmd/laputa 是 ADR-0001 §3.5 "laputa.exe 退役为 deprecated" 的最终落地
  - **ADR-0002** Governed Pipeline + Agentic RAG — 本提案不改 planner 或 Agentic RAG 能力
  - **GARDEN-PLAN §3.3 #4** "mentle 启动问题修复时间: ⚠️未修复, 通过 facade + graceful degradation 隔离" — 本提案不动这条
  - **draft ADR-0004 / 0005** — 由松本后续提案, 涉及 ingest / tiered / section-owner

---

*作者: 松本 (大湿)*
*日期: 2026-07-15*
*提案目标: 边角料治理, 主架构不动*
