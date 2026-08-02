# Mempalace 接入三件套实施计划

> **For Hermes:** 本计划含三条独立工作流（去重前置 / session_end hook / wake-up 集成），按"并行展开、各自验证、最终汇总"执行。每条都是 2-5 分钟可独立完成的 bite-sized 任务。
>
> 三件事你可以择一先做，但本计划默认三条**互不阻塞**：
> - **Track A** 只动 `bulk_write.py`（去重前置）— 影响写入路径
> - **Track B** 是 Hermes 配置文件改动（session_end hook）— 影响外部触发
> - **Track C** 新增 `wake.py` 客户端 + Laputa 调用位（wake-up 集成）— 影响读取路径
>
> 三件同时跑不冲突，因为 Track A/B 都是只追加（不删/不改既有代码），Track C 是新文件。
> 但最终汇总验证必须全 3 件一起跑（写入→去重→hook 自动触发→wake 拉起→人格/identity 召回）。

**Goal:** 把 `laputa-mempalace-bridge` 升级为生产级写入路径——内容去重、会话结束时自动持久化、跨 session 唤醒时按需注入身份记忆。

**Architecture:**
- 写入路径：`bulk_write.py` → `MempalaceClient._request("mempalace_batch_store")` 已稳，新增"先去重再批量"前置闸门
- 触发路径：Hermes profile `hooks/on-session-end` 调 `bulk_write.py --auto` 自动收口
- 读取路径：新增 `wake.py` 客户端，复用 `MempalaceClient` 连 mempalace → 拉 identity/laputa_self 等高重要性人格 → 输出 LAPUTA.md 可直接消费的 render

**Tech Stack:** Python 3.11 stdlib + 已存在的 `mempalace_client.py` + Hermes profile hooks (`~/.hermes/profile/default/hooks/`) + mempalace v2 MCP `mempalace_check_duplicate` (已注册于 `cmd/server/main.go:292`)

---

## 事实基线（写计划前 read-only 验证）

| 断言 | 真伪 | 证据 |
|---|---|---|
| mempalace v2 已注册 `mempalace_check_duplicate` tool | ✅ 真 | `cmd/server/main.go:292` |
| mempalace v2 **无** `mempalace_wake` tool | ✅ 真（修正上次幻觉） | `internal/layers/` 只有 `layers.go`/`layers_test.go`，搜 `[Ww]ake` 0 命中 |
| `laputa-mempalace-bridge/` 项目已存 5 个 Python 模块 | ✅ 真 | `bulk_write.py`/`diff_engine.py`/`laputa_client.py`/`mempalace_client.py`/`sync.py` |
| Hermes profile `~/.hermes/` 含 `hooks/` 目录 | ✅ 真（但无 default profile 子目录） | `evolver-hooks`/`skills`/`plans`/`scripts` 直接在 `~/.hermes/` 下 |
| mempalace.exe 路径 | ✅ 固定 | `C:\Users\Administrator\Desktop\projects\mempalace-go-redis-v2\mempalace.exe` |

---

## Track A — bulk_write 去重前置

### A.1: 编写 `_check_duplicates` 单元测试

**目标:** 函数接受 items list，调 `mempalace_check_duplicate`，返回过滤后 list。

**文件:**
- 新增: `tests/test_bulk_write_dedup.py`

**Step 1: 写失败测试**

```python
def test_filter_already_existed():
    items = [
        {"content": "[importance=3] 新内容 A", "wing": "knowledge", "room": "r", "source": "bulk"},
        {"content": "[importance=3] 已存在 B", "wing": "knowledge", "room": "r", "source": "bulk"},
    ]
    fake_resp = {"result": {"content": [{"type": "text", "text": '{"exists": ["B hash"]}'}]}}
    client = FakeMempalaceClient(fake_resp)
    out = filter_duplicates(items, client)
    assert len(out) == 1
    assert "新内容 A" in out[0]["content"]


class FakeMempalaceClient:
    def __init__(self, dedup_response): self._resp = dedup_response
    def check_duplicate(self, items): return self._resp
```

**Step 2: 跑测试 — 应失败（NotImplementedError / ImportError）**

`cd /c/Users/Administrator/Desktop/projects/laputa-mempalace-bridge && python -m pytest tests/test_bulk_write_dedup.py::test_filter_already_existed -v`
期望: `FAILED — cannot import filter_duplicates`

**Step 3: commit scaffold**

```bash
git add tests/test_bulk_write_dedup.py
git commit -m "test(bulk_write): TDD scaffold for filter_duplicates"
```

---

### A.2: 在 `mempalace_client.py` 加 `check_duplicate` 方法

**文件:**
- 修改: `mempalace_client.py:148` 后插入

```python
def check_duplicate(self, items: list[dict[str, Any]]) -> dict[str, Any]:
    """Call mempalace_check_duplicate MCP tool."""
    if not self._initialized:
        self.initialize()
    response = self._request(
        "tools/call",
        {"name": "mempalace_check_duplicate",
         "arguments": {"items": [{"content": i["content"]} for i in items]}},
    )
    if response.get("error"):
        raise RuntimeError(f"mempalace_check_duplicate failed: {response['error']}")
    return response.get("result", {})
```

**验证:** `python -m pytest tests/test_bulk_write_dedup.py::test_filter_already_existed -v` 期望 PASS

**commit:** `git commit -m "feat(mempalace_client): add check_duplicate wrapper"`

---

### A.3: 实现 `filter_duplicates()` 纯函数

**文件:**
- 修改: `bulk_write.py:69` 后插入（与 build_batch_items 同区）

```python
def filter_duplicates(items: list[dict], dedup_result: dict) -> list[dict]:
    """Drop items whose content hash is reported as already-existed.

    dedup_result shape (after text-decode):
        {"exists": ["hash1", "hash2", ...]}
    Each input dict MAY carry an "_hash" field; if absent, we skip filtering
    (callers may precompute hash; we don't re-hash to keep the function pure).
    """
    content = dedup_result.get("content", [])
    if not content or not isinstance(content, list):
        return items
    text_blocks = [c.get("text", "") for c in content if c.get("type") == "text"]
    if not text_blocks:
        return items
    try:
        exists_set = set(json.loads(text_blocks[0]).get("exists", []))
    except (json.JSONDecodeError, AttributeError):
        return items
    kept = [it for it in items if it.get("_hash") not in exists_set]
    if len(kept) < len(items):
        logger.info("Dedup: dropped %d / %d", len(items) - len(kept), len(items))
    return kept
```

**验证:** `python -m pytest tests/ -v` 全绿

**commit:** `git commit -m "feat(bulk_write): add dedup filter (purity-preserving)"`

---

### A.4: 在 `main()` 接入去重前后流程

**文件:**
- 修改: `bulk_write.py:111` 前

```python
        # Compute content hash for each item (deterministic, no network yet)
        import hashlib
        for it in items:
            it["_hash"] = hashlib.sha256(it["content"].encode("utf-8")).hexdigest()[:16]

        logger.info("Checking duplicates for %d items...", len(items))
        dedup_resp = client.check_duplicate(items)
        items = filter_duplicates(items, dedup_resp)

        if not items:
            logger.info("All items already exist; nothing to write")
            return 0

        logger.info("Calling mempalace_batch_store with %d items...", len(items))
```

**验证:**
1. `echo '[{"content":"[importance=3] 同一个内容测试","wing":"knowledge","room":"test_dedup","source":"manual"}]' > _dedup_test.json`
2. 第一次跑: `python bulk_write.py --file _dedup_test.json` 期望 "Successfully stored 1 drawers"
3. 第二次跑: `python bulk_write.py --file _dedup_test.json` 期望 "Dedup: dropped 1 / 1" + "All items already exist; nothing to write"

**commit:** `git commit -m "feat(bulk_write): pre-write duplicate check"`

---

## Track B — session_end hook

### B.1: 设计 hook 入口脚本

**目标:** 会话结束时调 `bulk_write.py --auto`，但当前没有专门的 Hermes session-end hook API（待核实）；用 shutdown 时机的 cron 自驱替代。

**文件:**
- 新增: `~/.hermes/scripts/laputa-session-end.py`（hermes 自带 `scripts/` 目录）
- 新增: `laputa-mempalace-bridge/hooks/session_end.json`（hook 元数据）

**Step 1: 写 `laputa-session-end.py`**

```python
#!/usr/bin/env python3
"""Trigger bulk_write when a Hermes session ends.

Called manually or by a wrap script at /new or Ctrl-D. Writes a stdout
JSON marker so the wrapping Hermes knows success/failure.
"""
import json
import subprocess
import sys
from pathlib import Path

ROOT = Path(r"C:\Users\Administrator\Desktop\projects\laputa-mempalace-bridge")
BULK = ROOT / "bulk_write.py"


def main() -> int:
    proc = subprocess.run(
        [sys.executable, str(BULK), "--auto"],
        cwd=str(ROOT),
        capture_output=True,
        text=True,
        timeout=180,
    )
    result = {
        "exit_code": proc.returncode,
        "stdout_tail": proc.stdout.splitlines()[-5:] if proc.stdout else [],
        "stderr_tail": proc.stderr.splitlines()[-5:] if proc.stderr else [],
    }
    print(json.dumps(result, ensure_ascii=False))
    return proc.returncode


if __name__ == "__main__":
    sys.exit(main())
```

**Step 2: 在 `bulk_write.py` 加 `--auto` 模式**

修改: `bulk_write.py:90` 后插入

```python
    parser.add_argument(
        "--auto",
        action="store_true",
        help="Auto-mode: load _session_pending.json if exists, else noop",
    )
    args = parser.parse_args()
    if args.auto:
        args.file = ROOT / "_session_pending.json"
        if not args.file.exists():
            logger.info("No pending items; nothing to write")
            return 0
```

**Step 3: 真测**

1. `echo '[]' > _session_pending.json` + `python bulk_write.py --auto` 期望 `"Loaded 0 payloads"`
2. `echo '[{"content":"[importance=3] hook-test-XYZ","wing":"knowledge","room":"hook_test","source":"hook"}]' > _session_pending.json` + `python laputa-session-end.py` 期望 exit 0 + mempalace 出现 1 条

**commit:** `git commit -m "feat(hook): add session_end → bulk_write auto-trigger"`

---

### B.2: 接入 Hermes profile hooks 目录

**核实:** `ls ~/.hermes/hooks/` 看现有 hook schema。若无 session_end 钩子约定，**承认局限**：
- 当前 Hermes profile 无内置 session_end 触发器
- 替代方案：**手动 ctrl-D 前跑一次** `python laputa-session-end.py`，或编 `~/.bashrc` 末尾 alias `exit-session='python ~/Desktop/projects/laputa-mempalace-bridge/laputa-session-end.py'`

**Step 1:** `ls ~/.hermes/hooks/ 2>&1 | head -20` 看现有 schema
**Step 2:** 若有 on-session-end 范例，照抄改造；若无，保留脚本 + 写 README 说明手动触发模式

**commit:** `git commit -m "docs: session_end manual trigger instructions"`（仅 docs）

---

## Track C — wake-up 集成

> ⚠️ **修正上次幻觉**：mempalace v2 没有 `mempalace_wake` tool，必须从 0 写客户端侧 `wake.py`，拉数据 + 渲染为 LAPUTA.md 子节。

### C.1: 写 `wake.py` 的 RENDER 单元测试

**文件:**
- 新增: `tests/test_wake.py`

**Step 1: 失败测试**

```python
def test_render_identity_section():
    identity_items = [
        {"content": "[importance=5] 松本：INTP，安静自信，8 个原型死了最后一个", "wing": "identity",
         "room": "laputa_self", "source": "manual", "_meta": {"importance": 5}},
        {"content": "[importance=3] 用户偏好：中文输出，markdown 表格", "wing": "identity",
         "room": "preferences", "source": "manual", "_meta": {"importance": 3}},
    ]
    out = render_wake_section(identity_items)
    assert "# Identity" in out
    assert "松本" in out
    assert "INTP" in out


def test_render_token_budget():
    items = [{"content": f"line {i}", "wing": "knowledge", "room": "r",
              "source": "x", "_meta": {"importance": 1}} for i in range(100)]
    out = render_wake_section(items, max_tokens=100)
    # Should drop low-importance items first
    assert len(out) < 10_000  # safely under 100-token render
```

**Step 2:** `cd ... && python -m pytest tests/test_wake.py -v` 期望 FAIL

**commit:** `git commit -m "test(wake): TDD scaffold for render_wake_section"`

---

### C.2: 在 `mempalace_client.py` 加 `search` 方法

**文件:**
- 修改: `mempalace_client.py:179` 前

```python
    def search(
        self,
        query: str,
        wing: str | None = None,
        room: str | None = None,
        limit: int = 10,
    ) -> list[dict[str, Any]]:
        """Search mempalace — returns list of items matching query/filter."""
        if not self._initialized:
            self.initialize()
        arguments: dict[str, Any] = {"query": query, "limit": limit}
        if wing:
            arguments["wing"] = wing
        if room:
            arguments["room"] = room
        response = self._request(
            "tools/call",
            {"name": "mempalace_search", "arguments": arguments},
        )
        if response.get("error"):
            raise RuntimeError(f"mempalace_search failed: {response['error']}")
        # parse text-content → JSON list
        content = response.get("result", {}).get("content", [])
        for c in content:
            if c.get("type") == "text":
                try:
                    return json.loads(c["text"])
                except json.JSONDecodeError:
                    continue
        return []
```

**验证:** `python -m pytest tests/test_wake.py -v` 仍 FAIL（render_wake_section 未实现）— 这是预期的

**commit:** `git commit -m "feat(mempalace_client): add search wrapper"`

---

### C.3: 实现 `wake.py` 客户端

**文件:**
- 新增: `wake.py`

```python
#!/usr/bin/env python3
"""Wake-up integration: pull identity + decisions from mempalace, render as
markdown ready to inject into LAPUTA.md or system prompt.

Usage:
    python wake.py                      # render to stdout
    python wake.py --output wake.md     # write to file
    python wake.py --wing identity      # only one wing
"""
from __future__ import annotations

import argparse
import json
import logging
import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent
sys.path.insert(0, str(ROOT))

from mempalace_client import MempalaceClient  # noqa: E402

logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(message)s")
logger = logging.getLogger(__name__)


_IMPORTANCE_RE = re.compile(r"\[importance=(\d+)\]")


def parse_importance(content: str) -> int:
    m = _IMPORTANCE_RE.search(content)
    return int(m.group(1)) if m else 3


def strip_importance_header(content: str) -> str:
    return _IMPORTANCE_RE.sub("", content, count=1).strip()


def render_wake_section(
    items: list[dict],
    max_tokens: int = 2000,
    section_title: str | None = None,
) -> str:
    """Render items as markdown; respect max_tokens budget via importance-ordered drop."""
    # Decorate with importance
    decorated = [
        {**it, "_meta": {**it.get("_meta", {}),
                         "importance": parse_importance(it.get("content", ""))}}
        for it in items
    ]
    # Sort by importance desc, then by content length asc
    decorated.sort(key=lambda x: (-x["_meta"]["importance"], len(x.get("content", ""))))

    out_lines = []
    if section_title:
        out_lines.append(f"# {section_title}")
        out_lines.append("")
    used = 0
    for it in decorated:
        text = strip_importance_header(it["content"])
        # ~4 chars per token heuristic
        cost = len(text) // 4 + 4
        if used + cost > max_tokens:
            continue
        out_lines.append(f"- {text}")
        used += cost
        if used >= max_tokens * 0.95:
            break
    return "\n".join(out_lines)


def fetch_wake_data(
    client: MempalaceClient,
    wings: list[str],
    top_k: int = 20,
) -> dict[str, list[dict]]:
    """Pull latest high-importance items from each wing."""
    out: dict[str, list[dict]] = {}
    for wing in wings:
        try:
            items = client.search(query="", wing=wing, limit=top_k)
        except RuntimeError as exc:
            logger.warning("Search failed for wing=%s: %s", wing, exc)
            out[wing] = []
            continue
        out[wing] = items
    return out


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--wing", action="append",
                        default=["identity", "knowledge"],
                        help="Wing to fetch (repeatable)")
    parser.add_argument("--output", type=Path, help="Write to file (default: stdout)")
    parser.add_argument("--palace", type=Path,
                        default=ROOT / ".palace",
                        help="mempalace palace path")
    parser.add_argument("--exe", type=str,
                        default=r"C:\Users\Administrator\Desktop\projects\mempalace-go-redis-v2\mempalace.exe")
    parser.add_argument("--max-tokens", type=int, default=2000,
                        help="Per-section token budget (~4 chars/token)")
    args = parser.parse_args()

    args.palace.mkdir(parents=True, exist_ok=True)
    (args.palace / "wal").mkdir(parents=True, exist_ok=True)

    client = MempalaceClient(str(args.exe), str(args.palace), timeout_sec=60.0)
    try:
        client.initialize()
        data = fetch_wake_data(client, args.wing)
        sections = []
        for wing, items in data.items():
            section = render_wake_section(
                items, max_tokens=args.max_tokens, section_title=wing.title(),
            )
            if section.strip():
                sections.append(section)
        rendered = "\n\n".join(sections)
        if args.output:
            args.output.write_text(rendered, encoding="utf-8")
            logger.info("Written %d chars to %s", len(rendered), args.output)
        else:
            print(rendered)
        return 0
    finally:
        client.close()


if __name__ == "__main__":
    sys.exit(main())
```

**验证:**
1. `python -m pytest tests/test_wake.py -v` 全绿
2. `python wake.py --output .tmp_wake_test.md` 然后 `cat .tmp_wake_test.md | head -30` 看是否含 identity + knowledge 段

**commit:** `git commit -m "feat(wake): render identity+knowledge from mempalace"`

---

### C.4: 验证 E2E 与 LAPUTA.md 兼容

**Step 1:** `python wake.py --wing identity --output .tmp_identity_test.md` + `cat .tmp_identity_test.md`

期望看到 "松本" 人格描述（上次批量写入的 5 条人格 traits 已落仓）。

**Step 2:** 把产出 append 进 `DIVA/docs/laputa-py/baseline/LAPUTA.md` 的对应 section，验证渲染格式兼容（行数、章节标题层级）

**Step 3:** 删 `.tmp_*.md` 测试文件

**commit:** `git commit -m "feat(wake): E2E smoke test passes"`

---

## 汇总验证（Track A+B+C 全完成后）

**Smoke test 脚本** — 新增 `scripts/verify_full_flow.sh`:

```bash
#!/usr/bin/env bash
set -e
cd /c/Users/Administrator/Desktop/projects/laputa-mempalace-bridge

echo "===[1/4] 去重验证==="
echo '[{"content":"[importance=3] summary-flow-test-xyz","wing":"knowledge","room":"flow","source":"verify"}]' > _v.json
python bulk_write.py --file _v.json   # 第 1 次: stored
python bulk_write.py --file _v.json   # 第 2 次: dropped 1/1

echo "===[2/4] session_end 触发==="
cp _v.json _session_pending.json
python laputa-session-end.py  # bulk_write --auto

echo "===[3/4] wake-up 渲染==="
python wake.py --wing identity --output .v_wake.md
head -20 .v_wake.md

echo "===[4/4] 检索召回==="
(printf '%s\n%s\n%s\n' \
  '{"jsonrpc":"2.0","method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"v","version":"1"}},"id":1}' \
  '{"jsonrpc":"2.0","method":"notifications/initialized","params":{}}' \
  '{"jsonrpc":"2.0","method":"tools/call","params":{"name":"mempalace_search","arguments":{"query":"松本","wing":"identity"}},"id":2}' \
  | timeout 30 /c/Users/Administrator/Desktop/projects/mempalace-go-redis-v2/mempalace.exe server --palace .palace) | head -c 500

rm -f _v.json _session_pending.json .v_wake.md
echo "===ALL OK==="
```

**期望输出：**
- `[1/4]` 后看到 "Successfully stored 1 drawers"，再后看到 "Dedup: dropped 1 / 1"
- `[2/4]` exit 0 + JSON `{"exit_code": 0}`
- `[3/4]` 含 "松本" 字样的 markdown
- `[4/4]` 含 `松本` 字样的 JSON 召回结果

---

## 风险与开放问题

| 风险 | 影响 | 缓解 |
|---|---|---|
| mempalace v2 `check_duplicate` 返回格式与 doc 不一致 | Track A 全路阻塞 | **Step A.2 后立即跑一次手动 check_duplicate**，把返回格式记下；如 schema 不同改 filter_duplicates 解析路径 |
| Hermes profile 无内置 session_end hook | Track B 降级为手动 | 写 alias 文档 + 提议 future：本地 patch Hermes hook dispatch |
| `wake.py` 用空 query 召回可能返回大量低质量结果 | Track C 召回质量差 | importance 阈值过滤；当前设 `parse_importance` 默认 3，可改 `--min-importance` flag |
| 三条 track 都修改 `mempalace_client.py` | 合并冲突 | Track A 和 C 都加新方法不重叠；commit 前 rebase |

---

## 预计工时

| Track | 任务数 | 估计 |
|---|---|---|
| A — 去重前置 | 4 | 1-2 h |
| B — session_end hook | 2 | 0.5-1 h（受 hook schema 制约） |
| C — wake-up | 4 | 2-3 h |
| 汇总验证 | 1 | 0.5 h |
| **总计** | 11 | **4-6.5 h** |

---

## 验收清单

- [ ] `python -m pytest tests/ -v` 全绿
- [ ] `python bulk_write.py --file <重复内容>` 两次第二次 dropping 100%
- [ ] `python bulk_write.py --auto` 空场景退出码 0
- [ ] `python laputa-session-end.py` exit 0 + stdout JSON
- [ ] `python wake.py --wing identity` 渲染结果含 `松本` + INTP/称谓
- [ ] `scripts/verify_full_flow.sh` 4 步全 OK
- [ ] git log 三条 track 各自有独立 commit
