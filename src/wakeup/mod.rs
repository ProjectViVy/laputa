//! WakePack 生成模块，负责构建受 token 预算控制的唤醒上下文包。

use crate::config::MempalaceConfig;
use crate::knowledge_graph::{KnowledgeGraph, ResonantRelation};
use crate::rhythm::{load_latest_capsule, RhythmCapsule};
use crate::searcher::RecallQuery;
use crate::vector_storage::{MemoryRecord, VectorStorage};
use anyhow::Result;
use chrono::Utc;
use serde::{Deserialize, Serialize};
use std::path::Path;

const MAX_WAKEPACK_TOKENS: usize = 1200;
const SEVEN_DAYS_SECONDS: i64 = 7 * 24 * 60 * 60;
const MAX_RECENT_MEMORIES: usize = 8;
const MAX_RELATIONS: usize = 6;
const DEFAULT_TEXT_CHARS: usize = 160;
const MIN_CAPSULE_CHARS: usize = 80;

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
pub struct IdentityField {
    pub key: String,
    pub value: String,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq, Default)]
pub struct IdentityProfile {
    pub user_name: Option<String>,
    pub user_type: Option<String>,
    pub created_at: Option<String>,
    pub fields: Vec<IdentityField>,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
pub struct MemorySummary {
    pub id: i64,
    pub wing: String,
    pub room: String,
    pub valid_from: i64,
    pub heat_i32: i32,
    pub summary: String,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub struct WakePack {
    pub identity: IdentityProfile,
    pub recent_state: Vec<MemorySummary>,
    pub weekly_capsule: Option<RhythmCapsule>,
    pub key_relations: Vec<ResonantRelation>,
    pub token_count: usize,
}

impl WakePack {
    pub fn to_json(&self) -> Result<String> {
        Ok(serde_json::to_string(self)?)
    }
}

pub struct WakePackGenerator {
    config: MempalaceConfig,
}

impl WakePackGenerator {
    pub fn new(config: MempalaceConfig) -> Self {
        Self { config }
    }

    pub fn generate(&self, wing: Option<String>) -> Result<WakePack> {
        let identity = self.load_identity();
        let recent_state = self.load_recent_state(wing.as_deref())?;
        let weekly_capsule = self.load_weekly_capsule();
        let key_relations = self.load_key_relations()?;

        let mut pack = WakePack {
            identity,
            recent_state,
            weekly_capsule,
            key_relations,
            token_count: 0,
        };

        self.fit_to_budget(&mut pack)?;
        Ok(pack)
    }

    pub fn generate_json(&self, wing: Option<String>) -> Result<String> {
        self.generate(wing)?.to_json()
    }

    fn load_identity(&self) -> IdentityProfile {
        let identity_path = self.config.config_dir.join("identity.md");
        parse_identity_profile(&identity_path)
    }

    fn load_recent_state(&self, wing: Option<&str>) -> Result<Vec<MemorySummary>> {
        let db_path = self.config.config_dir.join("vectors.db");
        let index_path = self.config.config_dir.join("vectors.usearch");
        if !db_path.exists() {
            return Ok(vec![]);
        }

        let store = match VectorStorage::new(db_path, index_path) {
            Ok(store) => store,
            Err(_) => return Ok(vec![]),
        };

        let now = Utc::now().timestamp();
        let mut query = RecallQuery::by_time_range(now - SEVEN_DAYS_SECONDS, now)
            .with_limit(MAX_RECENT_MEMORIES);
        if let Some(wing) = wing {
            query = query.with_wing(wing.to_string());
        }

        let records = store.recall_by_time_range(&query)?;
        for record in &records {
            let _ = store.touch_memory(record.id);
        }
        Ok(records
            .into_iter()
            .take(MAX_RECENT_MEMORIES)
            .map(memory_to_summary)
            .collect())
    }

    fn load_weekly_capsule(&self) -> Option<RhythmCapsule> {
        load_latest_capsule(&self.config.config_dir).map(|mut capsule| {
            capsule.content = trim_text(&capsule.content, 240);
            capsule
        })
    }

    fn load_key_relations(&self) -> Result<Vec<ResonantRelation>> {
        let knowledge_path = self.config.config_dir.join("knowledge.db");
        if !knowledge_path.exists() {
            return Ok(vec![]);
        }

        let graph = KnowledgeGraph::new(knowledge_path.to_str().unwrap_or("knowledge.db"))?;
        Ok(graph.top_relations(50, MAX_RELATIONS)?)
    }

    fn fit_to_budget(&self, pack: &mut WakePack) -> Result<()> {
        self.recompute_token_count(pack)?;

        while pack.token_count >= MAX_WAKEPACK_TOKENS {
            if pack.recent_state.len() > 3 {
                pack.recent_state.pop();
            } else if pack.key_relations.len() > 2 {
                pack.key_relations.pop();
            } else if let Some(capsule) = &mut pack.weekly_capsule {
                if capsule.content.chars().count() > MIN_CAPSULE_CHARS {
                    let next = (capsule.content.chars().count() - 40).max(MIN_CAPSULE_CHARS);
                    capsule.content = trim_text(&capsule.content, next);
                } else {
                    pack.weekly_capsule = None;
                }
            } else if let Some(memory) = pack
                .recent_state
                .iter_mut()
                .find(|item| item.summary.chars().count() > 80)
            {
                let next = (memory.summary.chars().count() - 30).max(80);
                memory.summary = trim_text(&memory.summary, next);
            } else if pack.identity.fields.len() > 3 {
                pack.identity.fields.pop();
            } else {
                break;
            }

            self.recompute_token_count(pack)?;
        }

        Ok(())
    }

    fn recompute_token_count(&self, pack: &mut WakePack) -> Result<()> {
        pack.token_count = 0;
        let serialized = serde_json::to_string(pack)?;
        pack.token_count = estimate_tokens(&serialized);
        Ok(())
    }
}

fn parse_identity_profile(path: &Path) -> IdentityProfile {
    let Ok(content) = std::fs::read_to_string(path) else {
        return IdentityProfile::default();
    };

    let mut profile = IdentityProfile::default();
    for line in content.lines() {
        let trimmed = line.trim();
        if trimmed.is_empty() || trimmed.starts_with('#') {
            continue;
        }

        let Some((key, value)) = trimmed.split_once(':') else {
            continue;
        };

        let key = key.trim().to_string();
        let value = value.trim().to_string();
        if value.is_empty() {
            continue;
        }

        match key.as_str() {
            "user_name" => profile.user_name = Some(value.clone()),
            "user_type" => profile.user_type = Some(value.clone()),
            "created_at" => profile.created_at = Some(value.clone()),
            _ => {}
        }

        profile.fields.push(IdentityField { key, value });
    }

    profile
}

fn memory_to_summary(record: MemoryRecord) -> MemorySummary {
    MemorySummary {
        id: record.id,
        wing: record.wing,
        room: record.room,
        valid_from: record.valid_from,
        heat_i32: record.heat_i32,
        summary: trim_text(&record.text_content, DEFAULT_TEXT_CHARS),
    }
}

fn trim_text(input: &str, max_chars: usize) -> String {
    let trimmed = input.trim();
    if trimmed.chars().count() <= max_chars {
        return trimmed.to_string();
    }

    let mut end = 0;
    for (count, (index, ch)) in trimmed.char_indices().enumerate() {
        if count == max_chars {
            break;
        }
        end = index + ch.len_utf8();
    }

    format!("{}...", &trimmed[..end])
}

fn estimate_tokens(serialized: &str) -> usize {
    ((serialized.chars().count() as f32) / 1.5).ceil() as usize
}
