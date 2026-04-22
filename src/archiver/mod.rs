//! Archive candidate marking for Phase 1.
//! Physical export, packing, and deletion remain out of scope.

mod exporter;

use crate::api::LaputaError;
use crate::vector_storage::{MemoryRecord, VectorStorage};
use std::fs;
use std::path::Path;

pub use exporter::{ArchiveExportResult, ArchiveExporter};

pub const DEFAULT_ARCHIVE_THRESHOLD: i32 = 2_000;
pub const DEFAULT_CHECK_INTERVAL_DAYS: u64 = 1;

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct ArchiveConfig {
    pub enabled: bool,
    pub archive_threshold: i32,
    pub check_interval_days: u64,
}

impl Default for ArchiveConfig {
    fn default() -> Self {
        Self {
            enabled: false,
            archive_threshold: DEFAULT_ARCHIVE_THRESHOLD,
            check_interval_days: DEFAULT_CHECK_INTERVAL_DAYS,
        }
    }
}

impl ArchiveConfig {
    pub fn load_from_dir(config_dir: &Path) -> Result<Self, LaputaError> {
        let path = config_dir.join("laputa.toml");
        let content = fs::read_to_string(&path).map_err(|error| {
            LaputaError::ConfigError(format!("failed to read {}: {error}", path.display()))
        })?;

        Self::from_toml_str(&content)
    }

    pub fn from_toml_str(content: &str) -> Result<Self, LaputaError> {
        let mut config = Self::default();
        let mut in_archive_section = false;

        for raw_line in content.lines() {
            let line = raw_line.split('#').next().unwrap_or("").trim();
            if line.is_empty() {
                continue;
            }

            if line.starts_with('[') && line.ends_with(']') {
                in_archive_section = &line[1..line.len() - 1] == "archive";
                continue;
            }

            if !in_archive_section {
                continue;
            }

            let Some((key, value)) = line.split_once('=') else {
                continue;
            };
            let key = key.trim();
            let value = value.trim();

            match key {
                "enabled" => {
                    config.enabled = value.parse::<bool>().map_err(|error| {
                        LaputaError::ConfigError(format!("invalid enabled value {value}: {error}"))
                    })?;
                }
                "archive_threshold" => {
                    config.archive_threshold = value.parse::<i32>().map_err(|error| {
                        LaputaError::ConfigError(format!(
                            "invalid archive_threshold value {value}: {error}"
                        ))
                    })?;
                }
                "check_interval_days" => {
                    config.check_interval_days = value.parse::<u64>().map_err(|error| {
                        LaputaError::ConfigError(format!(
                            "invalid check_interval_days value {value}: {error}"
                        ))
                    })?;
                }
                _ => {}
            }
        }

        config.validate()?;
        Ok(config)
    }

    pub fn validate(&self) -> Result<(), LaputaError> {
        if !(0..=10_000).contains(&self.archive_threshold) {
            return Err(LaputaError::ArchiveError(format!(
                "archive_threshold must be within 0..=10000, got {}",
                self.archive_threshold
            )));
        }

        if self.check_interval_days == 0 {
            return Err(LaputaError::ArchiveError(
                "check_interval_days must be greater than 0".to_string(),
            ));
        }

        Ok(())
    }
}

pub struct ArchiveMarker<'a> {
    storage: &'a VectorStorage,
    config: ArchiveConfig,
}

impl<'a> ArchiveMarker<'a> {
    pub fn new(storage: &'a VectorStorage, config: ArchiveConfig) -> Result<Self, LaputaError> {
        config.validate()?;
        Ok(Self { storage, config })
    }

    pub fn load_from_dir(
        storage: &'a VectorStorage,
        config_dir: &Path,
    ) -> Result<Self, LaputaError> {
        Self::new(storage, ArchiveConfig::load_from_dir(config_dir)?)
    }

    pub fn config(&self) -> &ArchiveConfig {
        &self.config
    }

    pub fn run_daily_check(&self) -> Result<usize, LaputaError> {
        if !self.config.enabled {
            return Ok(0);
        }

        self.storage
            .mark_low_heat_memories_as_archive_candidates(self.config.archive_threshold)
            .map(|marked| marked.len())
            .map_err(|error| LaputaError::ArchiveError(error.to_string()))
    }

    pub fn list_candidates(&self, limit: usize) -> Result<Vec<MemoryRecord>, LaputaError> {
        self.storage
            .list_archive_candidates(limit)
            .map_err(|error| LaputaError::ArchiveError(error.to_string()))
    }
}
