# Story: 热度干预冗余查询优化

**Story ID:** patch-1a  
**Story Key:** patch-1a-heat-performance  
**Status:** done  
**Created:** 2026-04-16  
**Updated:** 2026-04-19  
**Project:** 天空之城 (Laputa)

**Origin:** `deferred-work.md` Epic 5 P1-P2

---

## 用户故事

As a **开发者**，  
I want **移除 `mark_important` / `mark_forget` 中多余的预查询**，  
So that **手动干预路径减少一次数据库往返，同时保持现有错误语义和返回结果不变**。

---

## 验收标准

1. **P1**: `mark_important()` 不再先调用 `get_memory_by_id()` 做存在性检查；改为直接执行 `UPDATE`，再基于 `rows_affected` 判断是否命中目标记录。
2. **P2**: `mark_forget()` 采用与 `mark_important()` 相同的直接 `UPDATE` 模式，消除相同冗余查询。
3. 记录存在时，两个接口仍然返回更新后的完整 `MemoryRecord`，字段语义保持不变：
   - `mark_important`: `heat_i32 = 9000`、`is_archive_candidate = false`、`reason` 更新
   - `mark_forget`: `heat_i32 = 0`、`is_archive_candidate = true`、`reason` 更新
4. 记录不存在时，两个接口仍返回可被上层识别为“Memory not found”的错误；不得因本次优化破坏 CLI/MCP 的错误映射行为。
5. 补充或更新自动化测试，覆盖：
   - `mark_important` 成功路径
   - `mark_important` 缺失记录路径
   - `mark_forget` 成功路径
   - `mark_forget` 缺失记录路径
   - 失败路径无副作用

---

## 缺陷清单

| ID | 问题 | 文件位置 | 影响 |
|----|------|----------|------|
| **P1** | `mark_important` 先查再改，产生冗余查询 | `Laputa/src/vector_storage.rs:641-658` | 同一次干预路径多一次 `SELECT`，增加本地 SQLite 往返 |
| **P2** | `mark_forget` 先查再改，产生冗余查询 | `Laputa/src/vector_storage.rs:661-678` | 与 P1 相同，路径不必要地重复读取 |

---

## 实施任务

- [x] 将 `mark_important()` 改为“直接 `UPDATE` -> 检查 `rows_affected` -> `get_memory_by_id()` 返回结果”
- [x] 将 `mark_forget()` 改为同样的流程，不再预先调用 `get_memory_by_id()`
- [x] 保持未命中时错误消息继续包含 `Memory not found: id={memory_id}`，避免破坏 `cli/handlers.rs` 与 `mcp_server/mod.rs` 的字符串映射
- [x] 不修改 `apply_intervention()`、CLI 参数层、MCP tool schema；本 patch 只收敛 `VectorStorage` 内部实现
- [x] 在 `Laputa/tests/test_user_intervention.rs` 增补 `mark_forget` 缺失记录测试，确保两条路径都固定错误与副作用行为

---

## Dev Notes

### 业务与故事上下文

- Epic 5 负责热度机制与手动治理。
- Story 5.3 已定义用户干预语义：
  - `--important` -> `heat = 9000` 并锁定
  - `--forget` -> `heat = 0`，标记归档候选
- 本 patch 不改变干预语义，只修复实现层性能浪费。

### 当前代码现状

- `get_memory_by_id()` 已统一返回完整 `MemoryRecord`，不存在时返回 `.context("Memory not found")`。
- `mark_important()` / `mark_forget()` 当前流程都是：
  1. `get_memory_by_id(memory_id)` 预检查存在性
  2. 执行 `UPDATE`
  3. 命中 0 行时返回 `anyhow!("Memory not found: id={memory_id}")`
  4. 再次 `get_memory_by_id(memory_id)` 取回更新后记录
- 第 1 步与第 3 步职责重复；保留第 4 步是合理的，因为调用方仍需要更新后的完整记录。

### 实现护栏

1. **只删预查询，不删最终回读**
   - 本 patch 的目标是把 3 次数据库交互降到 2 次，而不是重构成 SQL `RETURNING` 或修改返回类型。
   - `mark_important()` / `mark_forget()` 仍需返回完整 `MemoryRecord` 给现有调用链和测试。

2. **保持错误语义稳定**
   - `Laputa/src/cli/handlers.rs` 与 `Laputa/src/mcp_server/mod.rs` 当前都通过 `message.contains("Memory not found")` 将 `anyhow::Error` 映射成 `LaputaError::NotFound`。
   - 因此优化时不能把错误文本改成别的语义，比如 `Record missing` 或仅返回空结果。

3. **不要扩大 patch 范围**
   - 不要顺手把 `mark_emotion_anchor_with_reason()`、`update_memory_emotion()` 等其他路径一并重构，除非发现完全相同且阻塞 AC 的缺陷。
   - 不要在本故事中引入新的错误类型转换策略；这会影响 CLI/MCP 与既有测试。

4. **保持字段语义不变**
   - `mark_important`: `heat_i32 = 9000`、`is_archive_candidate = 0`
   - `mark_forget`: `heat_i32 = 0`、`is_archive_candidate = 1`
   - `reason = ?1` 的写入行为必须保留

### 建议实现方式

- 推荐保持现有函数签名不变：
  - `pub fn mark_important(&self, memory_id: i64, reason: &str) -> Result<MemoryRecord>`
  - `pub fn mark_forget(&self, memory_id: i64, reason: &str) -> Result<MemoryRecord>`
- 推荐流程：
  1. 直接执行 `UPDATE ... WHERE id = ?2`
  2. 若 `rows_affected == 0`，返回 `anyhow!("Memory not found: id={memory_id}")`
  3. 否则调用 `self.get_memory_by_id(memory_id)` 返回更新后的记录

### 测试要求

- 继续使用 `Laputa/tests/test_user_intervention.rs`
- 现有测试已覆盖：
  - `mark_important` 成功
  - `mark_forget` 成功
  - `Important` 缺失记录无副作用
- 本 patch 至少补齐：
  - `Forget` 缺失记录无副作用
- 测试断言应继续固定：
  - 错误文本包含 `Memory not found`
  - 已存在记录在失败路径不被污染

### 架构与非功能约束

- NFR-2: 离线可用性
  - 仅调整本地 SQLite 查询路径，不引入任何外部依赖
- NFR-4: 本地响应性能
  - 目标就是减少手动干预路径一次不必要查询
- NFR-5: 数据可解释性
  - `reason` 字段写入必须保持
- NFR-10: 可测试性
  - 通过现有自动化测试固定行为

### Project Structure Notes

- 主要修改文件：
  - `Laputa/src/vector_storage.rs`
  - `Laputa/tests/test_user_intervention.rs`
- 不应新增新模块、新配置或新文档类型。

## 参考资料

- `_bmad-output/implementation-artifacts/deferred-work.md`
  - Epic 5 P1-P2 deferred items
- `_bmad-output/planning-artifacts/epics.md`
  - Epic 5 / Story 5.3 用户干预接口
- `_bmad-output/planning-artifacts/prd.md`
  - NFR-2 离线可用性
  - NFR-4 本地响应性能
  - NFR-5 数据可解释性
  - NFR-10 可测试性
- `Laputa/src/vector_storage.rs`
  - `get_memory_by_id()`
  - `mark_important()`
  - `mark_forget()`
- `Laputa/src/cli/handlers.rs`
  - `map_anyhow_error()`
- `Laputa/src/mcp_server/mod.rs`
  - `map_anyhow_error()`
- `Laputa/tests/test_user_intervention.rs`

## Dev Agent Record

### Agent Model Used

GPT-5 Codex

### Debug Log References

- 2026-04-19: 重新分析 Epic 5、`deferred-work.md`、`vector_storage.rs`、CLI/MCP 错误映射与现有用户干预测试。
- 2026-04-19: 移除 `mark_important()` / `mark_forget()` 的存在性预查询，保留 `rows_affected` 未命中错误与最终 `get_memory_by_id()` 回读。
- 2026-04-19: 执行 `CARGO_TARGET_DIR=target-codex-patch-1a cargo test --test test_user_intervention`，5 个测试全部通过。
- 2026-04-19: 执行完整 `cargo test` 时，并行构建触发 Windows 页面文件不足 / rustc OOM；串行 `cargo test -j1` 运行 20 分钟超时，未能完成完整回归。

### Completion Notes List

- 已将占位 patch story 重写为可执行的 dev-ready 文档。
- 已明确本 patch 的真实边界：删除预查询，保留最终回读与现有错误语义。
- 已补充 CLI/MCP 字符串错误映射这一隐藏约束，避免实现时误改错误文本导致回归。
- 已将 `mark_important()` 成功路径从“预查 + UPDATE + 回读”收敛为“UPDATE + 回读”。
- 已将 `mark_forget()` 成功路径收敛为同样的“UPDATE + 回读”流程。
- 已补齐 `Forget` 缺失记录无副作用测试，并保留 `Important` 缺失记录无副作用覆盖。
- 完整回归因本机 Windows 页面文件/内存限制未完成；本 story 相关聚焦测试已通过。

### File List

- `_bmad-output/implementation-artifacts/patch-1a-heat-performance.md`
- `_bmad-output/implementation-artifacts/sprint-status.yaml`
- `Laputa/src/vector_storage.rs`
- `Laputa/tests/test_user_intervention.rs`

## Change Log

- 2026-04-16: 创建占位 Story，覆盖 Epic 5 P1-P2
- 2026-04-19: 重写为 ready-for-dev 的完整 patch story，补充实现护栏、测试要求与上层错误映射约束
- 2026-04-19: 完成热度干预冗余预查询优化，补齐缺失记录测试并标记 ready for review
