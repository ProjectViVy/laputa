use crate::api::LaputaError;
use crate::config::{FullExportState, MempalaceConfig};
use crate::knowledge_graph::KnowledgeGraph;
use crate::rhythm::load_latest_capsule;
use crate::vector_storage::{MemoryRecord, VectorStorage};
use chrono::{Duration, Utc};
use serde_json::json;
use std::fs;
use std::path::{Path, PathBuf};

const DEFAULT_EXPORT_ROOT_DIR: &str = "exports";
const CORE_MEMORY_HEAT_THRESHOLD: i32 = 5_000;

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct FullExportResult {
    pub export_dir: PathBuf,
    pub identity_path: PathBuf,
    pub relation_path: PathBuf,
    pub capsule_count: usize,
    pub exported_memory_count: usize,
    pub exported_at: i64,
}

/// Exports a portable, reviewable subject bundle for migration and restore flows.
pub struct FullExporter {
    config: MempalaceConfig,
}

impl FullExporter {
    /// Creates a full exporter bound to the current runtime config directory.
    pub fn new(config: MempalaceConfig) -> Result<Self, LaputaError> {
        Ok(Self { config })
    }

    /// Exports identity, relation view, latest capsule information, and high-heat core memories.
    pub fn export_full(
        &self,
        output_dir: Option<PathBuf>,
    ) -> Result<FullExportResult, LaputaError> {
        let exported_at = Utc::now().timestamp();
        let export_dir = output_dir.unwrap_or_else(|| self.default_export_dir(exported_at));
        prepare_export_dir(&export_dir)?;

        let identity_content = self.load_identity()?;
        let identity_path = export_dir.join("identity.md");
        fs::write(&identity_path, identity_content).map_err(io_to_storage_error)?;

        let relation_content = self.render_relations()?;
        let relation_path = export_dir.join("relation.md");
        fs::write(&relation_path, relation_content).map_err(io_to_storage_error)?;

        let capsule_count = self.export_capsules(&export_dir)?;
        let core_memories = self.load_core_memories()?;
        self.export_core_memories(&export_dir, &core_memories)?;

        self.write_manifest(&export_dir, exported_at, capsule_count, core_memories.len())?;

        let mut config = self.config.clone();
        config
            .save_full_export_state(FullExportState {
                last_export_path: export_dir.clone(),
                last_exported_at: exported_at,
                last_exported_memory_count: core_memories.len(),
            })
            .map_err(|error| LaputaError::ConfigError(error.to_string()))?;

        Ok(FullExportResult {
            export_dir,
            identity_path,
            relation_path,
            capsule_count,
            exported_memory_count: core_memories.len(),
            exported_at,
        })
    }

    fn default_export_dir(&self, exported_at: i64) -> PathBuf {
        self.config
            .config_dir
            .join(DEFAULT_EXPORT_ROOT_DIR)
            .join(format!("full-export-{exported_at}"))
    }

    fn load_identity(&self) -> Result<String, LaputaError> {
        let identity_path = self.config.config_dir.join("identity.md");
        if !identity_path.exists() {
            return Err(LaputaError::NotFound(format!(
                "identity.md is missing at {}",
                identity_path.display()
            )));
        }

        let content = fs::read_to_string(&identity_path).map_err(io_to_storage_error)?;
        let trimmed = content.trim();
        if trimmed.is_empty() {
            return Err(LaputaError::ValidationError(format!(
                "identity.md is empty at {}",
                identity_path.display()
            )));
        }
        Ok(format!("{trimmed}\n"))
    }

    fn render_relations(&self) -> Result<String, LaputaError> {
        let knowledge_path = self.config.config_dir.join("knowledge.db");
        if !knowledge_path.exists() {
            return Ok("# Relations\n\n## Current\n- No active relations exported.\n".to_string());
        }

        let graph = KnowledgeGraph::new(knowledge_path.to_str().unwrap_or("knowledge.db"))
            .map_err(|error| LaputaError::StorageError(error.to_string()))?;
        let current = graph
            .top_relations(-101, 100)
            .map_err(|error| LaputaError::StorageError(error.to_string()))?;

        let mut lines = vec![
            "# Relations".to_string(),
            "".to_string(),
            "## Current".to_string(),
        ];
        if current.is_empty() {
            lines.push("- No active relations exported.".to_string());
        } else {
            for relation in &current {
                lines.push(format!(
                    "- {} -> {} | {} | resonance: {}",
                    relation.subject, relation.object, relation.predicate, relation.resonance
                ));
            }
        }

        let start = (Utc::now() - Duration::days(30)).date_naive().to_string();
        let end = Utc::now().date_naive().to_string();
        let changes = graph
            .relation_changes_between(&start, &end, 10, 20)
            .map_err(|error| LaputaError::StorageError(error.to_string()))?;

        lines.push("".to_string());
        lines.push("## Recent Changes".to_string());
        if changes.is_empty() {
            lines.push("- No recent relation changes exported.".to_string());
        } else {
            for change in changes {
                lines.push(format!(
                    "- {}: {} -> {} | {} -> {} (delta {})",
                    change
                        .valid_from
                        .unwrap_or_else(|| "unknown-date".to_string()),
                    change.subject,
                    change.object,
                    change
                        .previous_resonance
                        .map(|value| value.to_string())
                        .unwrap_or_else(|| "none".to_string()),
                    change.current_resonance,
                    change.delta
                ));
            }
        }

        Ok(lines.join("\n") + "\n")
    }

    fn export_capsules(&self, export_dir: &Path) -> Result<usize, LaputaError> {
        let capsules_dir = export_dir.join("capsules");
        fs::create_dir_all(&capsules_dir).map_err(io_to_storage_error)?;

        let Some(capsule) = load_latest_capsule(&self.config.config_dir) else {
            return Ok(0);
        };

        let capsule_path = capsules_dir.join("recent-latest.md");
        fs::write(capsule_path, capsule.content).map_err(io_to_storage_error)?;
        Ok(1)
    }

    fn load_core_memories(&self) -> Result<Vec<MemoryRecord>, LaputaError> {
        let db_path = self.config.config_dir.join("vectors.db");
        let index_path = self.config.config_dir.join("vectors.usearch");
        let store = VectorStorage::new(&db_path, &index_path)
            .map_err(|error| LaputaError::StorageError(error.to_string()))?;

        let mut records = store
            .get_memories(None, None, usize::MAX)
            .map_err(|error| LaputaError::StorageError(error.to_string()))?
            .into_iter()
            .filter(|record| record.heat_i32 > CORE_MEMORY_HEAT_THRESHOLD)
            .collect::<Vec<_>>();
        records.sort_by(|left, right| {
            right
                .heat_i32
                .cmp(&left.heat_i32)
                .then_with(|| right.valid_from.cmp(&left.valid_from))
                .then_with(|| left.id.cmp(&right.id))
        });
        Ok(records)
    }

    fn export_core_memories(
        &self,
        export_dir: &Path,
        core_memories: &[MemoryRecord],
    ) -> Result<(), LaputaError> {
        let memories_dir = export_dir.join("memories");
        fs::create_dir_all(&memories_dir).map_err(io_to_storage_error)?;
        let path = memories_dir.join("core-memories.jsonl");

        let mut lines = Vec::with_capacity(core_memories.len());
        for record in core_memories {
            lines.push(
                serde_json::to_string(&json!({
                    "id": record.id,
                    "text_content": record.text_content,
                    "wing": record.wing,
                    "room": record.room,
                    "valid_from": record.valid_from,
                    "valid_to": record.valid_to,
                    "heat_i32": record.heat_i32,
                    "last_accessed": record.last_accessed.timestamp(),
                    "access_count": record.access_count,
                    "emotion_valence": record.emotion_valence,
                    "emotion_arousal": record.emotion_arousal,
                }))
                .map_err(|error| LaputaError::StorageError(error.to_string()))?,
            );
        }

        let body = if lines.is_empty() {
            String::new()
        } else {
            format!("{}\n", lines.join("\n"))
        };
        fs::write(path, body).map_err(io_to_storage_error)
    }

    fn write_manifest(
        &self,
        export_dir: &Path,
        exported_at: i64,
        capsule_count: usize,
        memory_count: usize,
    ) -> Result<(), LaputaError> {
        let manifest = json!({
            "export_type": "full",
            "exported_at": exported_at,
            "export_root": export_dir.to_string_lossy(),
            "identity_path": "identity.md",
            "relation_path": "relation.md",
            "capsule_count": capsule_count,
            "capsule_export_status": if capsule_count > 0 { "exported" } else { "not_available" },
            "memory_count": memory_count,
            "memory_export_threshold": format!("heat_i32 > {}", CORE_MEMORY_HEAT_THRESHOLD),
            "memory_path": "memories/core-memories.jsonl",
        });

        let manifest_path = export_dir.join("manifest.json");
        let body = serde_json::to_string_pretty(&manifest)
            .map_err(|error| LaputaError::StorageError(error.to_string()))?;
        fs::write(manifest_path, body).map_err(io_to_storage_error)
    }
}

fn prepare_export_dir(export_dir: &Path) -> Result<(), LaputaError> {
    if export_dir.exists() {
        fs::remove_dir_all(export_dir).map_err(io_to_storage_error)?;
    }
    fs::create_dir_all(export_dir).map_err(io_to_storage_error)
}

fn io_to_storage_error(error: std::io::Error) -> LaputaError {
    LaputaError::StorageError(error.to_string())
}
