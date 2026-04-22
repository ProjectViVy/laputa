use laputa::storage::memory::ensure_memory_schema;
use laputa::vector_storage::VectorStorage;
use rusqlite::{params, Connection};
use serial_test::serial;
use tempfile::tempdir;

fn seed_memory(db_path: &std::path::Path, heat_i32: i32, text_content: &str) -> i64 {
    let conn = Connection::open(db_path).unwrap();
    ensure_memory_schema(&conn).unwrap();
    conn.execute(
        "INSERT INTO memories (
            text_content, wing, room, valid_from, last_accessed, heat_i32, emotion_valence, emotion_arousal
         ) VALUES (?1, ?2, ?3, ?4, ?4, ?5, ?6, ?7)",
        params![text_content, "self", "journal", 1_i64, heat_i32, 0_i32, 0_u32],
    )
    .unwrap();
    conn.last_insert_rowid()
}

#[test]
#[serial]
fn test_mark_emotion_anchor_persists_heat_and_emotion() {
    let dir = tempdir().unwrap();
    let db_path = dir.path().join("vectors.db");
    let index_path = dir.path().join("vectors.usearch");
    let memory_id = seed_memory(&db_path, 5_000, "anchor me");

    let store = VectorStorage::new(&db_path, &index_path).unwrap();
    let updated = store.mark_emotion_anchor(memory_id, 25, 80).unwrap();

    assert_eq!(updated.heat_i32, 7_000);
    assert_eq!(updated.emotion_valence, 25);
    assert_eq!(updated.emotion_arousal, 80);

    drop(store);

    let reopened = VectorStorage::new(&db_path, &index_path).unwrap();
    let persisted = reopened.get_memory_by_id(memory_id).unwrap();
    assert_eq!(persisted.heat_i32, 7_000);
    assert_eq!(persisted.emotion_valence, 25);
    assert_eq!(persisted.emotion_arousal, 80);
}

#[test]
#[serial]
fn test_mark_emotion_anchor_caps_heat_and_clamps_emotion_values() {
    let dir = tempdir().unwrap();
    let db_path = dir.path().join("vectors.db");
    let index_path = dir.path().join("vectors.usearch");
    let memory_id = seed_memory(&db_path, 9_500, "high heat");

    let store = VectorStorage::new(&db_path, &index_path).unwrap();
    let updated = store.mark_emotion_anchor(memory_id, 150, 120).unwrap();

    assert_eq!(updated.heat_i32, 10_000);
    assert_eq!(updated.emotion_valence, 100);
    assert_eq!(updated.emotion_arousal, 100);
}

#[test]
#[serial]
fn test_mark_emotion_anchor_clamps_negative_valence() {
    let dir = tempdir().unwrap();
    let db_path = dir.path().join("vectors.db");
    let index_path = dir.path().join("vectors.usearch");
    let memory_id = seed_memory(&db_path, 4_000, "negative valence");

    let store = VectorStorage::new(&db_path, &index_path).unwrap();
    let updated = store.mark_emotion_anchor(memory_id, -150, 40).unwrap();

    assert_eq!(updated.heat_i32, 6_000);
    assert_eq!(updated.emotion_valence, -100);
    assert_eq!(updated.emotion_arousal, 40);
}

#[test]
#[serial]
fn test_mark_emotion_anchor_returns_error_for_missing_memory_without_side_effects() {
    let dir = tempdir().unwrap();
    let db_path = dir.path().join("vectors.db");
    let index_path = dir.path().join("vectors.usearch");
    let existing_id = seed_memory(&db_path, 5_000, "keep intact");

    let store = VectorStorage::new(&db_path, &index_path).unwrap();
    let err = store
        .mark_emotion_anchor(existing_id + 999, 10, 20)
        .unwrap_err();

    assert!(
        err.to_string().contains("Memory not found"),
        "unexpected error: {err:#}"
    );

    let untouched = store.get_memory_by_id(existing_id).unwrap();
    assert_eq!(untouched.heat_i32, 5_000);
    assert_eq!(untouched.emotion_valence, 0);
    assert_eq!(untouched.emotion_arousal, 0);
}
