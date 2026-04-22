// vector_storage.rs — MemPalace Pure-Rust Storage Engine (replaces ChromaDB)
//
// Zero-network, embedded storage combining:
//   • fastembed-rs  → CPU/ONNX text embeddings (AllMiniLML6V2, 384-dim)
//   • rusqlite      → relational source of truth
//   • usearch       → SIMD-accelerated HNSW ANN index

use std::path::Path;
use std::sync::Arc;
use std::time::{SystemTime, UNIX_EPOCH};

use crate::api::LaputaError;
use crate::heat::HeatService;
use crate::searcher::RecallQuery;
use crate::storage::memory::{
    datetime_to_unix_timestamp, ensure_memory_schema, heat_from_i32, unix_timestamp_to_datetime,
    DEFAULT_HEAT_I32, MAX_HEAT_I32, MIN_HEAT_I32,
};
use anyhow::{anyhow, Context, Result};
use chrono::{DateTime, Duration, Utc};
use fastembed::{EmbeddingModel, InitOptions, TextEmbedding};
use rusqlite::{params, Connection, OptionalExtension};
use std::path::PathBuf;
use usearch::{Index, IndexOptions, MetricKind, ScalarKind};

const VECTOR_DIMS: usize = 384;
const HNSW_M: usize = 16;
const HNSW_EF_CONSTRUCTION: usize = 128;

pub use crate::storage::memory::LaputaMemoryRecord as MemoryRecord;

#[derive(Debug, Clone, PartialEq, Eq)]
pub enum UserIntervention {
    Important {
        reason: String,
    },
    Forget {
        reason: String,
    },
    EmotionAnchor {
        valence: i32,
        arousal: u32,
        reason: String,
    },
}

/// Represents a chronological validity window for a memory.
#[derive(Debug, Clone, Default)]
pub struct TemporalRange {
    pub valid_from: Option<i64>,
    pub valid_to: Option<i64>,
}

#[derive(Debug, Clone, Default)]
pub struct SemanticSearchFilter {
    pub wing: Option<String>,
    pub room: Option<String>,
    pub include_discarded: bool,
    pub sort_by_heat: bool,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Default)]
pub enum EmotionSort {
    #[default]
    Recent,
    ValenceDesc,
    ArousalDesc,
    AbsoluteValenceDesc,
}

#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct EmotionQuery {
    pub wing: Option<String>,
    pub room: Option<String>,
    pub min_valence: Option<i32>,
    pub max_valence: Option<i32>,
    pub min_arousal: Option<u32>,
    pub max_arousal: Option<u32>,
    pub include_discarded: bool,
    pub limit: usize,
    pub sort: EmotionSort,
}

#[derive(Clone, Copy)]
pub struct MemoryInsert<'a> {
    pub text_content: &'a str,
    pub wing: &'a str,
    pub room: &'a str,
    pub source_file: Option<&'a str>,
    pub source_mtime: Option<f64>,
    pub valid_from: i64,
    pub heat_i32: i32,
    pub emotion_valence: i32,
    pub emotion_arousal: u32,
    pub is_archive_candidate: bool,
    pub reason: Option<&'a str>,
    pub discard_candidate: bool,
    pub merged_into_id: Option<i64>,
}

fn now_unix() -> i64 {
    SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .expect("system clock before Unix epoch")
        .as_secs() as i64
}

fn compute_decayed_importance(base_score: f32, last_accessed: i64, access_count: i64) -> f32 {
    let days_since = ((now_unix() - last_accessed) as f32 / 86400.0).max(0.0);
    let freq_boost = (1.0 + access_count as f32).ln().max(1.0);
    base_score * 0.9f32.powf(days_since) * freq_boost
}

fn build_index() -> Result<Index> {
    let opts = IndexOptions {
        dimensions: VECTOR_DIMS,
        metric: MetricKind::Cos,
        quantization: ScalarKind::F32,
        connectivity: HNSW_M,
        expansion_add: HNSW_EF_CONSTRUCTION,
        expansion_search: 64,
        ..Default::default()
    };
    Index::new(&opts).map_err(|e| anyhow!("usearch index creation failed: {e}"))
}

/// The pure-Rust vector storage engine combining SQLite metadata and usearch HNSW index.
pub struct VectorStorage {
    pub embedder: Option<Arc<TextEmbedding>>,
    pub db: Connection,
    pub index: Index,
}

impl VectorStorage {
    pub fn new(db_path: impl AsRef<Path>, index_path: impl AsRef<Path>) -> Result<Self> {
        let cache_dir = std::env::var("MEMPALACE_MODELS_DIR")
            .ok()
            .map(PathBuf::from)
            .filter(|p| p.exists())
            .or_else(|| {
                std::env::current_exe()
                    .ok()
                    .and_then(|exe| exe.parent().map(|p| p.join("models")))
                    .filter(|p| p.exists())
            });

        let mut init_opts =
            InitOptions::new(EmbeddingModel::AllMiniLML6V2).with_show_download_progress(false);

        if let Some(cache) = cache_dir {
            init_opts = init_opts.with_cache_dir(cache);
        }

        let embedder = TextEmbedding::try_new(init_opts).ok().map(Arc::new);
        Self::new_with_optional_embedder(db_path, index_path, embedder)
    }

    pub fn new_with_embedder(
        db_path: impl AsRef<Path>,
        index_path: impl AsRef<Path>,
        embedder: Arc<TextEmbedding>,
    ) -> Result<Self> {
        Self::new_with_optional_embedder(db_path, index_path, Some(embedder))
    }

    fn new_with_optional_embedder(
        db_path: impl AsRef<Path>,
        index_path: impl AsRef<Path>,
        embedder: Option<Arc<TextEmbedding>>,
    ) -> Result<Self> {
        let db = Connection::open(db_path.as_ref())
            .with_context(|| format!("Cannot open SQLite at {:?}", db_path.as_ref()))?;
        ensure_memory_schema(&db)?;

        let index_path = index_path.as_ref();
        let index = if index_path.exists() {
            let idx = build_index()?;
            idx.load(
                index_path
                    .to_str()
                    .ok_or_else(|| anyhow!("Non-UTF8 index path"))?,
            )
            .map_err(|e| anyhow!("Failed to load usearch index: {e}"))?;
            idx
        } else {
            build_index()?
        };

        Ok(Self {
            embedder,
            db,
            index,
        })
    }

    pub fn add_memory(
        &mut self,
        text: &str,
        wing: &str,
        room: &str,
        source_file: Option<&str>,
        source_mtime: Option<f64>,
    ) -> Result<i64> {
        self.add_memory_record(MemoryInsert {
            text_content: text,
            wing,
            room,
            source_file,
            source_mtime,
            valid_from: now_unix(),
            heat_i32: DEFAULT_HEAT_I32,
            emotion_valence: 0,
            emotion_arousal: 0,
            is_archive_candidate: false,
            reason: None,
            discard_candidate: false,
            merged_into_id: None,
        })
    }

    pub fn add_memory_record(&mut self, insert: MemoryInsert<'_>) -> Result<i64> {
        validate_heat_i32(insert.heat_i32)?;
        let vector = self.embed_single(insert.text_content)?;

        self.db.execute(
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
             VALUES (?1, ?2, ?3, ?4, ?5, ?6, ?7, 0, 5.0, ?8, ?9, ?10, ?11, ?12, ?13, ?14)",
            params![
                insert.text_content,
                insert.wing,
                insert.room,
                insert.source_file,
                insert.source_mtime,
                insert.valid_from,
                insert.valid_from,
                insert.heat_i32,
                insert.emotion_valence,
                insert.emotion_arousal,
                insert.is_archive_candidate as i64,
                insert.reason,
                insert.discard_candidate as i64,
                insert.merged_into_id
            ],
        )?;

        let row_id = self.db.last_insert_rowid();

        let needed = self.index.size() + 1;
        if needed > self.index.capacity() {
            let new_cap = (needed * 2).max(64);
            self.index
                .reserve(new_cap)
                .map_err(|e| anyhow!("usearch reserve failed: {e}"))?;
        }

        self.index
            .add(row_id as u64, &vector)
            .map_err(|e| anyhow!("usearch add failed: {e}"))?;

        Ok(row_id)
    }

    pub fn get_source_mtime(&self, source_file: &str) -> Result<Option<f64>> {
        let mut stmt = self.db.prepare(
            "SELECT source_mtime FROM memories WHERE source_file = ?1 ORDER BY id DESC LIMIT 1",
        )?;
        let mtime = stmt
            .query_row(params![source_file], |row| row.get::<_, Option<f64>>(0))
            .optional()?;
        Ok(mtime.flatten())
    }

    pub fn search_room(
        &self,
        query: &str,
        wing: &str,
        room: &str,
        limit: usize,
        at_time: Option<i64>,
    ) -> Result<Vec<MemoryRecord>> {
        if limit == 0 {
            return Ok(vec![]);
        }
        let at_time = at_time.unwrap_or_else(now_unix);
        let query_vector = self.embed_single(query)?;

        let mut stmt = self.db.prepare_cached(
            "SELECT id FROM memories
             WHERE wing = ?1 AND room = ?2
               AND discard_candidate = 0
               AND valid_from <= ?3
               AND (valid_to IS NULL OR valid_to >= ?3)",
        )?;

        let candidate_ids: Vec<u64> = stmt
            .query_map(params![wing, room, at_time], |row| row.get::<_, i64>(0))?
            .collect::<rusqlite::Result<Vec<_>>>()?
            .into_iter()
            .map(|id| id as u64)
            .collect();

        if candidate_ids.is_empty() {
            return Ok(vec![]);
        }

        let candidate_set: std::collections::HashSet<u64> = candidate_ids.iter().cloned().collect();
        let results = self
            .index
            .filtered_search(&query_vector, limit, |key: u64| {
                candidate_set.contains(&key)
            })
            .map_err(|e| anyhow!("usearch filtered_search failed: {e}"))?;

        if results.keys.is_empty() {
            return Ok(vec![]);
        }

        let id_placeholders: String = results
            .keys
            .iter()
            .enumerate()
            .map(|(i, _)| format!("?{}", i + 1))
            .collect::<Vec<_>>()
            .join(", ");

        let sql = format!(
            "SELECT id, text_content, wing, room, source_file, valid_from, valid_to, last_accessed, access_count, importance_score, heat_i32, emotion_valence, emotion_arousal, is_archive_candidate, reason, discard_candidate, merged_into_id
             FROM memories WHERE id IN ({id_placeholders})"
        );

        let mut stmt = self.db.prepare(&sql)?;
        let params_vec: Vec<&dyn rusqlite::ToSql> = results
            .keys
            .iter()
            .map(|k| k as &dyn rusqlite::ToSql)
            .collect();

        let mut record_map: std::collections::HashMap<i64, MemoryRecord> = stmt
            .query_map(params_vec.as_slice(), row_to_memory_record)?
            .collect::<rusqlite::Result<Vec<_>>>()?
            .into_iter()
            .map(|r| (r.id, r))
            .collect();

        let mut ordered: Vec<MemoryRecord> = results
            .keys
            .iter()
            .zip(results.distances.iter())
            .filter_map(|(&key, &dist)| {
                let id = key as i64;
                record_map.remove(&id).map(|mut rec| {
                    rec.score = 1.0 - dist;
                    rec
                })
            })
            .collect();

        ordered.sort_by(|a, b| {
            b.score
                .partial_cmp(&a.score)
                .unwrap_or(std::cmp::Ordering::Equal)
        });
        Ok(ordered)
    }

    pub fn semantic_search(
        &self,
        query_vector: &[f32],
        top_k: usize,
        wing: Option<&str>,
        room: Option<&str>,
        include_discarded: bool,
        sort_by_heat: bool,
    ) -> Result<Vec<(MemoryRecord, f32)>> {
        if top_k == 0 || self.index.size() == 0 {
            return Ok(vec![]);
        }

        let filter = SemanticSearchFilter {
            wing: wing.map(str::to_string),
            room: room.map(str::to_string),
            include_discarded,
            sort_by_heat,
        };
        let candidate_ids = self.semantic_candidate_ids(&filter)?;
        if candidate_ids.is_empty() {
            return Ok(vec![]);
        }

        let candidate_set: std::collections::HashSet<u64> = candidate_ids.iter().cloned().collect();
        let results = self
            .index
            .filtered_search(query_vector, top_k, |key: u64| candidate_set.contains(&key))
            .map_err(|e| anyhow!("usearch filtered_search failed: {e}"))?;

        self.materialize_semantic_results(results.keys, results.distances, sort_by_heat)
    }

    pub fn search(&self, query: &str, limit: usize) -> Result<Vec<MemoryRecord>> {
        if limit == 0 {
            return Ok(vec![]);
        }
        let query_vector = self.embed_single(query)?;

        let results = self
            .index
            .search(&query_vector, limit)
            .map_err(|e| anyhow!("usearch search failed: {e}"))?;

        if results.keys.is_empty() {
            return Ok(vec![]);
        }

        let id_placeholders: String = results
            .keys
            .iter()
            .enumerate()
            .map(|(i, _)| format!("?{}", i + 1))
            .collect::<Vec<_>>()
            .join(", ");

        let sql = format!(
            "SELECT id, text_content, wing, room, source_file, valid_from, valid_to, last_accessed, access_count, importance_score, heat_i32, emotion_valence, emotion_arousal, is_archive_candidate, reason, discard_candidate, merged_into_id
             FROM memories WHERE id IN ({id_placeholders}) AND discard_candidate = 0"
        );

        let mut stmt = self.db.prepare(&sql)?;
        let params_vec: Vec<&dyn rusqlite::ToSql> = results
            .keys
            .iter()
            .map(|k| k as &dyn rusqlite::ToSql)
            .collect();

        let mut record_map: std::collections::HashMap<i64, MemoryRecord> = stmt
            .query_map(params_vec.as_slice(), row_to_memory_record)?
            .collect::<rusqlite::Result<Vec<_>>>()?
            .into_iter()
            .map(|r| (r.id, r))
            .collect();

        let mut ordered: Vec<MemoryRecord> = results
            .keys
            .iter()
            .zip(results.distances.iter())
            .filter_map(|(&key, &dist)| {
                let id = key as i64;
                record_map.remove(&id).map(|mut rec| {
                    rec.score = 1.0 - dist;
                    rec
                })
            })
            .collect();

        ordered.sort_by(|a, b| {
            b.score
                .partial_cmp(&a.score)
                .unwrap_or(std::cmp::Ordering::Equal)
        });
        Ok(ordered)
    }

    pub fn get_memories(
        &self,
        wing: Option<&str>,
        room: Option<&str>,
        limit: usize,
    ) -> Result<Vec<MemoryRecord>> {
        let limit = i64::try_from(limit).unwrap_or(i64::MAX);
        let (sql, params_dyn): (String, Vec<Box<dyn rusqlite::ToSql>>) = match (wing, room) {
            (Some(w), Some(r)) => (
                format!("SELECT id, text_content, wing, room, source_file, valid_from, valid_to, last_accessed, access_count, importance_score, heat_i32, emotion_valence, emotion_arousal, is_archive_candidate, reason, discard_candidate, merged_into_id FROM memories WHERE wing = ?1 AND room = ?2 AND discard_candidate = 0 ORDER BY valid_from DESC LIMIT {limit}"),
                vec![Box::new(w.to_string()), Box::new(r.to_string())],
            ),
            (Some(w), None) => (
                format!("SELECT id, text_content, wing, room, source_file, valid_from, valid_to, last_accessed, access_count, importance_score, heat_i32, emotion_valence, emotion_arousal, is_archive_candidate, reason, discard_candidate, merged_into_id FROM memories WHERE wing = ?1 AND discard_candidate = 0 ORDER BY valid_from DESC LIMIT {limit}"),
                vec![Box::new(w.to_string())],
            ),
            (None, Some(r)) => (
                format!("SELECT id, text_content, wing, room, source_file, valid_from, valid_to, last_accessed, access_count, importance_score, heat_i32, emotion_valence, emotion_arousal, is_archive_candidate, reason, discard_candidate, merged_into_id FROM memories WHERE room = ?1 AND discard_candidate = 0 ORDER BY valid_from DESC LIMIT {limit}"),
                vec![Box::new(r.to_string())],
            ),
            (None, None) => (
                format!("SELECT id, text_content, wing, room, source_file, valid_from, valid_to, last_accessed, access_count, importance_score, heat_i32, emotion_valence, emotion_arousal, is_archive_candidate, reason, discard_candidate, merged_into_id FROM memories WHERE discard_candidate = 0 ORDER BY valid_from DESC LIMIT {limit}"),
                vec![],
            ),
        };
        let mut stmt = self.db.prepare(&sql)?;
        let params_ref: Vec<&dyn rusqlite::ToSql> = params_dyn.iter().map(|p| p.as_ref()).collect();
        let records = stmt
            .query_map(params_ref.as_slice(), row_to_memory_record)?
            .collect::<rusqlite::Result<Vec<_>>>()?;
        Ok(records)
    }

    pub fn recall_by_time_range(&self, query: &RecallQuery) -> Result<Vec<MemoryRecord>> {
        query.validate().map_err(anyhow::Error::new)?;

        let mut sql = String::from(
            "SELECT id, text_content, wing, room, source_file, valid_from, valid_to, \
             last_accessed, access_count, importance_score, heat_i32, emotion_valence, \
             emotion_arousal, is_archive_candidate, reason, discard_candidate, merged_into_id \
             FROM memories WHERE valid_from >= ?1 AND valid_from <= ?2",
        );
        let mut params_dyn: Vec<Box<dyn rusqlite::ToSql>> =
            vec![Box::new(query.start), Box::new(query.end)];
        let mut next_idx = 3;

        if let Some(wing) = &query.wing {
            sql.push_str(&format!(" AND wing = ?{next_idx}"));
            params_dyn.push(Box::new(wing.clone()));
            next_idx += 1;
        }

        if let Some(room) = &query.room {
            sql.push_str(&format!(" AND room = ?{next_idx}"));
            params_dyn.push(Box::new(room.clone()));
        }

        if !query.include_discarded {
            sql.push_str(" AND discard_candidate = 0");
        }

        sql.push_str(&format!(
            " ORDER BY heat_i32 DESC, valid_from DESC LIMIT {}",
            query.validated_limit()
        ));

        let mut stmt = self.db.prepare(&sql)?;
        let params_ref: Vec<&dyn rusqlite::ToSql> = params_dyn.iter().map(|p| p.as_ref()).collect();
        let records = stmt
            .query_map(params_ref.as_slice(), row_to_memory_record)?
            .collect::<rusqlite::Result<Vec<_>>>()?;
        Ok(records)
    }

    pub fn get_all_ids(&self, wing: Option<&str>) -> Result<Vec<i64>> {
        if let Some(w) = wing {
            let mut stmt = self
                .db
                .prepare("SELECT id FROM memories WHERE wing = ?1 AND discard_candidate = 0")?;
            let ids = stmt
                .query_map(params![w], |row| row.get(0))?
                .collect::<rusqlite::Result<Vec<i64>>>()?;
            Ok(ids)
        } else {
            let mut stmt = self
                .db
                .prepare("SELECT id FROM memories WHERE discard_candidate = 0")?;
            let ids = stmt
                .query_map([], |row| row.get(0))?
                .collect::<rusqlite::Result<Vec<i64>>>()?;
            Ok(ids)
        }
    }

    pub fn get_memory_by_id(&self, id: i64) -> Result<MemoryRecord> {
        self.db.query_row(
            "SELECT id, text_content, wing, room, source_file, valid_from, valid_to, last_accessed, access_count, importance_score, heat_i32, emotion_valence, emotion_arousal, is_archive_candidate, reason, discard_candidate, merged_into_id FROM memories WHERE id = ?1",
            params![id],
            row_to_memory_record,
        )
        .context("Memory not found")
    }

    pub fn source_db_path(&self) -> Result<PathBuf> {
        let mut stmt = self.db.prepare("PRAGMA database_list")?;
        let rows = stmt.query_map([], |row| {
            Ok((row.get::<_, String>(1)?, row.get::<_, String>(2)?))
        })?;

        for row in rows {
            let (name, path) = row?;
            if name == "main" {
                if path.is_empty() {
                    return Err(anyhow!("main SQLite database path is empty"));
                }
                return Ok(PathBuf::from(path));
            }
        }

        Err(anyhow!("main SQLite database path not found"))
    }

    pub fn update_memory_summary(&self, id: i64, new_summary: &str) -> Result<()> {
        self.db.execute(
            "UPDATE memories SET text_content = ?1 WHERE id = ?2",
            params![new_summary, id],
        )?;
        Ok(())
    }

    pub fn update_memory_after_merge(
        &self,
        id: i64,
        new_summary: &str,
        new_heat_i32: i32,
        reason: &str,
    ) -> Result<()> {
        validate_heat_i32(new_heat_i32)?;
        self.db.execute(
            "UPDATE memories
             SET text_content = ?1,
                 heat_i32 = ?2,
                 reason = ?3
             WHERE id = ?4",
            params![new_summary, new_heat_i32, reason, id],
        )?;
        Ok(())
    }

    pub fn apply_intervention(
        &self,
        memory_id: i64,
        intervention: UserIntervention,
    ) -> Result<MemoryRecord> {
        match intervention {
            UserIntervention::Important { reason } => self.mark_important(memory_id, &reason),
            UserIntervention::Forget { reason } => self.mark_forget(memory_id, &reason),
            UserIntervention::EmotionAnchor {
                valence,
                arousal,
                reason,
            } => self.mark_emotion_anchor_with_reason(memory_id, valence, arousal, Some(&reason)),
        }
    }

    pub fn mark_important(&self, memory_id: i64, reason: &str) -> Result<MemoryRecord> {
        let rows_affected = self.db.execute(
            "UPDATE memories
             SET heat_i32 = 9000,
                 is_archive_candidate = 0,
                 reason = ?1
             WHERE id = ?2",
            params![reason, memory_id],
        )?;

        if rows_affected == 0 {
            return Err(anyhow!("Memory not found: id={memory_id}"));
        }

        self.get_memory_by_id(memory_id)
    }

    pub fn mark_forget(&self, memory_id: i64, reason: &str) -> Result<MemoryRecord> {
        let rows_affected = self.db.execute(
            "UPDATE memories
             SET heat_i32 = 0,
                 is_archive_candidate = 1,
                 reason = ?1
             WHERE id = ?2",
            params![reason, memory_id],
        )?;

        if rows_affected == 0 {
            return Err(anyhow!("Memory not found: id={memory_id}"));
        }

        self.get_memory_by_id(memory_id)
    }

    pub fn mark_emotion_anchor(
        &self,
        memory_id: i64,
        valence: i32,
        arousal: u32,
    ) -> Result<MemoryRecord> {
        self.mark_emotion_anchor_with_reason(memory_id, valence, arousal, None)
    }

    pub fn update_memory_emotion(
        &self,
        memory_id: i64,
        valence: i32,
        arousal: u32,
    ) -> Result<MemoryRecord> {
        let mut updated = self
            .get_memory_by_id(memory_id)
            .with_context(|| format!("Memory not found: id={memory_id}"))?;
        updated.update_emotion(valence, arousal);

        let rows_affected = self.db.execute(
            "UPDATE memories
             SET emotion_valence = ?1,
                 emotion_arousal = ?2
             WHERE id = ?3",
            params![updated.emotion_valence, updated.emotion_arousal, memory_id],
        )?;

        if rows_affected == 0 {
            return Err(anyhow!("Memory not found: id={memory_id}"));
        }

        self.get_memory_by_id(memory_id)
    }

    pub fn list_memories_by_emotion(&self, query: &EmotionQuery) -> Result<Vec<MemoryRecord>> {
        if query.limit == 0 {
            return Ok(Vec::new());
        }

        let mut sql = String::from(
            "SELECT id, text_content, wing, room, source_file, valid_from, valid_to, \
             last_accessed, access_count, importance_score, heat_i32, emotion_valence, \
             emotion_arousal, is_archive_candidate, reason, discard_candidate, merged_into_id \
             FROM memories WHERE is_archive_candidate = 0",
        );
        let mut params_dyn: Vec<Box<dyn rusqlite::ToSql>> = Vec::new();
        let mut next_idx = 1;

        if let Some(wing) = &query.wing {
            sql.push_str(&format!(" AND wing = ?{next_idx}"));
            params_dyn.push(Box::new(wing.clone()));
            next_idx += 1;
        }

        if let Some(room) = &query.room {
            sql.push_str(&format!(" AND room = ?{next_idx}"));
            params_dyn.push(Box::new(room.clone()));
            next_idx += 1;
        }

        if let Some(min_valence) = query.min_valence {
            sql.push_str(&format!(" AND emotion_valence >= ?{next_idx}"));
            params_dyn.push(Box::new(min_valence));
            next_idx += 1;
        }

        if let Some(max_valence) = query.max_valence {
            sql.push_str(&format!(" AND emotion_valence <= ?{next_idx}"));
            params_dyn.push(Box::new(max_valence));
            next_idx += 1;
        }

        if let Some(min_arousal) = query.min_arousal {
            sql.push_str(&format!(" AND emotion_arousal >= ?{next_idx}"));
            params_dyn.push(Box::new(i64::from(min_arousal)));
            next_idx += 1;
        }

        if let Some(max_arousal) = query.max_arousal {
            sql.push_str(&format!(" AND emotion_arousal <= ?{next_idx}"));
            params_dyn.push(Box::new(i64::from(max_arousal)));
            next_idx += 1;
        }

        if !query.include_discarded {
            sql.push_str(&format!(" AND discard_candidate = ?{next_idx}"));
            params_dyn.push(Box::new(0_i64));
        }

        let order_clause = match query.sort {
            EmotionSort::Recent => " ORDER BY valid_from DESC, id DESC",
            EmotionSort::ValenceDesc => " ORDER BY emotion_valence DESC, emotion_arousal DESC, valid_from DESC, id DESC",
            EmotionSort::ArousalDesc => " ORDER BY emotion_arousal DESC, ABS(emotion_valence) DESC, valid_from DESC, id DESC",
            EmotionSort::AbsoluteValenceDesc => {
                " ORDER BY ABS(emotion_valence) DESC, emotion_arousal DESC, valid_from DESC, id DESC"
            }
        };
        sql.push_str(order_clause);
        sql.push_str(&format!(" LIMIT {}", query.limit));

        let mut stmt = self.db.prepare(&sql)?;
        let params_ref: Vec<&dyn rusqlite::ToSql> = params_dyn.iter().map(|p| p.as_ref()).collect();
        let records = stmt
            .query_map(params_ref.as_slice(), row_to_memory_record)?
            .collect::<rusqlite::Result<Vec<_>>>()?;
        Ok(records)
    }

    fn mark_emotion_anchor_with_reason(
        &self,
        memory_id: i64,
        valence: i32,
        arousal: u32,
        reason: Option<&str>,
    ) -> Result<MemoryRecord> {
        let record = self
            .get_memory_by_id(memory_id)
            .with_context(|| format!("Memory not found: id={memory_id}"))?;
        let new_heat_i32 = (record.heat_i32 + 2_000).min(MAX_HEAT_I32);
        let mut updated = record.with_updated_heat(heat_from_i32(new_heat_i32));
        updated.update_emotion(valence, arousal);

        let rows_affected = self.db.execute(
            "UPDATE memories
             SET heat_i32 = ?1,
                 emotion_valence = ?2,
                 emotion_arousal = ?3,
                 reason = COALESCE(?4, reason)
             WHERE id = ?5",
            params![
                updated.heat_i32,
                updated.emotion_valence,
                updated.emotion_arousal,
                reason,
                memory_id
            ],
        )?;

        if rows_affected == 0 {
            return Err(anyhow!("Memory not found: id={memory_id}"));
        }

        self.get_memory_by_id(memory_id)
    }

    pub fn touch_memory(&self, id: i64) -> Result<()> {
        let now = datetime_to_unix_timestamp(&unix_timestamp_to_datetime(now_unix())?);
        self.db.execute(
            "UPDATE memories SET access_count = access_count + 1, last_accessed = ?1 WHERE id = ?2",
            params![now, id],
        )?;
        Ok(())
    }

    pub fn list_decay_candidates(
        &self,
        older_than_unix: i64,
        limit: usize,
    ) -> Result<Vec<MemoryRecord>> {
        let limit = i64::try_from(limit).unwrap_or(i64::MAX);
        let mut stmt = self.db.prepare(
            "SELECT id, text_content, wing, room, source_file, valid_from, valid_to, last_accessed, access_count, importance_score, heat_i32, emotion_valence, emotion_arousal, is_archive_candidate, reason, discard_candidate, merged_into_id
             FROM memories
             WHERE discard_candidate = 0
               AND last_accessed <= ?1
             ORDER BY last_accessed ASC, id ASC
             LIMIT ?2",
        )?;

        let records = stmt
            .query_map(params![older_than_unix, limit], row_to_memory_record)?
            .collect::<rusqlite::Result<Vec<_>>>()?;
        Ok(records)
    }

    pub fn mark_low_heat_memories_as_archive_candidates(&self, threshold: i32) -> Result<Vec<i64>> {
        let mut stmt = self.db.prepare(
            "SELECT id
             FROM memories
             WHERE discard_candidate = 0
               AND is_archive_candidate = 0
               AND heat_i32 < ?1
             ORDER BY heat_i32 ASC, last_accessed ASC, id ASC",
        )?;

        let ids = stmt
            .query_map(params![threshold], |row| row.get::<_, i64>(0))?
            .collect::<rusqlite::Result<Vec<_>>>()?;

        if ids.is_empty() {
            return Ok(Vec::new());
        }

        let tx = self.db.unchecked_transaction()?;
        {
            let mut update = tx.prepare(
                "UPDATE memories
                 SET is_archive_candidate = 1
                 WHERE id = ?1
                   AND is_archive_candidate = 0",
            )?;

            for id in &ids {
                update.execute(params![id])?;
            }
        }
        tx.commit()?;

        Ok(ids)
    }

    pub fn list_archive_candidates(&self, limit: usize) -> Result<Vec<MemoryRecord>> {
        if limit == 0 {
            return Ok(Vec::new());
        }

        let limit = i64::try_from(limit).unwrap_or(i64::MAX);
        let mut stmt = self.db.prepare(
            "SELECT id, text_content, wing, room, source_file, valid_from, valid_to, last_accessed, access_count, importance_score, heat_i32, emotion_valence, emotion_arousal, is_archive_candidate, reason, discard_candidate, merged_into_id
             FROM memories
             WHERE is_archive_candidate = 1
               AND discard_candidate = 0
             ORDER BY heat_i32 ASC, last_accessed ASC, id ASC
             LIMIT ?1",
        )?;

        let records = stmt
            .query_map(params![limit], row_to_memory_record)?
            .collect::<rusqlite::Result<Vec<_>>>()?;
        Ok(records)
    }

    pub fn export_archive_candidates_to(&self, target_db_path: &Path) -> Result<usize> {
        let source_db_path = self.source_db_path()?;
        let target = Connection::open(target_db_path)?;
        ensure_memory_schema(&target)?;

        let escaped_source_path = source_db_path.to_string_lossy().replace('\'', "''");
        target.execute_batch(&format!(
            "ATTACH DATABASE '{escaped_source_path}' AS source;
             INSERT INTO memories (
                id,
                text_content,
                wing,
                room,
                source_file,
                source_mtime,
                valid_from,
                valid_to,
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
             SELECT
                id,
                text_content,
                wing,
                room,
                source_file,
                source_mtime,
                valid_from,
                valid_to,
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
             FROM source.memories
             WHERE is_archive_candidate = 1
               AND discard_candidate = 0
             ORDER BY heat_i32 ASC, last_accessed ASC, id ASC;
             DETACH DATABASE source;"
        ))?;

        let exported_count = target.query_row("SELECT COUNT(*) FROM memories", [], |row| {
            row.get::<_, i64>(0)
        })?;

        usize::try_from(exported_count).context("archive export count overflow")
    }

    pub fn update_heat_fields_if_unchanged(
        &self,
        id: i64,
        expected_last_accessed: i64,
        expected_access_count: u32,
        new_heat_i32: i32,
        is_archive_candidate: bool,
    ) -> Result<bool> {
        validate_heat_i32(new_heat_i32)?;
        let rows = self.db.execute(
            "UPDATE memories
             SET heat_i32 = ?1,
                 is_archive_candidate = ?2
             WHERE id = ?3
               AND last_accessed = ?4
               AND access_count = ?5",
            params![
                new_heat_i32,
                if is_archive_candidate { 1_i64 } else { 0_i64 },
                id,
                expected_last_accessed,
                i64::from(expected_access_count),
            ],
        )?;
        Ok(rows > 0)
    }

    pub fn run_heat_decay_pass(&self, heat_service: &HeatService) -> Result<usize> {
        self.run_heat_decay_pass_at(heat_service, Utc::now())
    }

    pub fn run_heat_decay_pass_at(
        &self,
        heat_service: &HeatService,
        now: DateTime<Utc>,
    ) -> Result<usize> {
        if !heat_service.config().enabled {
            return Ok(0);
        }

        let cutoff = now - Duration::hours(heat_service.config().update_interval_hours as i64);
        let candidates = self.list_decay_candidates(cutoff.timestamp(), usize::MAX)?;
        let mut updated = 0;

        for candidate in candidates {
            if self.apply_decay_to_candidate(candidate, heat_service, now, cutoff.timestamp())? {
                updated += 1;
            }
        }

        Ok(updated)
    }

    pub fn delete_memory(&self, id: i64) -> Result<()> {
        self.db
            .execute("DELETE FROM memories WHERE id = ?1", params![id])?;
        Ok(())
    }

    pub fn has_source_file(&self, source_file: &str) -> Result<bool> {
        let count: i64 = self.db.query_row(
            "SELECT COUNT(*) FROM memories WHERE source_file = ?1 LIMIT 1",
            params![source_file],
            |row| row.get(0),
        )?;
        Ok(count > 0)
    }

    pub fn get_wings_rooms(&self) -> Result<Vec<(String, String)>> {
        let mut stmt = self
            .db
            .prepare("SELECT DISTINCT wing, room FROM memories ORDER BY wing, room")?;
        let pairs = stmt
            .query_map([], |row| {
                Ok((row.get::<_, String>(0)?, row.get::<_, String>(1)?))
            })?
            .collect::<rusqlite::Result<Vec<_>>>()?;
        Ok(pairs)
    }

    pub fn save_index(&self, index_path: impl AsRef<Path>) -> Result<()> {
        let path = index_path
            .as_ref()
            .to_str()
            .ok_or_else(|| anyhow!("Non-UTF8 path"))?;
        self.index
            .save(path)
            .map_err(|e| anyhow!("Save failed: {e}"))
    }

    pub fn memory_count(&self) -> Result<u64> {
        self.db
            .query_row(
                "SELECT COUNT(*) FROM memories WHERE discard_candidate = 0",
                [],
                |row| row.get::<_, i64>(0),
            )
            .map(|n| n as u64)
            .context("Count failed")
    }

    pub fn index_size(&self) -> usize {
        self.index.size()
    }

    pub fn embed_single(&self, text: &str) -> Result<Vec<f32>> {
        let Some(embedder) = &self.embedder else {
            return self.embed_single_without_embedder(text);
        };
        let mut batch = embedder
            .embed(vec![text.to_string()], None)
            .context("fastembed failed")?;
        let vec = batch.pop().ok_or_else(|| anyhow!("Empty batch"))?;
        if vec.len() != VECTOR_DIMS {
            return Err(anyhow!("Expected {VECTOR_DIMS}-dim, got {}", vec.len()));
        }
        Ok(vec)
    }

    #[cfg(test)]
    fn embed_single_without_embedder(&self, text: &str) -> Result<Vec<f32>> {
        let mut vec = vec![0.0; VECTOR_DIMS];
        for (idx, byte) in text.bytes().enumerate() {
            vec[idx % VECTOR_DIMS] += byte as f32 / 255.0;
        }
        Ok(vec)
    }

    #[cfg(not(test))]
    fn embed_single_without_embedder(&self, _text: &str) -> Result<Vec<f32>> {
        Err(anyhow!(
            "Embeddings unavailable: fastembed model is not initialised"
        ))
    }

    fn semantic_candidate_ids(&self, filter: &SemanticSearchFilter) -> Result<Vec<u64>> {
        let mut sql = String::from("SELECT id FROM memories WHERE is_archive_candidate = 0");
        let mut params_dyn: Vec<Box<dyn rusqlite::ToSql>> = Vec::new();
        let mut next_idx = 1;

        if let Some(wing) = &filter.wing {
            sql.push_str(&format!(" AND wing = ?{next_idx}"));
            params_dyn.push(Box::new(wing.clone()));
            next_idx += 1;
        }

        if let Some(room) = &filter.room {
            sql.push_str(&format!(" AND room = ?{next_idx}"));
            params_dyn.push(Box::new(room.clone()));
            next_idx += 1;
        }

        if !filter.include_discarded {
            sql.push_str(&format!(" AND discard_candidate = ?{next_idx}"));
            params_dyn.push(Box::new(0_i64));
        }

        let mut stmt = self.db.prepare(&sql)?;
        let params_ref: Vec<&dyn rusqlite::ToSql> = params_dyn.iter().map(|p| p.as_ref()).collect();
        let ids = stmt
            .query_map(params_ref.as_slice(), |row| row.get::<_, i64>(0))?
            .collect::<rusqlite::Result<Vec<_>>>()?
            .into_iter()
            .map(|id| id as u64)
            .collect();
        Ok(ids)
    }

    fn materialize_semantic_results(
        &self,
        keys: Vec<u64>,
        distances: Vec<f32>,
        sort_by_heat: bool,
    ) -> Result<Vec<(MemoryRecord, f32)>> {
        if keys.is_empty() {
            return Ok(vec![]);
        }

        let id_placeholders: String = keys
            .iter()
            .enumerate()
            .map(|(i, _)| format!("?{}", i + 1))
            .collect::<Vec<_>>()
            .join(", ");

        let sql = format!(
            "SELECT id, text_content, wing, room, source_file, valid_from, valid_to, last_accessed, access_count, importance_score, heat_i32, emotion_valence, emotion_arousal, is_archive_candidate, reason, discard_candidate, merged_into_id
             FROM memories WHERE id IN ({id_placeholders})"
        );

        let mut stmt = self.db.prepare(&sql)?;
        let params_vec: Vec<&dyn rusqlite::ToSql> =
            keys.iter().map(|key| key as &dyn rusqlite::ToSql).collect();
        let mut record_map: std::collections::HashMap<i64, MemoryRecord> = stmt
            .query_map(params_vec.as_slice(), row_to_memory_record)?
            .collect::<rusqlite::Result<Vec<_>>>()?
            .into_iter()
            .map(|record| (record.id, record))
            .collect();

        let mut ordered: Vec<(MemoryRecord, f32)> = keys
            .iter()
            .zip(distances.iter())
            .filter_map(|(&key, &distance)| {
                record_map.remove(&(key as i64)).map(|mut record| {
                    let similarity = 1.0 - distance;
                    record.score = similarity;
                    (record, similarity)
                })
            })
            .collect();

        if sort_by_heat {
            ordered.sort_by(
                |(left_record, left_similarity), (right_record, right_similarity)| {
                    right_record
                        .heat_i32
                        .cmp(&left_record.heat_i32)
                        .then_with(|| {
                            right_similarity
                                .partial_cmp(left_similarity)
                                .unwrap_or(std::cmp::Ordering::Equal)
                        })
                        .then_with(|| right_record.valid_from.cmp(&left_record.valid_from))
                },
            );
        } else {
            ordered.sort_by(
                |(left_record, left_similarity), (right_record, right_similarity)| {
                    right_similarity
                        .partial_cmp(left_similarity)
                        .unwrap_or(std::cmp::Ordering::Equal)
                        .then_with(|| right_record.valid_from.cmp(&left_record.valid_from))
                        .then_with(|| right_record.heat_i32.cmp(&left_record.heat_i32))
                },
            );
        }

        Ok(ordered)
    }

    fn apply_decay_to_candidate(
        &self,
        mut record: MemoryRecord,
        heat_service: &HeatService,
        now: DateTime<Utc>,
        cutoff_unix: i64,
    ) -> Result<bool> {
        loop {
            let expected_last_accessed = datetime_to_unix_timestamp(&record.last_accessed);
            let expected_access_count = record.access_count;
            let recalculated_heat = heat_service.calculate_at(&record, now);
            let archive_candidate = heat_service
                .should_archive(recalculated_heat)
                .map_err(|error| anyhow!(error.to_string()))?;

            if self.update_heat_fields_if_unchanged(
                record.id,
                expected_last_accessed,
                expected_access_count,
                recalculated_heat,
                archive_candidate,
            )? {
                return Ok(true);
            }

            let refreshed = self.get_memory_by_id(record.id)?;
            if datetime_to_unix_timestamp(&refreshed.last_accessed) > cutoff_unix {
                return Ok(false);
            }

            if refreshed.access_count == record.access_count
                && datetime_to_unix_timestamp(&refreshed.last_accessed) == expected_last_accessed
            {
                return Ok(false);
            }

            record = refreshed;
        }
    }
}

fn validate_heat_i32(heat_i32: i32) -> Result<()> {
    if !(MIN_HEAT_I32..=MAX_HEAT_I32).contains(&heat_i32) {
        return Err(LaputaError::ValidationError(format!(
            "heat_i32 out of range [{MIN_HEAT_I32}, {MAX_HEAT_I32}]: {heat_i32}"
        ))
        .into());
    }

    Ok(())
}

#[cfg(test)]
mod tests {
    use super::{validate_heat_i32, MemoryInsert, VectorStorage};
    use rusqlite::Connection;
    use tempfile::tempdir;

    fn make_insert(heat_i32: i32) -> MemoryInsert<'static> {
        MemoryInsert {
            text_content: "guardrail",
            wing: "self",
            room: "journal",
            source_file: None,
            source_mtime: None,
            valid_from: 1,
            heat_i32,
            emotion_valence: 0,
            emotion_arousal: 0,
            is_archive_candidate: false,
            reason: None,
            discard_candidate: false,
            merged_into_id: None,
        }
    }

    #[test]
    fn test_validate_heat_i32_accepts_boundaries() {
        assert!(validate_heat_i32(0).is_ok());
        assert!(validate_heat_i32(10_000).is_ok());
    }

    #[test]
    fn test_add_memory_record_accepts_heat_boundaries_without_clamping() {
        for heat_i32 in [0, 10_000] {
            let dir = tempdir().unwrap();
            let db_path = dir.path().join("vectors.db");
            let index_path = dir.path().join("vectors.usearch");
            let conn = Connection::open(&db_path).unwrap();
            crate::storage::memory::ensure_memory_schema(&conn).unwrap();
            drop(conn);

            let mut store =
                VectorStorage::new_with_optional_embedder(&db_path, &index_path, None).unwrap();
            let row_id = store.add_memory_record(make_insert(heat_i32)).unwrap();
            let record = store.get_memory_by_id(row_id).unwrap();

            assert_eq!(record.heat_i32, heat_i32);
            assert_eq!(store.index_size(), 1);
        }
    }
}

fn row_to_memory_record(row: &rusqlite::Row<'_>) -> rusqlite::Result<MemoryRecord> {
    let last_accessed_unix: i64 = row.get(7)?;
    let access_count: u32 = row.get(8)?;
    let base_score: f32 = row.get(9)?;
    let last_accessed = unix_timestamp_to_datetime(last_accessed_unix).map_err(|err| {
        rusqlite::Error::FromSqlConversionFailure(
            7,
            rusqlite::types::Type::Integer,
            Box::new(std::io::Error::new(
                std::io::ErrorKind::InvalidData,
                err.to_string(),
            )),
        )
    })?;

    Ok(MemoryRecord {
        id: row.get(0)?,
        text_content: row.get(1)?,
        wing: row.get(2)?,
        room: row.get(3)?,
        source_file: row.get(4)?,
        valid_from: row.get(5)?,
        valid_to: row.get(6)?,
        score: 0.0,
        importance: compute_decayed_importance(base_score, last_accessed_unix, access_count as i64),
        heat_i32: row.get(10)?,
        last_accessed,
        access_count,
        emotion_valence: row.get(11)?,
        emotion_arousal: row.get(12)?,
        is_archive_candidate: row.get::<_, i64>(13)? != 0,
        reason: row.get(14)?,
        discard_candidate: row.get::<_, i64>(15)? != 0,
        merged_into_id: row.get(16)?,
    })
}

impl Drop for VectorStorage {
    fn drop(&mut self) {
        let _ = self.db.execute_batch("PRAGMA wal_checkpoint(TRUNCATE);");
    }
}
