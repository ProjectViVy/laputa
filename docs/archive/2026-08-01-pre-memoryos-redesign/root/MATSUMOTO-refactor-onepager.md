# MATSUMOTO 整体改造 · 1 页摘要（2026 Q3）

> 完整计划: `MATSUMOTO-overall-refactor-2026Q3.md`（77KB / 6 Wave / 28 Task）
> 同步副本: `~/.hermes/plans/2026-06-28_180000-matsumoto-overall-refactor.md`

---

## 一句话目标

把松本（v0 单进程 laputa-py）升级为 **双进程 + 结构化目标引擎 + 可执行守护**的工程化 agent harness，对齐 ulw-loop + OmO 范式。

---

## 五大模块

| # | 模块 | 现状 | 改造内容 |
|---|---|---|---|
| 1 | **8-file Authority 治理**（MEMORY/LONGMEM/SOUL/USERPROFILE/USER/IDENTITY/TASK/THOUGHTS） | ✅ 已实装 | 不动，仅做基线确认 |
| 2 | **ulw-loop 核心**（state machine / evidence / steering） | ❌ 0 | W2 全量翻译 TS → Python（~1400 行 + 25 tests） |
| 3 | **ulw-loop MCP Server**（5 tools: create_goals / record_evidence / checkpoint / steer / status） | ❌ 0 | W3 用 `mcp` 官方 SDK，挂进现有 daemon |
| 4 | **Hermes Skill 层**（`loop-engineering` skill + `laputa-cli` wrapper） | ❌ 0 | W4 skill 工作流规范 + e2e test |
| 5 | **Daemon 闭环 + Curator 巡检** | ⚠️ 半重构 | W1 清理 + W5 挂载 + ulw-loop 健康检查 |

---

## 6 Wave 时间线

| Wave | 主题 | Task | 估时 | 风险 |
|---|---|---|---|---|
| **W1** | 工作区清理 + 基线 | 5 | 0.5d | 低 |
| **W2** | ulw-loop 核心（types / storage / evidence / steering / apply / checkpoint） | 6 | 1d | 中 |
| **W3** | MCP Server（5 tools + daemon 挂载 + 重连退避） | 5 | 1d | 中 |
| **W4** | Skill 注入（loop-engineering + laputa-cli + e2e） | 4 | 0.5d | 低 |
| **W5** | Daemon 闭环（审计 + 挂载 + Curator 巡检 + cronjob） | 5 | 1d | 高 |
| **W6** | 验证 + 文档 + `v0.2.0-rc1` | 3 | 0.5d | 低 |

**总计**: 28 Task / ~4.5 个开发日

---

## 10 条核心设计决策

| # | 决策 | 理由 |
|---|---|---|
| 1 | MCP 库选 `mcp` 官方 Python SDK | 标准协议、跨客户端、官方维护 |
| 2 | 锁用 `filelock` 库 | 跨平台、Windows 兼容、API 友好 |
| 3 | ulw-loop plan 存 **per-repo** `.laputa/ulw-loop/` | 与 8-file 全局治理解耦 |
| 4 | ulw-loop MCP **塞进现有 daemon** | 复用 lock + curator 调度 |
| 5 | `weakens()` 正则防御绕过尝试 | 5 行挡 80% 攻击 |
| 6 | **7 种结构化 steering kind**，自然语言全 reject | 防伪 + 可审计 |
| 7 | checkpoint 失败 3 次同类 blocker → `needs_user_decision` | 避免死循环、主动升级人类 |
| 8 | 提供 `laputa-cli` Python wrapper | JSON 输出、错误友好、易测 |
| 9 | Curator 加 ulw-loop 巡检 | 统一调度、统一通知 |
| 10 | 先 `v0.2.0-rc1` 等稳定再 stable | 重大架构变更的稳妥做法 |

---

## 三大改造目标

1. **清理半重构**（W1）—— 提交 main 的 uncommitted 改动；恢复误删的 `queue.py`；丢弃 `mcp/server/bridge` 三个未完成目录
2. **注入 ulw-loop 范式**（W2-W4）—— 把"loop engineering"从 prompt 升级为带状态机的工程产品
3. **守护进程闭环**（W5）—— Daemon 启动 + Curator 周期任务 + ulw-loop MCP 全部跑通

---

## 5 个待澄清问题

| # | 问题 | 建议 |
|---|---|---|
| 1 | MCP 协议：stdio（单 client）还是 socket（多 client）？ | daemon 走 socket |
| 2 | daemon 端口分配？ | 存 `~/.laputa/daemon.port` |
| 3 | ulw-loop plan 路径：per-repo vs global？ | per-repo |
| 4 | 8-file 治理 vs ulw-loop plan 关系？ | 完全独立，curator 写摘要进 MEMORY.MD |
| 5 | 标签策略？ | 先 `v0.2.0-rc1` 等一周再 stable |

---

## 关键风险

- ⚠️ **Daemon 集成破坏现有 palace_bridge / curator 调度**（W5.1 先 audit 再动手）
- ⚠️ **TS → Python 翻译语义漂移**（测试用例 1:1 翻译对齐 ulw-loop 原版）
- ⚠️ **工作区有 6+ uncommitted 改动**（含误删 queue.py）—— W1 先稳

---

## 新增/修改文件统计

| 类别 | 数量 |
|---|---|
| 新建模块 | 7（ulw/types/storage/evidence/steering/apply/checkpoint/mcp_tools） |
| 新建 test | 8（ulw/* + e2e + reconnect + curator + frontmatter） |
| 新建 skill | 1（loop-engineering，含 wrapper） |
| 修改模块 | 5（cli / daemon / memory_provider / curator / pyproject） |
| 文档 | 3（CHANGELOG / README / ulw-loop-integration.md） |

---

## 执行准备清单

- [x] 确认 main 分支继续推进
- [ ] 大湿授权本计划执行
- [ ] 验证 `mcp` + `filelock` 可装（Python 环境）
- [ ] 确认 daemon 端口范围（建议 8765-8799）

---

**执行方式**: 大湿拍板 → `subagent-driven-development` 按 W1→W6 顺序 → 每 Wave 一次 checkpoint → W6 发 `v0.2.0-rc1`。
