use laputa::searcher::{Searcher, SemanticSearchOptions};
use laputa::storage::memory::ensure_memory_schema;
use laputa::vector_storage::VectorStorage;
use rusqlite::{params, Connection};
use serial_test::serial;
use tempfile::tempdir;

#[allow(clippy::too_many_arguments)]
fn insert_memory(
    conn: &Connection,
    text_content: &str,
    wing: &str,
    room: &str,
    valid_from: i64,
    heat_i32: i32,
    discard_candidate: bool,
    is_archive_candidate: bool,
) -> i64 {
    conn.execute(
        "INSERT INTO memories (
            text_content, wing, room, valid_from, last_accessed, heat_i32, discard_candidate, is_archive_candidate
         ) VALUES (?1, ?2, ?3, ?4, ?4, ?5, ?6, ?7)",
        params![
            text_content,
            wing,
            room,
            valid_from,
            heat_i32,
            if discard_candidate { 1_i64 } else { 0_i64 },
            if is_archive_candidate { 1_i64 } else { 0_i64 }
        ],
    )
    .unwrap();
    conn.last_insert_rowid()
}

fn seed_vector(store: &mut VectorStorage, id: i64, vector: Vec<f32>) {
    let needed = store.index.size() + 1;
    if needed > store.index.capacity() {
        let new_cap = (needed * 2).max(16);
        store.index.reserve(new_cap).unwrap();
    }
    store.index.add(id as u64, &vector).unwrap();
}

fn vector(a: f32, b: f32) -> Vec<f32> {
    let mut values = vec![0.0; 384];
    values[0] = a;
    values[1] = b;
    values
}

#[test]
#[serial]
fn test_vector_storage_semantic_search_returns_similarity_and_top_k() {
    let dir = tempdir().unwrap();
    let db_path = dir.path().join("vectors.db");
    let index_path = dir.path().join("vectors.usearch");
    let conn = Connection::open(&db_path).unwrap();
    ensure_memory_schema(&conn).unwrap();

    let first = insert_memory(
        &conn, "closest", "self", "journal", 100, 5_000, false, false,
    );
    let second = insert_memory(&conn, "second", "self", "journal", 101, 6_000, false, false);
    let third = insert_memory(&conn, "third", "self", "journal", 102, 7_000, false, false);
    drop(conn);

    let mut store = VectorStorage::new(&db_path, &index_path).unwrap();
    seed_vector(&mut store, first, vector(1.0, 0.0));
    seed_vector(&mut store, second, vector(0.8, 0.2));
    seed_vector(&mut store, third, vector(0.2, 0.8));

    let results = store
        .semantic_search(
            &vector(1.0, 0.0),
            2,
            Some("self"),
            Some("journal"),
            false,
            false,
        )
        .unwrap();

    assert_eq!(results.len(), 2);
    assert_eq!(results[0].0.text_content, "closest");
    assert!(results[0].1 >= results[1].1);
    assert!((results[0].1 - 1.0).abs() < 0.0001);
}

#[test]
#[serial]
fn test_vector_storage_semantic_search_filters_discarded_and_archived_by_default() {
    let dir = tempdir().unwrap();
    let db_path = dir.path().join("vectors.db");
    let index_path = dir.path().join("vectors.usearch");
    let conn = Connection::open(&db_path).unwrap();
    ensure_memory_schema(&conn).unwrap();

    let keep = insert_memory(&conn, "keep", "self", "journal", 100, 5_000, false, false);
    let discarded = insert_memory(
        &conn,
        "discarded",
        "self",
        "journal",
        101,
        9_000,
        true,
        false,
    );
    let archived = insert_memory(
        &conn, "archived", "self", "journal", 102, 9_500, false, true,
    );
    drop(conn);

    let mut store = VectorStorage::new(&db_path, &index_path).unwrap();
    for id in [keep, discarded, archived] {
        seed_vector(&mut store, id, vector(1.0, 0.0));
    }

    let default_results = store
        .semantic_search(
            &vector(1.0, 0.0),
            10,
            Some("self"),
            Some("journal"),
            false,
            false,
        )
        .unwrap();
    let default_texts: Vec<&str> = default_results
        .iter()
        .map(|(record, _)| record.text_content.as_str())
        .collect();
    assert_eq!(default_texts, vec!["keep"]);

    let included = store
        .semantic_search(
            &vector(1.0, 0.0),
            10,
            Some("self"),
            Some("journal"),
            true,
            false,
        )
        .unwrap();
    let included_texts: Vec<&str> = included
        .iter()
        .map(|(record, _)| record.text_content.as_str())
        .collect();
    assert!(included_texts.contains(&"keep"));
    assert!(included_texts.contains(&"discarded"));
    assert!(!included_texts.contains(&"archived"));
}

#[test]
#[serial]
fn test_vector_storage_semantic_search_can_rerank_by_heat() {
    let dir = tempdir().unwrap();
    let db_path = dir.path().join("vectors.db");
    let index_path = dir.path().join("vectors.usearch");
    let conn = Connection::open(&db_path).unwrap();
    ensure_memory_schema(&conn).unwrap();

    let high_similarity = insert_memory(
        &conn,
        "high similarity low heat",
        "self",
        "journal",
        100,
        4_000,
        false,
        false,
    );
    let high_heat = insert_memory(
        &conn,
        "lower similarity high heat",
        "self",
        "journal",
        101,
        9_500,
        false,
        false,
    );
    drop(conn);

    let mut store = VectorStorage::new(&db_path, &index_path).unwrap();
    seed_vector(&mut store, high_similarity, vector(1.0, 0.0));
    seed_vector(&mut store, high_heat, vector(0.7, 0.3));

    let similarity_order = store
        .semantic_search(
            &vector(1.0, 0.0),
            2,
            Some("self"),
            Some("journal"),
            false,
            false,
        )
        .unwrap();
    assert_eq!(
        similarity_order[0].0.text_content,
        "high similarity low heat"
    );

    let heat_order = store
        .semantic_search(
            &vector(1.0, 0.0),
            2,
            Some("self"),
            Some("journal"),
            false,
            true,
        )
        .unwrap();
    assert_eq!(heat_order[0].0.text_content, "lower similarity high heat");
}

#[test]
#[serial]
fn test_vector_storage_semantic_search_empty_index_returns_empty() {
    let dir = tempdir().unwrap();
    let db_path = dir.path().join("vectors.db");
    let index_path = dir.path().join("vectors.usearch");
    let conn = Connection::open(&db_path).unwrap();
    ensure_memory_schema(&conn).unwrap();
    drop(conn);

    let store = VectorStorage::new(&db_path, &index_path).unwrap();
    let results = store
        .semantic_search(
            &vector(1.0, 0.0),
            5,
            Some("self"),
            Some("journal"),
            false,
            false,
        )
        .unwrap();
    assert!(results.is_empty());
}

#[test]
fn test_semantic_search_options_default() {
    let options = SemanticSearchOptions::default();
    assert_eq!(options.wing, None);
    assert_eq!(options.room, None);
    assert!(!options.include_discarded);
    assert!(!options.sort_by_heat);
}

#[tokio::test]
async fn test_searcher_semantic_search_graceful_when_storage_or_embeddings_unavailable() {
    let searcher = Searcher::new(laputa::config::MempalaceConfig::default());
    let results = searcher
        .semantic_search(
            "query",
            5,
            SemanticSearchOptions {
                wing: None,
                room: None,
                include_discarded: false,
                sort_by_heat: false,
            },
        )
        .await
        .unwrap();
    assert!(results.is_empty());
}

#[test]
fn test_format_semantic_json_results_empty() {
    let result =
        Searcher::format_semantic_json_results("query", &SemanticSearchOptions::default(), &[]);
    assert_eq!(result["query"], "query");
    assert!(result["results"].as_array().unwrap().is_empty());
}
