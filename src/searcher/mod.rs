mod hybrid;
mod recall;

use crate::config::MempalaceConfig;
use crate::vector_storage::{EmotionQuery, MemoryRecord, VectorStorage};
use crate::wakeup::WakePackGenerator;
use anyhow::Result;
use std::collections::HashMap;
use std::path::PathBuf;

pub use hybrid::{
    compute_composite_score, load_hybrid_ranking_config, merge_hybrid_results,
    normalize_heat_score, normalize_time_score, HybridQuery, HybridRankingConfig,
    HybridSearchResult,
};
pub use recall::RecallQuery;

// Note: Custom VectorStorage (fastembed + usearch + rusqlite) is used.

/// High-level search interface for retrieving context from the Palace.
pub struct Searcher {
    pub config: MempalaceConfig,
}

#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct SemanticSearchOptions {
    pub wing: Option<String>,
    pub room: Option<String>,
    pub include_discarded: bool,
    pub sort_by_heat: bool,
}

#[derive(Debug, Clone, PartialEq)]
pub struct SearchResult {
    pub record: MemoryRecord,
    pub similarity: f32,
    pub rank: usize,
}

impl Searcher {
    pub fn new(config: MempalaceConfig) -> Self {
        Searcher { config }
    }

    fn open_vector_storage(&self) -> Option<VectorStorage> {
        VectorStorage::new(
            self.config.config_dir.join("vectors.db"),
            self.config.config_dir.join("vectors.usearch"),
        )
        .ok()
    }

    pub fn add_memory(
        &self,
        text: &str,
        wing: &str,
        room: &str,
        source_file: Option<&str>,
        source_mtime: Option<f64>,
    ) -> Result<i64> {
        let Some(mut store) = self.open_vector_storage() else {
            return Err(anyhow::anyhow!(
                "Vector storage unavailable; memory was not persisted"
            ));
        };
        let id = store
            .add_memory(text, wing, room, source_file, source_mtime)
            .map_err(|_| anyhow::anyhow!("Vector storage unavailable; memory was not persisted"))?;
        store.save_index(self.config.config_dir.join("vectors.usearch"))?;
        Ok(id)
    }

    pub fn delete_memory(&self, memory_id: i64) -> Result<()> {
        let Some(store) = self.open_vector_storage() else {
            let _ = memory_id;
            return Err(anyhow::anyhow!(
                "Vector storage unavailable; memory deletion was not persisted"
            ));
        };
        store.delete_memory(memory_id)?;
        store.save_index(self.config.config_dir.join("vectors.usearch"))?;
        Ok(())
    }

    pub fn update_memory_emotion(
        &self,
        memory_id: i64,
        valence: i32,
        arousal: u32,
    ) -> Result<MemoryRecord> {
        let Some(store) = self.open_vector_storage() else {
            return Err(anyhow::anyhow!(
                "Vector storage unavailable; emotion update was not persisted"
            ));
        };
        store.update_memory_emotion(memory_id, valence, arousal)
    }

    pub fn list_memories_by_emotion(&self, query: &EmotionQuery) -> Result<Vec<MemoryRecord>> {
        let Some(store) = self.open_vector_storage() else {
            return Ok(vec![]);
        };
        store.list_memories_by_emotion(query)
    }

    pub async fn wake_up(&self, wing: Option<String>) -> Result<String> {
        WakePackGenerator::new(self.config.clone()).generate_json(wing)
    }

    pub async fn recall_by_time_range(&self, query: RecallQuery) -> Result<Vec<MemoryRecord>> {
        let Some(store) = self.open_vector_storage() else {
            return Ok(vec![]);
        };

        let records = store.recall_by_time_range(&query)?;
        for record in &records {
            let _ = store.touch_memory(record.id);
        }
        Ok(records)
    }

    pub async fn semantic_search(
        &self,
        query: &str,
        top_k: usize,
        options: SemanticSearchOptions,
    ) -> Result<Vec<SearchResult>> {
        if top_k == 0 {
            return Ok(vec![]);
        }

        let Some(store) = self.open_vector_storage() else {
            return Ok(vec![]);
        };

        let query = truncate_embedding_input(query);
        let query_vector = match store.embed_single(&query) {
            Ok(vector) => vector,
            Err(_) => return Ok(vec![]),
        };

        let results = store.semantic_search(
            &query_vector,
            top_k,
            options.wing.as_deref(),
            options.room.as_deref(),
            options.include_discarded,
            options.sort_by_heat,
        )?;

        for (record, _) in &results {
            let _ = store.touch_memory(record.id);
        }

        Ok(results
            .into_iter()
            .enumerate()
            .map(|(index, (mut record, similarity))| {
                record.score = similarity;
                SearchResult {
                    record,
                    similarity,
                    rank: index + 1,
                }
            })
            .collect())
    }

    pub async fn hybrid_search(&self, mut query: HybridQuery) -> Result<Vec<HybridSearchResult>> {
        if query.ranking_config == HybridRankingConfig::default() {
            if let Some(config) = load_hybrid_ranking_config(&self.config.config_dir) {
                query.ranking_config = config;
            }
        }

        let time_results = self
            .recall_by_time_range(query.recall_query.clone())
            .await
            .unwrap_or_default();

        let semantic_results = if query.semantic_query.trim().is_empty() {
            vec![]
        } else {
            self.semantic_search(
                &query.semantic_query,
                query.semantic_limit,
                SemanticSearchOptions {
                    wing: query.recall_query.wing.clone(),
                    room: query.recall_query.room.clone(),
                    include_discarded: query.recall_query.include_discarded,
                    sort_by_heat: false,
                },
            )
            .await
            .unwrap_or_default()
        };

        Ok(merge_hybrid_results(&query, time_results, semantic_results))
    }

    pub fn merge_hybrid_results(
        query: &HybridQuery,
        time_results: Vec<MemoryRecord>,
        semantic_results: Vec<SearchResult>,
    ) -> Vec<HybridSearchResult> {
        hybrid::merge_hybrid_results(query, time_results, semantic_results)
    }

    pub fn build_where_clause(
        wing: Option<&String>,
        room: Option<&String>,
    ) -> Option<serde_json::Value> {
        let mut where_clause = HashMap::<String, serde_json::Value>::new();
        if let (Some(w), Some(r)) = (wing, room) {
            let mut and_vec = Vec::new();
            let mut w_map = HashMap::<String, serde_json::Value>::new();
            w_map.insert("wing".to_string(), serde_json::Value::String(w.to_string()));
            and_vec.push(serde_json::Value::Object(w_map.into_iter().collect()));

            let mut r_map = HashMap::<String, serde_json::Value>::new();
            r_map.insert("room".to_string(), serde_json::Value::String(r.to_string()));
            and_vec.push(serde_json::Value::Object(r_map.into_iter().collect()));

            where_clause.insert("$and".to_string(), serde_json::Value::Array(and_vec));
        } else if let Some(w) = wing {
            where_clause.insert("wing".to_string(), serde_json::Value::String(w.to_string()));
        } else if let Some(r) = room {
            where_clause.insert("room".to_string(), serde_json::Value::String(r.to_string()));
        }

        if where_clause.is_empty() {
            None
        } else {
            Some(serde_json::to_value(where_clause).unwrap())
        }
    }

    pub fn format_search_results(
        query: &str,
        wing: Option<&String>,
        room: Option<&String>,
        docs: &[String],
        metas: &[Option<serde_json::Map<String, serde_json::Value>>],
        dists: &[f32],
    ) -> String {
        if docs.is_empty() || docs[0].is_empty() {
            return format!("\n  No results found for: \"{}\"", query);
        }

        let mut output = String::new();
        output.push_str(&format!("\n{}", "=".repeat(60)));
        output.push_str(&format!("\n  Results for: \"{}\"", query));
        if let Some(w) = &wing {
            output.push_str(&format!("\n  Wing: {}", w));
        }
        if let Some(r) = &room {
            output.push_str(&format!("\n  Room: {}", r));
        }
        output.push_str(&format!("\n{}\n", "=".repeat(60)));

        for i in 0..docs.len() {
            let doc = &docs[i];
            let meta = &metas[i];
            let dist = dists[i];

            let similarity = 1.0 - dist;
            let wing_name = meta
                .as_ref()
                .and_then(|m: &serde_json::Map<String, serde_json::Value>| m.get("wing"))
                .and_then(|v: &serde_json::Value| v.as_str())
                .unwrap_or("?");
            let room_name = meta
                .as_ref()
                .and_then(|m: &serde_json::Map<String, serde_json::Value>| m.get("room"))
                .and_then(|v: &serde_json::Value| v.as_str())
                .unwrap_or("?");
            let source = meta
                .as_ref()
                .and_then(|m: &serde_json::Map<String, serde_json::Value>| m.get("source_file"))
                .and_then(|v: &serde_json::Value| v.as_str())
                .unwrap_or("");
            let source_name = PathBuf::from(source)
                .file_name()
                .and_then(|s| s.to_str())
                .unwrap_or("?")
                .to_string();

            output.push_str(&format!("\n  [{}] {} / {}", i + 1, wing_name, room_name));
            output.push_str(&format!("\n      Source: {}", source_name));
            output.push_str(&format!("\n      Match:  {:.3}\n", similarity));

            let trimmed = doc.trim();
            for line in trimmed.split('\n') {
                output.push_str(&format!("\n      {}", line));
            }
            output.push('\n');
            output.push_str(&format!("\n  {}", "─".repeat(56)));
        }

        output
    }

    pub async fn search(
        &self,
        query: &str,
        wing: Option<String>,
        room: Option<String>,
        n_results: usize,
    ) -> Result<String> {
        let results = self
            .semantic_search(
                query,
                n_results,
                SemanticSearchOptions {
                    wing: wing.clone(),
                    room: room.clone(),
                    ..SemanticSearchOptions::default()
                },
            )
            .await?;

        if results.is_empty() {
            return Ok(format!("\n  No results found for: \"{}\"", query));
        }

        let docs: Vec<String> = results
            .iter()
            .map(|result| result.record.text_content.clone())
            .collect();
        let metas: Vec<Option<serde_json::Map<String, serde_json::Value>>> = results
            .iter()
            .map(|result| {
                let record = &result.record;
                let mut m = serde_json::Map::new();
                m.insert(
                    "wing".to_string(),
                    serde_json::Value::String(record.wing.clone()),
                );
                m.insert(
                    "room".to_string(),
                    serde_json::Value::String(record.room.clone()),
                );
                m.insert(
                    "valid_from".to_string(),
                    serde_json::Value::Number(record.valid_from.into()),
                );
                if let Some(source_file) = &record.source_file {
                    m.insert(
                        "source_file".to_string(),
                        serde_json::Value::String(source_file.clone()),
                    );
                }
                Some(m)
            })
            .collect();
        let dists: Vec<f32> = results
            .iter()
            .map(|result| 1.0 - result.similarity)
            .collect();

        let output =
            Self::format_search_results(query, wing.as_ref(), room.as_ref(), &docs, &metas, &dists);
        Ok(output)
    }

    pub fn format_semantic_json_results(
        query: &str,
        options: &SemanticSearchOptions,
        results: &[SearchResult],
    ) -> serde_json::Value {
        let hits: Vec<serde_json::Value> = results
            .iter()
            .map(|result| {
                let record = &result.record;
                serde_json::json!({
                    "rank": result.rank,
                    "similarity": result.similarity,
                    "record": {
                        "id": record.id,
                        "text": &record.text_content,
                        "wing": &record.wing,
                        "room": &record.room,
                        "source_file": &record.source_file,
                        "valid_from": record.valid_from,
                        "valid_to": record.valid_to,
                        "heat_i32": record.heat_i32,
                        "last_accessed": record.last_accessed.to_rfc3339(),
                        "access_count": record.access_count,
                        "emotion_valence": record.emotion_valence,
                        "emotion_arousal": record.emotion_arousal,
                        "is_archive_candidate": record.is_archive_candidate,
                        "discard_candidate": record.discard_candidate,
                        "merged_into_id": record.merged_into_id
                    }
                })
            })
            .collect();

        serde_json::json!({
            "query": query,
            "filters": {
                "wing": &options.wing,
                "room": &options.room,
                "include_discarded": options.include_discarded,
                "sort_by_heat": options.sort_by_heat,
            },
            "results": hits
        })
    }

    pub fn format_json_results(
        query: &str,
        wing: Option<&String>,
        room: Option<&String>,
        docs: &[String],
        metas: &[Option<serde_json::Map<String, serde_json::Value>>],
        dists: &[f32],
    ) -> serde_json::Value {
        let mut hits = Vec::new();
        if !docs.is_empty() && !docs[0].is_empty() {
            for i in 0..docs.len() {
                hits.push(serde_json::json!({
                    "text": docs[i],
                    "wing": metas[i].as_ref().and_then(|m: &serde_json::Map<String, serde_json::Value>| m.get("wing")).and_then(|v: &serde_json::Value| v.as_str()).unwrap_or("unknown"),
                    "room": metas[i].as_ref().and_then(|m: &serde_json::Map<String, serde_json::Value>| m.get("room")).and_then(|v: &serde_json::Value| v.as_str()).unwrap_or("unknown"),
                    "source_file": PathBuf::from(metas[i].as_ref().and_then(|m: &serde_json::Map<String, serde_json::Value>| m.get("source_file")).and_then(|v: &serde_json::Value| v.as_str()).unwrap_or("?")).file_name().and_then(|s| s.to_str()).unwrap_or("?"),
                    "similarity": 1.0 - dists[i]
                }));
            }
        }

        serde_json::json!({
            "query": query,
            "filters": {
                "wing": wing,
                "room": room
            },
            "results": hits
        })
    }

    pub async fn search_memories(
        &self,
        query: &str,
        wing: Option<String>,
        room: Option<String>,
        n_results: usize,
    ) -> Result<serde_json::Value> {
        let options = SemanticSearchOptions {
            wing,
            room,
            ..SemanticSearchOptions::default()
        };
        let results = self
            .semantic_search(query, n_results, options.clone())
            .await?;
        Ok(Self::format_semantic_json_results(
            query, &options, &results,
        ))
    }
}

fn truncate_embedding_input(query: &str) -> String {
    const MAX_TOKENS: usize = 512;
    let mut parts = query.split_whitespace();
    let mut truncated = Vec::with_capacity(MAX_TOKENS);

    for _ in 0..MAX_TOKENS {
        let Some(part) = parts.next() else {
            break;
        };
        truncated.push(part);
    }

    if truncated.is_empty() {
        query.to_string()
    } else {
        truncated.join(" ")
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_searcher_new() {
        let config = MempalaceConfig::default();
        let searcher = Searcher::new(config);
        assert_eq!(
            searcher.config.collection_name,
            MempalaceConfig::default().collection_name
        );
    }

    #[test]
    fn test_format_search_results_empty() {
        let res = Searcher::format_search_results("hello", None, None, &[], &[], &[]);
        assert!(res.contains("No results found for: \"hello\""));

        let res2 =
            Searcher::format_search_results("world", None, None, &[String::new()], &[None], &[0.0]);
        assert!(res2.contains("No results found for: \"world\""));
    }

    #[test]
    fn test_format_search_results_with_data() {
        let docs = vec!["this is a test document".to_string()];

        let mut meta1 = serde_json::Map::new();
        meta1.insert(
            "wing".to_string(),
            serde_json::Value::String("engineering".to_string()),
        );
        meta1.insert(
            "room".to_string(),
            serde_json::Value::String("rust".to_string()),
        );
        meta1.insert(
            "source_file".to_string(),
            serde_json::Value::String("/path/to/some/file.txt".to_string()),
        );
        let metas = vec![Some(meta1)];
        let dists = vec![0.1_f32];

        let wing = Some("engineering".to_string());
        let room = Some("rust".to_string());

        let res = Searcher::format_search_results(
            "test",
            wing.as_ref(),
            room.as_ref(),
            &docs,
            &metas,
            &dists,
        );

        assert!(res.contains("Results for: \"test\""));
        assert!(res.contains("Wing: engineering"));
        assert!(res.contains("Room: rust"));
        assert!(res.contains("[1] engineering / rust"));
        assert!(res.contains("Source: file.txt"));
        assert!(res.contains("Match:  0.900"));
        assert!(res.contains("this is a test document"));
    }

    #[test]
    fn test_format_search_results_missing_metadata() {
        let docs = vec!["missing meta".to_string()];
        let metas = vec![None];
        let dists = vec![0.5_f32];

        let res = Searcher::format_search_results("meta", None, None, &docs, &metas, &dists);
        assert!(res.contains("[1] ? / ?"));
        assert!(res.contains("Source: ?"));
    }

    #[test]
    fn test_format_json_results_empty() {
        let res = Searcher::format_json_results("hello", None, None, &[], &[], &[]);
        assert_eq!(res["query"], "hello");
        assert!(res["results"].as_array().unwrap().is_empty());
    }

    #[test]
    fn test_format_json_results_with_data() {
        let docs = vec!["this is a json doc".to_string()];

        let mut meta1 = serde_json::Map::new();
        meta1.insert(
            "wing".to_string(),
            serde_json::Value::String("ops".to_string()),
        );
        meta1.insert(
            "room".to_string(),
            serde_json::Value::String("general".to_string()),
        );
        meta1.insert(
            "source_file".to_string(),
            serde_json::Value::String("/another/path/docs.md".to_string()),
        );
        let metas = vec![Some(meta1)];
        let dists = vec![0.2_f32];

        let wing = Some("ops".to_string());

        let res = Searcher::format_json_results("json", wing.as_ref(), None, &docs, &metas, &dists);

        assert_eq!(res["query"], "json");
        assert_eq!(res["filters"]["wing"], "ops");
        assert_eq!(res["filters"]["room"], serde_json::Value::Null);

        let results = res["results"].as_array().unwrap();
        assert_eq!(results.len(), 1);
        let first = &results[0];
        assert_eq!(first["text"], "this is a json doc");
        assert_eq!(first["wing"], "ops");
        assert_eq!(first["room"], "general");
        assert_eq!(first["source_file"], "docs.md");

        // Due to f32 float precision 1.0 - 0.2 might be 0.800000011920929
        let sim = first["similarity"].as_f64().unwrap();
        assert!((sim - 0.8).abs() < 0.0001);
    }

    #[test]
    fn test_format_json_results_missing_metadata() {
        let docs = vec!["no meta doc".to_string()];
        let metas = vec![None];
        let dists = vec![0.0_f32];

        let res = Searcher::format_json_results("missing", None, None, &docs, &metas, &dists);
        let results = res["results"].as_array().unwrap();
        assert_eq!(results.len(), 1);
        let first = &results[0];
        assert_eq!(first["wing"], "unknown");
        assert_eq!(first["room"], "unknown");
        assert_eq!(first["source_file"], "?");
    }

    #[test]
    fn test_build_where_clause_empty() {
        let res = Searcher::build_where_clause(None, None);
        assert_eq!(res, None);
    }

    #[test]
    fn test_build_where_clause_wing_only() {
        let wing = "engineering".to_string();
        let res = Searcher::build_where_clause(Some(&wing), None).unwrap();
        assert_eq!(res["wing"], "engineering");
    }

    #[test]
    fn test_build_where_clause_room_only() {
        let room = "rust".to_string();
        let res = Searcher::build_where_clause(None, Some(&room)).unwrap();
        assert_eq!(res["room"], "rust");
    }

    #[test]
    fn test_build_where_clause_wing_and_room() {
        let wing = "engineering".to_string();
        let room = "rust".to_string();
        let res = Searcher::build_where_clause(Some(&wing), Some(&room)).unwrap();

        let and_arr = res["$and"].as_array().unwrap();
        assert_eq!(and_arr.len(), 2);

        let mut has_wing = false;
        let mut has_room = false;

        for item in and_arr {
            if item.get("wing").is_some() {
                assert_eq!(item["wing"], "engineering");
                has_wing = true;
            }
            if item.get("room").is_some() {
                assert_eq!(item["room"], "rust");
                has_room = true;
            }
        }
        assert!(has_wing);
        assert!(has_room);
    }

    #[tokio::test]
    async fn test_search_graceful_when_unavailable() {
        let config = MempalaceConfig::default();
        let searcher = Searcher::new(config);

        let res = searcher.search("query", None, None, 5).await;
        assert!(res.is_ok());

        let res2 = searcher.search_memories("query", None, None, 5).await;
        assert!(res2.is_ok());
    }

    #[tokio::test]
    async fn test_recall_by_time_range_graceful_when_unavailable() {
        let config = MempalaceConfig::default();
        let searcher = Searcher::new(config);

        let res = searcher
            .recall_by_time_range(RecallQuery::by_time_range(100, 200))
            .await;
        assert!(res.is_ok());
    }

    #[test]
    fn test_format_search_results_multiline_doc() {
        let docs = vec!["line 1\nline 2\nline 3".to_string()];
        let metas = vec![None];
        let dists = vec![0.1_f32];

        let res = Searcher::format_search_results("multi", None, None, &docs, &metas, &dists);
        assert!(res.contains("line 1"));
        assert!(res.contains("line 2"));
        assert!(res.contains("line 3"));
    }

    #[test]
    fn test_format_search_results_empty_pure() {
        assert!(
            Searcher::format_search_results("none", None, None, &[], &[], &[])
                .contains("No results found")
        );
    }

    #[test]
    fn test_format_json_results_empty_pure() {
        let res = Searcher::format_json_results("none", None, None, &[], &[], &[]);
        assert_eq!(res["results"].as_array().unwrap().len(), 0);
    }

    #[test]
    fn test_truncate_embedding_input_caps_word_count() {
        let query = (0..600)
            .map(|index| format!("token{index}"))
            .collect::<Vec<_>>()
            .join(" ");

        let truncated = truncate_embedding_input(&query);
        assert_eq!(truncated.split_whitespace().count(), 512);
        assert!(truncated.starts_with("token0 token1"));
        assert!(!truncated.contains("token599"));
    }
}
