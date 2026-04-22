use anyhow::{anyhow, Result};
use chrono::{DateTime, Utc};
use rusqlite::{params, Connection};
use serde::{Deserialize, Serialize};
use std::fs;
use std::path::{Path, PathBuf};

use crate::dialect::canonical_emotion_code;
use crate::diary::memory_gate::{config_gate, MemoryGateAction};
use crate::storage::memory::{ensure_memory_schema, DEFAULT_HEAT_I32};
use crate::vector_storage::{MemoryInsert, VectorStorage};

pub mod memory_gate;

const DEFAULT_WING: &str = "self";
const DEFAULT_ROOM: &str = "journal";
const DIARY_META_PREFIX: &str = "DIARY_META:";

#[derive(Serialize, Deserialize, Debug, Clone, PartialEq)]
pub struct DiaryEntry {
    pub id: i64,
    pub agent: String,
    pub content: String,
    pub timestamp: String,
    pub tags: Vec<String>,
    pub emotion: Option<String>,
    pub emotion_code: Option<String>,
    pub wing: String,
    pub room: String,
    pub heat_i32: i32,
}

#[derive(Serialize, Deserialize, Debug, Clone, PartialEq)]
pub struct DiaryWriteRequest {
    pub agent: String,
    pub content: String,
    #[serde(default)]
    pub tags: Vec<String>,
    pub emotion: Option<String>,
    pub timestamp: Option<String>,
    pub wing: Option<String>,
    pub room: Option<String>,
}

#[derive(Serialize, Deserialize, Debug, Clone, PartialEq)]
struct DiaryMeta {
    agent: String,
    tags: Vec<String>,
    emotion: Option<String>,
    emotion_code: Option<String>,
    timestamp: String,
    wing: String,
    room: String,
}

impl DiaryWriteRequest {
    fn into_normalized(self) -> Result<NormalizedDiaryWrite> {
        let agent = self.agent.trim().to_string();
        if agent.is_empty() {
            return Err(anyhow!("Diary agent cannot be empty"));
        }

        let content = self.content.trim().to_string();
        if content.is_empty() {
            return Err(anyhow!("Diary content cannot be empty"));
        }

        let timestamp = match self.timestamp {
            Some(timestamp) => DateTime::parse_from_rfc3339(timestamp.trim())
                .map_err(|_| anyhow!("Invalid RFC3339 timestamp: {}", timestamp))?
                .with_timezone(&Utc),
            None => Utc::now(),
        };

        let tags = self
            .tags
            .into_iter()
            .map(|tag| tag.trim().to_string())
            .filter(|tag| !tag.is_empty())
            .fold(Vec::<String>::new(), |mut acc, tag| {
                if !acc.contains(&tag) {
                    acc.push(tag);
                }
                acc
            });

        let emotion = self
            .emotion
            .as_ref()
            .map(|emotion| emotion.trim().to_lowercase())
            .filter(|emotion| !emotion.is_empty());
        let emotion_code = emotion
            .as_ref()
            .map(|emotion| {
                canonical_emotion_code(emotion)
                    .map(str::to_string)
                    .ok_or_else(|| anyhow!("Unknown emotion: {}", emotion))
            })
            .transpose()?;

        let wing = self
            .wing
            .map(|value| value.trim().to_string())
            .filter(|value| !value.is_empty())
            .unwrap_or_else(|| DEFAULT_WING.to_string());
        let room = self
            .room
            .map(|value| value.trim().to_string())
            .filter(|value| !value.is_empty())
            .unwrap_or_else(|| DEFAULT_ROOM.to_string());

        Ok(NormalizedDiaryWrite {
            agent,
            content,
            tags,
            emotion,
            emotion_code,
            timestamp,
            wing,
            room,
        })
    }
}

#[derive(Debug, Clone, PartialEq)]
struct NormalizedDiaryWrite {
    agent: String,
    content: String,
    tags: Vec<String>,
    emotion: Option<String>,
    emotion_code: Option<String>,
    timestamp: DateTime<Utc>,
    wing: String,
    room: String,
}

pub struct Diary {
    config_dir: PathBuf,
}

impl Diary {
    pub fn new<P: AsRef<Path>>(db_path: P) -> Result<Self> {
        let db_path = db_path.as_ref();
        let config_dir = db_path
            .parent()
            .map(Path::to_path_buf)
            .unwrap_or_else(|| PathBuf::from("."));
        fs::create_dir_all(&config_dir)?;
        Ok(Self { config_dir })
    }

    pub fn new_in_memory() -> Result<Self> {
        let temp_dir = tempfile::tempdir()?;
        let config_dir = temp_dir.keep();
        fs::create_dir_all(&config_dir)?;
        Ok(Self { config_dir })
    }

    pub fn write(&self, request: DiaryWriteRequest) -> Result<i64> {
        let normalized = request.into_normalized()?;
        let text = render_memory_text(&normalized)?;
        let source = source_for_agent(&normalized.agent);
        let insert = MemoryInsert {
            text_content: &text,
            wing: &normalized.wing,
            room: &normalized.room,
            source_file: Some(&source),
            source_mtime: None,
            valid_from: normalized.timestamp.timestamp(),
            heat_i32: DEFAULT_HEAT_I32,
            emotion_valence: 0,
            emotion_arousal: 0,
            is_archive_candidate: false,
            reason: None,
            discard_candidate: false,
            merged_into_id: None,
        };

        match self.open_vector_storage() {
            Ok(mut storage) => self.write_with_memory_gate(&mut storage, insert),
            Err(_) => self.insert_memory_without_index(insert),
        }
    }

    pub fn write_entry(&self, agent: &str, content: &str) -> Result<i64> {
        self.write(DiaryWriteRequest {
            agent: agent.to_string(),
            content: content.to_string(),
            tags: vec![],
            emotion: None,
            timestamp: None,
            wing: None,
            room: None,
        })
    }

    pub fn read_entries(&self, agent: &str, last_n: usize) -> Result<Vec<DiaryEntry>> {
        let conn = self.open_memory_db()?;
        let mut stmt = conn.prepare(
            "SELECT id, text_content, valid_from, wing, room, heat_i32
             FROM memories
             WHERE source_file = ?1
             ORDER BY valid_from DESC
             LIMIT ?2",
        )?;

        let rows = stmt.query_map(params![source_for_agent(agent), last_n as i64], |row| {
            let text_content: String = row.get(1)?;
            let wing: String = row.get(3)?;
            let room: String = row.get(4)?;
            let heat_i32: i32 = row.get(5)?;
            let (meta, content) = parse_memory_text(&text_content).map_err(|err| {
                rusqlite::Error::FromSqlConversionFailure(
                    1,
                    rusqlite::types::Type::Text,
                    Box::new(std::io::Error::new(
                        std::io::ErrorKind::InvalidData,
                        err.to_string(),
                    )),
                )
            })?;

            Ok(DiaryEntry {
                id: row.get(0)?,
                agent: meta.agent,
                content,
                timestamp: meta.timestamp,
                tags: meta.tags,
                emotion: meta.emotion,
                emotion_code: meta.emotion_code,
                wing,
                room,
                heat_i32,
            })
        })?;

        let mut entries = Vec::new();
        for row in rows {
            entries.push(row?);
        }

        entries.reverse();
        Ok(entries)
    }

    pub fn read_all_entries(&self, agent: &str) -> Result<Vec<DiaryEntry>> {
        self.read_entries(agent, usize::MAX)
    }

    pub fn delete_entry(&self, id: i64) -> Result<()> {
        let storage = self.open_vector_storage()?;
        storage.delete_memory(id)?;
        storage.save_index(self.index_path())?;
        Ok(())
    }

    pub fn get_stats(&self) -> Result<(i64, i64)> {
        let conn = self.open_memory_db()?;
        let total: i64 = conn.query_row(
            "SELECT COUNT(*) FROM memories WHERE source_file LIKE 'diary://%'",
            [],
            |row| row.get(0),
        )?;
        let agents: i64 = conn.query_row(
            "SELECT COUNT(DISTINCT source_file) FROM memories WHERE source_file LIKE 'diary://%'",
            [],
            |row| row.get(0),
        )?;
        Ok((total, agents))
    }

    pub fn memory_db_path(&self) -> PathBuf {
        self.config_dir.join("vectors.db")
    }

    pub fn index_path(&self) -> PathBuf {
        self.config_dir.join("vectors.usearch")
    }

    fn open_memory_db(&self) -> Result<Connection> {
        let conn = Connection::open(self.memory_db_path())?;
        ensure_memory_schema(&conn)?;
        Ok(conn)
    }

    fn open_vector_storage(&self) -> Result<VectorStorage> {
        VectorStorage::new(self.memory_db_path(), self.index_path())
    }

    fn write_with_memory_gate(
        &self,
        storage: &mut VectorStorage,
        insert: MemoryInsert<'_>,
    ) -> Result<i64> {
        let gate = config_gate(&crate::config::MempalaceConfig::new(Some(
            self.config_dir.clone(),
        )));
        let decision = gate.judge(
            storage,
            insert.text_content,
            Some(insert.wing),
            Some(insert.room),
        )?;

        match decision.action {
            MemoryGateAction::Store => {
                let stored = MemoryInsert {
                    reason: Some(decision.reason.as_str()),
                    discard_candidate: false,
                    merged_into_id: None,
                    ..insert
                };
                match storage.add_memory_record(stored) {
                    Ok(id) => {
                        storage.save_index(self.index_path())?;
                        Ok(id)
                    }
                    Err(_) => self.insert_memory_without_index(stored),
                }
            }
            MemoryGateAction::Discard => {
                let discarded = MemoryInsert {
                    reason: Some(decision.reason.as_str()),
                    discard_candidate: true,
                    merged_into_id: None,
                    ..insert
                };
                let id = self.insert_memory_without_index(discarded)?;
                Ok(id)
            }
            MemoryGateAction::Merge { target_id, .. } => {
                let target = storage.get_memory_by_id(target_id)?;
                gate.merge_into_existing(storage, &target, insert.text_content, &decision.reason)?;
                Ok(target_id)
            }
        }
    }

    fn insert_memory_without_index(&self, insert: MemoryInsert<'_>) -> Result<i64> {
        let conn = self.open_memory_db()?;
        conn.execute(
            "INSERT INTO memories (
                text_content,
                wing,
                room,
                source_file,
                source_mtime,
                valid_from,
                last_accessed,
                access_count,
                importance_score,
                heat_i32,
                emotion_valence,
                emotion_arousal,
                is_archive_candidate,
                reason,
                discard_candidate,
                merged_into_id
             )
             VALUES (?1, ?2, ?3, ?4, ?5, ?6, ?6, 0, 5.0, ?7, ?8, ?9, ?10, ?11, ?12, ?13)",
            params![
                insert.text_content,
                insert.wing,
                insert.room,
                insert.source_file,
                insert.source_mtime,
                insert.valid_from,
                insert.heat_i32,
                insert.emotion_valence,
                insert.emotion_arousal,
                insert.is_archive_candidate as i64,
                insert.reason,
                insert.discard_candidate as i64,
                insert.merged_into_id,
            ],
        )?;
        Ok(conn.last_insert_rowid())
    }
}

fn render_memory_text(write: &NormalizedDiaryWrite) -> Result<String> {
    let meta = DiaryMeta {
        agent: write.agent.clone(),
        tags: write.tags.clone(),
        emotion: write.emotion.clone(),
        emotion_code: write.emotion_code.clone(),
        timestamp: write.timestamp.to_rfc3339(),
        wing: write.wing.clone(),
        room: write.room.clone(),
    };
    Ok(format!(
        "{}{}\n{}",
        DIARY_META_PREFIX,
        serde_json::to_string(&meta)?,
        write.content
    ))
}

fn parse_memory_text(text: &str) -> Result<(DiaryMeta, String)> {
    let (meta_line, content) = text
        .split_once('\n')
        .ok_or_else(|| anyhow!("Diary memory payload missing metadata header"))?;
    let payload = meta_line
        .strip_prefix(DIARY_META_PREFIX)
        .ok_or_else(|| anyhow!("Diary memory payload missing DIARY_META prefix"))?;
    let meta: DiaryMeta = serde_json::from_str(payload)?;
    Ok((meta, content.to_string()))
}

fn source_for_agent(agent: &str) -> String {
    format!("diary://{}", agent.trim())
}

pub fn get_diary_path() -> String {
    let home = std::env::var("HOME").unwrap_or_else(|_| ".".into());
    let path = PathBuf::from(&home).join(".mempalace").join("diary.db");

    if let Some(parent) = path.parent() {
        let _ = std::fs::create_dir_all(parent);
    }

    path.to_string_lossy().to_string()
}

pub fn write_diary_at<P: AsRef<Path>>(db_path: P, agent: &str, content: &str) -> Result<()> {
    let diary = Diary::new(db_path)?;
    let id = diary.write_entry(agent, content)?;
    println!("Diary entry {} written for agent {}", id, agent);
    Ok(())
}

pub fn read_diary_at<P: AsRef<Path>>(
    db_path: P,
    agent: &str,
    last_n: usize,
) -> Result<Vec<DiaryEntry>> {
    let diary = Diary::new(db_path)?;
    diary.read_entries(agent, last_n)
}

pub fn write_diary(agent: &str, content: &str) -> Result<()> {
    let path = get_diary_path();
    let diary = Diary::new(&path)?;
    let id = diary.write_entry(agent, content)?;
    println!("✓ Diary entry {} written for agent {}", id, agent);
    Ok(())
}

pub fn read_diary(agent: &str, last_n: usize) -> Result<Vec<DiaryEntry>> {
    let path = get_diary_path();
    let diary = Diary::new(&path)?;
    diary.read_entries(agent, last_n)
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::config::MempalaceConfig;
    use crate::storage::Layer1;
    use serial_test::serial;
    use tempfile::tempdir;

    #[test]
    fn test_diary_request_defaults_and_emotion_mapping() {
        let normalized = DiaryWriteRequest {
            agent: "test-agent".to_string(),
            content: "First entry".to_string(),
            tags: vec!["focus".to_string(), "focus".to_string(), "  ".to_string()],
            emotion: Some("joy".to_string()),
            timestamp: Some("2026-04-14T10:00:00Z".to_string()),
            wing: None,
            room: None,
        }
        .into_normalized()
        .unwrap();

        assert_eq!(normalized.tags, vec!["focus".to_string()]);
        assert_eq!(normalized.emotion_code.as_deref(), Some("joy"));
        assert_eq!(normalized.wing, DEFAULT_WING);
        assert_eq!(normalized.room, DEFAULT_ROOM);
    }

    #[test]
    fn test_diary_request_rejects_unknown_emotion() {
        let result = DiaryWriteRequest {
            agent: "test-agent".to_string(),
            content: "First entry".to_string(),
            tags: vec![],
            emotion: Some("mystery".to_string()),
            timestamp: None,
            wing: None,
            room: None,
        }
        .into_normalized();

        assert!(result.is_err());
        assert!(result.unwrap_err().to_string().contains("Unknown emotion"));
    }

    #[test]
    fn test_diary_request_rejects_blank_content() {
        let result = DiaryWriteRequest {
            agent: "test-agent".to_string(),
            content: "   ".to_string(),
            tags: vec![],
            emotion: None,
            timestamp: None,
            wing: None,
            room: None,
        }
        .into_normalized();

        assert!(result.is_err());
        assert!(result.unwrap_err().to_string().contains("cannot be empty"));
    }

    #[test]
    fn test_render_and_parse_diary_memory_text_roundtrip() {
        let normalized = DiaryWriteRequest {
            agent: "test-agent".to_string(),
            content: "Today I shipped the feature".to_string(),
            tags: vec!["work".to_string()],
            emotion: Some("joy".to_string()),
            timestamp: Some("2026-04-14T10:00:00Z".to_string()),
            wing: Some("self".to_string()),
            room: Some("journal".to_string()),
        }
        .into_normalized()
        .unwrap();

        let rendered = render_memory_text(&normalized).unwrap();
        let (meta, content) = parse_memory_text(&rendered).unwrap();
        assert_eq!(meta.agent, "test-agent");
        assert_eq!(meta.tags, vec!["work".to_string()]);
        assert_eq!(meta.emotion_code.as_deref(), Some("joy"));
        assert_eq!(content, "Today I shipped the feature");
    }

    #[test]
    #[serial]
    fn test_diary_write_persists_memory_record_with_defaults() {
        let dir = tempdir().unwrap();
        let diary_db = dir.path().join("diary.db");
        let diary = Diary::new(&diary_db).unwrap();

        let memory_id = diary
            .write(DiaryWriteRequest {
                agent: "writer".to_string(),
                content: "Captured in memory storage".to_string(),
                tags: vec!["journal".to_string(), "mvp".to_string()],
                emotion: Some("trust".to_string()),
                timestamp: Some("2026-04-14T11:00:00Z".to_string()),
                wing: None,
                room: None,
            })
            .unwrap();

        assert!(memory_id > 0);

        let conn = Connection::open(dir.path().join("vectors.db")).unwrap();
        let row = conn
            .query_row(
                "SELECT text_content, wing, room, source_file, heat_i32 FROM memories WHERE id = ?1",
                params![memory_id],
                |row| {
                    Ok((
                        row.get::<_, String>(0)?,
                        row.get::<_, String>(1)?,
                        row.get::<_, String>(2)?,
                        row.get::<_, Option<String>>(3)?,
                        row.get::<_, i32>(4)?,
                    ))
                },
            )
            .unwrap();

        let (meta, content) = parse_memory_text(&row.0).unwrap();
        assert_eq!(content, "Captured in memory storage");
        assert_eq!(meta.tags, vec!["journal".to_string(), "mvp".to_string()]);
        assert_eq!(meta.emotion_code.as_deref(), Some("trust"));
        assert_eq!(row.1, DEFAULT_WING);
        assert_eq!(row.2, DEFAULT_ROOM);
        assert_eq!(row.3.as_deref(), Some("diary://writer"));
        assert_eq!(row.4, DEFAULT_HEAT_I32);
    }

    #[test]
    #[serial]
    fn test_diary_read_entries_returns_recent_entries_in_chronological_order() {
        let dir = tempdir().unwrap();
        let diary_db = dir.path().join("diary.db");
        let diary = Diary::new(&diary_db).unwrap();

        for i in 0..5 {
            diary
                .write(DiaryWriteRequest {
                    agent: "reader".to_string(),
                    content: format!("entry {}", i),
                    tags: vec!["journal".to_string()],
                    emotion: None,
                    timestamp: Some(format!("2026-04-14T10:0{}:00Z", i)),
                    wing: None,
                    room: None,
                })
                .unwrap();
        }

        let entries = diary.read_entries("reader", 3).unwrap();
        assert_eq!(entries.len(), 3);
        assert_eq!(entries[0].content, "entry 2");
        assert_eq!(entries[1].content, "entry 3");
        assert_eq!(entries[2].content, "entry 4");
    }

    #[test]
    #[serial]
    fn test_diary_low_value_entry_is_marked_discard_candidate() {
        let dir = tempdir().unwrap();
        let diary_db = dir.path().join("diary.db");
        let diary = Diary::new(&diary_db).unwrap();

        let memory_id = diary
            .write(DiaryWriteRequest {
                agent: "discarder".to_string(),
                content: "ok".to_string(),
                tags: vec![],
                emotion: None,
                timestamp: Some("2026-04-14T10:30:00Z".to_string()),
                wing: None,
                room: None,
            })
            .unwrap();

        let conn = Connection::open(dir.path().join("vectors.db")).unwrap();
        let row = conn
            .query_row(
                "SELECT discard_candidate, reason FROM memories WHERE id = ?1",
                params![memory_id],
                |row| Ok((row.get::<_, i64>(0)?, row.get::<_, Option<String>>(1)?)),
            )
            .unwrap();

        assert_eq!(row.0, 1);
        assert!(row
            .1
            .as_deref()
            .unwrap_or_default()
            .contains("discard_candidate"));
    }

    #[tokio::test]
    #[serial]
    async fn test_diary_write_is_visible_to_layer1() {
        let dir = tempdir().unwrap();
        let config = MempalaceConfig::new(Some(dir.path().to_path_buf()));
        let diary_db = dir.path().join("diary.db");
        let diary = Diary::new(&diary_db).unwrap();

        diary
            .write(DiaryWriteRequest {
                agent: "layer1".to_string(),
                content: "I wrote a journal memory".to_string(),
                tags: vec!["personal".to_string()],
                emotion: Some("joy".to_string()),
                timestamp: Some("2026-04-14T12:00:00Z".to_string()),
                wing: None,
                room: None,
            })
            .unwrap();

        let layer1 = Layer1::new(config, Some(DEFAULT_WING.to_string()));
        let rendered = layer1.generate().await;
        assert!(rendered.contains("JOURNAL"));
        assert!(rendered.contains("I wrote a journal memory"));
    }
}
