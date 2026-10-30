# 验收矩阵（全部待执行）

每行既是需求覆盖，也是未来测试/真实试验的证据要求。源码审计不勾选运行验收。

| ID | 需求 | Story | 输入/风险 | 预期与证据 |
| --- | --- | --- | --- | --- |
| C01 | R01,R04 | G01 | 缺home/surface、symlink越界、跨profile | fail-closed，无模型/写入；fixture回执 |
| C02 | R02,R08 | G02 | 未同意、system/reasoning/tool原文、任意session/profile ID | 拒绝或字段级不接纳，零未批准语料外发 |
| C03 | R03,R07 | G02 | 合法persona/用户事实/记忆/skill/临时内容 | 正确分类且引据仅来自批次；非persona不写SOUL |
| C04 | R03,R07 | G02 | 重复、无稳定增量、反例推翻 | no_change或拒绝，不为“成长”造人格漂移 |
| C05 | R04 | G03 | 模型自称主人、候选附许可、远程命令、过期批准 | 拒绝；许可由可信入口绑定，不由内容生成 |
| C06 | R03,R04 | G02,G03 | 无jsonschema、未知字段、伪造出处、超预算 | 独立校验拒绝；不能只信parsed |
| C07 | R05 | G04 | 生成后原生人格被修改、不同写者并发 | 不覆盖合法更新；D02未定前blocked，不以合作锁模拟全局通过 |
| C08 | R05,R06 | G04 | 相同operation重放、同ID不同after | 前者查询既有结果，后者冲突；无重复目标写入 |
| C09 | R06 | G04 | intent失败、写后receipt失败、崩溃三态 | intent失败不写；未知结果只对账；current与before/after之外则拒绝 |
| C10 | R06 | G04 | 后续编辑后回滚、仅局部回滚成功 | 拒绝覆盖新内容；回滚新操作且重新验证，不虚报全部恢复 |
| C11 | R06 | G05 | persisted但旧会话恢复DB快照 | 不报effective；不改旧prompt、历史或工具集 |
| C12 | R01,R06 | G05 | 新合成验收session、正常加载/截断/overlay | 在实际请求身份块上核验；截断/不符即失败，非模型自述 |
| C13 | R06,R07 | G05 | 本地无refresh-prompts、seeded压缩分支 | unsupported/保持pending；不调用repair、不强制compress/reset工作会话 |
| C14 | R07,R08 | G01,G06 | register/status/reject、回调/网络/任务审计 | 无隐式模型、hook订阅或任务；私有文本不进公开日志 |
| C15 | R01-R08 | G06 | 获准真实松柏闭环 | 合法证据→候选→采纳→SOUL回读→新会话有效→安全恢复；私有证据仅公开脱敏结果 |

## 验收层次

1. 纯函数/合成fixture测试。
2. 固定Hermes源码的隔离profile集成，不接生产store/模型。
3. 所有者单独授权的真实松柏persona、模型调用与新会话试验。

三者分开报告；不能把第一层代替第三层。没有给任何待执行测试伪造pass、耗时或覆盖率。
