# Garden × Hermes 内部合同（拟议，尚未实现）

只用于首个本地 profile 扩展，不是跨宿主公共 wire protocol。实现基线和状态见 [plans/index.md](plans/index.md)。所有路径和接口均为拟新增。

## 可信边界

`BoundContext` 只能由入口适配器构造：`home_key`（解析后的真实home，私有）、`surface`、`operator_scope`、`session_id?`、`consent_ref`。模型或命令正文不能传入任意home、sender、许可或文件路径。

首期只允许明确受信任的本地操作者上下文；未知/远程/跨profile拒绝。此规则不声称能阻挡同一OS用户任意运行的受信任Python或shell。QQ开放需后续可信transport绑定与负例验收，不能靠slash列表默认值推断。

## 内部数据

| 类型 | 必须字段 | 权威来源 |
| --- | --- | --- |
| EvidenceBatch | batch_id, home_key, consent_ref, purpose, items[] | 宿主/操作者，原文仅私有 |
| EvidenceItem | item_id, source_ref, visible_role, text | 显式选定可见文本；不接reasoning/system/tool原始字段 |
| AuthorityBaseline | target=persona, exact_before_bytes, rendered_budget, observed_at | 获准后从原生SOUL读取；不是模型提供 |
| Candidate | candidate_id, batch_id, target_kind, proposed_text, rationale, counterevidence[], source_item_ids[], baseline_ref, state | ID/绑定由host填，模型只生成允许的内容字段 |
| AdoptionDecision | candidate_id, actor_scope, policy_revision, verdict, expected_baseline_ref | 受信任操作者或另行批准的策略，不来自模型 |
| ApplyIntent | operation_id, candidate_id, exact_before_bytes, exact_after_bytes, decision_ref, state | 在写目标前私有持久化 |
| ActivationReceipt | operation_id, home_key, session_id, expected_identity_ref, observation, state | 原生加载/实际请求路径的私有证明 |

`target_kind` 仅 `persona | user_fact | memory_note | skill | discard`。非persona仅分流到已有原生工作流，不由首个人格writer自动写其他存储。拒绝未知字段、重复/越界 source_item_ids、非法目标、空/超预算人格和跨scope引用。模型JSON即使parsed成功也独立校验。

## 拟议函数边界

- `bind_local_context(ctx) -> BoundContext`：固定home与入口，失败不生成“默认身份”。
- `propose(bound, evidence, baseline, complete_structured) -> Candidate | NoChange`：一次逻辑模型调用；无写入、无工具循环。
- `assess(bound, candidate, decision, current_baseline) -> Eligible | Rejected`：许可、作用域、出处和版本检查。
- `apply(bound, candidate, decision, writer) -> ApplyReceipt`：G04关卡通过后才实现；writer必须满足选定并发合同，不能伪装native已有CAS。
- `reconcile(bound, operation_id) -> ApplyReceipt`：current==after→observed_after；current==before→observed_before；其他→unknown_conflict。结果说明实际观察，不推断是谁写的。
- `verify_activation(bound, operation_id, fresh_session_probe) -> ActivationReceipt`：只在声明的会话边界验证，不能修改现有缓存prompt。

这些是内部Python边界名，不是已经存在的Hermes API。宿主交互仅包裹实际的 `PluginContext`、`ctx.llm` 和获准后确认的原生写入/加载接口；不能直接把当前REST handler当可import的领域服务。

## 状态

候选：`proposed | no_change | rejected | approved`。
应用：`not_started | prepared | observed_before | persisted_verified | unknown_conflict | failed`。
激活：`not_requested | unsupported | pending | effective_verified | failed`。

`approved`不等于写过，`persisted_verified`不等于生效。完整成功必须同时有正确原生权威的回读和声明边界的有效加载证明。当前所有真实应用/激活均未执行。

## 幂等、恢复与回滚

operation_id由host生成且绑定candidate和精确after内容；重复查询返回既有结果。相同ID不同内容拒绝。崩溃后不再调用模型，不自动重放目标写入。

先保存intent（失败则不动目标）→按明确并发合同写入→独立回读→结果持久化。成功写入后回执失败报告unknown；reconcile根据真实文件判断，不据异常宣称未写入。

回滚也是一笔新操作，要求当前内容精确匹配本操作after且受同一写入合同保护，再应用before并重新验证激活。若后续有人编辑，拒绝回滚。未解决非合作写者竞态前，不提供“安全全局CAS回滚”的承诺。

## 私有存储

拟在当前home的 `plugin-data/garden/` 保存按run/operation分开的原子JSON文件与必要before/after。目录仅所有者可访问，原文文件权限收紧；候选和备份不作为独立prompt源。无新数据库、外部服务或长期记忆副本。日志只含ID、状态与安全摘要；不把raw模型异常或完整系统提示发到QQ/公开仓库。

保留期与明确删除语义需在真实数据使用前冻结；删除治理材料不级联删除原生记忆或会话。没有声明多文件ACID或跨重启exactly-once。
