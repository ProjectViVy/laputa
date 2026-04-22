use laputa::storage::memory::ensure_memory_schema;
use laputa::vector_storage::VectorStorage;
use rusqlite::{params, Connection};
use serial_test::serial;
use tempfile::tempdir;

#[test]
#[serial]
fn test_get_memories_hides_discard_candidates_by_default() {
    let dir = tempdir().unwrap();
    let db_path = dir.path().join("vectors.db");
    let index_path = dir.path().join("vectors.usearch");
    let conn = Connection::open(&db_path).unwrap();
    ensure_memory_schema(&conn).unwrap();

    conn.execute(
        "INSERT INTO memories (
            text_content, wing, room, valid_from, last_accessed, heat_i32, discard_candidate, reason
         ) VALUES (?1, ?2, ?3, ?4, ?4, ?5, ?6, ?7)",
        params!["keep me", "self", "journal", 1_i64, 5000_i32, 0_i64, "stored"],
    )
    .unwrap();
    conn.execute(
        "INSERT INTO memories (
            text_content, wing, room, valid_from, last_accessed, heat_i32, discard_candidate, reason
         ) VALUES (?1, ?2, ?3, ?4, ?4, ?5, ?6, ?7)",
        params![
            "hide me",
            "self",
            "journal",
            2_i64,
            5000_i32,
            1_i64,
            "low-value diary chatter marked as discard_candidate"
        ],
    )
    .unwrap();
    drop(conn);

    let store = VectorStorage::new(&db_path, &index_path).unwrap();
    let memories = store
        .get_memories(Some("self"), Some("journal"), 10)
        .unwrap();

    assert_eq!(memories.len(), 1);
    assert_eq!(memories[0].text_content, "keep me");
    assert_eq!(store.memory_count().unwrap(), 1);
}

#[test]
#[serial]
fn test_update_memory_after_merge_persists_reason_and_heat() {
    let dir = tempdir().unwrap();
    let db_path = dir.path().join("vectors.db");
    let index_path = dir.path().join("vectors.usearch");
    let conn = Connection::open(&db_path).unwrap();
    ensure_memory_schema(&conn).unwrap();
    conn.execute(
        "INSERT INTO memories (
            text_content, wing, room, valid_from, last_accessed, heat_i32
         ) VALUES (?1, ?2, ?3, ?4, ?4, ?5)",
        params!["winner", "self", "journal", 1_i64, 9500_i32],
    )
    .unwrap();
    let memory_id = conn.last_insert_rowid();
    drop(conn);

    let store = VectorStorage::new(&db_path, &index_path).unwrap();
    store
        .update_memory_after_merge(
            memory_id,
            "winner",
            10_000,
            "duplicate match > 0.8; merged into existing memory",
        )
        .unwrap();

    let merged = store.get_memory_by_id(memory_id).unwrap();
    assert_eq!(merged.heat_i32, 10_000);
    assert_eq!(
        merged.reason.as_deref(),
        Some("duplicate match > 0.8; merged into existing memory")
    );
}

#[test]
#[serial]
fn test_update_memory_after_merge_rejects_out_of_range_heat() {
    let dir = tempdir().unwrap();
    let db_path = dir.path().join("vectors.db");
    let index_path = dir.path().join("vectors.usearch");
    let conn = Connection::open(&db_path).unwrap();
    ensure_memory_schema(&conn).unwrap();
    conn.execute(
        "INSERT INTO memories (
            text_content, wing, room, valid_from, last_accessed, heat_i32, reason
         ) VALUES (?1, ?2, ?3, ?4, ?4, ?5, ?6)",
        params!["winner", "self", "journal", 1_i64, 9500_i32, "original"],
    )
    .unwrap();
    let memory_id = conn.last_insert_rowid();
    drop(conn);

    let store = VectorStorage::new(&db_path, &index_path).unwrap();
    let error = store
        .update_memory_after_merge(memory_id, "winner updated", 10_001, "invalid heat")
        .expect_err("out-of-range heat should be rejected");

    let laputa_error = error
        .downcast_ref::<laputa::api::LaputaError>()
        .expect("error should remain LaputaError");
    assert!(matches!(
        laputa_error,
        laputa::api::LaputaError::ValidationError(_)
    ));

    let merged = store.get_memory_by_id(memory_id).unwrap();
    assert_eq!(merged.text_content, "winner");
    assert_eq!(merged.heat_i32, 9500);
    assert_eq!(merged.reason.as_deref(), Some("original"));
}
