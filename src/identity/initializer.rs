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
    pub fn is_initialized(&self) -> bool {
        self.identity_path.exists()
    }

    /// 执行初始化，创建数据库、schema 与 identity.md，并返回数据库路径。
    pub fn initialize(&self, user_name: &str) -> Result<String, LaputaError> {
        if self.is_initialized() {
            return Err(LaputaError::AlreadyInitialized(
                self.identity_path.display().to_string(),
            ));
        }

        let parent_dir = self.db_path.parent().ok_or_else(|| {
            LaputaError::ConfigError(format!("Invalid database path: {}", self.db_path.display()))
        })?;

        fs::create_dir_all(parent_dir)?;

        let connection = Connection::open(&self.db_path)?;
        create_schema(&connection)?;
        drop(connection);

        let created_at = Utc::now().to_rfc3339_opts(SecondsFormat::Secs, true);
        let identity_content = format!(
            "## L0 — IDENTITY\n\nuser_name: {user_name}\nuser_type: {USER_TYPE}\ncreated_at: {created_at}\n"
        );

        fs::write(&self.identity_path, identity_content)?;

        Ok(self.db_path.display().to_string())
    }
}
