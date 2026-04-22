# Story 10.2: CLI/MCP 数据库路径统一

**Story ID:** 10.2  
**Story Key:** 10-2-cli-mcp-db-path-unification  
**Status:** ready-for-dev  
**Created:** 2026-04-22  
**Project:** 天空之城 (Laputa)

---

## Story

As a **开发者**，
I want **统一 CLI 与 MCP 使用单一数据库文件 `laputa.db`**，
So that **数据不会因模块不同而分散，且用户有一个清晰单一的数据存储点**。

---

## 验收标准

1. **Given** 当前存在两个数据库文件：
   - Identity 模块使用 `laputa.db`（正确）
   - 其他模块（CLI、MCP、Diary）使用 `vectors.db`（需统一）
   
   **When** 执行数据库路径统一修复
   **Then** 所有模块统一使用 `laputa.db`

2. **And** 所有相关代码路径同步修改：
   - CLI handlers (`src/cli/handlers.rs`)
   - MCP handlers (`src/mcp_server/mod.rs`)
   - Diary 模块 (`src/diary/mod.rs`)
   - 其他相关模块（storage、searcher、wakeup、rhythm 等）

3. **And** 向量索引文件名同步改为 `laputa.usearch`

4. **And** 单元测试验证路径一致性

5. **And** 修复后端到端链路仍能正常通过：
   - `laputa init --name "测试用户"`
   - `laputa diary write --content "测试日记"`
   - `laputa wakeup`
   - `laputa recall --time-range "today"`

---

## Tasks / Subtasks

- [ ] **Task 1: 定义统一数据库常量** (AC: #1, #2)
  - [ ] 在 `src/config.rs` 或新建 `src/constants.rs` 定义 `DB_FILE_NAME = "laputa.db"`
  - [ ] 定义 `USEARCH_INDEX_NAME = "laputa.usearch"`

- [ ] **Task 2: 修改 Diary 模块** (AC: #2)
  - [ ] 修改 `src/diary/mod.rs:memory_db_path()` 返回 `laputa.db`
  - [ ] 修改 `src/diary/mod.rs:index_path()` 返回 `laputa.usearch`

- [ ] **Task 3: 修改 CLI handlers** (AC: #2)
  - [ ] 修改 `src/cli/handlers.rs:handle_diary()` 使用统一路径
  - [ ] 修改 `src/cli/handlers.rs:handle_mark()` 使用统一路径

- [ ] **Task 4: 修改 MCP handlers** (AC: #2)
  - [ ] 修改 `src/mcp_server/mod.rs:laputa_diary_write()` 使用统一路径
  - [ ] 搜索并修改所有 MCP 中使用 `vectors.db` 的位置

- [ ] **Task 5: 修改其他相关模块** (AC: #2)
  - [ ] `src/storage/mod.rs`
  - [ ] `src/searcher/mod.rs`
  - [ ] `src/wakeup/mod.rs`
  - [ ] `src/rhythm/weekly.rs`
  - [ ] `src/vector_storage.rs`
  - [ ] `src/export/full.rs`

- [ ] **Task 6: 更新测试文件** (AC: #4)
  - [ ] 搜索所有测试文件中的 `vectors.db` 并改为 `laputa.db`
  - [ ] 确保测试 fixture 使用统一路径

- [ ] **Task 7: 验证端到端链路** (AC: #5)
  - [ ] 运行 `cargo test --test test_cli_flow`
  - [ ] 手工验证 CLI 链路：init → diary write → wakeup → recall

---

## Dev Notes

### 当前数据库路径分布

根据代码搜索，当前各模块使用的数据库路径如下：

| 模块 | 文件位置 | 当前路径 | 应改为 |
|------|----------|---------|--------|
| Identity | `src/identity/initializer.rs:10` | `laputa.db` | 保持不变 |
| Diary | `src/diary/mod.rs:273` | `vectors.db` | `laputa.db` |
| CLI handlers | `src/cli/handlers.rs:68,135` | `vectors.db` | `laputa.db` |
| MCP handlers | `src/mcp_server/mod.rs:536` | `vectors.db` | `laputa.db` |
| Storage | `src/storage/mod.rs:17` | `vectors.db` | `laputa.db` |
| Searcher | `src/searcher/mod.rs:47` | `vectors.db` | `laputa.db` |
| Wakeup | `src/wakeup/mod.rs:96` | `vectors.db` | `laputa.db` |
| Rhythm weekly | `src/rhythm/weekly.rs:33,113,163` | `vectors.db` | `laputa.db` |
| Vector storage | `src/vector_storage.rs:1299` | `vectors.db` | `laputa.db` |
| Export full | `src/export/full.rs:178` | `vectors.db` | `laputa.db` |

### 向量索引文件

- 当前：`vectors.usearch`
- 应改为：`laputa.usearch`

### 关键修改位置清单

**源码文件（需修改）：**
```
Laputa/src/diary/mod.rs:273,276-277
Laputa/src/cli/handlers.rs:68,135-136
Laputa/src/mcp_server/mod.rs:536,652,661,918,946,1940,2596
Laputa/src/storage/mod.rs:17
Laputa/src/searcher/mod.rs:47
Laputa/src/wakeup/mod.rs:96
Laputa/src/rhythm/weekly.rs:33,113,163
Laputa/src/vector_storage.rs:1299
Laputa/src/export/full.rs:178
```

**测试文件（需修改）：**
```
Laputa/tests/test_wakepack.rs:55,128,166
Laputa/tests/test_vector_storage_validation.rs:30,59
Laputa/tests/test_user_intervention.rs:34,57,80,122,156
Laputa/tests/test_timeline_recall.rs:52,102,132,151,189
Laputa/tests/test_semantic_search.rs:57,95,160,223
Laputa/tests/test_rhythm.rs:74,194
Laputa/tests/test_memory_record.rs:129
Laputa/tests/test_memory_gate.rs:11,54,90
Laputa/tests/test_hybrid_search.rs:146
Laputa/tests/test_heat.rs:261,309,361
Laputa/tests/test_export_full.rs:44,175
Laputa/tests/test_emotion_dimension.rs:39,61,76,107
Laputa/tests/test_emotion_anchor.rs:24,48,64,80
Laputa/tests/test_archiver.rs:38,96,159,223,314,340
```

### Architecture Compliance

根据架构文档 (ADR-011)：
- 配置管理使用 `config.toml`
- 数据库路径应为 `laputa.db`（架构文档第 584 行已指定）

架构文档 `storage.db_path = "./laputa.db"` (line 584) 已明确使用 `laputa.db`，代码需与此保持一致。

### 实现策略建议

1. **常量优先策略**：
   - 在 `src/config.rs` 或新建 `src/constants.rs` 定义全局常量
   - 所有模块引用该常量，而非硬编码字符串

2. **向后兼容考虑**：
   - 可考虑在初始化时检测旧 `vectors.db` 并迁移
   - 或在配置文件中支持路径覆盖

3. **搜索替换注意事项**：
   - 使用全局常量而非硬编码字符串
   - 确保索引文件 `.usearch` 同步改名
   - 注意测试 fixture 中的路径

### Previous Story Intelligence

**Story 9.3 (新服务器独立运行验收)** 已完成：
- 仓库可独立 clone、构建、运行
- 最小 CLI 链路验证通过：init → diary write → wakeup
- Epic 9 完成后 Epic 10 开始

**Deferred Work (`deferred-work.md`)** 已记录：
- C1: 数据库路径不一致问题（本 Story 解决）

### Project Structure Notes

```
Laputa/
├── src/
│   ├── identity/       # 使用 laputa.db (正确)
│   ├── diary/          # 使用 vectors.db (需修改)
│   ├── cli/            # 使用 vectors.db (需修改)
│   ├── mcp_server/     # 使用 vectors.db (需修改)
│   ├── storage/        # 使用 vectors.db (需修改)
│   ├── searcher/       # 使用 vectors.db (需修改)
│   ├── wakeup/         # 使用 vectors.db (需修改)
│   ├── rhythm/         # 使用 vectors.db (需修改)
│   ├── vector_storage/ # 使用 vectors.db (需修改)
│   └── config.rs       # 配置管理
```

### References

- [Source: architecture.md#ADR-011] - 配置管理策略，db_path = "./laputa.db"
- [Source: architecture.md#5.1] - 命名模式规范
- [Source: epics.md#Story 10.2] - Story 定义
- [Source: deferred-work.md#C1] - Deferred 问题记录

---

## Dev Agent Record

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

(待开发时填写)

### Completion Notes List

(待开发时填写)

### File List

(待开发时填写)

---

## Change Log

- 2026-04-22: Story 创建，状态设置为 ready-for-dev