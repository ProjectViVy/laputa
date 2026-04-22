//! Rhythm capsule generation and retrieval.
mod capsule;
mod scheduler;
mod weekly;

pub use capsule::{CapsuleHotEvent, CapsuleRelationChange, RhythmCapsule, SummaryCapsule};
pub use scheduler::{RhythmConfig, RhythmScheduler, SchedulerExecutionLog, WeeklyTaskRunner};
pub use weekly::WeeklyCapsuleGenerator;

use std::path::Path;

pub fn load_latest_capsule(config_dir: &Path) -> Option<RhythmCapsule> {
    let config = crate::config::MempalaceConfig::new(Some(config_dir.to_path_buf()));
    if let Some(capsule) = WeeklyCapsuleGenerator::load_latest_from_db(&config) {
        return Some(capsule);
    }

    let candidates = [
        config_dir.join("rhythm").join("latest-weekly-capsule.md"),
        config_dir.join("rhythm").join("weekly-capsule.md"),
        config_dir.join("rhythm").join("capsule.md"),
    ];

    for path in candidates {
        if !path.exists() {
            continue;
        }

        let Ok(content) = std::fs::read_to_string(&path) else {
            continue;
        };

        let trimmed = content.trim();
        if trimmed.is_empty() {
            continue;
        }

        return Some(RhythmCapsule {
            source: path.to_string_lossy().to_string(),
            content: trimmed.to_string(),
        });
    }

    None
}
