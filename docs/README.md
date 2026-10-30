# 当前文档索引

## 当前权威

| 文档 | 用途 | 状态 |
| --- | --- | --- |
| [ADR-0019](architecture/0019-hermes-first-persona-evolution.md) | 归档重启、职责边界、Hermes-first 决策 | 方向接受；实现未授权 |
| [PRD](planning/PRD.md) | R01–R08与首期范围 | 源码核对后计划稿；验收未执行 |
| [架构骨架](planning/ARCHITECTURE-SPINE.md) | 薄原生插件、单一权威、写入/激活边界 | CONCERNS；D01/D02待决 |
| [源码核对](planning/HERMES-SOURCE-AUDIT.md) | 固定Hermes源码的能力/风险证据 | 静态审计完成；不等于运行验收 |
| [内部合同](planning/CONTRACTS.md) | 绑定、候选、许可、应用及激活状态 | 拟议；未实现 |
| [Story计划索引](planning/plans/index.md) | G01–G06、DAG及逐项执行计划 | 全部实施未授权 |
| [验收矩阵](planning/CONFORMANCE.md) | C01–C15与需求/Story映射 | 全部待执行 |
| [开发准入](planning/DEVELOPMENT-READINESS.md) | 依赖波次、待决项、开工门槛 | 仓库准备与产品实现分开记录 |
| [阶段状态](planning/status.json) | 机器可读授权与范围 | 产品实现始终 false，直到另行授权 |
| [整理验收回执](planning/preparation-verification.json) | 归档、关闭和主线切换的真实回读证据 | 仓库准备完成；不是产品验收 |

## 历史证据

- [归档入口](archive/README.md)：旧实现、旧分支、PR 候选、备份与安全恢复方式。
- [归档清单](archive/2026-10-31/archive-manifest.json)：原始 SHA 与固定归档映射。
- [重启前开放工作快照](archive/2026-10-31/open-work-items-before-reset.json)：Issue/PR 正文及现有评论/评审。
- [Issue #14 来源快照](archive/2026-10-31/sources/issue-14.md)。
- [48ec533 设计稿原文](archive/2026-10-31/sources/protocol-draft-48ec533.md)：保留原 ADR 编号和原文，仅作来源；不与当前 ADR-0019 竞争。

## Hermes 调研入口

- [官方文档索引](https://hermes-agent.nousresearch.com/docs/llms.txt)
- [Which File Does What](https://hermes-agent.nousresearch.com/docs/user-guide/which-file-does-what)
- [Personality / SOUL](https://hermes-agent.nousresearch.com/docs/user-guide/features/personality)
- [原生 Context Files](https://hermes-agent.nousresearch.com/docs/user-guide/features/context-files)
- [原生技能 Curator](https://hermes-agent.nousresearch.com/docs/user-guide/features/curator)

源码核对基线与已查明的文档/实现差异见 HERMES-SOURCE-AUDIT；尚未做真实写入或生效验收。
