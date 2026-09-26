# Laputa External Agent SDK — 子代理 TODOLIST

**项目：** Laputa External Agent SDK (`laputa-agent/1`)  
**方法：** BMAD project-sized  
**当前状态：** `execution / Wave 0 gate in progress / CONCERNS`  
**实施授权：** **是，仅 Wave 0**  
**用途：** 大湿/Coordinator 手动派工；E01–E04 仍须等待 Wave 0 关闭。  
**更新：** 2026-09-04

> 大湿已于 2026-09-04 接受 ADR-0015 并授权 Wave 0。当前可执行 E00-S01..S04：证据基线、契约冻结、conformance 设计与 readiness gate。E01–E04 的源码、测试、fixture、配置、生成物、提交、push 仍被 Gate 0 阻塞。

---

## 0. 外部 Agent 强制规则

每个子代理在接受任何 Story 前，必须确认并在首条回报中逐项声明：

1. 已读 [PRD](prd.md)、[Architecture Spine](ARCHITECTURE-SPINE.md)、[API Contract](api-contract.md)、[Test Inventory](test-inventory.md)、[Readiness](implementation-readiness.md)、对应 Epic 和 [ADR-0015](../../architecture/0015-laputa-external-agent-contract.md)。
2. 已知 authority：Laputa=Persona/ACTMEM；Mentle canonical SQLite=唯一 Memory authority；Garden=runtime/adapters；EvoMap=capability artifacts。
3. 已知 automatic lane 只有 Frozen Core + bounded requested Mentle evidence；WORLD、ACTMEM、history、raw source/path 永远 explicit-only。
4. 已知 REST=canonical、MCP=thin adapter、SDK=typed contract/client+lifecycle helper；不得另造第二条语义。
5. 已知 AGENT-VIVY 是第一方 conformance host，已有 Session/Run/Journal/Policy/Tool；不得重造 Agent runtime，不得把 `vivy.rpc.v1`/Face/ChannelHost/plugin ABI 当 Laputa SDK。
6. 已记录开始时两仓库的 `git status --porcelain=v1 -uall`、HEAD、分支；不得重置、清理、暂存、覆盖或接管既有 dirty diff。
7. 已声明本 Story 的允许路径、禁止路径和依赖；共享文件只能由 Coordinator 指定的单一 Integrator 修改。

### Gate 0 期间禁止跳关

- 禁止修改 `garden/`、`laputa/`、`mentle/`、`agent-vivy/` 下任何源文件、测试、fixture、配置、脚本、生成物。
- 禁止创建 `implementation/` 完成记录、把 Story 标 `done`、把 YAML 的 `implementation_authorized` 改为 `true`。
- 禁止 commit、push、创建分支/worktree、运行会改变运行时状态的命令。
- 子代理只能返回发现与 proposed documentation diff；Coordinator 是 `sprint-status.yaml` 的唯一写者。

### 实施授权后的持续规则（预先冻结）

- 每 Story 一个 Agent、一个窄 diff、一个实施记录；Worker 不 commit、不 push。
- 最多 2 个并行 Agent，且路径完全不交叉；`server.go`、MCP registration、SDK contract source、Vivy runtime composition、shared conformance fixtures 均为串行集成锁。
- 所有外部写入均走 Domain Service：Mentle `facade.Service`、Persona policy/review、ACTMEM service；禁止 raw SQLite/Searcher/vector/file writer。
- 不得加 alias、fallback、dual read/write、第二 authority、自动 Persona 生成、自动 WORLD/ACTMEM 注入。
- 可降级的是 timeout/5xx/429 等 data-plane availability；认证/授权/校验/CAS conflict 必须 fail closed 且不得重试。

### 子代理交付格式

```text
Story: E##-S##
Mode: planning-only | implementation-authorized
Status: proposed | blocked | done (only when authorized)
Evidence: path:line / command output / official URL
Allowed-path compliance:
Actual modified files:
Acceptance criteria:
Tests or document validation:
Risks / conflicts / unresolved decisions:
Commit: no
```

---

## 1. 敏捷排期与依赖

| Wave | Epic | 目标 | 依赖 | 可并行 | 当前状态 |
|---|---|---|---|---|---|
| 0 | E00 | Evidence、ADR、contract、conformance gate | Owner review | E00-S01 + research-only E00-S03 | `backlog` |
| 1 | E01 | Canonical REST discovery/bind/bootstrap/capture | Wave 0 | S01/S02 serial；S03/S04可在S02后分支 | `blocked` |
| 2A | E02 | Typed SDK + Vivy lifecycle bridge | E01 | SDK client lane 与 Vivy adapter design lane，最终串行集成 | `blocked` |
| 2B | E03 | MCP equivalence + explicit tools | E01 | 可与 E02 并行；shared contract fixture locked | `blocked` |
| 3 | E04 | Fault/restart/security/release | E02 + E03 | security audit/sample can parallel; final gate serial | `blocked` |

**建议迭代节奏（实施授权后）：** 每 Wave 一个集成 checkpoint，不按日历虚报工期。Wave 0 是文档/门禁；Wave 1 建立 canonical truth；Wave 2A/2B 并行但由同一 contract fixture 收口；Wave 3 只做完整性与 release 证据。

---

# Wave 0 — Epic 0：Contract and Gates

详细定义：[Epic 0](epics/epic-00-contract-and-gates.md)

| Story | 当前可否派发 | 推荐角色 | 当前允许范围 | 后续实施范围 | 完成条件 |
|---|---|---|---|---|---|
| E00-S01 Live-surface evidence ledger | 可，文档研究 | Contract Analyst | `docs/bmad/laputa-external-agent-sdk-2026-09/**` | 无 | Garden/Vivy/示例工程 path:line 证据和 dirty baseline 完整 |
| E00-S02 ADR/public boundary freeze | 可，文档建议 | Architect | 本包 + `docs/architecture/0015-*` | 无 | Owner 接受/修订 ADR；无 placeholder budget/ambiguous capability |
| E00-S03 Conformance harness design | 可，文档设计 | Test Architect | 本包 `test-inventory.md` | future test directories only after authorization | REST/SDK/MCP case matrix 和 negative sentinels 完整 |
| E00-S04 Readiness gate | 可，文档核验 | BMAD Coordinator | 本包 | 无 | 链接/YAML/Story IDs 验证、owner decision recorded |

**Wave 0 gate：** 大湿接受 ADR-0015、明确“开始实施”、两仓库的 worktree/branch ownership 已决定后，方可进入 Wave 1。

---

# Wave 1 — Epic 1：Canonical REST Access Plane

详细定义：[Epic 1](epics/epic-01-canonical-access-plane.md)

```text
E01-S01 Discovery ─┐
E01-S02 Binding ───┼─► E01-S03 Bootstrap/Search/Expand ─┐
                   └─► E01-S04 Capture/Receipt ─────────┼─► E01-S05 contract suite
                                                         │
```

| Story | 推荐角色 | 实施主范围（授权后） | 禁止事项 | 核心验收 |
|---|---|---|---|---|
| E01-S01 | Garden Contract Agent | `garden/internal/server/**`、contract fixtures | 不广告未实现能力/泄露路径或 token | secret-free manifest、version negotiation |
| E01-S02 | Garden Security Agent | binding/auth middleware + contract tests | Actor header 授权、第二 Session/Run | profile/principal/session immutable binding |
| E01-S03 | Garden Recall Agent | recall/domain/DTO + tests | WORLD/ACTMEM/history 自动 lane | six-slot bootstrap、query-scoped Search→Expand |
| E01-S04 | Garden Capture Agent | capture domain/receipt + tests | raw storage writes、capture 反向失败 host Run | 202/receipt/idempotency/conflict/index_pending |
| E01-S05 | Garden Integrator | conformance fixtures/routes | 将 stub 当 live | REST black-box/restart/negative tests |

**共享锁：** `garden/internal/server/server.go`、canonical contract fixtures、error envelope；仅 Integrator 在前置 Story 交接后修改。

---

# Wave 2A — Epic 2：Typed SDK and AGENT-VIVY Lifecycle

详细定义：[Epic 2](epics/epic-02-sdk-and-vivy-lifecycle.md)

| Story | 推荐角色 | 实施主范围（授权后） | 禁止事项 | 核心验收 |
|---|---|---|---|---|
| E02-S01 | SDK Core Agent | future standalone `laputa-sdk-go/**` | imports Vivy/Garden internal/Eino/SQLite/MCP internal | public framework-neutral DTO/client boundary |
| E02-S02 | SDK Reliability Agent | SDK client/tests | retry 4xx/CAS/forbidden；吞掉 degraded | typed errors, deadline/retry/receipt handling |
| E02-S03 | Vivy Runtime Agent | `agent-vivy/internal/memory/**` + composition seam | `sdk/plugin/**`、Vivy RPC/Face/ChannelHost | MemoryPort pre-model bootstrap, audit event |
| E02-S04 | Vivy Reliability Agent | Vivy capture queue/hook/tests | fake/append terminal event、修改 terminal Run outcome | durable terminal capture/restart semantics |
| E02-S05 | Vivy Integrator | Vivy e2e/conformance only | MCP used as hidden lifecycle path | real run lifecycle conformance |

**跨仓库规则：** Garden 与 Vivy 必须各有独立 worktree；SDK module location/owner 在 E00-S02 冻结前不得创建。Vivy 中 `studio/` submodule 不属于本项目修改范围。

---

# Wave 2B — Epic 3：MCP Conformance and Explicit Tools

详细定义：[Epic 3](epics/epic-03-mcp-conformance-and-tools.md)

| Story | 推荐角色 | 实施主范围（授权后） | 禁止事项 | 核心验收 |
|---|---|---|---|---|
| E03-S01 | MCP Adapter Agent | `garden/cmd/garden-mcp/**` | 第二检索/写入/auth 实现 | canonical mapping + declared capability/bounds |
| E03-S02 | MCP Protocol Agent | MCP error/result DTO/tests | text-only error | code/request/trace/retryable/reason equivalence |
| E03-S03 | Tool Policy Agent | explicit MCP tools/policy tests | default exposed ACTMEM maintenance/repair/review decision | explicit only + Agent capability matrix |
| E03-S04 | MCP Integrator | shared conformance fixture | route-specific semantic drift | REST/MCP success/error equivalence |

**锁：** `garden/cmd/garden-mcp/main.go`、shared fixture/error envelope；不可与 E01-S05 同时修改。

---

# Wave 3 — Epic 4：Reliability and Release

详细定义：[Epic 4](epics/epic-04-reliability-and-release.md)

| Story | 推荐角色 | 实施主范围（授权后） | 核心验收 |
|---|---|---|---|
| E04-S01 | Fault QA Agent | focused fault/restart test scope | timeout/429/5xx/restart/replay fail semantics |
| E04-S02 | Security Auditor | public API/import/boundary scan | no raw authority/storage/internal leakage |
| E04-S03 | Compatibility QA Agent | second minimal black-box client sample | proves SDK is not Vivy-coupled |
| E04-S04 | Release Integrator | docs/verification/status + final cross-repo gate | supported status only on full evidence/owner acceptance |

**Final gate：** ALL conformance rows, affected Garden/Vivy suites, manifest truthfulness, structured REST/MCP equivalence, import scan, boundary scan and `git diff --check` pass. Worker never push；Coordinator only commits after explicit owner review.

---

## 2. 参考工程借鉴边界

| 参考工程 | 允许借鉴 | 明确禁止照搬 |
|---|---|---|
| TencentDB-Agent-Memory | Host adapter/RuntimeContext 分层；recall/capture/tools/degrade；pending writes/retry/observability | v2/v3 双语义；静态 header session；自动 L3 Persona 直写；大而全管理面 |
| memsearch | source-first；content hash；incremental/derived index；Search→Expand→Source | Persona/profile 全目录扫描；path 当授权；Markdown direct write；Milvus authority；Unix hook 作为 Windows 标准 |
| AGENT-VIVY | Session/Run/Journal/RunHook/Tool/Config lifecycle seams | export internal、SDK plugin ABI、vivy.rpc.v1 或 MCP outbound 充当 Laputa protocol |

## 3. Coordinator checklist

- [ ] Parent verifies child evidence before any document merge/status change.
- [ ] Parent alone modifies `sprint-status.yaml`.
- [ ] Parent rechecks actual Garden/Vivy git status immediately before implementation dispatch.
- [ ] Parent refuses any child that attempts source/test/config edits during this planning phase.
- [ ] Parent requests owner authorization before Wave 1.
