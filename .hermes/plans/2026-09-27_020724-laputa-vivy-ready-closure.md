# Laputa 库拆分与 Vivy-ready 闭环实施计划

## Goal

在不修改 AGENT-VIVY 的前提下，将现有 Laputa、Mentle、Garden 整理成可由独立 Go 宿主在进程内导入、与单体应用共享领域语义且经本地跨传输一致性验证的库与适配层。

## Current context / assumptions

- 工作树：`C:/Users/Administrator/Desktop/garden-modular-monolith`，分支 `feat/laputa-modular-monolith`，基线提交 `21c032c2964e3d58f6549e0563e45f4ce1342b0a`；执行前重新检查 HEAD/status。主工作树 `C:/Users/Administrator/Desktop/garden` 和旧 SDK worktree `C:/Users/Administrator/Desktop/garden-laputa-agent-sdk` 有未提交改动，**绝不 reset/clean/stash/rebase/cherry-pick/覆盖**。Vivy 在 `C:/Users/Administrator/Desktop/morediva/diva-go/agent-vivy`，同样只读。未获单独授权不 push、不迁移真实用户数据、不公开发布、不改公网暴露/认证模型/权威语义。
- 约束：`docs/architecture/0012*`、`0013*`、`0014*`、`0015-laputa-external-agent-contract.md`、`0016-laputa-embeddable-modular-monolith.md`；`docs/bmad/laputa-modular-monolith-2026-09/conformance.md` 的 C01–C08 必须通过，C09 在本批仅证明“宿主侧准备就绪”，不能声称 Vivy 已接入。DIVA Rust Persona 语义优先；七个大写 Markdown 文件是 Persona 权威；WORLD/ACTMEM 仅显式调用；Mentle `canonical.sqlite3` 和同事务 `index_jobs` 是唯一记忆权威；`vectors.db` 为 bbolt 派生索引，不是 SQLite；Vivy Journal 不是第二份记忆权威。
- 当前三个嵌套 Go module：`laputa/go.mod` (`github.com/dashimaki/laputa`)、`mentle/go.mod` (`github.com/dashimaki/mentle`)、`garden/go.mod` (`github.com/dashimaki/garden`，本地 `replace ../laputa,../mentle`)；远端仓库是 `github.com/ProjectViVy/laputa`，没有现成模块 tag。**本计划交付本地可导入/可构建能力和接入手册，不假装这些 `dashimaki` 路径已在远端可 `go get`**。远程模块路径、tag、独立仓库/公开发布须另经 owner 拍板。
- 当前 Mentle `facade.Options` 已有 `ConfigDir/PalacePath/ModelsDir`，双显式路径时不加载 ambient Viper 配置；尚无关闭后重开验证。`facade.Init` 调 `embedder.New`，该函数找不到指定模型时会回退 cwd 模型、再尝试下载。`mentle/facade/canonical.go` 和 `mentle/storage/sqlite/store.go` 使用 mattn CGO 驱动；`garden/internal/{activity,evolution,ingest,mailbox,personactx,recall,report}` 多处亦然。`CGO_ENABLED=0 go test ./facade` 已实测失败于 KG 的 `go-sqlite3` stub；不能仅替换 Mentle 就宣称 Vivy-ready。
- Garden 主装配在 `garden/main.go`；`garden/internal/server/server.go` 中 HTTP handler 直接调 `recall.FastService`、`ingest.Service`、`facade.Service`，认证在 `garden/internal/server/auth.go`；`garden/cmd/garden-mcp/main.go` 为单独的 stdio→REST 适配器。禁止在公开库里暴露 `database/sql` 连接、内部 Persona 存储、向量索引或服务端可伪造 principal 的入口。
- “所有闭环”的本地完成定义：可复用的三库入口、单体应用保持原路由/Console/MCP、纯 Go + 离线启动、冻结/召回/终态捕获/权限/重启/派生恢复的一套行为契约，在同进程和 HTTP 上跑通；MCP 对其实际广告的工具做映射一致性；独立临时 Go 消费者编译运行；接入文档说明 Vivy 应如何绑定已有 Session/Run/Journal，但绝不触碰其代码或配置。真正 Vivy 端集成和正式分发不属本批。

## Architecture / proposed approach

保留三个 Go module，新增窄公开 `garden/agentapi`（宿主领域调用）和内部 `garden/internal/runtimecore`（共同资源装配/关闭）；`garden/internal/server` 及 `garden/main.go` 只作为调用同一 `agentapi` 的应用适配，MCP 仍是 REST 上的薄适配，不复制业务规则。Mentle 给完全显式配置增加“严格本地模型 / 离线模式”，SQLite 统一使用经既有文件升级、重启和事务故障验证的纯 Go 驱动；Garden 需要自己的纯 Go SQLite 连接封装，避免借 Mentle 的存储层承担 Garden 状态。所有领域 API 用显式可信 `Binding` 绑定单一 profile/principal；请求体里的 actor/role/profile 不能提权；传输层只负责认证、解析和序列化。

## Execution contract (每个编码小步都适用)

下面编号 1–32 是顺序化的 **2–5 分钟聚焦操作**；复杂功能要按编号连续做，不允许一次性写完大模块。每个出现 `[RED/GREEN]` 的步骤：先仅写该步骤测试，运行列出的聚焦命令并看见“预期行为缺失”而非语法错误；再做最小生产变更，重跑聚焦命令应为 exit 0；运行所属模块 `go test ./... -count=1`（Garden 先构建 Console），最后 `git diff --check`，通过后独立 commit。若旧行为测试立即绿色，调整测试为能够证明新增能力的真实失败例，不制造假 RED。每步仅操作此 worktree；完成一步立即记录测试结果/commit，失败即停，不扩大范围。提交示例：`git add -- <本步明确文件> && git commit -m 'feat(mentle): ...'`，不得 `git add .`、`git push`。计划文档不是已执行证据。所有 Go 命令在各自模块目录执行（终端 `workdir` 或 `cd` shell builtin），Git Bash POSIX 语法；不要以 `/c/...` 路径喂给 Go/git 原生程序。

### Phase A：基线、Mentle 初始化与纯 Go

1. **锁定现场**（只读）：`git status --short --branch && git rev-parse HEAD`；预期本分支 HEAD 为上述基线，若已前进，审阅新增 diff 并更新执行记录后再继续。分别 `git -C C:/Users/Administrator/Desktop/garden status --short`、`git -C C:/Users/Administrator/Desktop/garden-laputa-agent-sdk status --short`、`git -C C:/Users/Administrator/Desktop/morediva/diva-go/agent-vivy status --short`，预期后三者现有脏文件保持不变。读 `docs/architecture/0012*` 至 `0016*`、`docs/bmad/laputa-modular-monolith-2026-09/conformance.md`、`mentle/facade/{facade.go,canonical.go,outbox.go}`、`garden/main.go`、`garden/internal/server/{server.go,auth.go}`，将首批 C01–C08 基线 fixture 名保留，不得把曾通过的旧测试当新入口通过。
2. **M1 重开 RED/GREEN**：在 `mentle/facade/init_options_test.go` 添加 `TestInitExplicitPalaceReopensCanonical`: 使用已有 `bundledModelsDir(t)`、`t.TempDir()` 和双显式路径初始化；向 Catalog 写入一条具有确定 idempotency key 的记忆（调用示例取 `facade_test.go:TestCanonicalMemoryLifecycleAndIdempotency` 的真实 `CreateMemoryWithIndexStatus` 参数与写法，不要直接 SQL）；`Close` 后重新 `Init` 同路径，用 `GetMemory` 比对 ID/version/content，确认只出现一份，且其他新 palace 查不到它。先通过“当前无可复用 reopen API/资源清理”失败的测试界定缺口；若旧实现已经通过，记录为现存能力并 **不要为造 RED 修改代码**。聚焦命令：`GOSUMDB=off go test ./facade -run '^TestInitExplicitPalaceReopensCanonical$' -count=1 -v`；GREEN 后 `GOSUMDB=off go test ./... -count=1`，应看到 `ok github.com/dashimaki/mentle/facade` 和 exit 0。
3. **初始化失败资源回收 RED/GREEN**：同文件加入 `TestInitFailureReleasesVectorStore`，给 KG 指定不可写路径/可控故障（在 `facade.Init` 当前 KG 打开路径处注入仅供测试的工厂；不修改真实 canonical 文件），Init 返回后确认同 palace `vectors.db` 可立即删除/重新打开，Windows 不得出现 sharing violation。最小修正 `mentle/facade/facade.go` 的 `vectorDB` 错误回收和 `mentle/internal/kg/knowledge_graph.go` 的 `initDB()` 错误时 `db.Close()`；复跑 `go test ./facade ./internal/kg -count=1` 和全模块；提交。该测试对 CGO=0 的既有 stub 报错亦应能安全清理临时目录。
4. **纯 Go 驱动依赖**：进入 `mentle`，先 `go get modernc.org/sqlite@latest` 和 `go mod tidy`，将解析得到的确切版本提交到 `mentle/go.mod`、`mentle/go.sum`，不在本计划伪造版本号；若下载失败，记录网络阻断并停止，不能伪称通过。用 `go list -m modernc.org/sqlite` 预期输出 `modernc.org/sqlite v...` 且 exit 0。驱动仅是工具，任何行为变更要遵守后续 RED/GREEN；不能先替换全局导入。
5. **Mentle SQLite 连接单元测试 RED/GREEN**：先改 `mentle/storage/sqlite/store_test.go`（无则新建），`TestOpenPureGoWALAndForeignKeys`: `Open(t.TempDir()+"/kg.sqlite3")`，`PRAGMA journal_mode` 应为 `wal`，`PRAGMA foreign_keys` 为 `1`，创建约束表后插入不存在的外键应失败，关闭重开数据仍在；额外 `TestOpenPureGoCGODisabled` 通过外部 `CGO_ENABLED=0 go test ./storage/sqlite -count=1` 实际调用（无需测试内启动子进程）。RED 后在 `mentle/storage/sqlite/store.go` 使用 `database/sql` + `_ "modernc.org/sqlite"`，`sql.Open("sqlite", path)`，`SetMaxOpenConns(1)`，在 Exec(PRAGMA) 失败时 `db.Close()`；成功设置 WAL、`foreign_keys=ON`、`busy_timeout=5000`，每条错误 return 原因。仅处理该 helper，不动 canonical；复跑 `CGO_ENABLED=0 GOSUMDB=off go test ./storage/sqlite ./internal/kg -count=1`。旧 KG 文件副本的读取验证须在下一步的 fixture 中完成，不允许真实 palace 迁移。
6. **Canonical driver RED/GREEN**：先在 `mentle/facade/canonical_test.go`（若无则新建；如 `facade_test.go` 已有相同 fixture 则扩展）添加 `TestCanonicalPureGoReopenAndOutbox`: 创建临时 catalog，写入记忆且断言 `index_jobs` 与 memory 同事务，关闭、以新驱动重开、校验 ID/version/幂等 key/待处理 job；在由当前 CGO 驱动制造的测试 SQLite 文件副本上重复“旧文件可打开”断言。修改 `mentle/facade/canonical.go` 的 `_ "github.com/mattn/go-sqlite3"` 与 `sql.Open("sqlite3", path+... DSN)` 为 `_ "modernc.org/sqlite"` + `sql.Open("sqlite", path)`，在单连接上显式 Exec `PRAGMA journal_mode=WAL`, `PRAGMA foreign_keys=ON`, `PRAGMA busy_timeout=5000`，失败必 `db.Close()`，原有 schema/SQL/idempotency/outbox 不改。跑 `CGO_ENABLED=0 GOSUMDB=off go test ./facade -run '^TestCanonicalPureGoReopenAndOutbox$' -count=1 -v`，再 `CGO_ENABLED=0 GOSUMDB=off go test ./... -count=1`，应不含 `go-sqlite3 requires cgo`。纯 Go 驱动对 SQL 返回的时间/布尔/错误可能不同，发现差异先补真实回归 fixture，再作最小兼容修正；不调整 canonical schema。
7. **严格离线模型 RED/GREEN**：`mentle/internal/embedder/hugot_test.go` 新增 `TestNewLocalMissingModelDoesNotUseCWDFallbackOrNetwork`：切到拥有 `models/onnx/model.onnx` 的目录但显式指定空 `modelsDir`；调用新增 `NewLocal(modelName,modelsDir)` 应返回 `ErrLocalModelMissing`，并断言指定目录/外部路径没有被创建/写入。新增 `TestNewLocalBundledModelSucceeds` 显式指定真实 fixture；模型 fixture 缺失时测试应失败并报告 gate unavailable，不能 `t.Skip` 假通过。先看编译 RED，随后在 `mentle/internal/embedder/hugot.go` 新增严格 `NewLocal`，只检查 `modelsDir/model.onnx` 或 `modelsDir/onnx/model.onnx`，两者不存在立即返回可 `errors.Is` 的 sentinel；存在时复用现有 `newFromPath`。原 `New` 下载/回退保持旧 CLI 行为。`GOSUMDB=off go test ./internal/embedder -run '^TestNewLocal' -count=1 -v` 预期 GREEN。
8. **嵌入式模式传递**：在 `mentle/facade/init_options_test.go` 新增 `TestInitOfflineMissingModelCannotDownload`，使用 `Options{PalacePath: t.TempDir(), ModelsDir: emptyDir, RequireLocalModel: true}`，应快速失败 `ErrLocalModelMissing`，不碰 cwd 模型、外部网络、canonical；另一个同模型 fixture 初始化成功且关闭后可重开。RED 后在 `mentle/facade/facade.go` 的 `Options` 添加 `RequireLocalModel bool`，`Init` 只在此显式为 true 时调用 `embedder.NewLocal`，默认继续旧 `New`，不得静默回退或下载。`CGO_ENABLED=0 GOSUMDB=off go test ./facade -run '^TestInitOffline' -count=1 -v` 及 `CGO_ENABLED=0 GOSUMDB=off go test ./... -count=1` 预期 exit 0。**Lexical-only** 若未落实不要标成能力：再设单独 `TestInitLexicalOnlyKeepsCanonicalJobsPending`，只在能维持 canonical commit+pending outbox/无空 vector 假成功时才新增对应模式；否则作为明确 stop gate，不以 offline local-model 代替 lexical-only。

### Phase B：Garden 纯 Go 状态库与单一领域入口

9. **Garden 纯 Go 连接 helper**：先在 `garden/internal/sqliteconn/open_test.go`（新建）写 WAL/busy_timeout/关库重开/不支持 CGO 时读写 fixture，RED；`garden/go.mod` 加 `modernc.org/sqlite`（`go get modernc.org/sqlite@latest`、`go mod tidy`，若与 Mentle 不同版本则对齐依赖版本），在新 `garden/internal/sqliteconn/open.go` 实现一个 `Open(path string) (*sql.DB,error)`：`sql.Open("sqlite",path)`、`SetMaxOpenConns(1)`、依次 Exec 三条 PRAGMA，错误时 Close。`CGO_ENABLED=0 GOSUMDB=off go test ./internal/sqliteconn -count=1 -v` 应 `ok`。不修改 `mentle/storage/sqlite` 的归属来给 Garden 共享 helper。
10. **Frozen/trace/ingest 状态迁移**：逐文件、一次一个旧/新数据库 fixture 的 RED→GREEN，按顺序：`garden/internal/personactx/frozen.go` + `frozen_test.go`、`garden/internal/recall/trace.go` + `trace_test.go`、`garden/internal/ingest/service.go` + `service_test.go`。每次仅把旧 `sql.Open("sqlite3", path+?...` 和驱动导入替换为 `sqliteconn.Open(path)`；保留表定义和状态语义。`ingest` 专项增加旧文件重开、相同 event+hash replay 返回原 ingestion、相同 event+不同 hash `ErrEventConflict`、Close→Open 后未处理作业重新排队的 fixture；`CGO_ENABLED=0 GOSUMDB=off go test ./internal/personactx ./internal/recall ./internal/ingest -count=1` 应 `ok`，每文件完成即 commit。
11. **其余 Garden 状态迁移**：同样逐文件的旧文件重开/事务/外键测试与小提交，文件是 `garden/internal/activity/{event.go,transient.go,checkpoint.go}`、`garden/internal/evolution/{store.go,events.go}`、`garden/internal/mailbox/mailbox.go`、`garden/internal/report/service.go`；`garden/internal/report/service_test.go` 的测试连接也改用 helper 或等价纯 Go 入口。执行 `CGO_ENABLED=0 GOSUMDB=off go test ./internal/activity ./internal/evolution ./internal/mailbox ./internal/report -count=1`，预期 exit 0；`go mod tidy` 后 `git grep -n 'go-sqlite3\|sql.Open("sqlite3"' -- garden mentle` 预期没有生产代码命中（文档可留历史）；不得逐文件保留混合驱动写同一 state DB。
12. **主模块双构建 gate**：准备 `garden/console/dist` 前先在 `garden/console` 执行 `npm install --no-package-lock --ignore-scripts && npm run build`，预期显示 `✓ built`；`garden/console/embed.go` 需要 dist 才能编译。之后 `CGO_ENABLED=0 GOSUMDB=off go test ./... -count=1`（`garden`）、`CGO_ENABLED=0 GOSUMDB=off go test -tags=e2e ./e2e/... -count=1`、Mentle 和 Laputa 各 `CGO_ENABLED=0 GOSUMDB=off go test ./... -count=1`，预期全 exit 0；再重复 CGO 默认配置测试，避免只修非 CGO 模式。只清理本 worktree 此次生成的 `garden/console/{dist,node_modules}`，对原先干净的受构建影响的 `tsconfig.tsbuildinfo` 从本分支 HEAD 还原；不 `git clean`、不动主工作树。
13. **最小领域接口 RED**：新建 `garden/agentapi/contract.go` 与 `contract_test.go`，先定义公开 DTO 并写 compile-contract 测试；禁止公开方法签名含 `garden/internal/...` 类型、`*sql.DB`、HTTP `Request`、`*facade.Service` 或原始 Persona 文件路径。至少包含 `Binding{ProfileID,Principal,AgentID}`（trusted constructor 绑定，调用请求不带 principal）、`BootstrapRequest{SessionID,Intent,BudgetChars}`、`CaptureRequest{SessionID,EventID,Phase,Content,ContentHash,OccurredAt}`、`CaptureAccepted{IngestionID,SessionID,EventID,Status}`、`Error{Code,Message}`；JSON 命名采用现有 REST wire profile，字段必填与 256–64000 budget 保持既有语义。以下是 **完整 DTO 骨架**，可直接作为 `contract.go` 初稿，再以真实测试驱动扩展（不加可伪造的 `Actor` 字段）：

   ```go
   package agentapi

   import "time"

   type Principal string
   const (
       PrincipalRead Principal = "read"
       PrincipalUser Principal = "user"
       PrincipalAgent Principal = "agent"
       PrincipalAutodream Principal = "autodream"
       PrincipalOperator Principal = "operator"
   )
   type Binding struct {
       ProfileID string
       Principal Principal
       AgentID string
   }
   type BootstrapRequest struct {
       SessionID string `json:"session_id"`
       Intent string `json:"intent"`
       BudgetChars int `json:"budget_chars"`
   }
   type CaptureRequest struct {
       SessionID string `json:"session_id"`
       EventID string `json:"event_id"`
       Phase string `json:"phase"`
       Content string `json:"content"`
       ContentHash string `json:"content_hash"`
       OccurredAt time.Time `json:"occurred_at,omitempty"`
   }
   type CaptureAccepted struct {
       IngestionID string `json:"ingestion_id"`
       SessionID string `json:"session_id"`
       EventID string `json:"event_id"`
       Status string `json:"status"`
   }
   type Error struct { Code string `json:"code"`; Message string `json:"message"` }
   func (e *Error) Error() string { return e.Code + ": " + e.Message }
   ```

   `go test ./agentapi -count=1` 应在 DTO 缺失时先编译 RED，完成后 `ok`；明确 `Binding` 只能由受信宿主装配器构造，`ProfileID` 必须匹配服务启动时固定的一个 profile，不从模型输入解析。
14. **冻结/预算/错误映射契约**：`garden/agentapi/contract_test.go` 加表驱动测试：六 section 映射 `identity/relationship/redline/user/dream/dark`；WORLD/ACTMEM 字串不得进入自动 `context`；budget 255/64001 返回 `invalid_request`；缺少 SessionID 不得跨 profile 偷读；模型检索失败保留 Frozen Core 且 `degraded=true`；权限拒绝用 `principal_forbidden` 而非降级。公开 `ContextView` 需镜像 `garden/internal/recall/fast.go:ContextView` 的 JSON `trace_id/scope/mode/frozen_core/cards/evidence/context/budget_chars/degraded/warnings/recall_trace_id`，但字段类型一律公开 DTO 或公共 `facade` 类型；不要暴露 `personactx` internal 类型。RED 后仅实现转换/分类函数，`go test ./agentapi -run '^TestContract' -count=1 -v` 应 `ok`。
15. **共同 runtime 装配起点**：新建 `garden/internal/runtimecore/runtime.go` 和 `runtime_test.go`。测试先证明 `Open(ctx, Config{PersonaDir,PalacePath,ModelsDir,StateDB,ProfileID,RequireLocalModel:true})` 只在给定根目录创建 Persona/Mentle/Garden 状态，不读取 ambient 配置、不监听端口；`Close()` 两次无资源泄漏；Mentle 缺席可 Frozen-only，但 Persona 非 ready 时遵守原 read/CAS 错误；另一 profile 请求拒绝。实现时按 `garden/main.go:30-53,75-152` 的**原顺序**搬 Persona、ACTMEM、Mentle、Frozen Store、FastRecall、activity、ingest/trace 等数据面初始化及逆序关闭，不引入第二张 authority 表；HTTP 管理面 pipeline/evolution/report/mailbox 等先留应用层，只有复用时才按单项搬。`CGO_ENABLED=0 GOSUMDB=off go test ./internal/runtimecore -count=1 -v` 应 `ok`。
16. **公开装配，不泄漏句柄**：`garden/agentapi/service.go` 和 `service_test.go` 先写 `Open` 创建 `runtimecore`，绑定只读 `Binding`，`Service.Close` 释放底层；对外只允许 `Bootstrap/FastRecall/Capture/ReadPersona/ReadActmem/SearchCards/ReadEvidence/IndexHealth` 这样的领域操作，不返回 `runtimecore.Runtime` 指针或 `Facade`。先只做 bootstrap tracer：建一次 session Frozen Core，用 `Service.Bootstrap` 得到六 section、bounded context；测试 `TestBootstrapNoWorldActmem` RED→GREEN，命令 `go test ./agentapi -run '^TestBootstrapNoWorldActmem$' -count=1 -v`。首次公开入口编译时 `go list -deps ./agentapi` 不应依赖 `garden/console` 或 `garden/internal/server`，避免把嵌入式库绑上前端产物。
17. **统一 principal/profile 策略**：在 `garden/agentapi/policy.go` + `policy_test.go` 先表驱动 RED：同一固定 profile 的 read/agent/user/operator 权限表；Agent 不能 approve Persona review、不能通过请求体 actor 冒充 user/operator；未绑定/错误 profile 一律拒绝，不能使用 loopback GET 免认证；异常/无效 principal fail-closed。`garden/internal/server/auth.go` 可保留 token→principal 的认证逻辑，但 allow/deny 操作规则应下沉到同一个 `agentapi.Authorize(binding,operation)`，HTTP `requirePrincipal` 只做 token 验证和授权调用；**不要**把 `X-Garden-Actor` 映射成 principal。验证 `go test ./agentapi ./internal/server -run 'Test.*(Principal|Capability|Review)' -count=1 -v`，预期权限负例通过。
18. **Fast recall / Persona / ACTMEM 独立功能**：一项一个 RED→GREEN：`agentapi/service_test.go` 加 `TestFastRecallDegradedKeepsFrozenCore`、`TestPersonaReadRequiresExplicitOperation`、`TestActmemReadRequiresExplicitOperation`，分别把现有 `garden/internal/recall/fast.go`、`laputa/persona/service.go:GetDocument`、`laputa/actmem` 的公开服务方法薄包装到 `agentapi`。禁用“自动上下文捎带 WORLD/ACTMEM”，不得复制 Persona CAS 规则；`go test ./agentapi -run 'Test(FastRecall|PersonaRead|ActmemRead)' -count=1 -v` 应全部 PASS；`garden/internal/recall/fast_test.go` 的既有预算/降级用例仍 PASS。
19. **捕获路径/终态负例**：`garden/agentapi/service_test.go` 写 `TestCaptureRejectsNonTerminal` 和 `TestCaptureTerminalReplayAndConflictAfterRestart`：只接受明确配置为终态的 `session_end`（现有 ingest 另有 `precompact`，只能走单独显式调用，不得由 Vivy terminal 适配模糊映射）；内容 hash 必须是 `sha256:<hex>` 且由原内容校验；event identity 规则为宿主 durable event 的 `run_id:event_seq`，不可基于当前时间；同 key 同 hash 返同 ingestion_id，不同 hash 冲突；Close→Open 后继续查得。RED 后调用原 `ingest.Service.Submit` 与 `Get`（见 `garden/internal/ingest/service.go`），不要在 `agentapi` 新建队列或第二份去重库。`go test ./agentapi -run '^TestCapture' -count=1 -v` 及 `go test ./internal/ingest -count=1` 应 `ok`。
20. **派生健康/补偿**：`mentle/facade/{facade_test.go,index_health_test.go}` 加“canonical commit 成功但向量失败 ⇒ pending job+index health degraded ⇒ 重开恢复”fixture；`garden/agentapi/service_test.go` 对应验证只返回 canonical 已接纳状态或明确降级，不谎报全事务失败。参考既有 `TestCanonicalCommitSurvivesDerivedIndexFailureAndRecovery`；不新造 JSONL WAL 或替代 index_jobs。`go test ./facade -run 'TestCanonicalCommitSurvivesDerivedIndexFailureAndRecovery|TestIndexHealth' -count=1` 和 `go test ./agentapi -run '^TestIndexHealth' -count=1` 应 PASS。

### Phase C：单体应用与传输适配共用领域层

21. **HTTP bootstrap 切换**：`garden/internal/server/server.go` 的 `Server` 增加 `AgentAPI` 字段；`handleBootstrap` 的读取/预算/错误结果从 `agentapi.Service.Bootstrap` 获取，保持 JSON 字段、HTTP 200/400/503 和既有认证不变；先扩展 `garden/internal/server/server_test.go:TestFastRecallAndBootstrapEndpoints`，对同一个 session 同时调 in-process/HTTP，除 trace ID/时间等非语义字段外字段一致，负例不泄 WORLD/ACTMEM。`go test ./internal/server -run '^TestFastRecallAndBootstrapEndpoints$' -count=1 -v` RED→GREEN。若旧测试直接构造 `Server{FastRecall:...}`，用**测试注入的共享领域服务**迁移测试，不能保留运行时双实现分支。
22. **HTTP fast/materials/health 切换**：分别在 `garden/internal/server/{recall_handlers.go,material_handlers.go,admin_handlers.go,server.go}` 现有 handler 上新增共享 fixture；每次一个 handler RED→GREEN，把域调用切到 `AgentAPI.FastRecall/SearchCards/ReadEvidence/IndexHealth`，只让 HTTP 负责解码、认证、状态码、JSON。全量 `go test ./internal/server -count=1` 应 PASS；不要为非公开领域操作暴露原始 Mentle DB。
23. **HTTP ingest/memory、保护面切换**：`garden/internal/server/server.go:handleSessionSubmit` 走 `AgentAPI.Capture`，但兼容原 REST 的显式 `precompact/session_end` 路由规则；对 `/v2/memories` 的写、读和 Persona/ACTMEM API 必须保留各自策略，逐 route 写阳性/阴性一致性测试后才改。相关文件 `server.go`, `persona_api.go`, `actmem_handlers.go`, `memory_server_test.go`, `persona_api_test.go`；避免将所有 CRUD 一口气搬入 `AgentAPI`。每个 route 拆一小步；运行 `go test ./internal/server -run 'Test.*(Memory|Persona|Actmem|Session)' -count=1 -v`，预期 PASS。
24. **主应用切换**：`garden/main.go` 改成创建唯一 `runtimecore`/`agentapi` 实例，然后为 `server.Server` 注入同一个实例；旧数据面初始化删除而不是并行保留。管理面（pipeline,report,evolution,mailbox,Console）原样保持或作为可选 app wiring，`garden/internal/lifecycle` 不变。`garden/console` `npm install --no-package-lock --ignore-scripts && npm run build` 后运行 `CGO_ENABLED=0 GOSUMDB=off go test ./... -count=1` 与 `CGO_ENABLED=0 GOSUMDB=off go test -tags=e2e ./e2e/... -count=1`，预期 PASS；`go list -deps ./agentapi` 仍不包含 `garden/console`。
25. **MCP 薄层一致性**：只对 `garden/cmd/garden-mcp/main.go` 中**实际注册**的工具（Persona status/get/propose/P16、ACTMEM、index-health、memory/evidence、activity、record_signal 等逐项以代码登记）写 `main_test.go` fixture：工具→REST 输出 DTO/错误码与相应 HTTP 一致，未经证据的 bootstrap/capture 不广告。现有 stdio 客户端继续通过 Garden HTTP 入口，绝不复制领域 policy；`go test ./cmd/garden-mcp -count=1 -v`、server+agentapi 对应 fixture 预期 PASS。若旧 dirty SDK worktree 有 agent binding/capture 草稿，仅阅读并以逐 hunk 新测试驱动重建，**不**隐式合并、覆盖或标记已实现。
26. **Console 与单体回归**：`garden/console/src/api/client.ts`、`src/modules/persona/`、`src/modules/memory/` 只在 API 回归 fixture 指出真差异时修改；保持现有地址 `127.0.0.1:7373`，`/health` 和全部现有 `/v2` CRUD/管理面。运行 `npm run build`（console），`CGO_ENABLED=0 GOSUMDB=off go test ./... -count=1`（Garden），`go test -tags=e2e ./e2e/... -count=1`；检查无 body/header actor 提权回归。app 的 HTTP/MCP/Console 仍可独立分发为同一产品套件，但本计划不发布。

### Phase D：外部消费证明、文档与结案

27. **共享 conformance corpus**：`garden/agentapi/conformance_test.go` 放一张可重用 fixture 表，至少覆盖 C01–C08：六槽/WORLD+ACTMEM 阴性、budget 255/64001、Mentle 不可用 Frozen-only、profile/agent 伪造禁止、同事件重放/改 hash 冲突、派生失败已提交、restart、离线模型拒绝。`garden/internal/server/conformance_test.go` 用同样输入调用 `httptest.NewServer` 的真实 handler，按字段/标准错误码比较，不比较 UUID/时钟；MCP 仅比较它实际广告的同名能力。运行 `go test ./agentapi ./internal/server ./cmd/garden-mcp -run 'Conformance' -count=1 -v`，预期所有 fixture 列出 PASS，**不得 `t.Skip` 填绿**。
28. **独立宿主 import smoke**：新增 `garden/agentapi/testdata/consumer/go.mod`、`main.go` 用下列 **本地验证专用** 模块布局（不进入 Vivy，不当作发布路径；真实测试中使用绝对本机路径或在该目录以 `go mod edit -replace` 设置）：

   ```go
   module laputa-consumer-smoke

   go 1.26.4

   require github.com/dashimaki/garden v0.0.0

   replace github.com/dashimaki/garden => ../../..
   replace github.com/dashimaki/laputa => ../../../../laputa
   replace github.com/dashimaki/mentle => ../../../../mentle
   ```

   上述相对路径以 `garden/agentapi/testdata/consumer` 为基点；实现者须用 `python -c 'from pathlib import Path; p=Path("garden/agentapi/testdata/consumer"); print([(str((p/x).resolve()),(p/x).exists()) for x in ("../../..","../../../../laputa","../../../../mentle")])'` 在仓库根读回路径；任一 `False` 必须修正层数再写 go.mod。`main.go` 只 import `github.com/dashimaki/garden/agentapi` 和标准库，构造显式临时目录的离线 Config，调用公开入口 `Open→Bootstrap→Close`（具体构造以 13–16 步收敛后的**真实**签名为准，严禁导入 `garden/internal` 或 HTTP server）。在 consumer 目录 `CGO_ENABLED=0 GOSUMDB=off go test ./... -count=1 && CGO_ENABLED=0 GOSUMDB=off go build ./...`，预期 exit 0；`go list -deps ./...` 不包含 `garden/console`、`garden/internal/server`。同时 `go vet ./...`、`go test -race ./agentapi ./internal/ingest`（Windows Go 环境支持 race 时）应无数据竞态；若 race 不支持必须报告，不得宣称已通过。
29. **README 与事实状态**：改 `mentle/README.md`、`laputa/README.md`、`garden/README.md`（若无则新建）、`docs/bmad/laputa-modular-monolith-2026-09/{README.md,conformance.md}`，写清最小公开 Go API、显式 paths/profile/principal/offline 绑定、session/terminal event 映射、对 Vivy `MemoryPort` 的 3 个调用点（预模型 bootstrap、只读 context candidate、终态事件 capture）、`run_id:event_seq` 规则、关闭顺序、CGO=0 测试命令与未实现能力。禁止写“Vivy 已集成”“模块已发布”。新增 `docs/bmad/laputa-modular-monolith-2026-09/vivy-handoff.md`，只含外部接入示例和真正需要 Vivy 侧修改的 TODO，不修改 Vivy 文件。`git diff --check` 和文档相对链接检查应 exit 0。
30. **端到端本地 gate**：依次在三个模块执行 `CGO_ENABLED=0 GOSUMDB=off go test ./... -count=1`；Garden 前先 `npm install --no-package-lock --ignore-scripts && npm run build`，再 `CGO_ENABLED=0 GOSUMDB=off go test -tags=e2e ./e2e/... -count=1`；consumer 目录 `CGO_ENABLED=0 GOSUMDB=off go build ./...`；再默认 CGO 模式重跑 Mentle/Garden/Laputa；`go vet ./...` 各模块，收集每条命令的 exit code 和首尾 `ok` 行。对 CI 不具备的模型文件不允许将必测用例默默 skip；本地 fixture 不可用时 Gate=BLOCKED。恢复/出错矩阵必须标注 P0/P1/P2 和实际 failing package:case。
31. **清理与审阅**：`git diff --check && git status --short --branch && git diff --stat`，仅本 worktree 源码/测试/文档变化且不存在生成 dist/node_modules/误改锁文件；若必须清理自己生成物，按所有权逐路径删除，绝不 `git clean`。对 `C:/Users/Administrator/Desktop/garden`、旧 SDK worktree、Vivy 重新 `git status --short --branch`，状态应与第 1 步相同。对公开 DTO 的信息泄露、principal/profile 绑定、Persona 写权限、index_jobs 原子性、并发 Close/worker 生命周期做独立 review，发现 P0 则不得宣布完成。
32. **提交/最终报告**：每个垂直切片已独立提交；最后仅提交有证据的文档/fixture，`git log --oneline <baseline>..HEAD`、`git status --short --branch`，工作区须干净，不 push。报告 C01–C08 每项的测试名+状态、MCP 广告工具覆盖清单、CGO=0 和 restart 实测结果、独立 consumer 路径、剩余 C09（仅 Vivy 本人可以完成）及外部发布阻断。若任一 gate 不绿，报告 BLOCKED 和下一条准确修复任务，不把部分功能描述为“所有闭环”。

## Tests / validation rubric

- 每步 RED 的预期：对应新增 `Test...` 明确 FAIL（或新 API 缺失的编译失败），而非模型丢失/路径错误/网络超时；GREEN 的预期：指定 `go test ... -run ... -count=1 -v` 输出 `--- PASS` 与 exit 0，整个受影响模块 `go test ./... -count=1` exit 0，随后提交。路径和包以本节命令为准；遇到不一致先修正计划记录，不靠猜。
- 统一正式 gate：`CGO_ENABLED=0` 的 Mentle/Garden/Laputa 全模块与 Garden e2e、默认 CGO 回归、独立消费者 build/test、C01–C08 conformance、坏模型/坏权限/坏数据拒绝、真正的 canonical 旧文件重开与派生恢复；C09 只交接，不声明 Vivy 集成。
- 旧 SDK dirty worktree 的草稿不计入 PASS；MCP 只保证已注册工具的 conformance，未注册工具不得出现在 capability discovery；纯本地测试通过不等于 `github.com/dashimaki/...` 可远程获取。

## Risks, tradeoffs, open questions

1. **真实最大工程量在 Garden 内部状态库和政策收敛**：多个模块各自用 mattn/`sqlite3`，不能仅修 Mentle；转换成纯 Go 驱动后，PRAGMA、事务锁、时间扫描、备份恢复必须对真实旧文件副本验证。不要把驱动变更包装成“只改 go.mod”。
2. **模型体积/下载/纯 Go 推理**：`hugot.New` 会从 cwd 回退或下载，明确 `RequireLocalModel` 且本地模型 fixture 可用是离线交付门槛。Lexical-only 不得绕过 index_jobs 或制造向量身份假阳性；若该模式不安全，应作为阻断决策上报，而不是隐含 fallback。
3. **HTTP vs in-process 权限**：旧 HTTP 的 loopback 只读豁免属于传输边界，嵌入式调用不能继承；真实主体由受信宿主装配绑定，不能由 prompt/request body 伪造。profile 选择当前单 profile 绑定，不引入自动跨 profile 路由。
4. **Session/capture 语义**：Vivy 的 durable event identity 和 Journal 不归本库创建；本批以 synthetic terminal fixture 保证接口与恢复语义，不能以此替代 Vivy RunHook 的实际接入测验。旧 ingest `precompact/session_end` 与终态必须明确映射，不能“任何事件均捕获”。
5. **发布路径未决**：remote `ProjectViVy/laputa` 与现有 module path `github.com/dashimaki/{garden,laputa,mentle}` 不一致且无 tag；计划仅交付本地三模块 import smoke。远程模块路径重命名、单/多仓库、版本 tag、公开发布为独立 owner 决策，不得暗自完成。
6. **工时拆分**：每个 2–5 分钟步骤只是一次受控编辑/测试循环，不承诺整个项目在分钟级完成。M1/纯 Go/领域入口/适配器是依赖链，不能并行修改同一状态文件或越过未通过的 gate。
