use serde::{Deserialize, Serialize};

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
pub struct RhythmCapsule {
    pub source: String,
    pub content: String,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
pub struct CapsuleHotEvent {
    pub memory_id: i64,
    pub wing: String,
    pub room: String,
    pub heat_i32: i32,
    pub summary: String,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
pub struct CapsuleRelationChange {
    pub subject: String,
    pub predicate: String,
    pub object: String,
    pub previous_resonance: Option<i32>,
    pub current_resonance: i32,
    pub delta: i32,
    pub valid_from: Option<String>,
    pub source_file: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub struct SummaryCapsule {
    pub week_id: String,
    pub week_start: String,
    pub week_end: String,
    pub keywords: Vec<String>,
    pub hot_events: Vec<CapsuleHotEvent>,
    pub relation_changes: Vec<CapsuleRelationChange>,
    pub source_record_count: usize,
    pub original_tokens: usize,
    pub compressed_tokens: usize,
    pub compression_ratio: f64,
    pub created_at: String,
    pub incomplete: bool,
    pub compressed_content: String,
}

impl SummaryCapsule {
    pub fn render_markdown(&self) -> String {
        let keywords = if self.keywords.is_empty() {
            "none".to_string()
        } else {
            self.keywords.join(", ")
        };

        let hot_events = if self.hot_events.is_empty() {
            "- none".to_string()
        } else {
            self.hot_events
                .iter()
                .map(|event| {
                    format!(
                        "- [{} / {} | heat:{}] {}",
                        event.wing, event.room, event.heat_i32, event.summary
                    )
                })
                .collect::<Vec<_>>()
                .join("\n")
        };

        let relation_changes = if self.relation_changes.is_empty() {
            "- none".to_string()
        } else {
            self.relation_changes
                .iter()
                .map(|relation| {
                    let previous = relation
                        .previous_resonance
                        .map(|value| value.to_string())
                        .unwrap_or_else(|| "new".to_string());
                    format!(
                        "- {} {} {} ({} -> {}, delta={})",
                        relation.subject,
                        relation.predicate,
                        relation.object,
                        previous,
                        relation.current_resonance,
                        relation.delta
                    )
                })
                .collect::<Vec<_>>()
                .join("\n")
        };

        format!(
            "# Weekly Capsule {week_id}\n\
Week Range: {week_start} -> {week_end}\n\
Incomplete: {incomplete}\n\
Source Records: {source_record_count}\n\
Keywords: {keywords}\n\
\n\
## Hot Events\n\
{hot_events}\n\
\n\
## Relation Changes\n\
{relation_changes}\n\
\n\
## AAAK\n\
{compressed_content}\n\
\n\
## Token Stats\n\
Original Tokens: {original_tokens}\n\
Summary Tokens: {compressed_tokens}\n\
Compression Ratio: {compression_ratio:.1}x\n",
            week_id = self.week_id,
            week_start = self.week_start,
            week_end = self.week_end,
            incomplete = self.incomplete,
            source_record_count = self.source_record_count,
            keywords = keywords,
            hot_events = hot_events,
            relation_changes = relation_changes,
            compressed_content = self.compressed_content,
            original_tokens = self.original_tokens,
            compressed_tokens = self.compressed_tokens,
            compression_ratio = self.compression_ratio,
        )
    }
}
