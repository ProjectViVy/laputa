# 内部合同：共享 Markdown 人格源与短提交（拟议 v3）

修订 `2026-10-31-markdown-shared-persona-v3`；未实现。替换 v2 的 JSON RunRecord/ControlState、持久工作器与任务队列合同。本文件不是公共跨宿主 wire protocol。

## 1. 权威与绑定

PERSONA.md 是共享的唯一人格编辑源；SOUL.md 是 Hermes 原生运行入口/生成结果；可选 HOST.md 是获准私有宿主补充，不得改写共享使命/权限。无双向同步、无额外人格 prompt 注入。

BoundScope 由宿主可信配置/入口构造：persona_id、canonical_source_path、resolved_home、host_supplement_path?、soul_path、surface、consent_ref、policy_revision、approved_mission_digest、native_review_id、native_attempt_id、native_cancel_guard。源根目录与当前 home 都须明确批准；模型/正文不得指定路径、身份、许可或取消状态。

初始只允许获准本地入口；QQ/群/远程默认拒绝。共享目录不授予其他 profile 自动写权限，也不等于对互联网公开或所有 OS 用户可写。

## 2. 任务内类型（不持久为数据库）

| 类型 | 必须内容 | 来源 |
| --- | --- | --- |
| VisibleEvidence | batch_id, consent_ref, items(item_id, turn_id, role, text, source_ref) | 获准当前可见回合，非完整历史/工具/reasoning/system |
| PersonaSnapshot | persona_id, exact_source_bytes, protected_ranges, mutable_ranges, source_digest, host_bytes/digest?, soul_before_bytes/digest | 获准源/宿主文件读取；不能由模型填基线 |
| PersonaPatch | candidate_id, source_item_ids, allowed_region_alias, replacement_text, rationale, counterevidence, source_baseline_ref | ID/绑定由代码补；模型只给允许内容 |
| MissionChangeGrant | persona_id, expected_mission_digest, approved_change_digest, trusted_actor, scope/expiry | 用户可信入口对具体变更授权；自动人格阶段不能自产 |
| PreparedAssembly | source_after_bytes, soul_after_bytes, exact_before_refs, source/host/policy versions, permissions | 通过校验后代码组装；模型不生成路径/mission/版本 |
| ApplyResult | operation_id, observed_source_version, observed_soul_version, result_kind, reason_code | 真正独立回读；不能据“没有异常”猜成功 |
| ActivationReceipt | operation_id, session_id, expected_identity_ref, actual_identity_match, state | 指定获准新 session 实际请求路径；不是文件读取或自述 |

模型结果仅 `no_change|proposal`。proposal 输出 allowed_region_alias/replacement_text、source_item_ids、rationale、counterevidence；未知字段、伪造来源、改段边界/使命/权限、越预算拒绝。用户事实/记忆/技能仅分流，不自动写其他存储。

## 3. 使命与允许段

使命段及权限/用户红线属于 protected_ranges；自动阶段不得增删改字节、标题、边界或重新生成使命。空/未赋使命不由模型补齐；缺失/重复/解析失败拒绝自动写，不自己修补格式。

源使命须同时匹配 BoundScope 的 approved_mission_digest；外部编辑后的新使命未获用户更新批准 pin 时，自动组装/写入也拒绝，不只比较本次 patch 有没有改使命。pin 来自既有受信任宿主绑定/用户授权流程，不新增业务数据库。

普通人格/表达/自我认识可在明确 mutable_ranges 内自动变化；不是整篇重写权限。模型判断“我获准了”、高信心、引用用户或人名都不能产生 MissionChangeGrant。改变使命的提案即使放在人格段也应拒绝/转用户；语义不确定不能伪称安全证明。

用户修改使命是明确的人工授权流程，不作为无人值守 persona 阶段的自动分支；具体授权必须绑定当前使命版本和拟议改动。HOST 补充与组装也不能绕过这个约束。

## 4. 拟议函数边界

- `bind_stage(host_stage_context, approved_binding) -> BoundScope | Denied`
- `read_snapshot(bound) -> PersonaSnapshot | InvalidSource | UnmanagedTarget`
- `propose(bound, evidence, snapshot, complete_structured) -> PersonaPatch | NoChange | Failure`
- `validate_patch(bound, patch, snapshot) -> Allowed | NeedsUserAuthorization | Rejected`
- `assemble(source_after, approved_host_supplement) -> SoulBytes | BudgetError`：纯函数，无 LLM/IO。
- `commit_once(bound, snapshot, prepared) -> ApplyResult`：同实例互斥，提交前版本/取消/保护检查，一次替换每个文件。
- `inspect_last_apply(bound) -> ApplyResult | Inconsistent`：读固定 Markdown 恢复材料，不扫描任务库。
- `rebuild_host(bound, known_partial_apply) -> ApplyResult`：仅源 after/host before 且授权/版本仍有效时重新组装 host，不重发模型/重写源。
- `verify_activation(bound, result, authorized_fresh_session_probe) -> ActivationReceipt`

这些是新设计的内部 Python 边界，不是已存在的 Hermes API。受管 stage 接缝见 [bridge](HERMES-REVIEW-BRIDGE.md)。

## 5. 运行与取消

persona 阶段在同一原生 review 任务内同步执行；不能在 end 后用 spawn_task/独立 worker 接手。native_cancel_guard 来自父任务，进入模型前、模型后、提交前检查。模型网络取消尽力而为，不保证 token 未花费。

短提交有明确起点：若取消先到，不开始提交；若提交已开始，完成/报告这次短提交，不把取消说成追溯撤销已写文件。主前台保持原生有限等待/优先权，不等待人格模型无限返回。register/unload 不创建任务库；卸载后的迟到阶段无资格开始提交。

## 6. 文件提交与恢复

模型/组装在临界区外；同实例 persona 互斥区内重读 source/HOST/SOUL 版本并检查许可、使命与取消，准备完整临时 Markdown 和最后一笔恢复材料，逐个 atomic replace/readback。

没有全写者 CAS、跨进程写者协议或两个文件 ACID。可检测漂移冲突拒绝；外部编辑器在最后检查后竞写的窗口当前不消除。不是要求所有者每次开长维护窗口。

结果：no_change / needs_user_authorization / rejected / cancelled_before_commit / persisted_verified / source_saved_host_pending / observed_before / inconsistent_conflict / failed_before_write / outcome_unknown。persisted_verified 仍不是 effective。

固定私有 last-apply/ 包含 APPLY.md 与 SOURCE.before.md/SOURCE.after.md/SOUL.before.md/SOUL.after.md。APPLY.md 用有限 Markdown 字段记录目标绑定、版本摘要、提交阶段、观察、简短理由/usage；不存全聊天和隐藏推理。它不是 per-run 台账/队列/索引。未知/未完成上笔不能直接覆盖。

源/host 都 after：持久回读成立；源 after/host before：待重组装 host；都 before：观察到未应用，不自动重试；其他组合/第三种内容：停止检查。回滚须新授权，且源/host 仍分别等于原 after；否则拒绝覆盖后续修改。

## 7. 激活、隐私与配置

激活仅证明指定 session/version，其他旧会话不被宣称更新。无 refresh-prompts 时不以 repair/DB/reset 当前会话替代。

共享根实际目录、初始 source/mission、HOST、SOUL 接管、证据用途/路由/预算与保留规则在真实启用前批准；本轮没有真实文件迁移。运行文件为 Markdown；仓库 status.json 只是计划元数据，不部署为 Laputa 运行数据库。Hermes 既有会话/记忆存储不更换。

权限收紧与路径检查按目标 OS 验证；同进程恶意插件/同 OS 用户不是该设计可隔离的攻击者。日志只安全 ID/状态，不带真实人格、原文、完整 prompt、私有路径或 provider 原始异常。控制台查看/编辑及其 API 不在本期。
