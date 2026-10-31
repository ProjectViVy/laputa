# ADR-0020：原生受管 review、共享 Markdown 人格与组装

日期 2026-10-31；修订 `2026-10-31-plain-multifile-persona-v4`；状态：所有者设计原则接受，产品实施未授权。承接 [ADR-0019](0019-hermes-first-persona-evolution.md)，细节唯一源为 [详细架构](../planning/ARCHITECTURE-SPINE.md)。v2 独立 worker/JSON 台账方案退出当前计划。

## 所有者决定

- 新 Python 代码重写，旧 Go/DIVA/VIVY 仅参考架构/经验。
- 使命不允许自动改写，必须用户明确授权；普通人格按文件权限有界自动演化，源正文不需要分块。
- 并发先轻量，短时一次性编辑文件，暂不做全写者 CAS 或长期维护窗口。
- 运行数据纯 Markdown + 组装，不建 Laputa 数据库或 JSON 业务台账。
- Laputa 必须有本机共享目录的人格文件；已澄清不是公开 GitHub。
- 最小宿主扩展接受；控制台查看/编辑入口未来可能提供，但不在本期。

## 技术收敛

共享人格目录中的 MISSION/IDENTITY/RELATIONSHIP/REDLINE/DREAM/DARK 等独立纯文档是源，Hermes SOUL.md 为确定性组装输出；可选 HOST.md 保存获准私有宿主补充，无双向同步/额外人格注入。首次接管/真实资料移动另行批准。

persona 阶段留在同一个原生 review 的受管任务内，继承主任务取消/有界前台优先，不能在 end 后另起独立工作器。只处理获准可见证据；有需要时一次有界辅助人格归纳，模型/交互在文件临界区外。

共享源与宿主 SOUL 各一次替换，不声称跨文件 ACID；只保留最后一笔必要 Markdown 恢复材料。检测漂移拒绝，外部编辑器最后检查后的竞态暂不解决，不能拿瞬时编辑当全局文件锁。

## 已否决/退出

v3 的单 PERSONA.md、固定正文分块、段解析/区域 patch 已被所有者纠正取消。权限与组装规则由程序/宿主绑定按文件管理，不要求人类维护机器格式。USER/WORLD 保留历史职责参考，但首期不自动拼入 SOUL/同步原生事实库。


持久 RunRecord/ControlState、JSON 索引、durable queue、独立 persona worker、第二梦境时钟、v2 将 SOUL 同时作为唯一编辑源的设计。仓库 status.json 仍只是计划元数据，不作为运行存储。

归档 ADR-0017 的 mission verbatim/no generated Mission 只作原则证据，恢复独立文档布局，但不照搬旧 FrozenCore schema、固定字数/截断规则或旧数据库。

## 授权

接受设计不等于开工。当前产品代码、测试、依赖/core 修改、真实数据/模型费用、配置/安装/启用、源目录初始化、SOUL 接管/修改和新验收会话均未授权。D01/D02/D03 不再重复等待用户选择，部署与运行参数在实际启用前确认。