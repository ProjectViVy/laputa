# ADR-0004 配套：Garden ↔ Hermes HTTP API Contract

> **Status**: accepted contract — Contract Freeze 0（2026-07-15）  
> **Version**: `garden-hermes/1`  
> **Date**: 2026-07-15  
> **Garden owner**: Garden 施工方  
> **Hermes owner**: Hermes plugin 施工方  
> **关联**: [ADR-0004 提案](./0004-garden-memory-orchestration-proposal.md) · [讨论记录](./0004-garden-memory-orchestration-discussion.md) · [Hermes 现状](./0004-hermes-context-for-codex.md)

---

## 0. 给松本的一页摘要

双方并行施工，只通过本文档定义的 HTTP 契约协作：

- Garden 负责服务端 API、Mentle facade、Pipeline/Skill、持久化、治理和测试。
- Hermes 负责新 plugin、declared native tools、session lifecycle hook 和 basic context 注入。
- Hermes 不直接调用 Mentle，不传 wing、room、top_k、model、pipeline 或 capability。
- 现阶段不加鉴权；Garden 默认且必须监听 `127.0.0.1:7373`，不能继续使用可能绑定所有网卡的 `:7373`。
- v1 不提供不可逆 purge；DELETE 只做 soft delete。
- session ingest 是 lifecycle HTTP hook，不是模型工具；`session_id`、`event_id` 必填，precompact/session-end 必须幂等。
- 高级查询只有一个模型工具：`garden_search(query)`，plugin 内部映射为 advanced resolve。
- 不启动 subagent。Garden 高级能力在单进程 pipeline 中执行，可调用受控 LLM HTTP。

---

## 1. 网络与信任模型

### 1.1 Base URL

```text
http://127.0.0.1:7373
```

Garden 默认监听地址必须从当前 `:7373` 改为 `127.0.0.1:7373`。`GARDEN_ADDR` 仍可显式覆盖；无鉴权版本若配置为非 loopback，启动时必须输出高风险警告。

### 1.2 鉴权决策

v1 **不使用** bearer token、mTLS 或 Unix socket。这是本地可信进程模型下的阶段性决策，不代表后续版本永久无鉴权。

约束：

- 只接受 loopback 部署作为受支持配置。
- 不实现远程部署承诺。
- `X-Garden-Actor` 等 header 仅用于审计，不能作为安全凭据。
- v1 禁止通过 HTTP 执行不可逆 purge。
- 请求体有大小限制，所有 handler 有超时。

### 1.3 通用请求 Header

| Header | 必填 | 说明 |
|---|---:|---|
| `Content-Type: application/json` | 有 body 时是 | UTF-8 JSON |
| `X-Garden-Request-ID` | 否 | plugin 可生成；缺失时 Garden 生成 |
| `X-Garden-Actor` | 否 | `hermes_agent`、`user_request`、`lifecycle`、`report_system`、`admin`；仅审计标签 |
| `X-Garden-Session-ID` | 否 | 与当前 Hermes session 关联 |
| `Idempotency-Key` | mutation 推荐 | 重试时复用；session endpoint 另有必填 `event_id` |

### 1.4 通用响应 Header

```text
Content-Type: application/json
X-Garden-Request-ID: req_...
```

---

## 2. 通用错误格式

为兼容 Garden 现有 `{"error":"..."}` 客户端，`error` 保持字符串，新增机器可读字段：

```json
{
  "error": "memory content is required",
  "code": "invalid_request",
  "retryable": false,
  "request_id": "req_01...",
  "details": {}
}
```

### 2.1 错误码

| HTTP | code | retryable | 场景 |
|---:|---|---:|---|
| 400 | `invalid_request` | false | JSON/schema/字段错误 |
| 404 | `memory_not_found` | false | ID 不存在或已不可见 |
| 409 | `version_conflict` | false | `expected_version` 不匹配 |
| 409 | `idempotency_conflict` | false | 同 key 对应不同 body |
| 413 | `payload_too_large` | false | 超过请求大小限制 |
| 422 | `mutation_rejected` | false | governance 拒绝 mutation |
| 429 | `busy` | true | worker/并发上限 |
| 500 | `internal_error` | true | 未分类服务端错误 |
| 503 | `mentle_unavailable` | true | Mentle 初始化/存储不可用 |
| 503 | `pipeline_unavailable` | true | 目标 pipeline 不可运行 |
| 504 | `timeout` | true | pipeline 或外部 LLM 超时 |

Hermes plugin 只对 `retryable=true` 的错误做有界重试；不得无限重试。

---

## 3. 核心数据类型

### 3.1 Memory

```json
{
  "id": "mem_01J...",
  "kind": "fact",
  "content": "Laputa 最终采用 A 架构。",
  "status": "active",
  "version": 1,
  "scope": "project:garden",
  "tags": ["laputa", "architecture"],
  "source": {
    "type": "user",
    "session_id": "sess_01...",
    "event_id": "evt_01..."
  },
  "valid_from": "2026-07-15T10:00:00Z",
  "valid_to": null,
  "supersedes": ["mem_old"],
  "superseded_by": null,
  "created_at": "2026-07-15T10:00:00Z",
  "updated_at": "2026-07-15T10:00:00Z",
  "metadata": {}
}
```

枚举：

- `kind`: `fact`、`preference`、`decision`、`session_digest`、`note`
- `status`: `active`、`superseded`、`deleted`、`pending_review`
- `source.type`: `user`、`agent`、`session`、`import`、`report_projection`

wing、room、embedding、BM25 channel、内部 KG ID 不属于公共 Memory schema。

### 3.2 ContextPackage

保持当前结构并增加实际执行模式：

```json
{
  "trace_id": "run_01...",
  "mode": "advanced",
  "context": "... [ev_001]",
  "evidence": [
    {
      "id": "ev_001",
      "source": "memory",
      "locator": "mem_01J...",
      "excerpt": "...",
      "score": 0.82
    }
  ],
  "confidence": 0.75,
  "degraded": false,
  "warnings": []
}
```

---

## 4. Memory CRUD

### 4.1 Create

```http
POST /v1/memories
```

Hermes native tool 最小请求：

```json
{
  "content": "Laputa 最终采用 A 架构。"
}
```

完整可选请求：

```json
{
  "content": "Laputa 最终采用 A 架构。",
  "kind": "decision",
  "scope": "project:garden",
  "tags": ["laputa"],
  "source": {"type":"user"},
  "metadata": {}
}
```

响应：`201 Created`，返回完整 Memory。

规则：

- `content` trim 后不能为空，最大 64 KiB。
- ID 由 Garden 生成；Hermes 不生成 `memory:` key。
- Create 可以触发冲突 proposal，但 v1 默认不做不可逆删除。
- 提供 `Idempotency-Key` 时，同 key + 同 body 返回第一次结果；同 key + 不同 body 返回 409。

### 4.2 Get

```http
GET /v1/memories/{id}
```

响应：`200 OK`，返回完整 Memory。默认可以读取 `superseded`，但 `deleted` 返回 404；管理端以后可增加显式 include_deleted，不进入 Hermes 工具。

### 4.3 Update

```http
PATCH /v1/memories/{id}
```

Hermes native tool 请求：

```json
{
  "content": "Laputa 最终采用 A2 架构。"
}
```

完整请求：

```json
{
  "content": "Laputa 最终采用 A2 架构。",
  "reason": "用户纠正",
  "expected_version": 1,
  "tags": ["laputa", "architecture"]
}
```

响应：`200 OK`，返回 version 递增后的 Memory。

规则：

- 至少提供一个可修改字段。
- 更新必须同步向量、BM25、KG/timeline 和审计记录。
- `expected_version` 可选；提供后用于乐观并发控制。
- 实现可以保存历史 revision，但公共 ID 保持不变。

### 4.4 Delete

```http
DELETE /v1/memories/{id}
```

响应：

```json
{
  "id": "mem_01J...",
  "deleted": true,
  "status": "deleted"
}
```

规则：

- 只做 soft delete/tombstone。
- 重复 DELETE 幂等，仍返回 `deleted=true`。
- 从普通 list、basic/advanced query、报告新输入中立即排除。
- v1 不接受 `purge=true`。

### 4.5 List

```http
GET /v1/memories?limit=50&cursor=...&status=active&kind=decision
```

响应：

```json
{
  "items": [],
  "next_cursor": null
}
```

规则：

- 默认 `status=active`、`limit=50`；最大 200。
- 稳定排序为 `updated_at desc, id desc`。
- Hermes 的 `garden_memory_list` 无输入，plugin 固定调用默认列表。

---

## 5. 旧 CRUD 兼容

Garden 当前接受：

```json
{"key":"memory:...","value":"...","meta":{}}
```

迁移期规则：

- `POST /v1/memories` 同时接受新 `{content,...}` 和旧 `{key,value,meta}`；混用返回 400。
- 旧请求维持旧响应 `{"id":"memory:..."}`，并记录 deprecation warning。
- `GET/DELETE /v1/memories/{key}` 继续接受 URL 编码后的 `memory:`/`section:` key。
- 新 `PATCH` 只适用于 canonical memory ID。
- 旧 `GET /v1/memories?prefix=...` 保持 prefix list。
- 新 canonical list 由 Hermes plugin 固定调用 `GET /v1/memories?view=canonical`，避免改变旧无参数 list 的默认 section 行为。
- 兼容入口至少保留一个发布周期；退役需另立决定。

---

## 6. Session Ingest

### 6.1 Submit

```http
POST /v1/sessions
```

请求：

```json
{
  "session_id": "sess_01J...",
  "event_id": "evt_01J...",
  "phase": "precompact",
  "content": "完整或增量 transcript",
  "content_hash": "sha256:...",
  "workspace": "C:\\Users\\Administrator\\Desktop\\garden",
  "occurred_at": "2026-07-15T10:00:00Z"
}
```

必填：`session_id`、`event_id`、`phase`、`content`、`content_hash`。

枚举：`phase=precompact|session_end`。

限制：

- `content` 最大 4 MiB。
- `content_hash` 是 UTF-8 content 的 SHA-256，由 plugin 计算，Garden 校验。
- `event_id` 是幂等键：同 event + 同 hash 返回第一次结果；同 event + 不同 hash 返回 409。
- 同一 session 的 precompact 与 session_end 可以是不同 event；Garden 根据 content hash/source coverage 避免重复生成相同 digest。

响应：`202 Accepted`

```json
{
  "ingestion_id": "ing_01J...",
  "session_id": "sess_01J...",
  "event_id": "evt_01J...",
  "status": "accepted"
}
```

### 6.2 Status

```http
GET /v1/ingestions/{ingestion_id}
```

响应：

```json
{
  "ingestion_id": "ing_01J...",
  "status": "accepted|running|completed|completed_degraded|failed",
  "memory_ids": ["mem_01J..."],
  "trace_id": "run_01J...",
  "warnings": [],
  "error": null
}
```

Hermes lifecycle hook 提交成功后不必轮询；status 供调试、恢复和验收使用。Garden 使用单进程 worker，不启动 subagent。

---

## 7. Context Resolve

```http
POST /v1/context/resolve
```

请求：

```json
{
  "intent": "为什么 Laputa 最后选择 A？",
  "session_id": "sess_01J...",
  "mode": "advanced"
}
```

规则：

- `intent` 必填，最大 16 KiB。
- `mode=basic|advanced|auto`；缺省 `auto`，兼容现有请求。
- Hermes native tool `garden_search(query)` 固定映射为 `intent=query, mode=advanced`。
- Hermes 不可传 pipeline/skill 名称。
- advanced 必须经过 Garden capability gate；LLM/KG/timeline 不可用时降级 basic，并在响应中返回实际 `mode` 与 warnings。
- 服务端总超时默认 30 秒。

响应：`200 OK`，格式为 §3.2 ContextPackage。

---

## 8. Basic Context Bootstrap

```http
POST /v1/context/bootstrap
```

请求：

```json
{
  "session_id": "sess_01J...",
  "intent": "当前用户消息，可为空",
  "budget_chars": 8000
}
```

响应：

```json
{
  "trace_id": "run_01J...",
  "context": "紧凑治理与记忆块",
  "evidence": [],
  "degraded": false,
  "warnings": []
}
```

该 endpoint 由 Hermes plugin 的启动/turn hook 调用，不注册为模型工具。`budget_chars` 由 plugin 固定配置，模型不可填写。

---

## 9. Report

```http
GET /v1/reports/latest?cadence=daily
```

`cadence=daily|weekly|monthly`，必填。

响应：

```json
{
  "cadence": "daily",
  "window_start": "2026-07-15T00:00:00Z",
  "window_end": "2026-07-16T00:00:00Z",
  "source_ids": ["mem_01J..."],
  "source_hash": "sha256:...",
  "title": "...",
  "summary": "...",
  "highlights": [],
  "open_questions": [],
  "generated_at": "..."
}
```

无报告时返回 404 `report_not_found`。报告生成由 Garden scheduler/pipeline 负责，不由 Hermes 模型工具触发。

---

## 10. Health

```http
GET /health
```

响应保持现有结构并允许增加版本信息：

```json
{
  "status": "ok|degraded",
  "components": {
    "garden": "ok",
    "laputa": "ok",
    "mentle": "ok|degraded",
    "pipeline": "ok|degraded",
    "planner": "ok|degraded"
  },
  "api_contract": "garden-hermes/1"
}
```

---

## 11. Hermes Declared Native Tools

Hermes plugin 只注册以下模型工具：

### 11.1 `garden_memory_create`

```json
{"type":"object","properties":{"content":{"type":"string"}},"required":["content"],"additionalProperties":false}
```

### 11.2 `garden_memory_get`

```json
{"type":"object","properties":{"id":{"type":"string"}},"required":["id"],"additionalProperties":false}
```

### 11.3 `garden_memory_update`

```json
{"type":"object","properties":{"id":{"type":"string"},"content":{"type":"string"}},"required":["id","content"],"additionalProperties":false}
```

### 11.4 `garden_memory_delete`

```json
{"type":"object","properties":{"id":{"type":"string"}},"required":["id"],"additionalProperties":false}
```

### 11.5 `garden_memory_list`

```json
{"type":"object","properties":{},"additionalProperties":false}
```

### 11.6 `garden_search`

```json
{"type":"object","properties":{"query":{"type":"string"}},"required":["query"],"additionalProperties":false}
```

session ingest、bootstrap、health 和 report scheduler 都不是模型工具。

---

## 12. 双方责任矩阵

| 项目 | Garden 方 | Hermes 方 |
|---|---|---|
| HTTP route/schema | 实现并维护 | 按合同调用 |
| canonical memory ID | 生成 | 只保存/传回 |
| CRUD mutation | facade + pipeline | native tool adapter |
| session transcript | 接收、幂等、处理 | 进程内收集并在 hook 提交 |
| precompact/session_end | 识别 event/去重 | 正确触发、生成 event_id/hash |
| basic context | bootstrap endpoint | hook 注入 |
| advanced query | pipeline/Skill/fallback | `garden_search(query)` |
| wing/room/KG | 内部管理 | 不可见 |
| auth | v1 无；loopback 限制 | 不发送 token |
| retries | 幂等支持 | 只重试 retryable 错误，有限次数 |
| subagent | 禁止启动 | 不要求 Garden 启动 |

---

## 13. 并行开发与冻结点

### Contract Freeze 0

松本确认本文以下内容后，双方即可并行：

- endpoint 路径；
- native tool 名称与最小 schema；
- session envelope 与幂等规则；
- v1 无鉴权、loopback-only；
- DELETE 只做 soft delete；
- context request/response；
- 错误格式。

### Garden 交付顺序

1. loopback 默认地址、错误 envelope、contract fixtures。
2. facade canonical mutation。
3. CRUD 与兼容层。
4. session accept/status 与 ingest pipeline。
5. basic/advanced、Skill registry。
6. bootstrap 与 report。

### Hermes 交付顺序

1. 新 plugin 骨架与 Garden client。
2. CRUD native tools。
3. `garden_search(query)`。
4. transcript buffer、precompact/session-end hook。
5. bootstrap 注入。

---

## 14. Contract Test Fixtures

双方各自保存相同 fixture，并在 CI/本地测试：

```text
fixtures/
  memory-create.request.json
  memory-create.response.json
  memory-update.request.json
  memory-delete.response.json
  session-precompact.request.json
  session-accepted.response.json
  context-advanced.request.json
  context-package.response.json
  error-invalid-request.response.json
```

Garden 验证 handler 输出符合 fixture；Hermes 验证 client 能发送/解析 fixture。fixture 变更视为合同变更，必须双方同步。

---

## 15. 松本需要确认的 6 个合同决策

- [x] 接受 v1 无鉴权、仅支持 loopback。
- [x] 接受 native tool 使用 `garden_memory_*` 命名，而不是旧 `garden_crud_*`。
- [x] 接受 DELETE 仅 soft delete，v1 无 purge。
- [x] 接受 session endpoint 异步返回 202，并提供 ingestion status。
- [x] 接受 canonical list 使用 `?view=canonical`，以保留旧 prefix/list 行为。
- [x] 接受 `garden_search(query)` 固定请求 advanced，但 Garden 可以因 capability/故障降级 basic。

以上六项确认后，本文状态从 `proposed contract` 改为 `accepted contract`，双方开始按 fixture 并行施工。
