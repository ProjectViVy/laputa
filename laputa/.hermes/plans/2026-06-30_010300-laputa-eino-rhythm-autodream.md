# Laputa + Eino 最小节律（Rhythm）实现计划

> **For Hermes:** Use subagent-driven-development skill to implement this plan task-by-task.

**Goal:** 在 Go Laputa 中集成 CloudWeGo Eino 框架，实现一个最小可运行的节律引擎（Rhythm Engine），能够按 daily/weekly/monthly 周期读取 Laputa section，调用 LLM 生成报告，并写回对应 section。

**Architecture:** 新增 `rhythm` 包，内部包含 `RhythmEngine`（调度/触发）、`ReportGenerator`（Eino ChatModel 调用）、`PromptBuilder`（从 Laputa section 组装 prompt）。不引入 daemon，引擎由外部 cronjob 或 CLI 触发。AutoDream 只保留"触发链"概念，本计划不实现完整 AutoDream 状态机。

**Tech Stack:** Go 1.23+, CloudWeGo Eino, Laputa FileStore, OpenAI-compatible LLM API.

---

## Current Context

- Go Laputa 已实现 14 section FileStore + Engine + Snapshot API（`laputa.go`）。
- 无 Eino 依赖，无 LLM 调用，无节律逻辑。
- 设计依据：`laputa-py/baseline/LAPUTA.md` v0.0.6 final、`laputa-py/baseline/MENTLE.md` v0.2。

---

## Proposed Approach

1. 添加 Eino 依赖到 `go.mod`。
2. 创建 `internal/rhythm/` 包：
   - `rhythm.go`：节律引擎，支持 daily/weekly/monthly 触发。
   - `generator.go`：Eino ChatModel 封装，生成报告。
   - `prompt.go`：从 Laputa snapshot 构建 prompt。
   - `types.go`：RhythmKind、ReportResult 等类型。
3. 创建 `cmd/laputa-rhythm/` 或扩展 `laputa` CLI：提供 `rhythm run --kind daily` 命令。
4. 写测试：mock LLM，验证 prompt 组装和 section 写入。
5. 更新 `ARCHITECTURE.md` 和 `README.md`。

---

## Step-by-Step Plan

### Task 1: Add Eino dependency

**Objective:** 让项目能够导入 `github.com/cloudwego/eino`。

**Files:**
- Modify: `go.mod`
- Modify: `go.sum`（自动生成）

**Step 1: Add require**

```go
require (
    github.com/cloudwego/eino v0.3.0
    github.com/cloudwego/eino-ext/libs/acl/openai v0.0.0
)
```

> 注：版本以实际可用为准，执行时若不存在则替换为最新 tag。

**Step 2: Tidy modules**

Run:
```bash
export PATH="/c/Program Files/go/bin:$PATH"
export GOPROXY=https://proxy.golang.org,direct
export HTTPS_PROXY=socks5h://127.0.0.1:7892
cd /c/Users/Administrator/Desktop/projects/laputa
go mod tidy
```

Expected: `go.mod` 和 `go.sum` 更新，无错误。

**Step 3: Verify import**

Create temporary file `cmd/eino_smoke/main.go`:
```go
package main

import (
    _ "github.com/cloudwego/eino/schema"
)

func main() {}
```

Run:
```bash
go build ./cmd/eino_smoke
```

Expected: build succeeds.

**Step 4: Commit**

```bash
git add go.mod go.sum cmd/eino_smoke/main.go
git commit -m "deps: add cloudwego/eino dependency"
```

---

### Task 2: Define rhythm types

**Objective:** 定义节律引擎的核心类型。

**Files:**
- Create: `internal/rhythm/types.go`

**Step 1: Write code**

```go
package rhythm

import "time"

// RhythmKind represents the reporting cadence.
type RhythmKind string

const (
    RhythmDaily   RhythmKind = "daily"
    RhythmWeekly  RhythmKind = "weekly"
    RhythmMonthly RhythmKind = "monthly"
)

// ReportResult is what the LLM produces for a rhythm run.
type ReportResult struct {
    Title       string `json:"title"`
    Summary     string `json:"summary"`
    Highlights  []string `json:"highlights"`
    OpenQuestions []string `json:"open_questions,omitempty"`
    GeneratedAt time.Time `json:"generated_at"`
}

// Config holds rhythm engine configuration.
type Config struct {
    BaseURL string
    APIKey  string
    Model   string
}
```

**Step 2: Verify build**

Run:
```bash
go build ./internal/rhythm
```

Expected: PASS.

**Step 3: Commit**

```bash
git add internal/rhythm/types.go
git commit -m "feat(rhythm): define RhythmKind, ReportResult, Config"
```

---

### Task 3: Build prompt from Laputa snapshot

**Objective:** 从 Laputa 读取 identity/memory_md/history_md 并组装成 LLM prompt。

**Files:**
- Create: `internal/rhythm/prompt.go`
- Modify: `internal/rhythm/types.go`（如需要）

**Step 1: Write failing test**

Create: `internal/rhythm/prompt_test.go`

```go
package rhythm

import (
    "strings"
    "testing"
)

func TestBuildPrompt(t *testing.T) {
    snapshot := map[string]any{
        "identity": map[string]any{
            "role": "assistant",
        },
        "memory_md": map[string]any{
            "summary": "user prefers concise output",
        },
        "history_md": map[string]any{
            "timeline": []map[string]any{
                {"event": "started project"},
            },
        },
    }

    prompt := BuildPrompt(RhythmDaily, snapshot)
    if !strings.Contains(prompt, "daily") {
        t.Errorf("prompt should mention rhythm kind")
    }
    if !strings.Contains(prompt, "concise output") {
        t.Errorf("prompt should include memory_md summary")
    }
}
```

Run:
```bash
go test ./internal/rhythm -run TestBuildPrompt -v
```

Expected: FAIL — `BuildPrompt undefined`。

**Step 2: Implement BuildPrompt**

Create: `internal/rhythm/prompt.go`

```go
package rhythm

import (
    "encoding/json"
    "fmt"
)

// BuildPrompt constructs an LLM prompt from a Laputa snapshot.
func BuildPrompt(kind RhythmKind, snapshot map[string]any) string {
    sectionsJSON, _ := json.Marshal(snapshot)
    return fmt.Sprintf(`You are an autonomous rhythm reporter for an AI agent.

Cadence: %s

Laputa snapshot:
%s

Generate a structured report with: title, summary, highlights, open_questions.
Respond in JSON matching ReportResult.`, kind, string(sectionsJSON))
}
```

**Step 3: Verify test passes**

Run:
```bash
go test ./internal/rhythm -run TestBuildPrompt -v
```

Expected: PASS.

**Step 4: Commit**

```bash
git add internal/rhythm/prompt.go internal/rhythm/prompt_test.go
git commit -m "feat(rhythm): build LLM prompt from Laputa snapshot"
```

---

### Task 4: Implement Eino report generator

**Objective:** 用 Eino ChatModel 调用 LLM 并解析 JSON 报告。

**Files:**
- Create: `internal/rhythm/generator.go`
- Create: `internal/rhythm/generator_test.go`

**Step 1: Write failing test**

```go
package rhythm

import (
    "context"
    "testing"
)

func TestGeneratorGenerate_Mock(t *testing.T) {
    gen := NewMockGenerator()
    result, err := gen.Generate(context.Background(), RhythmDaily, "test prompt")
    if err != nil {
        t.Fatalf("generate: %v", err)
    }
    if result.Title == "" {
        t.Error("expected non-empty title")
    }
}
```

Run:
```bash
go test ./internal/rhythm -run TestGeneratorGenerate_Mock -v
```

Expected: FAIL — `NewMockGenerator undefined`。

**Step 2: Implement mock + real generator**

Create: `internal/rhythm/generator.go`

```go
package rhythm

import (
    "context"
    "encoding/json"
    "fmt"
    "time"

    "github.com/cloudwego/eino/schema"
    "github.com/cloudwego/eino/components/model"
)

// Generator generates rhythm reports from prompts.
type Generator interface {
    Generate(ctx context.Context, kind RhythmKind, prompt string) (*ReportResult, error)
}

// EinoGenerator uses a ChatModel to generate reports.
type EinoGenerator struct {
    chatModel model.ChatModel
}

// NewEinoGenerator creates a generator from an Eino ChatModel.
func NewEinoGenerator(cm model.ChatModel) *EinoGenerator {
    return &EinoGenerator{chatModel: cm}
}

// Generate calls the LLM and parses the JSON response.
func (g *EinoGenerator) Generate(ctx context.Context, kind RhythmKind, prompt string) (*ReportResult, error) {
    resp, err := g.chatModel.Generate(ctx, []*schema.Message{
        {Role: schema.User, Content: prompt},
    })
    if err != nil {
        return nil, fmt.Errorf("llm generate: %w", err)
    }

    var result ReportResult
    if err := json.Unmarshal([]byte(resp.Content), &result); err != nil {
        return nil, fmt.Errorf("parse report: %w", err)
    }
    result.GeneratedAt = time.Now().UTC()
    return &result, nil
}

// MockGenerator is a deterministic generator for tests.
type MockGenerator struct {
    Result *ReportResult
}

func NewMockGenerator() *MockGenerator {
    return &MockGenerator{
        Result: &ReportResult{
            Title:     "Daily Rhythm Report",
            Summary:   "All systems nominal.",
            Highlights: []string{"completed task A"},
        },
    }
}

func (m *MockGenerator) Generate(ctx context.Context, kind RhythmKind, prompt string) (*ReportResult, error) {
    r := *m.Result
    r.GeneratedAt = time.Now().UTC()
    return &r, nil
}
```

> 注：Eino API 可能不同，执行时根据实际 eino 版本调整 `schema.Message` 和 `model.ChatModel` 用法。

**Step 3: Verify test passes**

Run:
```bash
go test ./internal/rhythm -run TestGeneratorGenerate_Mock -v
```

Expected: PASS.

**Step 4: Commit**

```bash
git add internal/rhythm/generator.go internal/rhythm/generator_test.go
git commit -m "feat(rhythm): add Eino report generator with mock"
```

---

### Task 5: Implement RhythmEngine

**Objective:** 把 Laputa snapshot → prompt → LLM → write report 串起来。

**Files:**
- Create: `internal/rhythm/rhythm.go`
- Create: `internal/rhythm/rhythm_test.go`

**Step 1: Write failing test**

```go
package rhythm

import (
    "context"
    "testing"

    "github.com/dashimaki/laputa"
)

func TestRhythmEngineRun_Daily(t *testing.T) {
    ctx := context.Background()
    store, _ := laputa.NewFileStore(t.TempDir())
    engine := laputa.NewEngine(store)
    _ = engine.Initialize(ctx)

    rhythm := NewEngine(engine, NewMockGenerator(), Config{})
    err := rhythm.Run(ctx, RhythmDaily)
    if err != nil {
        t.Fatalf("run: %v", err)
    }

    daily, _ := engine.GetSection(ctx, laputa.SectionDaily)
    reports, ok := daily["reports"].([]any)
    if !ok || len(reports) == 0 {
        t.Errorf("expected daily report to be written, got %v", daily)
    }
}
```

Run:
```bash
go test ./internal/rhythm -run TestRhythmEngineRun_Daily -v
```

Expected: FAIL — `NewEngine undefined`（rhythm 包内）。

**Step 2: Implement RhythmEngine**

Create: `internal/rhythm/rhythm.go`

```go
package rhythm

import (
    "context"
    "fmt"
    "time"

    "github.com/dashimaki/laputa"
)

// Engine runs rhythm reports against Laputa.
type Engine struct {
    laputa    *laputa.Engine
    generator Generator
    config    Config
}

// NewEngine creates a rhythm engine.
func NewEngine(laputaEngine *laputa.Engine, generator Generator, config Config) *Engine {
    return &Engine{
        laputa:    laputaEngine,
        generator: generator,
        config:    config,
    }
}

// Run executes one rhythm cycle and writes the report to Laputa.
func (e *Engine) Run(ctx context.Context, kind RhythmKind) error {
    snapshot, err := e.laputa.Snapshot(ctx)
    if err != nil {
        return fmt.Errorf("snapshot: %w", err)
    }

    prompt := BuildPrompt(kind, snapshot)
    report, err := e.generator.Generate(ctx, kind, prompt)
    if err != nil {
        return fmt.Errorf("generate: %w", err)
    }

    var target laputa.SectionName
    switch kind {
    case RhythmDaily:
        target = laputa.SectionDaily
    case RhythmWeekly:
        target = laputa.SectionWeekly
    case RhythmMonthly:
        target = laputa.SectionMonthly
    default:
        return fmt.Errorf("unknown rhythm kind: %s", kind)
    }

    section, err := e.laputa.GetSection(ctx, target)
    if err != nil {
        return fmt.Errorf("read target section: %w", err)
    }

    reports, _ := section["reports"].([]any)
    entry := map[string]any{
        "title":       report.Title,
        "summary":     report.Summary,
        "highlights":  report.Highlights,
        "open_questions": report.OpenQuestions,
        "generated_at": report.GeneratedAt.Format(time.RFC3339),
    }
    section["reports"] = append(reports, entry)

    if err := e.laputa.SetSection(ctx, target, section); err != nil {
        return fmt.Errorf("write target section: %w", err)
    }
    return nil
}
```

**Step 3: Verify test passes**

Run:
```bash
go test ./internal/rhythm -run TestRhythmEngineRun_Daily -v
```

Expected: PASS.

**Step 4: Commit**

```bash
git add internal/rhythm/rhythm.go internal/rhythm/rhythm_test.go
git commit -m "feat(rhythm): wire snapshot -> prompt -> llm -> laputa section"
```

---

### Task 6: Add CLI command

**Objective:** 提供 `laputa rhythm daily` 命令触发节律运行。

**Files:**
- Create: `cmd/laputa/main.go`
- Modify: `go.mod`（如有新依赖）

**Step 1: Create CLI**

```go
package main

import (
    "context"
    "flag"
    "fmt"
    "os"

    "github.com/dashimaki/laputa"
    "github.com/dashimaki/laputa/internal/rhythm"
)

func main() {
    var (
        dir     = flag.String("dir", ".laputa", "laputa data directory")
        kind    = flag.String("kind", "daily", "rhythm kind: daily|weekly|monthly")
        baseURL = flag.String("base-url", "https://api.openai.com/v1", "LLM base URL")
        apiKey  = flag.String("api-key", os.Getenv("OPENAI_API_KEY"), "LLM API key")
        model   = flag.String("model", "gpt-4o-mini", "LLM model")
    )
    flag.Parse()

    ctx := context.Background()
    store, err := laputa.NewFileStore(*dir)
    if err != nil {
        fmt.Fprintf(os.Stderr, "store: %v\n", err)
        os.Exit(1)
    }
    engine := laputa.NewEngine(store)
    if err := engine.Initialize(ctx); err != nil {
        fmt.Fprintf(os.Stderr, "init: %v\n", err)
        os.Exit(1)
    }

    // TODO: wire real Eino ChatModel when config is provided
    var gen rhythm.Generator = rhythm.NewMockGenerator()
    if *apiKey != "" {
        // gen = rhythm.NewEinoGenerator(...)
        _ = *baseURL
        _ = *model
    }

    re := rhythm.NewEngine(engine, gen, rhythm.Config{
        BaseURL: *baseURL,
        APIKey:  *apiKey,
        Model:   *model,
    })

    if err := re.Run(ctx, rhythm.RhythmKind(*kind)); err != nil {
        fmt.Fprintf(os.Stderr, "run: %v\n", err)
        os.Exit(1)
    }
    fmt.Println("rhythm report generated")
}
```

**Step 2: Verify build**

Run:
```bash
go build ./cmd/laputa
```

Expected: PASS.

**Step 3: Run smoke test**

```bash
./laputa -dir /tmp/laputa-smoke -kind daily
```

Expected: "rhythm report generated"，且 `/tmp/laputa-smoke/sections/07-daily.json` 包含一条 report。

**Step 4: Commit**

```bash
git add cmd/laputa/main.go
git commit -m "feat(cli): add laputa rhythm command"
```

---

### Task 7: Wire real Eino ChatModel

**Objective:** 当提供 API key 时，使用真正的 OpenAI-compatible ChatModel。

**Files:**
- Modify: `internal/rhythm/generator.go`
- Modify: `cmd/laputa/main.go`

**Step 1: Add constructor**

在 `generator.go` 增加：

```go
import (
    "github.com/cloudwego/eino-ext/libs/acl/openai"
    "github.com/cloudwego/eino/components/model"
)

// NewOpenAIGenerator creates a generator backed by an OpenAI-compatible API.
func NewOpenAIGenerator(baseURL, apiKey, model string) (Generator, error) {
    cm, err := openai.NewChatModel(context.Background(), &openai.ChatModelConfig{
        BaseURL: baseURL,
        APIKey:  apiKey,
        Model:   model,
    })
    if err != nil {
        return nil, err
    }
    return NewEinoGenerator(cm), nil
}
```

> 注：`openai.ChatModelConfig` 字段名以实际 eino-ext 版本为准。

**Step 2: Update CLI**

在 `cmd/laputa/main.go` 中替换 mock 逻辑：

```go
var gen rhythm.Generator
var genErr error
if *apiKey != "" {
    gen, genErr = rhythm.NewOpenAIGenerator(*baseURL, *apiKey, *model)
} else {
    gen = rhythm.NewMockGenerator()
}
if genErr != nil {
    fmt.Fprintf(os.Stderr, "generator: %v\n", genErr)
    os.Exit(1)
}
```

**Step 3: Verify build**

Run:
```bash
go build ./cmd/laputa
```

Expected: PASS.

**Step 4: Commit**

```bash
git add internal/rhythm/generator.go cmd/laputa/main.go
git commit -m "feat(rhythm): wire real OpenAI-compatible ChatModel"
```

---

### Task 8: Integration test with real LLM (optional)

**Objective:** 验证真实 LLM 调用路径。

**Files:**
- Create: `internal/rhythm/integration_test.go`（带 `//go:build integration` tag）

**Step 1: Write integration test**

```go
//go:build integration

package rhythm

import (
    "context"
    "os"
    "testing"

    "github.com/dashimaki/laputa"
)

func TestIntegration_RhythmDaily(t *testing.T) {
    apiKey := os.Getenv("OPENAI_API_KEY")
    if apiKey == "" {
        t.Skip("OPENAI_API_KEY not set")
    }

    ctx := context.Background()
    store, _ := laputa.NewFileStore(t.TempDir())
    engine := laputa.NewEngine(store)
    _ = engine.Initialize(ctx)

    gen, err := NewOpenAIGenerator("https://api.openai.com/v1", apiKey, "gpt-4o-mini")
    if err != nil {
        t.Fatalf("generator: %v", err)
    }

    re := NewEngine(engine, gen, Config{})
    if err := re.Run(ctx, RhythmDaily); err != nil {
        t.Fatalf("run: %v", err)
    }

    daily, _ := engine.GetSection(ctx, laputa.SectionDaily)
    reports := daily["reports"].([]any)
    if len(reports) == 0 {
        t.Error("expected report")
    }
}
```

**Step 2: Document how to run**

```bash
export OPENAI_API_KEY=sk-...
go test ./internal/rhythm -tags integration -run TestIntegration_RhythmDaily -v
```

Expected: PASS（消耗 API 调用）。

**Step 3: Commit**

```bash
git add internal/rhythm/integration_test.go
git commit -m "test(rhythm): add integration test for real LLM"
```

---

### Task 9: Update documentation

**Objective:** 让 README 和 ARCHITECTURE 反映节律能力。

**Files:**
- Modify: `README.md`
- Modify: `ARCHITECTURE.md`

**Step 1: Update README.md**

在 Quick Start 后增加：

```markdown
## Rhythm Reports

Generate periodic reports using an LLM:

```bash
# Mock generator (no API key)
go run ./cmd/laputa -kind daily

# Real LLM
export OPENAI_API_KEY=sk-...
go run ./cmd/laputa -kind daily -api-key $OPENAI_API_KEY
```
```

**Step 2: Update ARCHITECTURE.md**

在 Architecture 图后增加 Rhythm 层：

```markdown
## Rhythm Layer

```
Laputa Engine
    ↓ snapshot
Rhythm Engine
    ↓ prompt
Eino ChatModel
    ↓ report JSON
Laputa Section (daily/weekly/monthly)
```
```

**Step 3: Commit**

```bash
git add README.md ARCHITECTURE.md
git commit -m "docs: document rhythm engine and CLI"
```

---

### Task 10: Final verification

**Objective:** 全量构建和测试通过。

**Step 1: Run all tests**

```bash
go build ./...
go test ./... -v -count=1
```

Expected: all PASS.

**Step 2: Run CLI smoke**

```bash
rm -rf /tmp/laputa-final && go run ./cmd/laputa -dir /tmp/laputa-final -kind weekly
cat /tmp/laputa-final/sections/08-weekly.json
```

Expected: JSON 包含一条 weekly report。

**Step 3: Commit any final changes**

```bash
git add -A
git commit -m "chore: final rhythm verification"
```

---

## Tests / Validation

| Test | Command | Expected |
|---|---|---|
| Unit: prompt | `go test ./internal/rhythm -run TestBuildPrompt -v` | PASS |
| Unit: generator | `go test ./internal/rhythm -run TestGeneratorGenerate_Mock -v` | PASS |
| Unit: rhythm engine | `go test ./internal/rhythm -run TestRhythmEngineRun_Daily -v` | PASS |
| Build | `go build ./...` | OK |
| All tests | `go test ./... -v -count=1` | PASS |
| Integration | `go test ./internal/rhythm -tags integration -run TestIntegration_RhythmDaily -v` | PASS（需 API key） |
| CLI smoke | `go run ./cmd/laputa -kind daily` | report generated |

---

## Risks, Tradeoffs, and Open Questions

| # | Risk / Tradeoff | Mitigation |
|---|---|---|
| 1 | Eino API 版本可能与计划中的代码不匹配 | 执行时根据实际 import 路径和结构体字段调整 |
| 2 | LLM 返回非 JSON | 加 retry / 用 Eino structured output 或 JSON mode |
| 3 | 真实 API key 在测试环境可能不存在 | mock generator 覆盖默认路径，integration test 用 build tag |
| 4 | 节律触发依赖外部 cronjob | 本计划不实现 daemon，后续可加 Hermes cronjob 调用 CLI |
| 5 | AutoDream 状态机未实现 | 本计划只做"节律触发链"，完整 AutoDream 需后续 PRD |

### Open Questions

1. 大湿希望节律引擎默认用 mock 还是必须配 API key 才能运行？
2. 报告生成后是否需要写 changelog section？
3. 是否需要把 rhythm report 同时索引到 mempalace？
4. daily/weekly/monthly 的触发时间/时区如何确定？

---

## Files Likely to Change

- `go.mod`
- `go.sum`
- `internal/rhythm/types.go`
- `internal/rhythm/prompt.go`
- `internal/rhythm/prompt_test.go`
- `internal/rhythm/generator.go`
- `internal/rhythm/generator_test.go`
- `internal/rhythm/rhythm.go`
- `internal/rhythm/rhythm_test.go`
- `internal/rhythm/integration_test.go`
- `cmd/laputa/main.go`
- `cmd/eino_smoke/main.go`（临时，可删除）
- `README.md`
- `ARCHITECTURE.md`
