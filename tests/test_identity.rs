use laputa::api::LaputaError;
use laputa::identity::IdentityInitializer;
use rusqlite::Connection;
use serial_test::serial;
use tempfile::tempdir;

#[test]
#[serial]
fn test_initialize_creates_db_and_identity() {
    let dir = tempdir().unwrap();
    let initializer = IdentityInitializer::new(dir.path());

    let result = initializer.initialize("大湿");

    assert!(result.is_ok());
    assert!(dir.path().join("laputa.db").exists());
    assert!(dir.path().join("identity.md").exists());

    let content = std::fs::read_to_string(dir.path().join("identity.md")).unwrap();
    assert_eq!(
        content.lines().next(),
        Some("## L0 — IDENTITY"),
        "identity.md 顶部标题必须符合协议"
    );
    assert!(content.contains("user_name: 大湿"));
    assert!(content.contains("user_type: 个人记忆助手"));
    assert!(content.contains("created_at: "));
}

#[test]
#[serial]
fn test_reinitialize_returns_error() {
    let dir = tempdir().unwrap();
    let initializer = IdentityInitializer::new(dir.path());

    initializer.initialize("大湿").unwrap();
    let result = initializer.initialize("大湿");

    assert!(matches!(result, Err(LaputaError::AlreadyInitialized(_))));
}

#[test]
#[serial]
fn test_initialize_returns_db_path() {
    let dir = tempdir().unwrap();
    let initializer = IdentityInitializer::new(dir.path());

    let db_path = initializer.initialize("大湿").unwrap();

    assert!(db_path.ends_with("laputa.db"));
    assert!(std::path::Path::new(&db_path).exists());
}

#[test]
#[serial]
fn test_schema_created_with_required_columns() {
    let dir = tempdir().unwrap();
    let initializer = IdentityInitializer::new(dir.path());

    initializer.initialize("大湿").unwrap();

    let conn = Connection::open(dir.path().join("laputa.db")).unwrap();
    let mut statement = conn.prepare("PRAGMA table_info(memories)").unwrap();
    let columns = statement
        .query_map([], |row| row.get::<_, String>(1))
        .unwrap()
        .collect::<Result<Vec<_>, _>>()
        .unwrap();

    for required_column in [
        "id",
        "text_content",
        "wing",
        "room",
        "source_file",
        "valid_from",
        "valid_to",
        "heat_i32",
        "last_accessed",
        "access_count",
        "is_archive_candidate",
        "emotion_valence",
        "emotion_arousal",
    ] {
        assert!(
            columns.iter().any(|column| column == required_column),
            "缺少 schema 列: {required_column}"
        );
    }

    let count: i64 = conn
        .query_row("SELECT COUNT(*) FROM memories", [], |row| row.get(0))
        .unwrap();
    assert_eq!(count, 0);
}
