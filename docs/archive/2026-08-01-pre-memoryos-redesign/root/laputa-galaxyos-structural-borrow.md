# 在 Laputa 中吸收 GalaxyOS 结构思想的建议

## 核心原则

- 只借鉴 GalaxyOS 的流程与数据结构，**不引入 PyTorch / ODE / 神经网络模块**。
- 所有机制必须落在 Laputa 的**文件治理 + 审计日志**中，可观测、可回滚、可人工审计。
- 记忆更新不是“得分低就删”，而是基于**明确规则**和**证据链**。

---

## 一、GalaxyOS 可借鉴的四个结构点

### 1. 记忆状态字段

在现有记忆条目上增加状态字段，替代隐式的“权重衰减”：

| 状态 | 含义 | 触发条件 |
|------|------|----------|
| `active` | 当前有效 | 默认状态 |
| `expired` | 已过期 | 超过自然时间边界，且无新证据支持 |
| `superseded` | 被新事实覆盖 | 新事实与旧事实冲突，且置信度更高 |
| `uncertain` | 待确认 | 与新事实部分冲突，但证据不足 |
| `rejected` | 用户明确否定 | 用户直接说“没这回事/记错了” |

> 只有 `rejected` 或明确垃圾/测试数据才物理删除。其他状态只变更标记，保留历史。

---

### 2. 写入时冲突扫描

每次写入新记忆前，对现有 `active` 记忆进行冲突扫描：

- 新记忆："周末打球、社交、喝酒"
- 现有 active 记忆："周一感冒看病"
- 冲突检测：调用轻量 LLM 或规则判断，生成 `conflict_report`

冲突扫描必须输出：
- `conflict_type`：`temporal` / `causal` / `state` / `none`
- `old_memory_id`：被冲突的旧记忆
- `new_memory_id`：新写入的记忆
- `confidence`：覆盖置信度（0.0-1.0）
- `reason`：一句话说明

---

### 3. 冲突结果必须记录

冲突消解不是默默改库，而是要写入审计日志：

```json
{
  "event": "memory_superseded",
  "old_memory_id": "mem-001",
  "new_memory_id": "mem-042",
  "state": "superseded",
  "confidence": 0.87,
  "reason": "新事实'周末打球、社交、喝酒'与旧事实'周一感冒看病'冲突，推断感冒已恢复。",
  "timestamp": "2026-06-30T14:23:00Z"
}
```

旧记忆本身也被追加一个 `supersession` 字段：

```json
{
  "id": "mem-001",
  "content": "用户周一感冒看病",
  "state": "superseded",
  "superseded_by": "mem-042",
  "superseded_at": "2026-06-30T14:23:00Z",
  "superseded_reason": "新事实表明身体已恢复活跃"
}
```

---

### 4. 淘汰规则要明确

| 操作 | 条件 | 不是 |
|------|------|------|
| 标记 `superseded` | 新事实与旧事实冲突，且置信度 > 阈值 | 不是“得分低” |
| 标记 `expired` | 超过自然时间边界，无更新证据 | 不是“好久没提” |
| 标记 `rejected` | 用户明确否定 | 不是系统猜测 |
| 物理删除 | 仅垃圾/测试数据，或用户明确要求 | 不是自动清理 |

---

## 二、反向艾宾浩斯：使用强化机制

### 定义

与传统艾宾浩斯遗忘曲线（**不用就衰减**）相对，**反向艾宾浩斯**指：

> **记忆被主动检索、引用、确认后，其有效期会被强化，而不是简单衰减。**

### 为什么需要

- 艾宾浩斯描述“遗忘”：时间越长，回忆成本越高。
- 反向艾宾浩斯描述“巩固”：每次成功使用，都会刷新记忆的“最近确认时间”，让它更不容易被衰减掉。

### 落地方式

给记忆条目增加 `last_confirmed` 字段：

```json
{
  "id": "mem-042",
  "content": "用户喜欢周末打球",
  "created_at": "2026-06-15T10:00:00Z",
  "last_confirmed": "2026-06-30T14:23:00Z",
  "confirm_count": 3
}
```

每次该记忆被成功召回并产生有效输出，记录一个 `memory_reinforced` 事件：

```json
{
  "event": "memory_reinforced",
  "memory_id": "mem-042",
  "old_last_confirmed": "2026-06-28T09:10:00Z",
  "new_last_confirmed": "2026-06-30T14:23:00Z",
  "confirm_count": 3,
  "reason": "成功用于推荐周末活动"
}
```

### 权重公式（示意）

```
effective_weight = base_weight * exp(- (t - last_confirmed) / tau)
```

- `t`：当前时间
- `last_confirmed`：最近一次成功确认时间
- `tau`：半衰期参数，可配置
- `confirm_count` 可作为 `base_weight` 的加成系数

### 与单纯艾宾浩斯的区别

| 机制 | 行为 | 例子 |
|------|------|------|
| 艾宾浩斯 | 不用就衰减 | 感冒看病 30 天后权重下降 |
| 反向艾宾浩斯 | 用了就刷新 | 每次提到“打球”都强化这条记忆 |

---

## 三、与 GalaxyOS 的本质区别

| 维度 | GalaxyOS | Laputa 方案 |
|------|----------|-------------|
| 核心手段 | PyTorch / LTC / CfC / SSM / Neural ODE | 文件治理 + 审计日志 + 轻量规则 |
| 遗忘/强化 | 神经网络包装的时间门 | 艾宾浩斯 + 反向艾宾浩斯 |
| 冲突处理 | 5 维评分 + 黑箱淘汰 | 状态字段 + 冲突记录 + 明确规则 |
| 可解释性 | 低 | 高 |
| 维护成本 | 极高 | 低 |
| 上线风险 | 高（耦合 OpenClaw 插槽） | 低（独立 provider） |

---

## 四、落地建议

1. **在 `LAPUTA.md` 中扩展记忆状态字段定义**：新增 `active / expired / superseded / uncertain / rejected`。
2. **在 `Diary.Write()` 路径中增加 `conflict_scan` 钩子**：写入新记忆前扫描现有 active 记忆。
3. **审计日志新增事件类型**：`memory_superseded`、`memory_reinforced`、`memory_expired`。
4. **实现反向艾宾浩斯**：给记忆增加 `last_confirmed` 和 `confirm_count` 字段，每次成功召回刷新。
5. **保留简单规则**：先跑通符号化流程，再考虑是否需要更复杂的模型。

---

> 记忆系统的难点不是“有没有神经网络”，而是“有没有清晰的证据链和可回滚的治理结构”。
