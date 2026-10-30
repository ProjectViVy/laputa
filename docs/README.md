# 当前文档索引

## 当前权威

| 文档 | 用途 | 状态 |
| --- | --- | --- |
| [ADR-0019](architecture/0019-hermes-first-persona-evolution.md) | 归档重启、职责边界、Hermes-first 决策 | 方向接受；实现未授权 |
| [PRD](planning/PRD.md) | 松柏人格沉淀与进化首期目标 | 准备稿；验收尚未执行 |
| [架构骨架](planning/ARCHITECTURE-SPINE.md) | 原生权威、证据和应用/生效边界 | 设计约束；具体接口待核实 |
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

入口存在不代表接口、热加载、写入或适配已经验收。实际版本和首期可复用能力在 H0 调研中固定。
