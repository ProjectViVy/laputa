# ADR-0003 Edge Cleanup — Codex Execution TODO

> 对应 ADR: `0003-edge-cleanup-proposal.md`
> 工作目录: `~/Desktop/garden`
> 环境: `GOSUMDB=off` (本机 sumdb 不可达, 命令前必加)
> 操作系统: Windows, git-bash / MSYS

---

## 总览: 4 步顺序

```
P3 (空文件) → P1 (deprecated binary) → P5 (注释) → P4 (归档)
  3 min          10 min                   15 min       10 min
```

每步完成后 ✅ 在该 todo 后面打勾再进入下一步。

---

## Pre-flight (任何动作前)

```bash
cd ~/Desktop/garden
git status                       # 期望: 干净 + 8 untracked
git log --oneline -1             # 期望: 311c62b 或更新
ls docs/architecture/            # 期望: 0001/0002/0003 + garden-laputa-architecture.html
```

✅ Pre-flight OK

---

## TODO P3 — 删空文件 / 空 yaml (零风险, 3 min)

**P3.1** — 删空 `probe` 文件

```bash
cd ~/Desktop/garden
rm probe
git status  # 应少一行 ?? probe
```

**P3.2** — 删空骨架 `config.example.yaml`

```bash
cd ~/Desktop/garden/garden
grep -rn "config.example.yaml" .       # 先确认没人引用
cat README.md | grep -i "config.example"   # 同上
```

如果**没人引用**, 删:

```bash
rm config/config.example.yaml
git status
```

**P3.3** — 验证 build 仍 ok

```bash
cd ~/Desktop/garden/garden
GOSUMDB=off go build ./...
GOSUMDB=off go test ./internal/...
```

期望: 全过 (此步没改任何 Go 代码, 应立即 pass)

**P3.4** — 不 commit (`probe` 是 untracked, config.example.yaml 也是 untracked, 一并丢掉 — 等 P5 后一次 commit)

✅ P3 done

---

## TODO P1 — 删 deprecated lapua fallback (10 min)

**P1.1** — 删 cmd/laputa 整个目录

```bash
cd ~/Desktop/garden
rm -rf laputa/cmd/laputa
ls laputa/cmd/  # 应只剩 laputa.exe 文件 (或空)
```

**P1.2** — 删两个老 binary

```bash
cd ~/Desktop/garden/laputa
rm laputa.exe laputa.exe~
find . -name '*.exe*'   # 应空
```

**P1.3** — 删加 .gitignore 防 binary 重新进

```bash
cd ~/Desktop/garden/laputa
grep -q '\.exe' .gitignore || echo '*.exe' >> .gitignore
cat .gitignore
```

**P1.4** — 验证

```bash
cd ~/Desktop/garden/laputa
GOSUMDB=off go build ./...
GOSUMDB=off go test ./governance/...
```

期望: governance 测试全部 pass (删 cmd/laputa 不影响 governance 包)

**P1.5** — 复查 deps

```bash
cd ~/Desktop/garden
grep -rn "cmd/laputa" .    # 应无命中 (除 docs/architecture/ 历史文档)
git grep "deprecated" laputa/  # 应无命中
```

✅ P1 done

---

## TODO P5 — 同步 main.go 注释 / 残留 forward-pointer (15 min)

**P5.1** — main.go 头注释

```bash
cd ~/Desktop/garden/garden
head -20 main.go
```

如果头注释里有 "see GARDEN-PLAN.md" / "Phase 1 决定" / "TODO" 之类 forward-pointer, 删 / 简化。

**P5.2** — README 残留 grep

```bash
cd ~/Desktop/garden
grep -rn "cmd/laputa\|config.example" README.md GARDEN-PLAN.md docs/
```

发现一处改一处:

  - "binary fallback" 描述: 删
  - "config.example.yaml" 引用: 删
  - "deprecated" 注释: 删
  - "待定" 章节: 评估, 如果是 P1/P3 范围内的删, 否则保留

**P5.3** — 移除 main.go 注释里 "Phase 0/1/2 决定"

```bash
cd ~/Desktop/garden/garden
grep -n "Phase\|ADR-0001\|superseded" main.go
```

把 `// Phase X 决定` 类注释替换为:

```go
// See docs/architecture/0001-garden-merge.md for context.
```

或直接删 (代码本身可读)。

**P5.4** — 验证最后 build

```bash
cd ~/Desktop/garden/garden
GOSUMDB=off go build ./...
```

✅ P5 done

---

## TODO P4 — 归档 untracked 历史草稿 (10 min)

**P4.1** — 建 archive/historical/ 子目录

```bash
cd ~/Desktop/garden
mkdir -p docs/archive/historical/
```

**P4.2** — 复制 7 个 untracked 文件入 archive/historical/

(untracked 现在 8 个, 含 garden-laputa-architecture.html — 这个是本 ADR 自己造的不归档)

```bash
cd ~/Desktop/garden
git add -N 7_个_文件名  # 先 intent-to-add, 看是否对
# 实际:
mv LAPUTA-plan1-remaining-2026Q3.md                docs/archive/historical/
mv MATSUMOTO-laputa-eino-rhythm-autodream.md       docs/archive/historical/
mv MATSUMOTO-overall-refactor-2026Q3.md            docs/archive/historical/
mv MATSUMOTO-refactor-onepager.md                  docs/archive/historical/
mv SESSION-REPORT-2026-07-08.md                    docs/archive/historical/
mv laputa-galaxyos-structural-borrow.md           docs/archive/historical/
mv laputa-mempalace-batch-extensions-2026Q3.md     docs/archive/historical/
```

**P4.3** — 写 `INDEX.md`

```bash
cd ~/Desktop/garden/docs/archive/historical/
```

写一个 `INDEX.md` (codex 写, 模板如下):

```markdown
# Historical Design Documents

7 篇历史设计 / 计划文档保留归档, 已被后续 ADR / 实施取代。

| 文件 | 日期 | 涵盖主题 | 被取代于 |
|---|---|---|---|
| LAPUTA-plan1-remaining-2026Q3.md | 2026-06-30 | phase 0 前的剩余事项排期 | Phase 0 完成 (2026-07-09, commit 7e16be3) |
| MATSUMOTO-laputa-eino-rhythm-autodream.md | 2026-06-28 | autoDream / rhythm 早期设计 | ADR-0001 + laputa/ARCHITECTURE.md |
| MATSUMOTO-overall-refactor-2026Q3.md | 2026-06-28 | 整体 Q3 重构笔记 | ADR-0001 (Garden 单 exe) |
| MATSUMOTO-refactor-onepager.md | 2026-06-28 | 上文 one-pager | 同上 |
| SESSION-REPORT-2026-07-08.md | 2026-07-08 | 会话决策记录 | 信息存档, 无后续动作 |
| laputa-galaxyos-structural-borrow.md | 2026-06-30 | galaxyOS 借鉴探索 | 无后续, 仅作记录 |
| laputa-mempalace-batch-extensions-2026Q3.md | 2026-07-04 | mentle 批量扩展草案 | ADR-0002 (Phase 5 已实现) |

不删的原因: 历史决策可追溯. 读 garden 顶层 README / ADR-0001/0002 时若需知道为什么这么设计, 这里给完整上下文。
```

✅ P4 done

---

## Final: 总验证 + commit

**F.1** — 4 个 test entrypoint 全过

```bash
cd ~/Desktop/garden/laputa && GOSUMDB=off go test ./governance/...
cd ~/Desktop/garden/mentle && GOSUMDB=off go test ./facade/...
cd ~/Desktop/garden/garden && GOSUMDB=off go test ./internal/...
cd ~/Desktop/garden/garden && GOSUMDB=off go test -tags=e2e ./e2e/...
```

期望: 4 个全部 PASS. e2e 4-5 秒属正常.

**F.2** — garden 二进制重启 + smoke

```bash
cd ~/Desktop/garden/garden
GOSUMDB=off go build -o garden.exe .

# 后台跑
./garden.exe &
GARDEN_PID=$!
sleep 2

curl -s http://127.0.0.1:7373/health
# 期望: {"components":{"garden":"ok","laputa":"ok","mentle":"ok","pipeline":"ok","planner":"degraded"},"status":"degraded"}

curl -s -X POST http://127.0.0.1:7373/v1/memories \
  -H 'Content-Type: application/json' \
  -d '{"key":"section:01-identity","value":"{\"agent\":\"matsumoto\"}"}'
# 期望: {"id":"section:01-identity"}

curl -s http://127.0.0.1:7373/v1/pipelines
# 期望: pipelines 列表非空 (agentic_recall_v1)

curl -s -X POST http://127.0.0.1:7373/v1/context/resolve \
  -H 'Content-Type: application/json' \
  -d '{"intent":"smoke edge cleanup","session_id":"codex-verify"}'
# 期望: 200 + trace_id + evidence (degraded warning 可接受)

# 清理
kill $GARDEN_PID
```

**F.3** — git status 自检

```bash
cd ~/Desktop/garden
git status
```

期望看到:

  - 改动: `laputa/cmd/`, `laputa/*.exe*`, `garden/internal/rag/openai.go`, `garden/main.go`, `garden/config/config.example.yaml` 删除
  - 新增: `docs/archive/historical/*` 内容
  - 残留 untracked: `docs/architecture/garden-laputa-architecture.html` (不是本提案范围, 不处理)

**F.4** — 提交

```bash
cd ~/Desktop/garden
git add laputa/cmd/ laputa/*.exe* garden/internal/rag/ garden/main.go garden/config/ docs/archive/historical/
git commit -m "edge cleanup: P1-P5

- P1 remove deprecated cmd/laputa + 2 laputa.exe binaries
- P3 remove empty config.example.yaml + 0-byte probe
- P4 archive 7 historical design docs into docs/archive/historical/ with INDEX.md
- P5 sync forward-pointer comments in main.go + README

Verification:
- 4 test entrypoints: governance / facade / garden unit / e2e all pass
- /health: garden/laputa/mentle/pipeline ok; planner 状态取决于外部 planner 配置
- POST /v1/context/resolve returns real evidence

See docs/architecture/0003-edge-cleanup-proposal.md"
```

---

## 失败模式 + 怎么办

| 失败 | 怎么办 |
|---|---|
| e2e 失败 | 看 `garden/e2e/e2e_test.go` — 它需要临时端口 + 临时目录; 检查 firewall / 端口冲突 |
| `go build` 提示 unused import | 删对应 import 即可 |
| `GOSUMDB` 报错 | 确认命令前 `GOSUMDB=off ` 前缀加了 |
| README 改完 build 失败 | README 不参与 build, 不可能; 看 go.mod |
| `find . -name '*.exe*'` 仍返回 | 检查 laputa/.gitignore — 加 `*.exe` 或 `*.exe~` |

---

## 不在这次任务

  - 14 section 中 10–14 的 owner / schema 拍板 (提案 ADR-0004 / 0005, 松本单独来)
  - 松本提的 7 件事 (auto-extract ingest pipeline 等) (提案 ADR-0004)
  - Rust 路径全部不动
  - mempalace-py 不动 (已物理隔离)

---

*TODOs author: 松本*
*生成: 2026-07-15*
*对应 ADR: 0003*
