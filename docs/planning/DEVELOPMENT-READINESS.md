# 开发准入：详细设计完成，执行仍 NO-GO

修订 `2026-10-31-native-review-v2`；设计 CONCERNS；`implementation_authorized: false`。仓库整理证据见 [preparation-verification](preparation-verification.json)，源码事实见 [HERMES-SOURCE-AUDIT](HERMES-SOURCE-AUDIT.md)。静态文档校验不是产品测试。

## 唯一设计与计划

Python 重写；旧 Go 仅参考。主路径为原生 review 实际 attempt 生命周期 + 获准可见证据投影 + 有界人格归纳 + 原生 SOUL 应用/新会话验证。不保留手动候选主流程，不另加反思 Agent/时钟/记忆引擎。

[PRD](PRD.md) R01–R09 → [详细架构](ARCHITECTURE-SPINE.md) / [bridge](HERMES-REVIEW-BRIDGE.md) → [内部合同](CONTRACTS.md) → [G01–G06](plans/index.md) → [C01–C32](CONFORMANCE.md)。状态和 DAG 的唯一机器源为 [status.json](status.json)；旧 S01–S06/旧版 G 职责均不再派工。

## 关卡

| 决策/授权 | 需要冻结的内容 | 影响 |
| --- | --- | --- |
| D01（待所有者） | candidate_only/人工采纳/有限自动；允许片段、成熟/否决、策略撤回 | G03 采纳、G04/G06；未定只产候选 |
| D02（待所有者） | 独占维护窗口及明确 R05 缩窄，或另行设计原生共享版本写入 | G04/G06；原子替换不是 CAS |
| D03（待所有者） | 最小原生 bridge 范围/维护方式；未来隔离 core 修改授权 | G02；当前 core 不改，不承诺已有事件 |
| 产品开工 | 精确允许代码/测试/依赖及隔离目录 | 全部 Story；当前无 Ready |
| 真实数据/预算 | 本地入口、可见证据与基线读取、用途、模型、私有位置/保留 | 真实模型与首次采集；QQ/群/远程默认拒绝 |
| 安装/启用/写入/激活 | 确切宿主版本、配置、D01/D02 条件、新验收 session | 真实松柏闭环；不能改当前 QQ 工作会话来验收 |

这些不阻止继续设计，但实现者不得自行填空。D03 的最小 bridge 不自动授权 D02 的全写者合同，更不代表 core/生产部署获准。

## 执行监督

主链 G01 → G02 → G03 → G04 → G06；并行分支 G01 → G05 → G06（严格依赖见计划）。松柏管理合同、派工、审查、集成与最终验证。共享文件按所有权串行，实施先测后写；前置证据/授权未到不派实现者。

本轮仅 Markdown/状态文档与其静态验证，不创建产品代码、测试或依赖，不读真实人格/历史/凭据，不调真实模型，不启用 hook/plugin/cron，不改运行配置或远端看板，不 push。
