use crate::api::LaputaError;
use crate::archiver::ArchiveConfig;
use crate::config::{ArchiveState, MempalaceConfig};
use crate::vector_storage::VectorStorage;
use chrono::Utc;
use rusqlite::{params, Connection};
use std::fs;
use std::path::{Path, PathBuf};

const DEFAULT_ARCHIVE_EXPORT_DIR: &str = "archives";

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct ArchiveExportResult {
    pub export_path: PathBuf,
    pub exported_count: usize,
    pub exported_at: i64,
}

/// Exports archive candidates into a standalone SQLite file.
pub struct ArchiveExporter<'a> {
    storage: &'a VectorStorage,
    config: MempalaceConfig,
    archive_config: ArchiveConfig,
}

impl<'a> ArchiveExporter<'a> {
    /// Creates a new archive exporter using the runtime config directory for state persistence.
    pub fn new(
        storage: &'a VectorStorage,
        config: MempalaceConfig,
        archive_config: ArchiveConfig,
    ) -> Result<Self, LaputaError> {
        archive_config.validate()?;
        Ok(Self {
            storage,
            config,
            archive_config,
        })
    }

    /// Exports `is_archive_candidate = true` records into a dedicated SQLite file.
    pub fn export_candidates(
        &self,
        output_path: Option<PathBuf>,
    ) -> Result<ArchiveExportResult, LaputaError> {
        let candidates = self
            .storage
            .list_archive_candidates(usize::MAX)
            .map_err(|error| LaputaError::ArchiveError(error.to_string()))?;

        if candidates.is_empty() {
            return Err(LaputaError::ArchiveError(
                "no archive candidates available for export".to_string(),
            ));
        }

        let exported_at = Utc::now().timestamp();
        let export_path = output_path.unwrap_or_else(|| self.default_export_path(exported_at));
        prepare_export_path(&export_path)?;

        let source_db_path = self
            .storage
            .source_db_path()
            .map_err(|error| LaputaError::ArchiveError(error.to_string()))?;
        let exported_count = self
            .storage
            .export_archive_candidates_to(&export_path)
            .map_err(|error| LaputaError::ArchiveError(error.to_string()))?;

        self.write_export_metadata(&export_path, &source_db_path, exported_count, exported_at)?;

        let mut config = self.config.clone();
        config
            .save_archive_state(ArchiveState {
                last_export_path: export_path.clone(),
                last_exported_at: exported_at,
                last_exported_count: exported_count,
                last_source_db_path: source_db_path,
            })
            .map_err(|error| LaputaError::ConfigError(error.to_string()))?;

        Ok(ArchiveExportResult {
            export_path,
            exported_count,
            exported_at,
        })
    }

    fn default_export_path(&self, exported_at: i64) -> PathBuf {
        self.config
            .config_dir
            .join(DEFAULT_ARCHIVE_EXPORT_DIR)
            .join(format!("archive-export-{exported_at}.sqlite"))
    }

    fn write_export_metadata(
        &self,
        export_path: &Path,
        source_db_path: &Path,
        exported_count: usize,
        exported_at: i64,
    ) -> Result<(), LaputaError> {
        let conn = Connection::open(export_path)
            .map_err(|error| LaputaError::ArchiveError(error.to_string()))?;

        conn.execute_batch(
            "CREATE TABLE IF NOT EXISTS archive_export_metadata (
                key TEXT PRIMARY KEY,
                value TEXT NOT NULL
             );",
        )
        .map_err(|error| LaputaError::ArchiveError(error.to_string()))?;

        let metadata = [
            ("exported_at", exported_at.to_string()),
            (
                "source_db_path",
                source_db_path.to_string_lossy().into_owned(),
            ),
            (
                "archive_threshold",
                self.archive_config.archive_threshold.to_string(),
            ),
            ("exported_count", exported_count.to_string()),
        ];

        for (key, value) in metadata {
            conn.execute(
                "INSERT OR REPLACE INTO archive_export_metadata (key, value) VALUES (?1, ?2)",
                params![key, value],
            )
            .map_err(|error| LaputaError::ArchiveError(error.to_string()))?;
        }

        Ok(())
    }
}

fn prepare_export_path(export_path: &Path) -> Result<(), LaputaError> {
    let Some(parent) = export_path.parent() else {
        return Err(LaputaError::InvalidPath(format!(
            "export path {} has no parent directory",
            export_path.display()
        )));
    };

    fs::create_dir_all(parent).map_err(|error| LaputaError::ArchiveError(error.to_string()))?;
    if export_path.exists() {
        fs::remove_file(export_path)
            .map_err(|error| LaputaError::ArchiveError(error.to_string()))?;
    }
    Ok(())
}
