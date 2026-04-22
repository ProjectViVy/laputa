use laputa::storage::memory::ensure_memory_schema;
use laputa::vector_storage::{UserIntervention, VectorStorage};
use rusqlite::{params, Connection};
use serial_test::serial;
use tempfile::tempdir;

fn seed_memory(db_path: &std::path::Path, heat_i32: i32, text_content: &str) -> i64 {
    let conn = Connection::open(db_path).unwrap();
    ensure_memory_schema(&conn).unwrap();
    conn.execute(
        "INSERT INTO memories (
            text_content, wing, room, valid_from, last_accessed, heat_i32, emotion_valence, emotion_arousal, is_archive_candidate, reason
         ) VALUES (?1, ?2, ?3, ?4, ?4, ?5, ?6, ?7, ?8, ?9)",
        params![
            text_content,
            "self",
            "journal",
            1_i64,
            heat_i32,
            0_i32,
            0_u32,
            0_i64,
            Option::<String>::None
        ],
    )
    .unwrap();
    conn.last_insert_rowid()
}

#[test]
#[serial]
fn test_mark_important_sets_locked_heat_and_reason() {
    let dir = tempdir().unwrap();
    let db_path = dir.path().join("vectors.db");
    let index_path = dir.path().join("vectors.usearch");
    let memory_id = seed_memory(&db_path, 4_200, "important memory");

    let store = VectorStorage::new(&db_path, &index_path).unwrap();
    let updated = store
        .apply_intervention(
            memory_id,
            UserIntervention::Important {
                reason: "keep visible".to_string(),
            },
        )
        .unwrap();

    assert_eq!(updated.heat_i32, 9_000);
    assert_eq!(updated.reason.as_deref(), Some("keep visible"));
    assert!(!updated.is_archive_candidate);
}

#[test]
#[serial]
fn test_mark_forget_sets_archive_candidate_and_reason() {
    let dir = tempdir().unwrap();
    let db_path = dir.path().join("vectors.db");
    let index_path = dir.path().join("vectors.usearch");
    let memory_id = seed_memory(&db_path, 7_800, "forget memory");

    let store = VectorStorage::new(&db_path, &index_path).unwrap();
    let updated = store
        .apply_intervention(
            memory_id,
            UserIntervention::Forget {
                reason: "no longer needed".to_string(),
            },
        )
        .unwrap();

    assert_eq!(updated.heat_i32, 0);
    assert!(updated.is_archive_candidate);
    assert_eq!(updated.reason.as_deref(), Some("no longer needed"));
}

#[test]
#[serial]
fn test_mark_emotion_anchor_persists_heat_emotion_and_reason() {
    let dir = tempdir().unwrap();
    let db_path = dir.path().join("vectors.db");
    let index_path = dir.path().join("vectors.usearch");
    let memory_id = seed_memory(&db_path, 8_500, "emotion memory");

    let store = VectorStorage::new(&db_path, &index_path).unwrap();
    let updated = store
        .apply_intervention(
            memory_id,
            UserIntervention::EmotionAnchor {
                valence: 150,
                arousal: 120,
                reason: "strong emotional signal".to_string(),
            },
        )
        .unwrap();

    assert_eq!(updated.heat_i32, 10_000);
    assert_eq!(updated.emotion_valence, 100);
    assert_eq!(updated.emotion_arousal, 100);
    assert_eq!(updated.reason.as_deref(), Some("strong emotional signal"));
}

#[test]
#[serial]
fn test_mark_emotion_anchor_boundary_heat_transitions_through_apply_intervention() {
    let cases = [
        (0, 25, 80, 2_000, 25, 80, "cold anchor"),
        (9_000, 150, 120, 10_000, 100, 100, "capped anchor"),
        (10_000, -150, 20, 10_000, -100, 20, "locked anchor"),
    ];

    for (
        initial_heat_i32,
        valence,
        arousal,
        expected_heat_i32,
        expected_valence,
        expected_arousal,
        reason,
    ) in cases
    {
        let dir = tempdir().unwrap();
        let db_path = dir.path().join("vectors.db");
        let index_path = dir.path().join("vectors.usearch");
        let memory_id = seed_memory(&db_path, initial_heat_i32, reason);

        let store = VectorStorage::new(&db_path, &index_path).unwrap();
        let updated = store
            .apply_intervention(
                memory_id,
                UserIntervention::EmotionAnchor {
                    valence,
                    arousal,
                    reason: reason.to_string(),
                },
            )
            .unwrap();

        assert_eq!(updated.heat_i32, expected_heat_i32);
        assert_eq!(updated.emotion_valence, expected_valence);
        assert_eq!(updated.emotion_arousal, expected_arousal);
        assert_eq!(updated.reason.as_deref(), Some(reason));

        let persisted = store.get_memory_by_id(memory_id).unwrap();
        assert_eq!(persisted.heat_i32, expected_heat_i32);
        assert_eq!(persisted.emotion_valence, expected_valence);
        assert_eq!(persisted.emotion_arousal, expected_arousal);
        assert_eq!(persisted.reason.as_deref(), Some(reason));
    }
}

fn assert_memory_untouched_after_missing_intervention(
    intervention: UserIntervention,
    existing_heat_i32: i32,
) {
    let dir = tempdir().unwrap();
    let db_path = dir.path().join("vectors.db");
    let index_path = dir.path().join("vectors.usearch");
    let existing_id = seed_memory(&db_path, existing_heat_i32, "keep intact");

    let store = VectorStorage::new(&db_path, &index_path).unwrap();
    let err = store
        .apply_intervention(existing_id + 404, intervention)
        .unwrap_err();

    assert!(
        err.to_string().contains("Memory not found"),
        "unexpected error: {err:#}"
    );

    let untouched = store.get_memory_by_id(existing_id).unwrap();
    assert_eq!(untouched.heat_i32, existing_heat_i32);
    assert_eq!(untouched.reason, None);
    assert!(!untouched.is_archive_candidate);
}

#[test]
#[serial]
fn test_missing_important_memory_returns_error_without_side_effects() {
    assert_memory_untouched_after_missing_intervention(
        UserIntervention::Important {
            reason: "missing target".to_string(),
        },
        5_000,
    );
}

#[test]
#[serial]
fn test_missing_forget_memory_returns_error_without_side_effects() {
    assert_memory_untouched_after_missing_intervention(
        UserIntervention::Forget {
            reason: "missing target".to_string(),
        },
        5_000,
    );
}
