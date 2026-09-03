# Garden Authority & Recovery — 手动派工 TODOLIST

**项目：** Garden Authority & Recovery Convergence
**流程：** BMAD Project-sized
**状态：** `execution / COMPLETE`
**GOAL：** `active → complete`；Owner 已明确要求执行并完成 Waves 0–3
**用途：** 大湿手动向子智能体分派 Story
**更新：** 2026-09-03

> 本文件是派工入口。详细验收标准以对应 Epic 文档、`api-contract.md`、`test-inventories.md` 为准。不得只凭本页标题自由发挥。

长时运行、状态机、并发锁、dirty-tree 采用、阻塞恢复和 Wave 提交规范见 [GOAL Execution Runbook](GOAL-EXECUTION-RUNBOOK.md)。本清单定义 Story 内容，Runbook 定义 Story 如何被持续执行。

---

## 0. 所有 Agent 开工前必须阅读

1. [PRD](prd.md)
2. [Architecture Spine](ARCHITECTURE-SPINE.md)
3. [API Contract](api-contract.md)
4. [Test Inventories](test-inventories.md)
5. [Dirty-tree Baseline](baseline-inventory.md)
6. 自己负责的 Epic 文档
7. [GOAL Execution Runbook](GOAL-EXECUTION-RUNBOOK.md)
8. 涉及 Persona/ACTMEM 时读取 DIVA 权威实现：
   - `C:/Users/Administrator/Desktop/morediva/agent-diva/agent-diva-laputa/`

### 统一执行规则

- [x] 开工前记录 `git status --porcelain=v1 -uall`、HEAD、目标文件 diff。
- [x] 先声明允许修改路径；不得接管、覆盖或回滚已有 dirty diff。
- [x] 一个 Story 对应一个 Agent、一个明确 diff、一个实施记录。
- [x] Story 开始即创建实施记录并由 Coordinator 将状态推进为 `claimed/in-progress`；完成后才可标记 `done`。
- [x] 同时最多两个 Agent，且声明路径和共享锁必须完全不相交。
- [x] 不执行 `reset/checkout/clean/stash/rebase` 等破坏性 Git 操作。
- [x] Worker 不 commit；每个 Wave 门禁通过后只由 Coordinator 创建一次审计过的 checkpoint commit；永远不 push。
- [x] 测试必须返回真实命令和真实输出；禁止把 stub/规划描述成完成。
- [x] 每个 Story 开始时创建并持续更新：`implementation/eNN-sNN.md`。
- [x] 只有集成 Agent 可以修改共享 wiring：`garden/main.go`、`garden/internal/server/server.go`、Console 顶级路由和 `sprint-status.yaml`。
- [x] `WORLD.MD`、`ACTMEM.MD` 不得自动进入 bootstrap、Fast/Deep Recall、ContextView、planner 或 Console 后台轮询。
- [x] 不得增加 JSON compatibility、alias、fallback 或 dual-write。

### Agent 交付格式

```text
Story: E##-S##
状态: done | blocked
实际修改文件:
测试命令与结果:
验收标准逐项结果:
发现的既有问题:
未完成/阻塞:
是否 commit: no
```

---

# Wave 0 — Epic 0：实施前门禁

详细定义：[Epic 0](epics/epic-00-baseline-and-gates.md)

| Story | 状态 | 建议负责人 | 允许范围 | 完成条件 |
|---|---|---|---|---|
| E00-S01 基线隔离 | ✅ 规划完成 | Baseline Agent | `docs/bmad/**` | 已建立 dirty-tree 分类和 Story diff 规则 |
| E00-S02 恢复测试基线 | ✅ 完成 | QA Agent | `garden/internal/server/report_handlers_test.go`，必要时测试 clock seam | 定向测试及 Garden/Laputa/Mentle 全量通过；不改生产报告语义 |
| E00-S03 冻结 API 契约 | ✅ 规划完成 | Contract Agent | `docs/bmad/**` | `api-contract.md` 已完成 |
| E00-S04 deletion/boundary scanner | ✅ 完成 | Guard Agent | 新 scanner 包、CLI、测试和实施记录 | report mode 可列出现状；enforce mode 有违规返回非零；Mentle boundary 覆盖 |
| E00-S05 测试矩阵 | ✅ 规划完成 | Test Architect | `docs/bmad/**` | 91 个用例已归属到 Story |
| Gate 0-A 接受 ADR-0014 | ✅ 已接受 | Owner | `docs/architecture/0014-*` | ADR-0014 已按现稿接受 |
| Gate 0-B Readiness 重跑 | ✅ 完成 | Integrator | `implementation-readiness.md`、`sprint-status.yaml` | C-02 关闭、scanner/report 与 dirty-tree adoption 完成，Verdict=`PASS` |

## Wave 0 推荐并行派工

```text
Agent QA    → E00-S02
Agent Guard → E00-S04
```

两项完成后只派一个 Integrator 执行 Gate 0-B。**Gate 0 未 PASS，不得开始 Wave 1。**

---

# Wave 1A — Epic 1：Persona Authority

详细定义：[Epic 1](epics/epic-01-persona-authority.md)

## 推荐依赖顺序

```text
E01-S01 ─► E01-S02 ─► E01-S03
                    └► E01-S04 ─► E01-S05
所有完成 ─────────────────────► E01-S06
```

| Story | 状态 | 建议负责人 | 主修改范围 | 核心验收 |
|---|---|---|---|---|
| E01-S01 DIVA text/CAS/no-op parity | ✅ 完成 | Persona Core A | `laputa/persona/text.go`、`service.go`、单测 | base=0 不绕过；相同内容不增 revision；并发 CAS |
| E01-S02 immutable history/failure atomicity | ✅ 完成 | Persona Core B | `laputa/persona/service.go`、history tests | snapshot create-new；失败不破坏现状；metadata/stale 正确 |
| E01-S03 五文件原子初始化与 repair | ✅ 完成 | Persona Storage | `laputa/persona/**` | 同盘 staging；无半初始化；显式 repair；不导入 JSON |
| E01-S04 Persona write-class | ✅ 完成 | Persona Policy | Persona policy/service tests | user/agent/operator 分离；USER/WORLD/DREAM/DARK 规则与 DIVA 一致 |
| E01-S05 review lifecycle | ✅ 完成 | Persona Review | Persona review service/tests | ready/allowlist/cap；stale 持久化；accept/reject 语义 |
| E01-S06 Persona domain adapter tests | ✅ 完成 | Persona Integrator | `garden/internal/server/persona_*`，最终由集成 Agent改路由 | HTTP 契约、principal、错误码、初始化/repair/review 全覆盖 |

### Persona Agent 禁止事项

- 不得自行发明 Garden 特有 Persona 语义；与 DIVA 冲突时 DIVA 胜出。
- 不得使用 `X-Garden-Actor` 授权。
- 不得保留 `/files`、`/requests` 作为兼容别名。
- E01-S01～S05 不直接修改 Garden composition root。

---

# Wave 1B — Epic 3：Mentle Recovery

可与 Epic 1 并行。详细定义：[Epic 3](epics/epic-03-mentle-recovery.md)

## 推荐依赖顺序

```text
E03-S01 ─► E03-S02 ─► E03-S03 ─► E03-S07
                     E03-S04 ─► E03-S05 ─► E03-S06 ─┘
E03-S08 可在 S02 后独立推进
E03-S09 在 S02/S03/S05 完成后收尾
```

| Story | 状态 | 建议负责人 | 主修改范围 | 核心验收 |
|---|---|---|---|---|
| E03-S01 SQL 原子 CAS | ✅ 完成 | Mentle Canonical A | `mentle/facade/canonical.go`、并发测试 | `WHERE version=?` + RowsAffected；一成功一冲突 |
| E03-S02 所有 mutation 汇入 Facade | ✅ 完成 | Mentle Adapter | `mentle/cmd/server`、miner/import 调用点 | REST/MCP/miner 均落 canonical + index job；无直接 Searcher mutation |
| E03-S03 recoverable index outbox | ✅ 完成 | Mentle Recovery A | catalog schema、job worker、fault tests | state/attempt/error/lease/backoff；poison job 隔离；启动不阻塞 |
| E03-S04 embedding identity | ✅ 完成 | Mentle Embedding | embedder/facade identity tests | dimension/metric/normalize/model/provider 不匹配显式失败 |
| E03-S05 staged rebuild/repair | ✅ 完成 | Mentle Recovery B | repair/reindex 新实现与 fault tests | canonical 永不移动；staging 验证；失败回滚旧索引 |
| E03-S06 tombstone/BM25 correctness | ✅ 完成 | Mentle Retrieval | hybrid/BM25/rebuild tests | 只重建 active current；无 50k 静默遗漏；旧版本不挤占候选 |
| E03-S07 live IndexHealth | ✅ 完成 | Mentle Observability | facade health + Garden adapter契约 | 实时 count/identity/jobs/tombstone/rebuild reason |
| E03-S08 session ingest 幂等 | ✅ 完成 | Mentle Ingest | `internal/miner`、cursor/lease tests | 稳定 ID；重复导入无重复；崩溃续传 |
| E03-S09 退役业务 WAL authority | ✅ 完成 | Mentle Cleanup | WAL 调用点、repair docs/tests | JSONL 不重建 canonical；无双权威声明；资源正确关闭 |

### Mentle Agent 禁止事项

- 不实现 Redis、Qdrant、Chroma、LanceDB。
- 不从 JSONL WAL 重建 canonical SQLite。
- repair 不得 rename/delete/overwrite canonical DB。
- Mentle 不得依赖 Persona、WORLD 或 ACTMEM authority。

---

# Wave 2 — Epic 2：ACTMEM + Frozen Core + Clean Break

前置：Epic 1 完成；Epic 0 scanner 可用。详细定义：[Epic 2](epics/epic-02-actmem-and-clean-break.md)

## 推荐依赖顺序

```text
E02-S01 ─► E02-S02 ─► E02-S03
E02-S04 ────────────────────┐
E02-S05 ────────────────────┼► E02-S06 ─► E02-S07 ─► E02-S08
E02-S03 ────────────────────┘
```

| Story | 状态 | 建议负责人 | 主修改范围 | 核心验收 |
|---|---|---|---|---|
| E02-S01 ACTMEM model/store | ✅ 完成 | ACTMEM Core | 新 `laputa/actmem/**` | DIVA path/frontmatter/CAS/no-op/caps/atomic write |
| E02-S02 maintenance + capsule | ✅ 完成 | ACTMEM Lifecycle | `laputa/actmem/**` | edit/complete/drop/fold；capsule CRUD；路径安全 |
| E02-S03 explicit read/query adapters | ✅ 完成 | ACTMEM Adapter | ACTMEM REST/MCP 服务和测试 | bounded 1200；维护工具独立；无自动 promotion |
| E02-S04 restart-safe Frozen Core | ✅ 完成 | Garden Context | 新 `internal/personactx/**`、SQLite repo | 六文件、session immutable、进程重启不漂移 |
| E02-S05 checkpoint/report 脱离 Governance | ✅ 完成 | Garden Runtime | activity/report repositories | SQLite 持久化；不写 Laputa sections |
| E02-S06 recall/bootstrap 原子切换 | ✅ 完成 | Garden Recall Integrator | recall/context/bootstrap/trace | Frozen Core + Mentle evidence；WORLD/ACTMEM 从类型和文本中消失 |
| E02-S07 删除旧 runtime composition | ✅ 完成 | Clean-break Integrator | `main.go`、server wiring、旧 governance/cognitive runtime | 删除旧 routes/wiring/tests；scanner enforce=0 |
| E02-S08 clean-break e2e | ✅ 完成 | E2E Agent | `garden/e2e/**` | Persona、ACTMEM restart、Frozen Core、显式 WORLD、忽略旧 sections |

### 共享文件锁

E02-S06 和 E02-S07 都会修改 `main.go/server.go/recall`，**禁止并行**。S07 必须由 S06 的集成 Agent或明确交接后的单一 Agent 执行。

---

# Wave 3 — Epic 4：Adapters、MCP、Console、Release

前置：Epic 2、Epic 3 完成。详细定义：[Epic 4](epics/epic-04-adapters-console-release.md)

## 推荐依赖顺序

```text
E04-S01 ─► E04-S02 ─► E04-S03 ─► E04-S04
                  └► E04-S05 ─► E04-S06
E04-S07 可在 S02 后并行
全部完成 ─────────────────────► E04-S08
```

| Story | 状态 | 建议负责人 | 主修改范围 | 核心验收 |
|---|---|---|---|---|
| E04-S01 principal capability auth | ✅ 完成 | Security Agent | auth middleware/config/tests | constant-time token；user/agent/operator；actor header 不提权 |
| E04-S02 REST clean-break cutover | ✅ 完成 | API Integrator | server routes/DTO/contract tests | 新契约唯一；旧 files/requests/governance/world routes 404 |
| E04-S03 Garden MCP domain alignment | ✅ 完成 | MCP Agent | `garden/cmd/garden-mcp/**` | agent 权限合法；Persona protected write→review；ACTMEM/health 正确 |
| E04-S04 修正 memory_search | ✅ 完成 | MCP Search Agent | MCP search + tests | 结果真正匹配 query；recent 单独标注；degraded/empty 分离 |
| E04-S05 Persona/Memory workspace | ✅ 完成 | Console Feature Agent | Persona/Memory modules | 初始化、编辑、review、history；ACTMEM/evidence/activity 分离 |
| E04-S06 替换旧 Console IA | ✅ 完成 | Console Integrator | App/Sidebar/Governance Map/types/tests | 四工作区；删除 Governance Map/compat/WORLD polling |
| E04-S07 EvoMap bounded candidate | ✅ 完成 | Evolution Agent | `internal/evolution/**` | bounded ACTMEM/evidence refs；不能写 Persona 或安装 artifact |
| E04-S08 最终验收 | ✅ 完成 | Release Integrator | 测试/运行手册/状态文档 | 全套测试、e2e、Console、scanner、diff check 全绿 |

---

# 执行收口

首批派工模板已由本次 GOAL 执行消耗，不再作为新的派工指令。每个 Story 的证据见 [`implementation/`](implementation/)；运行、恢复和四个 checkpoint 提交规则见 [GOAL Execution Runbook](GOAL-EXECUTION-RUNBOOK.md)。

# 当前状态摘要

```text
已完成：E00-S01..S05、E01-S01..S06、E02-S01..S08、E03-S01..S09、E04-S01..S08
Owner 决策：ADR-0014 已接受；GOAL 已激活并完成
Readiness：PASS
Feature implementation：完成；宿主适配器按范围 Deferred
Commit：四个 Wave checkpoint；Push：禁止
```
