use crate::config::MempalaceConfig;
use crate::dialect::Dialect;
use crate::knowledge_graph::{KnowledgeGraph, RelationChange};
use crate::rhythm::capsule::{
    CapsuleHotEvent, CapsuleRelationChange, RhythmCapsule, SummaryCapsule,
};
use crate::searcher::RecallQuery;
use crate::vector_storage::VectorStorage;
use anyhow::Result;
use chrono::{DateTime, Datelike, Duration, NaiveDate, Utc};
use regex::Regex;
use rusqlite::{params, Connection};
use std::collections::{HashMap, HashSet};
use std::fs;

const MIN_COMPLETE_RECORDS: usize = 7;
const HOT_EVENT_THRESHOLD: i32 = 5_000;
const MAX_HOT_EVENTS: usize = 12;
const MAX_RELATION_CHANGES: usize = 12;
const MAX_KEYWORDS: usize = 12;

pub struct WeeklyCapsuleGenerator {
    config: MempalaceConfig,
}

impl WeeklyCapsuleGenerator {
    pub fn new(config: MempalaceConfig) -> Self {
        Self { config }
    }

    pub fn generate_for_week(&self, anchor: DateTime<Utc>) -> Result<Option<SummaryCapsule>> {
        let (week_id, week_start, week_end) = week_bounds(anchor);
        let db_path = self.config.config_dir.join("vectors.db");
        if !db_path.exists() {
            return Ok(None);
        }

        let store = VectorStorage::new(&db_path, self.config.config_dir.join("vectors.usearch"))?;
        let records = store.recall_by_time_range(
            &RecallQuery::by_time_range(week_start.timestamp(), week_end.timestamp())
                .with_limit(1_000),
        )?;

        if records.is_empty() {
            return Ok(None);
        }

        let keywords = extract_keywords(&records);
        let dialect = Dialect::default();
        let hot_events = records
            .iter()
            .filter(|record| record.heat_i32 > HOT_EVENT_THRESHOLD)
            .take(MAX_HOT_EVENTS)
            .map(|record| CapsuleHotEvent {
                memory_id: record.id,
                wing: record.wing.clone(),
                room: record.room.clone(),
                heat_i32: record.heat_i32,
                summary: dialect
                    .compress(&record.text_content, None)
                    .replace('\n', " ")
                    .trim()
                    .to_string(),
            })
            .collect::<Vec<_>>();

        let relation_changes =
            self.load_relation_changes(week_start.date_naive(), week_end.date_naive())?;
        let incomplete = records.len() < MIN_COMPLETE_RECORDS;
        let created_at = Utc::now().to_rfc3339();
        let raw_text = build_raw_capsule_text(
            &week_id,
            &keywords,
            &records
                .iter()
                .map(|record| record.text_content.clone())
                .collect::<Vec<_>>(),
            &relation_changes,
            incomplete,
        );

        let mut metadata = HashMap::new();
        metadata.insert("wing".to_string(), "rhythm".to_string());
        metadata.insert("room".to_string(), "weekly_capsule".to_string());
        metadata.insert("date".to_string(), week_start.date_naive().to_string());
        metadata.insert("source_file".to_string(), format!("{week_id}.md"));
        let compressed_content = dialect.compress_propositions(&raw_text, Some(metadata), 8, 9);
        let stats = dialect.compression_stats(&raw_text, &compressed_content);

        let capsule = SummaryCapsule {
            week_id: week_id.clone(),
            week_start: week_start.date_naive().to_string(),
            week_end: week_end.date_naive().to_string(),
            keywords,
            hot_events,
            relation_changes,
            source_record_count: records.len(),
            original_tokens: stats["original_tokens_est"].as_u64().unwrap_or(0) as usize,
            compressed_tokens: stats["summary_tokens_est"].as_u64().unwrap_or(0) as usize,
            compression_ratio: stats["size_ratio"].as_f64().unwrap_or(1.0),
            created_at,
            incomplete,
            compressed_content,
        };

        self.persist_capsule(&capsule, week_start.timestamp(), week_end.timestamp())?;
        self.write_capsule_files(&capsule)?;

        Ok(Some(capsule))
    }

    pub fn load_latest_from_db(config: &MempalaceConfig) -> Option<RhythmCapsule> {
        let db_path = config.config_dir.join("vectors.db");
        let conn = Connection::open(db_path).ok()?;
        let mut stmt = conn
            .prepare(
                "SELECT week_id, markdown_content
                 FROM capsules
                 ORDER BY week_start DESC, created_at DESC
                 LIMIT 1",
            )
            .ok()?;

        stmt.query_row([], |row| {
            Ok(RhythmCapsule {
                source: format!("sqlite:capsules:{}", row.get::<_, String>(0)?),
                content: row.get(1)?,
            })
        })
        .ok()
    }

    fn load_relation_changes(
        &self,
        week_start: NaiveDate,
        week_end: NaiveDate,
    ) -> Result<Vec<CapsuleRelationChange>> {
        let knowledge_path = self.config.config_dir.join("knowledge.db");
        if !knowledge_path.exists() {
            return Ok(vec![]);
        }

        let graph = KnowledgeGraph::new(knowledge_path.to_str().unwrap_or("knowledge.db"))?;
        let changes = graph.relation_changes_between(
            &week_start.to_string(),
            &week_end.to_string(),
            10,
            MAX_RELATION_CHANGES,
        )?;

        Ok(changes
            .into_iter()
            .map(map_relation_change)
            .collect::<Vec<_>>())
    }

    fn persist_capsule(
        &self,
        capsule: &SummaryCapsule,
        week_start: i64,
        week_end: i64,
    ) -> Result<()> {
        let db_path = self.config.config_dir.join("vectors.db");
        let conn = Connection::open(db_path)?;
        crate::storage::memory::ensure_memory_schema(&conn)?;

        let capsule_json = serde_json::to_string(capsule)?;
        let markdown_content = capsule.render_markdown();

        conn.execute(
            "INSERT INTO capsules (
                week_id,
                week_start,
                week_end,
                capsule_json,
                markdown_content,
                compressed_content,
                original_tokens,
                summary_tokens,
                compression_ratio,
                incomplete,
                created_at
             ) VALUES (?1, ?2, ?3, ?4, ?5, ?6, ?7, ?8, ?9, ?10, ?11)
             ON CONFLICT(week_id) DO UPDATE SET
                week_start = excluded.week_start,
                week_end = excluded.week_end,
                capsule_json = excluded.capsule_json,
                markdown_content = excluded.markdown_content,
                compressed_content = excluded.compressed_content,
                original_tokens = excluded.original_tokens,
                summary_tokens = excluded.summary_tokens,
                compression_ratio = excluded.compression_ratio,
                incomplete = excluded.incomplete,
                created_at = excluded.created_at",
            params![
                capsule.week_id,
                week_start,
                week_end,
                capsule_json,
                markdown_content,
                capsule.compressed_content,
                capsule.original_tokens as i64,
                capsule.compressed_tokens as i64,
                capsule.compression_ratio,
                capsule.incomplete as i64,
                capsule.created_at,
            ],
        )?;

        Ok(())
    }

    fn write_capsule_files(&self, capsule: &SummaryCapsule) -> Result<()> {
        let rhythm_dir = self.config.config_dir.join("rhythm");
        fs::create_dir_all(&rhythm_dir)?;
        let markdown = capsule.render_markdown();
        fs::write(rhythm_dir.join("latest-weekly-capsule.md"), &markdown)?;
        fs::write(
            rhythm_dir.join(format!("weekly-capsule-{}.md", capsule.week_id)),
            markdown,
        )?;
        Ok(())
    }
}

fn map_relation_change(change: RelationChange) -> CapsuleRelationChange {
    CapsuleRelationChange {
        subject: change.subject,
        predicate: change.predicate,
        object: change.object,
        previous_resonance: change.previous_resonance,
        current_resonance: change.current_resonance,
        delta: change.delta,
        valid_from: change.valid_from,
        source_file: change.source_file,
    }
}

fn build_raw_capsule_text(
    week_id: &str,
    keywords: &[String],
    texts: &[String],
    relation_changes: &[CapsuleRelationChange],
    incomplete: bool,
) -> String {
    let keyword_line = if keywords.is_empty() {
        "none".to_string()
    } else {
        keywords.join(", ")
    };

    let relation_lines = if relation_changes.is_empty() {
        "No meaningful relation shifts this week.".to_string()
    } else {
        relation_changes
            .iter()
            .map(|change| {
                format!(
                    "{} {} {} moved by {} points (now {}).",
                    change.subject,
                    change.predicate,
                    change.object,
                    change.delta,
                    change.current_resonance
                )
            })
            .collect::<Vec<_>>()
            .join(" ")
    };

    format!(
        "Weekly capsule for {week_id}. Incomplete week: {incomplete}. Keywords: {keyword_line}. \
         Source material: {texts}. Relation shifts: {relation_lines}",
        texts = texts.join(" "),
    )
}

fn extract_keywords(records: &[crate::vector_storage::MemoryRecord]) -> Vec<String> {
    let token_re = Regex::new(r"[A-Za-z][A-Za-z_-]{2,}").expect("keyword regex must compile");
    let stop_words = stop_words();
    let mut scores: HashMap<String, f64> = HashMap::new();

    for record in records {
        let weight = 1.0
            + (record.heat_i32 as f64 / 10_000.0)
            + (record.emotion_valence.max(0) as f64 / 100.0);
        for token in token_re.find_iter(&record.text_content) {
            let word = token.as_str().to_lowercase();
            if stop_words.contains(word.as_str()) {
                continue;
            }
            *scores.entry(word).or_insert(0.0) += weight;
        }
    }

    let mut ranked = scores.into_iter().collect::<Vec<_>>();
    ranked.sort_by(|left, right| {
        right
            .1
            .partial_cmp(&left.1)
            .unwrap_or(std::cmp::Ordering::Equal)
            .then_with(|| left.0.cmp(&right.0))
    });

    ranked
        .into_iter()
        .take(MAX_KEYWORDS)
        .map(|(word, _)| word)
        .collect()
}

fn stop_words() -> HashSet<&'static str> {
    [
        "about",
        "after",
        "and",
        "across",
        "before",
        "capsule",
        "clarity",
        "confirmed",
        "deepened",
        "delivered",
        "documented",
        "during",
        "easier",
        "final",
        "focus",
        "flow",
        "gains",
        "high",
        "integration",
        "made",
        "memory",
        "notes",
        "output",
        "quality",
        "reflection",
        "review",
        "signals",
        "stable",
        "strong",
        "team",
        "that",
        "the",
        "this",
        "tracked",
        "trust",
        "weekly",
        "while",
        "with",
        "work",
    ]
    .into_iter()
    .collect()
}

fn week_bounds(anchor: DateTime<Utc>) -> (String, DateTime<Utc>, DateTime<Utc>) {
    let weekday = anchor.weekday().num_days_from_monday() as i64;
    let start = anchor
        .date_naive()
        .and_hms_opt(0, 0, 0)
        .expect("midnight must be valid")
        - Duration::days(weekday);
    let start = DateTime::<Utc>::from_naive_utc_and_offset(start, Utc);
    let end = start + Duration::days(7) - Duration::seconds(1);
    let iso = start.iso_week();
    (format!("{}-W{:02}", iso.year(), iso.week()), start, end)
}

#[cfg(test)]
mod tests {
    use super::*;
    use chrono::TimeZone;

    #[test]
    fn test_week_bounds_align_to_iso_week() {
        let anchor = Utc.with_ymd_and_hms(2026, 4, 8, 12, 0, 0).unwrap();
        let (week_id, start, end) = week_bounds(anchor);
        assert_eq!(week_id, "2026-W15");
        assert_eq!(start.date_naive().to_string(), "2026-04-06");
        assert_eq!(end.date_naive().to_string(), "2026-04-12");
    }
}
