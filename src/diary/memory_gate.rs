use anyhow::Result;

use crate::config::MempalaceConfig;
use crate::dialect::Dialect;
use crate::storage::memory::{LaputaMemoryRecord, MAX_HEAT_I32};
use crate::vector_storage::VectorStorage;

const DEFAULT_DUPLICATE_THRESHOLD: f32 = 0.8;
const DUPLICATE_TOP_K: usize = 5;
const MERGE_HEAT_BONUS: i32 = 500;
const LOW_VALUE_MAX_WORDS: usize = 3;
const LOW_VALUE_PHRASES: &[&str] = &[
    "ok",
    "okay",
    "thanks",
    "thank you",
    "got it",
    "noted",
    "test",
    "ping",
    "hello",
    "hi",
];

#[derive(Debug, Clone, PartialEq)]
pub enum MemoryGateAction {
    Store,
    Merge { target_id: i64, similarity: f32 },
    Discard,
}

#[derive(Debug, Clone, PartialEq)]
pub struct MemoryGateDecision {
    pub action: MemoryGateAction,
    pub reason: String,
    pub discard_candidate: bool,
    pub merged_into_id: Option<i64>,
}

pub struct MemoryGate {
    threshold: f32,
}

impl Default for MemoryGate {
    fn default() -> Self {
        Self {
            threshold: DEFAULT_DUPLICATE_THRESHOLD,
        }
    }
}

impl MemoryGate {
    pub fn new(threshold: f32) -> Self {
        Self { threshold }
    }

    pub fn judge(
        &self,
        storage: &VectorStorage,
        candidate_text: &str,
        wing: Option<&str>,
        room: Option<&str>,
    ) -> Result<MemoryGateDecision> {
        if is_low_value(candidate_text) {
            return Ok(MemoryGateDecision {
                action: MemoryGateAction::Discard,
                reason: "low-value diary chatter marked as discard_candidate".to_string(),
                discard_candidate: true,
                merged_into_id: None,
            });
        }

        let results = match (wing, room) {
            (Some(w), Some(r)) => storage.search_room(candidate_text, w, r, DUPLICATE_TOP_K, None),
            _ => storage.search(candidate_text, DUPLICATE_TOP_K),
        };

        let results = match results {
            Ok(results) => results,
            Err(_) => {
                return Ok(MemoryGateDecision {
                    action: MemoryGateAction::Store,
                    reason: "stored without duplicate search because embeddings are unavailable"
                        .to_string(),
                    discard_candidate: false,
                    merged_into_id: None,
                });
            }
        };

        if let Some((target_id, similarity)) = self.pick_duplicate(results) {
            return Ok(MemoryGateDecision {
                action: MemoryGateAction::Merge {
                    target_id,
                    similarity,
                },
                reason: format!(
                    "duplicate match > {:.1}; merged into existing memory {}",
                    self.threshold, target_id
                ),
                discard_candidate: false,
                merged_into_id: Some(target_id),
            });
        }

        Ok(MemoryGateDecision {
            action: MemoryGateAction::Store,
            reason: "stored as a new memory after MemoryGate review".to_string(),
            discard_candidate: false,
            merged_into_id: None,
        })
    }

    pub fn merge_into_existing(
        &self,
        storage: &VectorStorage,
        target: &LaputaMemoryRecord,
        candidate_text: &str,
        reason: &str,
    ) -> Result<()> {
        let dialect = Dialect::default();
        let merged_summary =
            dialect.merge_aaaks(&[target.text_content.clone(), candidate_text.to_string()]);
        let merged_heat = (target.heat_i32 + MERGE_HEAT_BONUS).min(MAX_HEAT_I32);
        storage.update_memory_after_merge(target.id, &merged_summary, merged_heat, reason)
    }

    pub fn pick_duplicate(&self, candidates: Vec<LaputaMemoryRecord>) -> Option<(i64, f32)> {
        candidates
            .into_iter()
            .find(|record| record.score >= self.threshold && !record.discard_candidate)
            .map(|record| (record.id, record.score))
    }
}

pub fn config_gate(_config: &MempalaceConfig) -> MemoryGate {
    MemoryGate::default()
}

fn is_low_value(text: &str) -> bool {
    let trimmed = text.trim().to_lowercase();
    if trimmed.is_empty() {
        return true;
    }

    if LOW_VALUE_PHRASES.contains(&trimmed.as_str()) {
        return true;
    }

    let words = trimmed.split_whitespace().count();
    words <= LOW_VALUE_MAX_WORDS
        && !trimmed.contains("decide")
        && !trimmed.contains("learn")
        && !trimmed.contains("plan")
        && !trimmed.contains("because")
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn low_value_detector_marks_short_chatter() {
        assert!(is_low_value("ok"));
        assert!(is_low_value("thank you"));
        assert!(is_low_value("hello there"));
        assert!(!is_low_value("I decided to adopt a Rust memory schema"));
    }

    #[test]
    fn duplicate_picker_uses_threshold_and_skips_discarded() {
        let gate = MemoryGate::default();
        let mut discarded = LaputaMemoryRecord::new(
            1,
            "discarded".to_string(),
            "self".to_string(),
            "journal".to_string(),
            None,
            0,
            None,
            0.0,
            5.0,
        );
        discarded.score = 0.95;
        discarded.discard_candidate = true;

        let mut winner = LaputaMemoryRecord::new(
            2,
            "winner".to_string(),
            "self".to_string(),
            "journal".to_string(),
            None,
            0,
            None,
            0.0,
            5.0,
        );
        winner.score = 0.81;

        assert_eq!(
            gate.pick_duplicate(vec![discarded, winner]),
            Some((2, 0.81))
        );
    }
}
