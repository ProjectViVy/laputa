# MATSUMOTO 整体性改造计划（2026 Q3）

> **For Hermes:** Use subagent-driven-development skill to implement this plan task-by-task.

**Goal:** 把松本（MATSUMOTO）从"v0 单进程 laputa-py"升级为**双进程 + 结构化目标引擎 + 可执行守护**的工程化 agent harness，对齐 ulw-loop 范式 + OmO/LazyCodex 的 harness engineering 范式。

**Architecture:**

```
┌─────────────────────────────────────────────────────────────┐
│  MATSUMOTO Agent (Hermes)                                   │
│  ├── Plugin Layer: laputa memory provider (已实现)          │
│  ├── Skill Layer: skills/loop-engineering (新增)            │
│  └── MCP Client → 调 ulw-loop 三个 tool                    │
└──────────────┬──────────────────────────────────────────────┘
               │ JSON-RPC over stdio / socket
┌──────────────┴──────────────────────────────────────────────┐
│  Laputa Daemon (独立进程)                                   │
│  ├── 8-file Authority 治理 (已完成 core/)                   │
│  ├── Curator 守护 (已完成 curator/)                          │
│  └── ulw-loop MCP Server (新增)                              │
│      ├─ create_goal    / update_goal    / get_goal          │
│      ├─ record_evidence / checkpoint                       │
│      └─ steer          (7 种结构化 kind)                    │
└──────────────┬──────────────────────────────────────────────┘
               │
        ┌──────┴───────┐
        │ MemPalace    │  (可选外部 store，全量 session)
        │ + 缓存       │
        └──────────────┘
```

**Tech Stack:** Python 3.11+, laputa-py 0.1.x, ulw-loop paradigm (TypeScript 移植), pytest, Hermes cronjob, mempalace 3.5+

---

## 当前上下文

### 工作区状态（2026-06-28）

| 路径 | 状态 | 说明 |
|---|---|---|
| `~/Desktop/projects/laputa-py` | **main 分支，uncommitted 改动** | git status 显示大量 D（删除）和 M |
| 已删除目录 | `bridge/`, `mcp/`, `server/`, `provider/queue.py` | 之前 Wave 重构删了未提交 |
| 已修改 | `cli.py`, `curator/curator.py`, `curator/report_generator.py` | 未提交 |
| 实装代码 | `core/` (1331 行), `curator/` (1340 行), `provider/memory_provider.py` (1113 行) | 功能已经远超计划 |
| 上游计划 | `.hermes/plans/2026-06-26_laputa-mcp-refactor.md` | 12 决策已定、8 文件已定、目录结构已定 |

### 已拍板决策（不重议）

来自 `2026-06-26_laputa-mcp-refactor.md` 12 项 + memory 中的 5 项 + 此次研究新增：

| # | 决策 | 来源 | 状态 |
|---|---|---|---|
| 1 | Laputa 独占 mempalace（停止 Hermes mempalace-mcp） | memory | ✅ |
| 2 | 8 文件 Authority 治理（MEMORY/LONGMEM/SOUL/USERPROFILE/USER/IDENTITY/TASK/THOUGHTS） | 6-26 计划 | ✅ |
| 3 | SOUL 只能 Autodream 写，Agent 不可越权 | memory | ✅ |
| 4 | 锁机制用队列（write queue），写瞬间完成 | 6-26 计划 | ✅ |
| 5 | 缓存 `~/.laputa/cache/autodream_staging/`，每天清理 | 6-26 计划 | ✅ |
| 6 | 日报基于 MEMORY.MD，无修改跳过 | 6-26 计划 | ✅ |
| 7 | Curator 守护进程用 Hermes cronjob 触发 | 6-26 计划 | ✅ |
| 8 | MCP 化可接 Codex/Claude | 6-26 计划 | ✅ |
| 9 | Tool execute() 直接改 Result，不做兼容层 | memory | ✅ |
| 10 | MCP 重连指数退避（1s→60s） | memory | ✅ |
| 11 | **新增**：ulw-loop 范式注入（state machine + evidence + steering） | 此次研究 | ⏳ 待确认 |
| 12 | **新增**：MCP server 暴露 ulw-loop 三件套 | 此次研究 | ⏳ 待确认 |

### 三大改造目标

1. **清理半重构** —— 提交当前 main 的 uncommitted 改动 / 补回误删目录
2. **注入 ulw-loop 范式** —— 把"loop engineering"从 prompt 升级为带状态机的工程产品
3. **守护进程闭环** —— Daemon 启动 + Curator 周期任务 + ulw-loop MCP 全部跑通

### 假设（Assumptions）

- 大湿保留 main 分支继续推进（不改策略到 `agent-diva-pro`）
- Python 生态，不引入 TypeScript（ulw-loop 是 TS，需要翻译/移植核心思想而非代码）
- laputa-py 是**核心**，ulw-loop 是**增强**，不重写 laputa-py
- 不引入新依赖（除非必要，比如 `pydantic` 已有则用，没有则补）
- 计划**不破坏**现有 Hermes memory provider 协议

---

## 总体计划（6 Wave / 28 Task）

| Wave | 主题 | Task 数 | 估时 | 风险 |
|---|---|---|---|---|
| **W1** | 工作区清理 + 基线回稳 | 5 | 0.5d | 低 |
| **W2** | ulw-loop 核心移植（state machine + types） | 6 | 1d | 中 |
| **W3** | ulw-loop MCP Server 接入 | 5 | 1d | 中 |
| **W4** | Hermes Skill 层注入 | 4 | 0.5d | 低 |
| **W5** | Daemon 闭环 + Curator 联动 | 5 | 1d | 高 |
| **W6** | 验证 + 文档 + 收尾 | 3 | 0.5d | 低 |

**总计**: 28 Task / ~4.5 个开发日

---

## Wave 1: 工作区清理 + 基线回稳

**目标**: main 分支回到可发布状态，所有 uncommitted 改动要么提交要么丢弃，丢失的 bridge/mcp/server 目录决定去留。

### Task 1.1: 审查 uncommitted 改动

**Objective:** 看清楚 `git status` 报的所有 D/M 改动是否要保留。

**Files:** 无文件变更（只读审查）

**Step 1:** 列出所有改动文件与 diff 大小
```bash
cd ~/Desktop/projects/laputa-py
git status --porcelain
git diff --stat
```

**Step 2:** 分类
- `M src/laputa/cli.py` — 看是新增命令还是 bug 修
- `M src/laputa/curator/curator.py` — 看是否引入新逻辑
- `M src/laputa/curator/report_generator.py` — 同上
- `D src/laputa/bridge/*` — bridge 之前是 palace 桥接，**已被 provider 直连替代**（commit `04fc502`）→ 确认丢弃
- `D src/laputa/mcp/*` — 上一版 MCP server 实装，**功能不完整** → 决定丢弃或恢复（W3 重建）
- `D src/laputa/server/*` — daemon HTTP server stub，**功能未实装** → 丢弃
- `D src/laputa/provider/queue.py` — 写队列，**核心机制** → 必须恢复（见 W2 验证）
- `D .hermes/plans/2026-06-24_mentle-integration-remaining.md` — Mentle 计划 → 与本计划并行，保留删除

**Step 3:** 输出决策到 `/tmp/laputa-w1-decisions.md`，不直接 commit

**Expected:** 决策文档列出 6+ 文件的去留/恢复策略

**Verify:** 文档存在且每条有 rationale

---

### Task 1.2: 恢复 provider/queue.py（写队列核心）

**Objective:** 找回被误删的写队列实现——这是 6-26 决策 #4 的核心。

**Files:**
- Create: `src/laputa/provider/queue.py`（从 git reflog 找回或重写）
- Test: `tests/test_queue.py`（已存在，确认仍可跑）

**Step 1:** 查 git reflog
```bash
git reflog | grep -i queue
git log --all --oneline -- src/laputa/provider/queue.py
```

**Step 2:** 若能找到原版：
```bash
git show <commit>:src/laputa/provider/queue.py > src/laputa/provider/queue.py
```

若不能，从测试 `test_queue.py` 反推接口（用 read_file 读 30+ 行测试看 import 和用法）

**Step 3:** 跑测试
```bash
cd ~/Desktop/projects/laputa-py
python -m pytest tests/test_queue.py -v
```

**Expected:** 全部 pass（或识别出 1-2 个 import 错误待修）

**Step 4:** 若测试 fail 且无法快速修，**打 stub**：
```python
# src/laputa/provider/queue.py 最小可用版
import asyncio
from typing import Any, Callable, Awaitable

class WriteQueue:
    """FIFO serial write queue. Receives instantly, writes complete fast."""
    def __init__(self, write_fn: Callable[[Any], Awaitable[None]]):
        self._write_fn = write_fn
        self._queue: asyncio.Queue = asyncio.Queue()
        self._task: asyncio.Task | None = None

    async def submit(self, item: Any) -> None:
        await self._queue.put(item)
        if self._task is None:
            self._task = asyncio.create_task(self._drain())

    async def _drain(self) -> None:
        while True:
            item = await self._queue.get()
            try:
                await self._write_fn(item)
            finally:
                self._queue.task_done()
```

**Verify:** `pytest tests/test_queue.py` pass

**Commit:** `git add src/laputa/provider/queue.py && git commit -m "fix: restore write queue stub (W1.2)"`

---

### Task 1.3: 丢弃已删的 mcp/server/bridge 目录

**Objective:** 确认 mcp/server/bridge 是不完整旧实现，正式丢弃。

**Files:**
- Delete: `src/laputa/mcp/` (已空,只含 `__pycache__`)
- Delete: `src/laputa/server/` (已空)
- Delete: `src/laputa/bridge/` (已空)

**Step 1:** 确认目录确实没东西
```bash
find src/laputa/mcp src/laputa/server src/laputa/bridge -type f 2>/dev/null
```
Expected: 只有 `__pycache__/*.pyc`（gitignore 应已忽略）

**Step 2:** 删除空目录
```bash
cd ~/Desktop/projects/laputa-py
git rm -r src/laputa/mcp src/laputa/server src/laputa/bridge 2>/dev/null || rm -rf src/laputa/mcp src/laputa/server src/laputa/bridge
```

**Step 3:** 跑全测试确认没破坏 import
```bash
python -m pytest tests/ -x --ignore=tests/test_palace_bridge.py 2>&1 | tail -20
```

**Verify:** 测试全 pass，没有 import 错误

**Commit:** `git add -A && git commit -m "chore: drop incomplete mcp/server/bridge (W1.3)"`

---

### Task 1.4: 提交 cli/curator/report_generator 的 M 改动

**Objective:** 把已修改但未提交的文件审完提交。

**Files:**
- `src/laputa/cli.py`
- `src/laputa/curator/curator.py`
- `src/laputa/curator/report_generator.py`

**Step 1:** 看每个 diff
```bash
git diff src/laputa/cli.py | head -100
git diff src/laputa/curator/curator.py | head -100
git diff src/laputa/curator/report_generator.py | head -100
```

**Step 2:** 跑相关测试
```bash
python -m pytest tests/test_curator.py tests/test_report_index.py -v
```

**Step 3:** 若测试 pass，原子提交（一个 commit 包含三个相关文件）
```bash
git add src/laputa/cli.py src/laputa/curator/curator.py src/laputa/curator/report_generator.py
git commit -m "feat: curator integration (W1.4)"
```

**Verify:** `git log --oneline -3` 显示新 commit；`git status` clean

---

### Task 1.5: 跑基线测试 + 记录基线

**Objective:** 在干净 main 上跑全测试，记录基线 pass/fail 数。

**Files:**
- Create: `docs/baseline/w1-baseline.md`

**Step 1:** 全测试
```bash
cd ~/Desktop/projects/laputa-py
python -m pytest tests/ --tb=short 2>&1 | tee /tmp/w1-baseline.log
```

**Step 2:** 提取关键数字
```bash
grep -E "passed|failed|error" /tmp/w1-baseline.log | tail -5
```

**Step 3:** 写入基线文档
```markdown
# Wave 1 基线

| 指标 | 值 |
|---|---|
| 提交 | `git rev-parse HEAD` |
| 测试通过 | N |
| 测试失败 | M |
| 测试错误 | K |
| 耗时 | T 秒 |
| 主要失败 | (test_X.py::test_Y 原因) |
```

**Verify:** 文档存在，含完整数据

**Commit:** `git add docs/baseline/w1-baseline.md && git commit -m "docs: w1 baseline" --allow-empty`

---

## Wave 2: ulw-loop 核心移植（state machine + types）

**目标**: 把 ulw-loop 的核心思想翻译成 Python，落到 laputa-py 内部模块，**不依赖外部 TypeScript 代码**。

### Task 2.1: 设计 Goal/Criterion/Steering 数据类型

**Objective:** 翻译 ulw-loop 的 TypeScript 类型到 Python Pydantic/dataclass。

**Files:**
- Create: `src/laputa/ulw/types.py`
- Create: `tests/ulw/test_types.py`

**Step 1:** 写测试
```python
# tests/ulw/test_types.py
from laputa.ulw.types import Goal, Criterion, Status, SteeringKind, ULWLoopError

def test_goal_minimal():
    g = Goal(id="G001", title="t", objective="o", status="pending", success_criteria=[], attempt=0)
    assert g.id == "G001"

def test_criterion_with_evidence():
    c = Criterion(id="C1", scenario="curl /health", expected_evidence="/tmp/x.txt",
                  user_model="happy", status="pass",
                  captured_evidence="HTTP 200")
    assert c.status == "pass"

def test_status_enum():
    for s in ["pending", "in_progress", "complete", "failed", "blocked", "review_blocked", "needs_user_decision"]:
        Status(s)  # 应不抛

def test_steering_kinds():
    expected = {"add_subgoal", "split_subgoal", "reorder_pending",
                "revise_pending_wording", "revise_criterion",
                "annotate_ledger", "mark_blocked_superseded"}
    assert set(SteeringKind) == expected

def test_ulw_error_codes():
    e = ULWLoopError("x", "ULW_LOOP_GOAL_NOT_FOUND", {"goalId": "G1"})
    assert e.code == "ULW_LOOP_GOAL_NOT_FOUND"
```

**Step 2:** 跑测试 fail
```bash
cd ~/Desktop/projects/laputa-py
python -m pytest tests/ulw/test_types.py -v
# Expected: ModuleNotFoundError: No module named 'laputa.ulw'
```

**Step 3:** 写实现
```python
# src/laputa/ulw/types.py
from __future__ import annotations
from dataclasses import dataclass, field
from datetime import datetime
from typing import Literal, Optional, Any
from enum import Enum

class Status(str, Enum):
    PENDING = "pending"
    IN_PROGRESS = "in_progress"
    COMPLETE = "complete"
    FAILED = "failed"
    BLOCKED = "blocked"
    REVIEW_BLOCKED = "review_blocked"
    NEEDS_USER_DECISION = "needs_user_decision"

class CriterionStatus(str, Enum):
    PENDING = "pending"
    PASS = "pass"
    FAIL = "fail"
    BLOCKED = "blocked"

class UserModel(str, Enum):
    HAPPY = "happy"
    EDGE = "edge"
    REGRESSION = "regression"
    ADVERSARIAL = "adversarial"

class SteeringKind(str, Enum):
    ADD_SUBGOAL = "add_subgoal"
    SPLIT_SUBGOAL = "split_subgoal"
    REORDER_PENDING = "reorder_pending"
    REVISE_PENDING_WORDING = "revise_pending_wording"
    REVISE_CRITERION = "revise_criterion"
    ANNOTATE_LEDGER = "annotate_ledger"
    MARK_BLOCKED_SUPERSEDED = "mark_blocked_superseded"

class LedgerEventKind(str, Enum):
    PLAN_CREATED = "plan_created"
    GOAL_STARTED = "goal_started"
    GOAL_RESUMED = "goal_resumed"
    GOAL_COMPLETED = "goal_completed"
    GOAL_BLOCKED = "goal_blocked"
    GOAL_FAILED = "goal_failed"
    GOAL_NEEDS_USER_DECISION = "goal_needs_user_decision"
    GOAL_RETRYED = "goal_retried"
    AGGREGATE_COMPLETED = "aggregate_completed"
    GOAL_ADDED = "goal_added"
    STEERING_ACCEPTED = "steering_accepted"
    STEERING_REJECTED = "steering_rejected"
    FINAL_REVIEW_FAILED = "final_review_failed"
    GOAL_REVIEW_BLOCKED = "goal_review_blocked"
    EVIDENCE_CAPTURED = "evidence_captured"
    CRITERION_FAILED = "criterion_failed"
    CRITERION_BLOCKED = "criterion_blocked"
    CRITERIA_REVISED = "criteria_revised"

@dataclass
class Criterion:
    id: str
    scenario: str
    expected_evidence: str
    user_model: UserModel = UserModel.HAPPY
    status: CriterionStatus = CriterionStatus.PENDING
    captured_evidence: Optional[str] = None
    captured_at: Optional[str] = None
    notes: Optional[str] = None

@dataclass
class Goal:
    id: str
    title: str
    objective: str
    status: Status = Status.PENDING
    success_criteria: list[Criterion] = field(default_factory=list)
    attempt: int = 0
    created_at: Optional[str] = None
    updated_at: Optional[str] = None
    evidence: Optional[str] = None
    failed_at: Optional[str] = None
    failure_reason: Optional[str] = None
    blocked_reason: Optional[str] = None
    blocker_signature: Optional[str] = None
    blocker_occurrence_count: Optional[int] = None
    required_external_decision: Optional[str] = None
    non_retriable: bool = False
    steering_status: Optional[str] = None
    superseded_by: list[str] = field(default_factory=list)
    supersedes: list[str] = field(default_factory=list)

@dataclass
class LedgerEntry:
    at: str
    kind: LedgerEventKind
    goal_id: Optional[str] = None
    criterion_id: Optional[str] = None
    status: Optional[str] = None
    evidence: Optional[str] = None
    captured_evidence: Optional[str] = None
    before: Optional[dict] = None
    after: Optional[dict] = None
    message: Optional[str] = None
    steering: Optional[dict] = None
    mutation_kind: Optional[str] = None
    idempotency_key: Optional[str] = None

class ULWLoopError(Exception):
    def __init__(self, message: str, code: str, details: dict[str, Any] | None = None):
        super().__init__(message)
        self.code = code
        self.details = details or {}
```

**Step 4:** 跑测试
```bash
python -m pytest tests/ulw/test_types.py -v
# Expected: 5 passed
```

**Verify:** 5 passed

**Commit:** `git add src/laputa/ulw/types.py tests/ulw/test_types.py && git commit -m "feat(ulw): add core types (W2.1)"`

---

### Task 2.2: Plan 持久化层（读写 .laputa/ulw-loop/）

**Objective:** Plan 文件 + ledger.jsonl 的原子读写，模仿 ulw-loop 的 `plan-io.ts`。

**Files:**
- Create: `src/laputa/ulw/storage.py`
- Create: `tests/ulw/test_storage.py`

**Step 1:** 写测试
```python
# tests/ulw/test_storage.py
import json
from pathlib import Path
from laputa.ulw.types import Goal, Criterion, LedgerEntry, LedgerEventKind
from laputa.ulw.storage import PlanStorage

def test_ensure_dir(tmp_path: Path):
    s = PlanStorage(tmp_path)
    s.ensure()
    assert (tmp_path / ".laputa" / "ulw-loop" / "goals.json").parent.exists()

def test_write_read_plan(tmp_path: Path):
    s = PlanStorage(tmp_path)
    s.ensure()
    plan = {"goals": [{"id": "G001", "title": "t", "objective": "o", "status": "pending", "success_criteria": [], "attempt": 0}]}
    s.write_plan(plan)
    loaded = s.read_plan()
    assert loaded["goals"][0]["id"] == "G001"

def test_append_ledger(tmp_path: Path):
    s = PlanStorage(tmp_path)
    s.ensure()
    entry = LedgerEntry(at="2026-06-28T00:00:00Z", kind=LedgerEventKind.PLAN_CREATED, evidence="created")
    s.append_ledger(entry)
    s.append_ledger(entry)  # 第二条
    lines = (tmp_path / ".laputa" / "ulw-loop" / "ledger.jsonl").read_text().strip().split("\n")
    assert len(lines) == 2
    assert json.loads(lines[0])["kind"] == "plan_created"

def test_mutation_lock_serializes(tmp_path: Path):
    import threading
    s = PlanStorage(tmp_path)
    s.ensure()
    counter = {"v": 0}
    def write():
        s.with_lock(lambda: counter.__setitem__("v", counter["v"] + 1))
    threads = [threading.Thread(target=write) for _ in range(5)]
    for t in threads: t.start()
    for t in threads: t.join()
    assert counter["v"] == 5  # 没死锁，全部执行
```

**Step 2:** 写实现
```python
# src/laputa/ulw/storage.py
import json
import os
import threading
from datetime import datetime, timezone
from pathlib import Path
from filelock import FileLock  # 新增依赖
from .types import LedgerEntry

ULW_DIR = Path(".laputa") / "ulw-loop"
GOALS_FILE = ULW_DIR / "goals.json"
LEDGER_FILE = ULW_DIR / "ledger.jsonl"
LOCK_FILE = ULW_DIR / ".lock"

class PlanStorage:
    def __init__(self, repo_root: Path | str):
        self.repo_root = Path(repo_root).resolve()
        self.dir = self.repo_root / ULW_DIR
        self.goals_path = self.dir / "goals.json"
        self.ledger_path = self.dir / "ledger.jsonl"
        self.lock_path = self.dir / ".lock"

    def ensure(self) -> None:
        self.dir.mkdir(parents=True, exist_ok=True)
        if not self.goals_path.exists():
            self.goals_path.write_text(json.dumps({"goals": []}, indent=2))

    def read_plan(self) -> dict:
        return json.loads(self.goals_path.read_text())

    def write_plan(self, plan: dict) -> None:
        self.goals_path.write_text(json.dumps(plan, indent=2, default=str))

    def append_ledger(self, entry: LedgerEntry) -> None:
        with self.ledger_path.open("a") as f:
            f.write(json.dumps(entry.__dict__, default=str) + "\n")

    def with_lock(self, fn):
        """Cross-process + thread-safe mutation."""
        self.ensure()
        lock = FileLock(str(self.lock_path))
        with lock:
            return fn()

def iso() -> str:
    return datetime.now(timezone.utc).isoformat()
```

**Step 3:** 跑测试
```bash
pip install filelock  # 若未装
python -m pytest tests/ulw/test_storage.py -v
```

**Verify:** 4 passed

**Commit:** `git add src/laputa/ulw/storage.py tests/ulw/test_storage.py && git commit -m "feat(ulw): plan storage with file lock (W2.2)"`

---

### Task 2.3: Evidence 采集（record-evidence 翻译）

**Objective:** 实现 `record_evidence()` 核心逻辑。

**Files:**
- Create: `src/laputa/ulw/evidence.py`
- Create: `tests/ulw/test_evidence.py`

**Step 1:** 测试
```python
# tests/ulw/test_evidence.py
import pytest
from pathlib import Path
from laputa.ulw.types import Goal, Criterion, CriterionStatus, ULWLoopError
from laputa.ulw.evidence import record_evidence, criteria_summary

def _seed(tmp_path: Path):
    from laputa.ulw.storage import PlanStorage
    s = PlanStorage(tmp_path)
    s.ensure()
    plan = {"goals": [{
        "id": "G001", "title": "t", "objective": "o", "status": "pending",
        "success_criteria": [{"id": "C1", "scenario": "x", "expected_evidence": "y", "user_model": "happy", "status": "pending"}],
        "attempt": 0
    }]}
    s.write_plan(plan)
    return s

def test_record_pass(tmp_path: Path):
    s = _seed(tmp_path)
    result = record_evidence(tmp_path, goal_id="G001", criterion_id="C1",
                              status="pass", evidence="tmux: HTTP 200")
    assert result["criterion"]["status"] == "pass"
    assert result["criterion"]["captured_evidence"] == "tmux: HTTP 200"
    # ledger 写入
    lines = s.ledger_path.read_text().strip().split("\n")
    assert "evidence_captured" in lines[0]

def test_record_empty_evidence_rejected(tmp_path: Path):
    _seed(tmp_path)
    with pytest.raises(ULWLoopError) as e:
        record_evidence(tmp_path, "G001", "C1", "pass", "   ")
    assert e.value.code == "ULW_LOOP_EVIDENCE_REQUIRED"

def test_unknown_goal_rejected(tmp_path: Path):
    _seed(tmp_path)
    with pytest.raises(ULWLoopError) as e:
        record_evidence(tmp_path, "G999", "C1", "pass", "x")
    assert e.value.code == "ULW_LOOP_GOAL_NOT_FOUND"

def test_criteria_summary(tmp_path: Path):
    _seed(tmp_path)
    record_evidence(tmp_path, "G001", "C1", "pass", "x")
    plan = {"goals": [{
        "id": "G001", "title": "t", "objective": "o", "status": "pending",
        "success_criteria": [
            {"id": "C1", "scenario": "x", "expected_evidence": "y", "user_model": "happy", "status": "pass", "captured_evidence": "x"},
            {"id": "C2", "scenario": "x2", "expected_evidence": "y2", "user_model": "happy", "status": "pending"}
        ],
        "attempt": 0
    }]}
    from laputa.ulw.storage import PlanStorage
    PlanStorage(tmp_path).write_plan(plan)
    summary = criteria_summary(tmp_path)
    assert summary["passCount"] == 1
    assert summary["pendingCount"] == 1
    assert "G001" in summary["goalsWithUnresolvedCriteria"]
```

**Step 2:** 实现
```python
# src/laputa/ulw/evidence.py
from pathlib import Path
from .storage import PlanStorage, iso
from .types import LedgerEntry, LedgerEventKind, ULWLoopError

def _find_goal(plan: dict, goal_id: str) -> dict:
    for g in plan["goals"]:
        if g["id"] == goal_id:
            return g
    raise ULWLoopError(f"Goal not found: {goal_id}", "ULW_LOOP_GOAL_NOT_FOUND", {"goalId": goal_id})

def _find_criterion(goal: dict, criterion_id: str) -> dict:
    for c in goal["success_criteria"]:
        if c["id"] == criterion_id:
            return c
    raise ULWLoopError(f"Criterion not found: {criterion_id}", "ULW_LOOP_CRITERION_NOT_FOUND", {"goalId": goal["id"], "criterionId": criterion_id})

def _ledger_kind(status: str) -> LedgerEventKind:
    return {
        "pass": LedgerEventKind.EVIDENCE_CAPTURED,
        "fail": LedgerEventKind.CRITERION_FAILED,
        "blocked": LedgerEventKind.CRITERION_BLOCKED,
    }[status]

def record_evidence(repo_root: Path, goal_id: str, criterion_id: str, status: str, evidence: str, notes: str | None = None) -> dict:
    """Atomically record evidence. status in {pass, fail, blocked}."""
    evidence = evidence.strip()
    if not evidence:
        raise ULWLoopError("Evidence must be non-empty", "ULW_LOOP_EVIDENCE_REQUIRED")

    storage = PlanStorage(repo_root)
    def _do():
        plan = storage.read_plan()
        goal = _find_goal(plan, goal_id)
        criterion = _find_criterion(goal, criterion_id)
        prev = criterion["status"]
        now = iso()
        criterion["status"] = status
        criterion["capturedEvidence"] = evidence
        criterion["capturedAt"] = now
        if notes:
            criterion["notes"] = notes
        goal["updatedAt"] = now
        plan["updatedAt"] = now
        storage.write_plan(plan)
        storage.append_ledger(LedgerEntry(
            at=now, kind=_ledger_kind(status),
            goal_id=goal_id, criterion_id=criterion_id,
            status=status, evidence=evidence, captured_evidence=evidence,
            before={"status": prev},
            after={"status": status, "evidence": evidence, "capturedAt": now, "prevStatus": prev},
        ))
        return {"plan": plan, "goal": goal, "criterion": criterion}

    return storage.with_lock(_do)

def criteria_summary(repo_root: Path) -> dict:
    plan = PlanStorage(repo_root).read_plan()
    total = passed = pending = failed = blocked = 0
    unresolved = []
    for g in plan["goals"]:
        g_unresolved = False
        for c in g["success_criteria"]:
            total += 1
            s = c["status"]
            if s != "pass":
                g_unresolved = True
            if s == "pass": passed += 1
            elif s == "pending": pending += 1
            elif s == "fail": failed += 1
            elif s == "blocked": blocked += 1
        if g_unresolved:
            unresolved.append(g["id"])
    return {
        "totalCriteria": total, "passCount": passed, "pendingCount": pending,
        "failCount": failed, "blockedCount": blocked,
        "goalsWithUnresolvedCriteria": unresolved,
    }
```

**Step 3:** 跑测试
```bash
cd ~/Desktop/projects/laputa-py
python -m pytest tests/ulw/test_evidence.py -v
```

**Verify:** 4 passed

**Commit:** `git add src/laputa/ulw/evidence.py tests/ulw/test_evidence.py && git commit -m "feat(ulw): evidence recording (W2.3)"`

---

### Task 2.4: Steering 验证器（防伪核心）

**Objective:** 翻译 ulw-loop 的 `steering.ts` 验证逻辑，**`weakens()` 必须有**。

**Files:**
- Create: `src/laputa/ulw/steering.py`
- Create: `tests/ulw/test_steering.py`

**Step 1:** 测试
```python
# tests/ulw/test_steering.py
import pytest
from laputa.ulw.steering import validate_steering, SteeringAudit

PROTECTED = ["status", "aggregateCompletion", "qualityGate", "completedAt"]
WEAKEN_WORDS = ["skip", "bypass", "weaken", "auto-complete", "mark complete"]

def _base_proposal(kind="add_subgoal", **overrides):
    p = {"kind": kind, "source": "cli", "evidence": "we found X", "rationale": "we need to do Y", **overrides}
    return p

def test_add_subgoal_valid():
    p = _base_proposal(title="Sub", objective="Do X")
    audit = validate_steering({}, p)
    assert audit["accepted"] is True

def test_missing_evidence_rejected():
    p = _base_proposal(title="Sub", objective="Do X")
    p["evidence"] = ""
    audit = validate_steering({}, p)
    assert audit["accepted"] is False
    assert "missing evidence" in audit["rejectedReasons"]

def test_missing_rationale_rejected():
    p = _base_proposal(title="Sub", objective="Do X")
    p["rationale"] = ""
    audit = validate_steering({}, p)
    assert audit["accepted"] is False

def test_invalid_kind_rejected():
    p = _base_proposal(kind="hack_system", title="t", objective="o")
    audit = validate_steering({}, p)
    assert "invalid kind" in str(audit["rejectedReasons"])

def test_protected_fields_rejected():
    p = _base_proposal(title="t", objective="o", status="complete")  # 写 protected
    audit = validate_steering({}, p)
    assert "protected payload" in str(audit["rejectedReasons"])

def test_weaken_attempt_rejected():
    p = _base_proposal(title="Skip tests", objective="auto-complete", rationale="we need to bypass quality gate")
    audit = validate_steering({}, p)
    assert "weakened completion" in str(audit["rejectedReasons"])

def test_done_plan_rejected():
    p = _base_proposal(title="t", objective="o")
    plan = {"aggregateCompletion": {"status": "complete"}}
    audit = validate_steering(plan, p)
    assert "plan already complete" in str(audit["rejectedReasons"])

@pytest.mark.parametrize("phrase", WEAKEN_WORDS)
def test_weaken_regex_catches_all(phrase):
    p = {"kind": "add_subgoal", "source": "cli",
         "evidence": "x", "rationale": f"need to {phrase} the quality gate and tests",
         "title": "t", "objective": "o"}
    audit = validate_steering({}, p)
    assert audit["accepted"] is False
```

**Step 2:** 实现
```python
# src/laputa/ulw/steering.py
import re
from typing import Any

STEERING_KINDS = {
    "add_subgoal", "split_subgoal", "reorder_pending",
    "revise_pending_wording", "revise_criterion",
    "annotate_ledger", "mark_blocked_superseded",
}
SOURCES = {"user_prompt_submit", "finding", "cli"}
PROTECTED = {"aggregateCompletion", "codexObjective", "codexObjectiveAliases",
             "originalConstraints", "qualityGate", "status", "completedAt", "completionStatus"}

WEAKEN_RE = re.compile(
    r"\b(skip|bypass|weaken|remove|omit|auto[-\s]?complete|mark complete|complete faster)\b",
    re.IGNORECASE,
)
WEAKEN_CONTEXT_RE = re.compile(
    r"\b(test|tests|verification|review|quality gate|complete|completion)\b",
    re.IGNORECASE,
)


def _text(value: Any) -> str:
    if isinstance(value, str):
        return value
    if isinstance(value, dict):
        return " ".join(_text(v) for v in value.values())
    if isinstance(value, list):
        return " ".join(_text(v) for v in value)
    return ""


def weakens(value: Any) -> bool:
    text = _text(value).lower()
    return bool(WEAKEN_RE.search(text) and WEAKEN_CONTEXT_RE.search(text))


def _has_protected(value: Any) -> bool:
    if not isinstance(value, (dict, list)):
        return False
    if isinstance(value, dict):
        for k, v in value.items():
            if k in PROTECTED or "complete" in k.lower():
                return True
            if _has_protected(v):
                return True
    if isinstance(value, list):
        return any(_has_protected(v) for v in value)
    return False


def validate_steering(plan: dict, proposal: dict) -> dict:
    """Return audit dict. accepted=False with rejectedReasons on failure."""
    reasons: list[str] = []

    if not isinstance(proposal, dict):
        reasons.append("proposal must be an object")
        return _audit(proposal, reasons)

    kind = proposal.get("kind")
    if kind not in STEERING_KINDS:
        reasons.append(f"invalid kind: {kind}")

    source = proposal.get("source")
    if source not in SOURCES:
        reasons.append(f"invalid source: {source}")

    evidence = proposal.get("evidence", "")
    if not (isinstance(evidence, str) and evidence.strip()):
        reasons.append("missing evidence")

    rationale = proposal.get("rationale", "")
    if not (isinstance(rationale, str) and rationale.strip()):
        reasons.append("missing rationale")

    if _has_protected(proposal):
        reasons.append("protected payload")

    if weakens(proposal):
        reasons.append("weakened completion")

    if isinstance(plan, dict) and plan.get("aggregateCompletion", {}).get("status") == "complete":
        reasons.append("plan already complete")

    return _audit(proposal, reasons)


def _audit(proposal: Any, reasons: list[str]) -> dict:
    return {
        "accepted": len(reasons) == 0,
        "rejectedReasons": reasons,
        "kind": proposal.get("kind") if isinstance(proposal, dict) else None,
        "source": proposal.get("source") if isinstance(proposal, dict) else None,
    }
```

**Step 3:** 跑测试
```bash
python -m pytest tests/ulw/test_steering.py -v
```

**Verify:** 全部 passed（含 parametrize 5 个 weaken 用例）

**Commit:** `git add src/laputa/ulw/steering.py tests/ulw/test_steering.py && git commit -m "feat(ulw): steering validator with weaken defense (W2.4)"`

---

### Task 2.5: apply_steering + idempotency

**Objective:** 把通过验证的 steering 应用到 plan，去重。

**Files:**
- Create: `src/laputa/ulw/apply.py`
- Create: `tests/ulw/test_apply.py`

**Step 1:** 测试
```python
# tests/ulw/test_apply.py
import pytest
from pathlib import Path
from laputa.ulw.storage import PlanStorage
from laputa.ulw.apply import apply_steering

def _seed(tmp_path: Path):
    s = PlanStorage(tmp_path); s.ensure()
    s.write_plan({"goals": [{"id": "G001", "title": "old", "objective": "old obj",
                              "status": "pending", "success_criteria": [], "attempt": 0}]})
    return s

def test_add_subgoal_appends(tmp_path: Path):
    _seed(tmp_path)
    p = {"kind": "add_subgoal", "source": "cli",
         "evidence": "blocker found", "rationale": "split needed",
         "title": "Subgoal 1", "objective": "Do X"}
    result = apply_steering(tmp_path, p)
    assert result["accepted"] is True
    plan = PlanStorage(tmp_path).read_plan()
    assert len(plan["goals"]) == 2
    assert plan["goals"][1]["id"] == "G002"

def test_idempotency_key_dedup(tmp_path: Path):
    _seed(tmp_path)
    p = {"kind": "add_subgoal", "source": "cli", "idempotencyKey": "abc",
         "evidence": "x", "rationale": "y", "title": "Sub", "objective": "Do"}
    apply_steering(tmp_path, p)
    result2 = apply_steering(tmp_path, p)
    assert result2["deduped"] is True
    plan = PlanStorage(tmp_path).read_plan()
    assert len(plan["goals"]) == 2  # 没重复加

def test_rejected_proposal_not_applied(tmp_path: Path):
    _seed(tmp_path)
    p = {"kind": "add_subgoal", "source": "cli",
         "evidence": "x", "rationale": "y",  # 缺 title/objective
         "title": "", "objective": ""}
    result = apply_steering(tmp_path, p)
    assert result["accepted"] is False
    plan = PlanStorage(tmp_path).read_plan()
    assert len(plan["goals"]) == 1  # 没动

def test_rejected_appends_ledger(tmp_path: Path):
    _seed(tmp_path)
    p = {"kind": "hack", "source": "cli", "evidence": "x", "rationale": "y"}
    apply_steering(tmp_path, p)
    lines = (tmp_path / ".laputa" / "ulw-loop" / "ledger.jsonl").read_text().strip().split("\n")
    assert "steering_rejected" in lines[0]
```

**Step 2:** 实现
```python
# src/laputa/ulw/apply.py
import json
from pathlib import Path
from .storage import PlanStorage, iso
from .steering import validate_steering
from .types import LedgerEntry, LedgerEventKind


def _next_id(plan: dict) -> str:
    max_n = 0
    for g in plan["goals"]:
        if g["id"].startswith("G"):
            try:
                n = int(g["id"][1:])
                max_n = max(max_n, n)
            except ValueError:
                pass
    return f"G{max_n + 1:03d}"


def _find_existing_with_idempotency(storage: PlanStorage, key: str) -> bool:
    if not storage.ledger_path.exists():
        return False
    for line in storage.ledger_path.read_text().strip().split("\n"):
        if not line:
            continue
        e = json.loads(line)
        if e.get("idempotencyKey") == key and e.get("kind") == "steering_accepted":
            return True
    return False


def apply_steering(repo_root: Path, proposal: dict) -> dict:
    storage = PlanStorage(repo_root)
    storage.ensure()
    key = proposal.get("idempotencyKey") or proposal.get("promptSignature")

    def _do():
        plan = storage.read_plan()
        if key and _find_existing_with_idempotency(storage, key):
            return {"accepted": True, "deduped": True, "plan": plan}

        audit = validate_steering(plan, proposal)
        now = iso()
        if audit["accepted"]:
            next_plan = _apply(plan, proposal, now)
            storage.write_plan(next_plan)
            entry = LedgerEntry(
                at=now, kind=LedgerEventKind.STEERING_ACCEPTED,
                evidence=proposal.get("evidence", ""),
                message=proposal.get("rationale", ""),
                mutation_kind=proposal.get("kind"),
                idempotency_key=key,
                before=plan, after=next_plan,
            )
            storage.append_ledger(entry)
            return {"accepted": True, "deduped": False, "plan": next_plan, "audit": audit}
        else:
            entry = LedgerEntry(
                at=now, kind=LedgerEventKind.STEERING_REJECTED,
                evidence=proposal.get("evidence", ""),
                message=proposal.get("rationale", ""),
                mutation_kind=proposal.get("kind"),
                idempotency_key=key,
            )
            storage.append_ledger(entry)
            return {"accepted": False, "rejectedReasons": audit["rejectedReasons"], "plan": plan}

    return storage.with_lock(_do)


def _apply(plan: dict, proposal: dict, now: str) -> dict:
    import copy
    next_plan = copy.deepcopy(plan)
    kind = proposal["kind"]
    if kind == "add_subgoal":
        next_plan["goals"].append({
            "id": _next_id(next_plan),
            "title": proposal.get("title", ""),
            "objective": proposal.get("objective", ""),
            "status": "pending",
            "success_criteria": [],
            "attempt": 0,
            "createdAt": now,
            "updatedAt": now,
            "evidence": proposal.get("evidence", ""),
        })
    elif kind == "annotate_ledger":
        # 不动 plan，只在调用层追加 ledger
        pass
    elif kind == "reorder_pending":
        order = proposal.get("pendingOrder") or proposal.get("after", {}).get("pendingGoalIds", [])
        ordered = [g for gid in order for g in next_plan["goals"] if g["id"] == gid]
        rest = [g for g in next_plan["goals"] if g["id"] not in order]
        next_plan["goals"] = ordered + rest
    # 其他 kind 后续 Wave 处理
    next_plan["updatedAt"] = now
    return next_plan
```

**Step 3:** 跑测试
```bash
cd ~/Desktop/projects/laputa-py
python -m pytest tests/ulw/test_apply.py -v
```

**Verify:** 4 passed

**Commit:** `git add src/laputa/ulw/apply.py tests/ulw/test_apply.py && git commit -m "feat(ulw): apply steering with idempotency (W2.5)"`

---

### Task 2.6: Checkpoint 状态机（核心 gate）

**Objective:** 实现 `checkpoint()` —— 整个 ulw-loop 最关键的状态机。

**Files:**
- Create: `src/laputa/ulw/checkpoint.py`
- Create: `tests/ulw/test_checkpoint.py`

**Step 1:** 测试
```python
# tests/ulw/test_checkpoint.py
import pytest
from pathlib import Path
from laputa.ulw.types import ULWLoopError
from laputa.ulw.storage import PlanStorage
from laputa.ulw.evidence import record_evidence
from laputa.ulw.checkpoint import checkpoint

def _seed_pass_criteria(tmp_path: Path, n: int = 2):
    s = PlanStorage(tmp_path); s.ensure()
    s.write_plan({"goals": [{
        "id": "G001", "title": "t", "objective": "o", "status": "in_progress",
        "success_criteria": [
            {"id": f"C{i}", "scenario": "x", "expected_evidence": "y", "user_model": "happy", "status": "pending"}
            for i in range(1, n + 1)
        ],
        "attempt": 0, "essentialCriterionIds": [f"C{i}" for i in range(1, n + 1)],
    }]})
    return s

def test_checkpoint_complete_passes_when_all_pass(tmp_path: Path):
    _seed_pass_criteria(tmp_path)
    for i in range(1, 3):
        record_evidence(tmp_path, "G001", f"C{i}", "pass", f"ev{i}")
    result = checkpoint(tmp_path, goal_id="G001", status="complete", evidence="all good")
    assert result["goal"]["status"] == "complete"

def test_checkpoint_blocked_when_criteria_pending(tmp_path: Path):
    _seed_pass_criteria(tmp_path)
    record_evidence(tmp_path, "G001", "C1", "pass", "ev")
    with pytest.raises(ULWLoopError) as e:
        checkpoint(tmp_path, "G001", "complete", "x")
    assert e.value.code == "ulw_loop_criteria_not_all_pass"

def test_blocked_3x_escalates_to_needs_user(tmp_path: Path):
    _seed_pass_criteria(tmp_path, 0)
    s = PlanStorage(tmp_path)
    s.write_plan({"goals": [{
        "id": "G001", "title": "t", "objective": "o", "status": "in_progress",
        "success_criteria": [], "attempt": 0,
    }]})
    for _ in range(3):
        try:
            checkpoint(tmp_path, "G001", "blocked", "external auth required: X")
        except ULwLoopError:
            pass
    plan = s.read_plan()
    assert plan["goals"][0]["status"] == "needs_user_decision"
    assert plan["goals"][0]["nonRetriable"] is True

def test_unknown_goal_raises(tmp_path: Path):
    PlanStorage(tmp_path).ensure()
    with pytest.raises(ULWLoopError):
        checkpoint(tmp_path, "G999", "complete", "x")
```

**Step 2:** 实现
```python
# src/laputa/ulw/checkpoint.py
import re
from pathlib import Path
from .storage import PlanStorage, iso
from .evidence import criteria_summary
from .types import LedgerEntry, LedgerEventKind, ULWLoopError

BLOCKER_RE = re.compile(r"external authorization: (\S+)", re.IGNORECASE)


def _find_goal(plan: dict, goal_id: str) -> dict:
    for g in plan["goals"]:
        if g["id"] == goal_id:
            return g
    raise ULWLoopError(f"Unknown goal: {goal_id}", "ulw_loop_goal_not_found")


def _classify_blocker(evidence: str) -> str | None:
    m = BLOCKER_RE.search(evidence)
    return m.group(1) if m else None


def _same_blocker_count(plan: dict, signature: str) -> int:
    return sum(
        1 for g in plan["goals"]
        if g.get("blockerSignature") == signature
    )


def _require_all_pass(goal: dict) -> None:
    for c in goal["success_criteria"]:
        if c["status"] != "pass":
            raise ULWLoopError(
                f"Goal {goal['id']} has unresolved criteria",
                "ulw_loop_criteria_not_all_pass",
                {"goalId": goal["id"], "unresolved": [{"id": c["id"], "status": c["status"]} for c in goal["success_criteria"] if c["status"] != "pass"]},
            )


def _require_essential_pass(goal: dict) -> None:
    essential = set(goal.get("essentialCriterionIds") or [c["id"] for c in goal["success_criteria"]])
    unresolved = [c for c in goal["success_criteria"] if c["id"] in essential and c["status"] != "pass"]
    if unresolved:
        raise ULWLoopError(
            f"Goal {goal['id']} has unresolved essential criteria",
            "ulw_loop_criteria_not_all_pass",
            {"goalId": goal["id"], "unresolved": [{"id": c["id"], "status": c["status"]} for c in unresolved]},
        )


def checkpoint(repo_root: Path, goal_id: str, status: str, evidence: str, codex_goal_json: dict | None = None, quality_gate_json: dict | None = None) -> dict:
    """状态机核心: complete / failed / blocked. Raises ULWLoopError on invalid state."""
    evidence = evidence.strip()
    if not evidence:
        raise ULWLoopError("Evidence required", "ulw_loop_evidence_required")
    if status not in ("complete", "failed", "blocked"):
        raise ULWLoopError(f"Invalid status: {status}", "ulw_loop_invalid_status")

    storage = PlanStorage(repo_root)

    def _do():
        plan = storage.read_plan()
        goal = _find_goal(plan, goal_id)
        now = iso()
        ledger_kind = None

        if status == "complete":
            # 默认是 aggregate mode: 用 essential 通过即放行
            _require_essential_pass(goal)
            goal["status"] = "complete"
            goal["completedAt"] = now
            goal["evidence"] = evidence
            goal.pop("failedAt", None)
            goal.pop("failureReason", None)
            goal.pop("blockedReason", None)
            goal.pop("blockerSignature", None)
            goal.pop("blockerOccurrenceCount", None)
            if plan.get("activeGoalId") == goal["id"]:
                plan.pop("activeGoalId")
            ledger_kind = LedgerEventKind.GOAL_COMPLETED
        else:
            # failed / blocked: 检测是否外部授权 blocker
            signature = _classify_blocker(evidence)
            count = (_same_blocker_count(plan, signature) + 1) if signature else 0
            needs_decision = bool(signature) and count >= 3
            goal["status"] = "needs_user_decision" if needs_decision else status
            goal["updatedAt"] = now
            if status == "failed" or needs_decision:
                goal["failedAt"] = now
                goal["failureReason"] = evidence
            if status == "blocked" or needs_decision:
                goal["blockedReason"] = evidence
            if signature:
                goal["blockerSignature"] = signature
                goal["blockerOccurrenceCount"] = count
                goal["requiredExternalDecision"] = f"Resolve external authorization: {signature}"
            if needs_decision:
                goal["nonRetriable"] = True
            if plan.get("activeGoalId") == goal["id"]:
                plan.pop("activeGoalId")
            ledger_kind = {
                "failed": LedgerEventKind.GOAL_FAILED,
                "blocked": LedgerEventKind.GOAL_BLOCKED,
                "needs_user_decision": LedgerEventKind.GOAL_NEEDS_USER_DECISION,
            }[goal["status"]]

        goal["updatedAt"] = now
        plan["updatedAt"] = now
        storage.write_plan(plan)
        storage.append_ledger(LedgerEntry(
            at=now, kind=ledger_kind,
            goal_id=goal_id, status=goal["status"], evidence=evidence,
        ))
        return {"plan": plan, "goal": goal}

    return storage.with_lock(_do)
```

**Step 3:** 跑测试
```bash
cd ~/Desktop/projects/laputa-py
python -m pytest tests/ulw/test_checkpoint.py -v
```

**Verify:** 4 passed

**Commit:** `git add src/laputa/ulw/checkpoint.py tests/ulw/test_checkpoint.py && git commit -m "feat(ulw): checkpoint state machine with auto-escalation (W2.6)"`

---

## Wave 3: ulw-loop MCP Server

**目标**: 把 ulw-loop 暴露为 MCP tools，让 Hermes/Codex/Claude 都能调。

### Task 3.1: 选定 MCP 库

**Objective:** 选一个轻量 MCP 库，不引入 FastAPI 之类的重框架。

**Files:** `requirements.txt` / `pyproject.toml` 修改

**Step 1:** 评估
- `mcp` (官方 Python SDK) — 标准但有 asyncio 复杂度
- `fastmcp` — 装饰器风格，轻量
- **自实现 JSON-RPC over stdio** — 完全可控，~150 行

**决策（自主拍板）**: 用 **`mcp` 官方 SDK**（标准、跨客户端、官方维护）。理由：Codex/Claude 都是 MCP 原生客户，标准协议最稳。

**Step 2:** 加依赖
```toml
# pyproject.toml
dependencies = [
    "mcp>=1.0.0",
    # ... 现有
]
```

```bash
pip install mcp
```

**Verify:** `python -c "import mcp; print(mcp.__version__)"` 输出版本

**Commit:** `git add pyproject.toml && git commit -m "chore: add mcp dependency (W3.1)"`

---

### Task 3.2: ulw-loop 三个 tool schema

**Objective:** 定义 MCP tool schema：`create_goal` / `record_evidence` / `checkpoint` / `steer`。

**Files:**
- Create: `src/laputa/ulw/mcp_tools.py`
- Create: `tests/ulw/test_mcp_tools.py`

**Step 1:** 实现（用官方 mcp SDK 的装饰器风格）
```python
# src/laputa/ulw/mcp_tools.py
from pathlib import Path
from mcp.server.fastmcp import FastMCP
from .evidence import record_evidence as _record_evidence
from .checkpoint import checkpoint as _checkpoint
from .apply import apply_steering

mcp = FastMCP("ulw-loop")


@mcp.tool(description="Create or extend a ulw-loop plan from a brief")
def create_goals(brief: str, brief_file: str | None = None, session_id: str | None = None) -> str:
    """Initialize a plan from a brief. Writes to .laputa/ulw-loop/goals.json."""
    from .storage import PlanStorage
    repo_root = Path.cwd()
    storage = PlanStorage(repo_root); storage.ensure()
    text = brief
    if brief_file:
        text = Path(brief_file).read_text()
    # 简化版：把 brief 作为一个 goal
    plan = storage.read_plan()
    if not plan["goals"]:
        plan["goals"].append({
            "id": "G001", "title": text[:60], "objective": text,
            "status": "pending", "success_criteria": [], "attempt": 0,
        })
        storage.write_plan(plan)
    return f"Plan initialized. {len(plan['goals'])} goal(s). Path: {storage.goals_path}"


@mcp.tool(description="Record observable evidence for a criterion")
def record_evidence(goal_id: str, criterion_id: str, status: str, evidence: str, notes: str = "") -> str:
    """status ∈ {pass, fail, blocked}. Evidence must be non-empty observable artifact."""
    from .storage import PlanStorage
    try:
        result = _record_evidence(Path.cwd(), goal_id, criterion_id, status, evidence, notes or None)
        return f"Recorded: {goal_id}/{criterion_id} = {status}"
    except Exception as e:
        return f"ERROR [{getattr(e, 'code', 'unknown')}]: {e}"


@mcp.tool(description="Checkpoint a goal: complete | failed | blocked")
def checkpoint(goal_id: str, status: str, evidence: str) -> str:
    """State machine gate. Raises on missing criteria."""
    from .storage import PlanStorage
    try:
        result = _checkpoint(Path.cwd(), goal_id, status, evidence)
        return f"Checkpointed: {goal_id} = {result['goal']['status']}"
    except Exception as e:
        return f"ERROR [{getattr(e, 'code', 'unknown')}]: {e}"


@mcp.tool(description="Apply structured steering mutation (7 kinds)")
def steer(kind: str, evidence: str, rationale: str, title: str = "", objective: str = "",
          goal_id: str = "", criterion_id: str = "", idempotency_key: str = "") -> str:
    """7 kinds: add_subgoal, split_subgoal, reorder_pending, revise_pending_wording,
    revise_criterion, annotate_ledger, mark_blocked_superseded.
    Rejects protected fields and weaken attempts automatically."""
    from .storage import PlanStorage
    proposal = {
        "kind": kind, "source": "cli",
        "evidence": evidence, "rationale": rationale,
    }
    if title: proposal["title"] = title
    if objective: proposal["objective"] = objective
    if goal_id: proposal["targetGoalId"] = goal_id
    if criterion_id: proposal["criterionId"] = criterion_id
    if idempotency_key: proposal["idempotencyKey"] = idempotency_key
    result = apply_steering(Path.cwd(), proposal)
    if result["accepted"]:
        if result.get("deduped"):
            return "DEDUPED: same idempotency_key"
        return f"ACCEPTED: {kind} applied"
    return f"REJECTED: {result.get('rejectedReasons', [])}"


@mcp.tool(description="Read plan + criteria summary")
def status() -> str:
    from .storage import PlanStorage
    from .evidence import criteria_summary
    plan = PlanStorage(Path.cwd()).read_plan()
    summary = criteria_summary(Path.cwd())
    return f"Goals: {len(plan['goals'])}\nCriteria: {summary}"
```

**Step 2:** 写 import smoke test
```python
# tests/ulw/test_mcp_tools.py
def test_mcp_tools_importable():
    from laputa.ulw.mcp_tools import mcp, create_goals, record_evidence, checkpoint, steer, status
    assert mcp.name == "ulw-loop"
    # 5 个 tool 全注册
    tools = asyncio.run(mcp.list_tools())
    names = {t.name for t in tools}
    assert {"create_goals", "record_evidence", "checkpoint", "steer", "status"} <= names
```

**Step 3:** 跑测试
```bash
cd ~/Desktop/projects/laputa-py
python -m pytest tests/ulw/test_mcp_tools.py -v
```

**Verify:** 1 passed

**Commit:** `git add src/laputa/ulw/mcp_tools.py tests/ulw/test_mcp_tools.py && git commit -m "feat(ulw): MCP server with 5 tools (W3.2)"`

---

### Task 3.3: MCP Server 启动入口

**Objective:** 提供 `laputa-mcp` CLI 启动 ulw-loop MCP server。

**Files:**
- Create: `src/laputa/cli_ulw_mcp.py` 或合并到 `cli.py`
- Create: `tests/test_ulw_mcp_cli.py`

**Step 1:** 在 `cli.py` 加子命令（参考现有 cli.py 结构）
```python
# src/laputa/cli.py 新增
def ulw_mcp_server():
    """Start the ulw-loop MCP server over stdio."""
    from .ulw.mcp_tools import mcp
    mcp.run()

# 在 main() 加：
# elif cmd == "ulw-mcp": ulw_mcp_server()
```

**Step 2:** 测试 CLI 入口
```python
# tests/test_ulw_mcp_cli.py
from click.testing import CliRunner
from laputa.cli import main

def test_ulw_mcp_in_help():
    runner = CliRunner()
    result = runner.invoke(main, ["--help"])
    assert "ulw-mcp" in result.output
```

**Step 3:** 跑测试 + smoke run
```bash
cd ~/Desktop/projects/laputa-py
python -m laputa.cli ulw-mcp < /dev/null  # 应启动后立即退出（无 stdin）
```

**Verify:** CLI 列出 ulw-mcp；smoke run 不抛 import 错

**Commit:** `git add src/laputa/cli.py tests/test_ulw_mcp_cli.py && git commit -m "feat(cli): ulw-mcp server entry (W3.3)"`

---

### Task 3.4: 与 Laputa Daemon 集成（复用现有 daemon 端口）

**Objective:** 决定 ulw-loop MCP 走独立进程还是塞进现有 Laputa Daemon。

**决策（自主拍板）**: **塞进现有 daemon**。理由：
- laputa-py 已有 daemon 进程（per `git log` "feat: Laputa Daemon auto-start + Hermes integration"）
- 一个端口 / 一组 MCP 客户端连接，省事
- 跨进程 lock 已有，复用

**Files:**
- Modify: `src/laputa/server/daemon.py`（已存在，确认结构）

**Step 1:** 读 daemon 主文件
```bash
cd ~/Desktop/projects/laputa-py
find . -name "daemon*.py" -not -path "*/node_modules/*" 2>/dev/null
```

**Step 2:** 在 daemon 启动时挂载 ulw-loop MCP（不破坏现有 palace_bridge / curator 调度）

具体代码由 daemon 实际结构决定——这是**集成任务**，先看代码再写。

**Verify:** daemon 启动后，外部 MCP client 能连上

**Commit:** `git add src/laputa/server/ && git commit -m "feat(daemon): mount ulw-loop MCP (W3.4)"`

---

### Task 3.5: MCP 重连指数退避

**Objective:** 客户端（Hermes 端）连接 daemon 失败时按 1s→60s 退避。

**Files:**
- Modify: `src/laputa/provider/memory_provider.py`（已有 daemon client 代码）
- Create: `tests/test_reconnect.py`

**Step 1:** 测试
```python
# tests/test_reconnect.py
import time
from laputa.provider.reconnect import backoff

def test_backoff_starts_at_1s():
    assert backoff(0) == 1

def test_backoff_doubles():
    assert backoff(1) == 2
    assert backoff(2) == 4
    assert backoff(3) == 8

def test_backoff_caps_at_60s():
    assert backoff(10) == 60
    assert backoff(20) == 60

def test_backoff_includes_jitter():
    """Jitter should be ±20%."""
    vals = {backoff(5) for _ in range(20)}
    assert len(vals) > 1  # 至少有一次差异
```

**Step 2:** 实现
```python
# src/laputa/provider/reconnect.py
import random

def backoff(attempt: int, base: float = 1.0, cap: float = 60.0) -> float:
    """1s → 2 → 4 → ... → 60, with ±20% jitter."""
    raw = min(cap, base * (2 ** attempt))
    jitter = raw * 0.2 * (random.random() * 2 - 1)
    return max(0.1, raw + jitter)
```

**Step 3:** 在 memory_provider.py 的 daemon 连接处用
```python
# memory_provider.py 修改点
from .reconnect import backoff
for attempt in range(10):
    try:
        self._daemon = connect_daemon()
        break
    except ConnectionError:
        time.sleep(backoff(attempt))
```

**Verify:** `pytest tests/test_reconnect.py` 4 passed

**Commit:** `git add src/laputa/provider/reconnect.py src/laputa/provider/memory_provider.py tests/test_reconnect.py && git commit -m "feat(provider): exponential backoff for daemon reconnect (W3.5)"`

---

## Wave 4: Hermes Skill 层注入

**目标**: 教大湿（松本）怎么用 ulw-loop，把工作流规范压成 skill。

### Task 4.1: 创建 `loop-engineering` skill

**Objective:** 写一个 Hermes skill 教大湿在合适场景使用 ulw-loop 三件套。

**Files:**
- Create: `~/.hermes/skills/loop-engineering/SKILL.md`
- Create: `~/.hermes/skills/loop-engineering/references/full-workflow.md`

**Step 1:** 写 SKILL.md（参考 ulw-loop `SKILL.md` 风格）
```markdown
---
name: loop-engineering
description: Use ulw-loop (durable goal state + evidence + steering) for long-running multi-step tasks. Triggers when the user asks for durable execution, "loop until done", multi-criterion verification, or any task likely to need context-resume.
metadata:
  short-description: Goal-like loop with evidence-bound completion
---

# loop-engineering

Use this skill when the user asks for `ulw-loop`, durable goal execution, evidence-led work, or any multi-step task with verifiable success criteria.

## When to use

- Task has 3+ verifiable success criteria
- Task is likely to span multiple sessions (need resume)
- Task needs evidence (test, curl, screenshot, transcript) not vibes
- Task has natural check-in points (per-criterion, per-goal)

## When NOT to use

- Single quick fix → no skill needed
- Pure research question → use `financial-research` or `web-research-digest`
- One-shot edit → just do it

## Workflow (READ references/full-workflow.md FIRST)

1. `laputa-cli create-goals --brief "..."` to initialize plan
2. Define success criteria upfront (essential vs nice-to-have)
3. Per-criterion cycle: PLAN → DELEGATE → EVIDENCE → CLEANUP → RECORD
4. Use `laputa-cli steer --kind <kind>` for structured changes (NEVER natural language)
5. `laputa-cli checkpoint` to advance goals (auto-validates criteria)
6. Read `references/full-workflow.md` for the full spec (238 lines, non-negotiables)
```

**Step 2:** 复制 ulw-loop `full-workflow.md` 的精炼版（去掉 Codex 专属 tool 名，替换为 `laputa-cli`）

**Step 3:** 写一个 frontmatter trigger test
```python
# tests/test_skill_frontmatter.py
from pathlib import Path

def test_loop_engineering_skill_exists():
    skill = Path.home() / ".hermes" / "skills" / "loop-engineering" / "SKILL.md"
    assert skill.exists()
    content = skill.read_text()
    assert content.startswith("---")
    assert "name: loop-engineering" in content
    assert "description:" in content
```

**Verify:** skill 文件存在，frontmatter 正确

**Commit:** `git add skills/ tests/test_skill_frontmatter.py && git commit -m "feat(skill): add loop-engineering skill (W4.1)"`

---

### Task 4.2: 提供 `laputa-cli` Python wrapper skill

**Objective:** 让 skill 调用 ulw-loop 工具时不需要记命令。

**Files:**
- Create: `~/.hermes/skills/loop-engineering/scripts/laputa-cli.py`
- Create: `~/.hermes/skills/loop-engineering/scripts/README.md`

**Step 1:** 写 wrapper（薄包装 + 友好错误）
```python
#!/usr/bin/env python3
"""laputa-cli: thin wrapper for ulw-loop MCP, callable from any skill."""
import argparse
import json
import sys
from pathlib import Path

# 复用 laputa.ulw 模块
sys.path.insert(0, str(Path(__file__).resolve().parents[4] / "projects" / "laputa-py" / "src"))
from laputa.ulw.evidence import record_evidence
from laputa.ulw.checkpoint import checkpoint
from laputa.ulw.apply import apply_steering

def main():
    p = argparse.ArgumentParser(prog="laputa-cli")
    sub = p.add_subparsers(dest="cmd", required=True)

    rec = sub.add_parser("record-evidence")
    rec.add_argument("--goal-id", required=True)
    rec.add_argument("--criterion-id", required=True)
    rec.add_argument("--status", required=True, choices=["pass", "fail", "blocked"])
    rec.add_argument("--evidence", required=True)
    rec.add_argument("--notes", default="")
    rec.add_argument("--repo-root", default=".")

    cp = sub.add_parser("checkpoint")
    cp.add_argument("--goal-id", required=True)
    cp.add_argument("--status", required=True, choices=["complete", "failed", "blocked"])
    cp.add_argument("--evidence", required=True)
    cp.add_argument("--repo-root", default=".")

    st = sub.add_parser("steer")
    st.add_argument("--kind", required=True)
    st.add_argument("--evidence", required=True)
    st.add_argument("--rationale", required=True)
    st.add_argument("--title", default="")
    st.add_argument("--objective", default="")
    st.add_argument("--goal-id", default="")
    st.add_argument("--idempotency-key", default="")
    st.add_argument("--repo-root", default=".")

    args = p.parse_args()
    try:
        if args.cmd == "record-evidence":
            r = record_evidence(Path(args.repo_root), args.goal_id, args.criterion_id, args.status, args.evidence, args.notes or None)
            print(json.dumps({"ok": True, "criterion": r["criterion"]}, default=str))
        elif args.cmd == "checkpoint":
            r = checkpoint(Path(args.repo_root), args.goal_id, args.status, args.evidence)
            print(json.dumps({"ok": True, "goal_status": r["goal"]["status"]}, default=str))
        elif args.cmd == "steer":
            prop = {"kind": args.kind, "source": "cli", "evidence": args.evidence, "rationale": args.rationale}
            if args.title: prop["title"] = args.title
            if args.objective: prop["objective"] = args.objective
            if args.goal_id: prop["targetGoalId"] = args.goal_id
            if args.idempotency_key: prop["idempotencyKey"] = args.idempotency_key
            r = apply_steering(Path(args.repo_root), prop)
            print(json.dumps({"ok": r["accepted"], "deduped": r.get("deduped", False), "reasons": r.get("rejectedReasons", [])}))
    except Exception as e:
        print(json.dumps({"ok": False, "code": getattr(e, 'code', 'unknown'), "error": str(e)}))
        sys.exit(1)

if __name__ == "__main__":
    main()
```

**Step 2:** 写 README
```markdown
# laputa-cli

Thin wrapper for ulw-loop. Use this from any skill or script.

```bash
laputa-cli record-evidence --goal-id G001 --criterion-id C1 --status pass --evidence "HTTP 200 from /health"
laputa-cli checkpoint --goal-id G001 --status complete --evidence "all criteria pass"
laputa-cli steer --kind add_subgoal --evidence "blocker X" --rationale "split" --title "Sub" --objective "Do Y"
```

Exit 0 on success, 1 on validation error. JSON output.
```

**Step 3:** 加 PATH 进 `~/.hermes/scripts/laputa-cli` 软链
```bash
ln -sf ~/.hermes/skills/loop-engineering/scripts/laputa-cli.py ~/.hermes/scripts/laputa-cli
```

**Verify:** `laputa-cli --help` 列出 3 个子命令

**Commit:** `git add scripts/ && git commit -m "feat(skill): laputa-cli wrapper (W4.2)"`

---

### Task 4.3: 端到端 smoke test

**Objective:** 验证 skill 真的能用：创建 plan → record evidence → checkpoint。

**Files:**
- Create: `tests/e2e/test_loop_engineering_e2e.py`

**Step 1:** 测试
```python
# tests/e2e/test_loop_engineering_e2e.py
"""End-to-end: real CLI calls on a temp dir, simulate the full ulw-loop flow."""
import json
import subprocess
import sys
from pathlib import Path

LAPUTA_CLI = Path.home() / ".hermes" / "scripts" / "laputa-cli"

def _cli(*args, repo_root: Path) -> dict:
    r = subprocess.run([sys.executable, str(LAPUTA_CLI), *args, "--repo-root", str(repo_root)],
                       capture_output=True, text=True)
    return {"stdout": r.stdout, "stderr": r.stderr, "code": r.returncode}

def test_full_flow(tmp_path: Path):
    # 1. 初始化 plan
    from laputa.ulw.storage import PlanStorage
    s = PlanStorage(tmp_path); s.ensure()
    s.write_plan({"goals": [{
        "id": "G001", "title": "smoke", "objective": "verify flow",
        "status": "in_progress", "attempt": 0,
        "success_criteria": [
            {"id": "C1", "scenario": "x", "expected_evidence": "y", "user_model": "happy", "status": "pending"},
            {"id": "C2", "scenario": "x", "expected_evidence": "y", "user_model": "edge", "status": "pending"},
        ],
        "essentialCriterionIds": ["C1", "C2"],
    }]})

    # 2. record evidence
    r1 = _cli("record-evidence", "--goal-id", "G001", "--criterion-id", "C1",
              "--status", "pass", "--evidence", "tmux: HTTP 200", repo_root=tmp_path)
    assert r1["code"] == 0, r1

    r2 = _cli("record-evidence", "--goal-id", "G001", "--criterion-id", "C2",
              "--status", "pass", "--evidence", "tmux: HTTP 200 on edge", repo_root=tmp_path)
    assert r2["code"] == 0, r2

    # 3. checkpoint
    r3 = _cli("checkpoint", "--goal-id", "G001", "--status", "complete",
              "--evidence", "all pass", repo_root=tmp_path)
    assert r3["code"] == 0, r3
    out = json.loads(r3["stdout"])
    assert out["goal_status"] == "complete"

    # 4. ledger 应该有 3 条（2 evidence + 1 goal_completed）
    ledger = (tmp_path / ".laputa" / "ulw-loop" / "ledger.jsonl").read_text().strip().split("\n")
    assert len(ledger) == 3
```

**Step 2:** 跑
```bash
cd ~/Desktop/projects/laputa-py
python -m pytest tests/e2e/test_loop_engineering_e2e.py -v
```

**Verify:** 1 passed（**整个 ulw-loop 端到端工作**）

**Commit:** `git add tests/e2e/ && git commit -m "test(e2e): full ulw-loop flow (W4.3)"`

---

### Task 4.4: 写入 memory：让大湿记得用

**Objective:** 在 memory 工具里加一条规则：当任务有 ≥3 个可验证标准 → 用 loop-engineering skill。

**Files:** `memory` 工具调用

**Step 1:**
```python
# 通过 memory tool add
content = "loop-engineering skill 触发条件：任务有 ≥3 个 success criterion / 跨 session / 需要可观察 evidence。命令：laputa-cli record-evidence / checkpoint / steer。Steering 必须结构化（7 kind），自然语言 steering 一律 reject。"
```

**Verify:** `memory` 工具确认写入

---

## Wave 5: Daemon 闭环 + Curator 联动

**目标**: Laputa Daemon 自动启动 + Curator 周期任务跑通 + ulw-loop 接入 daemon。

### Task 5.1: 审计现有 daemon 代码

**Objective:** 看清楚 daemon 现状（per `git log` 已有 "Laputa Daemon auto-start + Hermes integration"）。

**Files:** 无变更（read-only）

**Step 1:**
```bash
cd ~/Desktop/projects/laputa-py
find . -name "daemon*.py" -not -path "*/__pycache__/*"
git log --oneline --all -- "**/daemon*.py" | head
```

**Step 2:** 读 daemon 入口、配置、生命周期

**Step 3:** 列出 daemon 现状文档
- 启动方式（auto-start via Hermes? cron? systemd?）
- 已有 MCP/handler
- 现有 curator 调度入口

**Verify:** 文档列出 daemon 5+ 关键点

**Commit:** N/A (audit task)

---

### Task 5.2: Daemon 启动时挂载 ulw-loop MCP

**Objective:** Wave 3.4 的实现任务，基于 5.1 审计结果。

**Files:**
- Modify: `src/laputa/server/daemon.py`（假设路径）

**Step 1:** 在 daemon 启动 hook 加：
```python
# daemon.py 修改点
from laputa.ulw.mcp_tools import mcp as ulw_mcp
# 在 daemon 启动时：
# ulw_mcp.run()  # stdio 模式——可能需要 socket 模式
```

**Step 2:** 写一个 daemon health check 端点暴露 ulw-loop 状态
```python
# daemon.py 新增
@app.route("/health/ulw-loop")
def ulw_loop_health():
    from laputa.ulw.storage import PlanStorage
    from laputa.ulw.evidence import criteria_summary
    try:
        s = PlanStorage(Path.cwd()); s.ensure()
        return {"ok": True, "summary": criteria_summary(Path.cwd())}
    except Exception as e:
        return {"ok": False, "error": str(e)}
```

**Step 3:** 跑 daemon
```bash
cd ~/Desktop/projects/laputa-py
python -m laputa.cli daemon-start
curl localhost:<port>/health/ulw-loop
```

**Verify:** health 端点返回 `ok: true`

**Commit:** `git add src/laputa/server/ && git commit -m "feat(daemon): mount ulw-loop mcp (W5.2)"`

---

### Task 5.3: Curator 增加 ulw-loop 巡检任务

**Objective:** Curator 每天检查 ulw-loop plan 状态，报告 lingering goals。

**Files:**
- Modify: `src/laputa/curator/curator.py`

**Step 1:** 在 curator 的 daily tasks 数组加
```python
{
    "name": "ulw-loop_health_check",
    "schedule": "0 9 * * *",  # 每天 9:00
    "fn": "_check_ulw_loop_health",
}

def _check_ulw_loop_health(self) -> str:
    from .ulw.storage import PlanStorage
    from .ulw.evidence import criteria_summary
    from pathlib import Path
    for repo in self.repos:  # 假设有 repo 列表
        s = Path(repo)
        if not (s / ".laputa" / "ulw-loop").exists():
            continue
        summary = criteria_summary(s)
        if summary["goalsWithUnresolvedCriteria"]:
            self._notify(f"Repo {repo.name}: {len(summary['goalsWithUnresolvedCriteria'])} goals unresolved")
    return "ulw-loop health check done"
```

**Step 2:** 测试
```python
# tests/curator/test_ulw_loop_check.py
def test_curator_ulw_loop_check(tmp_path):
    from laputa.ulw.storage import PlanStorage
    PlanStorage(tmp_path).ensure()
    PlanStorage(tmp_path).write_plan({"goals": [{
        "id": "G001", "title": "t", "objective": "o", "status": "pending",
        "success_criteria": [{"id": "C1", "scenario": "x", "expected_evidence": "y", "user_model": "happy", "status": "pending"}],
        "attempt": 0,
    }]})
    from laputa.curator.curator import Curator
    c = Curator(repos=[tmp_path])
    result = c._check_ulw_loop_health()
    assert "unresolved" in result
```

**Verify:** test pass

**Commit:** `git add src/laputa/curator/ tests/curator/ && git commit -m "feat(curator): ulw-loop health check (W5.3)"`

---

### Task 5.4: Hermes cronjob 注册

**Objective:** 配 Hermes cronjob 周期触发 daemon + curator。

**Files:** Hermes 配置文件

**Step 1:** 写入 cronjob
```bash
hermes cronjob create \
  --name "laputa-curator" \
  --schedule "0 9 * * *" \
  --prompt "运行 laputa curator 巡检：检查 ulw-loop 计划状态，报告 lingering goals" \
  --deliver telegram
```

**Step 2:** 验证 cronjob 列出
```bash
hermes cronjob list
```

**Verify:** cronjob 出现

**Commit:** `git add ~/.hermes/cron/ && git commit -m "feat(cron): register laputa-curator (W5.4)"`

---

### Task 5.5: 端到端 daemon 流程

**Objective:** 启动 daemon → 触发 curator → ulw-loop 健康检查 → 通知。

**Files:**
- Create: `scripts/dev/e2e_daemon.sh`

**Step 1:** 写脚本
```bash
#!/bin/bash
set -e
cd ~/Desktop/projects/laputa-py

echo "1. Start daemon..."
python -m laputa.cli daemon-start &
DAEMON_PID=$!
sleep 3

echo "2. Health check ulw-loop..."
curl -s http://localhost:8765/health/ulw-loop | jq

echo "3. Trigger curator ulw-loop check..."
python -c "from laputa.curator.curator import Curator; print(Curator(repos=['.'])._check_ulw_loop_health())"

echo "4. Cleanup..."
kill $DAEMON_PID
echo "DONE"
```

**Step 2:** 跑
```bash
chmod +x scripts/dev/e2e_daemon.sh
./scripts/dev/e2e_daemon.sh
```

**Verify:** 4 步全成功

**Commit:** `git add scripts/dev/ && git commit -m "test: daemon e2e (W5.5)"`

---

## Wave 6: 验证 + 文档 + 收尾

**目标**: 全套测试 + 文档齐 + 标签发布。

### Task 6.1: 全测试 + 覆盖率

**Objective:** 跑全测试，生成 coverage 报告。

**Files:**
- Create: `docs/coverage/w6-coverage.md`

**Step 1:**
```bash
cd ~/Desktop/projects/laputa-py
python -m pytest tests/ --cov=src/laputa --cov-report=term-missing > /tmp/w6-cov.txt 2>&1
tail -50 /tmp/w6-cov.txt
```

**Step 2:** 写入基线
- 总测试数
- 通过 / 失败 / 错误
- 覆盖率（ulw/ 模块应 ≥ 80%）

**Verify:** 报告存在，覆盖率达标

**Commit:** `git add docs/coverage/ && git commit -m "docs: w6 coverage baseline"`

---

### Task 6.2: 文档：ARCHITECTURE.md / CHANGELOG

**Objective:** 更新顶层文档，反映 ulw-loop 接入。

**Files:**
- Modify: `README.md`
- Modify: `CHANGELOG.md`
- Create: `docs/architecture/ulw-loop-integration.md`

**Step 1:** 更新 CHANGELOG
```markdown
## [Unreleased]

### Added (W2-W5)
- ulw-loop paradigm: durable goal state, evidence-bound completion, structured steering
- 5 ulw-loop MCP tools: create_goals, record_evidence, checkpoint, steer, status
- `laputa-cli` wrapper for skill-side invocation
- `loop-engineering` Hermes skill
- Daemon integration: ulw-loop MCP mounted
- Curator: daily health check on ulw-loop plans

### Security
- `weakens()` regex blocks steering that tries to skip tests / bypass quality gates
- Protected fields (status, aggregateCompletion, qualityGate) cannot be written via steering
- Idempotency keys prevent double-apply on retry
```

**Step 2:** 写 ulw-loop 集成文档
- 何时用 / 何时不用
- 5 个 tool 的 schema
- 完整 demo（用本计划作为 example）
- 与 OmO/LazyCodex 的差异

**Verify:** 文档齐全

**Commit:** `git add README.md CHANGELOG.md docs/ && git commit -m "docs: ulw-loop integration"`

---

### Task 6.3: 发版 tag

**Objective:** 打 `v0.2.0-rc1` tag，准备发版。

**Files:** N/A (git 操作)

**Step 1:**
```bash
cd ~/Desktop/projects/laputa-py
git tag -a v0.2.0-rc1 -m "ulw-loop paradigm integration"
git log --oneline -10
```

**Step 2:** 更新 pyproject.toml version
```toml
version = "0.2.0rc1"
```

**Verify:** tag 存在，version 正确

**Commit:** `git add pyproject.toml && git commit -m "chore: bump 0.2.0rc1"`

---

## Files Likely to Change

| 路径 | 状态 | Wave |
|---|---|---|
| `src/laputa/provider/queue.py` | restore | W1 |
| `src/laputa/ulw/__init__.py` | new | W2 |
| `src/laputa/ulw/types.py` | new | W2.1 |
| `src/laputa/ulw/storage.py` | new | W2.2 |
| `src/laputa/ulw/evidence.py` | new | W2.3 |
| `src/laputa/ulw/steering.py` | new | W2.4 |
| `src/laputa/ulw/apply.py` | new | W2.5 |
| `src/laputa/ulw/checkpoint.py` | new | W2.6 |
| `src/laputa/ulw/mcp_tools.py` | new | W3.2 |
| `src/laputa/provider/reconnect.py` | new | W3.5 |
| `src/laputa/provider/memory_provider.py` | modify | W3.5 |
| `src/laputa/cli.py` | modify | W3.3 |
| `src/laputa/server/daemon.py` | modify | W3.4, W5.2 |
| `src/laputa/curator/curator.py` | modify | W5.3 |
| `~/.hermes/skills/loop-engineering/SKILL.md` | new | W4.1 |
| `~/.hermes/skills/loop-engineering/scripts/laputa-cli.py` | new | W4.2 |
| `pyproject.toml` | modify | W3.1, W6.3 |
| `tests/ulw/*` | new | W2 |
| `tests/curator/test_ulw_loop_check.py` | new | W5.3 |
| `tests/e2e/test_loop_engineering_e2e.py` | new | W4.3 |
| `docs/architecture/ulw-loop-integration.md` | new | W6.2 |
| `CHANGELOG.md` | modify | W6.2 |

---

## Tests / Validation

### 单元测试

```bash
# 全测试
cd ~/Desktop/projects/laputa-py
python -m pytest tests/ -v

# 只跑 ulw 相关
python -m pytest tests/ulw/ -v
```

**预期**:
- W2 结束：6 文件，~25 tests pass
- W3 结束：+1 test (mcp_tools import) = ~26 tests
- W4 结束：+2 tests (frontmatter + e2e) = ~28 tests
- W5 结束：+1 test (curator ulw check) = ~29 tests
- W6 结束：基线记录

### 集成测试

```bash
# 启动 daemon + 跑 ulw-loop
./scripts/dev/e2e_daemon.sh
```

**预期**: 4 步全通过

### 端到端

```bash
# 完整 flow
pytest tests/e2e/ -v
```

**预期**: 1 passed

---

## 风险、权衡、待澄清问题

### 风险（高→低）

| 风险 | 影响 | 缓解 |
|---|---|---|
| **Daemon 集成破坏现有 palace_bridge / curator 调度** | 现有功能挂 | W5.1 先 audit，W5.2 小步改，每步跑测试 |
| **MCP 库版本不兼容** | 装不上 | W3.1 评估时选最稳的 `mcp` 官方 SDK |
| **TypeScript ulw-loop 翻译语义漂移** | 行为差异 | 测试用例尽量从 ulw-loop 测试用例 1:1 翻译 |
| **filelock 跨平台坑** | Windows 死锁 | 已选 `filelock`（成熟库），加 `tests/test_queue.py` 验证 |
| **worktree-only 改动还是 main 直改？** | git 纪律 | 大湿拍板：保留 main 继续推进（基于 6-26 计划） |

### 权衡

| 选择 | 取舍 |
|---|---|
| **W2 翻译 TypeScript → Python** vs 直接子进程调 ulw-loop | 翻译：自主可控、零外部依赖；子进程：快速但不透明。**选翻译** |
| **`mcp` 官方 SDK** vs 自实现 JSON-RPC | 官方：跨客户端稳；自实现：可控但要自己测。**选官方** |
| **ulw-loop MCP 塞进现有 daemon** vs 独立进程 | 塞：复用 lock / 一组端口；独立：隔离好。**选塞进**（现有 daemon 已稳定） |
| **filelock** vs portalocker vs `os.lockf` | filelock 跨平台 + API 友好；portalocker 也行但 Windows 行为差异。**选 filelock** |
| **Wave 顺序 W1→W6** vs 并行 | W1 串行（清理）；W2-W4 可并行但任务依赖；W5 必须 W2-W4 完。**选串行** |

### 待澄清问题

1. **MCP 协议选择**：stdio（默认）还是 socket？
   - stdio：单 client，安全
   - socket：多 client 共享
   - **建议**：daemon 模式走 socket（多 client），CLI 模式走 stdio

2. **daemon 端口冲突**：现有 daemon 用什么端口？要不要做端口分配？
   - **建议**：在 `~/.laputa/daemon.port` 存分配

3. **ulw-loop plan 存哪里**：`<repo>/.laputa/ulw-loop/`（per-repo） vs `~/.laputa/ulw-loop/<repo-hash>/`（global）？
   - **建议**：per-repo（与 ulw-loop 一致），方便 git ignore / 跨机器迁移

4. **Laputa 的 8 file 治理 vs ulw-loop plan 的关系**：
   - 8 files 是**长期身份 / 记忆**
   - ulw-loop plan 是**短期任务**
   - **建议**：8 files 不动，ulw-loop plan 完全独立；curator 把 ulw-loop 完成摘要定期写进 MEMORY.MD

5. **W6 标签策略**：要不要先发 `v0.2.0-rc1` 等大湿确认再 `v0.2.0`？
   - **建议**：先 rc1，看一周稳定再 stable

---

## 决策日志（与 6-26 计划呼应）

| # | 新决策 | 理由 |
|---|---|---|
| 1 | 引入 `mcp` 官方 Python SDK 作为 MCP 协议层 | 标准、跨客户端、官方维护 |
| 2 | 引入 `filelock` 作为跨进程锁 | 跨平台、Windows 兼容、API 友好 |
| 3 | ulw-loop plan 存 per-repo `.laputa/ulw-loop/` | 与 ulw-loop 原版一致；与 8 file 全局治理解耦 |
| 4 | ulw-loop MCP 挂进现有 daemon（不独立进程） | 复用 lock + curator 调度 + 端口 |
| 5 | weakening 防御用正则而非语义模型 | 轻量、零额外依赖、5 行代码 80% 防御力 |
| 6 | steering 走 7 种结构化 kind，自然语言 steering 全部 reject | 防伪 + 可审计 + 可重放 |
| 7 | checkpoint 失败 3 次同类 blocker → 自动 `needs_user_decision` | 避免无限循环、主动升级人类 |
| 8 | 提供 `laputa-cli` Python wrapper 而非要求 skill 直接调 Python 模块 | 错误信息友好、JSON 输出、易测试 |
| 9 | Curator 加 ulw-loop 巡检而非 ulw-loop 自己定时 | 复用 curator 调度、统一通知渠道 |
| 10 | 标签 `v0.2.0-rc1` 等大湿确认再 stable | 重大架构变更的稳妥做法 |

---

## 执行准备清单

- [ ] 确认 main 分支（确认 ✓）
- [ ] 确认大湿授权本计划执行（待确认）
- [ ] 检查 `mcp` 和 `filelock` 可装（依赖 Python 环境）
- [ ] 检查 daemon 端口范围（建议 8765-8799）

---

**Plan 状态**: 已保存到 `~/Desktop/MATSUMOTO-overall-refactor-2026Q3.md`

**下一步**: 等大湿确认后，用 `subagent-driven-development` skill 按 Wave 1→6 顺序执行。每 Wave 完成 checkpoint 一次。
