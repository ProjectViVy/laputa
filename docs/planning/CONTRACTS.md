# 内部合同：独立纯文本文件、文件级策略与组装（拟议 v4）

修订 `2026-10-31-plain-multifile-persona-v4`；未实现。取消 v3 单文件分块/段解析/区域 patch；不增加数据库或持久任务框架。

## 1. 源格式和权威

本机共享人格目录中的独立文档是编辑源集合，原生 SOUL.md 为单向组装输出。文件名白名单/用途/权限/顺序在程序或受信任绑定中；正文是普通 UTF-8 文本/Markdown，不要求标题、分隔符、frontmatter、JSON/YAML、特殊标记或内嵌版本。

文件键为明确闭集，不接受模型任意路径：MISSION.MD、IDENTITY.MD、RELATIONSHIP.MD、REDLINE.MD、DREAM.MD、DARK.MD、USER.MD、WORLD.MD。首期自动可写集合为 IDENTITY/RELATIONSHIP/DREAM/DARK；MISSION/REDLINE 需用户授权，USER/WORLD 不自动写、不默认组装。Hermes 原生 USER/MEMORY/skills 仍独立，不同步两套用户事实。

本期 SOUL 顺序：MISSION→IDENTITY→RELATIONSHIP→REDLINE→DREAM→DARK，空可选文件跳过，加可选私有 HOST。只有输出可以加组装标题，源原文不被重写/裁剪。未初始化必要源不得由模型编造。

## 2. 类型（任务内，不是运行数据库）

| 类型 | 必须字段/职责 |
| --- | --- |
| BoundScope | persona_id、approved_source_directory、resolved_home、固定 file_key→路径映射、assembly_order、auto_writable_files、soul_path、可选 HOST、consent/policy、approved_mission_digest、原生 attempt/cancel guard |
| SourceSnapshot | file_bytes: file_key→原文字节、source_versions、完整读集、host_bytes/version?、soul_before/version；由真实获准文件读取，不解析段 |
| VisibleEvidence | 获准当前回合 item_id/turn/role/text/source_ref，不含整段历史/system/tool/reasoning |
| FileReplacement | candidate_id、file_key、expected_file_version、replacement_text、source_item_ids、rationale、counterevidence；每 actual attempt 至多一个自动可写目标 |
| MissionChangeGrant | persona/file、expected_version、approved_change_digest、可信用户授权；模型不能生成 |
| PreparedAssembly | changed_file_key、该文件 before/after、全部输入版本、soul_after_bytes、绑定/许可 |
| ApplyResult | operation_id、源读集/目标的真实观察、result_kind、reason_code |
| ActivationReceipt | operation_id/session/version、实际请求身份匹配、runtime 绑定；只证明指定会话 |

模型输出 `no_change|proposal`；proposal 仅 file_key、replacement_text、引据、理由/反例。代码填 ID/版本/绑定。拒绝未知字段、未知/禁写文件键、任意路径、伪造引据、删除/重命名文件与自填授权；不因为正文有多个标题/列表/空行而失败。独立业务校验不依赖 jsonschema 恰好安装。

## 3. 使命与权限

MISSION.MD 全文件自动只读；REDLINE.MD 不自动放宽/改写。自动替换其他文件不能改写它们，未赋使命不由模型补齐。源使命须匹配受信任绑定的获准版本，外部更改文件不自动成为用户授权。新使命/批准 pin 的更新绑定具体用户授权，不能靠昵称、引用、模型信心或“自主探索”泛化。

改变使命/权限的语义提案不能靠换成 IDENTITY 目标绕过；明显越界拒绝，不确定交用户，不声称程序证明完整自然语言安全。源文件无内部授权字段；权限不由正文里的配置样文本决定。

## 4. 内部边界（全部拟议）

- bind_stage(host_context, approved_binding) → BoundScope/Denied
- read_sources(bound) → SourceSnapshot/InvalidFile/UnmanagedTarget：只读明确组装读集，首期不自动读取 USER/WORLD/备注/备份；正文不透明，不 glob 目录。
- propose(bound, evidence, snapshot, complete_structured) → FileReplacement/NoChange/Failure
- validate_replacement(bound, replacement, snapshot) → Allowed/NeedsUserAuthorization/Rejected
- assemble(file_bytes_by_key, fixed_order, optional_host) → SoulBytes/BudgetError：纯函数，不调用模型、不解析源章节。
- commit_once(bound, snapshot, prepared) → ApplyResult：锁外完成模型/文本，短提交前重查所有输入/许可/取消。
- inspect_last_apply / rebuild_host → 只在部分提交的精确已知版本下恢复，不重发模型/重写未变源。
- verify_activation → 仅在获准新 session 实际请求路径验证。

## 5. 同步与取消

Laputa 自己完成源文件修改后同轮重组装 SOUL。人类手改在下一次获准 review 或显式 rebuild 同步；无新证据可只组装，零额外模型。MISSION 改动仍需批准，当前 session 不承诺立即热刷新。

同一原生 review 任务保留父取消管理到 persona 阶段退出；不另起 end 后 worker。进入模型前/后/提交前检查。取消先到不提交；短提交已开始则完成/报告其真实结果，不把网络取消当追溯撤销。前台有界优先，卸载后旧 callback 无新提交资格。

## 6. 原子性与最小恢复

本期一次自动更新最多一个源文档；未变文件不重写。全部新文本先在锁外准备，短互斥区核验完整源读集/HOST/SOUL/使命 pin 后，持久必要 Markdown 恢复材料，再各自 atomic replace/readback。

没有目录整体事务/跨文件 ACID/全写者 CAS。检测漂移拒绝，非合作外部编辑器最后检查后的竞态本期不消除。私有 last-apply/：APPLY.md、before/<changed_file_key>、after/<changed_file_key>、SOUL.before.md/SOUL.after.md；其他输入的版本记录在附属回执，不写入源正文。未知上笔不能覆盖，清理不级联原生数据。

源 after、其余输入仍匹配、SOUL before：source_saved_host_pending，只允许在许可有效下重组装 host；都 after：persisted_verified；都 before：observed_before；其他/第三种内容：inconsistent_conflict，停止。回滚须新授权且当前仍等于原 after；不覆盖后续合法编辑。

## 7. 授权/隐私

真实目录/初始文档/使命 pin、HOST/原 SOUL 接管、证据/供应商/预算/保留及代码/启用/写入/新 session 均仍需授权。本轮只改文档，status.json 是计划元数据，非运行数据库。共享是本机、按 persona 绑定，不是公开互联网或所有用户可写。

日志仅安全状态/引用，真实正文和 before/after 只在获准私有范围；不存全聊天、隐藏推理或完整 prompt。控制台查看/编辑与其 API 后置。详细映射见 [架构](ARCHITECTURE-SPINE.md)，宿主阶段见 [bridge](HERMES-REVIEW-BRIDGE.md)。
