# Garden MemoryOS Console 前端预设计

**状态：** 预设计，未实现  
**日期：** 2026-08-14  
**依据：** ADR-0012、ADR-0013、`oil-frontend` 规则  
**目标：** 为新的 Laputa Markdown clean break 设计实际可落地的 Console 信息架构、界面空间、状态和代码归属。

---

## 1. 设计结论

Garden Console 从“治理图谱 + 工程遥测面板”改为 **四个核心工作区**：

1. **Persona**：查看、编辑、审阅七份 Markdown 权威文件和历史。
2. **Memory**：显式访问 `ACTMEM.MD`，并浏览 Mentle 材料、证据和 Garden activity。
3. **Evolution**：操作 EvoMap 的 candidate、proposal、artifact、mailbox 和 Hub 生命周期。
4. **Chat Approval**：审批危险的运行时操作，不处理 Persona 内容审阅，也不处理 ACTMEM 维护。

全局仍保留一个轻量 **Overview** 作为入口，不再承担治理决策。`Operations` 降为 Overview 或 Settings 下的运行状态抽屉/详情，不作为第五个认知工作区。

核心原则：

- 页面围绕用户任务和业务对象组织，不围绕 Go package 或旧 API 组织。
- 列表负责识别、比较、选择；详情负责完整内容、证据、历史和编辑。
- Persona、ACTMEM、Mentle evidence、EvoMap artifact、Chat Approval 是五类不同对象，不能用一个通用“治理/审批”模型承载。
- `WORLD.MD` 和 `ACTMEM.MD` 不自动进入首页、Recall、ContextView 或后台轮询；用户明确打开或查询时才读取。
- 没有真实后端能力的动作不显示为可用按钮；accepted-design/deferred 只用于架构库或明确的建设状态，不伪装成 live。

## 2. 用户任务与业务对象

| 工作区 | 用户主要任务 | 核心对象 | 权威数据源 | 主动作 |
| --- | --- | --- | --- | --- |
| Overview | 判断系统现在是否可用、哪里需要处理 | runtime health、ingestion、spool、index health | `/v2/admin/*`、health、Mentle live probes | 打开一个需要处理的工作区 |
| Persona | 理解和修改人格权威，审阅 agent 提案 | Persona document、review、revision | `laputa/persona` + Persona HTTP API | 打开文档或批准/拒绝一个 Persona review |
| Memory | 查看活动记忆和材料证据 | ACTMEM document/query、Memory card、Evidence、Activity session | `laputa/actmem`、Mentle、Garden activity | 显式读取/查询/维护 ACTMEM，或打开证据 |
| Evolution | 判断能力候选并管理产物生命周期 | EvoMap run、candidate、proposal、artifact、mailbox item | Garden EvoMap stores + Hub | 批准/拒绝 proposal，推进 EvoMap 生命周期 |
| Chat Approval | 判断危险操作是否允许执行 | runtime operation request | Garden approval service | 批准/拒绝一次运行时操作 |

不再把 `GovernanceProjection`、section、WORLD projection、generic audit entry 或 `ACTMEM` 当作同一种业务对象。

## 3. 全局壳层

### 3.1 导航

左侧导航调整为：

- `Overview`
- `Persona`
- `Memory`
- `Evolution`
- `Chat Approval`
- `Reports`
- `Materials & Evidence`
- `Recall Trace`
- `Operations`
- `Architecture Library`
- `Settings`

`Mailbox` 不再作为独立认知工作区；它作为 Evolution 内的一个固定视图/过滤器保留。`Ambitions & Suggestions` 作为 Reports 下的人类模块入口保留。

旧 `Governance Map` 删除，不改名为 Persona。它的图谱、节点 inspector、WORLD 自动查询和旧 ADR 引用全部移除。

### 3.2 Top Bar

Top Bar 只承担全局范围和系统状态：

- 当前 host scope：`single-host · local`。
- live/degraded/offline 状态。
- 语言切换：`EN / 中文`。
- 一个显式的全局 runtime status 入口，打开 Operations 详情。

删除当前的主题循环按钮和“Preview”全局切换。主题应由 Settings 的稳定选择保存；Preview 不是产品运行态，不得占据运行中的全局主操作。

### 3.3 状态标签

状态词固定使用：

- `live`：来自真实运行 API。
- `accepted-design`：文档已接受但当前后端未提供。
- `deferred`：明确后置。
- `loading`、`refreshing`、`ready`、`empty`、`filtered-empty`、`queued`、`processing`、`error`：界面内部异步状态。

删除 `compat` 作为可见产品状态。旧兼容实现只存在于删除计划/代码审计里，不能继续作为 Console 的运营模式。

## 4. Overview 入口

### 用户任务
判断“现在是否需要处理某件事”，然后进入对应工作区。

### 首屏结构
使用紧凑的三列/四列响应式布局：

1. **Runtime**：Garden、Mentle、Laputa、EvoMap、Mailbox 的 live/degraded 状态。
2. **Recovery**：pending spool、失败 ingestion、index health；只有存在异常时提高视觉层级。
3. **Attention**：待审阅 Persona reviews、EvoMap proposals、Chat Approval requests 的数量和最近一项。
4. **Current activity**：最近 activity/session 状态，不展示完整 audit 日志。

每个面板只显示能让用户决定下一步的字段。点击面板进入对应对象列表或详情，不复制第二套数据状态。

### 删除

- `Cognitive health` 中的 WORLD conflicts、STM freshness 等旧概念。
- `Governance health` 中的 Frozen Core completeness、rules version、section pending review 的旧字段。
- `Recent governance activity` generic section/action/actor 表格。
- 旧 `/v2/admin/audit` 作为首页轮询数据源。

Persona review、Evolution proposal、Chat Approval 各自从自己的 API 获取；不要用一条 generic audit 代替三种待办。

### 状态

- 首次加载：保留最终四区结构的局部 skeleton。
- 定时刷新：保留当前 snapshot，在对应面板显示 `refreshing`。
- 局部失败：只影响对应面板，保留其他 live 面板。
- 空状态：说明没有待处理对象，并提供一个进入对应工作区的动作；不在 Header 和空状态重复添加创建按钮。

## 5. Persona 工作区

### 用户任务
查看当前人格权威、修改允许直接修改的文件、审阅 agent 对受保护文件的提案、检查历史。

### 页面结构

```text
Persona
┌──────────────────────────────────────────────────────────────┐
│ 页面标题 + 当前 profile + setup status                       │
├──────────────┬───────────────────────────────┬───────────────┤
│ 文档列表     │ 当前文档详情 / Markdown 编辑器  │ Revision/history│
│ 7 documents  │ body + limits + source status  │ 或 Review queue │
└──────────────┴───────────────────────────────┴───────────────┘
```

宽屏为三栏，主区最大；窄屏按“列表 → 详情 → 历史/审阅”顺序折叠。页面只有一个主任务：查看或处理当前选中的 Persona document/review。

### 左栏：Document list

每项只显示：

- 文件名和人类可读名称：`IDENTITY.MD`、`RELATIONSHIP.MD`、`REDLINE.MD`、`USER.MD`、`DREAM.MD`、`DARK.MD`、`WORLD.MD`。
- 当前状态：`ready`、`missing`、`over limit`、`review pending`。
- 字符数/上限，例如 `612 / 800`。
- 当前 revision。

不显示内部 path、JSON schema、section 编号、原始 UUID 或 `_meta`。

### 中栏：Document detail/editor

默认是阅读状态，不直接进入编辑。显示：

- Markdown 正文，使用稳定的阅读宽度。
- 当前 revision 和更新时间。
- 字符限制和是否进入 Frozen Core。
- `WORLD.MD` 明确标记“tool-only”；不显示“加入上下文”或“投影”按钮。
- 对 `USER.MD` 区分“user preference”和“agent observation”，但不把两者拆成新的文件或 section。

编辑动作是显式的 `Edit document`。编辑器只提交完整 Markdown body，不提供字段 patch、JSON editor、路径选择器或 schema 表单。

### 右栏：History / Review

通过 tabs 切换：

- `History`：revision 列表、完整 snapshot、before/after textual diff。
- `Review queue`：只显示 Persona review 对象；每一项包含 document、proposal summary、actor、base revision、diff。

Agent 对 `IDENTITY.MD`、`RELATIONSHIP.MD`、`REDLINE.MD`、`WORLD.MD`、`USER.MD` preference 的修改只能在 Review queue 批准后落盘。

`DREAM.MD`、`DARK.MD`、`USER.MD` observation 的直接写入不进入 Chat Approval，也不显示为 generic governance mutation。

### 写入状态和失败边界

- 保存前：本地编辑草稿只属于当前文档。
- 保存中：只锁定当前文档的 Save 按钮和编辑器，不锁定整个页面。
- 成功：刷新当前 revision/history，再退出编辑态。
- 失败：保留正文、光标位置和当前文档，显示就近错误。
- stale revision：不覆盖新内容，提示刷新并显示差异，用户明确决定是否重新编辑。

### Initialization

当五个必需文件全部缺失时，显示一个明确的 `Initialize Persona` 主动作。初始化只创建 `IDENTITY.MD`、`RELATIONSHIP.MD`、`REDLINE.MD`、`USER.MD`、`WORLD.MD`，不创建 `DREAM.MD` 或 `DARK.MD`。

部分文件已存在时，不显示自动修复或迁移按钮，显示完整性错误和“打开文件/联系管理员”的恢复路径。

## 6. Memory 工作区

### 用户任务
显式查看跨会话活动记忆，或从 Mentle 材料中找到可验证证据。

### 页面结构

```text
Memory
┌───────────────────────────────┬──────────────────────────────┐
│ ACTMEM                         │ Mentle materials / evidence  │
│ explicit tool read/query       │ card list → evidence detail  │
├───────────────────────────────┴──────────────────────────────┤
│ Activity sessions / recovery / selected session detail         │
└──────────────────────────────────────────────────────────────┘
```

Memory 页面必须把 ACTMEM 和 Mentle 放在两个有明确标题和来源标签的区域，不把它们拼成一个“长期记忆”列表。

### ACTMEM 区域

默认显示轻量摘要和最后 revision，不自动加载全文。用户点击 `Read ACTMEM` 或提交查询后才请求 `/v2/actmem` / `/v2/actmem/query`。

可见动作：

- `Read ACTMEM`
- `Query ACTMEM`
- `Edit ACTMEM`（只有后端真实提供写入时显示）

不显示：自动投影、Promote to Mentle、Persona review、Chat Approval wrapper。

### Mentle 区域

继续使用 cards → bounded evidence 的两步结构：

- 列表只显示 title、kind、scope、validity/status、revision 等识别和判断字段。
- 详情显示证据片段、source URI、content hash、validity 和 provenance。
- 不在列表暴露全文，不把 evidence 自动写入 Persona 或 ACTMEM。

### Activity 区域

保留 session、event、transient spool 和 recovery 状态。Checkpoint 属于 Garden runtime，显示为运行状态，不显示成 memory document。

## 7. Evolution 工作区

### 用户任务
判断 EvoMap 能力候选，推进 proposal/artifact 生命周期，处理 mailbox 和 Hub 策略。

### 页面结构

```text
Evolution
┌──────────────────────────────────────────────────────────────┐
│ Hub status + lifecycle counts                                │
├───────────────┬──────────────────────────────┬───────────────┤
│ Runs/Candidates│ Proposal detail + evidence   │ Artifact state │
├───────────────┴──────────────────────────────┴───────────────┤
│ tabs: Proposals · Artifacts · Mailbox · Hub                   │
└──────────────────────────────────────────────────────────────┘
```

EvoMap 是能力产物工作区，不是 Persona 审阅工作区。保留现有 runs、candidates、proposals、leakage report、mailbox、Hub status 数据源，但改变承载结构：列表负责挑选对象，详情负责证据、泄漏状态和批准动作。

### Proposal detail

首屏显示：

- candidate name/kind/description。
- proposal status：`pending`、`approved`、`rejected`、`installed`、`published`。
- evidence refs 和 trace ref 的可追溯入口。
- leakage report 和 policy decision。
- 主动作：当前状态下唯一最重要的动作，例如 `Approve proposal`。

`Install`、`Publish`、`Hub send` 等低频动作进入详情的操作区，并遵循 EvoMap 的审批和隐私门。不得把“批准 Persona 修改”混入这里。

### 长任务状态

Run 和 provider 状态使用：

`queued → processing → ready | error`

列表只显示短状态和开始时间；详情显示错误原因、候选和恢复动作。单个 run 的轮询只更新该 run，不阻塞其它列表。

### Mailbox

Mailbox 作为 Evolution 内 tab 保留 inbox/outbox/dead-letter。Approve/reject 请求保持选中对象、理由和位置；提交失败不关闭详情，不清空理由。

## 8. Chat Approval 工作区

### 用户任务
决定一次危险运行时操作是否允许执行。

### 页面结构

```text
Chat Approval
┌───────────────────────────────┬──────────────────────────────┐
│ Pending operation requests     │ Selected request detail       │
│ identity + risk + state        │ command/tool/scope/evidence   │
│                               │ approve / reject              │
└───────────────────────────────┴──────────────────────────────┘
```

列表只显示：请求名称、发起 host、风险级别、scope、状态、创建时间。详情显示完整影响范围、工具、参数摘要、证据和执行结果。

危险审批采用共享三段式 drawer：Header 固定对象身份，Body 唯一滚动，Footer 固定 Approve/Reject。提交期间保持 drawer 和对象不变，只锁定对应按钮；失败时保留当前对象和输入。

Chat Approval 不显示 Persona diff，不显示 ACTMEM 内容，不负责 EvoMap artifact approval。

## 9. Reports、Materials、Trace、Operations

### Reports
保留现有 `/v2/reports`、`latest`、`generate`、`orientation` 和 human modules。页面用于阅读/生成报告，不向 Persona、ACTMEM 或 EvoMap 自动写入。

报告列表是时间/周期表格；详情显示完整报告、source refs、window 和 generator。生成是明确动作，后台状态使用 `queued/processing/ready/error`，失败时保留当前筛选和详情。

### Materials & Evidence
作为 Memory 的可深链页面保留，数据来源为 Mentle。Cards 列表与 Evidence 详情分离，避免在列表塞进完整证据。

### Recall Trace
作为排障详情页保留。显示 trace、step、budget、source refs、degraded/error。删除 `governance`、`world`、`ACTMEM` 作为自动 ContextView 来源的字段与文案。

### Operations
只显示 live runtime health、ingestion、spool、pipeline、index health 和独立 runtime audit。删除 generic governance audit 的 section/action/actor 表格。Persona history、Evolution events、Chat Approval events 由各自工作区展示。

## 10. 共享组件与代码归属

### 共享层

保留或建立这些稳定基础组件：

- `PageHeader`
- `StatusDot` / `StatusBadge`
- `WorkspaceShell` / `WorkspaceSplit`
- `DocumentList`
- `MarkdownViewer` / `MarkdownEditor`
- `RevisionTimeline` / `DiffViewer`
- `ReviewQueue`
- `EvidenceList` / `EvidenceDetail`
- `OperationList` / `ApprovalDrawer`
- `LoadingSkeleton` / `EmptyState` / `ErrorState`

共享组件负责自身 hover、focus、loading、disabled、滚动和稳定尺寸。页面只负责组合和数据连接。

### 业务模块归属

```text
src/modules/persona/
  api.ts types.ts hooks.ts components/ pages/
src/modules/memory/
  api.ts types.ts hooks.ts components/ pages/
src/modules/evolution/
  api.ts types.ts hooks.ts components/ pages/
src/modules/chat-approval/
  api.ts types.ts hooks.ts components/ pages/
src/modules/operations/
  api.ts types.ts hooks.ts components/ pages/
src/components/       shared primitives and shell
src/styles/            tokens and shared layout only
```

当前 `src/pages` 可作为迁移过渡，但新业务状态、请求和类型不能继续全部散落在页面文件中。`useApi` 需要补充真实的 `refreshing`、请求竞态、局部 error 保留旧 snapshot 和单项 busy 语义；不能通过每个页面独立补丁解决。

### 旧实现迁移

| 现有实现 | 处理 |
| --- | --- |
| `GovernanceMap.tsx` | 删除，不改名复用 |
| `data/governance.ts` | 删除旧节点、旧 ADR 和 WORLD projection 文案；稳定的系统架构文档移入 Architecture Library 数据 |
| `Inspector.tsx` | 不作为治理通用 inspector 复用；拆为业务详情组件或保留纯 drawer 基础能力 |
| `authority_handlers` API 类型 | 删除 generic projection/mutation/audit 类型 |
| `Operations.tsx` | 保留 runtime 面；移除 governance audit |
| `EvolutionPage.tsx` | 保留 EvoMap 数据和生命周期，按列表/详情重组 |
| `MailboxPage.tsx` | 保留能力 mailbox，移入 Evolution workspace |
| `TopBar` theme switcher | 移入 Settings；Top Bar 只保留 scope、health、language |
| `SafetyBanner` compat 文案 | 删除 compat；只显示 local-only、degraded、spool 等 live 状态 |
| `Overview.tsx` | 重组为 action-oriented health/attention entry，不再显示旧 cognitive/governance cards |

## 11. 视觉和空间系统

沿用现有 Garden 暗色工作台的基础 token，但收紧语义：

- cyan：Garden/runtime/active navigation。
- emerald：Laputa Persona authority。
- violet：Mentle materials/evidence。
- amber：EvoMap/report/external lifecycle。
- rose：危险审批、隐私/泄漏、错误；不用于普通分类。

减少当前“每个 panel 都有玻璃、渐变和阴影”的装饰。页面分区使用对齐、留白和轻分隔线；只有状态、选择、告警使用强调边或高亮。禁止在普通列表项上使用不同颜色的 accent border。

空间规则：

- Shell 只保留一个主纵向滚动区。
- Persona 三栏和 Evolution 三栏由 workspace 自己管理边界；只有详情内容和历史列表在确有独立浏览任务时内部滚动。
- Drawer 使用 Header/Body/Footer 三段式，Body 是唯一滚动区。
- 列表和详情不同时维护第二套选中对象；路由 query 或模块 hook 是当前选择的单一来源。
- 窄屏从双栏/三栏降为单列，主对象和主动作先出现。
- 长名称、diff、Markdown 正文和错误文本必须换行，不得撑破布局。

## 12. 状态矩阵

| 区域 | 首次 loading | 已有数据刷新 | 空 | 错误 | 长任务 |
| --- | --- | --- | --- | --- | --- |
| Persona documents | 文档列表 skeleton | 保留旧列表 + refreshing | setup required / no optional docs | 就近 retry | save/review processing |
| Persona detail | 正文区域 skeleton | 保留旧正文 | document missing | 当前文档错误 | save button loading |
| Review queue | 列表 skeleton | 保留旧 queue | no pending review | queue retry | approve/reject button loading |
| ACTMEM | 摘要 skeleton | 保留摘要 | no ACTMEM file | explicit read retry | query/maintenance processing |
| Mentle cards | 列表 skeleton | 保留已提交查询快照 | empty / filtered-empty | list retry | 单项 evidence loading |
| EvoMap run | run list skeleton | 保留旧列表 | no runs | list retry | queued/processing on run |
| Proposal | detail skeleton | 保留旧 proposal | candidate missing | detail retry | approve/install/publish scoped to item |
| Chat Approval | pending list skeleton | 保留旧列表 | no pending request | list retry | drawer action loading |
| Operations | health skeleton | panel-level refreshing | no spool/pipelines | panel retry | ingestion/pipeline object status |

## 13. 迁移顺序

1. 先删除旧导航概念和真实不再存在的 `compat`/WORLD 自动轮询文案；不先做视觉包装。
2. 建立 `modules/persona` 的类型、API hook、文档列表和只读详情。
3. 建立 History/Review queue 和 Markdown 编辑提交边界。
4. 建立 `modules/memory`，先接 ACTMEM explicit read/query，再接 Mentle cards/evidence/activity。
5. 把 Evolution/Mailbox 合并到 EvoMap workspace，保留其 API 和能力边界。
6. 建立 Chat Approval 列表/详情/共享 ApprovalDrawer。
7. 重组 Overview 与 Operations，删除 generic audit/cognitive/governance 旧数据流。
8. 最后删除 `GovernanceMap`、旧 `data/governance.ts`、无效类型、页面样式和 route fallback。

每一步都必须同时完成数据流、状态、文案、类型、组件和旧使用位置迁移；不保留长期双实现。

## 14. 验收

### 信息架构

- 首屏能直接进入 Persona、Memory、Evolution、Chat Approval 的待办对象。
- 旧 Governance Map、section、compat、WORLD projection 不再作为产品概念出现。
- EvoMap mailbox 与 Persona review 在 UI 上明确分离。

### 数据真实性

- Persona 文档、ACTMEM、Mentle evidence、EvoMap proposal、Chat Approval 各自只来自一个权威 API。
- 无真实写入能力的页面隐藏写按钮。
- 查询、列表、总数、分页属于同一已提交 snapshot。
- 失败保留对象、详情、输入和位置。

### 空间和状态

- 页面只有一个主要纵向滚动区；内部滚动有独立任务边界。
- loading 不与 empty/error 并存；刷新不清空已有内容。
- 单项操作不锁定全页。
- Drawer Header/Body/Footer 边界稳定，提交失败不提前关闭。

### 架构一致性

- `WORLD.MD`、`ACTMEM.MD` 不自动出现在任何 ContextView/Recall 结果。
- Persona review 不进入 Chat Approval 或 EvoMap proposal。
- EvoMap 是唯一能力 artifact 生命周期入口。
- Console 不读取 `.laputa/sections`，不依赖 generic governance DTO。

## 15. 明确不做

- 不把现有 Governance Map 换个名字继续保留。
- 不把 Persona、ACTMEM、Mentle、EvoMap 合成一个“Memory/治理”大列表。
- 不在 UI 中显示 JSON editor、section patch、内部 schema 或存储路径。
- 不用 mock 数据伪造 Persona/EvoMap/Approval 已经实现。
- 不在本轮直接改代码；本文件只完成整体前端预设计。
