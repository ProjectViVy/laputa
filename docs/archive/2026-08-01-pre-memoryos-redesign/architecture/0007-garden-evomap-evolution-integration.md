# ADR-0007: Garden 与 EvoMap/Evolver 的自主进化集成边界

> **Status:** proposed - 待松本评审
> **Date:** 2026-08-01
> **Scope:** 定义 Garden MemoryOS 如何接入 EvoMap/Evolver 的自主进化能力，以及 Laputa、Mentle、Garden、Evolver、宿主适配器之间的职责边界。
> **Does not decide:** EvoMap/Evolver 内部算法、GEP 协议未来版本、公共 Hub 的产品政策、Hermes/Claude Code/Codex 各自的 skill 文件格式细节。
> **Related:** [ADR-0004](./0004-garden-memory-orchestration-proposal.md)、[ADR-0005](./0005-memoryos-fast-recall-and-upsp-study.md)、[ADR-0006](./0006-laputa-lifecycle-and-mentle-index-contract.md)

---

## 0. Decision

Garden **不内置自主进化引擎**，也不复制或重新实现 EvoMap/Evolver 的 Gene、Capsule、演化策略和资产处理机制。

Garden 直接接入 EvoMap/Evolver 作为外部 evolution engine，但保留 MemoryOS 的治理边界、证据边界、人格边界、用户审批、审计、回滚和宿主分发决策。

```text
Laputa
  Agent 人格、authority、生命周期、策略、审批、审计、回滚

Mentle
  个人工作材料、执行证据、来源溯源、分类、检索、关系

Garden
  来源接入、活动编排、证据筛选、Evolver 调用、结果编排、宿主分发

EvoMap / Evolver
  GEP 资产处理、Gene/Capsule 机制、经验演化、策略生成、可选验证/发布流程

Hermes / Claude Code / Codex / OpenClaw
  宿主侧执行、结果回流、产物安装与运行时验证
```

核心定位：

```text
Garden = governed EvoMap client / MemoryOS evolution boundary
Garden != evolution engine
Evolver = external evolution engine
```

“直接使用 EvoMap 能力”成立，但“把 EvoMap/Evolver 变成 Garden 的内部核心模块”不成立。

---

## 1. 背景与已验证事实

### 1.1 Garden 的产品边界

Garden 指 Laputa-Garden / MemoryOS，而不是 agent-diva 内部的轻量 MemoryProvider。MemoryOS 的中心是：

```text
具有连续人格、受治理记忆和工作上下文的 Agent
  -> 能理解、定位和调用个人全部工作信息
```

现有分层已经在 ADR-0005 与 ADR-0006 中确定：

- Laputa 是 authority、人格和治理层；
- Mentle 是受治理的分类记忆材料土壤层；
- Garden 是来源接入、活动编排、召回和 ContextView 组装层；
- Fast Recall 与 Deep Recall 分离；
- candidate discovery、evidence read、final context 严格分离。

自主进化应在此基础上作为**外部能力接入**，不能改变既有 authority 方向。

### 1.2 Evolver 研究基线

本 ADR 以本地只读调查的 Evolver `v1.93.0` 为研究基线：

```text
Repository: C:/Users/Administrator/.workspace/evolver-survey
Commit: f2bc3264d2b3e0b55d7bf8a548ac9412b51a0a83
Package: @evomap/evolver 1.93.0
Node engine: >=22.12
Declared license: GPL-3.0-or-later
```

已核实的可读模块包括：

- `src/gep/schemas/gene.js`：Gene 的 category、signals、strategy、validation、constraints、preconditions、learning history 等字段；
- `src/gep/skill2gep.js`：从 `SKILL.md + real execution trace` 反向生成 Gene/Capsule 的机制、来源与验证约束；
- `src/gep/skill2gepAudit.js`：对隐藏执行输出中的路径、flag、结构化标识符和数字等潜在私有文本做机械泄漏审计；
- `src/gep/skill2recipes.js`：要求可运行 validation 才能进入可组合 Recipe；
- `src/gep/skillPublisher.js`：将 Gene 转换为面向人的 `SKILL.md`，并支持 Hub 资产发布语义。

`src/evolve.js`、`src/gep/solidify.js`、`src/gep/skillDistiller.js` 等文件当前存在混淆，不能作为 Garden 推断其内部完整行为的审计依据。本 ADR 只采用可读源码能够证明的机制；README 的未验证声明不作为实现契约。

### 1.3 许可证与供应链事实

Evolver 当前 `package.json` 声明 `GPL-3.0-or-later`，README 同时公告后续许可证策略可能转为 Source-Available。许可证、依赖和 Hub 数据边界尚未完成独立审查前：

- 不复制 Evolver 源码到 Garden；
- 不把 Evolver 作为 Garden bundled dependency；
- 不在 Garden 进程内静态链接或嵌入 Evolver；
- 不把公共 EvoMap Hub 作为 Garden 的默认数据出口。

这不是永久禁止任何依赖，而是当前集成阶段的合规前置条件。

---

## 2. 职责边界

### 2.1 Laputa：不可委托的职责

以下职责必须由 Laputa 保留，不能委托给 Evolver：

| 职责 | 说明 |
|---|---|
| Agent identity | `IDENTITY.MD`、人格边界和单一人格原则 |
| Authority | 哪些记忆、技能、偏好、策略可以成为持久权威 |
| User profile | 用户期望和用户拥有的约束 |
| Promotion approval | 从 candidate/proposal 晋升为持久能力的决定 |
| Installation approval | 是否安装到 Hermes、Claude Code、Codex 等宿主 |
| Publication approval | 是否允许离开本机、上传 Hub 或共享 |
| Audit | 输入证据、演化结果、审批、安装和回滚记录 |
| Rollback | 撤销已批准的技能版本或宿主安装 |
| Policy | 私密性、范围、风险、工具权限和发布策略 |

Evolver 生成的任何 Gene、Capsule、SkillDraft 或 Recipe 都只是候选资产，不自动获得 Laputa authority。

### 2.2 Mentle：材料和证据职责

Mentle 负责：

- 保存会话、代码、文档、Obsidian 和报告等工作材料；
- 保存执行结果、验证输出和来源引用；
- 提供 candidate discovery、evidence read 和 lineage 查询所需的材料；
- 维护 `source_uri`、revision、content hash、observed time 和同步状态；
- 支持 Garden 为 Evolution Evidence Bundle 选取最小必要证据。

Mentle 不负责：

- 把某段经验自动认定为长期能力；
- 直接修改 Laputa authority files；
- 决定是否把资产安装到宿主；
- 代替用户批准技能晋升或外发。

### 2.3 Garden：集成和编排职责

Garden 负责：

1. 从 Mentle/Laputa/宿主运行记录中发现可能可泛化的经验；
2. 进行 scope、权限、隐私、来源和证据完整性筛选；
3. 把最小化后的 Evidence Bundle 提交给 Evolver；
4. 接收并规范化 Gene、Capsule、SkillDraft、validation report 等候选结果；
5. 将候选结果放入 Laputa proposal/review 流程；
6. 调用宿主适配器生成 Hermes、Claude Code、Codex 等具体产物；
7. 记录安装、验证、启用、停用、回滚和失败事件；
8. 让同一个 Agent 人格在多个宿主上共享能力，而不是产生多个独立人格。

Garden 不负责重新实现 Evolver 的演化策略、GEP 资产算法或公共 Hub 机制。

### 2.4 Evolver：可以委托的职责

Evolver 可以承担：

- GEP Gene/Capsule schema 和资产处理；
- 从信号和真实执行轨迹中提炼候选策略；
- 依据 `repair / optimize / innovate / explore` 等 category 处理候选；
- 生成或转换 Gene、Capsule、Recipe 和 SkillDraft；
- 执行其支持的策略验证、质量检查和归因记录；
- 通过 MCP、CLI 或 sidecar 暴露演化能力；
- 在明确允许时处理 EvoMap Hub 的资产交换。

Evolver 不应获得 Garden 的：

- Laputa authority file 写权限；
- 用户 profile、SOUL、IDENTITY 等人格材料的任意写权限；
- Mentle 私有 corpus 的全量读取权限；
- 宿主安装和启用权限；
- 默认公共 Hub 上传权限。

---

## 3. Evolution Evidence Bundle

Garden 交给 Evolver 的输入不是 Mentle 全库、完整人格或完整聊天记录，而是最小化、可追溯、受策略约束的证据包。

建议逻辑结构：

```json
{
  "bundle_id": "evidence_...",
  "scope": "project:garden",
  "trigger": {
    "kind": "failure|correction|successful_pattern|user_request",
    "summary": "可泛化的工作问题摘要"
  },
  "outcome": {
    "status": "passed|failed|partial",
    "summary": "结果摘要"
  },
  "execution": {
    "trace_ref": "mentle://evidence/...",
    "validation_ref": "mentle://validation/...",
    "blast_radius": "declared"
  },
  "evidence_refs": [
    "mentle://material/..."
  ],
  "provenance": {
    "source_revision": "...",
    "content_hash": "...",
    "observed_at": "..."
  },
  "policy": {
    "allow_local_evolution": true,
    "allow_host_export": false,
    "allow_hub_publish": false,
    "redaction_required": true
  }
}
```

实际实现不要求立即采用此 JSON；这里冻结的是语义边界。

必须满足：

- 每个证据引用都能回到 Mentle source/material；
- 私人路径、项目秘密、token、内部标识符和人格内容在出 Garden 前经过过滤；
- 没有真实执行轨迹时，最多生成 Gene-only candidate，不得伪造成功 Capsule；
- validation 不覆盖声明的 Gene strategy 时，结果必须降级并标记缺口；
- evidence bundle 本身可审计、可撤回、可按 policy 重新处理。

---

## 4. Candidate、Proposal 与 Authority 的分层

```text
Mentle evidence
  -> Garden Evolution Evidence Bundle
  -> Evolver Gene/Capsule/SkillDraft candidate
  -> Garden normalization + leakage/policy report
  -> Laputa EvolutionProposal
  -> user review / governed approval
  -> Laputa authority apply
  -> host artifact build
  -> host validation
  -> explicit install/enable
```

必须区分以下对象：

| 对象 | 权威级别 | 默认行为 |
|---|---|---|
| Evidence Bundle | 证据材料 | 可撤回、可审计，不等于能力 |
| Gene | Evolver 候选策略资产 | 不自动安装、不自动成为 authority |
| Capsule | 带真实执行证据的候选资产 | 需验证和审批 |
| SkillDraft | 可读技能草稿 | 需宿主适配和审批 |
| EvolutionProposal | Laputa 治理提案 | 用户审核或明确治理流程 |
| Approved Skill | Laputa 批准的持久能力 | 受版本、范围和回滚管理 |
| Host Artifact | 某宿主的安装产物 | 必须独立验证和显式启用 |

Evolver 的结果不能直接写入：

- `SOUL.MD`、`IDENTITY.MD`、`USERPROFILE.MD`、`LONGMEM.MD`；
- Garden 的正式 Skill Registry authority；
- Hermes、Claude Code、Codex 的安装目录；
- 公共 EvoMap Hub。

所有这些写入或外发动作必须回到 Garden/Laputa 的治理路径。

---

## 5. 接入形态

### 5.1 第一优先：本地 sidecar / MCP

第一阶段优先采用进程外的本地 Evolver：

```text
Garden Evolution Adapter
  -> local Evolver MCP server or bounded CLI process
  -> local asset store
  -> normalized candidate result
```

原因：

- 隔离许可证和 Node 运行时边界；
- 避免 Evolver 的依赖污染 Garden Go 进程；
- 便于限制环境变量、文件目录、网络和 Hub 权限；
- 可替换 Evolver 版本而不改变 Garden 核心；
- 失败时 Garden 可以降级为“不执行进化”，不影响 Fast Recall。

Evolver 的 CLI/hook 模式可作为特定宿主的后续集成，但不能让宿主 hook 绕过 Garden 的 evidence、policy 和 approval spine。

### 5.2 不采用：Garden 进程内嵌

当前不采用：

```text
Garden Go process
  -> embed Evolver source/runtime/dependencies
```

除非未来完成独立许可证确认、供应链审查、API 稳定性确认和安全隔离评估，否则不把 Evolver 作为 Garden 的编译期或 bundled runtime dependency。

### 5.3 EvoMap Hub 默认关闭

Garden 的默认策略：

```text
local evolution: allowed by policy
host export: explicit approval
hub publish: disabled
hub fetch: explicit, scoped, audited
```

以下内容禁止默认发送到公共 Hub：

- 用户原始会话；
- Mentle 私有材料和完整证据；
- `SOUL.MD`、`USERPROFILE.MD`、`IDENTITY.MD`、`USER.MD`；
- 私有路径、仓库地址、token、内部服务信息；
- 未经脱敏的 validation stdout/stderr；
- 能推断个人项目、身份、关系或未公开决策的 Gene/Capsule。

即使资产已机械脱敏，是否可发布仍由 Laputa policy 和用户审批决定。机械 leakage audit 不能替代语义隐私审查。

---

## 6. 宿主适配与分发

Portable skill 和 host artifact 必须分离：

```text
Portable Skill / Gene
  宿主无关的触发条件、策略、约束、验证和来源

Host Artifact
  Hermes skill / plugin
  Claude Code plugin / skill / hook / MCP binding
  Codex skill / instruction / adapter
```

职责顺序：

```text
Evolver 生成候选策略资产
  -> Garden 规范化为 Portable Skill candidate
  -> Laputa 审批
  -> Host Adapter 渲染具体产物
  -> 在目标宿主隔离验证
  -> 用户批准安装/启用
  -> 运行结果回流 Garden
```

Host Adapter 必须记录：

- portable skill ID 和版本；
- 目标宿主及版本；
- 渲染器版本；
- 安装路径或宿主注册 ID；
- 验证命令、结果和时间；
- 失败时的清理与回滚信息。

同一个 Portable Skill 可以渲染成多个宿主产物，但每个产物都不是自动可信的复制品。宿主工具权限、hook 语义、上下文注入方式和验证标准可能不同，必须逐宿主验证。

---

## 7. Evolution Event 与审计

一次进化尝试至少应产生可关联的事件链：

```text
source evidence selected
  -> bundle sanitized
  -> evolver invocation
  -> candidate generated
  -> validation executed
  -> policy report created
  -> proposal submitted
  -> user decision
  -> authority applied
  -> artifact rendered
  -> host validated
  -> installed / rejected / rolled back
```

事件至少应能关联：

```text
request_id
bundle_id
candidate_id
proposal_id
portable_skill_id
host_artifact_id
source_refs
validation_refs
policy_decision
actor
created_at
```

进化失败不能静默丢弃。至少记录：失败阶段、错误摘要、是否可重试、是否生成 Gene-only 降级结果、是否需要用户介入。

Evolution event 是审计记录，不等于 authority mutation。任何长期变化仍必须经过 Laputa authority spine。

---

## 8. 与现有 ADR 的关系

### 8.1 对 ADR-0004 的修正

ADR-0004 中的 `Garden Skill Registry` 不应理解为 Garden 自主进化内核。其语义修正为：

```text
Garden Capability / Host Integration Registry
```

它可以登记：

- 外部 Evolver provider；
- Evolution Evidence Bundle schema 版本；
- Portable Skill candidate；
- Host Adapter；
- validation profile；
- approval/install/rollback state。

Gene/Capsule 的演化算法和资产生命周期仍属于外部 Evolver/GEP 能力域。

### 8.2 对 ADR-0005 的约束

Fast Recall 不得默认调用 Evolver。自主进化属于显式的后台/维护/用户请求能力，不进入普通上下文召回的默认热路径。

```text
Fast Recall
  不触发 evolution

Deep Recall
  可以作为 evidence discovery 输入，但不自动晋升能力

Evolution Run
  独立预算、独立 trace、独立 policy 和独立审批
```

### 8.3 对 ADR-0006 的约束

Evolver 产物可以引用 Mentle evidence，也可以在审批后成为 Laputa 管理的能力索引，但不能成为第二 authority store。

```text
Mentle = evidence/material substrate
Laputa = authority
Evolver = external evolution engine
Garden = governed orchestration
```

---

## 9. MVP 范围

第一版只做本地、显式、可审计的闭环：

```text
受控来源
  -> Garden 选择证据
  -> 生成最小 Evidence Bundle
  -> 调用本地 Evolver
  -> 接收 Gene / SkillDraft / validation result
  -> 创建 Laputa EvolutionProposal
  -> 用户审批
  -> 导出 Hermes / Claude Code / Codex 候选产物
  -> 逐宿主验证
```

第一版明确不做：

- 自动安装；
- 自动启用；
- 自动修改代码；
- 自动修改人格或用户 profile；
- 自动 STM -> LTM 或 memory -> skill 晋升；
- 公共 Hub 自动发布；
- Garden 内置第二套 Gene/Capsule 引擎；
- Evolver 常驻循环阻塞 Fast Recall；
- 跨宿主无验证直接复制 skill；
- 用热度、调用次数或相似度单独决定能力晋升。

建议实施顺序：

1. 定义 Evidence Bundle、Candidate、Proposal 和 Host Artifact 的 Garden 侧元数据；
2. 接入本地 Evolver MCP/CLI，限制目录、网络和 Hub 权限；
3. 实现本地 Gene/SkillDraft candidate normalization；
4. 接入 Laputa proposal/review/audit；
5. 实现 Hermes、Claude Code、Codex 的只读导出；
6. 增加隔离 validation 和显式安装；
7. 最后再评估 Hub fetch/publish。

---

## 10. MVP 验收标准

- Garden Fast Recall 在 Evolver 不可用时保持可用，并且不等待 Evolver；
- Evolver 只能读取 Garden 明确提交的 Evidence Bundle，不能扫描 Mentle 全库；
- 没有真实 execution trace 时不得生成“成功 Capsule”；
- validation 覆盖不足时结果必须标记 degraded 或 Gene-only；
- candidate 不会直接写入任何 Laputa authority file；
- candidate 不会直接安装到 Hermes、Claude Code 或 Codex；
- hub publish 默认关闭，且无显式 policy 时无法调用；
- 私密 source refs、路径、token 和用户人格材料不会进入默认外发 payload；
- 每次调用、验证、审批、安装和回滚都有可关联 audit event；
- 同一个 Portable Skill 导出到多个宿主时，每个 Host Artifact 都有独立版本和验证结果；
- Evolver 进程失败、超时或版本不兼容时，Garden 能返回明确的 degraded 状态；
- Evolver 的许可证和供应链审查完成前，Garden 不产生源码嵌入或 bundled dependency。

---

## 11. 未决事项

以下事项不在本 ADR 中擅自决定：

- Garden Evolution Adapter 的具体 HTTP/MCP/CLI schema；
- Laputa `EvolutionProposal` 是否新增独立 section，还是复用现有 proposal/audit 机制；
- Portable Skill 的最终规范名称和版本格式；
- 三个宿主的具体 artifact 文件布局；
- Evolver 未来许可证变更后的依赖策略；
- 是否允许从私有 EvoMap Hub 拉取经过签名的资产；
- 是否引入独立 sandbox executor，以及其 Windows 权限模型；
- 自动 activity/heartbeat 何时触发 Evolution Run。

这些事项应在完成接口设计和安全/许可证评审后，以后续 ADR 或实施计划冻结。

---

## 12. 研究来源

- `C:/Users/Administrator/.workspace/evolver-survey/package.json`
- `C:/Users/Administrator/.workspace/evolver-survey/README.zh-CN.md`
- `C:/Users/Administrator/.workspace/evolver-survey/src/gep/schemas/gene.js`
- `C:/Users/Administrator/.workspace/evolver-survey/src/gep/skill2gep.js`
- `C:/Users/Administrator/.workspace/evolver-survey/src/gep/skill2gepAudit.js`
- `C:/Users/Administrator/.workspace/evolver-survey/src/gep/skill2recipes.js`
- `C:/Users/Administrator/.workspace/evolver-survey/src/gep/skillPublisher.js`
- `C:/Users/Administrator/Desktop/garden/docs/architecture/0004-garden-memory-orchestration-proposal.md`
- `C:/Users/Administrator/Desktop/garden/docs/architecture/0005-memoryos-fast-recall-and-upsp-study.md`
- `C:/Users/Administrator/Desktop/garden/docs/architecture/0006-laputa-lifecycle-and-mentle-index-contract.md`
- `C:/Users/Administrator/Desktop/morediva/agent-diva/docs/architecture/evo-diva-architecture-2026-06-12.md`

本 ADR 只新增设计文档，不修改 Garden、Laputa、Mentle、Evolver 或宿主运行代码。
