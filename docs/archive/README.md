# 旧架构归档与恢复入口

日期：2026-10-31。所有者要求旧架构退出新主线，Issue/PR 全部收口，由松柏重新主责。

## 完整旧树

- 原 main：`9acc999301d5d575da2cea902a1c6d213763f381`
- 固定标签：`legacy/pre-reset-20261031`
- 旧 main 归档分支（已回读核验）：`archive/2026-10-31/main`
- [按固定提交浏览完整旧实现](https://github.com/ProjectViVy/laputa/tree/9acc999301d5d575da2cea902a1c6d213763f381)

该完整树保留 Go `laputa/`、`mentle/`、`garden/`、Console、旧 ADR、旧计划、脚本和既有测试记录。没有把它们复制进新主线再制造两套架构。

其他原分支保留原引用，并在 `archive/2026-10-31/<原分支名>` 建立固定归档引用。PR #9/#10/#11/#12 的 head 另外保存在 `archive/2026-10-31/pr-<编号>`，包括来自 fork 的 PR #12。

映射及完整 SHA 见 [archive-manifest.json](2026-10-31/archive-manifest.json)。旧 main 分支改名是否完成，以整理后的最终回执为准。

## 讨论与来源

- [重启前开放 Issue/PR 快照](2026-10-31/open-work-items-before-reset.json)：包含 4 个 Issue 与 4 个 PR 的正文和现有评论/评审。
- [Issue #14](2026-10-31/sources/issue-14.md)：新方向的来源，关闭不表示功能已完成。
- [48ec533 原稿](2026-10-31/sources/protocol-draft-48ec533.md)：保留原文与原编号；当前整合后的决策是 ADR-0019。

所有者后来明确的 Hermes-first / 松柏人格沉淀与进化方向由当前 ADR-0019 表达，优先于这份原稿。

## 离线恢复保障

切换前已创建 `laputa-pre-reset-20261031.bundle`，包含当时全部 Git 引用及已抓取的 PR head，执行过 `git bundle verify`。文件大小与 SHA-256 记录在归档清单中。

Bundle 和原始 API 快照保存在执行机器的所有者私有备份目录，不提交公开仓库。清单只记录文件名/哈希，不包含凭证或真实人格资料。

安全检查旧实现可在干净克隆中执行：

```sh
git fetch origin --tags
git switch -c inspect-legacy legacy/pre-reset-20261031
```

这只建立检查分支，不移动当前 main，不恢复部署，也不修改旧实例的数据。若要整体回切或恢复服务，需要另行明确授权，不 force push。

## 解释边界

旧 ADR/AGENTS/测试结果是旧提交的历史事实，不再是新主线的执行权威。关闭旧工作项是架构重启收口，不是功能验收、合并、部署或数据迁移。
