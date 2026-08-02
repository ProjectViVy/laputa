# Hermes 现状说明 — 给 Codex 施工用的配套

> **作者**: 松本
> **日期**: 2026-07-15
> **配套**: ADR-0004 (Garden 编排层)
> **不引用**: 不修改 ADR-0004, 只补充 hermes 侧事实以便施工。
> **真实来源**: 实地核查 `C:\Users\Administrator\.hermes\`, 非文档推测。

---

## 0. 写给 Codex 的一句话

Garden ADR-0004 的隐含前提是 "Hermes 是 Garden 的唯一对外消费者"。本文件给出 **Hermes 这一边的真实状态**, 让 codex 知道施工时哪些假设会成真、哪些不会。

---

## 1. Hermes 与 Garden 当前的关系

### 1.1 状态: **断开的**

- `~/.hermes/plugins/laputa/` 存在但**已 disabled**:
  ```
  ~/.hermes/plugins/laputa.disabled.20260627_055717/
    ├── __init__.py
    ├── plugin.yaml
    └── __pycache__/
  ```
- 时间戳 `20260627_055717` 表明: 2026-06-27 05:57:17 被 disable。
- 当前没有任何 Hermes plugin 在调用 Garden。
- 也就是说, 当前 Garden 进程和 Hermes 进程之间**完全独立**。Garden 跑起来只在响应外部 HTTP 调用。

**意味着**: 实施 Phase B/C 时, Hermes plugin **需要重新 enable 或新建一个 plugin** 来对接 Garden。

### 1.2 也没有 MCP server 在用

- `~/.hermes/skills/` 下只有 3 个, 都不是 laputa:
  ```
  evolver-hermes/
  evolver-mcp/
  evolver-native/
  ```
- 当前 `~/.hermes/sessions/` 目录**不存在** — Hermes 当前没有持久化 session 存储。

---

## 2. Hermes 的工具调用机制 — 真实模型

### 2.1 两条路径

**主路径 (declared native tool)**:
- Hermes 启动时, plugin 注入固定工具集
- 每个工具签名明确: 名称 + JSON Schema input + 返回结构
- 模型调用时按 schema 填字段
- plugin 内部封装好了 HTTP / DB / 文件 IO

**Escape hatch (bash / curl)**:
- Hermes 有 bash 工具, 可以 `curl -X POST ...`
- 没有通用 HTTP 工具(没有 "make_http_request" 这种)
- bash 能拼 curl 命令, 但模型记不住 schema / endpoint / auth 头

### 2.2 ADR-0004 §4.3 的设计含义

- `garden_search(query)` 必须作为 **declared native tool** 注入, 单参数 `query`
- wing / room / top_k / model / pipeline 名 / capabilities **不应出现在工具 schema**
- 这些由 Garden 根据 Laputa policy + Skill manifest + 系统配置内部决定
- 调试 / 迁移期可直接 curl `127.0.0.1:7373`, 但**不是 Hermes 的契约路径**

### 2.3 QQ 通道的 quirk(已知, 与 ADR 无关)

- QQ 通道**不渲染 clarify 工具的 choices 选项按钮**, 用户只看到 question 文本
- 施工时如果 Hermes 端工具有 UI 提示, 别依赖 QQ 不支持的渲染

---

## 3. Hermes 的 session 生命周期 — 真实情况

### 3.1 当前 session 不持久化

- `~/.hermes/sessions/` 当前**不存在**
- 当前 Hermes 进程每开一次就是新 process
- session 内容不落盘(本机观察)

### 3.2 ADR-0004 §4.2 的 "session ingest" 含义

- 描述 "Hermes 宿主 lifecycle 在 stop / precompact / session-end 调用" `POST /v1/sessions`
- 这是**未来设计**, 当前 Hermes 没有 lifecycle hook 挂上
- **实施 ADR-0004 Phase C 时, 配套要做**:
  - 在 Hermes 端建一个 plugin (或改 evolver-hermes skill)
  - 挂 session-end hook
  - hook 内部 POST `/v1/sessions` 把 transcript 发给 Garden
- 这条配套工作**不在 Garden 仓库内**, 是在 Hermes 端

### 3.3 pending writes 是个历史包袱

- `~/.hermes/laputa-pending-writes.md` (4 KB, 2026-06-24 写) 是当时给 mempalace palace 写 9 个 drawer 的脚本
- 状态: "待写入 (mempalace palace lock 阻塞)"
- 这是 morediva 时代的, 跟 Garden Laputa 阶段没直接关系, 但**别让 codex 把它当真要执行的脚本** — 已超期

---

## 4. 当前 Garden 鉴权状态

### 4.1 没有鉴权

- Garden 进程在 `127.0.0.1:7373` 上**裸跑**
- 任何 localhost 进程都可以调, 无 Authorization header
- 没有 mTLS, 没有 unix socket
- **意味着 ADR-0004 实施时必须决定**:
  - 选项 A: 维持无鉴权, curl 不受限, 但 model 可以在 plugin 里乱调
  - 选项 B: 加 bearer token, plugin 持有, curl 必须带 header
  - 选项 C: 加 mTLS / unix socket, 同 user 才能连
- 这个决策**影响 Phase B 实施**: 如果定 B/C, Phase B 必须同时上鉴权; 不上, Phase D 才补

### 4.2 替代: NetworkPolicy 也行

- 单纯 127.0.0.1 listener 已经防御 90% 外部
- 但**不影响同一台机器上的其他进程**
- 真正隔离得 listener 到 unix socket 或者加 firewall

---

## 5. Hermes 这一侧的具体约束

### 5.1 不能在 Garden 进程内开 subagent

- user 偏好: 不接 codex / claude-code / subagent (memory 持久化)
- Garden Phase 5 E2E 测试时就这么定: garden.exe 单进程
- **别让 codex 设计 "Garden 接到 ingest 后自动开 subagent 处理"** — 不符合偏好

### 5.2 GOSUMDB=off

- 本机 Go sumdb 不可达
- 所有 garden / laputa / mentle 的 `go build` / `go test` 命令**必须前缀 `GOSUMDB=off`**
- 没设的话 build 失败会卡 Phase 0 验证

### 5.3 Rust 隔离

- `~/Desktop/laputa-work/`, `~/Desktop/olv-rs/`, `~/Desktop/new-mentle/memtle` 是独立 Rust 路径
- Garden 项目**完全不动**
- 如果 codex 觉得 "memtle 能 import 进 mentle" — 不行, 隔离是用户硬约束

### 5.4 mempalace-py 已物理隔离

- `~/Desktop/morediva/.workspace/mempalace-py/` — Python 版 mempalace
- 跟 Garden Laputa 阶段**不共享代码**
- 别让 codex 觉得 "Python 版能 import 进来加快 facade" — 不能
- 但 **Hermes plugin 端** 跟 mempalace-py 时期是有继承关系的(`laputa-pending-writes.md` 那套 wing_laputa 是当时设计), 别全清掉

---

## 6. 历史的 `tmp_*.txt` 与 `laputa-pending-writes.md`

- `~/.hermes/tmp_identity.txt` (621 B), `tmp_relationship.txt` (792 B), `tmp_report_indexes.txt` (290 B)
- `~/.hermes/laputa-pending-writes.md` (4 KB)
- 这些是 6/24-6/27 时期尝试往 mempalace palace 注入 drawer 的草稿, 状态都没落地
- **codex 别当现状**:
  - 草稿里写的 "Laputa v1.0.0 11 sections", 当前已是 14 sections
  - 草稿里 "190+ skills", 当前实际是 100+ (skill_creator 在 evolver 里)
- 这些草稿可以参考结构, 不能照搬事实

---

## 7. ADR-0004 Phase 实施时 Hermes 端配套

| Phase | Garden 端 | Hermes 端配套 (不在 Garden 仓库) |
|---|---|---|
| **Phase A** facade mutation 基础 | facade schema + 原子 mutation | 无 |
| **Phase B** Garden CRUD + 权限 | PATCH + token | plugin 改造: native tool adapter `garden_crud_*` (5 个) |
| **Phase C** Session ingest | `/v1/sessions` + `session_ingest_v1` | plugin 改造: lifecycle hook (session_end / precompact) POST /v1/sessions |
| **Phase D** Basic/Advanced + Skill registry | pipeline + skill manifest | plugin 改造: native tool `garden_search(query)` |
| **Phase E** Report + 注入统一 | report pipeline + bootstrap API | plugin 改造: basic context 注入钩子 |

**给 codex 的提示**: Garden 仓库内的 todo 只走 Garden 侧。Hermes 端改造跟 codex 不在同一个工作空间, 可能会安排另一个 session 单独做。

---

## 8. 路径速查(给 codex)

```
~ = C:\Users\Administrator

# Hermes 配置
~/.hermes/
  plugins/
    laputa.disabled.20260627_055717/   ← 旧 plugin, 已 disable
  skills/                              ← evolver-* 三件套
  plans/                               ← 计划草稿
  scripts/                             ← 自定义脚本
  skins/                               ← 主题
  evolver-hooks/                       ← evolver 钩子
  tmp_identity.txt                     ← 历史草稿
  tmp_relationship.txt                 ← 历史草稿
  tmp_report_indexes.txt               ← 历史草稿
  laputa-pending-writes.md             ← 6/24 历史草稿
  .hermes_history                      ← bash 历史

# Garden Laputa 仓库
~/Desktop/garden/                      ← 主工作区
  laputa/                              ← V1 治理 (Git submodule)
  mentle/                              ← V2 记忆 (Git submodule)
  garden/                              ← 应用入口
  docs/architecture/                   ← ADR 系列

# 不动
~/Desktop/laputa-work/                 ← Rust 路径
~/Desktop/olv-rs/                      ← Rust 路径
~/Desktop/new-mentle/                  ← Rust 路径
~/Desktop/morediva/.workspace/mempalace-py/   ← Python 路径
```

---

## 9. 三条最容易踩的坑(给 codex 划重点)

1. **别假设 Hermes plugin 还在** — 当前 `laputa.disabled.20260627_055717/` 已 disable, 实施 Phase B/C 需要重建 plugin。这是另一个代码仓库的工作, **别在 Garden 仓库里写 plugin 代码**。

2. **端点鉴权状态要不要保留现状** — 当前 0 鉴权。Codex 实施 Phase B 时要么 (a) 维持, (b) 加 token。决策未定, 需松本拍。如果维持, ADR 文字不变; 如果加, Phase B 同步上, 别 Phase D 补。

3. **session ingest 是 lifecycle hook, 不是可选调用** — Hermes 端必须在 session_end / precompact 触发, 不是用户按钮。如果 Phase C 实施时 Garden 端 `/v1/sessions` 设计成 "可选工具", 跟 Hermes 端不兼容。Codex 设计 schema 时把 envelope session_id 作**必填**, 提示 Hermes 端挂 hook 时不会漏。

---

## 10. 不在本文件范围

- ADR-0004 本身的方案评审 — 不参与
- Garden 端 Phase A-E 实施细节 — 留给 codex 施工阶段
- Hermes plugin 工程 — 另一个 session 另立
- mempalace-py 内容 — 已物理隔离, 不引用

---

*作者: 松本*
*写于: 2026-07-15*
*配套: ADR-0004*
*真实来源: ~/.hermes/ 实地目录 + ADR-0004*
