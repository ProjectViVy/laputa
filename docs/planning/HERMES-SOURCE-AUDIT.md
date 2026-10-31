# Hermes 源码核对与设计约束

## 证据边界

- 固定源码：`bf0203a794a07aac3ff764972ab9087baa4e4f4e`，detached、工作区干净；Laputa 整理基线：`425a15b02d4975954362a91515f95709f46d3bfb`。
- 以下位置相对 Hermes 源码根。结论来自静态源码核对，不是当前进程、真实人格或适配运行验收。
- 启动器已定位到该源码树；未据此声称运行中进程所有已载入模块与磁盘完全相同。
- 非敏感配置查询 `memory.provider` 返回空；不能继承旧机器“已启用 MemPalace”的结论，也不能仅据此证明全部记忆插件均未运行。
- 未读取真实 SOUL、USER、MEMORY、会话/数据库、日志或凭证；未初始化 provider、运行产品测试、更新 Hermes 或调用真实模型。

## 可复用能力与边界

| 编号 | 已核实的静态事实 | 源码证据 | 设计影响 |
| --- | --- | --- | --- |
| E01 | 原生人格从已解析 home 的 SOUL 读取；不从 cwd 取人格 | `agent/prompt_builder.py:1540-1588`; `agent/system_prompt.py:225-252` | 绑定真实 home；只有原生 SOUL 是运行人格权威 |
| E02 | SOUL 在首个身份槽，缺失才使用默认人格 | `agent/system_prompt.py:550-556` | 不另注入一份 Garden 人格 |
| E03 | 自有人格扫描是警告保留，加载还会截断 | `agent/prompt_builder.py:83-109`; `agent/prompt_builder.py:1572-1585` | 扫描不是授权；不能静默截断后宣称全部候选已生效 |
| E04 | system prompt 可从 DB 恢复旧字节，内存 Agent 也会复用 | `agent/conversation_loop.py:817-897`; `gateway/session_execution.py:191-234` | 新对象或重启不保证新人格激活 |
| E05 | 真正新会话重置缓存；正常压缩可重建，seeded 维护分支例外 | `gateway/slash_commands_session.py:196-241`; `agent/conversation_compression.py:3082-3137` | 首期选择显式新验收会话；不强制压缩工作会话来刷新 |
| E06 | 当前官方文档有 refresh-prompts，本地 tracked Python 搜索及 parser/dispatch 均未发现 | `hermes_cli/subcommands/sessions.py:31-337`; `hermes_cli/sessions_cmd.py:1268` | 能力检测返回 unsupported；不能调用不存在的命令或以 repair-prompts 替换 |
| E07 | 原生 SOUL PUT 原子替换，但没有 expected-version 或激活回执 | `hermes_cli/web_routers/profiles.py:1015-1034` | 原子写不是 CAS；真实写入是独立 P0 关卡 |
| E08 | 项目指令文件审批门在 basename 检查前豁免自身 Hermes home | `tools/file_tools_write_guards.py:211-253` | 不能把项目 AGENTS 审批经验套给自有 SOUL 自动采纳 |
| E09 | 内建 memory 写盘与提示快照分离，有锁内重读和单文件原子写 | `tools/memory_tool_store.py:221-296`; `tools/memory_tool_store.py:338-367`; `tools/memory_tool_store.py:675-700` | 复用原生事实存储，不许声称跨 SOUL/USER/MEMORY 的事务 |
| E10 | 外部 provider 与内建 store 共存；静态提示和逐轮召回不同 | `agent/memory_provider.py:84-157`; `agent/memory_manager.py:269-299` | Garden 不新增 provider、镜像或长期记忆引擎 |
| E11 | 原生 review/Curator 已有技能经验维护，但 Curator LLM 路径不是人格进化 | `agent/background_review.py:1088-1195`; `agent/curator.py:1105-1136` | 技能更新归原生机制；Garden 只补人格候选治理缺口 |
| E12 | 未知 hook 名只告警仍注册，Observer 无通用副作用沙箱 | `hermes_cli/plugins.py:943-965`; `hermes_cli/plugins_dispatch.py:231-294` | 注册成功不等于可触发；自动主路径须先实现并验收同任务 persona stage，见 E20–E27 |
| E13 | 回合 payload 带原文/浅拷贝历史，API legacy 字段还能含系统提示 | `agent/turn_finalizer.py:453-479`; `agent/turn_api_request.py:43-88`; `agent/api_request_hooks.py:60-115` | 脱敏不等于采集同意；禁止整段遥测/历史进入语料 |
| E14 | session surface 来自 task-local context；profile 名称不是可信所有者授权 | `hermes_cli/plugin_session.py:17-50`; `hermes_cli/plugins.py:399-408` | 绑定解析 home 和入口；未知上下文拒绝 |
| E15 | 没配置 slash 管理名单时限制可关闭；transport 与 runtime profile 要分开 | `gateway/slash_access.py:86-104`; `gateway/slash_access.py:137-158` | `/garden` 不是天然 owner-only；首期不开放 QQ 入口 |
| E16 | ctx.llm 复用宿主凭证和辅助路由，不是完整 Agent 回合 | `agent/plugin_llm.py:446-580` | 显式传入获准证据，模型仅生成候选，不给写入工具 |
| E17 | 缺 jsonschema 时结构化校验会跳过 | `agent/plugin_llm.py:316-334` | 业务层必须独立严校验，不能只信 parsed |
| E18 | spawn_task 是进程内任务；host 模式取消/上下文另有边界 | `hermes_cli/plugins.py:424-434`; `hermes_cli/plugin_host_child.py:213-229`; `hermes_cli/plugin_host.py:105-127` | persona 留在同一原生 review 受管任务内，不另开 worker/持久任务 |
| E19 | session_search 可跨 profile 按 ID 读库；查询过滤不等于采集授权 | `tools/session_search_tool.py:601-650` | 首切片只接显式选中的当前可见文本，不默认扫历史 |

## 原生 review 主路径补充核对（2026-10-31，同一固定源码；v3 调整设计影响）

| 编号 | 静态事实 | 源码证据 | 对当前设计的影响 |
| --- | --- | --- | --- |
| E20 | 原生 memory/skill review 在回合条件满足后触发，skip/interrupt 可抑制 | `agent/turn_finalizer.py:778-792` | 不另建梦境时钟；不是每条聊天都触发 |
| E21 | extra_tools 只放行父工具表中已注册工具，dispatch 白名单不保证调用 | `agent/background_review.py:1088-1128` | 不把工具白名单当阶段生命周期；使用正式受管 persona stage |
| E22 | 同路由 review 继承完整 snapshot，异路由用 digest | `agent/background_review.py:1183-1193` | 人格证据另做获准可见投影，不导出 review 全上下文；有新增批次才做有界 persona completion |
| E23 | 有 try 前早退；动作摘要仅 actions 非空发布；finish 可在不同清理点发生 | `agent/background_review.py:1202-1207,1231-1246,1282-1300` | 现有 provider finally 不是完整 persona 阶段结束；扩展受管任务完成边界，不能另起 end 后 worker |
| E24 | deferred queue 为进程内；同 session 最新 snapshot 合并；重排有限 | `agent/review_idle_queue.py:1-10,97-107,163-178`; `run_agent.py:831-850` | 未执行队列项不冒充 attempt；只复用原生队列，不加持久任务/跨崩溃 exactly-once |
| E25 | kwargs 在 enqueue 传递，实际执行准备 token 并传播线程 context | `run_agent.py:775-827` | 必须冻结请求身份/投影，经队列传递，不能在后台读取变化后的当前身份 |
| E26 | 父 Agent 持 run token；cancel 置位并对已运行 fork 发中断，前台等待有界 | `agent/background_review.py:75-99,129-167` | 受管任务内 persona 阶段须保留取消/完成边界；前台不无限等待，取消不是文件锁 |
| E27 | finish 在 provider finally 和外层 cleanup 多处调用，具有幂等清理性质 | `agent/background_review.py:1202-1207,1290-1300` | 在内层释放父 token 后另开 callback 不会自动继承取消；G02 必须明确调整任务边界 |

E20–E27 是直接源码复核，未跑原生测试或读取真实会话。详细合同中的 persona stage/投影/能力宣告尚不存在；不能反过来当成现场事实。

## 当前风险与已接受范围

| 级别 | 事实/风险 | 当前处置 |
| --- | --- | --- |
| P0 | 使命不能由自动人格流程修改；共享源可能被外部改动 | MISSION.MD 全文件精确比较 + 获准使命 pin，缺/变则拒绝，具体变化用户授权 |
| P0 | 新共享编辑源/原生组装输出关系与首次 SOUL 接管 | G01/G06 固定源/home/迁移边界，未获真实授权不读/搬/写 |
| P1 | 现有 token 在 provider 阶段结束便可能释放，end 后新 worker 不受其管理 | G02 调整同任务阶段/取消与清理，不新增工作器；尚无运行验收 |
| P1 | 原子 replace 不是全写者 CAS，也非两个文件 ACID | 所有者已接受轻量短提交范围；可检测漂移拒绝、部分失败报告，任意外部竞态后置 |
| P1 | 会话可恢复旧 prompt，新源/新 SOUL 不等于旧会话生效 | G05 验指定新会话的实际身份块，不改当前 QQ prompt/DB |
| P1 | 原始 telemetry/辅助模型输入可含私有内容 | 明确当前可见投影与路由同意，HOST/全 SOUL 不自动外发 |
| P2 | JSON parsed/模型自述不等于业务安全证明 | 独立段/授权/来源校验，运行仅最小 Markdown 材料，无 JSON 业务台账 |

这些是静态事实/设计风险，不是本轮动态漏洞复现。v2 的 JSON RunRecord/ControlState、持久队列、独立 worker、全写者 CAS 待决和 SOUL 唯一编辑源均已被最新所有者决定替换。

## 旧讨论的定向核对

归档 ADR-0012 `9acc999301d5d575da2cea902a1c6d213763f381:docs/architecture/0012-laputa-markdown-clean-break.md:22-34` 明确同目录七份 uppercase 文件、Markdown bodies only；后续 ADR-0017 加 MISSION。此次按所有者纠正恢复独立正文/文件级策略，不再要求 v3 内部分块、标题锚点或源内元数据。

归档 `9acc999301d5d575da2cea902a1c6d213763f381:docs/architecture/0017-diva-cognitive-contract-deltas.md:19-47` 写明 No generated Mission、mission first/verbatim 和使命版本校验。当前只继承使命不自动生成/改写、用户授权与逐字组装原则，恢复多文件正文组织，但不机械继承旧 schema、固定字数/截断规则或数据库。没有重跑旧产品验收。

## 官方原文入口

- https://hermes-agent.nousresearch.com/docs/user-guide/which-file-does-what
- https://hermes-agent.nousresearch.com/docs/user-guide/features/personality
- https://hermes-agent.nousresearch.com/docs/user-guide/sessions#refresh-prompts-after-a-profile-update
- https://hermes-agent.nousresearch.com/docs/developer-guide/observer-hooks
- https://hermes-agent.nousresearch.com/docs/developer-guide/plugin-llm-access

官方文档是当前产品说明；本次实现设计锁定上述本地提交。文档/实现不一致之处单列，不升级、不回填虚构功能。