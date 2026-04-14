//! SQLite 初始化相关能力。

use crate::api::LaputaError;
use rusqlite::Connection;
use uuid::Uuid;

/// Laputa 版记忆记录结构。
#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
pub struct MemoryRecord {
    pub id: Uuid,
    pub text_content: String,
    pub wing: String,
    pub room: String,
    pub source_file: Option<String>,
    pub valid_from: i64,
    pub valid_to: Option<i64>,
    pub heat_i32: i32,
    pub last_accessed: i64,
    pub access_count: u32,
    pub is_archive_candidate: bool,
    pub emotion_valence: i32,
    pub emotion_arousal: u32,
}

/// 创建 Story 1.2 要求的最小 SQLite schema。
pub fn create_schema(conn: &Connection) -> Result<(), LaputaError> {
    conn.execute_batch(
        r#"
        CREATE TABLE IF NOT EXISTS memories (
            id              TEXT PRIMARY KEY,
            text_content    TEXT NOT NULL,
            wing            TEXT NOT NULL DEFAULT '',
            room            TEXT NOT NULL DEFAULT '',
            source_file     TEXT,
            valid_from      INTEGER NOT NULL,
            valid_to        INTEGER,
            heat_i32        INTEGER NOT NULL DEFAULT 5000,
            last_accessed   INTEGER NOT NULL,
            access_count    INTEGER NOT NULL DEFAULT 0,
            is_archive_candidate INTEGER NOT NULL DEFAULT 0,
            emotion_valence INTEGER NOT NULL DEFAULT 0,
            emotion_arousal INTEGER NOT NULL DEFAULT 0
        );

        CREATE INDEX IF NOT EXISTS idx_memories_valid_from ON memories(valid_from DESC);
        CREATE INDEX IF NOT EXISTS idx_memories_heat ON memories(heat_i32 DESC);
        CREATE INDEX IF NOT EXISTS idx_memories_wing ON memories(wing);
        "#,
    )?;

    Ok(())
}
