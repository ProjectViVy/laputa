# Laputa · Garden

> 新主线：开发准备阶段。没有交付新运行时、适配器或自动进化服务。

**Laputa** 是人格与记忆治理的规范与协议；**Garden** 是把证据转化为可验证更新的治理闭环。

**首期宿主只有 Hermes，首个真实对象是 AGENT-Songbai（松柏）自身的人格沉淀与进化系统。** 不先做 Vivy 适配，不以跨宿主框架或统一 Web 控制台作为首期前提。

## 当前边界

- 新代码按 Python/Hermes 最新决策重写；旧 Go 仅作架构/经验参考，不机械移植。
- 复用 Hermes 原生人格、记忆、技能及上下文机制，不制造第二套权威文件或长期记忆引擎。
- 详细设计采用原生 background review 生命周期驱动；最小宿主接缝尚待授权，手动为控制/调试入口，cron 不是第二反思时钟。
- 人格沉淀不是把全部聊天写入 SOUL；人格、用户事实、长期记忆和程序性技能必须分流。
- 进化闭环必须区分：证据收集、候选生成、按宿主策略应用、持久化回读、运行时生效验证及结果留存。
- 当前只允许仓库整理、归档和设计准备；产品实现、测试代码、运行时配置、真实人格/记忆修改与定时任务尚未授权。
- 项目所有者：mastwet；主责维护与验收：AGENT-Songbai。产品方向和高风险边界仍由所有者决定。

## 从这里开始

1. [当前决策 ADR-0019](docs/architecture/0019-hermes-first-persona-evolution.md)
2. [首期 PRD](docs/planning/PRD.md)
3. [详细架构](docs/planning/ARCHITECTURE-SPINE.md)
4. [开发准备、路线与准入](docs/planning/DEVELOPMENT-READINESS.md)
5. [归档入口与恢复](docs/archive/README.md)
6. [文档索引](docs/README.md)

## 旧架构

原 Go `laputa/`、`mentle/`、`garden/`、Console、旧 ADR、执行计划及其测试记录均退出新主线，在 Git 归档中完整保留。

- 旧 main 固定提交：`9acc999301d5d575da2cea902a1c6d213763f381`
- 固定标签：`legacy/pre-reset-20261031`
- 归档命名空间：`archive/2026-10-31/`

历史代码和旧测试通过记录不证明新架构已经实现。原有已部署服务、其他仓库和其他机器的工作区不在此次整理范围内。
详细设计修订：`2026-10-31-native-review-v2`；参见 [ADR-0020](docs/architecture/0020-native-review-driven-garden.md) 与 [拟议 Hermes review 接缝](docs/planning/HERMES-REVIEW-BRIDGE.md)。D01–D03 待决，实施/运行仍未授权。
