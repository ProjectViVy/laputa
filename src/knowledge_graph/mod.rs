pub mod relation;
pub mod resonance;

use crate::api::LaputaError;
pub use relation::{RelationChange, RelationKind, RelationRecord, ResonantRelation};
pub use resonance::Resonance;
use rusqlite::{params, Connection, OptionalExtension, Result};
use serde_json::{json, Value};
use std::path::Path;

pub struct KnowledgeGraph {
    conn: Connection,
}

impl KnowledgeGraph {
    pub fn new(path: &str) -> Result<Self> {
        if path != ":memory:" {
            if let Some(parent) = Path::new(path).parent() {
                let _ = std::fs::create_dir_all(parent);
            }
        }
        let conn = Connection::open(path)?;
        let kg = KnowledgeGraph { conn };
        kg._init_db()?;
        Ok(kg)
    }

    fn _init_db(&self) -> Result<()> {
        self.conn.execute_batch(
            "CREATE TABLE IF NOT EXISTS entities (
                id TEXT PRIMARY KEY,
                name TEXT NOT NULL,
                type TEXT DEFAULT 'unknown',
                properties TEXT DEFAULT '{}',
                created_at TEXT DEFAULT CURRENT_TIMESTAMP
            );

            CREATE TABLE IF NOT EXISTS triples (
                id TEXT PRIMARY KEY,
                subject TEXT NOT NULL,
                predicate TEXT NOT NULL,
                object TEXT NOT NULL,
                valid_from TEXT,
                valid_to TEXT,
                confidence REAL DEFAULT 1.0,
                source_closet TEXT,
                source_file TEXT,
                extracted_at TEXT DEFAULT CURRENT_TIMESTAMP,
                FOREIGN KEY (subject) REFERENCES entities(id),
                FOREIGN KEY (object) REFERENCES entities(id)
            );

            CREATE INDEX IF NOT EXISTS idx_triples_subject ON triples(subject);
            CREATE INDEX IF NOT EXISTS idx_triples_object ON triples(object);
            CREATE INDEX IF NOT EXISTS idx_triples_predicate ON triples(predicate);
            CREATE INDEX IF NOT EXISTS idx_triples_valid ON triples(valid_from, valid_to);",
        )?;
        Ok(())
    }

    fn _entity_id(&self, name: &str) -> String {
        name.to_lowercase().replace(' ', "_").replace('\'', "")
    }

    pub fn add_entity(
        &self,
        name: &str,
        entity_type: &str,
        properties: Option<Value>,
    ) -> Result<String> {
        let eid = self._entity_id(name);
        let props = properties.unwrap_or_else(|| json!({})).to_string();
        self.conn.execute(
            "INSERT OR REPLACE INTO entities (id, name, type, properties) VALUES (?1, ?2, ?3, ?4)",
            params![eid, name, entity_type, props],
        )?;
        Ok(eid)
    }

    #[allow(clippy::too_many_arguments)]
    pub fn add_triple(
        &self,
        subject: &str,
        predicate: &str,
        obj: &str,
        valid_from: Option<&str>,
        valid_to: Option<&str>,
        confidence: f64,
        source_closet: Option<&str>,
        source_file: Option<&str>,
    ) -> Result<String> {
        let sub_id = self._entity_id(subject);
        let obj_id = self._entity_id(obj);
        let pred = predicate.to_lowercase().replace(' ', "_");

        // Auto-create entities if they don't exist
        self.conn.execute(
            "INSERT OR IGNORE INTO entities (id, name) VALUES (?1, ?2)",
            params![sub_id, subject],
        )?;
        self.conn.execute(
            "INSERT OR IGNORE INTO entities (id, name) VALUES (?1, ?2)",
            params![obj_id, obj],
        )?;

        // Check for existing identical triple
        let mut stmt = self.conn.prepare(
            "SELECT id FROM triples WHERE subject=?1 AND predicate=?2 AND object=?3 AND valid_to IS NULL"
        )?;
        let mut rows = stmt.query(params![sub_id, pred, obj_id])?;
        if let Some(row) = rows.next()? {
            return row.get(0);
        }

        let triple_id = format!(
            "t_{}_{}_{}_{}",
            sub_id,
            pred,
            obj_id,
            &self.hash_now(valid_from)
        );

        self.conn.execute(
            "INSERT INTO triples (id, subject, predicate, object, valid_from, valid_to, confidence, source_closet, source_file)
             VALUES (?1, ?2, ?3, ?4, ?5, ?6, ?7, ?8, ?9)",
            params![
                triple_id,
                sub_id,
                pred,
                obj_id,
                valid_from,
                valid_to,
                confidence,
                source_closet,
                source_file,
            ],
        )?;
        Ok(triple_id)
    }

    fn hash_now(&self, seed: Option<&str>) -> String {
        use std::collections::hash_map::DefaultHasher;
        use std::hash::{Hash, Hasher};
        use std::time::SystemTime;

        let mut hasher = DefaultHasher::new();
        seed.unwrap_or("").hash(&mut hasher);
        SystemTime::now().hash(&mut hasher);
        format!("{:x}", hasher.finish())[..8].to_string()
    }

    pub fn invalidate(
        &self,
        subject: &str,
        predicate: &str,
        obj: &str,
        ended: Option<&str>,
    ) -> Result<()> {
        let sub_id = self._entity_id(subject);
        let obj_id = self._entity_id(obj);
        let pred = predicate.to_lowercase().replace(' ', "_");
        let end_date = ended.map(|s| s.to_string()).unwrap_or_else(|| {
            use chrono::Local;
            Local::now().format("%Y-%m-%d").to_string()
        });

        self.conn.execute(
            "UPDATE triples SET valid_to=?1 WHERE subject=?2 AND predicate=?3 AND object=?4 AND valid_to IS NULL",
            params![end_date, sub_id, pred, obj_id],
        )?;
        Ok(())
    }

    pub fn query_entity(
        &self,
        name: &str,
        as_of: Option<&str>,
        direction: &str,
    ) -> Result<Vec<Value>> {
        let eid = self._entity_id(name);
        let mut results = Vec::new();

        if direction == "outgoing" || direction == "both" {
            let mut query = "SELECT t.id, t.subject, t.predicate, t.object, t.valid_from, t.valid_to, t.confidence, t.source_closet, t.source_file, t.extracted_at, e.name as obj_name FROM triples t JOIN entities e ON t.object = e.id WHERE t.subject = ?1".to_string();
            let mut params_vec: Vec<String> = vec![eid.clone()];
            if let Some(date) = as_of {
                query += " AND (t.valid_from IS NULL OR t.valid_from <= ?2) AND (t.valid_to IS NULL OR t.valid_to >= ?3)";
                params_vec.push(date.to_string());
                params_vec.push(date.to_string());
            }

            let mut stmt = self.conn.prepare(&query)?;
            let rows = stmt.query_map(rusqlite::params_from_iter(params_vec.iter()), |row| {
                Ok(json!({
                    "direction": "outgoing",
                    "subject": name,
                    "predicate": row.get::<_, String>(2)?,
                    "object": row.get::<_, String>(10)?,
                    "valid_from": row.get::<_, Option<String>>(4)?,
                    "valid_to": row.get::<_, Option<String>>(5)?,
                    "confidence": row.get::<_, f64>(6)?,
                    "source_closet": row.get::<_, Option<String>>(7)?,
                    "current": row.get::<_, Option<String>>(5)?.is_none(),
                }))
            })?;

            for row in rows {
                results.push(row?);
            }
        }

        if direction == "incoming" || direction == "both" {
            let mut query = "SELECT t.id, t.subject, t.predicate, t.object, t.valid_from, t.valid_to, t.confidence, t.source_closet, t.source_file, t.extracted_at, e.name as sub_name FROM triples t JOIN entities e ON t.subject = e.id WHERE t.object = ?1".to_string();
            let mut params_vec: Vec<String> = vec![eid.clone()];
            if let Some(date) = as_of {
                query += " AND (t.valid_from IS NULL OR t.valid_from <= ?2) AND (t.valid_to IS NULL OR t.valid_to >= ?3)";
                params_vec.push(date.to_string());
                params_vec.push(date.to_string());
            }

            let mut stmt = self.conn.prepare(&query)?;
            let rows = stmt.query_map(rusqlite::params_from_iter(params_vec.iter()), |row| {
                Ok(json!({
                    "direction": "incoming",
                    "subject": row.get::<_, String>(10)?,
                    "predicate": row.get::<_, String>(2)?,
                    "object": name,
                    "valid_from": row.get::<_, Option<String>>(4)?,
                    "valid_to": row.get::<_, Option<String>>(5)?,
                    "confidence": row.get::<_, f64>(6)?,
                    "source_closet": row.get::<_, Option<String>>(7)?,
                    "current": row.get::<_, Option<String>>(5)?.is_none(),
                }))
            })?;

            for row in rows {
                results.push(row?);
            }
        }

        Ok(results)
    }

    /// 创建或更新主体关系，并保留时间线。
    #[allow(clippy::too_many_arguments)]
    pub fn upsert_relation(
        &self,
        subject: &str,
        object: &str,
        relation_type: RelationKind,
        resonance: i32,
        valid_from: Option<&str>,
        source_closet: Option<&str>,
        source_file: Option<&str>,
    ) -> std::result::Result<RelationRecord, LaputaError> {
        let resonance = Resonance::new(resonance)?;
        let sub_id = self.add_entity(subject, "unknown", None)?;
        let obj_id = self.add_entity(object, "unknown", None)?;
        let predicate = relation_type.as_str();

        let existing = self
            .conn
            .query_row(
                "SELECT sub.name, t.predicate, obj.name, t.confidence, t.valid_from, t.valid_to, t.source_closet, t.source_file
                 FROM triples t
                 JOIN entities sub ON t.subject = sub.id
                 JOIN entities obj ON t.object = obj.id
                 WHERE t.subject = ?1
                   AND t.object = ?2
                   AND t.valid_to IS NULL
                   AND t.predicate IN (?3, ?4, ?5)
                 ORDER BY COALESCE(t.valid_from, t.extracted_at) DESC
                 LIMIT 1",
                params![
                    sub_id,
                    obj_id,
                    RelationKind::all()[0],
                    RelationKind::all()[1],
                    RelationKind::all()[2]
                ],
                RelationRecord::from_row,
            )
            .optional()?;

        if let Some(current) = existing {
            if current.relation_type == relation_type && current.resonance == resonance.value() {
                return Ok(current);
            }

            self.close_current_relation_pair(subject, object, valid_from)?;
        }

        let triple_id = format!(
            "r_{}_{}_{}_{}",
            self._entity_id(subject),
            predicate,
            self._entity_id(object),
            self.hash_now(valid_from)
        );

        self.conn.execute(
            "INSERT INTO triples (id, subject, predicate, object, valid_from, valid_to, confidence, source_closet, source_file)
             VALUES (?1, ?2, ?3, ?4, ?5, NULL, ?6, ?7, ?8)",
            params![
                triple_id,
                self._entity_id(subject),
                predicate,
                self._entity_id(object),
                valid_from,
                resonance.as_confidence(),
                source_closet,
                source_file,
            ],
        )?;

        Ok(RelationRecord {
            subject: subject.to_string(),
            object: object.to_string(),
            relation_type,
            resonance: resonance.value(),
            valid_from: valid_from.map(str::to_string),
            valid_to: None,
            source_closet: source_closet.map(str::to_string),
            source_file: source_file.map(str::to_string),
            current: true,
        })
    }

    /// 返回某个实体当前有效的关系。
    pub fn get_current_relations(
        &self,
        entity: &str,
    ) -> std::result::Result<Vec<RelationRecord>, LaputaError> {
        self.query_relations(entity, true)
    }

    /// 返回某个实体的关系时间线，包括历史记录。
    pub fn get_relation_timeline(
        &self,
        entity: &str,
    ) -> std::result::Result<Vec<RelationRecord>, LaputaError> {
        self.query_relations(entity, false)
    }

    pub fn top_relations(&self, min_resonance: i32, limit: usize) -> Result<Vec<ResonantRelation>> {
        let mut stmt = self.conn.prepare(
            "SELECT sub.name, t.predicate, obj.name, t.confidence, t.valid_from, t.source_file
             FROM triples t
             JOIN entities sub ON t.subject = sub.id
             JOIN entities obj ON t.object = obj.id
             WHERE t.valid_to IS NULL
               AND t.predicate IN (?1, ?2, ?3)
             ORDER BY t.confidence DESC, t.valid_from DESC
             LIMIT ?4",
        )?;

        let rows = stmt.query_map(
            params![
                RelationKind::all()[0],
                RelationKind::all()[1],
                RelationKind::all()[2],
                limit as i64
            ],
            |row| {
                let confidence: f64 = row.get(3)?;
                let predicate: String = row.get(1)?;
                let relation_type = predicate.parse::<RelationKind>().ok().ok_or_else(|| {
                    rusqlite::Error::InvalidColumnType(
                        1,
                        "predicate".to_string(),
                        rusqlite::types::Type::Text,
                    )
                })?;
                Ok(ResonantRelation {
                    subject: row.get(0)?,
                    predicate,
                    object: row.get(2)?,
                    relation_type,
                    resonance: confidence_to_resonance(confidence),
                    confidence,
                    valid_from: row.get(4)?,
                    source_file: row.get(5)?,
                })
            },
        )?;

        let mut relations = Vec::new();
        for row in rows {
            let relation = row?;
            if relation.resonance > min_resonance {
                relations.push(relation);
            }
        }

        Ok(relations)
    }

    pub fn relation_changes_between(
        &self,
        start_date: &str,
        end_date: &str,
        min_delta: i32,
        limit: usize,
    ) -> Result<Vec<RelationChange>> {
        let mut stmt = self.conn.prepare(
            "SELECT sub.name, t.predicate, obj.name, t.confidence, t.valid_from, t.source_file
             FROM triples t
             JOIN entities sub ON t.subject = sub.id
             JOIN entities obj ON t.object = obj.id
             WHERE COALESCE(t.valid_from, t.extracted_at) >= ?1
               AND COALESCE(t.valid_from, t.extracted_at) <= ?2
               AND t.predicate IN (?3, ?4, ?5)
             ORDER BY COALESCE(t.valid_from, t.extracted_at) DESC, t.confidence DESC",
        )?;

        let rows = stmt.query_map(
            params![
                start_date,
                end_date,
                RelationKind::all()[0],
                RelationKind::all()[1],
                RelationKind::all()[2]
            ],
            |row| {
                Ok((
                    row.get::<_, String>(0)?,
                    row.get::<_, String>(1)?,
                    row.get::<_, String>(2)?,
                    row.get::<_, f64>(3)?,
                    row.get::<_, Option<String>>(4)?,
                    row.get::<_, Option<String>>(5)?,
                ))
            },
        )?;

        let mut changes = Vec::new();
        for row in rows {
            let (subject, predicate, object, confidence, valid_from, source_file) = row?;
            let current_resonance = confidence_to_resonance(confidence);

            let previous_confidence: Option<f64> = self
                .conn
                .query_row(
                    "SELECT confidence
                     FROM triples
                     WHERE subject = ?1
                       AND object = ?2
                       AND predicate IN (?4, ?5, ?6)
                       AND COALESCE(valid_from, extracted_at) < ?3
                     ORDER BY COALESCE(valid_from, extracted_at) DESC
                     LIMIT 1",
                    params![
                        self._entity_id(&subject),
                        self._entity_id(&object),
                        valid_from.clone().unwrap_or_else(|| end_date.to_string()),
                        RelationKind::all()[0],
                        RelationKind::all()[1],
                        RelationKind::all()[2]
                    ],
                    |row| row.get(0),
                )
                .optional()?;

            let previous_resonance = previous_confidence.map(confidence_to_resonance);
            let delta = previous_resonance
                .map(|previous: i32| (current_resonance - previous).abs())
                .unwrap_or(current_resonance.abs());

            if previous_resonance.is_none() || delta > min_delta {
                changes.push(RelationChange {
                    subject,
                    predicate,
                    object,
                    previous_resonance,
                    current_resonance,
                    delta,
                    valid_from,
                    source_file,
                });
            }

            if changes.len() >= limit {
                break;
            }
        }

        Ok(changes)
    }

    pub fn stats(&self) -> Result<Value> {
        let mut entity_count: i64 = 0;
        let mut triple_count: i64 = 0;

        self.conn
            .query_row("SELECT COUNT(*) FROM entities", [], |row| {
                entity_count = row.get(0)?;
                Ok(())
            })?;

        self.conn
            .query_row("SELECT COUNT(*) FROM triples", [], |row| {
                triple_count = row.get(0)?;
                Ok(())
            })?;

        Ok(json!({
            "entities": entity_count,
            "triples": triple_count,
            "status": "active"
        }))
    }

    fn query_relations(
        &self,
        entity: &str,
        current_only: bool,
    ) -> std::result::Result<Vec<RelationRecord>, LaputaError> {
        let eid = self._entity_id(entity);
        let mut query = String::from(
            "SELECT sub.name, t.predicate, obj.name, t.confidence, t.valid_from, t.valid_to, t.source_closet, t.source_file
             FROM triples t
             JOIN entities sub ON t.subject = sub.id
             JOIN entities obj ON t.object = obj.id
             WHERE (t.subject = ?1 OR t.object = ?1)
               AND t.predicate IN (?2, ?3, ?4)",
        );
        if current_only {
            query.push_str(" AND t.valid_to IS NULL");
        }
        query.push_str(" ORDER BY COALESCE(t.valid_from, t.extracted_at) ASC, t.extracted_at ASC");

        let mut stmt = self.conn.prepare(&query)?;
        let rows = stmt.query_map(
            params![
                eid,
                RelationKind::all()[0],
                RelationKind::all()[1],
                RelationKind::all()[2]
            ],
            RelationRecord::from_row,
        )?;

        let mut records = Vec::new();
        for row in rows {
            records.push(row?);
        }
        Ok(records)
    }

    fn close_current_relation_pair(
        &self,
        subject: &str,
        object: &str,
        ended: Option<&str>,
    ) -> Result<()> {
        let end_date = ended.map(|s| s.to_string()).unwrap_or_else(|| {
            use chrono::Local;
            Local::now().format("%Y-%m-%d").to_string()
        });

        self.conn.execute(
            "UPDATE triples
             SET valid_to = ?1
             WHERE subject = ?2
               AND object = ?3
               AND valid_to IS NULL
               AND predicate IN (?4, ?5, ?6)",
            params![
                end_date,
                self._entity_id(subject),
                self._entity_id(object),
                RelationKind::all()[0],
                RelationKind::all()[1],
                RelationKind::all()[2]
            ],
        )?;
        Ok(())
    }
}

fn confidence_to_resonance(confidence: f64) -> i32 {
    let scaled = if confidence <= 1.0 {
        (confidence * 100.0).round()
    } else {
        confidence.round()
    };

    scaled.clamp(-100.0, 100.0) as i32
}
