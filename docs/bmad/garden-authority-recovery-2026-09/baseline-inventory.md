# Story 0.1 — 实施基线与 dirty-tree 所有权清单

**Lifecycle:** Historical pre-GOAL baseline snapshot. The final execution state supersedes the planning-only observations below; see `sprint-status.yaml`, `implementation-readiness.md`, and `implementation/` for audited completion evidence.

**记录时间：** 2026-09-03T04:38:06+08:00
**仓库：** `C:/Users/Administrator/Desktop/garden`
**分支：** `main`
**HEAD：** `c6cdceee0830036c258e50523874ea5b9693810e`
**上游：** `origin/main`，本地 ahead 8 / behind 0
**结论：** 采用“**不移动、不回滚、不自动提交，当前 dirty tree 显式 scoped**”策略。

## 1. 基线证据

本清单写入前执行：

```bash
git status --short
git status --porcelain=v2 --branch
git diff --stat
git diff --cached --stat
git ls-files --others --exclude-standard
git diff --name-status
git diff --cached --name-status
```

结果摘要：

- `git status --porcelain=v2 --branch`：`main`，HEAD 为上述提交，`branch.ab +8 -0`。
- 未暂存 tracked diff：55 个路径，`2163 insertions(+), 2538 deletions(-)`。
- 已暂存 diff：8 个纯重命名；其中 `TODOLIST.MD` 的目标文件另有未暂存修改（状态 `RM`）。
- untracked：97 个文件；`git status --porcelain=v1 -uall` 共 159 条状态记录。
- 写入本清单后 untracked 新增本文件；这属于 Story 0.1 自身 diff，不回写上述“写入前”计数。
- 验证期间检测到持续的并发工作树漂移：未暂存 tracked 增至 56 个路径，最终复核时 untracked 增至 104 个文件。新增/并发改写项详见 D 类“基线采集后的并发漂移”。该漂移不是 Story 0.1 所作或所有。
- `git diff --stat` 输出了 LF→CRLF 警告；本 Story 未触碰这些既有文件。

## 2. 所有权解释规则

以下分类是**隔离/归属声明**，不是对既有改动正确性、完成度或可提交性的验收：

1. 本 Story 仅声明 `baseline-inventory.md` 以及 Story 状态所需的 `sprint-status.yaml` 行为本 Story diff。
2. “此前 Persona slice”“此前 Mentle 文档修正”表示已知来源类别，不表示本 Story 接管、验证或批准这些改动。
3. 同一文件混有多个主题、或无法由路径与 diff 明确证明来源时，一律归入“其他未知/非本批改动”，不得推定所有权。
4. 下列所有既有 dirty 内容均保持原位；不 stash、不 reset、不 checkout、不 clean、不移动、不回滚、不自动提交。

## 3. 分类清单

### A. 本批既有规划文档（非 Story 0.1 新增）

这些文件在基线采集时均为 untracked，作为本批已存在的规划/架构合同输入保留：

- `docs/bmad/garden-authority-recovery-2026-09/ARCHITECTURE-SPINE.md`
- `docs/bmad/garden-authority-recovery-2026-09/README.md`
- `docs/bmad/garden-authority-recovery-2026-09/implementation-readiness.md`
- `docs/bmad/garden-authority-recovery-2026-09/prd.md`
- `docs/bmad/garden-authority-recovery-2026-09/sprint-status.yaml`
- `docs/bmad/garden-authority-recovery-2026-09/epics/epic-00-baseline-and-gates.md`
- `docs/bmad/garden-authority-recovery-2026-09/epics/epic-01-persona-authority.md`
- `docs/bmad/garden-authority-recovery-2026-09/epics/epic-02-actmem-and-clean-break.md`
- `docs/bmad/garden-authority-recovery-2026-09/epics/epic-03-mentle-recovery.md`
- `docs/bmad/garden-authority-recovery-2026-09/epics/epic-04-adapters-console-release.md`
- `docs/architecture/0012-laputa-markdown-clean-break.md`
- `docs/architecture/0013-laputa-clean-break-implementation-architecture.md`

### B. 此前 Persona slice（非本 Story 所有）

可由 Persona 路径或单主题 diff 明确识别的既有实现与测试：

**tracked、未暂存：**

- `garden/internal/server/server.go`
- `garden/main.go`
- `laputa/go.mod`
- `laputa/go.sum`

**untracked：**

- `garden/internal/server/persona_handlers.go`
- `garden/internal/server/persona_server_test.go`
- `garden/internal/server/persona_write_handlers.go`
- `garden/console/src/modules/persona/PersonaPage.tsx`
- `garden/console/src/modules/persona/api.ts`
- `garden/console/src/modules/persona/types.ts`
- `laputa/persona/service.go`
- `laputa/persona/service_test.go`
- `laputa/persona/text.go`
- `laputa/persona/types.go`
- `laputa/persona/write_test.go`
- `e2e-tmp/set_persona.py`
- `e2e-tmp/persona-e2e/persona/IDENTITY.MD`
- `e2e-tmp/persona-e2e/persona/REDLINE.MD`
- `e2e-tmp/persona-e2e/persona/RELATIONSHIP.MD`
- `e2e-tmp/persona-e2e/persona/USER.MD`
- `e2e-tmp/persona-e2e/persona/WORLD.MD`
- `e2e-tmp/persona-e2e/persona/history/{IDENTITY.MD,REDLINE.MD,RELATIONSHIP.MD,USER.MD,WORLD.MD}/{1.diff,1.md,2.diff,2.md,log.jsonl}`（25 个生成证据文件）

说明：`garden/console/src/App.tsx` 虽含 Persona 路由，但同时包含 Sessions/Settings 改动，故按保守规则列入 D 类；其他 Console 公共组件、样式和语言包同样不宣称为纯 Persona diff。`e2e-tmp/state/*` 为运行产物，也列入 D 类。

### C. 此前 Mentle 文档修正（非本 Story 所有）

以下 tracked、未暂存文档 diff 修正了目录结构、后端实现状态、存储格式或接口说明：

- `mentle/AGENTS.md`
- `mentle/README.md`
- `mentle/retrieval/AGENTS.md`
- `mentle/storage/AGENTS.md`
- `mentle/storage/chroma/AGENTS.md`
- `mentle/storage/govector/AGENTS.md`
- `mentle/storage/qdrant/AGENTS.md`
- `mentle/storage/vectorstore/AGENTS.md`
- `mentle/vector/AGENTS.md`

Mentle 源码与测试不属于“文档修正”，统一列入 D 类。

### D. 其他未知/非本批改动（不接管、不批准）

**已暂存重命名（8）：**

- `docs/architecture/0001-memoryos-vnext-architecture.md` → `docs/archive/2026-08-14-laputa-clean-break/0001-memoryos-vnext-architecture.md`
- `docs/architecture/0002-laputa-cognitive-partition-decision.md` → `docs/archive/2026-08-14-laputa-clean-break/0002-laputa-cognitive-partition-decision.md`
- `docs/architecture/0003-operations-console-design.md` → `docs/archive/2026-08-14-laputa-clean-break/0003-operations-console-design.md`
- `docs/architecture/0004-cognitive-files-migration.md` → `docs/archive/2026-08-14-laputa-clean-break/0004-cognitive-files-migration.md`
- `docs/architecture/0005-report-system-design.md` → `docs/archive/2026-08-14-laputa-clean-break/0005-report-system-design.md`
- `docs/architecture/0008-legacy-compatibility-removal.md` → `docs/archive/2026-08-14-laputa-clean-break/0008-legacy-compatibility-removal.md`
- `TODOLIST.MD` → `docs/archive/2026-08-14-laputa-clean-break/status/TODOLIST-pre-clean-break.MD`（目标另有未暂存修改）
- `docs/architecture/stm-raw-first-proposal.html` → `docs/archive/2026-08-14-laputa-clean-break/stm-raw-first-proposal.html`

**tracked、未暂存（排除 B/C 后）：**

- 根与文档：`AGENTS.md`、`README.md`、`docs/README.md`、`docs/architecture/{0006-semantic-ingestion-and-obsidian-adapter.md,0007-evomap-mailbox.md,0009-ambition-user-suggestions-modules.md,0010-evomap-hub-transport-provider.md,AGENTS.md}`、`docs/archive/AGENTS.md`、`docs/archive/2026-08-14-laputa-clean-break/status/TODOLIST-pre-clean-break.MD`、`docs/dev/AGENTS.md`。
- Garden：`garden/AGENTS.md`、`garden/go.mod`、`garden/go.sum`、`garden/internal/AGENTS.md`；Console 的 `App.tsx`、`api/types.ts`、`components/{Sidebar.tsx,TopBar.tsx}`、`i18n/{en.json,zh.json}`、`pages/{GovernanceMap.tsx,Overview.tsx}`、`styles/{components.css,global.css,pages.css,tokens.css}`、`tsconfig.tsbuildinfo`。
- Laputa：`laputa/.laputa/AGENTS.md`、`laputa/.laputa/sections/AGENTS.md`、`laputa/AGENTS.md`、`laputa/README.md`、`laputa/governance/AGENTS.md`。
- Mentle 源码/测试：`mentle/cmd/server/main.go`、`mentle/facade/canonical.go`、`mentle/internal/embedder/hugot.go`、`mentle/internal/kg/{kg_test.go,knowledge_graph.go}`、`mentle/internal/room/yaml_loader_test.go`、`mentle/pkg/wal/{wal.go,wal_test.go}`、`mentle/storage/vectorstore/vectorstore.go`。

**untracked（排除 A/B 后）：**

- 工具/代理状态：`.agents/AGENTS.md`；`.omc/` 下 25 个文件（配置、日志/研究/技能说明、session 状态、checkpoint、trace 与 project-memory）。
- 根/文档：`TODOLIST.MD`、`docs/architecture/0011-recoverable-indexing-and-evidence-contract.md`、`docs/archive/2026-08-14-laputa-clean-break/README.md`、`docs/design/{garden-memoryos-console-frontend-pre-design.md,garden-openclaw-memory-provider.md}`、`docs/dev/reference/2026-08-04-mempalace-python-gap-audit.md`。
- e2e 非 Persona 归属或运行产物：`e2e-tmp/mcp_stdio_test.py`、`e2e-tmp/oc-schema.json`、`e2e-tmp/state/{garden.db,garden.db-shm,garden.db-wal}`。
- Garden：`garden/cmd/garden-mcp/main.go`、`garden/console/src/pages/{SessionsPage.tsx,SettingsPage.tsx}`。
- Mentle：`mentle/facade/embedding_identity_test.go`、`mentle/internal/embedder/identity.go`、`mentle/pkg/wal/checkpoint.go`。

**基线采集后的并发漂移与处置：**

- `garden/internal/server/report_handlers_test.go` 曾被并发 Story 0.2 修改；因 owner 明确要求本批暂不动代码，该 hunk 已撤销。
- `garden/cmd/architecture-guard/main.go`、`garden/internal/architectureguard/scanner.go`、`scanner_test.go` 曾由并发 Story 0.4 新增；同样因越过 planning-only 边界而删除，Story 0.4 退回 backlog。
- 纯规划产物 `api-contract.md`、`test-inventories.md`、`baseline-inventory.md` 保留；它们不改变运行代码。
- `.omc/`、Persona、Mentle及其他既有 dirty 内容仍不接管、不批准、不回滚。
- 后续 Story 必须以自己的起始快照识别并发漂移；未获 owner 明确授权不得进入 implementation phase。

## 4. 后续 Story 的强制 diff 边界

每个后续 Story 必须遵守以下流程，并且**只能声明自己的 diff**：

1. 开始前记录当时的 `git status --porcelain=v1 -uall`、`git diff --stat`、`git diff --cached --stat` 与 HEAD。
2. 在 Story 记录中预先声明允许修改的路径；不得把本清单 A–D 类既有改动笼统计入自己的成果。
3. 完成后给出“该 Story 实际修改路径”及逐文件 diff；对于开始前已 dirty 的文件，必须只声明可审计的新增 hunk，不能声明整文件所有权。
4. 如发现并发或来源不明的新 diff，立即归入未知并停下确认；不得 reset、checkout、clean、stash、覆盖或顺手修复。
5. Worker 不执行 `git add` 或 commit。Owner 已授权按 Wave checkpoint 提交；只允许 Coordinator 在对应 Wave 门禁通过后按 GOAL Runbook 显式暂存和提交，永远不 push。
6. 验证报告必须区分：既有失败、该 Story 新增失败、以及与该 Story 无关的工作树噪声。

## 5. Story 0.1 自身所有权

本 Story 的允许且实际写入范围仅为：

- `docs/bmad/garden-authority-recovery-2026-09/baseline-inventory.md`（新增）
- `docs/bmad/garden-authority-recovery-2026-09/sprint-status.yaml`（仅更新 C-01/Story 0.1 状态和 planning-only 下一动作；不得开启实施授权）

除上述两项外，本 Story 未修改源码、未移动或回滚文件、未暂存、未提交。

## 6. GOAL 启动前决策补充（2026-09-03）

Owner 已确认：

- 执行范围为 Waves 0–3 全量；
- ADR-0014 按现稿接受；
- 当前 dirty/untracked 成果在 GOAL 激活后经逐项审计，accepted 内容纳入 Wave 0 checkpoint；
- 每个 Wave 只创建一个审计后的 checkpoint commit，不 push；
- 并发上限为两个路径与共享锁完全不相交的 Worker；
- 未明确说 `运行 GOAL` 前，`implementation_authorized=false`。

本清单仍保留 Story 0.1 的历史采集事实。GOAL 激活时由 Coordinator 追加新的基线指纹和 accepted/preserved/generated/blocked 分类，不覆盖本节之前的证据。详细采用和漂移规则见 `GOAL-EXECUTION-RUNBOOK.md`。

## 7. GOAL 激活复核（2026-09-03）

Owner 已明确授权执行 Waves 0–3，Coordinator 按 runbook 完成启动复核：

- **accepted：** E00-S02 的月报时钟 seam 与确定性测试修复；E00-S04 的架构边界 scanner/CLI 及其测试；本轮新增的两个 Story 实施记录；Wave 0 门禁状态更新。
- **preserved：** 激活前已存在的 Persona slice、Mentle 文档/源码改动、Console 改动、架构文档归档与其他未知 dirty 内容；它们继续保留，但不因本复核自动获得本轮 Story 所有权。
- **generated：** `e2e-tmp/**` 下的 Go cache、数据库、Persona fixture、日志和测试产物；不得进入任何 checkpoint commit。
- **blocked/deferred：** scanner 在 clean-break 前报告 112 个旧运行时边界违规；这是 Epic 2 删除故事的预期门禁状态，不阻塞 Wave 0 或 Wave 1/3 开始。
- **safety:** 未执行 reset、checkout、clean、stash、rebase、广泛删除、移动或 push；所有后续提交仍由 Coordinator 使用显式路径/代码块完成。
