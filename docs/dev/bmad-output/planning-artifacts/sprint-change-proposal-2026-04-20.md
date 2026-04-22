---
date: "2026-04-20"
project: "天空之城 (Laputa)"
change_type: "major"
trigger: "Laputa 当前仍依赖同级工作区内容，无法作为独立 Git 项目迁移到新工作服务器运行"
recommended_mode: "batch"
---

# Sprint Change Proposal - Laputa 独立仓库化与迁移阻断修补

## 1. Issue Summary

### 问题定义

在当前 Sprint 执行后期，发现 `Laputa` 虽然功能主线已经基本完成，但**尚未满足“独立 Rust 项目”这一发布与迁移前提**。当前仓库仍然存在对同级工作区内容的直接依赖、文档级路径假设和上游仓库耦合表述，导致项目不能仅凭 `Laputa/` 目录在新工作服务器上直接运行。

### 触发背景

- 当前开发机器已经无法继续承载开发工作。
- 项目需要尽快迁移到新的工作服务器继续推进。
- 用户明确要求 `Laputa` 能作为**完整独立 Git 项目**运营，并进一步具备“上 cargo”的准备条件。

### 证据

1. `Laputa/Cargo.toml` 当前包含：

```toml
[patch.crates-io]
usearch = { path = "../mempalace-rs/patches/usearch" }
```

这意味着只复制 `Laputa/` 目录时，构建链会直接失效。

2. `Laputa/README.md`、`Laputa/AGENTS.md`、`Laputa/STATUS.md` 以及 planning artifacts 中仍大量以历史路径示例引用 `mempalace-rs`、`agent-diva`、`UPSP`、`LifeBook`，容易被误读为默认运行环境。

3. 原 PRD 与 Architecture 虽强调“可移植性”和“独立运行”，但原 Epic 1-8 主要覆盖功能闭环，没有单独建立“独立仓库化 / 发布阻断清理”的实施批次。

### 问题本质

这不是一个普通优化项，而是一个**发布阻断 + 迁移阻断**问题。

WHY 关键判断：

- 如果 `Laputa` 无法脱离父工作区，就不能迁移到新服务器稳定继续开发。
- 如果不能迁移成功，后续所有功能开发优先级都会失去意义。
- 因此现在的最小正确目标不是“正式 cargo publish”，而是先确保 `Laputa` 可以作为**独立 clone 的 Rust 项目** build / test / run。

---

## 2. Impact Analysis

### Epic Impact

受影响的原 Epic：

- **Epic 1: 记忆库初始化**
  - 原 Story 1.1 假设项目结构来自 `mempalace-rs` 继承，但没有要求“继承完成后必须脱离上游目录运行”。

- **Epic 6: CLI 与 MCP 工具接口**
  - CLI / MCP 已完成，但其价值依赖于可独立安装与运行；如果项目无法独立构建，接口交付价值被削弱。

- **Epic 8: 归档与数据迁移**
  - 现有 FR-14 解决“用户数据迁移”，但没有解决“项目自身迁移到新机器继续开发和运行”。

### Story Impact

当前所有已完成 story 的完成定义需要补充一个新的 cross-cutting 约束：

- “完成”不仅是功能 AC 达成，还必须满足**独立仓库可运行**的环境要求。

未来 Story 受影响点：

- 所有新增依赖、文档、发布脚本、测试基线都不得再次绑定父工作区。

### Artifact Conflicts

需要修正或补充的 artifacts：

- `prd.md`
  - 可移植性已有原则，但缺少“独立仓库化”作为近期交付约束的明确表达。

- `epics.md`
  - 需要新增一个最高优先级 patch epic，专门处理独立仓库化与迁移阻断。

- `architecture.md`
  - 现有文档保留了大量“从同级仓复制模块”的实施交接语句，需要在后续技术修正中转为“历史来源说明”。

- `README.md` / `AGENTS.md` / `STATUS.md`
  - 当前表述需要改为“历史来源”或“迁移说明”，避免让开发者误以为兄弟仓库是运行前提。

### Technical Impact

技术层面的具体影响包括：

1. 构建链影响
   - 需要移除 `Cargo.toml` 中任何依赖同级路径的构建期配置。

2. 测试链影响
   - 需要在不含兄弟仓的环境中验证最小 smoke tests。

3. 文档与元数据影响
   - 需要将仓库身份从“基于上游演化的内部派生项目”修正为“独立维护项目，有明确 lineage”。

4. 发布链影响
   - 独立仓库化是 crates.io 准备的前置条件，但本次修补不要求立即完成 publish。

---

## 3. Recommended Approach

### 推荐路径

**Direct Adjustment + New Patch Epic**

在不推翻 Epic 1-8 功能成果的前提下，新增一个紧急 patch epic：

- **Epic 9: 独立仓库化与迁移阻断修补**

### 不推荐路径

- **不推荐回滚**：功能主线已完成，回滚没有必要，只会增加迁移成本。
- **不推荐继续按原 backlog 推进**：因为所有新增工作都建立在不稳定的开发基座上。
- **不推荐先做 cargo publish**：这会把问题从“迁移可用性”误导到“外部分发”，顺序错误。

### 为什么这是正确路径

1. 它直接回应当前真实问题：新机器接不住开发。
2. 它改动范围清晰：聚焦依赖解耦、文档纠偏、独立运行验收。
3. 它保留现有功能资产，不引发大规模返工。
4. 它为后续 `cargo publish`、独立 Git 运营和 agent-diva 集成都打下必要基础。

### Effort / Risk / Timeline

- **工作量**：中等
- **风险级别**：高优先级、可控风险
- **时间影响**：需要立即插队，短期暂停非迁移相关需求

建议执行顺序：

1. 立即冻结非迁移功能开发
2. 先完成构建链解耦
3. 再完成文档与元数据独立化
4. 最后完成新服务器 smoke 验收

---

## 4. Detailed Change Proposals

### 4.1 Epics 文档新增

#### NEW EPIC

```md
### Epic 9: 独立仓库化与迁移阻断修补

**Epic 目标**: 让 Laputa 脱离当前父工作区与兄弟目录依赖，成为可独立 clone、独立构建、独立测试、独立运行的 Rust 项目，为新工作服务器迁移和后续 crates.io 发布做好前置准备。

**完成价值**: 项目可从当前机器安全迁移到新服务器继续开发与运行，发布链具备真实基础。

**FR 覆盖**: NFR-1, NFR-2, NFR-6（补强），发布阻断修补
```

### 4.2 Story 新增提案

#### Story 9.1

```md
### Story 9.1: 构建链路去同级路径依赖

As a 维护者，
I want 移除 Laputa 对父工作区和兄弟目录的构建期依赖，
So that 项目被单独 clone 到新工作服务器后也能直接编译与测试。
```

#### Story 9.2

```md
### Story 9.2: 仓库元数据与文档独立化

As a 新用户，
I want 从仓库元数据与 README 中看到 Laputa 是一个可独立运行的项目，
So that 我可以在新环境中直接理解、安装并启动它，而不是猜测缺失了哪些同级仓库。
```

#### Story 9.3

```md
### Story 9.3: 新服务器独立运行验收

As a 维护者，
I want 在不含旧工作区的干净环境中验证 Laputa 的最小运行链路，
So that 我可以确认这次修补真正解决了迁移阻断问题。
```

### 4.3 PRD 层变更建议

#### Section: Project-Type Requirements

OLD:

- 应允许后续与宿主系统集成，但 MVP 不依赖宿主存在即可独立运行。

NEW:

- 应允许后续与宿主系统集成，但 MVP 不依赖宿主存在即可独立运行。
- 项目必须可作为独立 Git 仓库在不含兄弟项目目录的环境中完成构建、测试与最小运行链路。

Rationale:

原约束强调“宿主独立”，但没有把“仓库独立 / 构建独立”落实为可验收要求。本次问题说明该约束需要显式化。

### 4.4 Architecture 层变更建议

需补充或修订的重点：

- 将所有“从同级目录复制模块”的指令改为“历史初始化步骤”，不再作为当前独立仓库运行前提。
- 增加“独立仓库化约束”小节，明确：
  - 禁止 path dependency 指向父工作区
  - 禁止将兄弟仓作为运行前提
  - 必须支持 fresh checkout build/test/run

---

## 5. Implementation Handoff

### 变更规模分类

**Major**

原因：

- 涉及 Sprint 优先级重排
- 涉及 planning artifacts 修订
- 涉及构建链、文档链、验收链三条主线
- 涉及迁移阻断，属于基础交付前提

### 建议交接对象

- **Product Manager / Product Owner**
  - 确认将 Epic 9 插入当前最高优先级 backlog

- **Developer Agent**
  - 执行 Story 9.1 / 9.2 / 9.3 的实现与验证

- **Architect（如需要）**
  - 对 `usearch` patch 与依赖来源策略做最终技术裁决

### Developer Handoff Success Criteria

1. `Laputa/` 被单独复制到新目录后可以 `cargo build`
2. 不再依赖 `../mempalace-rs` 等路径
3. README 和 Cargo 元数据指向 Laputa 自身
4. 在干净环境中至少跑通 `init -> diary write -> wakeup`
5. 输出迁移验证记录，证明新服务器可接手

### 推荐执行顺序

1. Story 9.1
2. Story 9.2
3. Story 9.3

### 最终建议

立即将 `Epic 9` 置为当前 Sprint 顶部优先级，暂停一切与迁移无关的新增开发。WHY？因为如果项目无法脱离旧机器继续运行，所有后续功能都没有交付基础。
