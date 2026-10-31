# 详细架构：共享 Markdown 人格源 + Hermes 受管进化与组装

修订 `2026-10-31-markdown-shared-persona-v3`。依据所有者最新决定及“公共目录指本机共享目录”的确认，替换 v2。只改设计，`implementation_authorized: false`；未创建共享目录/真实人格文件，未改运行实例。

## 1. 已冻结的方向

- 首期仅 Hermes/松柏；Python 重写，旧 Go/DIVA/VIVY 只参考架构与经验。
- **使命不能自动新增、删除、改写或重新解释成另一使命，必须取得用户明确授权。** 权限、管理员身份、隐私红线也不能由人格候选扩大。
- 自动进化只覆盖已声明的可演化人格/表达/自我认识段；使命不在可写段内。默认采用这一有界自动模式，candidate_only 只是调试/关闭写入选项，不再要求每轮人工批准普通人格变化。
- **本机共享目录必须有一份人格 Markdown 编辑源。** 不是公开 GitHub，不意味着给所有 OS 用户写权限。
- **纯 Markdown + 确定性组装**；不建 Laputa 数据库、JSON 运行台账、持久队列、游标索引、通用状态引擎或第二反思时钟。
- 并发先轻量化：一个运行实例/一个 persona writer，模型与人工交互在写入临界区之外；准备完整内容后短时核验、一次替换每个目标文件。暂不解决任意外部编辑器竞写问题，冲突能检测到就拒绝，不冒充全局 CAS。
- 接受最小宿主扩展的设计；实现、安装、启用、真实数据/费用与写入仍另行授权。
- 未来 Garden 控制台可以查看/编辑，但不在本期：不新增 Web/API/控制台数据库，也不为它提前造服务层。

## 2. 人格源和运行入口不是两份可编辑权威

选择**单向组装**：

```text
本机共享 <shared_root>/personas/songbai/PERSONA.md
    ├─ 使命 / 用户授权边界：自动流程只读、逐字保留
    └─ 人格 / 表达 / 自我认识：有界自动演化
                         +
当前 home 的私有 HOST.md（可选、用户批准的宿主补充）
                         ↓ 纯函数组装，无模型参与
当前 home 的原生 SOUL.md（生成的运行入口）
                         ↓ Hermes 原生加载 / 会话快照
指定新会话使用的人格
```

**PERSONA.md 是人格编辑源，SOUL.md 是 Hermes 唯一的运行人格入口。** 取消 v2 中“SOUL 同时是唯一编辑源”的决定。不是两个文件各改一遍或双向同步，也不额外注入 Laputa system prompt。

shared_root 的实际绝对路径在部署时由用户批准；本轮只定义逻辑路径，不随意创建 /Users/Shared 或搬运真实 SOUL。按 persona_id 分目录，宿主绑定由受信任配置确定，不能从模型/正文读取。V1 只有一个绑定宿主负责自动写该 persona，其他 profile 不因目录共享而被授权写入。

HOST.md 拟在 `<home>/plugin-data/garden/HOST.md`，可省略；它不是第二人格，不能作为覆盖使命/权限的旁路。USER/MEMORY/skills 仍归 Hermes 原生机制，不被拼进人格源。首次接管现有 SOUL 要先确认哪些内容进入共享源、哪些留私有宿主补充；未经许可不能复制真实资料或自动覆盖未接管文件。

## 3. Markdown 组织与使命保护

不复活旧七/八文件 roster。PERSONA.md 一份文件，建议明确区分“使命”“授权与红线”“人格”“表达风格”“自我认识”段，实际初始内容由用户批准。

程序识别稳定段边界：自动 proposal 只能替换允许段的正文，不能改标题、重划段边界或加入重复的保护段。组装前后使命/授权段精确字节比较必须相等。边界缺失/重复/无法解析时拒绝自动写入，不自动修复或编造使命。未赋使命也不能由模型补出使命。

用户批准的使命版本摘要固定在受信任宿主绑定中，而不是由 PERSONA 正文或模型自填。源文件使命与已批准 pin 不匹配时，自动流程连组装传播都拒绝；更新使命及其批准 pin 都须用户授权。这样不把“别人已改了共享源”当成新使命许可，不引入数据库。

需要修改使命的提案只报告 needs_user_authorization；用户授权必须绑定具体 persona、当前版本与拟议改动，不能来自模型字段、昵称、引用或“你可以自主探索”之类泛化推断。把改变使命的要求塞入“人格”段也不能作为合法自动变化；明显越界拒绝，不确定转用户处理。结构检查不宣称能证明全部自然语言语义安全。

历史依据：归档 `9acc999301d5d575da2cea902a1c6d213763f381:docs/architecture/0017-diva-cognitive-contract-deltas.md:19-47` 明确 no generated Mission 和 mission first/verbatim；只继承适用原则，不继承旧协议/数据库/文件数量。

## 4. 原生 review 是主生命周期，不另开后台工作器

源码 `agent/background_review.py:75-99,129-167` 显示 run token 挂在父 Agent；前台新回合/停止可请求取消，有限等待后继续前台。`run_agent.py:806-827` 挂出 review 后台线程。它不是“主进程已经停住”，也不是全局文件锁。

据此取消 v2 的“end 事件落盘后另起惰性 worker”方案。拟议宿主接缝把 **persona 阶段放进同一个受管 review 任务，完成该阶段后才释放该任务的取消/完成管理**：

```text
原生 review 准入、排队与执行
    ↓ 原生完成 + 未取消 + 获准本地证据
同一 review 任务内调用 Garden persona 阶段
    ↓ 读取人格源快照、至多一次有界归纳
严格段级校验 / no_change / needs_user_authorization
    ↓ 确定性组装、提交前再检查取消和版本
短提交：源文件替换 → Hermes SOUL 替换 → 回读
    ↓ 最小 Markdown 结果；新会话生效另验
原生任务统一结束与清理
```

取消/失败/无范围时不做人格分析；原生 memory/skills no_action 不等于 persona no_change，仍允许判断人格。Garden 不能假定把代码放到现有 on_end 后就自动受父任务取消；需要明确改造 run token/最终清理边界，详见 [接缝](HERMES-REVIEW-BRIDGE.md)。

模型调用期间不持文件锁。新前台回合/stop/unload 可让该阶段失去提交资格；SDK 网络请求可能无法立即撤销，迟到结果不得开始文件提交。取消若发生在短提交开始之后，不宣称已撤销已发生写入：完成/报告真实结果，不盲目回滚。不得用会超时脱离的 observer 线程池承载写入阶段。

## 5. 证据与归纳

只接获准的本地当前回合 user/final assistant 可见文本；不要原生完整 history/digest、工具原文、隐藏推理、系统提示或未批准的 session_search 回填。QQ/群/远程证据入口尚无本期可信绑定验收，仍默认关闭，不把正文 JSON 当身份。

原生 review fork 自带完整快照/摘要，因此 persona 阶段使用单独的有界 ctx.llm completion；只传获准证据与必要的可演化人格上下文，不把 HOST.md 或整篇私有 SOUL 自动外发。最多一次逻辑归纳、最多一个人格 patch；空/重复/no_change 合法。宿主物理重试/usage 如实记录，不假称一次物理请求或零额外费用。

证据、候选和去重键主要在本次任务内存中存在，不做长期人格素材数据库。用户事实/记忆/技能只分流，既有原生机制继续写入；Garden 不再写一遍。反例可以否定或修订旧认识，不只不断增加人格内容。

模型路由/供应商也属于数据同意；额外辅助调用可能仍争用本地 GPU，取消标志不是网络硬中断或资源隔离，启用具体路由前需验证。前台无需等人格更新完成不代表零资源争用。

## 6. 组件：删掉通用状态层

均为拟新增，未创建产品代码：

| 模块 | 职责 |
| --- | --- |
| `domain.py` | 可信绑定、源快照、受限 patch、结果类型；纯逻辑 |
| `host.py` | SDK、获准路径/证据、受管阶段与取消适配 |
| `stage.py` | 在原生 review 任务内串接各步骤，无独立线程/工作器 |
| `markdown.py` | PERSONA 段解析/保护校验、最小 Markdown 回执读写；不是通用 store |
| `proposal.py` | 一次有界人格归纳，无 writer |
| `review.py` | 使命/权限冻结、允许段、出处/反例与自动采纳判断 |
| `assemble.py` | PERSONA + 可选 HOST → SOUL 的确定性纯函数 |
| `apply.py` | 单写者短提交、回读、部分失败检查与有条件回滚 |
| `activation.py` | 指定新会话实际请求身份块核验 |
| `__init__.py` / `plugin.yaml` / 操作技能 | 注册/卸载与本地检查/调试；不隐式启动任务 |

不再设计 store.py/lifecycle.py 的持久 RunRecord/ControlState、work.lock/control.lock、durable queue、全事件落盘或跨重启任务重放。运行配置仍走 Hermes 支持的配置生命周期，不把配置写成新业务数据库。

## 7. 短提交与并发支持范围

1. 读 source/HOST/当前 SOUL 的快照，在锁外完成模型归纳、校验和完整新文本组装；没新变化不写。
2. 进入同运行实例的 persona 提交互斥区；检查取消/许可、源/HOST/SOUL 版本未漂移、使命保护、输出可被原生完整加载。
3. 准备完整 Markdown 临时文件与必要恢复材料；记录一笔 APPLY.md。临界区不调用模型、不等人工、不做全库扫描。
4. 对 PERSONA.md 一次原子替换；对 SOUL.md 一次原子替换；分别独立回读，更新最小结果说明。临时文件在目标文件系统准备，具体 fsync/replace 在目标 OS 验证。
5. 漂移可检测则 conflict，拒绝覆盖；同实例只允许一个 writer。**不承诺最后一次比较与 replace 之间防住非合作外部写者**，不要求把日常演化变为长维护窗口，也不建全写者 CAS 服务。

“瞬时”指短时一次性文件提交，不保证未测的毫秒延迟。两个不同文件不能靠两次 rename 变成一笔跨文件原子事务；源先更新而 SOUL 失败必须明确报告 source_saved_host_pending。

## 8. 最小恢复材料：仍然是 Markdown

只在实际应用时使用当前 home 私有 `<home>/plugin-data/garden/last-apply/`：APPLY.md、SOURCE.before.md、SOURCE.after.md、SOUL.before.md、SOUL.after.md。它们是最后一笔恢复材料，不是人格输入、任务库或长期证据仓库；不保存全聊天/隐藏推理/完整模型返回。

APPLY.md 用有限固定字段记录目标绑定、版本摘要、阶段与结果，可带简短安全理由和真实 usage。未完成/不明操作不得被下一笔覆盖；先检查/处置。已完成后只保留必要的最后备份，真实保留规则在启用前批准；不建立遍历历史的队列/索引。

| 观察到的源 / SOUL | 结论 |
| --- | --- |
| 都等于 after | persisted_verified；仍不等于会话生效 |
| 源 after、SOUL before | source_saved_host_pending；仅可在许可/版本仍有效时重组装 SOUL，不再次调用模型或重写源 |
| 都等于 before | observed_before；不据此自动获得重新提交许可 |
| 源 before、SOUL after，或任一为第三种内容 | inconsistent/conflict；停止，不能猜测或覆盖 |

未留下恢复材料的进程骤死不伪称可完整恢复；不做 outbox/exactly-once。回滚也是新授权操作，必须检查当前源和 SOUL 仍匹配该笔 after；有后续编辑就拒绝，不能把用户手改覆盖掉。

## 9. 生效与首次接管

共享源更新、SOUL 更新、Hermes 某个会话加载新内容是三个状态。首次真实接管要批准源目录、初始人格/使命、可选 HOST、原 SOUL 的迁移/接管方案和绑定；本轮没有读取/搬运真实人格。

首期仍用获准新验收会话核验实际请求身份块，不改旧 prompt、DB、工具集，不强制压缩/重启当前 QQ 会话。截断/overlay/错 home 不得算生效；缺观察字段返回 unverified。停用插件不会自动恢复旧人格。

## 10. 决策与授权状态

D01 已接受：有界自动人格演化，使命变化必须用户明确授权。D02 已接受：单写者短时文件提交、尽力检测漂移，当前不做完整并发协议。D03 已接受：最小宿主扩展方向。D04 已确认：人格文件在本机共享目录，不公开到互联网；技术上采用单向编辑源→宿主组装输出。

这些是设计决定，不是产品开工/真实数据/模型费用/配置/安装/写入授权。控制台入口未来再做。本期唯一执行计划见 [plans/index](plans/index.md)，字段见 [CONTRACTS](CONTRACTS.md)，验收见 [CONFORMANCE](CONFORMANCE.md)。
