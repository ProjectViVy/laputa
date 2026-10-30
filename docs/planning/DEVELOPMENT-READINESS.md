# 开发准入与计划状态

## 已完成的准备，不是产品实现

- 旧架构归档、看板收口与main切换：见 [preparation-verification.json](preparation-verification.json)。
- 本轮已完成固定源码的三路只读审计及父代理关键路径复核：见 [HERMES-SOURCE-AUDIT.md](HERMES-SOURCE-AUDIT.md)。
- 源码存在能力不等于真实宿主适配验收；未读真实人格/聊天/数据库，未运行产品测试。

## 当前判断

**计划完整性：CONCERNS；产品执行：NO-GO。** `implementation_authorized: false`。

首期Hermes/松柏自身人格沉淀与进化。采用薄原生插件、显式本地入口和原生SOUL权威，不做跨宿主平台。

## 唯一执行清单

原始准备阶段S01–S06是占位研究顺序，不再作为产品执行任务。源代码核对结果已收敛为 [plans/index.md](plans/index.md) 中G01–G06；每个Story有独立计划，状态/依赖的机器源是 [status.json](status.json)。不要并行维护第二套任务表。

[PRD](PRD.md)定义R01–R08；[架构骨架](ARCHITECTURE-SPINE.md)负责设计选择；[CONTRACTS.md](CONTRACTS.md)负责内部类型/状态；[CONFORMANCE.md](CONFORMANCE.md)负责验收映射。

## P0关卡

| 决策/授权 | 影响 | 当前状态 |
| --- | --- | --- |
| D01：显式确认采纳还是有限自动采纳 | G03/G04/G06 | 待所有者冻结，默认不自动写 |
| D02：独占维护窗口还是跨原生写者CAS | G04/G06，关系R05验收范围 | 待所有者决定；不以原子rename冒充CAS |
| 明确产品开工授权 | 全部产品Story | 未授权 |
| 真实persona/证据与模型使用范围、私有位置、保留策略 | G02真实调用、G06 | 未授权；合成规划不授予真实数据使用 |
| 真正的新验收会话与当前运行版本绑定 | G05/G06 | 待运行验证；不直接改当前QQ会话 |

这些关卡不阻止继续完善设计，但不能把Blocked计划交给实现代理“自行决定”。

## 交付监督

松柏负责拆解、派工、审查和最终验收；当前只允许只读审计子代理。开工后按DAG和文件所有权分派；独立Story通过不自动等于完整闭环通过。

本轮不安装依赖、不生成产品代码/测试、不启用插件、Observer或cron，不写人格/记忆/运行配置，不发布或push计划，不操作远端Issue/PR。
