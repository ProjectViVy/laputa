//! 主体身份初始化实现。

use crate::api::LaputaError;
use crate::storage::sqlite::create_schema;
use chrono::{SecondsFormat, Utc};
use rusqlite::Connection;
use std::fs;
use std::path::{Path, PathBuf};

const DB_FILE_NAME: &str = "laputa.db";
const IDENTITY_FILE_NAME: &str = "identity.md";
const USER_TYPE: &str = "个人记忆助手";
const MAX_USER_NAME_LENGTH: usize = 256;
const PATH_TRAVERSAL_PATTERNS: [&str; 4] = ["../", "..\\", "/..", "\\.."];

/// 初始化主体身份与最小数据库结构。
#[derive(Debug, Clone)]
pub struct IdentityInitializer {
    pub db_path: PathBuf,
    pub identity_path: PathBuf,
}

impl IdentityInitializer {
    /// 基于配置目录推断数据库与 identity 文件路径。
    pub fn new(config_dir: &Path) -> Self {
        Self {
            db_path: config_dir.join(DB_FILE_NAME),
            identity_path: config_dir.join(IDENTITY_FILE_NAME),
        }
    }

    /// 检查当前目录是否已经完成身份初始化。
    /// 原子性检查：identity.md 和 laputa.db 都存在才认为已初始化。
    pub fn is_initialized(&self) -> bool {
        self.identity_path.exists() && self.db_path.exists()
    }

    /// 执行初始化，创建数据库、schema 与 identity.md，并返回数据库路径。
    pub fn initialize(&self, user_name: &str) -> Result<String, LaputaError> {
        validate_user_name(user_name)?;

        if self.is_initialized() {
            return Err(LaputaError::AlreadyInitialized(
                self.identity_path.display().to_string(),
            ));
        }

        let parent_dir = self.db_path.parent().ok_or_else(|| {
            LaputaError::InvalidPath(format!("Invalid database path: {}", self.db_path.display()))
        })?;

        fs::create_dir_all(parent_dir)?;

        let connection = Connection::open(&self.db_path)?;
        if let Err(schema_error) = create_schema(&connection) {
            drop(connection);
            return cleanup_db_file_after_failure(&self.db_path, schema_error);
        }
        drop(connection);

        let created_at = Utc::now().to_rfc3339_opts(SecondsFormat::Secs, true);
        let identity_content = format!(
            "## L0 — IDENTITY\n\nuser_name: {user_name}\nuser_type: {USER_TYPE}\ncreated_at: {created_at}\n"
        );

        if let Err(write_error) = fs::write(&self.identity_path, identity_content) {
            let cleanup_result = fs::remove_file(&self.db_path);
            return match cleanup_result {
                Ok(()) => Err(write_error.into()),
                Err(cleanup_error) => Err(LaputaError::StorageError(format!(
                    "failed to write identity file: {write_error}; cleanup failed for {}: {cleanup_error}",
                    self.db_path.display()
                ))),
            };
        }

        Ok(self.db_path.display().to_string())
    }
}

fn cleanup_db_file_after_failure(
    db_path: &Path,
    primary_error: impl Into<LaputaError>,
) -> Result<String, LaputaError> {
    let primary_error = primary_error.into();
    match fs::remove_file(db_path) {
        Ok(()) => Err(primary_error),
        Err(cleanup_error) if cleanup_error.kind() == std::io::ErrorKind::NotFound => {
            Err(primary_error)
        }
        Err(cleanup_error) => Err(LaputaError::StorageError(format!(
            "{primary_error}; cleanup failed for {}: {cleanup_error}",
            db_path.display()
        ))),
    }
}

fn validate_user_name(user_name: &str) -> Result<(), LaputaError> {
    if user_name.trim().is_empty() || user_name.contains('\n') {
        return Err(LaputaError::ValidationError(
            "user_name must be non-empty and contain no newlines".to_string(),
        ));
    }

    if user_name.len() > MAX_USER_NAME_LENGTH {
        return Err(LaputaError::ValidationError(format!(
            "user_name must be at most {MAX_USER_NAME_LENGTH} characters"
        )));
    }

    if PATH_TRAVERSAL_PATTERNS
        .iter()
        .any(|pattern| user_name.contains(pattern))
    {
        return Err(LaputaError::ValidationError(
            "user_name must not contain path traversal fragments".to_string(),
        ));
    }

    if user_name.chars().any(|ch| ch.is_control() && ch != '\n') {
        return Err(LaputaError::ValidationError(
            "user_name must not contain control characters".to_string(),
        ));
    }

    Ok(())
}
