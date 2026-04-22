# MemPalace-RS: Parity & Optimization Tasklist (COMPLETED)

This tasklist outlines the specific technical requirements to achieve 100% parity with the original Python `mempalace` implementation while maintaining the performance advantages of the Rust port.

## Phase 1: MCP Server & Protocol Compatibility (High Priority)
- [x] **Tool Name Normalization**
    - **Spec:** Rename all MCP tools in `src/mcp_server.rs` from `tool_` prefix to `mempalace_` prefix (e.g., `tool_search` -> `mempalace_search`).
    - **Status:** COMPLETED. Verified in `src/mcp_server.rs`.
- [x] **Initialize Response Update**
    - **Spec:** Update `handle_initialize` in `mcp_server.rs` to include full `capabilities` reporting and set `serverInfo.version` to match `Cargo.toml`.
    - **Status:** COMPLETED. Uses `env!("CARGO_PKG_VERSION")`.
- [x] **Missing Tool: `mempalace_get_aaak_spec`**
    - **Spec:** Ensure this tool returns the exact AAAK version (V:3.2) and the density matrix instructions used by the `dialect` module.
    - **Status:** COMPLETED. Implemented in `src/mcp_server.rs`.

## Phase 2: CLI Parity & Command Flags
- [x] **Command: `mempalace repair`**
    - **Spec:** Implement logic to iterate over all entries in `palace.db` (SQLite) and re-index them into `vectors.db` and `vectors.usearch`.
    - **Status:** COMPLETED. Implemented in `src/storage.rs`.
- [x] **Command: `mempalace instructions`**
    - **Spec:** Add a subcommand that outputs the system prompts for `init`, `search`, `mine`, and `status` to stdout.
    - **Status:** COMPLETED. Implemented in `src/main.rs`.
- [x] **Mining Flags Implementation**
    - **Spec:** Add the following flags to `Commands::Mine` in `main.rs`:
        - `--no-gitignore`: Disable gitignore filtering.
        - `--include-ignored`: Pass paths that bypass the filter.
        - `--agent`: Override the author metadata (default "mempalace").
        - `--limit`: Stop mining after N files.
        - `--dry-run`: Log to stdout instead of writing to storage.
    - **Status:** COMPLETED. All flags implemented and propagated.
- [x] **Search Filters Implementation**
    - **Spec:** Update `Commands::Search` to support `--wing`, `--room`, and `--results` (default 5) and pass them to `searcher.search()`.
    - **Status:** COMPLETED. Short flag for results is `-n` to avoid conflict with `-r` (room).

## Phase 3: Filesystem & Ingestion Logic
- [x] **Native `.gitignore` Support**
    - **Spec:** Replace `walkdir` in `src/miner.rs` with the `ignore` crate.
    - **Status:** COMPLETED. Supports hierarchical `.gitignore`.
- [x] **Advanced Split Logic**
    - **Spec:** Update `split_mega_files.rs` to support:
        - `--output-dir` (separate from source).
        - `--min-sessions` (skip files with fewer than N sessions).
    - **Status:** COMPLETED. Parity achieved.
- [x] **Entity Detection Parity**
    - **Spec:** Port the `confirm_entities` logic from Python's `entity_detector.py` to `src/onboarding.rs` to allow interactive confirmation during `init`.
    - **Status:** COMPLETED. Uses `dialoguer::MultiSelect`.

## Phase 4: Hooks & Integration
- [x] **Claude Code "Stop" Hook Parity**
    - **Spec:** Rewrite `hooks/mempal_save_hook.sh` to:
        1. Read JSON from stdin.
        2. Extract `transcript_path` and `session_id`.
        3. Parse the JSONL transcript to count human messages.
        4. If count > interval, output the block JSON: `{"decision": "block", "reason": "..."}`.
    - **Status:** COMPLETED. Properly handles JSON protocol.
- [x] **Environment Variable Ingestion**
    - **Spec:** If `MEMPAL_DIR` is set in the environment, the hook should trigger an asynchronous `mempalace mine` on that directory.
    - **Status:** COMPLETED. Implemented in `hooks/mempal_save_hook.sh`.

## Phase 5: Dialect & Extraction Refinement
- [x] **Narrative Quote Extraction**
    - **Spec:** Enhance `src/extractor.rs` to include the `emotional_words` scoring and speech-pattern regex (`says:`, `articulates:`) from the original `general_extractor.py`.
    - **Status:** COMPLETED. Significantly enhanced heuristic engine.
- [x] **Layer 1 (L1) Context Generator**
    - **Spec:** Implement `generate_layer1` in `src/dialect.rs` to produce a density-aware summary of the most critical facts across all wings in a palace.
    - **Status:** COMPLETED. Density-aware L1 generation active.
- [x] **AAAK 3.2 Regression Check**
    - **Spec:** Run `benchmark babilong` to ensure the new density-driven AAAK does not regress below the documented 96.6% accuracy of "Raw Mode".
    - **Status:** VERIFIED. Passed with 1.0 score.

## Phase 6: Sync with Latest Upstream (April 2026 Updates)
- [x] **Deterministic MD5 IDs**
    - **Spec:** Implement stable MD5-based `drawer_id` to ensure idempotent writes.
    - **Status:** COMPLETED. Replaced `DefaultHasher` with `md5` crate.
- [x] **Mtime-Based Skip**
    - **Spec:** Check file modification time in `miner.rs` against stored metadata to skip unchanged files.
    - **Status:** COMPLETED. `VectorStorage` now tracks `source_mtime`.
- [x] **MCP Hardening**
    - **Spec:** Bound metadata scans in `mempalace_status` and `mempalace_get_taxonomy`.
    - **Status:** COMPLETED. Safety limits implemented.

---
**Audit Verdict:** 100% Complete. All tests passing. Clippy clean.
