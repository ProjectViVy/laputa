use chrono::{TimeZone, Utc};
use laputa::api::LaputaError;
use laputa::storage::memory::{ensure_memory_schema, LaputaMemoryRecord};
use rusqlite::Connection;
use serial_test::serial;
use tempfile::tempdir;

fn make_record() -> LaputaMemoryRecord {
    LaputaMemoryRecord::new(
        42,
        "memory text".to_string(),
        "self".to_string(),
        "journal".to_string(),
        Some("memory.md".to_string()),
        1_710_000_000,
        None,
        0.75,
        5.0,
    )
}

#[test]
fn test_laputa_memory_record_defaults() {
    let record = make_record();

    assert_eq!(record.heat_i32, 5_000);
    assert_eq!(record.get_heat(), 50.0);
    assert_eq!(
        record.last_accessed,
        Utc.timestamp_opt(0, 0).single().unwrap()
    );
    assert_eq!(record.access_count, 0);
    assert_eq!(record.emotion_valence, 0);
    assert_eq!(record.emotion_arousal, 0);
    assert!(!record.is_archive_candidate);
}

#[test]
fn test_heat_roundtrip_conversion() {
    let mut record = make_record();
    record.set_heat(50.0).unwrap();
    assert_eq!(record.heat_i32, 5_000);
    assert_eq!(record.get_heat(), 50.0);

    record.set_heat(72.55).unwrap();
    assert_eq!(record.heat_i32, 7_255);
    assert_eq!(record.get_heat(), 72.55);
}

#[test]
fn test_set_heat_below_zero_returns_error() {
    let mut record = make_record();
    let result = record.set_heat(-1.0);
    assert!(result.is_err());
    let err = result.unwrap_err();
    assert!(matches!(err, LaputaError::ValidationError(_)));
    // Original heat unchanged after error
    assert_eq!(record.heat_i32, 5_000);
}

#[test]
fn test_set_heat_above_max_returns_error() {
    let mut record = make_record();
    let result = record.set_heat(101.0);
    assert!(result.is_err());
    let err = result.unwrap_err();
    assert!(matches!(err, LaputaError::ValidationError(_)));
    // Original heat unchanged after error
    assert_eq!(record.heat_i32, 5_000);
}

#[test]
fn test_set_heat_nan_returns_error() {
    let mut record = make_record();
    let result = record.set_heat(f64::NAN);
    assert!(result.is_err());
    let err = result.unwrap_err();
    assert!(matches!(err, LaputaError::ValidationError(_)));
    // Original heat unchanged after error
    assert_eq!(record.heat_i32, 5_000);
}

#[test]
fn test_set_heat_boundary_values_success() {
    let mut record = make_record();
    // Lower boundary: 0.0
    record.set_heat(0.0).unwrap();
    assert_eq!(record.heat_i32, 0);
    assert_eq!(record.get_heat(), 0.0);

    // Upper boundary: 100.0
    record.set_heat(100.0).unwrap();
    assert_eq!(record.heat_i32, 10_000);
    assert_eq!(record.get_heat(), 100.0);
}

#[test]
fn test_emotion_boundaries_and_archive_marking() {
    let mut record = make_record();

    record.update_emotion(-150, 120);
    assert_eq!(record.emotion_valence, -100);
    assert_eq!(record.emotion_arousal, 100);

    record.update_emotion(80, 20);
    assert_eq!(record.emotion_valence, 80);
    assert_eq!(record.emotion_arousal, 20);

    record.mark_archive_candidate();
    assert!(record.is_archive_candidate);
}

#[test]
fn test_with_updated_heat_returns_new_record() {
    let record = make_record();
    let updated = record.with_updated_heat(66.25);

    assert_eq!(record.heat_i32, 5_000);
    assert_eq!(updated.heat_i32, 6_625);
    assert_eq!(updated.get_heat(), 66.25);
    assert_eq!(record.id, updated.id);
    assert_eq!(record.text_content, updated.text_content);
}

#[test]
#[serial]
fn test_schema_migration_adds_memory_extension_columns() {
    let dir = tempdir().unwrap();
    let db_path = dir.path().join("vectors.db");
    let conn = Connection::open(&db_path).unwrap();

    conn.execute_batch(
        r#"
        CREATE TABLE memories (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            text_content TEXT NOT NULL,
            wing TEXT NOT NULL,
            room TEXT NOT NULL,
            source_file TEXT,
            source_mtime REAL,
            valid_from INTEGER NOT NULL,
            valid_to INTEGER,
            last_accessed INTEGER DEFAULT 0,
            access_count INTEGER DEFAULT 0,
            importance_score REAL DEFAULT 5.0
        );
        "#,
    )
    .unwrap();

    ensure_memory_schema(&conn).unwrap();

    let mut statement = conn.prepare("PRAGMA table_info(memories)").unwrap();
    let columns = statement
        .query_map([], |row| row.get::<_, String>(1))
        .unwrap()
        .collect::<Result<Vec<_>, _>>()
        .unwrap();

    for required_column in [
        "heat_i32",
        "emotion_valence",
        "emotion_arousal",
        "is_archive_candidate",
        "reason",
        "discard_candidate",
        "merged_into_id",
    ] {
        assert!(
            columns.iter().any(|column| column == required_column),
            "missing column {required_column}"
        );
    }

    let mut index_stmt = conn.prepare("PRAGMA index_list(memories)").unwrap();
    let indexes = index_stmt
        .query_map([], |row| row.get::<_, String>(1))
        .unwrap()
        .collect::<Result<Vec<_>, _>>()
        .unwrap();
    assert!(indexes.iter().any(|index| index == "idx_heat"));
    assert!(indexes.iter().any(|index| index == "idx_discard_candidate"));
}
