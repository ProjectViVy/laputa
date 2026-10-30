# Hermes review 接缝：拟议合同，不是现成功能

修订 `2026-10-31-native-review-v2`；D03 待决；core/测试修改均未授权。以下 hook、字段、能力宣告和调用接口全部拟议，不应按现成命令/API 使用。

## 1. 现场事实与缺口

固定 Hermes `bf0203a794a07aac3ff764972ab9087baa4e4f4e`：

| 已存在位置 | 核对内容 | 缺口 |
| --- | --- | --- |
| `agent/turn_finalizer.py:778-792` | 满足原生 memory/skill 条件才 spawn；skip/interrupt 可抑制 | 不是每次聊天都 review；不另加时钟绕过门槛 |
| `run_agent.py:757-827` | enqueue/now、run token、线程与上下文传播 | 没有 Garden 可依赖的稳定 review/attempt 关联与完整生命周期合同 |
| `run_agent.py:831-850`、`agent/review_idle_queue.py:97-107,163-178` | preempt 重排、最新快照合并、运行前再查 enabled | 队列不是持久 job store；合并前的快照不是实际 attempt |
| `agent/background_review.py:1088-1128` | extra_tools 仅在 dispatch 白名单放行已注册父工具 | 不保证模型调用；不能当完整终态回调 |
| `agent/background_review.py:1183-1207,1231-1246,1282-1300` | 继承 snapshot/digest；早退；仅 actions 时发布摘要；finally 清理 | 摘要和 finish token 都不是统一、对外可用的成功/no-action/取消/失败终态 |
| `hermes_cli/plugins.py:943-965` | 未知 hook 注册可仅告警 | 注册成功不能证明有真实 fire site |

## 2. 最小扩展面（建议，待批准）

只增加 typed 生命周期通知、可信投影与版本宣告；不改 review 原有触发门槛、模型/工具集、memory/skill 逻辑或前台 prompt。

| 现有文件 | 拟议变更 |
| --- | --- |
| `agent/turn_finalizer.py` | 在既有触发点且 Garden 数据用途已批准时，从当前完成回合构造 allowlist 投影；缺可信本地 scope 就不采正文 |
| `run_agent.py` | 捕获不可变 ReviewRequestBinding，经 defer/requeue 传递；实际 run admission 后生成 attempt，线程启动失败也结算 |
| `agent/background_review.py` | 为 actual attempt 增加明确结果，外层统一 start/end fire site，覆盖现有 try 之前的早退；终态去重 |
| `hermes_cli/plugins.py` | 声明真实 hook 名/版本与只读能力查询，字段同时落文档；不能只加 VALID_HOOKS 而没有调用点 |

宿主单元测试路径由 G02 在获准的隔离源码树核实；新增测试文件是产品实施，不在本轮创建。不能 monkeypatch、私下改已安装 core 或反射窥探 review agent 来代替正式接缝。

## 3. 拟议事件

建议 hook 名 `on_background_review_started` 与 `on_background_review_finished`；合同版本 `hermes.background_review.v1`。名称是设计值，不是当前 VALID_HOOKS 项。

共同字段：schema_version、review_id、attempt_id、request_id、parent_session_id、parent_turn_id、bound_context_ref、source_projection_ref、execution_generation、occurred_at。所有 ID/绑定来自宿主，不从模型或正文提取。

- request_id：触发点生成，经排队传递；同请求重排保持不变。
- review_id：同逻辑 request 的 review 标识；重排保持不变。
- attempt_id：每次成功 run admission 新建；重排生成新的 attempt；准入失败则没有实际 attempt 事件。
- started：实际 admission 后、provider-capable 阶段之前，含 `projection: VisibleTurnProjection | null`；类型只有 batch_id、purpose、grant_ref/revision 和允许的 EvidenceItem 列表（item_id/parent_turn_id/source_ref/visible_role/text/source_digest）。source_projection_ref 是该投影的宿主不透明 ID；消费者通过此 payload 得到文本、准入后另行私有持久化，不要求 core 建证据库。绝不携带完整 snapshot/digest。
- finished：保留实际 `run_conversation` 的结构化结果用于分类；failed/interrupted 不能因正常 return 或存在 actions 判为成功，无法确认则 unknown。增加 outcome=`completed|cancelled|failed|skipped|unknown`、reason_code、action_summary_kind=`changed|no_action|unavailable`、usage_summary（存在才传）。no_action 是原生工具动作结算，不等于 Garden persona no_change。
- source_projection_ref：可指 started 的最小投影副本；end 不重新导出原文。没有投影不是许可去 session_search 补全文。

迟到/重复事件不改变绑定。相同 event_key 和相同事件摘要幂等；相同 ID 不同摘要冲突。每次重排的取消结算不得误终止新 attempt。

## 4. 精确生命周期

```text
原生门槛命中 → 冻结请求绑定/获准可见投影 → 原生立即执行或 idle queue
实际 run admission → started
    ├─ 启动前取消 → finished(cancelled)
    ├─ 不支持 provider / 明确跳过 → finished(skipped)
    ├─ 构建、线程启动或运行错误 → finished(failed)
    ├─ 被抢占 → finished(cancelled)；原生可按原策略 requeue 新 attempt
    └─ 正常返回 → finished(completed, changed|no_action|unavailable)
完成清理 → finished 只能对该 attempt 发布一次
```

实现要把目前外层 try 之前的 return 纳入统一结算。不要在 `finish_background_review_run()` 这一“provider 阶段可能已返回”的内部清理点直接宣布整个 review 完成；必须在外层结果分类与资源清理边界结算。回调异常隔离，不能把 Garden 故障升级为前台失败。

## 5. 数据/线程约束

core 不读取 Garden 私有 policy/control 文件。拟议 HostProjectionGrant 由所有者通过宿主支持的配置生命周期单独批准，绑定解析 home、允许本地 surface、用途、范围/尺寸与 grant revision，默认不存在/拒绝；这是 D03 新设计，不是当前设置。Garden 还须用自身 ControlState 再准入才持久化或分析。宿主无 grant 时只发元数据，不先广播文本再让插件决定是否有同意。插件暂停/撤回可立即阻止其持久化/分析；若要撤销宿主投影授权，须同时撤回宿主 grant 并验证运行加载。

该投影仍是受信任宿主进程内数据，V1 不承诺隔离同进程的恶意插件/同 OS 用户；需要互不信任插件的保密隔离时本部署不适用，不能靠 observer 标签冒充 sandbox。

- 入口范围先由宿主可信 local/session context 与明确同意确定；后台线程的 task-local 值缺失/变更不能退回 default owner。
- 仅最初触发回合的原始可见 user/final assistant 字段；历史、tool、reasoning、系统/技能注入字段不进入 persona evidence。父 review 仍按原逻辑处理自身快照，Garden 不接收那份快照。
- 委派/cron/远程/QQ/群/跨 profile 不因 review 标记或昵称取得许可。V1 不把 QQ origin 正文 JSON 当可信 transport 来源。
- raw focus、prompt、异常、完整 review messages 和 SDK/agent 对象不在事件合同中。投影用途只限获准人格分析，不形成通用遥测导出。
- 消费回调只验证/私有落盘/唤醒，不能在宿主 review 线程调用模型、apply、等待交互或做全库扫描。数据输入扩展不是 Observer 天然授予的采集权。

## 6. 可靠性与降级

V1 是 in-process fail-open 通知；正常执行路径有完整终态，骤死在交付前不保证补发。start 已持久而 end 未收：incomplete_host_review；未获持久 start：不能宣称知道该 attempt。plugin 未运行时不重放历史事件。

Garden 已持久化的 ready/intent 可以恢复；未知模型调用不自动重发，未知目标写入只对账。严格全事件持久性需要宿主 outbox/游标，是独立设计与授权，不隐含在“薄插件”里。

兼容必须同时验证：宿主版本/能力宣告、真实 fire sites、payload schema、全终态集成与上下文传播。缺一则 automatic unsupported、apply disabled；不得只看注册表或官网能力。升级/启用/回退都不改当前运行实例，待后续批准。
