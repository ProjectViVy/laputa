use laputa::api::LaputaError;
use laputa::storage::memory::ensure_memory_schema;
use laputa::vector_storage::{MemoryInsert, VectorStorage};
use rusqlite::Connection;
use serial_test::serial;
use tempfile::tempdir;

fn make_insert(heat_i32: i32) -> MemoryInsert<'static> {
    MemoryInsert {
        text_content: "guardrail",
        wing: "self",
        room: "journal",
        source_file: None,
        source_mtime: None,
        valid_from: 1,
        heat_i32,
        emotion_valence: 0,
        emotion_arousal: 0,
        is_archive_candidate: false,
        reason: None,
        discard_candidate: false,
        merged_into_id: None,
    }
}

#[test]
#[serial]
fn test_add_memory_record_rejects_negative_heat_without_side_effects() {
    let dir = tempdir().unwrap();
    let db_path = dir.path().join("vectors.db");
    let index_path = dir.path().join("vectors.usearch");
    let conn = Connection::open(&db_path).unwrap();
    ensure_memory_schema(&conn).unwrap();
    drop(conn);

    let mut store = VectorStorage::new(&db_path, &index_path).unwrap();
    let error = store
        .add_memory_record(make_insert(-1))
        .expect_err("heat_i32=-1 应被拒绝");

    let laputa_error = error
        .downcast_ref::<LaputaError>()
        .expect("应保留为 LaputaError");
    assert!(matches!(laputa_error, LaputaError::ValidationError(_)));
    assert!(laputa_error.to_string().contains("heat_i32 out of range"));

    let count: i64 = store
        .db
        .query_row("SELECT COUNT(*) FROM memories", [], |row| row.get(0))
        .unwrap();
    assert_eq!(count, 0, "越界 heat_i32 不应写入数据库");
    assert_eq!(store.index_size(), 0, "越界 heat_i32 不应写入向量索引");
}

#[test]
#[serial]
fn test_add_memory_record_rejects_excessive_heat_without_side_effects() {
    let dir = tempdir().unwrap();
    let db_path = dir.path().join("vectors.db");
    let index_path = dir.path().join("vectors.usearch");
    let conn = Connection::open(&db_path).unwrap();
    ensure_memory_schema(&conn).unwrap();
    drop(conn);

    let mut store = VectorStorage::new(&db_path, &index_path).unwrap();
    let error = store
        .add_memory_record(make_insert(10_001))
        .expect_err("heat_i32=10001 应被拒绝");

    let laputa_error = error
        .downcast_ref::<LaputaError>()
        .expect("应保留为 LaputaError");
    assert!(matches!(laputa_error, LaputaError::ValidationError(_)));
    assert!(laputa_error.to_string().contains("heat_i32 out of range"));

    let count: i64 = store
        .db
        .query_row("SELECT COUNT(*) FROM memories", [], |row| row.get(0))
        .unwrap();
    assert_eq!(count, 0, "越界 heat_i32 不应写入数据库");
    assert_eq!(store.index_size(), 0, "越界 heat_i32 不应写入向量索引");
}
