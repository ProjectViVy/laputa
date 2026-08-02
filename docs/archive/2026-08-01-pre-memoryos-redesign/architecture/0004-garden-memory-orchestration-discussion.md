# ADR-0004 原始讨论记录

> **Date**: 2026-07-15  
> **用途**: 保留 ADR-0004 形成过程中的原始需求、澄清和结论。  
> **对应提案**: [`0004-garden-memory-orchestration-proposal.md`](./0004-garden-memory-orchestration-proposal.md)  
> **说明**: 用户原始表述按讨论顺序保留；因一次消息中断而分开的两段在记录中连续展示。助手回复采用结论摘要，避免把临时误解当成正式决策。

---

## 1. 原始需求：从 session 到记忆治理

用户：

> 可以不可以帮忙设计一个架构：我们从头开始理：假设有一段session信息进来，那么garden会自动整理信息：提取关键信息作为facade存储。每一条都要进facade，我的建议是这个就整理成一条信息就得了。这是增。增的话有一个覆盖功能，例如有过时的内容，例如我谈laputa架构提到有ABC几种，最新的是A，那么会把过期的BC给删除掉了。是不是这条要走管道治理我不是很清楚。然后是删，删的话用户要求删，改也是用户要求改，查会可能比较复杂，因为可能涉及到laputa和mentle混合检索之类的，mentle也有很复杂的房间结构，最终目的是不要开放这么多的接口给hermes，将查询下放到garden。然后还有一件事，周报日报月报可能会检索到重复内容，因为可能本来就是压缩后的数据，我觉得说不定garden治理管道分为查report接口。然后还有初级和高级检索？然后至于说人格治理之类的，记忆注入之类的，都是laputa中压缩后的记录。我们现在的版本是这么设计的吗？我说到的和我们现在的设计有什么不同？你先看看

现状核对结论：

- 现版本的查询管道已实现 Laputa + Mentle hybrid/KG/timeline 的组合。
- Garden 尚未实现 session 自动整理、关键事实提取、冲突替代和治理写入。
- 当前 facade 写入基本是调用方给什么就存什么。
- 当前 rhythm 从完整 Laputa snapshot 生成报告，重复压缩风险真实存在。
- Laputa 中既有权威治理规则，也有压缩投影和派生报告，不能全部视为同一种压缩记忆。

---

## 2. 外挂记忆必须保留 CRUD

用户：

> 然后hermes应该还需要保留一个增删改查。因为这是外挂记忆库

形成的结论：

- Hermes 必须保留 Garden 级别的记忆 Create/Read/Update/Delete/List。
- 不取消 CRUD，而是隐藏 Mentle 的 wing、room、embedding、KG 等内部细节。
- Garden 当前已有 Create/Read/List/Delete，但缺少正式 PATCH、版本、soft delete、冲突关系和统一审计。
- 自动 session ingest 与手动新增最终应汇入同一 mutation pipeline，避免两套写入规则。

---

## 3. Pipeline 与 Agentic RAG 是否保留

用户：

> pipeline或者什么agentic rag等高级查询你觉得有必要保留吗

形成的结论：

- Pipeline 应保留并扩展为 Garden 的通用执行骨架，不只服务查询。
- 建议形成 `session_ingest_v1`、`memory_mutation_v1`、`context_resolve_v2`、`report_generate_v1`。
- Agentic RAG 应保留，但只作为高级查询路径，不应让所有简单查询默认承担 LLM、KG、timeline、多轮 refinement 的成本。
- Basic 默认执行 hybrid + governance + dedupe + extractive context；Advanced 才加入 planner、KG/timeline 和最多一轮 refinement。
- OpenAI planner 失败或治理禁止时应可靠降级到 RulePlanner/basic。

---

## 4. 高级查询经 HTTP，Hermes 原生工具最小化

用户：

> 我的建议是，如果涉及到高级查询那么久需要让hermes（松本）通过http请求去查，原生工具输入最好是越少越好。你觉得如何？

形成的结论：

- 赞同高级查询由 Hermes 显式发起，但不建议给模型一个通用 raw HTTP/curl 工具。
- 使用一个专用 native tool adapter：`garden_search(query)`。
- adapter 内部固定调用 Garden HTTP advanced resolve；Hermes 只提供 `query`。
- wing、room、top_k、model、planner、pipeline 名称、capabilities 都由 Garden 内部决定。
- basic context 可由宿主自动预取，不占用模型工具调用；证据不足或问题复杂时才调用高级查询。

---

## 5. Skill 概念的澄清

用户第一次说：

> 用skill

该表述一度被误解为要求立即调用现有开发文档 skill。随后用户澄清：

> 等下，停，我的意思是，用skill来治理garden高级功能，只是个想法而不是叫你用skill。

纠正后的正式结论：

- Skill 是 Garden 高级能力的治理与版本化单位，不是本次讨论要求调用的外部工作流工具。
- Pipeline 是执行引擎；Skill 声明能力、输入输出、所用 pipeline、版本和 fallback。
- 首批候选 Skill：advanced retrieval、conflict resolution、daily/weekly/monthly report、personality projection。
- Skill 可以分析和提出 mutation proposal，但不能绕过 Garden mutation pipeline 直接写 Mentle/Laputa。
- Laputa policy 管理 Skill 可用的 LLM、hybrid、KG、timeline 和读写范围。

误解处理记录：因误解临时创建的一份 `docs/dev` 文档已立即删除，没有保留业务代码或额外规划文件。

---

## 6. 讨论形成的架构原则

1. **Garden 是唯一编排入口**：Hermes 不直接操作 Mentle taxonomy 或多个 MCP 工具。
2. **CRUD 不缩水**：外挂记忆库必须允许用户直接增删改查。
3. **自动 ingest 与手动 CRUD 共用 mutation 底座**。
4. **每 session 一条主 digest**：内部允许结构化 claims，避免不可治理的大文本或过度碎片化。
5. **覆盖采用 supersede**：普通查询隐藏旧版本，历史查询仍可追溯；自动流程不做 purge。
6. **查询分级**：basic 自动、advanced 显式 HTTP；Hermes 工具输入保持最小。
7. **Pipeline 负责执行，Skill 负责高级能力封装，Laputa 负责权限治理**。
8. **报告独立管道**：按时间窗口、source IDs 和 hash 去重，不以完整旧 snapshot 无限递归压缩。
9. **Mentle 是规范记忆主存，Laputa 是治理/投影/报告，Garden 是最终裁决者**。

---

## 7. 尚未在讨论中拍板的问题

- 一 session 必须严格一条 digest，还是允许按主题拆成少数几条？
- 原始 transcript 是否长期保存和参与检索？
- 自动 supersede 的置信度与允许自动处理的字段范围。
- 删除默认 soft delete 还是用户可直接 hard delete。
- basic context 的自动注入时机与 token budget。
- `garden_search` 是否无条件 advanced，还是 Garden 可以自动降级。
- Skill registry 的权威存储位置。

这些问题保留给松本评审，不在讨论记录中伪造结论。
