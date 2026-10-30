# 详细架构：Garden × Hermes 原生反思驱动的人格沉淀

设计修订：`2026-10-31-native-review-v2`。状态：CONCERNS / 仅设计；`implementation_authorized: false`。本稿替换“手动候选命令为主路径”的初稿，不保留两套执行计划。规范字段在 [CONTRACTS](CONTRACTS.md)，宿主接缝在 [HERMES-REVIEW-BRIDGE](HERMES-REVIEW-BRIDGE.md)，任务源在 [status.json](status.json)。

## 1. 目标与权威

Laputa 定义权威、证据、采纳、冲突、恢复和有效性语义；Garden 是首个 Python 实现，服务 Hermes 中松柏自身的人格沉淀与进化。不先抽出通用 SDK、独立服务或多宿主框架。

旧 Go 版仅用于定向参考架构、闭环经验和故障案例。按最新决策重写 Python 代码，不翻译旧包，不继承旧 Effect、存储格式、接口、审批或调度。已定位旧 main `9acc999301d5d575da2cea902a1c6d213763f381`，不据此宣称旧版已重验或新实现已完成。

唯一运行人格是解析后 home 中的原生 SOUL。Garden 私有候选、证据、before/after 和回执不是另一人格源，绝不注入为额外 system prompt。USER、MEMORY、skills 与会话 overlay 仍归原生机制。

## 2. 系统形态与责任分配

**一个可独立分发的薄原生插件 + 一个拟议的最小 Hermes 生命周期接缝 + 操作技能。** 接缝不是新 Agent、记忆后端或新定时系统，且当前源码尚不具备完整合同；任何 core 修改、插件安装或运行启用均待另行授权。

| 单元 | 负责 | 不负责 |
| --- | --- | --- |
| Hermes | 原生后台 review 的门槛/排队/取消、会话/凭证/辅助模型、人格加载、既有 memory/skills | 不假定已经有人格候选治理或跨写者 CAS |
| 最小宿主接缝（拟议） | 可信调用绑定、获准可见证据投影、实际 review attempt 的 start/end 事件 | 不生成/采纳人格，不复制全部历史，不变更现有调度策略 |
| Garden 插件 | 验证范围、私有状态、受限人格候选、采纳关卡、应用/恢复/激活证据 | 不拥有新 Session、Recap、人格加载器或长期记忆库 |
| 原生 memory/skills review | 继续原有事实/技能维护 | Garden 不重复写同一事实或流程 |
| 本地控制入口 | inspect/status/pause/recover，以及已授权策略下的决策/应用 | 不是每轮反思必须人工发命令，不默认开放 QQ 群控制 |

```text
Hermes 原生 review 门槛 / idle queue / cancel
         │ 实际 attempt；携带触发时冻结的可信范围与可见证据投影
         ▼
拟议 start/end 事件 ──► Garden 私有 RunRecord（先落盘，短回调）
                            │ 合格终态 + 未撤权 + 新证据
                            ▼
                     单 profile 惰性工作器
                            │ 至多一次有界 ctx.llm 人格归纳
                            ▼
                 分类 / 出处 / 反例 / 候选或 no_change
                            │ 独立业务校验 + D01 采纳关卡
                            ▼
                 D02 写入关卡 → intent → 原生 SOUL → 回读
                            │ persisted_verified ≠ effective
                            ▼
                 获准的新验收会话 → 实际身份块证明 → 私有结果
```

## 3. 为什么不直接拿原生 review 的完整上下文生成人格

固定源码的同路由 review fork 会继承 messages_snapshot，异路由使用 digest；extra_tools 仅开放已在父工具表中的工具，不保证模型调用。见源码审计 E20–E23。

这不满足 R02 的“人格证据仅限获准可见文本”。因此 **V1 复用原生时机和生命周期，但不把完整 review 快照或原始 completion 当作人格输入**。有新增合格证据时，使用原生 ctx.llm 做一次独立、无工具循环的人格专用结构化归纳，不再重做普通 memory/skills 复盘；空、重复、不合范围和失败终态零新增调用。

这确实可能比“原生 review 必定提交候选”多一次辅助调用，是数据边界的明确成本，不宣称零额外 token。若以后原生提供满足同样证据边界的 persona 阶段，才考虑合并调用。V1 不同时维护 extra_tools 提案主路径与自动归纳主路径。

## 4. 触发、证据与可靠性范围

1. 只在所有者启用且授予数据用途后，宿主在触发 review 时投影本次完成回合的原始可见 user 文本和最终可见 assistant 文本；不采系统提示、注入技能、工具原文、reasoning、完整历史或 review 私有过程。
2. 投影绑定真实 home、受信任入口、原始回合和同意版本；在排队前冻结，通过宿主 kwargs/上下文传递，不能在后台线程读取“现在”的入口来冒充旧回合。
3. 首期只接明确获准的本地操作者入口。QQ（包括所有者 DM）、群、远程和其他 profile 的自动证据入口在可信 transport 绑定独立验收前拒绝；正文中的 origin JSON/昵称不构成身份。
4. 只有实际获准创建的 review attempt 产生 start/end；原生门槛未命中、尚未 dispatch 的排队项和被 coalesce 掉的快照不是已运行 attempt。不扫描历史补齐它们。
5. 成功完成才可推进人格分析；原生 no_action 也可分析人格，因为“未修改技能”不代表“没有人格增量”。cancelled/failed/skipped/unknown 只结算，不生成或应用候选。
6. 每次实际 attempt 可以结算 ignored_scope、no_new_evidence、no_change、candidate 或失败，不强制产生人格变化。
7. **正常控制流的终态覆盖，不是跨进程 exactly-once。** 宿主骤死在事件持久化前可丢事件；事件已收但 end 缺失只记 incomplete_host_review，不猜成功。严格不丢事件需另设宿主持久 outbox，不在 V1 偷加。

## 5. 内部组件与依赖

以下均为拟新增路径，尚无产品代码：

| 模块 | 职责 / 依赖方向 |
| --- | --- |
| `garden_hermes/domain.py` | 严格类型、状态转移、不可变绑定；无宿主/IO依赖 |
| `garden_hermes/host.py` | 固定 SDK/能力适配、真实 home/入口、ctx.llm、原生写入接缝；不 import Web handler 当领域服务 |
| `garden_hermes/store.py` | 私有 RunRecord、不可变内容 blob、锁、恢复读写；不是人格/记忆引擎 |
| `garden_hermes/lifecycle.py` | start/end 去重、惰性工作器、取消/撤权/恢复；依赖 domain/store/host |
| `garden_hermes/proposal.py` | 获准证据到 Candidate/NoChange；无 writer 引用 |
| `garden_hermes/review.py` | 确定性校验、D01 许可/采纳；不写 SOUL |
| `garden_hermes/apply.py` | D02 合同下写入、对账、有条件回滚；只处理 SOUL |
| `garden_hermes/activation.py` | 新验收会话能力检测与身份块证明；不改旧 prompt |
| `garden_hermes/__init__.py`, `plugin.yaml` | 最终统一注册/卸载、能力拒绝与本地命令接线 |
| `skills/garden/SKILL.md` | 操作与失败处置说明；不作为权限执行器 |

底层不依赖上层；proposal 不调用 apply；生命周期不决定权限；模型不生成 BoundContext/Decision/文件路径。无事件总线框架、外部队列、ORM、新数据库或 Web/MCP/REST。

## 6. 人格沉淀，而不是聊天摘要

模型只提出 target_kind、稳定自我认识、出处、理由、反例和替代/撤回关系。人格必须是有证据的自我认识；一时情绪、角色扮演、单次任务命令、第三人自称授权不能直接成为稳定自我。

- persona：暂定观察或可采纳增量；支持反例、修订、撤回旧认识，不只做单向累加。
- user_fact / memory_note / skill：返回指向原生机制的分流建议，不由 Garden writer 再写；不从原生 review 的原始记录抓文本证明已处理。
- discard / no_change：合法终点，保留安全理由码即可。

候选成熟度和采纳是两回事。模型认为 stable 不授予自动写入；程序结构校验不证明观点真实，也不构成自然语言安全沙箱。

## 7. D01：采纳与可演化范围（待所有者确认）

部署默认 `adoption_mode=candidate_only`，只产出候选。建议先用合成/真实验收证明闭环，再开放 **一次授权范围内的有限自动采纳**，而不是每轮都询问或无限制自改。

建议的自动写范围是 SOUL 内经所有者一次性批准的可演化片段：边界外字节必须完全不变，片段不承载管理员、工具、隐私或执行授权。新增此片段/标记本身也是一次真实 SOUL 修改，必须单独批准；当前不读/改真实 SOUL，不假定已有此格式。整篇人格重写或边界外变更转人工处理，不暗中扩大范围。

需要冻结：允许片段和主题、证据成熟条件、反例/否决条件、策略版本/撤回、人工采纳接口与超时拒绝。D01 没有决定前，有限自动模式和应用入口保持不可用；候选生成可先独立验证。

## 8. D02：写入并发合同（P0，不能靠“原子写”消除）

当前原生 SOUL PUT 只有原子替换，无跨桌面、文件工具和外部编辑器的共享 CAS。

- **建议首期采用明确受控的独占维护窗口**：所有者暂停其他 SOUL 编辑；Garden 合作写者串行，写前/写后核验，漂移拒绝。该选项缩窄 R05 支持范围，只有所有者明确接受后才成立，不适合宣称无人值守的任意并发热更新。
- 若要求任意原生入口并发而不丢更新：须先另行设计/授权统一 expected-version 写入合同；通用 shell/非合作外部编辑仍不能被普通合作锁神奇阻止。

D02 未定时 writer 不是“尽量安全版”，而是 blocked。具体可调用写入接缝由所选合同冻结；不能把 HTTP handler 当现成 SDK API，也不绕文件写入审批。

## 9. 私有持久化与执行生命周期

拟用宿主认可的当前 home 私有 `plugin-data/garden/`，是否与固定 SDK 数据目录一致由 G01 核实；不是配置为任意目录的公开服务。

- 每个 attempt 一个原子 JSON RunRecord，是该 run 的唯一状态权威；其引用的 evidence、baseline、intent before/after blob 不可变。无独立队列状态或另一份人格。可重建索引只作缓存。
- 内容先写临时文件、flush/fsync、原子替换，目录持久性按支持 OS 验证；权限目录 0700/文件 0600，拒绝路径/符号链接逃逸。不声称防同一 OS 用户的任意代码。
- `work.lock` 限制每 home 同时一个处理器；`control.lock` 只保护短状态变更与提交检查。锁顺序 work→control；pause/revoke 只取 control，不能等模型返回才能撤权。不得把这两把合作锁描述成 SOUL 全局 CAS。
- 原生回调仅验证/写事件状态/唤醒，不在 callback 调模型、等人工或写 SOUL。工作器按需启动，无独立周期时钟；生产状态为空时不轮询。
- register 只注册能力和控制入口，不读真实人格、扫描会话或启动模型/任务。启用后在下一原生事件或本地 `recover` 才有界扫描未终态记录；可选宿主 cron 只做已授权恢复触发，不作为另一反思时钟，V1 不要求创建 cron。
- plugin unload/pause 递增持久 generation，尽力取消并禁止旧 generation 提交。迟到 LLM 可已花费 token，不能保证网络调用撤销；返回后必须重新检查 generation/许可，旧任务无写入资格。锁未释放前不启动重叠处理器，不靠 PID 猜测强抢活锁。

## 10. 一次正常流程

1. 宿主冻结获准投影，实际 attempt start；Garden 写 waiting_host_end。
2. 终态 completed 到达并持久化，候选任务 ready；无 start/无投影/错 home 则拒绝或 incomplete，不事后扫 DB 拼证据。
3. 工作器检查启用/同意/预算/输入尺寸/基线读取许可；空批次零调用跳过。
4. 读获准的原生人格基线，构造 analysis_key 并查重；已知终态复用，未知分析隔离。仅将批准的可演化片段/必要人格描述作为编辑目标上下文（不当事件证据），连同 evidence 提交一次结构化 completion。完整 exact_before 留在本地，未获整篇外发同意不能把整篇 SOUL 送辅助模型。
5. 严格检查字段、引据、分类、反例、有效渲染预算和可写范围；格式失败不自动修复重发，留失败记录。
6. 候选持久化；candidate_only 等待本地受信任决策，或在已冻结 D01 策略下形成 Decision。
7. 获得 D02 写入条件后，再查 generation、同意/策略版本、候选精确基线；持久 intent → 目标写入 → 独立回读 → persisted_verified。
8. 没有获准的新会话验证时保持 activation pending；不自行中断当前 QQ 工作会话。
9. 在授权的新验收会话实际请求身份块中证明指定版本，形成只针对该 session/version 的 ActivationReceipt。其他旧会话不被宣称同时更新。

## 11. 故障、去重与恢复

| 边界 | 行为 |
| --- | --- |
| 重复相同事件 | 同 event_key/同摘要返回已知结果；同 ID 不同内容冲突 |
| 原生未完成/取消/失败 | 仅结算，不产生/写入人格；no_action 不是失败 |
| 模型开始前崩溃 | ready 可恢复，仍须复核许可/基线 |
| 模型调用中崩溃 | analysis_unknown，不自动补发；同 analysis_key 被隔离，须显式授权新分析 attempt |
| 模型结果有了但未持久化 | 仍属于未知分析，不根据日志重建虚构候选 |
| intent 持久化失败 | SOUL 不变 |
| SOUL 写入后回执失败/进程退出 | 不再写一次；只做 current 与精确 before/after 对账 |
| current == after / before / 其他 | observed_after / observed_before / unknown_conflict；不推断是谁写的 |
| 回滚时已有后续编辑 | 拒绝；回滚是新 intent/新许可/新激活验证，不覆盖新内容 |
| 记录损坏、空间/积压上限 | 隔离并暂停受影响处理；不删除未知 intent 或从默认身份继续 |
| 旧工作器迟到 / 同意已撤回 | 不准提交；记录取消/撤权，不把 timeout 视为撤销副作用 |

## 12. 生效、兼容与升级

固定源码 `bf0203a794a07aac3ff764972ab9087baa4e4f4e` 的缓存恢复/加载约束见源码审计。安装文件存在、hook 注册成功、磁盘 HEAD、运行进程能力和真实闭环分别验证。

首期不热改旧 prompt、不改 DB 缓存、不强制 compress/reset/restart、不用 repair-prompts 冒充 refresh。采用隔离合成 profile 及后续获准真实新 session；临时只读验收探针在实际请求路径内比较身份块，仅输出布尔/私有版本摘要，立即卸载，不保存 system prompt。截断/overlay/错路由必须暴露，缺观察字段返回 unverified。

宿主未实现拟议 bridge 或 schema 不匹配时，automatic=unsupported，apply disabled；本地候选调试不冒充自动闭环。升级前跑 bridge 合同与隔离集成；停用插件不回滚已写人格，不自动恢复旧 SOUL。

## 13. 配置、隐私与成本

拟议开关不是现成 Hermes config key：enabled、scope/consent、adoption_mode、policy_revision、证据/输出/积压上限、deadline、私有保留规则。缺少真实数据保留规则时不能启用采证；真实保留期待批准，不擅自设无限期。未终态/未知 intent 必须保留恢复依据，清理不能级联删除原生记忆或会话。

公开日志仅安全 ID/状态/理由码/真实 usage，不输出 source 文本、before/after、路径、provider 原始异常或 prompt。私有记录也不默认存隐藏推理或全部 completion。SDK ctx.llm 不返回密钥，Garden 不读 auth/.env。脱敏是辅助手段，不是数据同意的替代或绝对隐私保证。

辅助模型的实际供应商/模型/可能路由也属于数据同意，ctx.llm 不是“不外发”的保证；无法核实获准路由就不发送。V1 不承诺额外 persona 调用自动继承原生 managed-local GPU 的 idle/preempt 机制；若采用本地辅助路由，须先验证资源/抢占约束，否则该路由不启用。前台不等待 Garden 结果不代表零资源争用。

无新增证据零模型；合格批次最多一次逻辑人格调用、最多一个 persona 候选；宿主物理重试/usage 单列。不声明未测费用、token、延迟或跨平台性能上限。新回合一律不强迫人格变化；容量/预算不足记录 skipped_budget/capacity，不延迟前台等待人格完成。

## 14. 设计决策与实施边界

- 已接受方向：Hermes-first、松柏首例、Python 重写、旧 Go 仅参考、单一原生权威。
- 本稿选择的拟议技术主线：原生 review 生命周期驱动 + 投影证据 + 有界 persona completion + 薄插件治理；手动入口只是控制/调试，cron 不是主时钟。
- D01：采纳模式/可演化片段；D02：并发保障支持范围；D03：最小宿主 bridge 的范围/维护及未来 core 修改授权，均待所有者决定。
- 所有产品代码、测试代码、真实数据、模型费用、配置、插件启用、SOUL 修改和新会话试验仍未授权。详细设计不等于任何 Story Ready。

详见 [开发准入](DEVELOPMENT-READINESS.md)、[验收矩阵](CONFORMANCE.md) 与 [执行清单](plans/index.md)。
