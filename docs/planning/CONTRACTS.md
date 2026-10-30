# Garden × Hermes 内部合同（拟议 v2，未实现）

修订 `2026-10-31-native-review-v2`。仅针对首个 Hermes/profile，不是公共跨宿主 wire protocol。生命周期字段见 [宿主接缝](HERMES-REVIEW-BRIDGE.md)，责任见 [详细架构](ARCHITECTURE-SPINE.md)。

## 1. 身份、范围与许可

BoundContext 只能由可信入口/宿主适配器创建：home_key、resolved_home_ref、surface、operator_scope、parent_session_id、parent_turn_id、consent_ref、consent_revision、purpose、execution_generation。真实 home/路径私有；模型或正文不能指定 home、sender、scope、许可或目标路径。

V1 仅明确获准的本地操作者入口。默认拒绝未知、远程、QQ/群和跨 profile；未来 QQ 需单独验证真实 transport 绑定。profile_name/cwd/自填 origin JSON/工具可用性都不是授权。模型、输入内容和另一个候选不能发许可。

## 2. 类型与来源

| 类型 | 必须字段 | 权威 |
| --- | --- | --- |
| ReviewEvent | schema_version, kind, review_id, request_id, attempt_id, bound_ref, projection_ref, generation, occurred_at；start 可含受 grant 约束的 projection；end 增 outcome/reason/action_kind | 拟议宿主实际生命周期，不由模型返回 |
| EvidenceBatch | batch_id, home_key, consent_ref/revision, purpose, items[], evidence_digest | 可信投影/显式获准本地输入 |
| EvidenceItem | item_id, parent_turn_id, source_ref, visible_role, text, source_digest | 当前获准回合的 user/final assistant 可见文本；原生 metadata 不从正文解析 |
| AuthorityBaseline | baseline_ref, target=persona, exact_before_ref/digest, native_render_budget, observed_at | 获准后读取唯一原生 SOUL；不是模型输出 |
| Candidate | candidate_id, run_id, batch_id, target_kind, proposed_change, rationale, source_item_ids[], counterevidence[], supersedes_candidate_id?, baseline_ref, state | ID/绑定由代码补；模型只提供内容字段 |
| AdoptionDecision | decision_id, candidate_id, candidate_digest, actor_scope, policy_revision, verdict, expected_baseline_ref, generation | 本地可信决策或已冻结策略；candidate_only 不生成自动批准 |
| ApplyIntent | operation_id, candidate_id, decision_ref, exact_before_ref/digest, exact_after_ref/digest, writer_contract, generation, state | 目标写入前私有持久化；before/after 不可变 |
| ApplyReceipt | operation_id, observation, state, observed_target_digest, observed_at | 目标独立回读/对账；观察不推断写者身份 |
| ActivationReceipt | operation_id, home_key, session_id, expected_identity_ref, observed_identity_match, runtime/source_binding, state | 声明的实际请求路径，只证明指定 session/version |
| ControlState | home_key, enabled, execution_generation, consent_ref/revision, policy_ref/revision, retention_policy_ref | 私有 control.json；权限/撤回权威，不是另一份 run 状态 |
| RunRecord | schema_version, run_id, bound_ref, event_keys/digests, host_outcome, evidence_ref, analysis_key, analysis_state, candidate_ref?, decision_ref?, intent_ref?, apply_state, activation_ref?, record_revision, reason_code | 每 actual attempt 一个原子 JSON；唯一 run 状态权威 |

模型输出 schema：`result=no_change|proposal`；proposal 内容仅 target_kind、proposed_change、rationale、source_item_ids、counterevidence、supersedes_candidate_id。proposed_change 只含 text（非空字符串）与从宿主允许列表选择的 region_alias（persona 时必需）；实际文件/片段位置、精确 baseline 与最终拼接由代码提供，模型不能指定路径/字节偏移。counterevidence 为 `{source_item_id, explanation}` 列表；supersedes 仅允许宿主显式提供的同 scope 旧候选 ID，否则为 null。target_kind 仅 persona/user_fact/memory_note/skill/discard。最多一个 persona 候选；其他类仅分流建议，不自动写 USER/MEMORY/skills。

拒绝未知字段、错误类型、重复/越界 source IDs、跨 run/home 引据、未批准引用、空/超预算/边界外变更以及模型自填许可。反例只能引用同一获准批次/明确获准旧候选的安全引用；不据模型自述“已验证”。无 jsonschema 时也执行独立业务校验。模型 confidence/maturity 不产生写权限。

## 3. 拟议内部函数

- `bind_context(host_binding, policy) -> BoundContext | Denied`：无可信绑定 fail-closed，不回退 default。
- `ingest_event(bound, event) -> Stored | Duplicate | Denied | Conflict`：短回调，无模型/目标写入。
- `claim_run(bound, run_id) -> WorkLease | Busy | Denied`：work.lock 与 generation；不强抢活锁。
- `propose(bound, evidence, baseline, complete_structured) -> Candidate | NoChange | AnalysisFailure`：一次逻辑调用，无工具循环/Writer。
- `assess(bound, candidate, decision_policy, current_baseline) -> AdoptionDecision | Pending | Rejected`：D01 未定时只允许 Pending/Rejected。
- `prepare_apply(bound, candidate, decision, writer_contract) -> ApplyIntent | Blocked | Conflict`：D02 未定绝不构造可提交 writer。
- `commit_apply(bound, intent, writer) -> ApplyReceipt`：短 control.lock 下再次校验 generation/许可/基线；按选定写合同执行。
- `reconcile(bound, operation_id) -> ApplyReceipt`：只比较 current 与 exact before/after，不重放模型/写入。
- `verify_activation(bound, operation_id, authorized_fresh_session_probe) -> ActivationReceipt`：不能修改旧会话 prompt。

这些都不是已存在的 Hermes SDK 函数。现有宿主能力只通过真实 PluginContext/ctx.llm 等已核实接口封装；writer 接缝待 D02 冻结，不直接 import HTTP handler。

## 4. 状态机：三类状态不能互相冒充

**宿主状态**：waiting_host_end → completed/cancelled/failed/skipped/unknown；end 缺失为 incomplete_host_review。completed+no_action 可继续人格判断。

**分析/候选**：waiting_host_end → ready → analyzing → no_change/proposed/analysis_failed/analysis_unknown；无范围为 ignored_scope，无新证据为 no_new_evidence。proposed → approved/rejected；approved 未应用前策略/基线变动则需重新校验，不暗中复用旧许可。

**应用**：not_started → prepared → persisted_verified；异常可进入 outcome_unknown，再由 reconcile 进入 observed_before/observed_after/unknown_conflict。observed_before 不授予自动重试；observed_after 只证明当前字节相同，不证明归因。明确未动目标的写前错误为 failed_before_write。

**激活**：not_requested/unsupported/pending/effective_verified_for_session/failed/unverified。effective_verified_for_session 不能被总结为所有既有会话已更新；目标后续产生新版本时，旧 receipt 仍只是历史 session 事实。

## 5. 去重、generation 与取消

- event_key=(home_key, review_id, attempt_id, kind)，相同 key 不同内容拒绝。
- analysis_key=(home_key, evidence_digest, baseline_digest, analysis_policy_revision)。已知终态复用引用，未知分析隔离不自动重发。显式新分析尝试需新 attempt_id 和授权，不能抹去旧 unknown。
- operation_id 由代码生成并绑定精确候选/after/decision。相同 ID 只查询；相同 ID 不同 payload 冲突。新候选即便文字相同也不绕过基线与许可检查。
- ControlState 经 control.lock 原子持久化后才能确认持久撤权；写失败需报告失败并尽力阻断本进程，不虚报重启后仍已暂停。
- generation 在 pause/unload/revoke 后变化；结果持久化/决策/prepare/commit 前重查。迟到模型结果不得变成授权，timeout 不能保证上游没花 token。
- work.lock 保护每 home 单处理器；control.lock 保护短状态变更、撤权与提交。固定 work→control；控制操作不得等待 work。它们只约束合作进程，不是全局 SOUL CAS。

## 6. 写入、对账与回滚

顺序：不可变 before/after blob 持久化 → RunRecord 引用 durable intent → 选定 D02 合同下目标写入 → 重新打开独立回读 → RunRecord 结果持久化。断电/崩溃不构成多文件 ACID。

current==after → observed_after；current==before → observed_before；两者都不同 → unknown_conflict。若 before==after，直接 no_change，不生成写入操作。写后回执失败绝不据异常宣称“未写”。

回滚必须是新 operation、新 decision、同样并发合同，且 current 仍精确等于原操作 after；否则拒绝。回滚成功仍要重新验证激活，不覆盖后续合法编辑。未解决非合作写者竞态前不承诺全局安全回滚。

## 7. 存储与清理

拟用当前 home 的私有 plugin-data/garden：runs/<run_id>.json 为 run 状态源，blobs/<opaque_id> 为获准证据/精确恢复数据；索引仅缓存，不另设持久队列/另一人格。目录 0700、文件 0600，严格原子写与支持 OS 的 fsync 验证，拒绝路径逃逸。

未知 schema/损坏记录隔离，不能自动迁移或删除。保留规则必须在真实采集前批准；未终态、unknown 与需要恢复的 intent 暂停清理，等待明确处置。删除治理数据不级联删除原生会话/人格/记忆。日志仅安全 ID/状态/真实 usage，不输出原文、私有路径或完整 prompt。

## 8. 拟议本地控制面

inspect/status/pause/recover、受信任的 accept/reject/apply/rollback 是设计操作，不是已注册 slash 命令。accept 绑定精确 candidate/baseline；pause/revoke 可在模型执行时生效；recover 只做已许可状态恢复，不推断未知写入或重发未知模型。未批准采纳/writer/入口时返回 blocked，不向模型提供 apply 工具。QQ 发布与管理能力不因安装插件而扩大。
