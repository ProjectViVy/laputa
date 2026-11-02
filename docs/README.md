# 当前文档索引

当前修订 `2026-10-31-plain-multifile-persona-v4`。最新所有者决定优先；v2 的 JSON 台账/独立工作器/编辑源方案退出执行计划。

| 文档 | 用途 | 状态 |
| --- | --- | --- |
| [ADR-0020](architecture/0020-native-review-driven-garden.md) | 使命用户授权、共享多文档人格源、纯 Markdown、轻量写入、原生受管阶段 | 已实现并经合成验收；自动演化未开启 |
| [PRD](planning/PRD.md) | R01–R10 首期范围与控制台后置 | 当前计划 |
| [详细架构](planning/ARCHITECTURE-SPINE.md) | 不透明正文/文件级策略、单向组装、使命 pin、生命周期/短提交/部分失败 | 拟议实现，未运行 |
| [内部合同](planning/CONTRACTS.md) | 源/patch/组装/最小 Markdown 回执 | 未实现 |
| [宿主阶段](planning/HERMES-REVIEW-BRIDGE.md) | 同一个 review 任务内的 persona callback/取消 | 最小扩展方向接受，接口未实现 |
| [源码核对](planning/HERMES-SOURCE-AUDIT.md) | 固定源码与旧 Mission 讨论的定向证据 | 静态，不是运行验收 |
| [G01–G06](planning/plans/index.md) | 唯一 DAG/文件所有权/逐 Story 计划 | 全部实施未授权 |
| [验收矩阵](planning/CONFORMANCE.md) | C01–C41 / 需求映射 | 全部待执行 |
| [开发准入](planning/DEVELOPMENT-READINESS.md) | 已决定原则与未来开工/真实使用授权分开 | 无 Ready |
| [status.json](planning/status.json) | 仅仓库计划元数据，不部署为业务数据库 | 实现完成；安装为被动加载，演化关闭 |
| [安装记录](planning/INSTALLATION.md) | 宿主扩展与插件安装、重启观察、未授权边界 | 已安装未启用 |
| [ADR-0019](architecture/0019-hermes-first-persona-evolution.md) | 历史重启/Hermes-first 上位方向 | 后续细节由 ADR-0020 补充 |
| [整理证据](planning/preparation-verification.json) | 归档/主线切换回读 | 不是产品验收 |

本轮已撤销 v3 单文件分块：源文件无必需内部 schema，按文件控权；实际映射以详细架构/内部合同为准。

## 历史与官方来源

- [归档入口](archive/README.md)、[清单](archive/2026-10-31/archive-manifest.json)、[原工作项快照](archive/2026-10-31/open-work-items-before-reset.json)。归档 AGENTS/旧稿不是当前执行指令。
- https://hermes-agent.nousresearch.com/docs/llms.txt
- https://hermes-agent.nousresearch.com/docs/user-guide/which-file-does-what
- https://hermes-agent.nousresearch.com/docs/user-guide/features/personality
- https://hermes-agent.nousresearch.com/docs/developer-guide/plugin-llm-access

插件已作为被动组件安装在所有者本机默认 profile，宿主扩展已本地提交；**没有创建/搬运真实共享人格文件，没有开启自动演化，没有真实模型调用，SOUL 未改变**。源码能力、安装状态和真实验收分开记录。
角色隔离：共享目录必须是 personas/<persona_id>/ 分层；每个角色独立纯文档，绑定/使命授权/恢复按角色区分。本期只服务松柏，不做多角色运行管理。
