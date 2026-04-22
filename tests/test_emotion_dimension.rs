use laputa::dialect::canonical_emotion_code;
use laputa::storage::memory::ensure_memory_schema;
use laputa::vector_storage::{EmotionQuery, EmotionSort, VectorStorage};
use rusqlite::{params, Connection};
use serial_test::serial;
use tempfile::tempdir;

fn seed_memory(
    db_path: &std::path::Path,
    valid_from: i64,
    text_content: &str,
    valence: i32,
    arousal: u32,
) -> i64 {
    let conn = Connection::open(db_path).unwrap();
    ensure_memory_schema(&conn).unwrap();
    conn.execute(
        "INSERT INTO memories (
            text_content, wing, room, valid_from, last_accessed, heat_i32, emotion_valence, emotion_arousal
         ) VALUES (?1, ?2, ?3, ?4, ?4, ?5, ?6, ?7)",
        params![
            text_content,
            "self",
            "journal",
            valid_from,
            5_000_i32,
            valence,
            arousal
        ],
    )
    .unwrap();
    conn.last_insert_rowid()
}

#[test]
#[serial]
fn test_update_memory_emotion_persists_clamped_values() {
    let dir = tempdir().unwrap();
    let db_path = dir.path().join("vectors.db");
    let index_path = dir.path().join("vectors.usearch");
    let memory_id = seed_memory(&db_path, 1, "emotion target", 0, 0);

    let store = VectorStorage::new(&db_path, &index_path).unwrap();
    let updated = store.update_memory_emotion(memory_id, 150, 120).unwrap();

    assert_eq!(updated.emotion_valence, 100);
    assert_eq!(updated.emotion_arousal, 100);

    drop(store);

    let reopened = VectorStorage::new(&db_path, &index_path).unwrap();
    let persisted = reopened.get_memory_by_id(memory_id).unwrap();
    assert_eq!(persisted.emotion_valence, 100);
    assert_eq!(persisted.emotion_arousal, 100);
}

#[test]
#[serial]
fn test_update_memory_emotion_keeps_defaults_until_explicitly_set() {
    let dir = tempdir().unwrap();
    let db_path = dir.path().join("vectors.db");
    let index_path = dir.path().join("vectors.usearch");
    let memory_id = seed_memory(&db_path, 1, "default emotion", 0, 0);

    let store = VectorStorage::new(&db_path, &index_path).unwrap();
    let record = store.get_memory_by_id(memory_id).unwrap();

    assert_eq!(record.emotion_valence, 0);
    assert_eq!(record.emotion_arousal, 0);
}

#[test]
#[serial]
fn test_list_memories_by_emotion_filters_and_sorts_by_arousal() {
    let dir = tempdir().unwrap();
    let db_path = dir.path().join("vectors.db");
    let index_path = dir.path().join("vectors.usearch");
    seed_memory(&db_path, 1, "calm positive", 40, 20);
    let target_id = seed_memory(&db_path, 2, "intense positive", 65, 90);
    seed_memory(&db_path, 3, "negative intense", -70, 95);

    let store = VectorStorage::new(&db_path, &index_path).unwrap();
    let records = store
        .list_memories_by_emotion(&EmotionQuery {
            wing: Some("self".to_string()),
            room: Some("journal".to_string()),
            min_valence: Some(20),
            min_arousal: Some(10),
            max_arousal: Some(100),
            include_discarded: false,
            limit: 10,
            sort: EmotionSort::ArousalDesc,
            ..EmotionQuery::default()
        })
        .unwrap();

    assert_eq!(records.len(), 2);
    assert_eq!(records[0].id, target_id);
    assert!(records[0].emotion_arousal >= records[1].emotion_arousal);
    assert!(records.iter().all(|record| record.emotion_valence >= 20));
}

#[test]
#[serial]
fn test_list_memories_by_emotion_can_sort_by_absolute_valence() {
    let dir = tempdir().unwrap();
    let db_path = dir.path().join("vectors.db");
    let index_path = dir.path().join("vectors.usearch");
    let lower_abs = seed_memory(&db_path, 1, "moderate", 45, 70);
    let higher_abs = seed_memory(&db_path, 2, "strongly negative", -90, 40);

    let store = VectorStorage::new(&db_path, &index_path).unwrap();
    let records = store
        .list_memories_by_emotion(&EmotionQuery {
            limit: 10,
            sort: EmotionSort::AbsoluteValenceDesc,
            ..EmotionQuery::default()
        })
        .unwrap();

    assert_eq!(records.len(), 2);
    assert_eq!(records[0].id, higher_abs);
    assert_eq!(records[1].id, lower_abs);
}

#[test]
fn test_canonical_emotion_code_reuses_existing_dictionary() {
    assert_eq!(canonical_emotion_code(" joy "), Some("joy"));
    assert_eq!(canonical_emotion_code("trust_building"), Some("trust"));
    assert_eq!(canonical_emotion_code("unknown-emotion"), None);
}
