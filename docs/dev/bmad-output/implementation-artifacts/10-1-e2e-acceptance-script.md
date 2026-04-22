# Story 10.1: 端到端链路验收脚本

**Story ID:** 10.1  
**Story Key:** 10-1-e2e-acceptance-script  
**Status:** ready-for-dev  
**Created:** 2026-04-22  
**Project:** 天空之城 (Laputa)

---

## 用户故事

As a **产品验收者**，
I want **有一条自动化脚本验证完整的 MVP 链路**，
So that **我可以证明 PRD Success Criterion #7 已满足**。

---

## 验收标准

### AC1: init 命令验收

- **Given** Laputa 项目已编译完成
- **When** 执行 `laputa init --name "测试用户"`
- **Then** 成功创建 `vectors.db`（当前数据库文件名，Story 10-2 统一为 laputa.db）
- **And** 成功创建 `identity.md` 包含 `user_name: 测试用户`
- **And** 返回成功消息与数据库路径

### AC2: diary write 命令验收

- **Given** init 已成功执行
- **When** 执行 `laputa diary write --content "测试日记" --tags "test"`
- **Then** 成功写入 L1 层
- **And** 返回 memory_id（数值格式）
- **And** tags 正确解析为数组

### AC3: wakeup 命令验收

- **Given** diary write 已成功执行
- **When** 执行 `laputa wakeup`
- **Then** 返回 JSON 格式唤醒包
- **And** `token_count` 字段存在且值 < 1200（NFR-3）
- **And** `identity.user_name` 为 "测试用户"
- **And** `recent_state` 包含刚写入的记忆摘要

### AC4: recall 命令验收

- **Given** diary write 已成功执行
- **When** 执行 `laputa recall --time-range "YYYY-MM-DD~YYYY-MM-DD"`（当日日期范围）
- **Then** 返回今日记忆列表
- **And** 包含刚写入的测试日记内容
- **And** 格式正确（时间范围、条目数量）

### AC5: 验收报告生成

- **And** 验收报告写入 `_bmad-output/implementation-artifacts/mvp-acceptance-report.md`
- **And** 验收报告包含：
  - 执行时间（ISO 8601 格式）
  - 环境信息（Rust 版本、操作系统）
  - 各步骤结果（逐条 AC 对照）
  - 阻断原因（若失败）

---

## Tasks / Subtasks

- [ ] Task 1: 创建验收脚本框架 (AC: #5)
  - [ ] 1.1 创建 `tests/integration/test_mvp_acceptance.rs` 或 Shell 脚本
  - [ ] 1.2 定义验收流程：init → diary write → wakeup → recall
  - [ ] 1.3 设计报告生成模板

- [ ] Task 2: 实现 init 验收步骤 (AC: #1)
  - [ ] 2.1 执行 `laputa init --name "测试用户"`
  - [ ] 2.2 验证 vectors.db 文件创建
  - [ ] 2.3 验证 identity.md 内容包含正确 user_name
  - [ ] 2.4 检查命令返回成功状态

- [ ] Task 3: 实现 diary write 验收步骤 (AC: #2)
  - [ ] 3.1 执行 `laputa diary write --content "测试日记" --tags "test"`
  - [ ] 3.2 验证返回有效的 memory_id
  - [ ] 3.3 验证 L1 层写入成功（可选：直接检查数据库）

- [ ] Task 4: 实现 wakeup 验收步骤 (AC: #3)
  - [ ] 4.1 执行 `laputa wakeup`
  - [ ] 4.2 解析 JSON 输出
  - [ ] 4.3 验证 token_count < 1200
  - [ ] 4.4 验证 identity.user_name 正确
  - [ ] 4.5 验证 recent_state 非空

- [ ] Task 5: 实现 recall 验收步骤 (AC: #4)
  - [ ] 5.1 计算当日日期范围
  - [ ] 5.2 执行 `laputa recall --time-range "开始~结束"`
  - [ ] 5.3 验证返回结果包含测试日记
  - [ ] 5.4 验证输出格式正确

- [ ] Task 6: 生成验收报告 (AC: #5)
  - [ ] 6.1 汇总所有步骤结果
  - [ ] 6.2 收集环境信息（rustc --version, OS）
  - [ ] 6.3 写入 `_bmad-output/implementation-artifacts/mvp-acceptance-report.md`
  - [ ] 6.4 格式化为标准验收文档结构

---

## Dev Notes

### CLI 命令技术细节

**数据库路径约定**（当前状态，Story 10-2 后统一）：
- 主存储：`vectors.db`（CLI diary write 使用）
- 向量索引：`vectors.usearch`
- 知识图谱：`knowledge.db`
- 身份文件：`identity.md`

**命令格式**（来自 [handlers.rs](Laputa/src/cli/handlers.rs:27-43)）：

```
laputa init --name "用户名"
laputa diary write --content "内容" --tags "标签" --emotion "情绪码"
laputa wakeup [--wing "分区"]
laputa recall --time-range "YYYY-MM-DD~YYYY-MM-DD" [--limit N]
laputa mark --id <id> --important|--forget|--emotion-anchor
```

**时间范围格式**（来自 [handlers.rs:216-258](Laputa/src/cli/handlers.rs:216-258)）：
- 格式：`YYYY-MM-DD~YYYY-MM-DD`
- 边界验证：年份 ∈ [1900, 2100]
- 跨度限制：≤365天

**WakePack 结构**（来自 [wakeup/mod.rs](Laputa/src/wakeup/mod.rs:44-57)）：
```json
{
  "identity": {
    "user_name": "测试用户",
    "user_type": "个人记忆助手",
    "created_at": "...",
    "fields": [...]
  },
  "recent_state": [
    { "id": 1, "wing": "...", "room": "...", "heat_i32": 5000, "summary": "..." }
  ],
  "weekly_capsule": null,
  "key_relations": [],
  "token_count": <数字>
}
```

### 项目结构 Notes

**关键文件位置**：
- CLI handlers: `Laputa/src/cli/handlers.rs`
- CLI commands: `Laputa/src/cli/commands.rs`
- WakePack generator: `Laputa/src/wakeup/mod.rs`
- Storage: `Laputa/src/storage/mod.rs`
- MCP server: `Laputa/src/mcp_server/mod.rs`

**测试文件结构建议**：
```
Laputa/tests/
├── integration/
│   └── test_mvp_acceptance.rs  # 本 Story 验收脚本（Rust 实现）
└── fixtures/
    └── acceptance_fixture.rs   # 验收专用 fixture
```

或使用 Shell 脚本：
```
scripts/
└── mvp_acceptance.sh           # Bash 验收脚本（推荐用于快速验收）
```

### 前一个 Story (9-3) 的关键经验

**执行顺序问题**：
- 必须串行执行：`init` → 等待完成 → `diary write` → 等待完成 → `wakeup`
- 并行执行会导致 `Laputa is not initialized` 错误

**验收报告格式要求**（来自 [9-3-migration-validation-report.md](implementation-artifacts/9-3-migration-validation-report.md)）：
- 包含 AC 对照清单
- 每条 AC 明确标记 PASS/FAIL
- 失败时记录阻断原因与后续建议
- 时间戳使用 ISO 8601 格式

**测试覆盖建议**（来自 [deferred-work.md](implementation-artifacts/deferred-work.md:114-157)）：
- CLI 已有基础测试（handlers.rs 单元测试）
- 需补充 E2E 链路测试
- 注意 --config-dir 参数指定运行目录

### 环境要求

**运行前提**：
- `cargo build --release` 已完成
- 可执行文件 `laputa` 或 `target/release/laputa[.exe]` 存在
- 独立运行目录（使用 --config-dir 或临时目录）

**环境信息收集**：
```bash
rustc --version          # Rust 版本
cargo --version          # Cargo 版本
uname -a 或 systeminfo   # 操作系统信息
```

### References

- PRD Success Criterion #7: MVP 发布时至少具备一条端到端演示链路 [Source: prd.md:39]
- NFR-3: 唤醒包 token 上限 1200 [Source: prd.md:217]
- NFR-4: 响应性能要求 [Source: prd.md:220]
- CLI handlers 实现 [Source: Laputa/src/cli/handlers.rs]
- WakePack 生成逻辑 [Source: Laputa/src/wakeup/mod.rs:68-84]
- deferred-work C1 数据库路径问题 [Source: deferred-work.md:114]

---

## Dev Agent Record

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### File List