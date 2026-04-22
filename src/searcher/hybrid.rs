use super::{RecallQuery, SearchResult};
use crate::storage::memory::LaputaMemoryRecord;
use std::collections::HashMap;
use std::path::Path;

const DEFAULT_TOP_K: usize = 100;
const MAX_TOP_K: usize = 1_000;

#[derive(Debug, Clone, PartialEq)]
pub struct HybridRankingConfig {
    pub time_weight: f64,
    pub semantic_weight: f64,
    pub heat_weight: f64,
}

impl Default for HybridRankingConfig {
    fn default() -> Self {
        Self {
            time_weight: 0.3,
            semantic_weight: 0.4,
            heat_weight: 0.3,
        }
    }
}

#[derive(Debug, Clone, PartialEq)]
pub struct HybridQuery {
    pub recall_query: RecallQuery,
    pub semantic_query: String,
    pub top_k: usize,
    pub semantic_limit: usize,
    pub ranking_config: HybridRankingConfig,
}

impl HybridQuery {
    pub fn new(semantic_query: impl Into<String>, recall_query: RecallQuery) -> Self {
        Self {
            recall_query,
            semantic_query: semantic_query.into(),
            top_k: DEFAULT_TOP_K,
            semantic_limit: (DEFAULT_TOP_K * 2).min(MAX_TOP_K),
            ranking_config: HybridRankingConfig::default(),
        }
    }

    pub fn with_top_k(mut self, top_k: usize) -> Self {
        let top_k = top_k.clamp(1, MAX_TOP_K);
        self.top_k = top_k;
        self.semantic_limit = (top_k * 2).min(MAX_TOP_K);
        self
    }

    pub fn with_semantic_limit(mut self, semantic_limit: usize) -> Self {
        self.semantic_limit = semantic_limit.clamp(1, MAX_TOP_K);
        self
    }

    pub fn with_ranking_config(mut self, ranking_config: HybridRankingConfig) -> Self {
        self.ranking_config = ranking_config;
        self
    }
}

#[derive(Debug, Clone, PartialEq)]
pub struct HybridSearchResult {
    pub record: LaputaMemoryRecord,
    pub composite_score: f64,
    pub time_score: f64,
    pub semantic_score: f64,
    pub heat_score: f64,
}

impl HybridSearchResult {
    pub fn new(record: LaputaMemoryRecord) -> Self {
        Self {
            heat_score: normalize_heat_score(record.heat_i32),
            record,
            composite_score: 0.0,
            time_score: 0.0,
            semantic_score: 0.0,
        }
    }

    pub fn recompute_composite_score(&mut self, config: &HybridRankingConfig) {
        self.composite_score = compute_composite_score(
            self.time_score,
            self.semantic_score,
            self.heat_score,
            config,
        );
    }
}

pub fn normalize_time_score(valid_from: i64, start: i64, end: i64) -> f64 {
    if start > end {
        return 0.0;
    }
    if start == end {
        return if valid_from == start { 1.0 } else { 0.0 };
    }
    if valid_from < start || valid_from > end {
        return 0.0;
    }

    let center = start + ((end - start) / 2);
    let half_range = ((end - start) / 2).max(1);
    let distance = (valid_from - center).abs();
    1.0 - (distance as f64 / half_range as f64).min(1.0)
}

pub fn normalize_heat_score(heat_i32: i32) -> f64 {
    (heat_i32 as f64 / 10_000.0).clamp(0.0, 1.0)
}

pub fn compute_composite_score(
    time_score: f64,
    semantic_score: f64,
    heat_score: f64,
    config: &HybridRankingConfig,
) -> f64 {
    config.time_weight * time_score
        + config.semantic_weight * semantic_score
        + config.heat_weight * heat_score
}

pub fn merge_hybrid_results(
    query: &HybridQuery,
    time_results: Vec<LaputaMemoryRecord>,
    semantic_results: Vec<SearchResult>,
) -> Vec<HybridSearchResult> {
    let mut deduped: HashMap<i64, HybridSearchResult> = HashMap::new();

    for record in time_results {
        let mut result = HybridSearchResult::new(record);
        result.time_score = 1.0;
        result.recompute_composite_score(&query.ranking_config);
        deduped.insert(result.record.id, result);
    }

    for semantic in semantic_results {
        let record_id = semantic.record.id;
        if let Some(existing) = deduped.get_mut(&record_id) {
            existing.semantic_score = existing.semantic_score.max(semantic.similarity as f64);
            existing.recompute_composite_score(&query.ranking_config);
            continue;
        }

        let mut result = HybridSearchResult::new(semantic.record);
        result.semantic_score = semantic.similarity as f64;
        result.time_score = normalize_time_score(
            result.record.valid_from,
            query.recall_query.start,
            query.recall_query.end,
        );
        result.recompute_composite_score(&query.ranking_config);
        deduped.insert(record_id, result);
    }

    let mut merged: Vec<HybridSearchResult> = deduped.into_values().collect();
    merged.sort_by(|left, right| {
        right
            .composite_score
            .partial_cmp(&left.composite_score)
            .unwrap_or(std::cmp::Ordering::Equal)
            .then_with(|| right.record.heat_i32.cmp(&left.record.heat_i32))
            .then_with(|| right.record.valid_from.cmp(&left.record.valid_from))
            .then_with(|| right.semantic_score.total_cmp(&left.semantic_score))
    });
    merged.truncate(query.top_k);
    merged
}

pub fn load_hybrid_ranking_config(config_dir: &Path) -> Option<HybridRankingConfig> {
    let path = config_dir.join("config.toml");
    let content = std::fs::read_to_string(path).ok()?;

    let mut in_hybrid_section = false;
    let mut config = HybridRankingConfig::default();
    let mut touched = false;

    for raw_line in content.lines() {
        let line = raw_line.trim();
        if line.is_empty() || line.starts_with('#') {
            continue;
        }

        if line.starts_with('[') && line.ends_with(']') {
            in_hybrid_section = line == "[search.hybrid]";
            continue;
        }

        if !in_hybrid_section {
            continue;
        }

        let Some((key, value)) = line.split_once('=') else {
            continue;
        };
        let parsed = value.trim().trim_matches('"').parse::<f64>().ok();
        match (key.trim(), parsed) {
            ("time_weight", Some(value)) => {
                config.time_weight = value;
                touched = true;
            }
            ("semantic_weight", Some(value)) => {
                config.semantic_weight = value;
                touched = true;
            }
            ("heat_weight", Some(value)) => {
                config.heat_weight = value;
                touched = true;
            }
            _ => {}
        }
    }

    if touched {
        Some(config)
    } else {
        None
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::storage::memory::LaputaMemoryRecord;

    fn record(id: i64, valid_from: i64, heat_i32: i32) -> LaputaMemoryRecord {
        let mut record = LaputaMemoryRecord::new(
            id,
            format!("record-{id}"),
            "self".to_string(),
            "journal".to_string(),
            None,
            valid_from,
            None,
            0.0,
            5.0,
        );
        record.heat_i32 = heat_i32;
        record
    }

    #[test]
    fn test_merge_hybrid_results_respects_top_k_and_dedup() {
        let query = HybridQuery::new("query", RecallQuery::by_time_range(100, 200)).with_top_k(2);
        let time_results = vec![record(1, 150, 8_000), record(2, 170, 7_000)];
        let semantic_results = vec![
            SearchResult {
                record: record(1, 150, 8_000),
                similarity: 0.9,
                rank: 1,
            },
            SearchResult {
                record: record(3, 90, 9_000),
                similarity: 0.95,
                rank: 2,
            },
        ];

        let results = merge_hybrid_results(&query, time_results, semantic_results);
        assert_eq!(results.len(), 2);
        assert_eq!(results.iter().filter(|item| item.record.id == 1).count(), 1);
    }
}
