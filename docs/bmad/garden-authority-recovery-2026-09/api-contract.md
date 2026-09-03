# Garden clean-break API 契约（Story 0.3）

**状态：** Frozen / 后续 REST、MCP、Console 实现的共同契约
**冻结日期：** 2026-09-03
**契约版本：** `garden-clean-break/1`
**依据：** ADR-0012、ADR-0013、PRD、Architecture Spine，以及 DIVA `agent-diva-laputa` 的 Persona/ACTMEM 可观察语义

本文冻结目标模型，不描述当前实现。当前代码中的 `/v2/persona/files`、`/v2/persona/requests`、通用 Governance DTO、WORLD claim projection 和不完整的 IndexHealth DTO 都不是兼容依据。

## 1. 不变量与线格式

### 1.1 权威边界

- Persona 只有七种闭集 kind：`identity | relationship | redline | user | dream | dark | world`，正文只能是完整 Markdown 字符串。
- Persona 正文上限（可见 grapheme）：Identity 800、Relationship 600、Redline 400、User 800、Dream 40、Dark 300、World 1000。规范化为 CRLF/CR→LF、Unicode NFC、裁掉首尾空白；超限失败，不截断。
- `WORLD.MD` 与 `ACTMEM.MD` 只在显式 tool lane 中读取；不得出现在 bootstrap、Fast/Deep Recall、planner、trace content、Frozen Core 或自动 Console polling 中。
- ACTMEM 位于 `<profile>/actmem/ACTMEM.MD`，独立于 `persona/`、Persona review/history 和 Mentle。Pulse、Recap 各 1600 字符，Work 1600 字符；稳定读取最多 1200 字符；capsule 最多 800 字符；单条 Pulse 280、Recap 200 字符。
- Memory 的唯一权威是 Mentle canonical SQLite；vector/BM25 是可重建派生索引。canonical 提交后的索引失败不得把 canonical 成功改报为失败。
- 请求 JSON 一律 `Content-Type: application/json`、拒绝未知字段。时间是 UTC RFC 3339；revision/version 是 JSON 整数；hash 是 `sha256:<lowercase-hex>`。

### 1.2 请求头、principal 与审计

- 写端点必须使用 `Authorization: Bearer <local-capability-token>`。token 经常量时间比较后映射到 `user`、`agent` 或 `operator` principal；MCP 固定使用 agent-scoped token。
- `read` 是只读 principal：可由 read-scoped token取得；本批次中仅 loopback 的只读端点可在无 token 时映射为 `read`。写端点永不匿名。
- `X-Garden-Actor` 仅写入审计/历史的显示标签，**不参与授权、不改变 principal、不决定 write class**。缺省审计标签由服务端依据已验证 token subject 生成。
- `X-Garden-Request-ID` 可由客户端提供；缺省由服务端生成，并在响应头与错误 envelope 的 `request_id` 中返回。token 不得写日志。
- 下表中 `✓` 表示该 principal 可调用；`—` 表示返回 `403 principal_forbidden`。资源级/作用域级限制仍可返回领域错误。

## 2. 通用响应与稳定错误

成功响应直接使用各节 DTO，不再套通用 `data` envelope。所有错误使用：

```json
{
  "code": "persona_revision_conflict",
  "message": "base revision does not match current revision",
  "retryable": false,
  "request_id": "req_...",
  "details": {
    "expected_revision": 3,
    "current_revision": 4
  }
}
```

`code` 是程序分支依据；`message` 不是。`details` 始终为对象，可增加字段但不得改变既有字段语义。

### 2.1 通用错误码

| HTTP | code | 条件 | retryable |
|---:|---|---|---|
| 400 | `invalid_request` | query/path/body 组合非法、未知字段或不满足端点约束 | false |
| 400 | `malformed_json` | JSON 无法解析 | false |
| 401 | `authentication_required` | 写请求缺少 capability token | false |
| 401 | `invalid_capability_token` | token 不存在、失效或不匹配 | false |
| 403 | `principal_forbidden` | token 有效，但 principal 不允许该操作 | false |
| 413 | `payload_too_large` | HTTP body 超出端点上限 | false |
| 429 | `busy` | 锁/队列暂时繁忙 | true |
| 500 | `internal_error` | 未分类内部错误；不得泄漏路径/token | true |
| 503 | `service_unavailable` | 对应领域服务未配置或实时 probe 无法执行 | true |
| 504 | `timeout` | 请求 deadline 超时 | true |

## 3. Persona DTO

### 3.1 枚举与基础 DTO

```ts
type PersonaKind = "identity" | "relationship" | "redline" | "user" | "dream" | "dark" | "world";
type PersonaStatus = "uninitialized" | "ready" | "incomplete";
type ReviewState = "pending" | "accepted" | "rejected" | "stale";
type ReviewActor = "agent" | "autodream"; // 来自已验证 agent token claim，不来自 X-Garden-Actor/body

type PersonaDocumentState = {
  kind: PersonaKind;
  file_name: "IDENTITY.MD" | "RELATIONSHIP.MD" | "REDLINE.MD" | "USER.MD" | "DREAM.MD" | "DARK.MD" | "WORLD.MD";
  exists: boolean;
  valid: boolean;
  reason: string | null;
  revision: number;
  updated_at: string | null;
  pending_count: number;
  content_limit: number;
  frozen_limit: number | null;
  required: boolean;
  tool_only: boolean;
};

type PersonaDocument = {
  kind: PersonaKind;
  file_name: string;
  exists: boolean;
  valid: boolean;
  content: string;
  revision: number;
  content_hash: string;
  updated_at: string | null;
  pending_count: number;
};

type PersonaWriteResult = { changed: boolean; document: PersonaDocument };

type PersonaReview = {
  id: string;
  kind: PersonaKind;
  base_revision: number;
  base_hash: string;
  proposed_markdown: string;
  actor: ReviewActor;
  reason: string;
  created_at: string;
  state: ReviewState;
  decided_at: string | null;
};

type PersonaHistoryEntry = {
  revision: number;
  content_hash: string;
  actor: string;
  source: "user_direct" | "agent_p16" | "agent_p5_accepted" | "autodream_p5_accepted" | "init" | "history_resave";
  reason: string;
  base_revision: number;
  created_at: string;
};
```

历史列表不暴露服务器文件路径或 `snapshot`/`diff` 文件名；单 revision 响应在 entry 外增加 `content` 与 `unified_diff`。

### 3.2 Persona routes

| 方法与路径 | 请求 | 成功响应 | principal |
|---|---|---|---|
| `GET /v2/persona/documents` | 无；不得用 `include_content` | `200 {status, documents: PersonaDocumentState[]}`，固定七 kind 顺序；**仅元数据，不含正文** | read/user/agent/operator |
| `GET /v2/persona/documents/{kind}` | 无 | `200 PersonaDocument`；这是读取 WORLD 正文的唯一 Persona REST 路径，调用必须显式 | read/user/agent/operator |
| `PUT /v2/persona/documents/{kind}` | `PersonaDocumentWrite` | `200 PersonaWriteResult` | 见 write-class 矩阵 |
| `POST /v2/persona/initialize` | `PersonaInitialize` | `201 {status:"ready", documents: PersonaDocumentState[]}` | user |
| `POST /v2/persona/repair` | `PersonaRepair` | `200 {status, documents: PersonaDocumentState[]}` | user/operator |
| `GET /v2/persona/reviews?kind=&state=&limit=&cursor=` | filter 可省略；limit 1..200 | `200 {items: PersonaReview[], next_cursor:string|null}`，newest-first | read/user/agent/operator |
| `GET /v2/persona/reviews/{id}` | 无 | `200 PersonaReview` | read/user/agent/operator |
| `POST /v2/persona/reviews` | `PersonaReviewCreate` | `201 PersonaReview` | agent |
| `POST /v2/persona/reviews/{id}/approve` | `{"decision_reason"?:string}` | `200 PersonaReview`（state=`accepted`） | user |
| `POST /v2/persona/reviews/{id}/reject` | `{"decision_reason"?:string}` | `200 PersonaReview`（state=`rejected`） | user |
| `GET /v2/persona/history/{kind}?limit=&cursor=` | limit 1..200 | `200 {kind, items:PersonaHistoryEntry[], next_cursor:string|null}`，revision descending | read/user/agent/operator |
| `GET /v2/persona/history/{kind}/{revision}` | revision > 0 | `200 {kind, entry:PersonaHistoryEntry, content, unified_diff}` | read/user/agent/operator |

```ts
type PersonaDocumentWrite = {
  base_revision: number;          // 必填；现存文档上 0 绝不绕过 CAS
  content: string;                // 完整规范化后的目标文档；scope=observations 时为 Observations section body
  scope?: "document" | "observations"; // 默认 document；observations 仅 agent+user
  reason: string;
};

type PersonaInitialize = {
  identity: string;
  relationship: string;
  redline: string;
  user: string;   // 输入 Preferences body；服务端规范化为 ## Preferences
  world: string;
};

type PersonaRepair = {
  documents: Partial<Record<"identity"|"relationship"|"redline"|"user"|"world", string>>;
  reason: string;
};

type PersonaReviewCreate = {
  kind: PersonaKind;
  base_revision: number;
  base_hash: string;
  proposed_markdown: string; // 完整目标文档
  reason: string;
};
```

### 3.3 Persona write-class 矩阵

| 操作/目标 | read | user | agent | operator |
|---|:---:|:---:|:---:|:---:|
| documents list/read/history/review read | ✓ | ✓ | ✓ | ✓ |
| initialize required five | — | ✓ | — | — |
| direct PUT Identity/Relationship/Redline/WORLD | — | ✓ | — | — |
| direct PUT USER `scope=document`（Preferences 与 Observations） | — | ✓ | — | — |
| direct PUT DREAM/DARK `scope=document` | — | ✓ | ✓ | — |
| direct PUT USER `scope=observations`（必须保留 Preferences） | — | ✓ | ✓ | — |
| create protected Persona review | — | — | ✓ | — |
| approve/reject review | — | ✓ | — | — |
| explicit repair invalid required artifacts | — | ✓ | — | ✓ |

规则：

1. user direct write、agent P16 direct write和获批 review 都必须做 exact CAS。规范化内容与 current 相同返回 `changed=false`，不增加 revision/history。
2. agent 对 Identity/Relationship/Redline/USER Preferences/WORLD 只能 `POST /reviews`；不得由 `PUT` 隐式代建 review。
3. agent 的 USER direct write只替换 `## Observations`；Preferences 必须逐字语义保留。agent 的 USER review只可修改 Preferences、不可修改 Observations。
4. WORLD review必须保留用户保护内容，并通过 DIVA R6 bounded/reviewable claim entry gate；新增 claim 必须是 `## [domain] title` block 且含 `- status:`、`- source:`。WORLD 永不直接由 agent PUT。
5. 初始化仅在五个 required 文件全部不存在时执行，原子创建恰好五个文件及首 revision，不创建 DREAM/DARK。部分状态返回 `persona_incomplete`；不自动补全。
6. repair 只接收 owner 选择的、当前 invalid/missing 的 required kind；不得覆盖 valid 文档，不读/import `.laputa/sections`，operator 也不得绕过内容 cap、结构或历史规则。
7. review actor 从 agent capability 的可信 claim 得到（普通 MCP 为 `agent`，AutoDream 专用 token 为 `autodream`）；忽略 body/header 对该值的冒充。
8. review 创建前校验 profile ready、actor/kind、cap、CAS/hash、USER/WORLD scope 和每 kind 最多一个 pending。接受时再次 CAS；若已漂移，持久化为 `stale` 并写 `decided_at`，不覆盖新内容。direct write会将同 kind pending review 持久化为 stale。

### 3.4 Persona 稳定错误

| HTTP | code | 条件 |
|---:|---|---|
| 400 | `persona_kind_forbidden` | kind 不在闭集，或 scope/kind 对该 write class 非法（非 principal 级拒绝） |
| 400 | `persona_invalid_content` | 空 required 文档、UTF-8/Markdown 结构或 USER scope 非法 |
| 403 | `persona_world_protected_claim` | WORLD proposal 删除/改变用户保护内容 |
| 400 | `persona_world_entry_gate` | WORLD proposal 违反 R6 claim entry gate |
| 404 | `persona_review_not_found` | review id 不存在 |
| 404 | `persona_history_not_found` | kind/revision 历史不存在 |
| 409 | `persona_uninitialized` | 需要 ready profile 的操作发生于 uninitialized |
| 409 | `persona_incomplete` | profile 部分存在/不一致，需 repair |
| 409 | `persona_already_initialized` | initialize 发生于 ready profile |
| 409 | `persona_repair_not_required` | repair 发生于非 incomplete，或目标已 valid |
| 409 | `persona_revision_conflict` | `base_revision` 不等于 current（包括现存文档上的 0） |
| 409 | `persona_review_exists` | 同 kind 已有 pending review |
| 409 | `persona_review_stale` | review 非 pending，或接受时 base 已漂移；漂移必须落盘为 stale |
| 413 | `persona_cap_exceeded` | 正文超过 kind 可见字符 cap |
| 500 | `persona_storage_error` | 原子写、history/request persistence 或一致性 I/O 失败 |

## 4. ACTMEM DTO 与 routes

### 4.1 DTO

```ts
type ActmemView = {
  revision: number;
  updated_at: string;            // 缺失文件为 Unix epoch
  markdown: string;              // 最多 1200 字符
  truncated: boolean;
  sections: { pulse: string; recap: string; work: string }; // 与 markdown 共用 1200 总预算
};

type ActmemWriteResult = {
  changed: boolean;
  revision: number;
  updated_at: string;
};

type ActmemQueryHit = {
  section: "pulse" | "recap" | "work";
  work_section: "Goal" | "Open" | "Next" | "Constraints" | "Pointers" | null;
  line_index: number;
  excerpt: string;
};

type CapsuleSummary = { name:string; session_key:string; created_at:string; chars:number };
type CapsuleDocument = CapsuleSummary & { markdown:string };
```

ACTMEM 缺失不是 404：`GET /v2/actmem` 返回 revision 0 的规范空文档，且读取本身不创建目录/文件。

### 4.2 ACTMEM routes

| 方法与路径 | 请求 | 成功响应 | principal |
|---|---|---|---|
| `GET /v2/actmem` | `?max_chars=` 可省略，1..1200，默认 1200 | `200 ActmemView` | read/user/agent/operator |
| `POST /v2/actmem/query` | `ActmemQuery` | `200 {revision, items:ActmemQueryHit[], returned_chars, truncated}`；总返回正文 ≤ `max_chars` ≤1200 | read/user/agent/operator |
| `PUT /v2/actmem` | `ActmemPatch` | `200 ActmemWriteResult` | user/agent |
| `POST /v2/actmem/maintenance` | `ActmemMaintenance` tagged union | `200 {result:ActmemWriteResult, capsules?:CapsuleSummary[]}` | user/agent |
| `GET /v2/actmem/capsules` | 无 | `200 {items:CapsuleSummary[]}` newest-first | read/user/agent/operator |
| `GET /v2/actmem/capsules/{name}` | 无 | `200 CapsuleDocument`（≤800 字符） | read/user/agent/operator |
| `DELETE /v2/actmem/capsules/{name}` | 无 | `200 {name, deleted:true}` | user/agent |

```ts
type ActmemQuery = {
  query: string; // 必填，NFC+trim 后按 Unicode 大小写不敏感的 literal substring 匹配逻辑行；不是语义/向量检索
  sections?: ("pulse"|"recap"|"work")[];
  max_hits?: number;  // 1..50，默认 20
  max_chars?: number; // 1..1200，默认 1200
};

type ActmemPatch = {
  base_revision: number;
  pulse?: string;
  recap?: string;
  work?: string; // 必须只含五个注册的 ### subsection；缺项补为空 section
};

type ActmemMaintenance =
  | {operation:"append_pulse"; session_key:string; content:string}
  | {operation:"append_recap"; session_key:string; content:string}
  | {operation:"edit_work"; section:"Goal"|"Open"|"Next"|"Constraints"|"Pointers"; replacement:string; base_revision:number}
  | {operation:"complete_open_item"; item_index:number; base_revision:number}
  | {operation:"drop_item"; section:"Goal"|"Open"|"Next"|"Constraints"|"Pointers"; item_index:number; base_revision:number}
  | {operation:"fold_session"; session_key:string};
```

维护规则：PUT/maintenance 是直接 ACTMEM 写，不创建 Persona review、Chat Approval、Governance record、Mentle promotion 或 EvoMap proposal。PUT 与带 `base_revision` 的操作做 exact CAS；同内容 no-op 不增加 revision。空 append 是 no-op；非空 append 按 DIVA 单项 cap 截成带省略号的单项，并按 FIFO 保持 ring 总 cap。`fold_session` 只折叠指定 session marker，先生成 ≤800 字符 capsule，再从 Pulse/Recap 删除对应行；失败不得留下不可解释的半完成状态。

### 4.3 ACTMEM principal 矩阵

| 操作 | read | user | agent | operator |
|---|:---:|:---:|:---:|:---:|
| bounded read/query、capsule list/read | ✓ | ✓ | ✓ | ✓ |
| PUT section patch | — | ✓ | ✓ | — |
| append/edit/complete/drop/fold maintenance | — | ✓ | ✓ | — |
| capsule delete | — | ✓ | ✓ | — |

### 4.4 ACTMEM 稳定错误

| HTTP | code | 条件 |
|---:|---|---|
| 400 | `actmem_malformed` | head/front matter/Pulse-Recap-Work 结构非法 |
| 400 | `actmem_invalid_edit` | 未知 Work section、负数/越界 item index、非法 operation |
| 404 | `actmem_capsule_not_found` | capsule 不在服务端枚举的 projection 中（也覆盖 traversal/非法名称，不泄漏路径） |
| 409 | `actmem_revision_conflict` | exact CAS 失败 |
| 413 | `actmem_cap_exceeded` | Pulse/Recap/Work/capsule 总 cap 失败 |
| 500 | `actmem_storage_error` | ACTMEM/capsule 原子 I/O 失败 |

## 5. IndexHealth

### 5.1 route 与 DTO

| 方法与路径 | 请求 | 成功响应 | principal |
|---|---|---|---|
| `GET /v2/admin/index-health` | 无；不得接受客户端 startup label | `200 IndexHealth`，即使 degraded；实时 probe 完全不可执行时为 `503 index_health_unavailable` | read/user/agent/operator |

```ts
type EmbeddingIdentity = {
  model: string;
  dimension: number;
  metric: string;
  normalize: boolean;
  provider: string;
  version: string;
  captured_at: string;
};

type RebuildSummary = {
  status: "never" | "running" | "succeeded" | "failed";
  started_at: string | null;
  completed_at: string | null;
  canonical_snapshot_count: number | null;
  error_code: string | null;
};

type IndexHealth = {
  status: "ok" | "degraded" | "unavailable";
  observed_at: string;
  reasons: string[]; // 下列稳定 reason code，排序稳定
  canonical_active_count: number;
  vector_active_count: number;
  vector_physical_count: number;
  bm25_count: number;
  tombstone_count: number;
  tombstone_ratio: number;
  embedding_identity: EmbeddingIdentity | null;
  expected_embedding_identity: EmbeddingIdentity | null;
  pending_jobs: number;
  failed_jobs: number;
  oldest_pending_age_ms: number | null;
  last_rebuild: RebuildSummary;
};
```

稳定 `reasons[]`：`embedding_identity_missing`、`embedding_identity_mismatch`、`vector_count_diverged`、`bm25_count_diverged`、`tombstone_pressure`、`index_jobs_pending`、`index_jobs_failed`、`rebuild_running`、`last_rebuild_failed`、`canonical_probe_failed`、`vector_probe_failed`、`bm25_probe_failed`、`job_probe_failed`。存在任一 reason 至少为 degraded；权威/派生 probe 无法形成可信报告时为 unavailable。Garden `/health` 和 `/v2/admin/overview` 必须聚合同一个 live report，不得把 endpoint missing/stub/accepted-design 转成 ok。

IndexHealth 只有读权限，不授予 repair/rebuild。派生索引 rebuild 的命令/作业必须由 operator capability 触发，但其独立启动 API不在 Story 0.3 的冻结范围；后续若增加，必须是 `/v2/admin/*`、operator-only，且只能替换派生 artifact，不能修改 canonical SQLite。

| HTTP | code | 条件 |
|---:|---|---|
| 503 | `index_health_unavailable` | 无法完成最小 live probe；details 包含可公开的 reason codes |
| 500 | `index_health_internal_error` | probe aggregation 的未分类内部失败 |

## 6. Memory canonical mutation

现有资源名 `/v2/memories` 保留，但冻结为唯一 `facade.Service` mutation adapter；REST、MCP、miner/import/session adapter 都不得直接得到 Searcher/raw DB mutation capability。

### 6.1 DTO

```ts
type MemorySource = {
  type: "user" | "agent" | "session" | "import" | "report_projection";
  session_id?: string;
  event_id?: string;
  uri?: string;
  revision?: string;
};

type Memory = {
  id:string; kind:"fact"|"preference"|"decision"|"session_digest"|"note"|"source_artifact"|"semantic_unit";
  content:string; status:"active"|"deleted"; version:number; scope?:string;
  lifecycle?:string; collection?:string; tags:string[]; source:MemorySource;
  valid_from:string; valid_to:string|null; supersedes:string[]; superseded_by:string|null;
  created_at:string; updated_at:string; metadata:Record<string,unknown>;
};

type MemoryMutationResult = {
  memory: Memory;
  index_state: "applied" | "index_pending";
  index_job_id: string | null;
};

type MemoryDeleteResult = {
  id:string; deleted:true; status:"deleted"; version:number;
  index_state:"applied"|"index_pending"; index_job_id:string|null;
};
```

### 6.2 routes 与 principal

| 方法与路径 | 请求 | 成功响应 | read | user | agent | operator |
|---|---|---|:---:|:---:|:---:|:---:|
| `POST /v2/memories` | `CreateMemoryRequest`；可带 `Idempotency-Key` | `201 MemoryMutationResult` | — | ✓ | ✓ | — |
| `PATCH /v2/memories/{id}` | `UpdateMemoryRequest` | `200 MemoryMutationResult` | — | ✓ | ✓ | — |
| `DELETE /v2/memories/{id}` | `DeleteMemoryRequest` | `200 MemoryDeleteResult` | — | ✓ | ✓ | — |

```ts
type CreateMemoryRequest = {
  content:string; kind?:Memory["kind"]; scope?:string; tags?:string[];
  source?:MemorySource; metadata?:Record<string,unknown>; supersedes?:string[];
};
type UpdateMemoryRequest = {
  expected_version:number; // 必填，在 SQL WHERE version=? 的 commit boundary 校验
  content?:string; tags?:string[]; reason:string;
};
type DeleteMemoryRequest = {
  expected_version:number; // 必填；重复删除不伪装为成功
  reason:string;
};
```

- content 最大 64 KiB；更新至少含 content/tags 之一。
- canonical row 与 version-qualified `index_jobs` 在同一 SQLite transaction 提交。乐观并发必须在 SQL mutation predicate 中执行并检查恰好一行，不得只依赖 pre-read。
- canonical commit 后：索引已应用返回 `index_state=applied,index_job_id=null`；应用失败/延后仍返回 canonical 成功与 `index_state=index_pending,index_job_id=<id>`，IndexHealth degraded，由 outbox 重试。响应中的 `memory.version` 是已提交版本。
- embedding identity 在 canonical commit 前已知不兼容时拒绝 mutation；不得混写 vector。create idempotency key+相同 body 返回原结果；相同 key+不同 body 冲突。
- operator 只拥有显式修复/派生索引运维，不拥有业务 Memory 内容 mutation；需要内容变更时使用 user/agent capability。

### 6.3 Memory 稳定错误

| HTTP | code | 条件 |
|---:|---|---|
| 400 | `memory_invalid_content` | 空 content、非法 kind/source、无 mutable field 或格式非法 |
| 404 | `memory_not_found` | id 不存在或已 deleted |
| 409 | `memory_version_conflict` | expected_version 与 commit 时版本不符 |
| 409 | `memory_idempotency_conflict` | 同 Idempotency-Key 对应不同 body hash |
| 409 | `memory_embedding_identity_mismatch` | model/dimension/metric/normalize/provider/version 策略不兼容，需显式 rebuild |
| 413 | `memory_content_too_large` | content 超过 64 KiB |
| 503 | `memory_unavailable` | canonical Facade 不可用；不得 fallback 到 direct index/WAL |
| 500 | `memory_storage_error` | canonical transaction/storage 内部失败且未提交 |

## 7. 汇总 principal 矩阵

| 能力 | read | user | agent | operator |
|---|:---:|:---:|:---:|:---:|
| Persona 元数据/显式全文/review/history read | ✓ | ✓ | ✓ | ✓ |
| Persona 初始化 | — | ✓ | — | — |
| Persona user-direct write | — | ✓ | — | — |
| Persona P16 direct write（Dream/Dark/USER Observations） | — | ✓ | ✓ | — |
| Persona protected review create | — | — | ✓ | — |
| Persona review approve/reject | — | ✓ | — | — |
| Persona repair invalid required artifacts | — | ✓ | — | ✓ |
| ACTMEM bounded read/query/capsule read | ✓ | ✓ | ✓ | ✓ |
| ACTMEM direct maintenance/capsule delete | — | ✓ | ✓ | — |
| IndexHealth live read | ✓ | ✓ | ✓ | ✓ |
| Memory create/update/delete | — | ✓ | ✓ | — |
| 派生索引 rebuild/repair（API 本文未定义） | — | — | — | ✓ |

principal 不形成通用超集：operator 不是 Persona reviewer 或内容作者；user 不能冒充 agent proposal；agent 不能批准 review；read 永远不能写。

## 8. MCP 与 Console 映射约束

- MCP 使用 agent token：Persona protected change→`POST /v2/persona/reviews`；P16→对应 Persona PUT；ACTMEM→显式 read/query/maintenance；memory add/update/delete→上述 canonical mutation。MCP 不得调用 raw store 或通过 `X-Garden-Actor` 提权。
- Console 所有表格/编辑器使用本文 DTO。Persona document list 不含正文；用户选中 WORLD 后才 GET 单文档。Memory workspace 仅在用户显式动作后 GET/query ACTMEM。
- Console 不得把 404/503/stub/accepted-design 数据显示为 live/ok；IndexHealth 只使用 live endpoint。
- Chat Approval 不展示或决策 Persona review，ACTMEM maintenance 不生成 Chat Approval。

## 9. breaking removal（原子 cutover，无 alias）

目标路由启用的同一变更中必须删除以下 route family、DTO、handler、MCP mapping、Console caller、fixture 与文档化可用性：

| 删除 | 替代 | alias/fallback |
|---|---|---|
| `/v2/persona/files`、`/v2/persona/files/{kind}` 及其 `/history` 子路由 | `/v2/persona/documents[/{kind}]`、`/v2/persona/history/{kind}[/{revision}]` | **无** |
| `/v2/persona/requests`、`/{id}/accept`、`/{id}/reject` | `/v2/persona/reviews`、`/{id}/approve`、`/{id}/reject` | **无** |
| `/v2/persona/status` | `GET /v2/persona/documents` 的 `status`+元数据 | **无** |
| `/v2/governance/projection` | session Frozen Core + Persona 显式 read | **无** |
| `/v2/governance/mutations` | Persona/ACTMEM/Memory 各自 domain mutation | **无** |
| `/v2/governance/audit` 与作为 Governance alias 的 `/v2/admin/audit` | 各领域 history/review/operation telemetry；Chat Approval 保持独立 | **无** |
| `/v2/cognitive/world` | `GET /v2/persona/documents/world`（显式完整 Markdown read） | **无** |

另外删除 `WorldClaim`/`WorldResponse`、generic section/action/path/value/JSON Patch DTO、`AuditEntry.section` 等让 retired Governance 可继续使用的类型。不存在 redirect、HTTP 410 compatibility handler、双注册、双读/写、导入、legacy descriptor mapping 或 feature flag fallback；旧路径按普通未知 API 返回 404。历史 `.laputa/sections/*.json` 可留在磁盘作为 inert evidence，但新 runtime 不 probe 它们。

## 10. 实现验收引用

后续 Story 1.6、2.3、3.7、4.1–4.5 的 REST/MCP/Console contract test 必须直接引用本文：

1. 每个 route 有成功 DTO、principal denial、malformed/unknown field 和领域 stable code 测试。
2. WORLD/ACTMEM 不仅值为空，而是从 automatic DTO/type/rendered context 中结构性缺席。
3. 旧 `/files`、`/requests`、`/v2/governance/*`、`/v2/cognitive/world` 全部 404，且 repository scanner 无 target-runtime alias。
4. Memory index apply fault 返回 canonical success + `index_pending`；IndexHealth 同时 degraded。
5. capability token 决定 principal；伪造 `X-Garden-Actor` 不改变任何矩阵结果。
