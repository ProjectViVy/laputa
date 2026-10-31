# 最小 Hermes 扩展：受管 persona 阶段（拟议 v3）

修订 `2026-10-31-plain-multifile-persona-v4`。所有者已接受最小宿主扩展的设计方向；core/测试修改、安装和启用尚未授权。下述接口均拟议，不是当前 SDK 已有功能。

## 1. 已核实源码

固定 `bf0203a794a07aac3ff764972ab9087baa4e4f4e`：

| 位置 | 事实 / 影响 |
| --- | --- |
| `agent/background_review.py:75-99` | run token 挂父 Agent，finish 标记 request_done 并清指针 |
| `agent/background_review.py:129-167` | 前台/stop/session teardown 可取消，等待有界，超时仍继续前台；不是全局文件锁 |
| `agent/background_review.py:1183-1207` | provider fork 阶段的 finally 已调用 finish，之后才收消息/释放客户端 |
| `agent/background_review.py:1231-1246,1282-1300` | 有早退、动作摘要条件与 finally 清理；不能把现有 summary 当完整生命周期 |
| `run_agent.py:775-827` | 请求经 defer/now/线程上下文传递；实际 run token 先于 Thread.start |
| `agent/turn_finalizer.py:778-792` | 仍由原生门槛触发；不为 Garden 新建时钟 |

因此，“现有 review 受主任务管理”**不代表 v2 在完成事件之后另开的工作器也会被管理**。v3 删除独立 worker/持久运行状态，扩展同一个 review 的 persona 尾阶段与取消边界。

## 2. 扩展范围

| 现有文件 | 最小拟议变更 |
| --- | --- |
| `agent/turn_finalizer.py` | 原生触发时，在获准 local scope 下投影本回合 user/final assistant；无同意只元数据/无正文 |
| `run_agent.py` | 冻结入口/证据/请求绑定，经 queue/requeue 传递到 actual attempt，不取变化后的当前身份 |
| `agent/background_review.py` | 正常原生阶段结束且未取消时，在同一 worker 内进入 persona callback；完成释放要覆盖全部 provider-capable 阶段；早退/取消/错误统一清理 |
| `hermes_cli/plugins.py` | 明确单 persona stage 注册/能力版本/卸载语义，绑定当前 profile 与授权源；不得只注册无 fire site 的 hook |

不改原生触发门槛、memory/skills 责任，不做 monkeypatch、新进程、新队列或通用事件总线。实现仅在后续批准的隔离源码树，不能直接修改当前承载 QQ 的安装树。

## 3. 拟议 stage 合同

建议能力名 `background_review.persona_stage.v1`，建议注册边界 `register_background_review_persona_stage(callback)`。当前 SDK 不存在这一合同。

HostStageContext：实际 attempt/request/review 标识、可信解析 home/本地入口/parent turn、获准 VisibleEvidence、授权绑定/策略版本/用户批准使命 pin、取消查询/短提交准入 guard。模型不能提供此对象；只有一个已绑定 persona writer 消费该阶段。

宿主需明确 opt-in 的可见投影许可，默认无；core 不读取 Garden 私有数据库（本期也没有该数据库）。源路径/同意来自已批准的宿主配置/注册上下文。无授权不先广播正文再决定；不把完整 snapshot/digest、raw completion、focus、reasoning、SDK/agent 对象当证据导出。可信进程内插件并非安全沙箱。

```text
actual review admission
  → native memory/skills provider 阶段
  → 检查实际 completed/failed/interrupted 与 cancel token
  → completed（包括 no_action）且 local evidence 获准：同步 persona stage
       → snapshot → bounded proposal → validate → assemble
       → try_enter_short_commit → source replace → SOUL replace → readback
  → persona outcome（no_change/needs_user_authorization/updated/failed/cancelled）
  → 全阶段 final cleanup / 完成通知 / 释放父任务管理
```

不得仅依据函数正常 return/有 actions 推断成功；实际结果不明为 unknown，不分析/应用人格。重排保持 request/review 绑定但 actual attempt 新建；被合并未执行快照不冒充 attempt。

## 4. 取消与前台优先

保留 run token 的取消效力直到 persona 阶段退出；不能在原生 fork request finally 就把后续 persona 阶段从父任务管理中移除。具体拆分 provider client 清理与完整任务完成由实现冻结并用测试证明，不把已释放的 fork 当成可中断辅助网络请求。

stage 的网络超时/取消尽力而为；前台等待保持有界，不能为了等 persona 完成挂起主对话。取消先于提交准入：无文件修改；准入后的短提交：可完成并报告真实写入，不承诺取消能回滚已写文件。卸载/替换注册使旧 callback 失去新提交资格。

不能用 observer 超时线程池、spawn_task 或另起 worker 承载 persona 写入；否则 late side effect 无法冒充父任务已取消。模型阶段无文件锁，文件提交阶段无模型/交互等待。

## 5. 失败/版本与验收

错误隔离：Garden 失败不推翻前台回答/已完成原生技能维护；分别报告 native_outcome 与 persona_outcome。共享人格/host 部分提交见 CONTRACTS，仅保留最后一次必要 Markdown 恢复材料，不建事件 outbox/台账/持久任务重放。

升级必须证明实际 callback fire site、所用 profile、上下文传递、取消/有界前台、早退、schema/能力声明与组装写入边界。缺功能时 persona automatic unsupported，禁止假装启用；本地调试不冒充自动闭环。当前没有运行验收，core 不改。