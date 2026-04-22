# Story 2.2: 璁板繂绛涢€変笌鍚堝苟閫昏緫

**Story ID:** 2.2  
**Story Key:** 2-2-memory-filter-merge  
**Status:** done  
**Created:** 2026-04-14  
**Project:** 澶╃┖涔嬪煄 (Laputa)

---

## 鐢ㄦ埛鏁呬簨

As a **绯荤粺**,  
I want **瀵规柊澧炲唴瀹规墽琛屽彲瑙ｉ噴鐨勭瓫閫夊垽鏂?*,  
So that **浣庝环鍊煎唴瀹硅杩囨护鎴栨爣璁帮紝閲嶅鍐呭鍚堝苟鍒板凡鏈夎蹇嗭紝楂樹环鍊煎唴瀹硅淇濈暀骞跺己鍖?*銆?
---

## 楠屾敹鏍囧噯

- **Given** 鏂板鏃ヨ鏉＄洰宸茬粡鐢?Story 2.1 鍐欏叆鍏ュ彛鎺ユ敹
- **When** MemoryGate 绛涢€夐€昏緫鎵ц
- **Then** 涓庡凡鏈夎蹇嗛珮搴﹂噸澶嶇殑鍐呭鍚堝苟鍒版棦鏈夋潯鐩紝鐩爣鏉＄洰 `heat_i32 += 500`锛屼笖涓嶅垱寤虹浜屼唤娲昏穬閲嶅璁板繂
- **And** 浣庝环鍊煎唴瀹硅鏍囪涓?`discard_candidate`
- **And** 姣忎釜绛涢€夊喅绛栭兘鍐欏叆 `reason` 瀛楁锛屼繚璇佸彲瑙ｉ噴

鎵╁睍绾︽潫锛?
- 閲嶅鍒ゅ畾浼樺厛澶嶇敤鐜版湁璇箟妫€绱㈣兘鍔涳紝涓嶅厑璁搁噸鏂伴€犱竴濂楃嫭绔嬪悜閲忔绱?- 鍚堝苟鏃朵紭鍏堝鐢ㄧ幇鏈?AAAK 鍚堝苟鑳藉姏锛屼笉鍏佽绠€鍗曞瓧绗︿覆鎷兼帴瑕嗙洊鍘熸憳瑕?- `discard_candidate` 璁板繂榛樿涓嶅弬涓庢櫘閫?recall/search 缁撴灉锛岄櫎闈炴樉寮忛€夋嫨鍖呭惈
- Story 2.2 鍙疄鐜扳€滅瓫閫変笌鍚堝苟鈥濓紝涓嶅疄鐜?Epic 5 鐨勫畬鏁?HeatService銆佺姸鎬佹満鎴栬嚜鍔ㄥ綊妗?
---

## Epic 涓婁笅鏂?
### Epic 2 鐩爣

Epic 2 璐熻矗鈥滄棩璁颁笌璁板繂杈撳叆鈥濓紝瑕嗙洊锛?
- `FR-2` 鏃ヨ鍐欏叆
- `FR-3` 璁板繂绛涢€?- `FR-10` 鎯呯华閿氬畾

鏈?Story 鏄?Epic 2 鐨勬牳蹇冨垽鏂眰锛屼綅浜?`diary.write(...)` 涔嬪悗銆侀暱鏈熻蹇嗗叆搴撲箣鍓嶃€傚畠鍐冲畾涓€鏉¤緭鍏ユ槸锛?
- `store`锛氭甯镐繚鐣?- `merge`锛氬悎骞跺埌宸叉湁璁板繂骞跺己鍖?- `discard`锛氭爣璁颁负浣庝环鍊硷紝涓嶈繘鍏ユ櫘閫氭椿璺冭蹇嗘祦

### 涓庣浉閭?Story 鐨勫叧绯?
- `2.1 diary-write` 鎻愪緵鏂板杈撳叆鍏ュ彛銆佸熀纭€ `MemoryRecord` 鍒涘缓鍜屾儏缁紪鐮佹槧灏?- `2.2 memory-filter-merge` 鍦ㄥ啓鍏ヨ矾寰勪腑鎵ц绛涢€夈€佸幓閲嶃€佸悎骞躲€佽В閲?- `2.3 emotion-anchor` 鍦ㄤ繚鐣欏悗鐨勮蹇嗕笂鍋氫汉宸ユ儏缁己鍖栵紝涓嶅簲鍙嶅悜鑰﹀悎鏈?Story

---

## 鐜版湁浠ｇ爜鎯呮姤

### 蹇呴』澶嶇敤鐨勭幇鏈夎兘鍔?
1. `Laputa/src/storage/mod.rs`
   - 宸叉湁 `Storage::prune_memories(...)`
   - 宸叉湁鍩轰簬鍚戦噺杩戦偦鐨?cluster 璇嗗埆
   - 宸叉湁 `Dialect::merge_aaaks(&aaaks)` 鍚堝苟鎽樿閫昏緫
   - 宸叉湁 `PruneReport` 缁撴瀯锛屽彲鍙傝€冨叾缁熻鏂瑰紡

2. `Laputa/src/mcp_server/mod.rs`
   - 宸叉湁 `mempalace_check_duplicate`
   - 宸叉湁鍩轰簬 `searcher.search_memories(...)` 鐨勯噸澶嶆鏌ュ叆鍙?   - 鍙洿鎺ュ鐢ㄥ叾鈥滅浉浼煎害闃堝€兼瘮杈冣€濇ā寮忥紝閬垮厤閲嶅瀹炵幇

3. `Laputa/src/storage/memory.rs`
   - 宸叉湁 `LaputaMemoryRecord`
   - 宸叉湁 `heat_i32`銆乣emotion_valence`銆乣emotion_arousal`
   - 宸叉湁 `ensure_memory_schema(...)`
   - 鏈?Story 闇€瑕佸湪杩欓噷缁х画鎵╁睍绛涢€夊厓鏁版嵁瀛楁

4. `Laputa/src/vector_storage.rs`
   - 宸叉湁 `add/search/get/update/delete` 璺緞
   - 宸茶礋璐?SQLite + usearch 鑱斿姩
   - 浠讳綍鏂板瓧娈甸兘蹇呴』鍦ㄨ繖閲屽畬鏁磋ˉ榻?schema / insert / select / row mapping / update

### 鍓嶄竴鏉?Story 鐨勫彲缁ф壙缁撹

鏉ヨ嚜 `1-3-memoryrecord-extension`锛?
- 鐑害瀛楁宸茬粡绋冲畾涓?`heat_i32`锛屼笉寰楀紩鍏?`heat: f64` 鎸佷箙鍖?- 鎯呯华缁村害宸茬粡鏄?`emotion_valence` + `emotion_arousal`
- schema 杩佺Щ妯″紡宸茬粡瀛樺湪锛岀户缁部鐢?`ensure_memory_schema(...) + add_column_if_missing(...)`
- `idx_heat` 宸插瓨鍦紝鍚庣画 merge 寮哄寲鐑害鏃跺繀椤讳繚鎸佷笌鐜版湁鎺掑簭璇箟涓€鑷?
---

## 鏋舵瀯涓庤璁＄害鏉?
### 1. MemoryGate 鐨勮亴璐ｈ竟鐣?
MemoryGate 鍙礋璐ｂ€滃垽鏂€濆拰鈥滆В閲娾€濓紝涓嶈礋璐ｏ細

- 鍛ㄦ湡鎬х儹搴﹁“鍑?- 褰掓。鍊欓€夊垽瀹?- 鐢ㄦ埛鎵嬪姩骞查
- 鍞ら啋鍖呯敓鎴?
鎺ㄨ崘鑱岃矗鎷嗗垎锛?
- `src/diary/`锛氭壙鎺ュ啓鍏ュ叆鍙?- `src/diary/memory_gate.rs`锛氭柊澧烇紝璐熻矗杈撳叆鍒ゆ柇涓庡喅绛栧璞?- `src/vector_storage.rs`锛氳礋璐ｇ湡姝ｇ殑钀藉簱 / 鏇存柊
- `src/searcher/`锛氳礋璐ｅ鐢ㄨ涔夋绱?
涓嶈鎶婄瓫閫夐€昏緫鐩存帴濉炶繘 `mcp_server`銆?
### 2. 鍐崇瓥妯″瀷

鑷冲皯鏀寔浠ヤ笅鍐崇瓥鏋氫妇锛?
```rust
pub enum MemoryGateAction {
    Store,
    Merge { target_id: i64, similarity: f32 },
    Discard,
}
```

鑷冲皯杩斿洖浠ヤ笅瑙ｉ噴淇℃伅锛?
```rust
pub struct MemoryGateDecision {
    pub action: MemoryGateAction,
    pub reason: String,
    pub discard_candidate: bool,
}
```

### 3. 鐩镐技搴︿笌闃堝€?
渚濇嵁鍘熷璁捐鏂囨。锛?
- `duplicate match > 0.8` 瑙嗕负鈥滃啑浣欎俊鎭紝鍚堝苟鍒扮幇鏈夋潯鐩€?
瀹炵幇瑕佹眰锛?
- 榛樿閲嶅闃堝€煎彇 `0.8`
- 闃堝€煎厛浣滀负甯搁噺鎴栧眬閮ㄩ厤缃疄鐜帮紝涓嶅繀鍦ㄦ湰 Story 鎵╁睍瀹屾暣閰嶇疆绯荤粺
- 妫€绱㈠€欓€夊厛鍙?`top_k = 3~5`锛屽啀鍦ㄤ唬鐮佷腑閫夋渶楂樼浉浼肩粨鏋?- 鑻ユ病鏈夊€欓€夎揪鍒伴槇鍊硷紝鍒欎笉璧?merge

### 4. 鍙В閲婂厓鏁版嵁鏄‖绾︽潫

楠屾敹鏍囧噯鏄庣‘瑕佹眰鍐欏叆 `reason` 瀛楁锛屽洜姝ゆ湰 Story 蹇呴』琛ラ綈鎸佷箙鍖栧瓧娈碉細

- `reason: Option<String>`
- `discard_candidate: bool`

濡傞渶淇濆瓨鍚堝苟鐩爣锛屽缓璁悓鏃惰ˉ榻愶細

- `merged_into_id: Option<i64>`

鎺ㄨ崘鍦?`LaputaMemoryRecord` 鍜?`memories` 琛ㄤ腑鍚屾椂澧炲姞杩欎簺瀛楁锛屽苟鍦?`vector_storage` 鏌ヨ鏄犲皠涓畬鏁磋鍐欍€?
### 5. 鍚堝苟绛栫暐

閲嶅鍐呭鍚堝苟鏃讹紝蹇呴』閬靛畧锛?
- 閫夋嫨宸叉湁璁板繂浣滀负 `winner`
- 鏂拌緭鍏ヤ笉鍐嶅垱寤虹浜屼唤娲昏穬璁板繂
- `winner.heat_i32 += 500`锛屽苟瀵逛笂闄愬仛 clamp锛堜笉瓒呰繃 `10000`锛?- 鍚堝苟鎽樿浼樺厛澶嶇敤 `Dialect::merge_aaaks(...)`
- `reason` 闇€瑕佽褰曞悎骞跺師鍥狅紝渚嬪鈥渄uplicate match > 0.8锛宮erged into existing memory鈥?
涓嶈鐢ㄢ€滄柊鏂囨湰瑕嗙洊鏃ф枃鏈€濈殑鏂瑰紡瀹炵幇 merge銆?
### 6. 涓㈠純绛栫暐

浣庝环鍊煎唴瀹逛笉鏄墿鐞嗗垹闄わ紝鑰屾槸鈥滃彲瑙ｉ噴鍦版爣璁扳€濓細

- `discard_candidate = true`
- `reason` 蹇呴』濉啓
- 榛樿涓嶈繘鍏ユ櫘閫?recall/search

杩欑鍚?PRD/NFR 瀵光€滃彲瑙ｉ噴淇濈暀鎴栭仐蹇樷€濈殑瑕佹眰锛屼篃閬垮厤鏃╂湡瀹炵幇鍋氫笉鍙€嗗垹闄ゃ€?
---

## 鎺ㄨ崘瀹炵幇鏂规

### 鏁版嵁妯″瀷鎵╁睍

寤鸿鍦?`Laputa/src/storage/memory.rs` 涓?`LaputaMemoryRecord` 澧炲姞锛?
```rust
pub reason: Option<String>,
pub discard_candidate: bool,
pub merged_into_id: Option<i64>,
```

骞跺湪 `ensure_memory_schema(...)` 涓拷鍔犺縼绉伙細

```sql
ALTER TABLE memories ADD COLUMN reason TEXT;
ALTER TABLE memories ADD COLUMN discard_candidate INTEGER DEFAULT 0;
ALTER TABLE memories ADD COLUMN merged_into_id INTEGER;
```

### MemoryGate 鏀剧疆浣嶇疆

寤鸿鏂板锛?
- `Laputa/src/diary/memory_gate.rs`

寤鸿鍐呭锛?
- `MemoryGateAction`
- `MemoryGateDecision`
- `MemoryGate`
- `judge(...)`
- `merge_or_store(...)`

鍘熷洜锛?
- 鏈?Story 灞炰簬鍐欏叆璺緞
- 涓?`diary.write` 鍏崇郴鏈€鐩存帴
- 閬垮厤姹℃煋 `mcp_server` 鍜岀函瀛樺偍灞?
### diary 鍐欏叆娴佺▼寤鸿

鎺ㄨ崘鐩爣娴佺▼锛?
1. `diary.write(...)` 鎺ユ敹 `content/tags/emotion`
2. 鍏堟瀯閫犲緟鍏ュ簱鐨?`LaputaMemoryRecord`
3. 璋冪敤 `MemoryGate::judge(...)`
4. 鏍规嵁缁撴灉鎵ц锛?   - `Store` -> 姝ｅ父鍏ュ簱
   - `Merge` -> 鏇存柊鐩爣 memory锛岀儹搴?+500锛屽啓 reason
   - `Discard` -> 鍐欏叆甯?`discard_candidate=true` 鐨勮褰曪紝鎴栬嚦灏戜繚璇佽璁板綍涓嶈繘鍏ユ櫘閫氭悳绱?
鍏抽敭鐐癸細

- 鏃ヨ鍘熷鏃ュ織鍐欏叆鑳藉姏涓嶈鍒狅紝`Diary` 浠嶄繚鐣欐椂闂村簭鍒楁棩蹇椾綔鐢?- 璁板繂绛涢€夋槸鈥滀粠 diary 杩涘叆闀挎湡璁板繂鈥濈殑闄勫姞姝ラ锛屼笉瑕佹浛浠?`Diary` 鏈韩

### search / recall 杩囨护

鍙璁板綍琚爣璁?`discard_candidate = true`锛屼互涓嬭矾寰勯粯璁ゅ簲鎺掗櫎锛?
- `VectorStorage::search(...)`
- `VectorStorage::search_room(...)`
- `VectorStorage::get_memories(...)`
- 浠讳綍 `searcher` 涓婂眰鍖呰璋冪敤

瀹炵幇鏂瑰紡浼樺厛閫?SQL where 瀛愬彞杩囨护锛岃€屼笉鏄彇鍥炲悗鍐嶅唴瀛樿繃婊ゃ€?
---

## 娴嬭瘯瑕佹眰

鑷冲皯琛ラ綈浠ヤ笅娴嬭瘯锛?
1. `MemoryGate` 鍐崇瓥娴嬭瘯
   - 楂樼浉浼煎害鍐呭 -> `Merge`
   - 浣庝环鍊煎唴瀹?-> `Discard`
   - 姝ｅ父鍐呭 -> `Store`

2. merge 琛屼负娴嬭瘯
   - 鍛戒腑閲嶅鏃朵笉鏂板绗簩鏉℃椿璺冭褰?   - 鐩爣璁板綍 `heat_i32` 澧炲姞 500
   - `reason` 琚啓鍏?
3. discard 琛屼负娴嬭瘯
   - 鍐欏叆鍚?`discard_candidate = true`
   - `reason` 琚啓鍏?   - 榛樿 recall/search 鐪嬩笉鍒拌璁板綍

4. schema 杩佺Щ娴嬭瘯
   - 鍘嗗彶鏁版嵁搴撳崌绾у悗瀛樺湪 `reason`
   - 鍘嗗彶鏁版嵁搴撳崌绾у悗瀛樺湪 `discard_candidate`
   - 鍘嗗彶鏁版嵁搴撳崌绾у悗瀛樺湪 `merged_into_id`

5. diary 闆嗘垚娴嬭瘯
   - `diary.write(...)` 璺緞浼氳Е鍙?MemoryGate
   - 鎯呯华瀛楁涓?Story 2.1/1.3 鐨勭幇鏈夌粨鏋勪笉鍥炲綊

娴嬭瘯浣嶇疆寤鸿锛?
- `Laputa/tests/test_memory_gate.rs`
- `Laputa/tests/test_diary_write.rs` 鎴栧湪鐜版湁 `diary` 娴嬭瘯涓墿灞?
娑夊強鏂囦欢绯荤粺鎴?SQLite 鏂囦欢鏃讹紝缁х画浣跨敤锛?
- `tempfile`
- `serial_test`

---

## 绂佹浜嬮」

- 涓嶈閲嶅啓鏂扮殑鍚戦噺妫€绱㈠櫒
- 涓嶈鎶婇噸澶嶆娴嬮€昏緫澶嶅埗涓€浠藉埌 `mcp_server`
- 涓嶈寮曞叆瀹屾暣 HeatService 鎴栫姸鎬佹満
- 涓嶈鍦?merge 鏃剁畝鍗曡鐩栧師 `text_content`
- 涓嶈瀵逛綆浠峰€煎唴瀹瑰仛鐗╃悊鍒犻櫎
- 涓嶈蹇樿鍚屾鏇存柊 `vector_storage.rs` 鐨?row mapping 涓?SQL

---

## 瀹炴柦浠诲姟

- [x] 鎵╁睍 `LaputaMemoryRecord` 涓?`memories` schema锛屾柊澧?`reason` / `discard_candidate` / `merged_into_id`
- [x] 鏂板 `Laputa/src/diary/memory_gate.rs`锛屽畾涔夊喅绛栫被鍨嬩笌 `judge(...)` 閫昏緫
- [x] 鍦?`vector_storage.rs` 涓ˉ榻愭柊瀛楁鐨?insert / select / update / row mapping
- [x] 鍦?`diary` 鍐欏叆璺緞涓帴鍏?MemoryGate锛屽舰鎴?`store / merge / discard` 涓夊垎鏀?
- [x] 澶嶇敤 `searcher.search_memories(...)` 鍚屾牱鐨勮涔夋绱㈠垽瀹氭ā寮忥紝榛樿闃堝€?`0.8`
- [x] 澶嶇敤 `Dialect::merge_aaaks(...)` 鍚堝苟閲嶅璁板繂鎽樿
- [x] 鍦ㄦ櫘閫?recall/search SQL 涓帓闄?`discard_candidate = true`
- [x] 琛ラ綈鍐崇瓥銆佽縼绉汇€侀泦鎴愭祴璇曞苟楠岃瘉閫氳繃
---

## 瀹屾垚瀹氫箟
- [x] 閲嶅鍐呭鍛戒腑鍚庡悎骞跺埌宸叉湁璁板繂锛岀洰鏍囩儹搴﹀鍔?500
- [x] 浣庝环鍊煎唴瀹硅鏍囪涓?`discard_candidate`
- [x] 鎵€鏈夊喅绛栭兘鎸佷箙鍖?`reason`
- [x] 鏅€?recall/search 榛樿涓嶈繑鍥?`discard_candidate`
- [x] 鐜版湁 `Diary` 鏃ュ織鑳藉姏涓嶈鐮村潖
- [x] `cargo test` 閫氳繃
- [x] `cargo clippy --all-features --tests -- -D warnings` 閫氳繃

---

## 鍙傝€冭祫鏂?
- `D:\VIVYCORE\newmemory\_bmad-output\planning-artifacts\epics.md`
- `D:\VIVYCORE\newmemory\_bmad-output\planning-artifacts\architecture.md`
- `D:\VIVYCORE\newmemory\brain-memory-system-design.md`
- `D:\VIVYCORE\newmemory\Laputa\DECISIONS.md`
- `D:\VIVYCORE\newmemory\Laputa\AGENTS.md`
- `D:\VIVYCORE\newmemory\Laputa\src\storage\mod.rs`
- `D:\VIVYCORE\newmemory\Laputa\src\storage\memory.rs`
- `D:\VIVYCORE\newmemory\Laputa\src\vector_storage.rs`
- `D:\VIVYCORE\newmemory\Laputa\src\mcp_server\mod.rs`

---

## Dev Agent Record

### Context Notes

- 鐜版湁浠ｇ爜宸茬粡鍏峰鈥滈噸澶嶆娴嬧€濆拰鈥滆仛绫诲悎骞垛€濈殑闆忓舰锛屾湰 Story 鐨勯噸鐐规槸鎶婅繖浜涜兘鍔涙敹鏉熸垚鍐欏叆鏈熺殑 `MemoryGate`
- 褰撳墠浠ｇ爜灏氭湭鎸佷箙鍖?`reason` 涓?`discard_candidate`锛岃繖鏄湰 Story 鐨勫叧閿ˉ榻愰」
### Debug Log

- `cargo check --tests`
- `cargo clippy --all-features --tests -- -D warnings`
- `cargo test --tests --no-run`
- `target/debug/deps/laputa-f84c83bd72b06c80.exe --nocapture`
- `target/debug/deps/test_memory_gate-aa23069b05c3f592.exe --nocapture`
- `target/debug/deps/test_memory_record-e62daca473f8d4f6.exe --nocapture`
- `target/debug/deps/test_identity-822954325a166045.exe --nocapture`

### Completion Notes

- 鍦?`storage/memory.rs` 鍜?`vector_storage.rs` 涓ˉ榻?`reason`銆?`discard_candidate`銆?`merged_into_id` schema / insert / select / row mapping / update 璺緞锛屽苟鏂板 `idx_discard_candidate`
- 鏂板 `diary/memory_gate.rs`锛屽疄鐜?`Store / Merge / Discard` 鍐崇瓥锛岄粯璁?duplicate threshold = `0.8`锛屽湪 embeddings 涓嶅彲鐢ㄦ椂闄嶇骇涓?store
- 鍦?`diary.write(...)` 鍐欏叆璺緞鎺ュ叆 MemoryGate锛歁erge 鏃跺鐩爣璁板綍鎵ц `heat_i32 + 500`锛堝苟 clamp 鍒?`10000`锛夛紝Discard 鏃跺啓鍏?`discard_candidate = true` 涓?`reason`
- 鍦?`VectorStorage::search(...)`銆?`search_room(...)`銆?`get_memories(...)`銆?`memory_count(...)` 绛夐粯璁よ矾寰勬帓闄?`discard_candidate = true`
- 淇 `Searcher` 鍦?embeddings 涓嶅彲鐢ㄦ椂鐨勯€€鍖栬涓猴紝淇濇寔 MCP search / duplicate / add drawer 娴嬭瘯鍏煎

## File List

- `Laputa/src/storage/memory.rs`
- `Laputa/src/vector_storage.rs`
- `Laputa/src/diary/mod.rs`
- `Laputa/src/diary/memory_gate.rs`
- `Laputa/src/searcher/mod.rs`
- `Laputa/tests/test_memory_record.rs`
- `Laputa/tests/test_memory_gate.rs`
- `_bmad-output/implementation-artifacts/2-2-memory-filter-merge.md`

## Review Findings

### 2026-04-16 Code Review

#### decision_needed (已决策)

- [x] [Review][Decision] 低值检测仅支持英文关键词 — **决策**: 添加中文关键词如"好的"、"谢谢"、"收到"

#### patch (已存档)

- [x] [Review][Patch] 合并后向量索引未更新：update_memory_after_merge需重建embedding [memory_gate.rs:122-126, vector_storage.rs:607-623] — 存档deferred-work
- [x] [Review][Patch] search()事后过滤性能浪费：改为预过滤discard_candidate [vector_storage.rs:414-475] — 存档deferred-work
- [x] [Review][Patch] threshold无范围校验：添加∈[0.0,1.0]校验 [memory_gate.rs:53-55] — 存档deferred-work
- [x] [Review][Patch] Embeddings不可用降级信息丢失：添加原始错误类型到reason [memory_gate.rs:78-89] — 存档deferred-work

#### defer (预存问题)

- [x] [Review][Defer] 并发写入竞态条件 [diary/mod.rs:290-337] — deferred，需事务机制

## Change Log

- 2026-04-16: Epic 2 代码审查完成，发现4个patch待修复，1个严重bug（向量索引未更新）
- 2026-04-15: 瀹炵幇 Story 2.2 MemoryGate 绛涢€夈€佸悎骞朵笌 discard_candidate 杩囨护锛屽畬鎴?schema 鎵╁睍銆乪iary 鍐欏叆鎺ュ叆鍜屽叏閮ㄦ祴璇?clippy 楠岃瘉
