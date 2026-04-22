# Deferred Work

This file tracks deferred findings from code reviews and other workflows.

---

## Deferred from: code review of 1-3-memoryrecord-extension (2026-04-14)

- `LaputaMemoryRecord` 字段过多（13个） — 设计决策，使用 `#[allow(clippy::too_many_arguments)]` 压制警告
- `score` 字段语义不一致 — pre-existing issue，初始化为 0.0 但读取时赋值为相似度 (1.0 - dist)
- `importance` 动态计算未持久化 — 设计决策，`compute_decayed_importance()` 每次查询重新计算
- 时间戳转换使用 `expect()` — 边界已覆盖，`unix_timestamp_to_datetime(0)` 理论上不会失败

## Deferred from: code review of 1-3-patch-heat-validation (2026-04-14)

- `heat_to_i32()` 是 pub 函数，外部可绕过验证直接写入越界值 [memory.rs:108-110] — AC#2 设计意图，调用方负责
- 越界测试未验证错误消息具体内容 [test_memory_record.rs:55-58,66-68] — AC 未要求，可未来测试质量提升时处理
- set_heat(99.999) 舍入后 get_heat() 返回 100.0，精度不对称 [memory.rs:108-110] — i32 缩放设计固有特性
- 测试未固化「错误后 heat_i32 不变」实现顺序约束 [test_memory_record.rs:57-58] — 现有断言已部分覆盖
- update_emotion 使用 clamp 静默修正与 set_heat 拒绝策略不一致 [memory.rs:84-85] — Story 1-3 原始设计决策，超出本次范围

## Deferred from: Epic 1 综合代码审查 (2026-04-16)

### Patch 待办项（需后续修复）

- [ ] **P1: user_name 未过滤路径遍历字符** [initializer.rs:39-43] — 安全漏洞，应添加 `../`、`..\` 检查防止路径注入
- [ ] **P2: user_name 未限制最大长度** [initializer.rs:38-43] — 资源耗尽风险，建议限制 256 字符上限
- [ ] **P3: user_name 未过滤控制字符** [initializer.rs:38-43] — 应拒绝 null 字符 `\x00` 及其他控制字符
- [ ] **P4: 初始化失败留僵尸数据库文件** [initializer.rs:57-66] — identity.md 写入失败时应清理已创建的 laputa.db
- [ ] **P5: MemoryInsert 缺少 heat_i32 边界验证** [vector_storage.rs:220-276] — add_memory_record 应验证 heat_i32 在 [0, 10000] 范围

### Deferred 级发现（pre-existing 或已记录）

- [x] Schema 更新无事务保护 [memory.rs:127-179] — mempalace-rs 继承代码，pre-existing
- [x] heat_to_i32 无边界检查 [memory.rs:122-124] — Story 1.3-patch 已记录为 defer (AC#2 设计意图)
- [x] heat_from_i32 不验证输入范围 [memory.rs:116-118] — 对称函数，pre-existing
- [x] LaputaMemoryRecord score 未映射数据库 [memory.rs:23] — mempalace-rs 继承字段，pre-existing
- [x] 两 schema 定义不一致 [sqlite.rs vs memory.rs] — 设计决策：sqlite.rs 最小 schema，memory.rs 扩展
- [x] LaputaError 缺少 From<anyhow::Error> [error.rs] — 已通过 From<io::Error> 间接处理，pre-existing
- [x] 错误信息丢失原始错误链 [error.rs:40-44] — Story 1.2 Review Findings 已记录
- [x] user_name 可包含 Markdown 特殊字符 [initializer.rs:62-64] — 非安全风险，UX 层面，pre-existing
- [x] RecallQuery 未验证时间戳范围 [searcher/recall.rs] — mempalace-rs 继承，pre-existing
- [x] truncate_embedding_input 空输入处理 [searcher/mod.rs] — mempalace-rs 继承，pre-existing
- [x] unix_timestamp_to_datetime 极端时间戳 [memory.rs:275-279] — 已正确返回错误，pre-existing
- [x] now_unix 系统时钟异常 panic [vector_storage.rs:100-105] — expect 为设计决策，pre-existing
- [x] ensure_memory_schema 缺少 CHECK 约束 [memory.rs] — 与 sqlite.rs 职责分离，已记录
- [x] Drop WAL checkpoint 错误被忽略 [vector_storage.rs] — mempalace-rs 继承，pre-existing
- [x] SQLite INTEGER 与 Rust u32 映射 [sqlite.rs:45] — SQLite 类型系统固有特性

## Deferred from: Epic 2 代码审查 (2026-04-16)

### Patch 待办项（需后续修复）

- [x] **P1: 合并后向量索引未更新** [memory_gate.rs:122-126, vector_storage.rs:607-623] — 功能性bug，合并后embedding指向旧内容 — **存档2026-04-16**
- [x] **P2: embedding重复计算** [diary/mod.rs:179-183] — MemoryGate和add_memory_record双重embedding — **存档2026-04-16**
- [x] **P3: search()事后过滤性能浪费** [vector_storage.rs:414-475] — 应预过滤discard_candidate — **存档2026-04-16**
- [x] **P4: 目录创建错误被忽略** [diary/mod.rs:419] — `let _ = create_dir_all` 需改为显式错误 — **存档2026-04-16**
- [x] **P5: threshold无范围校验** [memory_gate.rs:53-55] — 需添加∈[0.0,1.0]校验 — **存档2026-04-16**
- [x] **P6: Embeddings不可用降级信息丢失** [memory_gate.rs:78-89] — reason中应添加原始错误类型 — **存档2026-04-16**
- [x] **P7: 边界值测试不完整** [test_emotion_anchor.rs] — 补充valence/arousal/heat精确边界测试 — **存档2026-04-16**

### Deferred 级发现（pre-existing 或已记录）

- [x] VectorStorage无线程安全保护 [vector_storage.rs] — 需Mutex/Arc保护，超出Story范围
- [x] panic风险：系统时钟异常 [.expect] — 系统级问题，pre-existing
- [x] 并发写入竞态条件 [diary/mod.rs:290-337] — 需事务机制，超出Story范围

## Deferred from: Epic 4 代码审查 (2026-04-16)

### Deferred 级发现（pre-existing 或设计决策）

- [x] and_hms_opt expect不良实践 [weekly.rs:362-364] — UTC午夜总是有效，但应使用更安全的API
- [x] 正则每次调用重新编译 [weekly.rs:279] — 性能优化项，MVP阶段可接受
- [x] 停用词列表只含英文 [weekly.rs:312-356] — MVP仅支持英文关键词，中文需后续扩展
- [x] 去重策略未完全实现 [weekly.rs] — Dev Notes提到的合并相似内容未实现，MVP阶段
- [x] 锁文件删除错误被静默忽略 [scheduler.rs:412] — 陈旧锁文件残留风险，需后续添加清理机制
- [x] Mutex unwrap可能panic传播 [scheduler.rs:334,345] — 需PoisonError处理，超出Story范围
- [x] Cron解析器只支持有限模式 [scheduler.rs:139-142] — MVP仅支持weekly cron，完整cron需后续
- [x] 配置无section时返回default [scheduler.rs:47-109] — 设计决策，无rhythm section返回enabled=false
- [x] start_date > end_date未验证 [knowledge_graph/mod.rs:398-404] — 参数顺序检查可后续添加

## Deferred from: Epic 5 代码审查 (2026-04-16)

### Decision Needed → Deferred（用户选择方案A，暂不修复）

- [x] **D1: Story 5-1 vs 5-4 状态边界矛盾** — `heat_i32=2000` 在 `state.rs` 判断为 ArchiveCandidate (`>=cold_threshold`)，但在 `vector_storage.rs` 中不被标记 (`<threshold`)。用户选择方案A（统一为不含2000），暂不修复，后续统一规格。推迟原因：需跨Story规格文档同步修改，影响面大。

### Patch 待办项（需后续修复）

- [ ] **P1: mark_important 重复查询冗余** [vector_storage.rs:641-658] — 先 get_memory_by_id 检查存在性，后 UPDATE，再 get_memory_by_id 返回，两次查询冗余，应移除第一次查询
- [ ] **P2: mark_forget 重复查询冗余** [vector_storage.rs:661-678] — 同上，两次 get_memory_by_id 调用冗余
- [ ] **P3: emotion-anchor 边界测试缺失** [test_user_intervention.rs] — 缺失 `heat=9000→10000` (净增1000) 的裁剪边界测试

### Deferred 级发现（pre-existing 或已记录）

- [x] 规格文档边界表述不一致 [5-4-archive-candidate.md:20-21] — 规格层面问题，需在 Epic 回顾时统一
- [x] decay_rate NaN 处理 [config.rs:119] — 已正确处理 `is_nan()` 检查
- [x] epoch 时间戳处理 [decay.rs:5-9] — 已正确返回 0.0 days
- [x] access_count=0 热度塌缩 [decay.rs:20-22] — 已正确返回 base heat

### Decision-needed 级发现（已推迟）

- [x] 压缩比验证阈值偏低 [test_rhythm.rs:187] — AAAK压缩比受内容长度动态影响，10x为MVP最低可接受值
- [x] 关键词数量上限偏离规范 [weekly.rs:20] — MVP阶段12个关键词足够，后续可扩展至20
- [x] 高热度事件数量上限偏离规范 [weekly.rs:18] — MVP阶段12个事件足够，后续可扩展至20
- [x] 调度触发时间与规范表述偏差 [laputa.toml:26] — 凌晨2点给数据完整性缓冲，设计决策
- [x] max_retries=0行为语义混乱 [scheduler.rs:448] — 语义为"重试次数上限（不含首次）"，可后续文档化
- [x] 锁文件竞态条件 [scheduler.rs:388-396] — MVP单实例部署，后续需文件锁机制

## Deferred from: Epic 6 代码审查 (2026-04-16)

### CRITICAL 级发现（必须立即修复）

- [ ] **C1: CLI与MCP数据库路径不一致** [handlers.rs:55 vs mod.rs:536] — CLI diary write 使用 `vectors.db`，MCP laputa_diary_write 使用 `diary.db`，数据完全隔离。需检查 Diary 模块设计意图后统一。

### HIGH 级发现（本 Sprint 修复）

- [ ] **H1: CLI parse_memory_id 缺少正值验证** [handlers.rs:245-261] — MCP 版本 (mod.rs:1037-1048) 有 `id > 0` 检查，CLI 版本缺失，允许 `-1`, `0` 通过
- [ ] **H2: init 命令未验证空白用户名** [handlers.rs:42] — `--name "   "` 会创建空白 identity.md

### MEDIUM 级发现（下批修复）

- [ ] **M1: limit 参数无上下限约束** [commands.rs:44, mod.rs:568] — 可传入 0 或超大值
- [ ] **M2: parse_time_range 未验证极端日期** [handlers.rs:203] — 0001-01-01 或 9999-12-31 可能导致时间戳溢出
- [ ] **M3: parse_time_range 无跨度限制** [handlers.rs:236] — >365天范围可能返回数十万条记录
- [ ] **M4: 路径转字符串静默回退** [mod.rs:64] — `unwrap_or("knowledge.db")` 应改为显式错误

### LOW 级发现（可选优化）

- [ ] **L1: u64→usize 截断风险** [mod.rs:568] — 32位系统上大数值可能截断
- [ ] **L2: 目录创建错误被忽略** [mod.rs:56] — `let _ = create_dir_all` 需改为显式处理
- [ ] **L3: 重复错误消息字符串** [handlers.rs:141-150] — 可提取常量

### Deferred 级发现（pre-existing 或设计决策）

- [x] tags 字段不支持逗号转义 [handlers.rs:194] — MVP阶段 CSV 格式足够，后续可扩展引号支持
- [x] MarkCommand --id 类型为 String [commands.rs:95] — Phase 1 设计决策，支持 UUID 扩展预留
- [x] 不必要的 clone() [mod.rs:589, handlers.rs:82-91] — 性能优化项，MVP阶段可接受

### 验收标准验证结果

| Story | AC | 状态 |
|-------|----|----|
| 6.1 | AC1: CLI 子命令支持 | ✅ PASSED |
| 6.1 | AC2: 错误返回 LaputaError | ✅ PASSED |
| 6.2 | AC1: 6个 laputa_* 工具 | ✅ PASSED |
| 6.2 | AC2: JSON-RPC 2.0 格式 | ✅ PASSED |
| 6.2 | AC3: snake_case 参数 | ✅ PASSED |

### 缺失测试路径

- CLI parse_memory_id 负值/零值拒绝
- limit 参数上下限边界
- 极端日期时间范围
- 空白用户名拒绝
- 日期跨度过大拒绝

## Deferred from: Epic 7 代码审查 (2026-04-16)

### Story 7-1 Deferred 发现

- [x] confidence_to_resonance 边界转换存在歧义 [knowledge_graph/mod.rs:581-589] — 用户决策：方案A（添加文档注释），暂不修改代码，推迟原因：需跨模块规格文档同步
- [x] SQL动态拼接存在注入风险模式 [knowledge_graph/mod.rs:341-344] — 模式问题但非当前代码实际漏洞，id_placeholders 来自数字ID
- [x] 错误被静默忽略(create_dir_all) [knowledge_graph/mod.rs:19] — 需改为 .with_context() 处理，推迟到批量修复时处理
- [x] FromStr错误类型无信息(RelationKind) [relation.rs:30-37] — 需提供描述性错误类型，推迟原因：API变更需同步所有调用方
- [x] 时间线同日期空隙风险 [knowledge_graph/mod.rs:550-578] — 需使用精确时间戳，推迟原因：需跨Story时间格式统一

### Story 7-2 Deferred 发现

- [x] update_emotion静默clamp无越界警告 [storage/memory.rs:91-94] — 用户决策：方案A（添加log::warn!()），暂不修改代码，推迟原因：需与Resonance策略统一时同步修改
- [x] 数据库与索引不一致风险 [vector_storage.rs:220-276] — 需使用事务或补偿机制，推迟原因：需架构层面设计
- [x] 容量调整可能溢出 [vector_storage.rs:263-269] — 需使用 saturating_mul(2)，推迟原因：低频场景
- [x] Drop实现忽略WAL checkpoint错误 [vector_storage.rs:1279-1283] — 设计决策，mempalace-rs 继承
- [x] N+1查询性能隐患(relation_changes_between) [knowledge_graph/mod.rs:398-488] — 性能优化项，后续可用 JOIN 优化
- [x] valid_from=None时历史记录排序 [knowledge_graph/mod.rs:530] — 已有 COALESCE 处理，pre-existing
- [x] Resonance缺少Default trait [knowledge_graph/resonance.rs:6-26] — 非阻塞，可后续添加

### Story 7-2 Deferred 发现

- [x] 错误被静默吞没(embed_single) [searcher/mod.rs:137-139] — 优雅降级设计，embedder 不可用时返回空结果
- [x] canonical_emotion_code空字符串处理未文档化 [dialect/mod.rs:9-15] — 行为正确（返回 None），仅缺文档
- [x] limit=0边界处理不一致 [vector_storage.rs:716-719] — 两套策略都有合理理由：list_memories_by_emotion 返回空，get_memories 返回全部
- [x] EmotionQuery参数无验证 [vector_storage.rs:70-82] — 查询参数逻辑验证非必需，SQL 会正确返回空结果
- [x] 公共字段暴露内部状态(VectorStorage) [vector_storage.rs:127-131] — 设计决策，便于测试和扩展
- [x] Regex未缓存编译 [dialect/mod.rs:334] — 性能优化项，后续可用 lazy_static 缓存

### 验收标准验证结果

| Story | AC | 状态 |
|-------|----|----|
| 7.1 | AC1: 创建/更新关系节点 | ✅ PASSED |
| 7.1 | AC2: 时间三元组记录变化 | ✅ PASSED |
| 7.2 | valence ∈ [-100,100] | ✅ PASSED |
| 7.2 | arousal ∈ [0,100] | ✅ PASSED |
| 7.2 | 统一 MemoryRecord 读取 | ✅ PASSED |
| 7.2 | EMOTION_CODES 复用 | ✅ PASSED |
| 7.2 | 独立数值维度落库 | ✅ PASSED |
| 7.2 | 可复用情绪查询入口 | ✅ PASSED |

## Deferred from: Epic 8 代码审查 (2026-04-16)

### Story 8-1 Deferred 发现

- [x] SQLite路径注入风险 [vector_storage.rs:919-921] — 转义已覆盖主要风险，需架构级改动才能完全消除
- [x] 导出路径时间戳碰撞 [exporter.rs:57-58] — 同秒多次导出可能覆盖，需添加UUID或计数器，推迟原因：MVP单实例部署低概率
- [x] 状态记录原子性缺失 [exporter.rs:72-80] — 配置保存失败但导出成功，需事务性设计，推迟原因：超出Story范围
- [x] 元数据表无版本字段 [exporter.rs:107-112] — 后续版本迁移是后续Story职责
- [x] 配置保存失败恢复 [exporter.rs:72-80] — 需要事务性设计，超出当前范围

### Story 8-2 Deferred 发现

- [x] KnowledgeGraph连接未关闭 [full.rs:110-111] — 需显式drop或scoped连接，推迟原因：低频导出场景
- [x] VectorStorage重复初始化开销 [full.rs:177-181] — 性能优化，每次导出重建实例，推迟原因：非当前Story范围
- [x] 空JSONL语义模糊 [full.rs:228-233] — 空文件合法，manifest已标记数量，非问题
- [x] capsule导出仅支持单个 [full.rs:164-175] — Story规范允许多个但实现单个可接受
- [x] relation导出无ID字段 [full.rs:125-129] — 恢复流程是后续Story，当前实现可解析

### Edge Case Deferred 发现

- [x] 非UTF8路径处理 [full.rs:110] — 需添加路径验证错误，推迟原因：Windows路径通常UTF8
- [x] Windows路径反斜杠 [vector_storage.rs:919] — ATTACH语法风险，需rusqlite原生路径绑定，推迟原因：需架构改动
- [x] remove_file目录失败 [exporter.rs:148-151] — 需添加目录检测逻辑，推迟原因：默认路径不会冲突
- [x] timestamp()安全 [full.rs:219] — 需使用timestamp_millis()更安全，推迟原因：数据已验证
- [x] 全库内存加载OOM [full.rs:184] — get_memories(usize::MAX)风险，需chunked加载，推迟原因：性能优化项
- [x] 导出目录删除无确认 [full.rs:263-267] — prepare_export_dir静默删除，推迟原因：导出幂等操作覆盖合理

### 验收标准验证结果

| Story | AC | 状态 |
|-------|----|----|
| 8.1 | AC1: 独立SQLite导出文件 | ✅ PASSED |
| 8.1 | AC2: 仅候选+最小元数据 | ✅ PASSED |
| 8.1 | AC3: 主库数据不变 | ✅ PASSED |
| 8.1 | AC4: 可审计路径结构 | ✅ PASSED |
| 8.1 | AC5: 候选标记不清空 | ✅ PASSED |
| 8.1 | AC6: 空候选返回明确错误 | ✅ PASSED |
| 8.1 | AC7: 失败不记录假状态 | ✅ PASSED |
| 8.2 | AC1: 所有组件完整导出 | ✅ PASSED |
| 8.2 | AC2: 文档化结构格式 | ✅ PASSED |
| 8.2 | AC3: identity.md优先读取 | ✅ PASSED |
| 8.2 | AC4: 从KG生成relation.md | ✅ PASSED |
| 8.2 | AC5: capsule fallback状态记录 | ✅ PASSED |
| 8.2 | AC6: 高热阈值过滤正确 | ✅ PASSED |
| 8.2 | AC7: 主库只读不改写 | ✅ PASSED |
| 8.2 | AC8: 失败不记录假路径 | ✅ PASSED |

### 审查总结

- Decision-needed: 0 (全部归入defer)
- Patch: 0 (全部归入defer)
- Deferred: 18
- Dismissed: 4

**结论：Epic 8 的两个 Story 全部验收标准通过，18项发现均推迟至后续优化处理。**

## Deferred from: code review of patch-1b-heat-validation (2026-04-19)

- 缺失极端负值/大正值测试 [vector_storage.rs:1252-1261] — 测试覆盖建议，当前测试覆盖 -1/0/10000/10001，但未覆盖如 `-999999` 或 `99999999` 等极端值（逻辑正确，可增加防御性测试）
- heat_to_i32 潜在溢出风险 [memory.rs:122-124] — pre-existing，`f64 * 100` 理论上可超出 i32 范围，本 patch 未涉及该函数修改

## Deferred from: code review of patch-1c-test-supplement (2026-04-19)

- 缺少 heat 负值边界测试 [test_user_intervention.rs:105-109] — 建议 `heat=-1` 负值输入场景验证，生产代码可能已有其他防护，AC 未要求
- 缺少 valence/arousal 精确边界值测试 [test_user_intervention.rs:105-109] — 当前测试覆盖超限裁剪场景（150/-150/120），精确边界值测试（100/-100/100）可作为后续优化

### 审查结论

所有 AC 通过，2项测试覆盖建议推迟至后续优化处理。

## Deferred from: code review of epic-1-patch-security-validation (2026-04-19)

### Defer 级发现（超出当前 Story 范围）

- [x] URL编码变体路径遍历未覆盖 [initializer.rs:91-98] — deferred，Story 明确限定为字符串模式检测（P1-P5），编码变体不在原始 deferred 批次范围
- [x] TOCTOU 竞态条件 [initializer.rs:42-46] — deferred，多进程并发初始化不在 MVP 范围，Story 范围限定为输入校验与失败清理
- [x] identity.md 内容未进行 YAML 转义 [initializer.rs:75-78] — deferred，非安全风险，Markdown 注入在此场景风险低，不在 AC 范围

### 审查统计

- Patch: 3（需立即修复）
- Defer: 3（超出范围）
- Dismissed: 2（测试覆盖建议）
- Decision-needed: 0

**结论：AC 2、4、5 完全满足验收要求，AC 1、3 实现正确但测试覆盖不完整。发现 3 个 heat 验证绕过漏洞需修复。**

## Deferred from: code review of patch-2-cli-mcp-critical (2026-04-19)

### Defer 级发现（超出当前 Story 范围或延后处理）

- [x] 路径穿越风险：用户名未验证路径分隔符 [cli/handlers.rs:45-51] — deferred, pre-existing，需跨模块统一设计路径安全策略，非本patch范围
- [x] H1: MCP JSON类型混淆无明确错误 [mcp_server/mod.rs:1057-1090] — deferred，低频边缘场景，已有基础验证，可后续增强错误消息
- [x] H1: 全宽数字/科学记数法未明确提示 [cli/handlers.rs:260-281] — deferred，parse失败已有ValidationError，消息可后续优化
- [x] M4: CLI未显式路径转换 [cli/handlers.rs:68] — deferred，错误会在深层抛出，非最外层验证
- [x] i64.MAX内存ID误导错误 [mcp_server/mod.rs:1057-1090] — deferred，超出i64范围的数值返回模糊错误，低频场景

### 审查统计

- Patch: 2（需立即修复）
- Defer: 5（延后处理）
- Dismissed: 5（噪音/设计决策）
- Decision-needed: 0

**结论：AC C1/H1/H2/M1/M2/M3/M4 中，H2和M1发现实现缺陷需修复，其余AC通过。**

---

## Deferred from: code review of 9-3-clean-server-migration-validation (2026-04-21)

- 验收记录缺少时间戳和版本信息 — Debug Log 已有时间记录，环境版本信息缺失不影响验收结论有效性
- 首次失败根因判断缺少验证 — 并行触发导致失败的推断，串行执行后链路已通过，不影响验收结论
- 验收记录缺少阻断原因格式规范 — 失败记录部分已有内容，格式问题不阻断验收

---

## Deferred from: code review of 9-1-standalone-build-decoupling (2026-04-21)

### vendor/usearch 上游库问题 (pre-existing)

- `unwrap()` on `CARGO_CFG_TARGET_OS` [vendor/usearch/build.rs:61] — pre-existing upstream issue，Cargo 正常设置此变量
- Windows `/sdl-` 禁用安全检查 [vendor/usearch/build.rs:73] — pre-existing upstream issue，上游库编译优化决策
- MSVC 兼容性定义抑制类型安全 [vendor/usearch/build.rs:74-75] — pre-existing upstream issue，`_ALLOW_*` 定义为解决 ABI 问题
- 编译重试循环非理想错误处理 [vendor/usearch/build.rs:100] — pre-existing upstream issue，上游库 SIMD fallback 设计
- 未处理 OS 目标缺少编译标志 [vendor/usearch/build.rs:63-90] — pre-existing upstream issue，非主流 OS 走默认配置
- 未知架构默认 x86 SIMD 目标 [vendor/usearch/build.rs:27] — pre-existing upstream issue
- `flag_if_supported` 静默忽略 [vendor/usearch/build.rs:65-89] — pre-existing upstream issue，cc crate 特性
- b1x8 二进制向量维度不匹配 [vendor/usearch/rust/lib.cpp] — pre-existing upstream issue，binary vector 特殊处理
- 错误消息误导 [vendor/usearch/rust/lib.cpp:121-125] — pre-existing upstream issue
- 临时字符串 c_str() 生命周期风险 [vendor/usearch/rust/lib.cpp:175-179] — pre-existing upstream issue
- 缓冲区操作无边界检查 [vendor/usearch/rust/lib.cpp:185-195] — pre-existing upstream issue
- unsafe 函数指针 cast [vendor/usearch/rust/lib.cpp:85-91] — pre-existing upstream issue
- Drop 实现潜在 double-free [vendor/usearch/rust/lib.rs:536-558] — pre-existing upstream issue
- unsafe Send/Sync 绕过线程安全验证 [vendor/usearch/rust/lib.rs:533-534] — pre-existing upstream issue
- filtered_search 闭包生命周期风险 [vendor/usearch/rust/lib.rs:729-740] — pre-existing upstream issue
- change_metric panic 非优雅终止 [vendor/usearch/rust/lib.rs:764] — pre-existing upstream issue
- MetricFunction 双重指针间接 [vendor/usearch/rust/lib.rs:485-490] — pre-existing upstream issue

### 审查统计

- Decision-needed: 1（AC3/AC4 验证）
- Patch: 2（测试代码改进）
- Defer: 17（上游库问题）
- Dismissed: 2

**结论：AC1、AC2、AC5 通过，AC3/AC4 需执行验证确认，vendor/usearch 发现均为上游库 pre-existing 问题。**

---

## Deferred from: code review of 9-2-repo-metadata-doc-independence (2026-04-21)

### Deferred 级发现（pre-existing 或测试增强建议）

- [x] panic! 错误信息不友好 [test_repo_metadata.rs:6] — 测试框架行为，非 AC 相关，pre-existing 风格问题
- [x] 断言语义验证不足 [test_repo_metadata.rs:67-69] — 测试设计改进项，当前实现满足 AC，属于增强而非修复
- [x] mcp_rs 依赖源未明确 [Cargo.toml:33] — pre-existing 依赖配置，不在本 Story scope
- [x] 变体路径检测遗漏 [test_repo_metadata.rs:46-64] — 测试覆盖增强，当前 AC 已满足（文档无 ../ 变体）
- [x] DECISIONS.md 链接可能失效 [README.md:58] — pre-existing 文档链接问题，不在本 Story scope
- [x] vendor/usearch 目录存在性 [Cargo.toml:55-56] — pre-existing build 配置，已在 Story 9-1 处理 standalone build

### 审查统计

- Decision-needed: 0
- Patch: 0
- Deferred: 6
- Dismissed: 2

**结论：所有验收标准 PASS，Clean Review。发现 6 项均为 pre-existing 或测试增强建议。**
