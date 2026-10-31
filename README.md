# Laputa · Garden

> 当前为详细设计阶段；没有交付新运行实现、安装插件或启用自动进化。

首期仅 **Hermes / AGENT-Songbai（松柏）自身的人格沉淀与进化**。Python 重写，旧 Go/DIVA/VIVY 仅参考设计经验。

## 已接受方向

- 本机共享目录先以 `personas/<persona_id>/` 按角色隔离，再按职责保存多份独立纯文本/Markdown 文件；程序按文件权限和固定顺序组装为原生 SOUL.md。正文无必需标题/分隔符/元数据；共享不是公开 GitHub，无双向同步/第二人格注入。
- IDENTITY/RELATIONSHIP/DREAM/DARK 等获准文件可自动进化；**使命的新增、删除、改写必须用户明确授权**，不能扩大管理员/工具/隐私权限。
- **纯 Markdown + 组装，不建 Laputa 数据库、JSON 运行台账、持久任务队列或独立 persona worker。** 现有 Hermes 原生会话/记忆存储不更换。
- 同一原生 review 受管任务内运行 persona 阶段；最小宿主扩展方向已接受，代码/安装尚未授权。
- 文件写入采用单实例单 writer 的短时一次性提交，检测漂移拒绝；当前不承诺任意外部写者 CAS/跨文件 ACID。
- 未来 Garden 控制台可能支持查看/编辑，但不在本期，不预建 Web/API/服务/数据库。

## 入口

1. [当前详细决定 ADR-0020](docs/architecture/0020-native-review-driven-garden.md)
2. [PRD](docs/planning/PRD.md)
3. [详细架构](docs/planning/ARCHITECTURE-SPINE.md)
4. [内部合同](docs/planning/CONTRACTS.md) / [最小宿主阶段](docs/planning/HERMES-REVIEW-BRIDGE.md)
5. [Story 计划](docs/planning/plans/index.md) / [开发准入](docs/planning/DEVELOPMENT-READINESS.md)
6. [文档索引](docs/README.md) / [归档](docs/archive/README.md)

修订 `2026-10-31-plain-multifile-persona-v4`。所有产品 Story `implementation_authorized=false`。目录/真实人格初始化与迁移、模型/数据、代码/测试/core、配置/安装/启用/写入和新会话验收仍待各自授权；本轮只改文档，不创建真实人格源。

## 历史

[ADR-0019](docs/architecture/0019-hermes-first-persona-evolution.md) 保留重启方向。旧 main `9acc999301d5d575da2cea902a1c6d213763f381`、标签 `legacy/pre-reset-20261031` 与 `archive/2026-10-31/` 完整保留。旧测试通过不等于新实现完成，既有部署/其他机器不会随此设计改变。