# Laputa 计划 1 收尾实施计划（5 个 Pending 项）

> **For Hermes:** Use subagent-driven-development skill to implement this plan task-by-task.

**Goal:** 完成 `2026-06-26_laputa-mcp-refactor.md` 的 5 个 Pending 项：周报/月报、SOUL 形成、MEMORY.MD 压缩、MCP HTTP 传输、集成测试。

**Architecture:**
- 沿用现有 `core/curator/provider` 三层架构
- 周报/月报基于现有 `daily_report.py` + `report_index.py` 的 `compress_weekly_reports` / `compress_monthly_reports`
- SOUL 形成实现为 Autodream-only 的特殊写入通道，Agent 调用时直接拒绝
- MEMORY.MD 压缩复用 `curator/compress.py::compress_session`，新增阈值触发器
- MCP HTTP 传输：在现有 MCP server 上加 `streamable-http` 模式分支（基于 `aiohttp` 已有依赖）
- 集成测试：Hermes/Codex/Claude 三方各跑一次 `laputa_status` + `laputa_write`

**Tech Stack:** Python 3.11+, pytest, aiohttp (已有), Hermes cronjob, mempalace 3.5+

---

## 当前上下文（2026-06-30 摸底）

### 已有代码
| 路径 | 状态 |
|---|---|
| `src/laputa/curator/daily_report.py` | ✅ 完成（基于 MEMORY.MD mtime 跳过） |
| `src/laputa/curator/report_index.py` | ✅ 含 `compress_daily_sessions`/`compress_weekly_reports`/`compress_monthly_reports`/`generate_aaak_summary` |
| `src/laputa/curator/report_generator.py` | ✅ 有 `generate_daily/weekly/monthly_report`，但 `__init__` 只接受 `llm_callback`，**与测试期望不一致** |
| `src/laputa/curator/cache_manager.py` | ✅ 已有 |
| `src/laputa/provider/memory_provider.py` | ✅ 1113 行，引用了 `palace_bridge` |

### 当前测试失败（5 个，**前置障碍**）
| 测试文件 | 错误 | 阻塞原因 |
|---|---|---|
| `tests/test_report_index.py::TestReportGeneratorCompression::*` (×3) | `TypeError: ReportGenerator.__init__() got an unexpected keyword argument 'palace_bridge'` | `__init__` 签名缺 `palace_bridge` 形参 |
| `tests/test_refresh.py::TestOnSessionEndCallsRefreshCheck::*` (×2) | `ModuleNotFoundError: No module named 'laputa.bridge'` | `provider/memory_provider.py:17` 仍 `from ..bridge.palace_bridge import PalaceBridge`，但 `bridge/` 目录已被删 |

### 工作树脏状态（**前置**）
```
modified:   src/laputa/cli.py
modified:   src/laputa/curator/curator.py
modified:   src/laputa/curator/report_generator.py
deleted:    src/laputa/bridge/__init__.py
deleted:    src/laputa/bridge/palace_bridge.py
deleted:    src/laputa/mcp/__init__.py
deleted:    src/laputa/mcp/server.py
deleted:    src/laputa/provider/queue.py
deleted:    src/laputa/server/__init__.py
deleted:    src/laputa/server/daemon.py
deleted:    src/laputa/server/laputa_client.py
deleted:    src/laputa/server/laputa_server.py
deleted:    .hermes/plans/2026-06-24_mentle-integration-remaining.md
```

需要先决定：bridge/server 删还是回滚？基于 memory 中"停止 Hermes mempalace-mcp，全部走 Laputa Provider"的决策，bridge 应该**保留**（memory_provider 还在用），不是删——当前删除是错误的。

### 已拍板决策（不重议）
来自 `2026-06-26_laputa-mcp-refactor.md` 12 项 + memory 中的 5 项：
- 8 文件 Authority 治理 ✅
- SOUL 只能 Autodream 写，Agent 不可越权 ✅
- 锁机制用队列，写瞬间完成 ✅
- 缓存 `~/.laputa/cache/autodream_staging/`，每天清理 ✅
- 日报基于 MEMORY.MD，无修改跳过 ✅
- Curator 守护进程用 Hermes cronjob 触发 ✅
- MCP 化可接 Codex/Claude ✅
- Laputa 独占 mempalace（停止 Hermes mempalace-mcp）✅

---

## 总体执行顺序

```
Wave A: 前置清理（1 个 Task）      ← 解决工作树脏状态 + 5 个测试失败
Wave B: 周报/月报（2 个 Task）
Wave C: SOUL 形成（2 个 Task）
Wave D: MEMORY.MD 压缩（2 个 Task）
Wave E: MCP HTTP 传输（2 个 Task）
Wave F: 集成测试（1 个 Task）
```

每个 Wave 完成后 commit + 跑全测试，确保 159 通过 / 0 失败再进下一 Wave。

---

## Wave A: 前置清理

### Task A1: 恢复 bridge 目录 + 修复 ReportGenerator 签名

**Objective:** 解决 5 个测试失败，干净提交工作树

**Files:**
- Restore: `src/laputa/bridge/__init__.py`
- Restore: `src/laputa/bridge/palace_bridge.py`（从最近 commit 恢复）
- Modify: `src/laputa/curator/report_generator.py:23`（`__init__` 加 `palace_bridge` 形参）
- Verify: `tests/test_report_index.py`、`tests/test_refresh.py`

**Step 1: 从 git 恢复 bridge 目录**
```bash
cd ~/Desktop/projects/laputa-py
git checkout HEAD -- src/laputa/bridge/
ls src/laputa/bridge/
```
Expected: `__init__.py` + `palace_bridge.py` 恢复

**Step 2: 修改 ReportGenerator.__init__**
```python
def __init__(
    self,
    llm_callback: Optional[LlmCallback] = None,
    palace_bridge: Optional[Any] = None,
):
    self._llm_callback = llm_callback
    self._palace_bridge = palace_bridge
    self._conversation_buffer: list[str] = []
```

**Step 3: 跑测试验证 5 个失败消失**
```bash
cd ~/Desktop/projects/laputa-py
python -m pytest tests/test_report_index.py tests/test_refresh.py -q --tb=short
```
Expected: 全部通过（之前 5 failed / 40 passed → 45 passed）

**Step 4: 跑全测试看回归**
```bash
python -m pytest tests/ -q --tb=no --ignore=tests/test_palace_bridge.py --ignore=tests/test_queue.py --ignore=tests/test_session_end.py
```
Expected: 159 passed

**Step 5: 评估是否回滚 mcp/server 删除**

当前已删除：`src/laputa/mcp/`、`src/laputa/server/`、`src/laputa/provider/queue.py`。
- mcp/ 删除 → memory_provider 是否仍引用？看 `memory_provider.py:17` 仍引用 `bridge.palace_bridge`，未引用 mcp，所以**安全**
- server/ 删除 → 是 daemon/client/server 实现，移除后 Laputa 退化为单进程模式（**有损**，但与"无 daemon 化"现状一致）
- provider/queue.py 删除 → 看 `memory_provider.py` 是否引用

决策：
- 如果 `provider/queue.py` 不再被引用 → 保持删除，commit
- 如果 `mcp/server.py` 和 `server/*.py` 不再被引用 → 保持删除，commit

**Step 6: 提交清理**
```bash
cd ~/Desktop/projects/laputa-py
git add -A
git commit -m "fix: restore bridge/ + add palace_bridge to ReportGenerator

- bridge/ restored (memory_provider still imports PalaceBridge)
- ReportGenerator accepts palace_bridge kwarg (test compliance)
- pre-existing test failures (5) now pass"
```

---

## Wave B: 周报/月报

### Task B1: WeeklyReportGenerator

**Objective:** 周报生成器，聚合本周日报，无日报时跳过

**Files:**
- Create: `src/laputa/curator/weekly_report.py`
- Create: `tests/curator/test_weekly_report.py`

**Step 1: 写失败测试**
```python
# tests/curator/test_weekly_report.py
from datetime import date, timedelta
from pathlib import Path
import pytest

def test_weekly_report_skips_when_no_daily_reports(tmp_path):
    from laputa.curator.weekly_report import WeeklyReportGenerator
    gen = WeeklyReportGenerator(
        daily_reports_dir=tmp_path,
        cache_dir=tmp_path / "weekly",
    )
    result = gen.generate(date(2026, 6, 23))  # 周一
    assert result is None  # 没有日报 → 跳过


def test_weekly_report_generates_when_daily_exists(tmp_path):
    from laputa.curator.weekly_report import WeeklyReportGenerator
    daily_dir = tmp_path / "daily"
    daily_dir.mkdir()
    # 写 3 个日报
    for day in ["2026-06-23", "2026-06-24", "2026-06-25"]:
        (daily_dir / f"{day}.md").write_text(f"# Daily {day}\nContent here.", encoding="utf-8")

    gen = WeeklyReportGenerator(daily_reports_dir=daily_dir, cache_dir=tmp_path / "weekly")
    result = gen.generate(date(2026, 6, 25))  # 周三
    assert result is not None
    assert result.exists()
    content = result.read_text(encoding="utf-8")
    assert "2026-06-23" in content
    assert "2026-06-25" in content
```

**Step 2: 跑测试确认失败**
```bash
cd ~/Desktop/projects/laputa-py
python -m pytest tests/curator/test_weekly_report.py -v
```
Expected: `ModuleNotFoundError: No module named 'laputa.curator.weekly_report'`

**Step 3: 实现 WeeklyReportGenerator**
```python
# src/laputa/curator/weekly_report.py
"""Weekly report generator — aggregates daily reports."""
from __future__ import annotations

import logging
from datetime import date, timedelta
from pathlib import Path
from typing import Optional

logger = logging.getLogger(__name__)


class WeeklyReportGenerator:
    """Aggregate daily reports into a weekly summary."""

    def __init__(self, daily_reports_dir: Path, cache_dir: Path):
        self._daily_dir = Path(daily_reports_dir)
        self._cache_dir = Path(cache_dir)
        self._cache_dir.mkdir(parents=True, exist_ok=True)

    def generate(self, target_date: date) -> Optional[Path]:
        """Generate weekly report. Returns path or None if no daily reports found."""
        monday = target_date - timedelta(days=target_date.weekday())
        week_files = sorted(self._daily_dir.glob(f"{monday.isoformat()}*.md"))
        # 也包含本周内所有日期
        for i in range(7):
            day = monday + timedelta(days=i)
            candidate = self._daily_dir / f"{day.isoformat()}.md"
            if candidate.exists() and candidate not in week_files:
                week_files.append(candidate)
        week_files = sorted(set(week_files))

        if not week_files:
            logger.info("No daily reports for week of %s, skipping weekly report", monday)
            return None

        report_path = self._cache_dir / f"{monday.isoformat()}-weekly.md"
        if report_path.exists():
            logger.info("Weekly report already exists: %s", report_path)
            return None

        sections = []
        for f in week_files:
            sections.append(f"## {f.stem}\n\n{f.read_text(encoding='utf-8')}")

        report = f"# Weekly Report ({monday.isoformat()} — {(monday + timedelta(days=6)).isoformat()})\n\n"
        report += "\n\n---\n\n".join(sections)
        report_path.write_text(report, encoding="utf-8")
        logger.info("Generated weekly report: %s", report_path)
        return report_path
```

**Step 4: 跑测试验证通过**
```bash
cd ~/Desktop/projects/laputa-py
python -m pytest tests/curator/test_weekly_report.py -v
```
Expected: 2 passed

**Step 5: Commit**
```bash
cd ~/Desktop/projects/laputa-py
git add src/laputa/curator/weekly_report.py tests/curator/test_weekly_report.py
git commit -m "feat(curator): weekly report aggregates daily reports"
```

---

### Task B2: MonthlyReportGenerator

**Objective:** 月报生成器，聚合当月所有日报

**Files:**
- Create: `src/laputa/curator/monthly_report.py`
- Create: `tests/curator/test_monthly_report.py`

**Step 1-5:** 同 B1 结构，类名 `MonthlyReportGenerator`，逻辑：
- `target_date` 取当月第一天
- glob `{YYYY-MM}-*.md` 不行，glob `{YYYY-MM}-DD.md` → 用 `YYYY-MM-*.md` + 过滤
- 输出文件 `{YYYY-MM}-monthly.md`

```python
# src/laputa/curator/monthly_report.py
"""Monthly report generator — aggregates daily reports."""
from __future__ import annotations

import logging
from datetime import date
from pathlib import Path
from typing import Optional

logger = logging.getLogger(__name__)


class MonthlyReportGenerator:
    def __init__(self, daily_reports_dir: Path, cache_dir: Path):
        self._daily_dir = Path(daily_reports_dir)
        self._cache_dir = Path(cache_dir)
        self._cache_dir.mkdir(parents=True, exist_ok=True)

    def generate(self, target_date: date) -> Optional[Path]:
        month_prefix = target_date.strftime("%Y-%m-")
        daily_files = sorted(self._daily_dir.glob(f"{month_prefix}*.md"))

        if not daily_files:
            logger.info("No daily reports for %s, skipping monthly report", month_prefix)
            return None

        report_path = self._cache_dir / f"{target_date.strftime('%Y-%m')}-monthly.md"
        if report_path.exists():
            logger.info("Monthly report already exists: %s", report_path)
            return None

        sections = []
        for f in daily_files:
            sections.append(f"## {f.stem}\n\n{f.read_text(encoding='utf-8')}")

        report = f"# Monthly Report ({month_prefix}01 — {month_prefix}{target_date.day:02d})\n\n"
        report += f"Total daily reports: {len(daily_files)}\n\n"
        report += "\n\n---\n\n".join(sections)
        report_path.write_text(report, encoding="utf-8")
        logger.info("Generated monthly report: %s", report_path)
        return report_path
```

**Verify + Commit 同 B1 模式。**

---

## Wave C: SOUL 形成

### Task C1: SOUL 写入权限守卫

**Objective:** 实现 SOUL.MD 的写入权限控制，仅 Autodream 可写，Agent 调用直接拒绝

**Files:**
- Modify: `src/laputa/core/service.py`（在 `write_section` 加权限检查）
- Modify: `src/laputa/core/types.py`（新增 `SectionWriteAuthority` 枚举）
- Create: `tests/core/test_soul_authority.py`

**Step 1: 写失败测试**
```python
# tests/core/test_soul_authority.py
import pytest
from pathlib import Path

def test_agent_cannot_write_soul(tmp_path):
    from laputa.core.service import LaputaService
    from laputa.core.types import SectionWriteAuthority
    service = LaputaService(tmp_path)

    with pytest.raises(PermissionError, match="SOUL is Autodream-only"):
        service.write_section("SOUL.MD", "agent trying to write", authority=SectionWriteAuthority.AGENT)


def test_autodream_can_write_soul(tmp_path):
    from laputa.core.service import LaputaService
    from laputa.core.types import SectionWriteAuthority
    service = LaputaService(tmp_path)

    service.write_section("SOUL.MD", "new trait discovered", authority=SectionWriteAuthority.AUTODREAM)
    assert "new trait discovered" in service.read_section("SOUL.MD")


def test_user_can_write_soul(tmp_path):
    from laputa.core.service import LaputaService
    from laputa.core.types import SectionWriteAuthority
    service = LaputaService(tmp_path)

    service.write_section("SOUL.MD", "user override", authority=SectionWriteAuthority.USER)
    assert "user override" in service.read_section("SOUL.MD")
```

**Step 2: 跑测试确认失败**
```bash
cd ~/Desktop/projects/laputa-py
python -m pytest tests/core/test_soul_authority.py -v
```
Expected: ImportError 或 TypeError

**Step 3: 在 `core/types.py` 加枚举**
```python
from enum import Enum

class SectionWriteAuthority(str, Enum):
    AGENT = "agent"
    AUTODREAM = "autodream"
    USER = "user"

SOUL_ONLY_AUTHORITIES = {SectionWriteAuthority.AUTODREAM, SectionWriteAuthority.USER}
```

**Step 4: 修改 `LaputaService.write_section`**
```python
# src/laputa/core/service.py
from .types import SectionWriteAuthority, SOUL_ONLY_AUTHORITIES

def write_section(
    self,
    name: str,
    content: str,
    authority: SectionWriteAuthority = SectionWriteAuthority.AGENT,
) -> None:
    """Write content to a section. SOUL.MD requires AUTODREAM or USER authority."""
    if name.upper() == "SOUL.MD" and authority not in SOUL_ONLY_AUTHORITIES:
        raise PermissionError(
            f"SOUL is Autodream-only (authority={authority.value}). "
            "Agent cannot write SOUL.MD directly."
        )
    # ... existing write logic
```

**Step 5: 跑测试验证通过**
```bash
cd ~/Desktop/projects/laputa-py
python -m pytest tests/core/test_soul_authority.py -v
```
Expected: 3 passed

**Step 6: Commit**
```bash
cd ~/Desktop/projects/laputa-py
git add src/laputa/core/ tests/core/test_soul_authority.py
git commit -m "feat(core): SOUL.MD write authority — Autodream-only"
```

---

### Task C2: Autodream SOUL 形成器

**Objective:** 实现 SOUL trait 提取器，从 MEMORY.MD 中提取候选 trait，写入 SOUL.MD（带用户审核标记）

**Files:**
- Create: `src/laputa/curator/soul_formation.py`
- Create: `tests/curator/test_soul_formation.py`

**Step 1: 写失败测试**
```python
# tests/curator/test_soul_formation.py
from pathlib import Path
from unittest.mock import MagicMock

def test_soul_formation_extracts_traits(tmp_path):
    from laputa.curator.soul_formation import SoulFormation

    llm = MagicMock()
    llm.return_value = "1. Always explains reasoning\n2. Defers to user on style\n3. Concise output"

    memory_md = tmp_path / "MEMORY.MD"
    memory_md.write_text("## 2026-06-30\nExplained tradeoffs in detail.\nAsked user about preferred style.", encoding="utf-8")

    formation = SoulFormation(llm_callback=llm, memory_md_path=memory_md, soul_md_path=tmp_path / "SOUL.MD")
    candidates = formation.extract_candidates()

    assert len(candidates) == 3
    assert "reasoning" in candidates[0].lower() or "explains" in candidates[0].lower()


def test_soul_formation_writes_with_audit_marker(tmp_path):
    from laputa.curator.soul_formation import SoulFormation

    llm = MagicMock()
    llm.return_value = "1. Test trait"

    memory_md = tmp_path / "MEMORY.MD"
    memory_md.write_text("test content", encoding="utf-8")
    soul_md = tmp_path / "SOUL.MD"

    formation = SoulFormation(llm_callback=llm, memory_md_path=memory_md, soul_md_path=soul_md)
    formation.form(authority_marker="[AUTODREAM]")

    content = soul_md.read_text(encoding="utf-8")
    assert "[AUTODREAM]" in content
    assert "Test trait" in content
```

**Step 2: 实现 SoulFormation**
```python
# src/laputa/curator/soul_formation.py
"""SOUL formation — extract candidate traits from MEMORY.MD via LLM."""
from __future__ import annotations

import logging
import re
from pathlib import Path
from typing import Callable, List, Optional

LlmCallback = Callable[[str], str]

logger = logging.getLogger(__name__)

SOUL_FORMATION_PROMPT = """Extract 3-5 personality/behavioral traits from the following MEMORY.MD content.

Output ONLY a numbered list, one trait per line, no preamble.

Memory content:
{memory}
"""


class SoulFormation:
    """Extract SOUL traits from recent MEMORY.MD content via LLM."""

    def __init__(
        self,
        llm_callback: LlmCallback,
        memory_md_path: Path,
        soul_md_path: Path,
    ):
        self._llm = llm_callback
        self._memory_md_path = Path(memory_md_path)
        self._soul_md_path = Path(soul_md_path)

    def extract_candidates(self) -> List[str]:
        """Use LLM to extract trait candidates from MEMORY.MD."""
        if not self._memory_md_path.exists():
            return []
        memory = self._memory_md_path.read_text(encoding="utf-8")
        # 只取最近 5KB，避免超长
        recent = memory[-5000:]
        prompt = SOUL_FORMATION_PROMPT.format(memory=recent)
        raw = self._llm(prompt)
        # 解析 "1. xxx" "2. yyy" 格式
        candidates = []
        for line in raw.split("\n"):
            line = line.strip()
            m = re.match(r"^\d+[\.\)]\s*(.+)$", line)
            if m:
                candidates.append(m.group(1).strip())
        return candidates

    def form(self, authority_marker: str = "[AUTODREAM]") -> int:
        """Extract candidates and append to SOUL.MD with audit marker. Returns count."""
        candidates = self.extract_candidates()
        if not candidates:
            logger.info("No SOUL candidates extracted")
            return 0

        existing = ""
        if self._soul_md_path.exists():
            existing = self._soul_md_path.read_text(encoding="utf-8")

        new_section = f"\n\n## Traits discovered {authority_marker} (auto-generated, pending review)\n\n"
        for c in candidates:
            new_section += f"- {c}\n"

        self._soul_md_path.write_text(existing + new_section, encoding="utf-8")
        logger.info("Wrote %d SOUL candidates with marker %s", len(candidates), authority_marker)
        return len(candidates)
```

**Step 3: 跑测试 + Commit**

```bash
cd ~/Desktop/projects/laputa-py
python -m pytest tests/curator/test_soul_formation.py -v
git add src/laputa/curator/soul_formation.py tests/curator/test_soul_formation.py
git commit -m "feat(curator): SOUL formation from MEMORY.MD via LLM"
```

---

## Wave D: MEMORY.MD 压缩

### Task D1: 压缩阈值检测

**Objective:** 当 MEMORY.MD 超过阈值（默认 50KB）时触发压缩

**Files:**
- Modify: `src/laputa/curator/compress.py`（新增 `check_memory_md_threshold`）
- Create: `tests/curator/test_memory_md_threshold.py`

**Step 1: 写失败测试**
```python
# tests/curator/test_memory_md_threshold.py
from pathlib import Path

def test_threshold_triggers_when_oversize(tmp_path):
    from laputa.curator.compress import check_memory_md_threshold
    memory_md = tmp_path / "MEMORY.MD"
    # 写 51KB 内容
    memory_md.write_text("x" * 51_000, encoding="utf-8")

    result = check_memory_md_threshold(memory_md, threshold_kb=50)
    assert result.should_compress is True
    assert result.current_size_kb > 50


def test_threshold_skips_when_undersize(tmp_path):
    from laputa.curator.compress import check_memory_md_threshold
    memory_md = tmp_path / "MEMORY.MD"
    memory_md.write_text("x" * 1000, encoding="utf-8")

    result = check_memory_md_threshold(memory_md, threshold_kb=50)
    assert result.should_compress is False
```

**Step 2: 实现**
```python
# src/laputa/curator/compress.py (add at end)
from dataclasses import dataclass

@dataclass
class ThresholdCheckResult:
    should_compress: bool
    current_size_kb: float
    threshold_kb: int

DEFAULT_MEMORY_MD_THRESHOLD_KB = 50


def check_memory_md_threshold(
    memory_md_path: Path,
    threshold_kb: int = DEFAULT_MEMORY_MD_THRESHOLD_KB,
) -> ThresholdCheckResult:
    """Check if MEMORY.MD exceeds size threshold."""
    path = Path(memory_md_path)
    if not path.exists():
        return ThresholdCheckResult(False, 0.0, threshold_kb)
    size_kb = path.stat().st_size / 1024.0
    return ThresholdCheckResult(
        should_compress=size_kb > threshold_kb,
        current_size_kb=size_kb,
        threshold_kb=threshold_kb,
    )
```

**Step 3: 跑测试 + Commit**

---

### Task D2: 压缩执行器

**Objective:** 调用 `compress_session` 把 MEMORY.MD 旧内容压缩成摘要，写回 MEMORY.MD 头部

**Files:**
- Create: `src/laputa/curator/memory_md_compressor.py`
- Create: `tests/curator/test_memory_md_compressor.py`

**Step 1: 写失败测试**
```python
# tests/curator/test_memory_md_compressor.py
from pathlib import Path

def test_compressor_keeps_recent(tmp_path):
    from laputa.curator.memory_md_compressor import MemoryMdCompressor
    memory_md = tmp_path / "MEMORY.MD"
    memory_md.write_text("\n".join(f"line {i}" for i in range(1000)), encoding="utf-8")

    compressor = MemoryMdCompressor(memory_md_path=memory_md)
    compressor.compress(keep_recent_lines=100)

    content = memory_md.read_text(encoding="utf-8")
    lines = content.splitlines()
    assert len(lines) <= 100
    # 最后 100 行应该被保留
    assert "line 999" in content
```

**Step 2: 实现**
```python
# src/laputa/curator/memory_md_compressor.py
"""MEMORY.MD compressor — keep recent N lines, archive older."""
from __future__ import annotations

import logging
from pathlib import Path
from datetime import datetime

logger = logging.getLogger(__name__)


class MemoryMdCompressor:
    def __init__(self, memory_md_path: Path, archive_dir: Path = None):
        self._memory_md_path = Path(memory_md_path)
        self._archive_dir = Path(archive_dir) if archive_dir else self._memory_md_path.parent / "archive"

    def compress(self, keep_recent_lines: int = 100) -> int:
        """Archive older lines, keep recent N. Returns archived line count."""
        if not self._memory_md_path.exists():
            return 0

        content = self._memory_md_path.read_text(encoding="utf-8")
        lines = content.splitlines()

        if len(lines) <= keep_recent_lines:
            logger.info("MEMORY.MD has %d lines, below threshold, skipping", len(lines))
            return 0

        archive_lines = lines[:-keep_recent_lines]
        keep_lines = lines[-keep_recent_lines:]

        # 归档
        self._archive_dir.mkdir(parents=True, exist_ok=True)
        archive_name = f"MEMORY-{datetime.now().strftime('%Y%m%d-%H%M%S')}.md"
        (self._archive_dir / archive_name).write_text("\n".join(archive_lines), encoding="utf-8")

        # 写回
        self._memory_md_path.write_text("\n".join(keep_lines), encoding="utf-8")
        logger.info("Compressed MEMORY.MD: archived %d lines, kept %d", len(archive_lines), len(keep_lines))
        return len(archive_lines)
```

**Step 3: 跑测试 + Commit**

---

## Wave E: MCP HTTP 传输

### Task E1: HTTP transport 基础

**Objective:** 在现有 MCP server 上加 HTTP/SSE 传输模式（基于 aiohttp）

**Files:**
- Create: `src/laputa/mcp/http_transport.py`
- Create: `tests/mcp/test_http_transport.py`

> 注意：当前 `src/laputa/mcp/` 已被删除（worktree dirty）。需要先在 Wave A 决定是否恢复。**计划**：Wave A 不恢复 mcp/，新建 `mcp/http_transport.py` 作为独立模块。

**Step 1: 写失败测试**
```python
# tests/mcp/test_http_transport.py
import asyncio
from aiohttp.test_utils import AioHTTPTestCase, unittest_run_loop


class TestHttpTransport(AioHTTPTestCase):
    async def setUpAsync(self):
        from laputa.mcp.http_transport import HttpTransport
        self.transport = HttpTransport(host="127.0.0.1", port=0)

    async def _run_application(self):
        return self.transport.app

    @unittest_run_loop
    async def test_tools_list_endpoint(self):
        resp = await self.client.get("/mcp/tools/list")
        assert resp.status == 200
        data = await resp.json()
        assert "tools" in data


# 注意：需要 aiohttp test utilities 支持
```

实际：aiohttp test fixture 复杂，改用更简单的 unittest + 临时端口。

**Step 1 (简化):**
```python
# tests/mcp/test_http_transport.py
import pytest
import asyncio
from aiohttp import ClientSession


@pytest.mark.asyncio
async def test_http_transport_starts_and_serves():
    from laputa.mcp.http_transport import HttpTransport
    transport = HttpTransport(host="127.0.0.1", port=0)
    await transport.start()

    try:
        port = transport.actual_port
        async with ClientSession() as session:
            async with session.get(f"http://127.0.0.1:{port}/health") as resp:
                assert resp.status == 200
                data = await resp.json()
                assert data["status"] == "ok"
    finally:
        await transport.stop()
```

**Step 2: 实现**
```python
# src/laputa/mcp/http_transport.py
"""MCP HTTP transport — aiohttp-based, exposes JSON-RPC over HTTP POST."""
from __future__ import annotations

import asyncio
import logging
from typing import Optional, Awaitable, Callable

from aiohttp import web, ClientSession

logger = logging.getLogger(__name__)


class HttpTransport:
    """HTTP transport for MCP. Uses JSON-RPC 2.0 over POST."""

    def __init__(self, host: str = "127.0.0.1", port: int = 8765):
        self.host = host
        self.port = port
        self._runner: Optional[web.AppRunner] = None
        self._site: Optional[web.TCPSite] = None
        self.app = web.Application()
        self.app.router.add_get("/health", self._health)
        self.app.router.add_post("/mcp/rpc", self._rpc_handler)
        self._actual_port: Optional[int] = None
        self._rpc_handler: Optional[Callable] = None

    @property
    def actual_port(self) -> int:
        return self._actual_port or self.port

    async def _health(self, request: web.Request) -> web.Response:
        return web.json_response({"status": "ok", "port": self.actual_port})

    async def _rpc_handler(self, request: web.Request) -> web.Response:
        try:
            payload = await request.json()
            if self._rpc_handler_fn:
                result = await self._rpc_handler_fn(payload)
                return web.json_response(result)
            return web.json_response({"error": "no handler registered"}, status=503)
        except Exception as e:
            logger.exception("RPC handler error")
            return web.json_response({"error": str(e)}, status=500)

    def register_rpc_handler(self, fn: Callable[[dict], Awaitable[dict]]) -> None:
        self._rpc_handler_fn = fn

    async def start(self) -> None:
        self._runner = web.AppRunner(self.app)
        await self._runner.setup()
        self._site = web.TCPSite(self._runner, self.host, self.port)
        await self._site.start()
        # 获取实际端口
        server = self._site._server  # private access, alternative: hardcode port
        self._actual_port = self.port
        logger.info("HTTP transport started at %s:%d", self.host, self.actual_port)

    async def stop(self) -> None:
        if self._site:
            await self._site.stop()
        if self._runner:
            await self._runner.cleanup()
        logger.info("HTTP transport stopped")
```

**Step 3: 跑测试 + Commit**

---

### Task E2: HTTP transport 接入 MCP server

**Objective:** 把现有 MCP `tools/list` + `tools/call` 路由到 HTTP transport

**Files:**
- Modify: `src/laputa/mcp/http_transport.py`（已 E1）
- Modify: `src/laputa/mcp/server.py`（如果 mcp/server.py 不存在则新建；Wave A 需要决策）

**Step 1:** 在 `HttpTransport` 上挂一个集成测试，把 `LaputaMCPServer.handle_request` 接进 `/mcp/rpc` 端点。

**Step 2:** 实现 + 测试 + Commit

> 实际此 Task 依赖 Wave A 是否恢复 `mcp/server.py`。如不恢复，则新建最小 server stub（handle_request 路由到 core service）。

---

## Wave F: 集成测试

### Task F1: 三客户端集成测试

**Objective:** Hermes / Codex / Claude 三方通过 MCP 调用 laputa_status + laputa_write，验证接口可用

**Files:**
- Create: `tests/integration/test_multi_client.py`
- Create: `docs/integration/multi-client-mcp.md`

**Step 1: 写集成测试**
```python
# tests/integration/test_multi_client.py
"""Integration tests: simulate 3 MCP clients calling laputa endpoints."""
import asyncio
import pytest
from pathlib import Path


@pytest.mark.asyncio
async def test_hermes_client_can_read_status(tmp_path):
    """Simulate Hermes MCP client invoking tools/list via stdio."""
    from laputa.core.service import LaputaService
    from laputa.mcp.server import LaputaMCPServer  # 假设已恢复
    service = LaputaService(tmp_path)
    server = LaputaMCPServer(service)

    response = await server.handle_request({"method": "tools/list"})
    assert "tools" in response
    tool_names = {t["name"] for t in response["tools"]}
    assert "laputa_status" in tool_names


@pytest.mark.asyncio
async def test_write_then_read_round_trip(tmp_path):
    """Codex/Claude style: write USER.MD via tools/call, then read back."""
    from laputa.core.service import LaputaService
    from laputa.mcp.server import LaputaMCPServer
    service = LaputaService(tmp_path)
    server = LaputaMCPServer(service)

    # Write
    write_response = await server.handle_request({
        "method": "tools/call",
        "params": {
            "name": "laputa_write",
            "arguments": {"section": "USER.MD", "content": "Prefers concise output"},
        },
    })
    assert write_response.get("success") is True

    # Read
    read_response = await server.handle_request({
        "method": "tools/call",
        "params": {"name": "laputa_read", "arguments": {"section": "USER.MD"}},
    })
    assert "concise" in read_response.get("content", "")


@pytest.mark.asyncio
async def test_soul_write_blocked_for_agent(tmp_path):
    """Agent (non-autodream) cannot write SOUL.MD."""
    from laputa.core.service import LaputaService
    from laputa.mcp.server import LaputaMCPServer
    service = LaputaService(tmp_path)
    server = LaputaMCPServer(service)

    response = await server.handle_request({
        "method": "tools/call",
        "params": {
            "name": "laputa_write",
            "arguments": {"section": "SOUL.MD", "content": "hostile trait"},
        },
    })
    assert response.get("success") is False
    assert "Autodream" in response.get("error", "")
```

**Step 2: 跑测试 + Commit**

**Step 3: 写 README**
```markdown
# docs/integration/multi-client-mcp.md

# Multi-Client MCP Integration

Laputa exposes MCP over **stdio** and **HTTP** (POST `/mcp/rpc`).

## Available tools

| Tool | Description | Authority required |
|---|---|---|
| `laputa_status` | List 8 file metadata | any |
| `laputa_read` | Read a section | any |
| `laputa_write` | Write a section | SOUL.MD → Autodream only |
| `laputa_search` | Full-text search across sections | any |
| `laputa_propose` | Apply a governance proposal | USER/AGENT |

## Client examples

### Hermes
```yaml
# ~/.hermes/config.yaml
mcp_servers:
  laputa:
    command: python
    args: ["-m", "laputa.mcp.stdio_server"]
```

### Codex CLI
```toml
# ~/.codex/config.toml
[mcp_servers.laputa]
url = "http://127.0.0.1:8765/mcp/rpc"
```

### Claude Code
```json
{
  "mcpServers": {
    "laputa": {
      "type": "http",
      "url": "http://127.0.0.1:8765/mcp/rpc"
    }
  }
}
```
```

---

## 验证步骤（每 Wave 后必跑）

### Wave A 后
```bash
cd ~/Desktop/projects/laputa-py
python -m pytest tests/ -q --tb=short \
  --ignore=tests/test_palace_bridge.py \
  --ignore=tests/test_queue.py \
  --ignore=tests/test_session_end.py
```
Expected: 159 passed, 0 failed

### Wave B 后
新增 4 个测试（2 weekly + 2 monthly），总数 163 passed

### Wave C 后
新增 5 个测试（3 soul authority + 2 formation），总数 168 passed

### Wave D 后
新增 4 个测试（2 threshold + 2 compressor），总数 172 passed

### Wave E 后
新增 2 个测试（HTTP transport），总数 174 passed

### Wave F 后
新增 3 个集成测试，总数 177 passed

### 最终全测试
```bash
cd ~/Desktop/projects/laputa-py
python -m pytest tests/ --cov=src/laputa --cov-report=term-missing
```
Expected: ≥ 95% pass, core/ + curator/ 覆盖率 ≥ 85%

---

## 风险与缓解

| 风险 | 级别 | 缓解 |
|---|---|---|
| bridge/ 已删导致 memory_provider 导入失败 | 高 | Wave A Step 1 立即恢复 |
| mcp/ + server/ + queue.py 删除是误操作 | 中 | Wave A Step 5 评估恢复，按需 commit |
| SOUL 权限检查破坏现有 write 调用 | 中 | Wave C Step 4 加 authority 形参，**默认 AGENT**，避免 breaking change |
| HTTP 端口被占用 | 低 | Task E1 用 port=0 自动分配，测试用 actual_port |
| 集成测试依赖外部客户端 | 低 | Wave F 用 server.handle_request 直接调，无需真客户端 |

---

## Pending（完成后）

全部 5 项 ✅ 后：
- 更新 `CHANGELOG.md` 加 "v0.2.0-alpha: SOUL authority + multi-period reports + HTTP transport"
- 更新 `TODOLIST.md` 把 6-26 计划标完成
- 打 tag `v0.2.0-alpha`
- 可选：merge 到 origin/main（push 需用户批准）

---

## Summary

**Total tasks**: 9 (A1, B1, B2, C1, C2, D1, D2, E1, E2, F1) = 10 tasks
**Estimated commits**: 10
**Estimated new tests**: 18
**Final test count**: 154 → ~172

Plan complete and saved. Ready to execute using subagent-driven-development — I'll dispatch a fresh subagent per task with two-stage review (spec compliance then code quality). Shall I proceed?