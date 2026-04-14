//! 统一错误定义。

use std::error::Error;
use std::fmt::{Display, Formatter};

/// Laputa 公共接口统一错误枚举。
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum LaputaError {
    StorageError(String),
    AlreadyInitialized(String),
    NotFound(String),
    ValidationError(String),
    HeatThresholdError(i32),
    ArchiveError(String),
    WakepackSizeExceeded(usize),
    ConfigError(String),
}

impl Display for LaputaError {
    fn fmt(&self, f: &mut Formatter<'_>) -> std::fmt::Result {
        match self {
            Self::StorageError(e) => write!(f, "Storage error: {e}"),
            Self::AlreadyInitialized(path) => write!(f, "Already initialized at: {path}"),
            Self::NotFound(target) => write!(f, "Not found: {target}"),
            Self::ValidationError(e) => write!(f, "Validation error: {e}"),
            Self::HeatThresholdError(heat) => write!(f, "Heat threshold error: heat={heat}"),
            Self::ArchiveError(e) => write!(f, "Archive error: {e}"),
            Self::WakepackSizeExceeded(tokens) => {
                write!(f, "Wakepack size exceeded: {tokens} tokens")
            }
            Self::ConfigError(e) => write!(f, "Config error: {e}"),
        }
    }
}

impl Error for LaputaError {}

impl From<rusqlite::Error> for LaputaError {
    fn from(error: rusqlite::Error) -> Self {
        Self::StorageError(error.to_string())
    }
}

impl From<std::io::Error> for LaputaError {
    fn from(error: std::io::Error) -> Self {
        Self::StorageError(error.to_string())
    }
}
