---
stepsCompleted:
  - step-01-init
  - step-02-discovery
  - step-02b-vision
  - step-02c-executive-summary
  - step-03-success
  - step-04-journeys
  - step-05-domain
  - step-06-innovation
  - step-07-project-type
  - step-08-scoping
  - step-09-functional-requirements
  - step-10-assumptions
  - step-11-complete
inputDocuments:
  - D:/VIVYCORE/newmemory/Laputa/README.md
  - D:/VIVYCORE/newmemory/Laputa/DECISIONS.md
  - D:/VIVYCORE/newmemory/agent-diva/README.md
  - D:/VIVYCORE/newmemory/_bmad-output/planning-artifacts/prd.md
  - D:/VIVYCORE/newmemory/_bmad-output/implementation-artifacts/sprint-status.yaml
workflowType: 'prd'
classification:
  projectType: Rust AI Agent 框架集成
  domain: AI 记忆系统
  complexity: 高
  projectContext: brownfield
---

# Product Requirements Document - Laputa 记忆系统接入 agent-diva

**Author:** 大湿  
**Date:** 2026-04-14

## Executive Summary

本 PRD 描述将 Laputa（天空之城）记忆系统完全接入并替代 agent-diva 现有记忆模块的产品需求和技术方案。

**核心目标：**
- 完全重写 agent-diva 的记忆系统，用 Laputa 作为唯一记忆底座
- 在 agent-diva 的 agent loop context assembly 阶段注入 Laputa 唤醒包
- 通过并行开发策略，Laputa 预留集成接口，agent-diva 同步改造
- 先用最小化集成测试项目（基于 agent-diva 解耦架构）验证方案，再扩展到完整 agent-diva

**为什么做：**
- agent-diva 现有的 memory/session 系统只是骨架，不成熟
- Laputa 提供完整的记忆生命周期管理（时间流、热度、情绪、节律整理、归档）
- 两个项目理念一致：agent-diva 从 nanobot 演化，Laputa 为 agent-diva 记忆模块开发
- 解耦设计允许渐进式替换，降低风险

**关键决策：**
1. **完全替代**：删除 agent-diva 原有 memory 内容，完全重写
2. **上下文注入**：在 agent loop 的 context assembly 阶段注入唤醒包（不是 MCP 调用）
3. **并行开发**：Laputa Phase 1 预留接口，agent-diva 同步改造
4. **最小化验证**：创建独立集成测试项目，避免现有高耦合模块干扰

## Executive Summary

本 PRD 描述将 Laputa（天空之城）记忆系统完全接入并替代 agent-diva 现有记忆模块的产品需求和技术方案。这是一次架构级技术升级，实现用户长期以来对 agent-diva 记忆系统的强烈呼吁，将 agent-diva 从"无记忆的工具"转变为"有记忆、有温度的 AI 伙伴"。

agent-diva 现有的 memory/session 系统只是骨架，无法满足用户对长期连续性的需求。Laputa 基于 mempalace-rs 演化，提供完整的记忆生命周期管理：五层记忆栈（L0-L4）、时间流主轴、热度衰减机制、情绪编码、节律整理与归档。通过将 Laputa 在 agent-diva 的 agent loop context assembly 阶段注入唤醒包（<1200 tokens），实现真正的记忆觉醒体验。

集成策略采用完全替代而非修补：删除 agent-diva 原有 memory 模块内容，用 Laputa 作为唯一记忆底座。首发通过 TUI 通道验证集成方案，先用最小化集成测试项目（基于 agent-diva 解耦架构）验证核心假设，再扩展到完整 agent-diva。并行开发策略要求 Laputa Phase 1 预留集成接口，agent-diva 同步改造，通过接口契约先行、特性开关隔离、频繁集成验证降低风险。

### What Makes This Special

**记忆觉醒时刻：** 用户第一次启动集成后的 agent-diva 时会惊喜发现："哇，原来 agent-diva 开始记事情了！"这是从 Stateless 工具到 Stateful 伙伴的质变。

**有温度的记忆：** 不是冷冰冰的聊天记录检索，而是通过情绪锚定（valence + arousal + EMOTION_CODES）和节律整理（周级胶囊），让记忆像人一样会判断重要性、会遗忘、会整理、有情绪色彩。

**架构验证：** 证明 agent-diva 解耦设计的价值——记忆模块可以完全替换而不破坏核心架构，为未来更多模块升级铺路。

**用户呼声实现：** 直接回应社区对记忆系统的强烈需求，按里程碑持续迭代 agent-diva 能力，从"能用"到"好用"的关键一步。

## Project Classification

| 维度 | 分类 |
|------|------|
| **项目类型** | Rust AI Agent 框架集成 |
| **领域** | AI 记忆系统 |
| **复杂度** | 高 |
| **项目上下文** | Brownfield（两个现有项目集成改造）|
| **首发通道** | TUI（终端界面）|

## Success Criteria

### User Success

1. **记忆觉醒时刻** - 用户启动 TUI 时发现 agent 记住了上次对话，产生"哇，原来 agent-diva 开始记事情了！"的惊喜
2. **跨会话连续性** - 写入的日记能在下次会话中被正确引用，不需要重新解释上下文
3. **有温度的记忆体验** - 情绪锚定和节律整理让用户感受到记忆的生命力，不是冷冰冰的检索
4. **无缝集成体验** - 用户不需要改变使用习惯，记忆能力自然增强

### Business Success

1. **双版本交付** - 缝合版（agent-diva 改造）和独立版（Laputa TUI）都完成
2. **按里程碑迭代** - Phase 1（TUI 验证）→ Phase 2（完整接入）→ Phase 3（高级特性）
3. **社区需求响应** - 实现用户对记忆系统的强烈呼吁，提升 agent-diva 用户满意度
4. **架构验证** - 证明解耦设计允许模块替换，为未来更多模块升级建立模式

### Technical Success

1. **唤醒包约束** - <1200 tokens 硬约束 100% 满足（运行时断言）
2. **数据一致性** - 日记写入→读取一致性 = 100%
3. **双版本验证** - 缝合版和独立版都通过集成测试
4. **质量门禁** - 零 flaky tests，CI 通过率 100%，Clippy 零警告
5. **性能要求** - TUI 启动→注入→检索全流程 <5s
6. **降级路径** - Laputa 失败时不导致 agent-diva 崩溃，有合理的 fallback 策略

### Measurable Outcomes

| 指标 | 目标 | 测量方式 |
|------|------|----------|
| 唤醒包注入成功率 | 100% | 集成测试 |
| Token 预算 | ≤1200 | 运行时断言 |
| 数据一致性 | 100% | 写入→读取测试 |
| 端到端延迟 | <5s | E2E 测试 |
| 双版本完成 | 2/2 | 交付清单 |
| 单元测试覆盖率 | ≥80% | CI 报告 |
| Flaky tests | 0 | CI 报告 |

## Product Scope

### MVP - Minimum Viable Product

**独立版（Laputa TUI）：**
- Laputa crate + 轻量 TUI 实现
- 唤醒包注入验证（<1200 tokens）
- 基本日记写入和读取
- 跨会话上下文保留
- 内存 mock 存储（不依赖真实 SQLite）

**缝合版（agent-diva 改造）：**
- 删除 agent-diva 原有 memory 模块
- 集成 Laputa crate 到 agent-diva workspace
- agent-diva TUI 能正常启动和注入唤醒包
- 接口契约定义和 Mock 实现
- 特性开关支持新旧系统切换

**共同要求：**
- 接口契约先行（MemoryProvider trait）
- 单元测试覆盖率 ≥80%
- 集成测试通过
- CI/CD 管道建立

### Growth Features (Post-MVP)

- 情绪锚定（valence + arousal + EMOTION_CODES）
- 节律整理（周级胶囊生成）
- 热度衰减机制
- 语义检索（向量搜索）
- 基本记忆管理命令（查看、搜索、标记）

### Vision (Future)

- 完整五层记忆栈（L0-L4）
- L4 归档层和考古工具
- soul 演化机制
- 多身份切换
- 多通道支持（Telegram、Discord、Slack 等）
- 月/季/年压缩链
- 艾宾浩斯反向复习
- 关系驱动召回

## User Journeys

### Journey 1: Primary User - First Memory Awakening

**User:** Alex, 25, independent developer using agent-diva for Rust project development

**Opening Scene:**
Alex has been using agent-diva for a week to assist with Rust project development. Every time he starts a new session, he has to re-explain project background, architecture decisions, and current progress. Yesterday he spent 15 minutes redescribing a complex module design problem, and today he needs to repeat it all over again. He sighs: "If only agent-diva could remember..."

**Rising Action:**
This morning, Alex updates agent-diva to the latest version and sees in the changelog: "Memory system integration (experimental)." Skeptical, he launches the TUI, preparing to re-explain the project context as usual.

**Climax (The Awakening Moment):**
```
Alex: Hi, I need to continue optimizing the memory management module from yesterday
Agent: Sure! I see yesterday we discussed using Arc<Mutex<T>> instead of Rc<RefCell<T>>,
       and you mentioned concurrency access conflicts in the Laputa project. Want to continue
       optimizing that module?
Alex: (surprised) Wait, you remember?!
Agent: Yes, I've loaded our last session context. Let me pull up the key decision points...
```

Alex pauses for a few seconds, then smiles: "Wow, this feels completely different!"

**Resolution:**
Alex no longer needs to re-explain context. The agent proactively references yesterday's discussion, even gently reminding Alex when he diverges: "This approach conflicts with yesterday's decision, want to review first?" Alex feels like agent-diva has transformed from a cold tool into a "partner with memory."

**Emotional Arc:** Skepticism (changelog) → Habitual preparation to repeat → Shock (agent proactively references) → Surprise → Relief → Excitement

---

### Journey 2: Primary User - Long-term Usage (Emotional Anchoring & Rhythm Organization)

**User:** Sarah, 30, product manager using agent-diva for requirements analysis and PRD writing

**Opening Scene:**
Sarah has been using the integrated agent-diva for two weeks. She records requirement discussions, user feedback, and priority decisions daily.

**Rising Action:**
One day, when Sarah launches agent-diva, the agent proactively says:

```
Agent: Good morning, Sarah! I noticed this week we mainly discussed 3 core features:
       1. Memory system upgrade (emotional anchoring) - You were very excited 😊
       2. TUI performance optimization - You had some concerns, but progress is smooth
       3. User research plan - You felt stressed and needed help breaking down tasks
       
       Want to see this week's progress summary first?
```

**Climax:**
Sarah sees that the agent not only remembers content but also captures her emotional changes:
- When discussing the memory system, her valence was +0.8 (excited)
- When discussing performance issues, arousal was 0.6 (anxious but controllable)
- Rhythm organization generated "weekly capsules", automatically extracting key decisions

Sarah remarks: "This doesn't feel like searching chat logs, more like talking to a colleague who understands me."

**Resolution:**
Sarah starts relying on this "warm memory." When stressed, she asks the agent: "How's my emotional state recently? Any patterns?" The agent provides insights based on emotional encoding and rhythm organization. Sarah feels agent-diva has become her "thinking partner."

---

### Journey 3: Primary User - Error Recovery (Degradation Path)

**User:** Mike, 28, DevOps engineer using agent-diva to monitor production systems

**Opening Scene:**
Mike is using agent-diva to troubleshoot a production issue when suddenly Laputa service times out due to database connection pool exhaustion.

**Rising Action:**
```
Mike: Help me check yesterday's discussion about that memory leak
Agent: (Laputa response timeout)
       Sorry, memory system is temporarily unavailable. But I can:
       1. Use locally cached last context (may not be latest)
       2. You describe the problem manually, and I'll start analyzing immediately
       Please choose, or should I just start with option 2?
```

**Climax:**
Mike doesn't panic. The agent's degradation strategy allows him to continue working:
- Local cache provides basic context
- Agent clearly communicates the degraded state
- Provides a clear recovery path

**Resolution:**
30 seconds later, Laputa service recovers. Agent proactively notifies: "Memory system has recovered, now loading full context... Okay, found yesterday's discussion." Mike feels that even when errors occur, the system has reasonable fallbacks that don't disrupt his workflow.

---

### Journey 4: Developer - Integration Verification

**User:** Dev Chen, 32, Rust developer responsible for integrating Laputa into agent-diva

**Opening Scene:**
Dev Chen needs to verify that Laputa and agent-diva integration works correctly. He creates a minimal integration test project, depending only on core, agent, and providers crates.

**Rising Action:**
```bash
# Create test project
cargo new laputa-integration-test --bin
cd laputa-integration-test

# Add local workspace or published dependencies for an integration-only harness
# The exact dependency source is an integration environment choice, not a Laputa runtime prerequisite.

# Run tests
cargo test -- wakeup_injection
```

**Climax:**
Tests pass, Dev Chen sees:
```
test wakeup_injection::test_token_budget ... ok (1187/1200 tokens)
test wakeup_injection::test_context_load ... ok
test diary::test_write_read_consistency ... ok (100%)
```

He breathes a sigh of relief, then runs the full integration test suite:
```
Running 47 tests...
47 passed, 0 failed
Coverage: 82.3%
Clippy warnings: 0
```

**Resolution:**
Dev Chen writes in the PR: "Dual version verification passed (standalone + integrated), all quality gates satisfied." He merges the PR and starts preparing for Phase 2 full integration.

---

### Journey Requirements Summary

| Journey | Revealed Capability Requirements |
|---------|----------------------------------|
| First Memory Awakening | Wakeup package injection, cross-session context retention, proactive history reference |
| Long-term Usage | Emotional anchoring, rhythm organization, weekly capsule generation, pattern recognition |
| Error Recovery | Degradation strategy, local cache, service recovery notification, clear user prompts |
| Developer Integration | Interface contracts, Mock implementation, integration test framework, quality gates |

## Domain-Specific Requirements

### 1. Temporal Accuracy & Precise Indexing

**Core Principle: Return exactly what the user asks for, never confuse, fabricate, or misplace.**

#### 1.1 Strict Time Boundary Isolation
- "Yesterday" = only retrieve memories from the past 24 hours
- "Last week" = only retrieve memories from the past 7 days
- **Prohibited**: Returning content from different time periods just because of semantic similarity
- Test case: User asks "the bug we discussed yesterday", system must NEVER return bug discussions from a month ago

#### 1.2 Precise Indexing Strategy
- **Time index first**: All queries must filter by time range first
- **Semantic index secondary**: Within the time-filtered candidate set, sort by semantic similarity
- **Hybrid retrieval sequence**:
  1. Parse user's temporal intent (yesterday/last week/recent/specific date)
  2. Filter candidates by time range
  3. Perform semantic matching within candidates
  4. Sort by relevance + freshness combined score

#### 1.3 Timestamp Precision
- Every memory record must have precise creation timestamp (Unix timestamp + timezone)
- Support relative time queries ("last time we discussed X")
- Time queries must be based on real time, not storage order

#### 1.4 Context Window Management
- When injecting wakeup package, must sort by time (most recent first)
- When token budget is tight, prefer discarding old memories, keep recent ones
- Explicitly label memory "freshness" tags (today/yesterday/last week/earlier)

---

### 2. Data Privacy & Security

#### 2.1 Local-First Storage
- Memory data stored locally in SQLite by default, not uploaded to cloud
- Users clearly know where their data is and how to access it

#### 2.2 LLM Provider Privacy
- When wakeup package is injected into prompt, users need to know this data will be sent to LLM provider
- Sensitive information (passwords, keys, personal identifiable information) should not be written to memory system

---

### 3. Technical Constraints

#### 3.1 Performance Requirements
- Wakeup package injection latency <5s (defined in success criteria)
- Single memory retrieval <500ms
- Batch retrieval (10 items) <2s

#### 3.2 Data Consistency
- Diary write→read consistency = 100%
- Crash recovery: SQLite transaction integrity guarantee
- Backup mechanism: Users can export memory data

#### 3.3 Degradation Strategy
- When Laputa fails, agent-diva continues with local cache
- Clearly inform user when in degraded state
- Auto-sync when service recovers

---

### 4. Risk Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| **Temporal confusion** | User trust collapses | Hard time boundary constraints + strict testing |
| **Hallucinated memories** | Agent fabricates non-existent memories | All references must have traceable memory IDs |
| **Privacy leakage** | Sensitive info seen by LLM provider | Sensitive info filtering + local storage |
| **Data膨胀** | Storage cost increase, slower retrieval | Heat decay + archival mechanism |
| **Retrieval failure** | User doesn't get desired memories | Degradation strategy + clear error messages |

## Innovation & Novel Patterns

### Detected Innovation Areas

1. **Temporal-First Memory Architecture**
   - Time boundary isolation as hard constraint, not soft preference
   - Hybrid retrieval: time filter first, semantic match second
   - Prevents temporal confusion that plagues existing memory systems

2. **Emotional Anchoring System**
   - valence + arousal + EMOTION_CODES encoding
   - Memories have emotional dimensions, not just semantic content
   - Rhythm organization creates "warm" memory experience

3. **Wakeup Package Injection**
   - Context assembly stage injection (<1200 tokens)
   - Simulates human "memory awakening" on session start
   - Not MCP call, not full RAG retrieval - novel middle ground

4. **Complete Module Replacement Pattern**
   - Proves agent-diva decoupled architecture value
   - Memory module fully replaceable without breaking core
   - Blueprint for future module upgrades

### Market Context & Competitive Landscape

| Solution | Approach | Limitation |
|----------|----------|------------|
| ChatGPT Memory | Simple key-value storage | No temporal dimension |
| LangChain Memory | Session-level context | Weak cross-session capability |
| Most AI Agent Frameworks | Stateless or simple vector DB | No time-aware retrieval |
| **Laputa + agent-diva** | **Time flow + semantic + emotional** | **First to combine all three** |

### Validation Approach

- **A/B Testing**: Memory-enabled vs memory-disabled agent-diva
- **Key Metrics**: Task completion time, user satisfaction, memory引用 accuracy
- **Critical Test**: Temporal accuracy (ask "yesterday" must never return "last month")
- **Token Budget Validation**: Runtime assertion <1200 tokens 100% of time

### Risk Mitigation

| Innovation Risk | Fallback Strategy |
|-----------------|-------------------|
| Emotional encoding too complex | MVP skip, keep time flow only |
| Rhythm organization ineffective | Use simple summary instead |
| Token budget insufficient | Adjust strategy, prioritize key decisions |

## Project-Type Specific Requirements

### Rust AI Agent Framework Integration

#### Technical Requirements
- **MemoryProvider Trait**: Interface contract for Laputa integration
- **Feature Flag Isolation**: Enable/disable Laputa without breaking agent-diva
- **Zero-Copy Serialization**: Minimize overhead in wakeup package injection
- **Async/Await Compatibility**: Laputa must integrate with agent-diva's async runtime

#### Functional Requirements
- **Context Assembly Hook**: Injection point in agent loop
- **Session Management**: Cross-session context retention
- **Error Degradation**: Fallback to local cache on Laputa failure
- **Token Budget Enforcement**: Runtime assertion <1200 tokens

#### Quality Requirements
- **Clippy Clean**: Zero warnings in production code
- **Test Coverage**: ≥80% unit test coverage
- **Integration Tests**: 47+ tests covering wakeup injection, diary CRUD, temporal accuracy
- **Performance Benchmarks**: End-to-end <5s, single retrieval <500ms

## Scoping & Phasing Strategy

### Phase 1: MVP Validation (Current Sprint)
**Goal:** Prove core integration works
- [ ] Standalone version: Laputa crate + minimal TUI
- [ ] Integrated version: agent-diva memory module replacement
- [ ] Wakeup package injection (<1200 tokens)
- [ ] Basic diary write/read
- [ ] Cross-session context retention
- [ ] Interface contract (MemoryProvider trait)
- [ ] Mock implementation for testing
- [ ] Feature flag support

**Exit Criteria:**
- Dual version verification passed
- 47+ integration tests passing
- Coverage ≥80%, Clippy 0 warnings
- End-to-end <5s

### Phase 2: Growth Features
**Goal:** Make it competitive
- [ ] Emotional anchoring (valence + arousal + EMOTION_CODES)
- [ ] Rhythm organization (weekly capsules)
- [ ] Heat decay mechanism
- [ ] Semantic retrieval (vector search)
- [ ] Basic memory management commands (view, search, tag)

### Phase 3: Vision Features
**Goal:** Full-featured memory system
- [ ] Complete 5-layer memory stack (L0-L4)
- [ ] L4 archival layer and archaeology tools
- [ ] Soul evolution mechanism
- [ ] Multi-identity switching
- [ ] Multi-channel support (Telegram, Discord, Slack)
- [ ] Monthly/quarterly/yearly compression chains
- [ ] Ebbinghaus reverse review
- [ ] Relationship-driven recall

## Functional Requirements

### FR-1: Wakeup Package Injection
- **Description**: Inject context into agent loop at context assembly stage
- **Priority**: P0 (MVP)
- **Details**:
  - Retrieve recent memories (last 24h) from Laputa
  - Format into wakeup package (<1200 tokens)
  - Inject into system prompt before user message
  - Runtime assertion: token count ≤1200

### FR-2: Diary Write/Read
- **Description**: Basic memory CRUD operations
- **Priority**: P0 (MVP)
- **Details**:
  - Write: Store user messages and agent responses with timestamps
  - Read: Retrieve memories by time range
  - Temporal filtering: Strict time boundary enforcement
  - Consistency: Write→read 100% accurate

### FR-3: Cross-Session Context Retention
- **Description**: Maintain context across agent-diva sessions
- **Priority**: P0 (MVP)
- **Details**:
  - Session ID tracking
  - Context serialization/deserialization
  - Automatic wakeup on session start

### FR-4: Error Degradation
- **Description**: Graceful fallback when Laputa unavailable
- **Priority**: P0 (MVP)
- **Details**:
  - Local cache of last context
  - Clear user notification of degraded state
  - Auto-recovery when Laputa returns

### FR-5: Feature Flag Support
- **Description**: Enable/disable Laputa without breaking agent-diva
- **Priority**: P0 (MVP)
- **Details**:
  - Compile-time feature flag
  - Runtime toggle for testing
  - Clean fallback to original memory system

### FR-6: Emotional Anchoring (Growth)
- **Description**: Encode emotions in memories
- **Priority**: P1 (Post-MVP)
- **Details**:
  - valence (-1.0 to +1.0)
  - arousal (0.0 to 1.0)
  - EMOTION_CODES (joy, anxiety, frustration, etc.)

### FR-7: Rhythm Organization (Growth)
- **Description**: Automatic weekly capsule generation
- **Priority**: P1 (Post-MVP)
- **Details**:
  - Aggregate weekly memories
  - Extract key decisions and patterns
  - Generate summary capsule

### FR-8: Heat Decay Mechanism (Growth)
- **Description**: Simulate forgetting over time
- **Priority**: P1 (Post-MVP)
- **Details**:
  - Heat score: i32, decays over time
  - >80: Lock (important, don't forget)
  - <20: Archive candidate
  - Runtime switch for decay rate

## Assumptions & Dependencies

### Assumptions
1. agent-diva 的解耦架构允许 memory 模块完全替换
2. Laputa crate 可以独立编译和测试
3. TUI 是首发通道，其他通道（Telegram、Discord）后续支持
4. 用户接受本地 SQLite 存储，不需要云端同步
5. LLM provider 不会滥用注入到 prompt 中的记忆数据

### Dependencies
1. **Laputa Phase 1**: 必须完成基础 CRUD 和唤醒包注入接口
2. **agent-diva改造**: 必须预留 context assembly hook
3. **SQLite**: 作为默认存储后端
4. **Rust async runtime**: tokio 或其他兼容的异步运行时

### Out of Scope (Phase 1)
- 多通道支持（Telegram、Discord、Slack）
- 云端同步和备份
- 完整的五层记忆栈（L0-L4）
- soul 演化机制
- 多身份切换
- 档案考古工具

## Success Metrics Dashboard

| Metric | Target | Measurement | Phase |
|--------|--------|-------------|-------|
| Wakeup injection success rate | 100% | Integration tests | Phase 1 |
| Token budget | ≤1200 | Runtime assertion | Phase 1 |
| Data consistency | 100% | Write→read tests | Phase 1 |
| End-to-end latency | <5s | E2E tests | Phase 1 |
| Dual version completion | 2/2 | Delivery checklist | Phase 1 |
| Unit test coverage | ≥80% | CI report | Phase 1 |
| Flaky tests | 0 | CI report | Phase 1 |
| Temporal accuracy | 100% | Time boundary tests | Phase 1 |
| User satisfaction (memory觉醒) | >4.5/5 | User surveys | Phase 2 |
| Emotional encoding accuracy | >90% | Manual review | Phase 2 |
