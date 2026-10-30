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
| E12 | 未知 hook 名只告警仍注册，Observer 无通用副作用沙箱 | `hermes_cli/plugins.py:943-965`; `hermes_cli/plugins_dispatch.py:231-294` | 注册成功不等于可触发；首切片不自动订阅 hook |
| E13 | 回合 payload 带原文/浅拷贝历史，API legacy 字段还能含系统提示 | `agent/turn_finalizer.py:453-479`; `agent/turn_api_request.py:43-88`; `agent/api_request_hooks.py:60-115` | 脱敏不等于采集同意；禁止整段遥测/历史进入语料 |
| E14 | session surface 来自 task-local context；profile 名称不是可信所有者授权 | `hermes_cli/plugin_session.py:17-50`; `hermes_cli/plugins.py:399-408` | 绑定解析 home 和入口；未知上下文拒绝 |
| E15 | 没配置 slash 管理名单时限制可关闭；transport 与 runtime profile 要分开 | `gateway/slash_access.py:86-104`; `gateway/slash_access.py:137-158` | `/garden` 不是天然 owner-only；首期不开放 QQ 入口 |
| E16 | ctx.llm 复用宿主凭证和辅助路由，不是完整 Agent 回合 | `agent/plugin_llm.py:446-580` | 显式传入获准证据，模型仅生成候选，不给写入工具 |
| E17 | 缺 jsonschema 时结构化校验会跳过 | `agent/plugin_llm.py:316-334` | 业务层必须独立严校验，不能只信 parsed |
| E18 | spawn_task 是进程内任务；host 模式取消/上下文另有边界 | `hermes_cli/plugins.py:424-434`; `hermes_cli/plugin_host_child.py:213-229`; `hermes_cli/plugin_host.py:105-127` | 不增加后台服务，不声称跨重启持久执行 |
| E19 | session_search 可跨 profile 按 ID 读库；查询过滤不等于采集授权 | `tools/session_search_tool.py:601-650` | 首切片只接显式选中的当前可见文本，不默认扫历史 |

## 风险排序与处置

| 级别 | 问题 | 处置 |
| --- | --- | --- |
| P0 | SOUL 多种原生写者没有统一 CAS；合作锁不能防任意编辑器竞写 | G04 保持 Blocked，先决定受控独占窗口还是上游统一写入改造 |
| P0 | 插件/命令注册、profile 名字和源文本都不是业务所有者授权 | G01/G03 严格绑定；首期仅受信任本地操作者，远程入口关闭 |
| P0 | 候选采纳策略未由所有者冻结；现有 gate 可默认关闭或有豁免 | 自动采纳默认为不可用，不据“可写文件”推导授权 |
| P1 | 新旧提示快照可能不同；官方新文档不等于本地已实现 | G05 只承诺能力检测和新会话验收，不偷偷动现有会话 |
| P1 | 遥测字符串/原始字段仍可能有隐私或隐藏内容 | 首切片不接 Observer/API/stream，明确 allowlist 字段 |
| P1 | 辅助结构化返回与后台取消不能直接当安全/事务边界 | 独立校验；未知结果对账，禁止自动重放写操作 |
| P2 | 技能 pin/ownership/ledger 不能泛化为完整人格回滚 | 不接管 Curator，不把技能回滚当 SOUL 回滚 |

以上是设计风险和源码边界，不是本轮动态复现的漏洞报告。

## 官方原文入口

- https://hermes-agent.nousresearch.com/docs/user-guide/which-file-does-what
- https://hermes-agent.nousresearch.com/docs/user-guide/features/personality
- https://hermes-agent.nousresearch.com/docs/user-guide/sessions#refresh-prompts-after-a-profile-update
- https://hermes-agent.nousresearch.com/docs/developer-guide/observer-hooks
- https://hermes-agent.nousresearch.com/docs/developer-guide/plugin-llm-access

官方文档是当前产品说明；本次实现设计锁定上述本地提交。文档/实现不一致之处单列，不升级、不回填虚构功能。
