use laputa::api::LaputaError;
use laputa::config::MempalaceConfig;
use laputa::searcher::{RecallQuery, Searcher};
use laputa::storage::memory::ensure_memory_schema;
use laputa::storage::MemoryStack;
use laputa::vector_storage::VectorStorage;
use rusqlite::{params, Connection};
use serial_test::serial;
use tempfile::tempdir;

fn insert_memory(
    conn: &Connection,
    text_content: &str,
    wing: &str,
    room: &str,
    valid_from: i64,
    heat_i32: i32,
    discard_candidate: bool,
) {
    conn.execute(
        "INSERT INTO memories (
            text_content, wing, room, valid_from, last_accessed, heat_i32, discard_candidate
         ) VALUES (?1, ?2, ?3, ?4, ?4, ?5, ?6)",
        params![
            text_content,
            wing,
            room,
            valid_from,
            heat_i32,
            if discard_candidate { 1_i64 } else { 0_i64 }
        ],
    )
    .unwrap();
}

#[test]
fn test_recall_query_defaults_and_limit_clamp() {
    let query = RecallQuery::by_time_range(100, 200);
    assert_eq!(query.start, 100);
    assert_eq!(query.end, 200);
    assert_eq!(query.limit, 100);
    assert!(!query.include_discarded);

    let query = RecallQuery::by_time_range(100, 200).with_limit(2_500);
    assert_eq!(query.limit, 1_000);
}

#[test]
#[serial]
fn test_vector_storage_recall_by_time_range_filters_and_sorts() {
    let dir = tempdir().unwrap();
    let db_path = dir.path().join("vectors.db");
    let index_path = dir.path().join("vectors.usearch");
    let conn = Connection::open(&db_path).unwrap();
    ensure_memory_schema(&conn).unwrap();

    insert_memory(&conn, "too early", "self", "journal", 90, 9_999, false);
    insert_memory(
        &conn,
        "latest medium heat",
        "self",
        "journal",
        160,
        6_000,
        false,
    );
    insert_memory(&conn, "highest heat", "self", "journal", 150, 9_000, false);
    insert_memory(
        &conn,
        "same heat older",
        "self",
        "journal",
        120,
        9_000,
        false,
    );
    insert_memory(&conn, "discarded", "self", "journal", 140, 9_500, true);
    insert_memory(&conn, "other room", "self", "ideas", 130, 8_500, false);
    drop(conn);

    let store = VectorStorage::new(&db_path, &index_path).unwrap();
    let query = RecallQuery::by_time_range(100, 160)
        .with_wing("self")
        .with_room("journal");

    let records = store.recall_by_time_range(&query).unwrap();
    let texts: Vec<&str> = records
        .iter()
        .map(|record| record.text_content.as_str())
        .collect();

    assert_eq!(
        texts,
        vec!["highest heat", "same heat older", "latest medium heat"]
    );
}

#[test]
#[serial]
fn test_vector_storage_recall_by_time_range_can_include_discarded_and_limit() {
    let dir = tempdir().unwrap();
    let db_path = dir.path().join("vectors.db");
    let index_path = dir.path().join("vectors.usearch");
    let conn = Connection::open(&db_path).unwrap();
    ensure_memory_schema(&conn).unwrap();

    insert_memory(&conn, "keep-a", "self", "journal", 110, 5_000, false);
    insert_memory(&conn, "discarded-top", "self", "journal", 120, 9_500, true);
    insert_memory(&conn, "keep-b", "self", "journal", 130, 7_000, false);
    drop(conn);

    let store = VectorStorage::new(&db_path, &index_path).unwrap();
    let query = RecallQuery::by_time_range(100, 200)
        .with_wing("self")
        .with_room("journal")
        .include_discarded(true)
        .with_limit(2);

    let records = store.recall_by_time_range(&query).unwrap();
    let texts: Vec<&str> = records
        .iter()
        .map(|record| record.text_content.as_str())
        .collect();

    assert_eq!(texts, vec!["discarded-top", "keep-b"]);
}

#[test]
#[serial]
fn test_vector_storage_recall_by_time_range_validates_range() {
    let dir = tempdir().unwrap();
    let db_path = dir.path().join("vectors.db");
    let index_path = dir.path().join("vectors.usearch");
    let conn = Connection::open(&db_path).unwrap();
    ensure_memory_schema(&conn).unwrap();
    drop(conn);

    let store = VectorStorage::new(&db_path, &index_path).unwrap();
    let query = RecallQuery::by_time_range(200, 100);
    let error = store.recall_by_time_range(&query).unwrap_err();
    let laputa_error = error.downcast_ref::<LaputaError>().unwrap();

    assert!(matches!(laputa_error, LaputaError::ValidationError(_)));
}

#[tokio::test]
#[serial]
async fn test_searcher_recall_by_time_range_integrates_with_vector_storage() {
    let dir = tempdir().unwrap();
    let config = MempalaceConfig::new(Some(dir.path().to_path_buf()));
    let db_path = dir.path().join("vectors.db");
    let conn = Connection::open(&db_path).unwrap();
    ensure_memory_schema(&conn).unwrap();
    insert_memory(
        &conn,
        "timeline result",
        "self",
        "journal",
        150,
        7_200,
        false,
    );
    drop(conn);

    let searcher = Searcher::new(config.clone());
    let records = searcher
        .recall_by_time_range(
            RecallQuery::by_time_range(100, 200)
                .with_wing("self")
                .with_room("journal"),
        )
        .await
        .unwrap();

    assert_eq!(records.len(), 1);
    assert_eq!(records[0].text_content, "timeline result");

    let reopened = VectorStorage::new(&db_path, dir.path().join("vectors.usearch")).unwrap();
    let updated = reopened.get_memory_by_id(records[0].id).unwrap();
    assert_eq!(updated.access_count, 1);
    assert!(updated.last_accessed.timestamp() >= 150);
}

#[tokio::test]
#[serial]
async fn test_memory_stack_recall_by_time_range_returns_formatted_output() {
    let dir = tempdir().unwrap();
    let config = MempalaceConfig::new(Some(dir.path().to_path_buf()));
    let db_path = dir.path().join("vectors.db");
    let conn = Connection::open(&db_path).unwrap();
    ensure_memory_schema(&conn).unwrap();
    insert_memory(
        &conn,
        "timeline snippet",
        "self",
        "journal",
        140,
        8_000,
        false,
    );
    drop(conn);

    let stack = MemoryStack::new(config);
    let output = stack
        .recall_by_time_range(
            RecallQuery::by_time_range(100, 200)
                .with_wing("self")
                .with_room("journal"),
        )
        .await;

    assert!(output.contains("timeline snippet"));
    assert!(output.contains("journal"));
}
