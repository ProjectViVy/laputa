# Vivy 接入交接（本地库已可消费；Vivy 尚未集成）

## 边界与现状

- 本分支 `feat/laputa-modular-monolith` 只修改 Laputa/Garden/Mentle 与独立 Go 消费夹具；**不修改 Vivy、不 push、不发布模块**。远端 Issue #1 和旧 SDK 工作树不在此分支交付范围。
- 模块名仍为 `github.com/dashimaki/garden`、`github.com/dashimaki/laputa`、`github.com/dashimaki/mentle`。当前独立示例使用本地 `replace`；远程版本/仓库地址不是已发布的接入方式。
- Laputa 仍负责 Persona（含显式 WORLD）和 ACTMEM 权威，Mentle canonical SQLite 是唯一记忆权威；BM25/向量、Garden transient spool 与 Vivy Journal **不是**第二套 canonical。Garden 负责能力检查、Frozen Core、召回与会话捕获。Vivy 不得绕开 Garden 直连 Mentle。
- `garden/agentapi` 是宿主的公开进程内入口；`garden/internal/runtimecore`、HTTP server、Console 都不是 Vivy 的 import 契约。`go list -deps ./agentapi` 已核对不含 `garden/internal/server`、Console、MCP 命令包。

## 宿主装配（示意，实际可执行代码见 `examples/vivy-embed-smoke/main.go`）

```go
client, err := agentapi.Open(ctx, agentapi.Config{
    PersonaDir: personaDir, PalacePath: palacePath,
    ModelsDir: modelsDir, StateDB: stateDB,
    ProfileID: trustedProfileID, AgentID: trustedAgentID,
    Platform: "vivy", Principal: agentapi.PrincipalAgent,
})
if err != nil { return err }
defer client.Close()
session, err := client.BindSession(hostSessionID)
if err != nil { return err }
view, err := session.Bootstrap(ctx, agentapi.BootstrapRequest{
    Intent: intent, BudgetChars: budget,
})
```

所有路径由受信宿主提供绝对路径；`ProfileID`、`AgentID`、`Principal`、`Platform` 在 `Open` 固定，模型输出、HTTP body、`X-Garden-Actor` 不能选择它们。Vivy 的 `Session/Run/Journal` 继续是宿主生命周期事实；不要合并其数据库与 Mentle canonical。关闭时先停止新的 Run/capture 调用，再 `client.Close()`；关闭后的 bound handle 返回 unavailable。

| Vivy 后续接点（本分支不实施） | Garden 公开调用 | 不变量 |
| --- | --- | --- |
| 首次模型请求前 | `BindSession(hostSessionID)` → `Bootstrap` | Frozen Core 只含六槽；WORLD/ACTMEM 不进入自动上下文；预算限制/降级状态照实传给模型上下文装配。 |
| 需要候选时 | `FastRecall`；显式工具调用可用 `ReadPersona`、`ReadActmem`、`SearchCards`、`ReadEvidence`、`IndexHealth` | Context Source 仅消费候选，不把其当作终态捕获钩子；显式 WORLD/ACTMEM 读取必须通过固定 principal 校验。 |
| Run/Journal 已持久记录终态后 | `Capture` → 必要时 `CaptureStatus(ingestionID,eventID)` | 只传 `completed/failed/canceled`；用 Journal 的 durable `run_id` 与单调 `event_seq` 建稳定事件身份，原文 SHA-256 为 `sha256:<hex>`；同事件同内容重放同 receipt，改内容冲突；非终态不捕获。 |

Vivy **还需**自己把上述三处接入 MemoryPort、模型请求装配和 Run/Journal 终态回调，并决定重试任务由谁发起。Garden 已持久化 ingest 状态与 event identity，不允许另建绕过 Garden 的写队列或双写 Persona/Mentle 权威。捕获被接纳不表示向量索引也已完成；查询 `IndexHealth`/派生 outbox 状态后才可声明派生健康。无模型时不应把 spool 收件说成已写入 canonical。

## 离线运行语义

| 条件 | 实际行为 |
| --- | --- |
| 显式模型目录下有完整本地模型 | 严格本地初始化 Mentle；canonical 可写、向量与 BM25 可派生；坏/损坏模型导致初始化失败，不回退下载或偷偷选 CWD。 |
| 模型缺失但已有 canonical.sqlite3 | 只读打开旧 canonical，使用 BM25 候选；禁止 canonical/向量写入，FastRecall 标记 `degraded`/lexical-only；摄取只进入持久化待处理状态，待完整模型恢复后 drain。旧 schema 不在只读模式迁移。 |
| 模型与 canonical 均缺失 | Frozen-only 上下文仍可用，Mentle 搜索明确 unavailable，摄取状态保持可恢复；不得创建伪 canonical 或假索引健康。 |

`RequireLocalModel` 与 `LexicalOnly` 是互斥的 Mentle 底层选项。公开宿主应始终提供显式目录；是否安装本地模型资产是接入部署决策，不是首次请求时下载的副作用。

## 可复验命令及尚未完成事项

```sh
# 在仓库根执行；Garden Go 测试前先构建 console embed dist（生成物不提交）。
(cd garden/console && npm install --no-package-lock --ignore-scripts && npm run build)
(cd mentle && CGO_ENABLED=0 GOSUMDB=off go test ./... -count=1 && CGO_ENABLED=0 GOSUMDB=off go vet ./...)
(cd laputa && CGO_ENABLED=0 GOSUMDB=off go test ./... -count=1 && CGO_ENABLED=0 GOSUMDB=off go vet ./...)
(cd garden && CGO_ENABLED=0 GOSUMDB=off go test ./... -count=1 && CGO_ENABLED=0 GOSUMDB=off go test -tags=e2e ./e2e/... -count=1 && CGO_ENABLED=0 GOSUMDB=off go vet ./...)
(cd examples/vivy-embed-smoke && CGO_ENABLED=0 GOSUMDB=off GOTOOLCHAIN=local go run . && CGO_ENABLED=0 GOSUMDB=off go vet ./...)
```

独立示例已经实测打印 `PASS: external consumer; offline pure-Go open/bind/bootstrap/recall/explicit WORLD/capture/replay/restart/close`；完整最终工作树 Gate 以实施签收后的**实际复跑**为准，不能据此声称 Vivy `just ci` 已通过。Garden 现有 REST/MCP 继续作为外部传输；MCP 只广告实际注册的工具，**不**广告 bootstrap/capture。旧 `laputa-agent/1` Wire/SDK worktree 是独立候选，不当成本分支已发布接口。

Vivy 只读依赖核对显示其 `go.mod` 与 Garden 同为 Go 1.26.4，但 `mcp-go` 与 `modernc.org/sqlite` 约束版本不同；真实 Vivy 模块依赖合并、`just ci`、RunHook 负例、首次模型请求与重启 live-path、远端模块发布/Issue 关闭都属于后续单独授权和验证。不要根据本地 `replace` smoke 推断 MVS 或远端拉取已通过。
