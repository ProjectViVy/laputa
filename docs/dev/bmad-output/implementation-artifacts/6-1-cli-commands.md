# Story 6.1: CLI 瀛愬懡浠ゅ疄鐜?
Status: review

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a 鐢ㄦ埛锛?I want 閫氳繃 CLI 瀛愬懡浠ゆ搷浣滆蹇嗗簱锛?so that 鏈湴寮€鍙戝拰娴嬭瘯渚挎嵎銆?
## Acceptance Criteria

1. **Given** CLI 鍏ュ彛 `laputa`
   **When** 鐢ㄦ埛鎵ц瀛愬懡浠わ細
   - `laputa init --name "澶ф箍"`
   - `laputa diary write --content "..." --tags "work"`
   - `laputa recall --time-range "2026-04-01~2026-04-13"`
   - `laputa wakeup`
   - `laputa mark --id <uuid> --important`
   **Then** 鍛戒护姝ｇ‘鎵ц骞惰繑鍥炵粨鏋?2. **And** 閿欒鎯呭喌杩斿洖 `LaputaError`

## Tasks / Subtasks

- [x] Task 1 (AC: 1) - ??? CLI ?????????
  - [x] ??`clap` derive API ??`src/cli/commands.rs` ?????? `Cli` ??`Commands`
  - [x] ??`src/main.rs` ??`Cli::parse()` ???????????????????????
  - [x] ?????????????????????`init`??diary write`??recall`??wakeup`??mark`
- [x] Task 2 (AC: 1, 2) - ???????????
  - [x] ??`src/cli/handlers.rs` ???????????????????????`Result<_, LaputaError>`
  - [x] `init` ??? `IdentityInitializer`
  - [x] `wakeup` ??? `Searcher::wake_up()`
  - [x] `recall` ?????? `MemoryStack` / `Searcher` ????????CLI ???????????
- [x] Task 3 (AC: 1) - ??? `diary write` ????????
  - [x] ??? `content`??tags` ?????
  - [x] ???????????? `diary` ????????????
  - [x] ??`src/cli/output.rs` ?????????????????
- [x] Task 4 (AC: 1) - ??? `mark` ?????Phase 1 ???
  - [x] ??? `--important`????????`--forget` / `--emotion-anchor` ??????????
  - [x] ?????? epic ?????? `<uuid>`?????????????????`i64 memory_id` ???????
  - [x] ????????????????????????????????? handler ???????????Story 5.3 ???
- [x] Task 5 (AC: 2) - ?????????
  - [x] CLI ????????????????? `Result<T, LaputaError>`
  - [x] ??`anyhow` / `rusqlite` / ??????????????`LaputaError`
  - [x] ??????????stderr ????????????????? `panic!`
- [x] Task 6 (AC: 1, 2) - ??? CLI ???
  - [x] ??? `tests/test_cli_flow.rs`
  - [x] ??? `init`??diary write`??recall`??wakeup` ????????
  - [x] ???????????????????ID??????????????????

## Dev Notes

### 瀹炵幇鐩爣

  - `commands.rs` 璐熻矗鍙傛暟妯″瀷
  - `handlers.rs` 璐熻矗涓氬姟璋冪敤
  - `output.rs` 璐熻矗缁熶竴杈撳嚭
- 杩欐牱 Story 6.2 鐨?MCP Tools 鎵嶈兘澶嶇敤鐩稿悓鐨勬牳蹇冭兘鍔涳紝鑰屼笉鏄啀鍐欎竴濂楀钩琛屽疄鐜般€?
### 褰撳墠浠ｇ爜鐜扮姸

- `Laputa/src/main.rs` 鐩墠鍙緭鍑?`Laputa v0.1.0`銆?- `Laputa/src/cli/mod.rs` 鐩墠鍙湁妯″潡娉ㄩ噴锛屾病鏈夊懡浠ゆ爲銆乭andler 鎴栬緭鍑烘牸寮忓疄鐜般€?- 宸叉湁鍙鐢ㄨ兘鍔涳細
  - `Laputa/src/identity/initializer.rs` 宸茶兘鍒濆鍖?`laputa.db` 涓?`identity.md`
  - `Laputa/src/diary/mod.rs` 宸茶兘鍐欏叆 / 璇诲彇鏃ヨ
  - `Laputa/src/searcher/mod.rs` 宸叉湁 `search()`銆乣search_memories()`銆乣wake_up()`
  - `Laputa/src/storage/mod.rs` 宸叉湁 `recall()` / `search()` / `wake_up()` 璺緞
- 缁熶竴閿欒绫诲瀷宸插瓨鍦ㄤ簬 `Laputa/src/api/error.rs`锛歚LaputaError`

### 鍏抽敭瀹炵幇椋庨櫓

1. **ID 绫诲瀷涓嶄竴鑷?*
   - Epic 6 鐨?AC 鐢ㄧ殑鏄?`laputa mark --id <uuid>`
   - 褰撳墠浠撳簱瀛樺湪涓ゅ ID 璇箟锛?     - `Laputa/src/storage/sqlite.rs` 閲岀殑 `MemoryRecord.id` 鏄?`Uuid`
     - `Laputa/src/storage/memory.rs`銆乣vector_storage.rs`銆乣searcher.rs`銆乣mcp_server/mod.rs` 褰撳墠涓昏浣跨敤 `i64 memory_id`
   - 鏈晠浜嬪繀椤绘槑纭竴涓复鏃惰惤鍦扮瓥鐣ワ紝閬垮厤 CLI 灞傚啀鍒堕€犵涓夊 ID 瑙勫垯
   - 鎺ㄨ崘锛歅hase 1 CLI 鍏堜笌褰撳墠杩愯鏃朵富璺緞瀵归綈锛屼紭鍏堟敮鎸?`i64`锛屽苟鍦?story 涓樉寮忚褰?UUID 鏀舵暃鐣欑粰鍚庣画閲嶆瀯

2. **CLI 灞備笉鑳介噸鍐欎笟鍔?*
   - `init` 搴旂洿鎺ヨ皟鐢?`IdentityInitializer`
   - `wakeup` 搴旇皟鐢?`Searcher::wake_up()`
   - `recall` 搴斿熀浜庣幇鏈?search / recall 鑳藉姏缁勮鍙傛暟
   - `mark` 鐨勭姸鎬佹洿鏂伴€昏緫鏈€缁堝簲澶嶇敤 Story 5.3锛岃€屼笉鏄湪 CLI 閲岀洿鎺ユ敼鏁版嵁搴?
3. **褰撳墠 diary 鎺ュ彛涓?Epic 鍛戒护绀轰緥骞朵笉瀹屽叏瀵归綈**
   - Epic 涓ず渚嬫槸 `diary write --content --tags`
   - 鐜版湁 `Diary::write_entry(agent, content)` 娌℃湁 `tags` 鍙傛暟
   - 闇€瑕佸湪鏈晠浜嬮噷鏄庣‘锛欳LI 鍙厛瑙ｆ瀽 `tags`锛屽啀鍐冲畾鏄啓鍏ュ唴瀹瑰墠缂€銆佹墿灞?diary 鎺ュ彛锛屾垨涓哄悗缁瓫閫夋ā鍧椾繚鐣欑粨鏋?   - 涓嶈兘鍋囪 `tags` 宸茬粡鏈夊簳灞傝惤鐐?
### 鏋舵瀯绾︽潫

1. **鐩綍缁撴瀯**
   - CLI 鏂囦欢蹇呴』钀藉湪锛?     - `Laputa/src/cli/mod.rs`
     - `Laputa/src/cli/commands.rs`
     - `Laputa/src/cli/handlers.rs`
     - `Laputa/src/cli/output.rs`
   - 娴嬭瘯鏂囦欢寤鸿浣跨敤 `Laputa/tests/test_cli_flow.rs`

2. **缁熶竴鎺ュ彛杈圭晫**
   - 鏋舵瀯鏂囨。瑕佹眰 `CLI -> Core` 閫氳繃 Rust 鍑芥暟璋冪敤杩涘叆鏍稿績閫昏緫
   - 涓暱鏈熺洰鏍囨槸鏀舵暃鍒?`MemoryOperation trait`
   - 浣嗗綋鍓嶄粨搴撳皻鏈瓨鍦?`api/operation.rs`锛屾墍浠ユ湰鏁呬簨涓嶈姹傚厛鎶?trait 琛ュ叏鍐嶅仛 CLI
   - 鐜板疄鍋氭硶搴旀槸锛氬厛閫氳繃 handlers 缁熶竴璋冪敤鐜版湁妯″潡锛岀粰鍚庣画 trait 鏀舵暃鐣欐帴鍙ｉ潰

3. **閿欒澶勭悊**
   - 鎵€鏈夊叕鍏?handler 杩斿洖 `Result<_, LaputaError>`
   - `main.rs` 璐熻矗灏嗛敊璇浆涓?stderr + 鍚堢悊閫€鍑虹爜
   - 绂佹鍦?CLI 灞備娇鐢?`unwrap()` / `expect()` 澶勭悊鐢ㄦ埛杈撳叆

4. **鍛藉悕涓庡弬鏁伴鏍?*
   - 鍛戒护銆佸弬鏁般€丣SON/MCP 鍙傛暟鍚嶉兘閬靛惊 `snake_case`
   - CLI 瀛愬懡浠ゆ樉绀哄悕鎸変骇鍝佸畾涔変繚鐣欑煭鍛戒护璇嶏細`init`銆乣diary`銆乣recall`銆乣wakeup`銆乣mark`

### 鍛戒护瀹炵幇寤鸿

- 椤跺眰鍛戒护鏍戝缓璁涓嬶細

```text
laputa
鈹溾攢鈹€ init --name <STRING>
鈹溾攢鈹€ diary
鈹?  鈹斺攢鈹€ write --content <STRING> [--tags <CSV>]
鈹溾攢鈹€ recall --time-range <START~END>
鈹溾攢鈹€ wakeup [--wing <STRING>]
鈹斺攢鈹€ mark --id <ID> --important
```

- `init`
  - 璋冪敤 `IdentityInitializer::initialize`
  - 杈撳嚭鏁版嵁搴撹矾寰勬垨鍒濆鍖栫粨鏋滄憳瑕?
- `diary write`
  - 璋冪敤鐜版湁 `Diary` 璺緞
  - 鑻ュ綋鍓嶅簳灞傚皻涓嶆敮鎸?tags锛孋LI 搴旀槑纭妸 tags 浣滀负 Phase 1 杈撳叆淇濈暀锛屼笉瑕侀潤榛樺悶鎺?
- `recall`
  - AC 缁欑殑鏄?`time-range`
  - 褰撳墠 repo 娌℃湁鐜版垚鐨?`recall --time-range` CLI 瑙ｆ瀽鍣紝闇€瑕佸湪 CLI 灞傝В鏋?`"start~end"`
  - 瑙ｆ瀽瀹屾垚鍚庤皟鐢ㄧ幇鏈?recall/search 璺緞锛涗笉瑕佸湪 CLI 灞傜洿鎺ュ啓 SQL

- `wakeup`
  - 鐩存帴澶嶇敤 `Searcher::wake_up`
  - 褰撳墠 `wakeup` 妯″潡杩樻槸鍗犱綅锛汣LI 浠嶅簲閫氳繃鐜版湁鍙敤鍏ュ彛宸ヤ綔

- `mark`
  - 杩欎釜鏁呬簨鍙渶鎶婂懡浠ゅ３銆佸弬鏁版牎楠屻€侀敊璇涔夊拰 handler 杈圭晫寤虹珛璧锋潵
  - 鍏蜂綋鈥滈噸瑕?閬楀繕/鎯呯华閿氱偣鈥濋€昏緫浠ュ悗缁?Story 5.3 涓轰富
  - 濡傛灉 `--important` 灏氭湭鍙墽琛屽埌搴曞眰锛屽簲杩斿洖娓呮櫚鐨?`ValidationError` / `NotFound` / `ConfigError`锛岃€屼笉鏄吉鎴愬姛

### 杈撳嚭涓?UX 瑕佹眰

- 鎴愬姛杈撳嚭搴旂畝鐭彲璇伙紝閫傚悎缁堢鐢ㄦ埛锛?  - 鍒濆鍖栨垚鍔燂細杩斿洖 db 璺緞
  - 鍐欏叆鎴愬姛锛氳繑鍥?entry id / memory id
  - recall / wakeup锛氳繑鍥炴鏂囨垨缁撴瀯鍖栨憳瑕?- 閿欒杈撳嚭蹇呴』璧?stderr锛屽苟鍖呭惈鐢ㄦ埛鍙悊瑙ｇ殑閿欒淇℃伅
- 涓嶈姹傛湰鏁呬簨寮曞叆褰╄壊缁堢鎴栧鏉傝〃鏍艰緭鍑猴紱鍏堜繚璇佷竴鑷存€у拰鍙祴璇曟€?
### 娴嬭瘯瑕佹眰

- 鏂板 `Laputa/tests/test_cli_flow.rs`
- 浼樺厛瑕嗙洊锛?  - `laputa init --name`
  - `laputa diary write --content`
  - `laputa wakeup`
  - `laputa recall --time-range`
  - `laputa mark --id ... --important`
- 澶辫触璺緞鑷冲皯瑕嗙洊锛?  - 缂哄皯蹇呴』鍙傛暟
  - 鏃犳晥 `time-range` 鏍煎紡
  - 鏈垵濮嬪寲鐩綍鎵ц鍛戒护
  - 鏃犳晥 / 涓嶅瓨鍦ㄧ殑 memory id
- 娑夊強鏂囦欢绯荤粺涓庢湰鍦版暟鎹簱鐨勬祴璇曞簲浣跨敤 `tempdir`
- 鑻ユ祴璇曞叡浜幆澧冨彉閲忔垨 HOME 璺緞锛屽缓璁厤鍚?`serial_test`

### 浠ｇ爜钀界偣寤鸿

```text
Laputa/
鈹溾攢鈹€ src/
鈹?  鈹溾攢鈹€ main.rs                # [MODIFY] CLI entry point
鈹?  鈹溾攢鈹€ cli/
鈹?  鈹?  鈹溾攢鈹€ mod.rs             # [MODIFY] 瀵煎嚭 CLI 妯″潡
鈹?  鈹?  鈹溾攢鈹€ commands.rs        # [NEW] clap 鍛戒护妯″瀷
鈹?  鈹?  鈹溾攢鈹€ handlers.rs        # [NEW] 鍛戒护澶勭悊鍣?鈹?  鈹?  鈹斺攢鈹€ output.rs          # [NEW] 杈撳嚭鏍煎紡
鈹?  鈹溾攢鈹€ identity/
鈹?  鈹?  鈹斺攢鈹€ initializer.rs     # [EXISTING] init 澶嶇敤
鈹?  鈹溾攢鈹€ diary/
鈹?  鈹?  鈹斺攢鈹€ mod.rs             # [EXISTING] diary write/read 澶嶇敤
鈹?  鈹溾攢鈹€ searcher/
鈹?  鈹?  鈹斺攢鈹€ mod.rs             # [EXISTING] wakeup/search 澶嶇敤
鈹?  鈹斺攢鈹€ api/
鈹?      鈹斺攢鈹€ error.rs           # [EXISTING] LaputaError
鈹斺攢鈹€ tests/
    鈹斺攢鈹€ test_cli_flow.rs       # [NEW] CLI 榛勯噾璺緞涓庨敊璇矾寰勬祴璇?```

### 鏈€鏂颁緷璧栦笌鎶€鏈俊鎭?
- 鏈粨搴撳綋鍓?CLI 渚濊禆宸查攣瀹氾細`clap = { version = "4.6.0", features = ["derive"] }`
- 鏍稿 clap 瀹樻柟鏂囨。鍚庡彲纭锛?  - 褰撳墠鎺ㄨ崘妯″紡鏄?derive API
  - `#[derive(Parser)]` + `#[derive(Subcommand)]` 閫傚悎澶氬眰瀛愬懡浠?  - 瀛愬懡浠ら€氳繃 `#[command(subcommand)]` 缁勭粐
- 瀵规湰鏁呬簨鐨勮姹傛槸锛?*娌跨敤褰撳墠浠撳簱 clap 鐗堟湰锛屼笉鍗囩骇渚濊禆**
- 鍙傝€冿細
  - `clap` derive 鏁欑▼涓庡瓙鍛戒护鏂囨。
  - docs.rs 涓婄殑鏈€鏂?`clap` derive API

### Project Structure Notes

- 蹇呴』閬靛畧 `AGENTS.md` 鐨勨€滄墿灞?mempalace-rs锛岃€屼笉鏄彟璧峰簳搴р€濈殑鍘熷垯銆?- CLI 鍙槸澶栭儴鎺ュ彛灞傦紝涓嶆槸鏂扮殑鏍稿績瀛樺偍灞傘€?- 褰撳墠 `Laputa` 鐩綍涓嶆槸 git 浠撳簱鏍癸紝鏃犳硶琛ュ厖鏈€杩戞彁浜ゆā寮忥紱瀹炵幇浠ョ幇鏈変唬鐮佷笌鏋舵瀯鏂囨。涓哄噯銆?
### References

- [Source: `_bmad-output/planning-artifacts/epics.md` - Epic 6 / Story 6.1]
- [Source: `_bmad-output/planning-artifacts/prd.md` - FR-15]
- [Source: `_bmad-output/planning-artifacts/architecture.md` - ADR-009 API 璁捐妯″紡]
- [Source: `_bmad-output/planning-artifacts/architecture.md` - ADR-010 閿欒澶勭悊鏍囧噯]
- [Source: `_bmad-output/planning-artifacts/architecture.md` - 5.1 鍛藉悕妯″紡]
- [Source: `_bmad-output/planning-artifacts/architecture.md` - 5.2 缁撴瀯妯″紡]
- [Source: `_bmad-output/planning-artifacts/architecture.md` - 6.2 瀹屾暣椤圭洰鐩綍缁撴瀯]
- [Source: `_bmad-output/planning-artifacts/architecture.md` - 6.3 鏋舵瀯杈圭晫瀹氫箟]
- [Source: `Laputa/src/main.rs`]
- [Source: `Laputa/src/cli/mod.rs`]
- [Source: `Laputa/src/identity/initializer.rs`]
- [Source: `Laputa/src/diary/mod.rs`]
- [Source: `Laputa/src/searcher/mod.rs`]
- [Source: `Laputa/src/storage/sqlite.rs`]
- [Source: `Laputa/src/storage/memory.rs`]
- [Source: `Laputa/src/api/error.rs`]
- [Source: `Laputa/tests/test_identity.rs`]
- [Source: `https://docs.rs/clap/latest/clap/_derive/_tutorial/`]
- [Source: `https://docs.rs/clap/latest/clap/_derive/`]

## Dev Agent Record

### Agent Model Used

GPT-5

### Debug Log References

- `cargo fmt`
- `cargo test --test test_cli_mark`
- `cargo test --test test_cli_flow`
- `cargo test`

### Completion Notes List

- ????????CLI ?????? `src/cli/commands.rs` ??? `init`??diary write`??recall`??wakeup`??mark` ????????? `src/cli/handlers.rs` ??????????????? `src/cli/output.rs` ???????????
- `init` ??? `IdentityInitializer`??`diary write` ??? `Diary`??`recall` / `wakeup` ??? `Searcher`??`mark` ??? `VectorStorage::apply_intervention`??? CLI ???????????
- ??`mark --id` ??? Phase 1 ????????????????`memory_id`??? UUID ????????? `ValidationError`?????? Story 5.3 / ID ???????????
- ?????CLI ???????????????????????????????????????????ime-range ???????????ID ???????????? `cargo test` ???????

## Change Log

- 2026-04-15: ??? Story 6.1 CLI ???????????????????????????? `review`

### File List

- `_bmad-output/implementation-artifacts/6-1-cli-commands.md`
- `_bmad-output/implementation-artifacts/sprint-status.yaml`
- `Laputa/src/cli/mod.rs`
- `Laputa/src/cli/commands.rs`
- `Laputa/src/cli/handlers.rs`
- `Laputa/src/cli/output.rs`
- `Laputa/src/main.rs`
- `Laputa/tests/test_cli_flow.rs`
- `Laputa/tests/test_cli_mark.rs`
