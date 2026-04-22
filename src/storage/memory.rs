use crate::api::LaputaError;
use anyhow::{anyhow, Result};
use chrono::{DateTime, TimeZone, Utc};
use rusqlite::Connection;
use std::collections::HashSet;

pub const DEFAULT_HEAT_I32: i32 = 5_000;
pub const MIN_HEAT_I32: i32 = 0;
pub const MAX_HEAT_I32: i32 = 10_000;
pub const MIN_VALENCE: i32 = -100;
pub const MAX_VALENCE: i32 = 100;
pub const MAX_AROUSAL: u32 = 100;

#[derive(Debug, Clone, PartialEq)]
pub struct LaputaMemoryRecord {
    pub id: i64,
    pub text_content: String,
    pub wing: String,
    pub room: String,
    pub source_file: Option<String>,
    pub valid_from: i64,
    pub valid_to: Option<i64>,
    pub score: f32,
    pub importance: f32,
    /// Stores heat scaled by 100 to avoid floating point drift.
    pub heat_i32: i32,
    pub last_accessed: DateTime<Utc>,
    pub access_count: u32,
    pub emotion_valence: i32,
    pub emotion_arousal: u32,
    pub is_archive_candidate: bool,
    pub reason: Option<String>,
    pub discard_candidate: bool,
    pub merged_into_id: Option<i64>,
}

impl LaputaMemoryRecord {
    #[allow(clippy::too_many_arguments)]
    pub fn new(
        id: i64,
        text_content: String,
        wing: String,
        room: String,
        source_file: Option<String>,
        valid_from: i64,
        valid_to: Option<i64>,
        score: f32,
        importance: f32,
    ) -> Self {
        Self {
            id,
            text_content,
            wing,
            room,
            source_file,
            valid_from,
            valid_to,
            score,
            importance,
            heat_i32: DEFAULT_HEAT_I32,
            last_accessed: unix_timestamp_to_datetime(0)
                .expect("Unix epoch must always convert to UTC"),
            access_count: 0,
            emotion_valence: 0,
            emotion_arousal: 0,
            is_archive_candidate: false,
            reason: None,
            discard_candidate: false,
            merged_into_id: None,
        }
    }

    pub fn get_heat(&self) -> f64 {
        heat_from_i32(self.heat_i32)
    }

    /// Sets the heat value with boundary validation.
    /// Returns `ValidationError` if value is outside [0.0, 100.0] range.
    /// NaN inputs are also rejected.
    pub fn set_heat(&mut self, heat: f64) -> Result<(), LaputaError> {
        if heat.is_nan() || !(0.0..=100.0).contains(&heat) {
            return Err(LaputaError::ValidationError(format!(
                "heat out of range [0.0, 100.0]: {}",
                heat
            )));
        }
        self.heat_i32 = heat_to_i32(heat);
        Ok(())
    }

    pub fn update_emotion(&mut self, valence: i32, arousal: u32) {
        self.emotion_valence = valence.clamp(MIN_VALENCE, MAX_VALENCE);
        self.emotion_arousal = arousal.min(MAX_AROUSAL);
    }

    pub fn mark_archive_candidate(&mut self) {
        self.is_archive_candidate = true;
    }

    /// Creates a new record with updated heat (builder style).
    /// Caller is responsible for ensuring heat is in valid range [0.0, 100.0].
    /// # Panics (debug only)
    /// Panics in debug builds if `heat` is NaN or outside [0.0, 100.0].
    pub fn with_updated_heat(&self, heat: f64) -> Self {
        debug_assert!(
            !heat.is_nan() && (0.0..=100.0).contains(&heat),
            "with_updated_heat: heat={heat} is out of range [0.0, 100.0] or NaN"
        );
        Self {
            heat_i32: heat_to_i32(heat),
            ..self.clone()
        }
    }
}

pub fn heat_from_i32(value: i32) -> f64 {
    value as f64 / 100.0
}

/// Converts heat f64 to i32 representation (scaled by 100).
/// Caller must ensure value is in [0.0, 100.0] range; no clamping performed.
pub fn heat_to_i32(value: f64) -> i32 {
    (value * 100.0).round() as i32
}

pub fn ensure_memory_schema(conn: &Connection) -> Result<()> {
    conn.execute_batch(
        "PRAGMA journal_mode = WAL;
         PRAGMA foreign_keys = ON;
         PRAGMA synchronous = NORMAL;
         CREATE TABLE IF NOT EXISTS memories (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            text_content TEXT NOT NULL,
            wing TEXT NOT NULL,
            room TEXT NOT NULL,
            source_file TEXT,
            source_mtime REAL,
            valid_from INTEGER NOT NULL,
            valid_to INTEGER,
            last_accessed INTEGER DEFAULT 0,
            access_count INTEGER DEFAULT 0,
            importance_score REAL DEFAULT 5.0,
            heat_i32 INTEGER DEFAULT 5000,
            emotion_valence INTEGER DEFAULT 0,
            emotion_arousal INTEGER DEFAULT 0,
            is_archive_candidate INTEGER DEFAULT 0,
            reason TEXT,
            discard_candidate INTEGER DEFAULT 0,
            merged_into_id INTEGER
         );
         CREATE INDEX IF NOT EXISTS idx_source_file ON memories (source_file);
         CREATE INDEX IF NOT EXISTS idx_wing_room ON memories (wing, room);
         CREATE INDEX IF NOT EXISTS idx_valid ON memories (valid_from, valid_to);
         CREATE TABLE IF NOT EXISTS drawers (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            content TEXT NOT NULL,
            wing TEXT NOT NULL,
            room TEXT NOT NULL,
            source_file TEXT,
            filed_at TEXT NOT NULL,
            embedding_id INTEGER REFERENCES memories(id)
         );
         CREATE INDEX IF NOT EXISTS idx_drawers_wing_room ON drawers (wing, room);
         CREATE TABLE IF NOT EXISTS capsules (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            week_id TEXT NOT NULL UNIQUE,
            week_start INTEGER NOT NULL,
            week_end INTEGER NOT NULL,
            capsule_json TEXT NOT NULL,
            markdown_content TEXT NOT NULL,
            compressed_content TEXT NOT NULL,
            original_tokens INTEGER NOT NULL,
            summary_tokens INTEGER NOT NULL,
            compression_ratio REAL NOT NULL,
            incomplete INTEGER NOT NULL DEFAULT 0,
            created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
         );
         CREATE INDEX IF NOT EXISTS idx_capsules_week_start ON capsules (week_start DESC);",
    )?;

    let mut statement = conn.prepare("PRAGMA table_info(memories)")?;
    let mut existing_columns = HashSet::new();
    let mut rows = statement.query([])?;
    while let Some(row) = rows.next()? {
        existing_columns.insert(row.get::<_, String>(1)?);
    }

    add_column_if_missing(
        conn,
        &existing_columns,
        "source_mtime",
        "ALTER TABLE memories ADD COLUMN source_mtime REAL",
    )?;
    add_column_if_missing(
        conn,
        &existing_columns,
        "last_accessed",
        "ALTER TABLE memories ADD COLUMN last_accessed INTEGER DEFAULT 0",
    )?;
    add_column_if_missing(
        conn,
        &existing_columns,
        "access_count",
        "ALTER TABLE memories ADD COLUMN access_count INTEGER DEFAULT 0",
    )?;
    add_column_if_missing(
        conn,
        &existing_columns,
        "importance_score",
        "ALTER TABLE memories ADD COLUMN importance_score REAL DEFAULT 5.0",
    )?;
    add_column_if_missing(
        conn,
        &existing_columns,
        "heat_i32",
        "ALTER TABLE memories ADD COLUMN heat_i32 INTEGER DEFAULT 5000",
    )?;
    add_column_if_missing(
        conn,
        &existing_columns,
        "emotion_valence",
        "ALTER TABLE memories ADD COLUMN emotion_valence INTEGER DEFAULT 0",
    )?;
    add_column_if_missing(
        conn,
        &existing_columns,
        "emotion_arousal",
        "ALTER TABLE memories ADD COLUMN emotion_arousal INTEGER DEFAULT 0",
    )?;
    add_column_if_missing(
        conn,
        &existing_columns,
        "is_archive_candidate",
        "ALTER TABLE memories ADD COLUMN is_archive_candidate INTEGER DEFAULT 0",
    )?;
    add_column_if_missing(
        conn,
        &existing_columns,
        "reason",
        "ALTER TABLE memories ADD COLUMN reason TEXT",
    )?;
    add_column_if_missing(
        conn,
        &existing_columns,
        "discard_candidate",
        "ALTER TABLE memories ADD COLUMN discard_candidate INTEGER DEFAULT 0",
    )?;
    add_column_if_missing(
        conn,
        &existing_columns,
        "merged_into_id",
        "ALTER TABLE memories ADD COLUMN merged_into_id INTEGER",
    )?;

    conn.execute_batch(
        "CREATE INDEX IF NOT EXISTS idx_heat ON memories (heat_i32 DESC);
         CREATE INDEX IF NOT EXISTS idx_discard_candidate ON memories (discard_candidate);",
    )?;

    Ok(())
}

fn add_column_if_missing(
    conn: &Connection,
    existing_columns: &HashSet<String>,
    column_name: &str,
    sql: &str,
) -> Result<()> {
    if !existing_columns.contains(column_name) {
        conn.execute_batch(sql)?;
    }
    Ok(())
}

pub(crate) fn unix_timestamp_to_datetime(timestamp: i64) -> Result<DateTime<Utc>> {
    Utc.timestamp_opt(timestamp, 0)
        .single()
        .ok_or_else(|| anyhow!("invalid unix timestamp: {timestamp}"))
}

pub(crate) fn datetime_to_unix_timestamp(timestamp: &DateTime<Utc>) -> i64 {
    timestamp.timestamp()
}
