# 开发准入：轻量 Markdown 设计已收敛，产品执行未授权

修订 `2026-10-31-markdown-shared-persona-v3`。`implementation_authorized: false`；当前仅设计，不派实现者。

## 已定，而非继续等待用户选项

D01：普通人格允许段自动演化，使命变化需明确用户授权。D02：单实例单 writer、短时文件提交、可检测漂移拒绝，暂不做全写者 CAS/长维护窗口。D03：最小宿主扩展方向接受。D04：本机共享目录人格文件，不公开 GitHub。

技术收敛：共享 PERSONA.md 编辑源 + optional private HOST.md → 原生 SOUL.md；纯 Markdown，不做 JSON 运行台账/数据库/队列；persona 阶段留在原生 review 受管任务内，不另起工作器。控制台查看/编辑未来再做，不在本期。

## 未来实施/真实启用准入

| 范围 | 当前状态 |
| --- | --- |
| 产品代码/测试/依赖、隔离 core 修改 | 尚未授权；最小扩展被接受不等于已批准修改当前安装 |
| shared_root 实际路径、初始人格/使命 pin、HOST/现有 SOUL 接管 | 运行前明确批准；本轮无创建/迁移/读真实人格 |
| 证据用途/入口、模型供应商/预算、私有恢复/保留 | 真实使用前批准；QQ/群/远程默认拒绝 |
| 配置/安装/启用、真实源/SOUL 修改、新验收 session | 需对应运行授权，不扰动当前 QQ 会话 |
| 宿主阶段/取消/组装/加载的真实合同验收 | 尚未执行；静态设计不算通过 |

## 唯一计划

[PRD](PRD.md) R01–R10 → [架构](ARCHITECTURE-SPINE.md)/[内部合同](CONTRACTS.md)/[bridge](HERMES-REVIEW-BRIDGE.md) → [G01–G06](plans/index.md) → [验收矩阵](CONFORMANCE.md)。[status.json](status.json) 仅为仓库计划元数据，不是运行数据库。

主链 G01→G02→G03→G04→G06，分支 G01→G05→G06。文件所有权与真实前置证据按计划，未获开工不创建源码/测试。v2 独立工作器、JSON 台账/队列与 D01–D03 待决阻塞已退出当前计划。

归档整理证据仍见 [preparation-verification](preparation-verification.json)，源码事实见 [HERMES-SOURCE-AUDIT](HERMES-SOURCE-AUDIT.md)。本轮不改运行配置/真实资料/人格、core 或远端，不安装/启动，不 push；只做文档静态检查。
