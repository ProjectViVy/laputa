use std::path::{Path, PathBuf};
use tempfile::{tempdir, TempDir};

/// Real filesystem fixture for tests that need isolated config and SQLite files.
#[allow(dead_code)]
pub struct TempDirFixture {
    temp_dir: TempDir,
}

#[allow(dead_code)]
impl TempDirFixture {
    pub fn new() -> Self {
        Self {
            temp_dir: tempdir().expect("tempdir fixture should be created"),
        }
    }

    pub fn path(&self) -> &Path {
        self.temp_dir.path()
    }

    pub fn join(&self, name: &str) -> PathBuf {
        self.path().join(name)
    }
}
