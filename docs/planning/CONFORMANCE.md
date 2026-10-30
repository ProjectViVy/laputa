# 验收矩阵：全部待执行

修订 `2026-10-31-native-review-v2`。本表是未来证据要求，不是测试结果。C01–C15 保留身份并更新 Story 映射，C16–C32 补全原生触发/生命周期/撤权与恢复。

| ID | 需求 | Story | 输入/风险 | 预期与证据 |
| --- | --- | --- | --- | --- |
| C01 | R01,R04 | G01 | 缺 home/surface/同意、symlink 越界、跨 profile | fail-closed；无模型/目标写入；可信绑定不取正文/default |
| C02 | R02,R08 | G02 | 未同意或夹带 system/tool 原文/reasoning/全历史 | 只有当前获准可见投影；缺绑定不采正文，不做历史回填 |
| C03 | R03,R07 | G03 | 人格/用户事实/记忆/技能/临时内容 | 正确分流；非 persona 不写 SOUL/其他存储 |
| C04 | R03,R07 | G03 | 空、重复、无稳定增量、角色扮演、反例 | 合法 no_new_evidence/no_change/拒绝，不强制造人格漂移 |
| C05 | R04 | G03 | 正文自称 owner、候选自带批准、超时/旧许可 | 拒绝；可信 decision 精确绑定候选/基线/策略版本 |
| C06 | R02,R04 | G03 | 无 jsonschema、未知字段、伪造引据、超预算 | 独立业务校验拒绝，不能仅信 SDK parsed |
| C07 | R05 | G04 | 生成后编辑、原生入口/外部写者竞态 | 按 D02 明确支持范围验收；缩窄 R05 需 owner 同意；不得假装全局 CAS |
| C08 | R05,R06 | G04 | 同 operation 重复、同 ID 不同 after/decision | 只查询既有结果/冲突；无重复目标写入 |
| C09 | R06 | G04 | intent 失败、目标成功回执失败、崩溃三态 | 写前失败不动目标；未知只对账 exact before/after/current |
| C10 | R06 | G04 | 后续编辑后回滚、局部恢复 | 拒绝覆盖新版本；回滚新操作/许可/激活证明 |
| C11 | R06 | G05 | 文件已改但旧 session 恢复 DB 快照 | 不报 effective，不修改旧 prompt/历史/工具 |
| C12 | R01,R06 | G05 | 新合成 session、正常加载/截断/overlay | 实际请求身份块核验，缺失/截断/冲突不能报成功 |
| C13 | R06,R07 | G05 | 无 refresh-prompts、seeded 压缩、缺探针字段 | unsupported/unverified；不调用 repair/强制 reset 工作会话 |
| C14 | R07,R08 | G01,G06 | register/status/reject 和安全日志 | 零隐式模型/采证/任务；enable 后才处理获准事件，日志无正文 |
| C15 | R01,R02,R03,R04,R05,R06,R07,R08,R09 | G06 | 获准真实松柏自动闭环 | 真实原生 review→有据候选→采纳→SOUL 回读→真正新 session 有效；安全回执 |
| C16 | R09 | G02 | 实际 review changed/no_action 正常完成 | 各有完整终态；no_action 仍可人格判断，不偷换 persona no_change |
| C17 | R09 | G02 | try 前取消/不支持 provider/线程启动失败/运行异常 | 每实际 attempt 正常控制流只结算一次；失败不应用人格 |
| C18 | R02,R09 | G02 | idle defer/coalesce/preempt/requeue | 触发时绑定不漂移；同请求新 attempt；未执行快照不冒充已运行 |
| C19 | R09 | G02,G06 | 未知 hook 能注册但无 fire site/schema 不匹配 | automatic unsupported、apply disabled；capability 与实际 fire sites 都验证 |
| C20 | R08,R09 | G01,G02 | 重复 start/end、同 key 不同事件内容 | 幂等/冲突；错 attempt 不结束新任务；无 start 的 end 隔离 |
| C21 | R02,R08 | G02,G03 | 原生 fork 带全历史、digest、异常或原始 completion | 不进入 Garden persona 输入/日志；只传 allowlist 投影与获准人格目标上下文 |
| C22 | R07 | G03 | 新证据/重复/空/超时、宿主重试 | 无新材料零调用；合格批次至多一次逻辑人格调用；物理 usage 如实报告 |
| C23 | R06,R09 | G02,G06 | 事件落盘前骤死、start 后骤死、模型中途骤死 | 不承诺全事件恢复；缺 end 不猜成功，analysis_unknown 不自动重发 |
| C24 | R04,R09 | G02,G04,G06 | pause/unload/revoke 后迟到回调/模型返回 | generation 失效，旧结果不能提交；不假称网络调用已取消 |
| C25 | R06,R09 | G06 | restart/recover、锁仍被活工作器持有 | 只恢复有 durable 依据的记录；不强抢活锁、不重写 unknown 目标 |
| C26 | R03,R04 | G03,G04 | 拟议可演化片段、片段外修改/权限提升/缺标记 | 未批准片段不自动补建；外部字节精确不变，越界需人工/拒绝 |
| C27 | R08 | G01,G06 | 保留期/容量、未终态 intent、损坏 schema | 未批准保留规则不启用采集；恢复材料不静默清理，不级联删原生记忆 |
| C28 | R04,R09 | G02,G06 | 后台线程 context 缺失、profile 路由变化、QQ origin 正文 | 信触发时绑定；不退 default owner，未知/远程拒绝 |
| C29 | R03 | G03 | 同一认识被反例推翻、替代旧候选、单次情绪 | 有据修订/拒绝而非只累加；supersedes 不能跨 scope/代替权限 |
| C30 | R07,R09 | G02,G06 | 回调长任务、队列为空、预算/积压不足 | callback 不调模型/等待人/写 SOUL；无新反思时钟，容量不足可见跳过 |
| C31 | R01,R06 | G05,G06 | 旧 activation receipt 与后续新 SOUL 版本 | receipt 只证明绑定 session/version，不冒领最新版本或全部会话 |
| C32 | R04,R05 | G01,G03,G04 | 模型挂起期间撤权、提交与撤权并发 | 控制操作不等 work.lock；control 临界区确定提交/撤权次序，不宣称追溯撤销已写目标 |

## 证据层次

1. 纯函数/合成 fixture/fake completion：类型、状态、权限、故障注入。
2. 固定且获准的隔离 Hermes 源码 + 独立合成 home + fake provider：真实 fire sites、线程/队列、原生加载/请求路径，不接生产库。
3. 所有者单独批准真实资料、模型预算、运行启用、D01/D02 条件与新验收 session 后：真实松柏闭环。

分别报告，第一/二层不能代替第三层。D02 未决时 C07 保持 blocked，不把缩窄范围悄悄算通过。没有 fake pass、耗时、覆盖率或已实现声明。全部要求由 [status](status.json) / [Story](plans/index.md) 关联。
