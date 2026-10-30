# Source snapshot: Issue #14

Historical source input; read with current ADR-0019, not an independent instruction.

URL: https://github.com/ProjectViVy/laputa/issues/14

## 背景与目标

新 Vivy 已经接入完整的 MemPalace 能力，Laputa 没有必要为了接入再复制一套长期记忆引擎。我们希望按“多、快、好、省”重新裁剪 Laputa：保留人格治理和演化的价值，同时让不同 Agent 可以复用其规范与工作流。

目标是将 Laputa 定位为**人格与记忆治理的规范和协议**，具体实现贴合宿主 Agent。Garden 提供可复用的语料收集、更新和治理工作流，也可以在后续提供统一管理入口。

## 方向

- **人格**：明确人格的权威来源、读取和修改方式，以及注入人格上下文的边界。新 Vivy 倾向使用精简的 Rust 原生实现，在主 Agent 的提示词中提供人格根层。
- **长期记忆**：由可替换的引擎负责。新 Vivy 当前使用完整的 MemPalace；将来可以研究 Mem0 等后端，协议不应把某一种引擎或数据格式变成普遍前提。替换不等于自动具备数据迁移或能力等价。
- **短期 Recap**：优先由 Agent 或其 harness 负责。规范说明所有权及接入方式，复用宿主现有的上下文、压缩和持久化机制。
- **Garden 闭环**：收集语料/证据 → 调用选定的 Agent 或宿主已有演化流程 → 生成候选更新 → 按宿主的写入策略应用 → 检查持久化和实际生效状态 → 保存结果与必要证据。候选生成本身不是闭环完成。
- **原生适配**：针对 Hermes、OpenClaw 等 Agent，先研究其现有的人格文件、记忆与自主进化机制，优先更新已有权威文件或使用其支持的接口，避免建立第二套人格副本或互相竞争的演化链路。
- **新 Vivy 的 AutoDream**：延续此前决策，候选更新通过演化链路自动采纳，不以逐项人工审核作为常态。其他宿主遵循各自的写入策略。
- Garden Web 控制台及其未来管理 Vivy 内部 Laputa 的方式保留为后续讨论。

## 与现有架构的协调

本次讨论稿已上传至 [设计分支及决策记录](https://github.com/ProjectViVy/laputa/blob/48ec533a076fe868b21afacdfde2b598eea97c6d/docs/architecture/0016-laputa-protocol-and-native-agent-governance.md)。它基于较早的本地快照，**尚未合并到最新 main**。

远端 main 已包含 [ADR-0016：可嵌入模块化单体](https://github.com/ProjectViVy/laputa/blob/main/docs/architecture/0016-laputa-embeddable-modular-monolith.md)、ADR-0017 和 [ADR-0018：VIVY 上下文与 Recap 策略](https://github.com/ProjectViVy/laputa/blob/main/docs/architecture/0018-vivy-unified-context-and-recap-policy.md)。整理进主线前需要：

1. 在最新 main 上重新整理设计稿，避免 ADR 编号冲突，并逐项说明对 ADR-0016/0017/0018 的继承、调整与待决问题。
2. 区分已有 Go 实现的合同和可跨宿主复用的规范；保留已验证实现作为证据，不把它直接当成新架构已完成。
3. 明确与宿主原生 Recap、现有角色/权限边界及人格修改策略的关系；不要为统一接口制造第二套权威数据。
4. 检查现有未提交工作区的改动，在不覆盖这些改动的前提下推进设计和后续实现。

## 待设计和验证

- 最小规范包含哪些语义，哪些属于宿主适配器；是否需要网络传输及其形式。
- 人格表示、读写和权限、并发冲突、版本/历史、运行中何时加载新状态。
- 后端能力发现与不支持操作的表达；长期记忆替换和迁移是否需要另立项目。
- 语料范围、Agent 调用、失败恢复、重复执行、更新回读，以及宿主已经内置演化链路时如何复用。
- 选择一个真实宿主完成最小纵向验证，再决定公共 SDK、Web 控制台和其他适配器的范围。

本 issue 记录架构方向和后续工作；不代表这些适配、后端替换或工作流已经实现。
