use chrono::Utc;
use laputa::config::MempalaceConfig;
use laputa::searcher::{
    load_hybrid_ranking_config, normalize_heat_score, normalize_time_score, HybridQuery,
    HybridRankingConfig, HybridSearchResult, RecallQuery, SearchResult, Searcher,
};
use laputa::storage::memory::{ensure_memory_schema, LaputaMemoryRecord};
use rusqlite::{params, Connection};
use serial_test::serial;
use tempfile::tempdir;

fn make_record(id: i64, text: &str, valid_from: i64, heat_i32: i32) -> LaputaMemoryRecord {
    let mut record = LaputaMemoryRecord::new(
        id,
        text.to_string(),
        "self".to_string(),
        "journal".to_string(),
        None,
        valid_from,
        None,
        0.0,
        5.0,
    );
    record.heat_i32 = heat_i32;
    record
}

fn insert_memory(
    conn: &Connection,
    text_content: &str,
    wing: &str,
    room: &str,
    valid_from: i64,
    heat_i32: i32,
) -> i64 {
    conn.execute(
        "INSERT INTO memories (
            text_content, wing, room, valid_from, last_accessed, heat_i32, discard_candidate
         ) VALUES (?1, ?2, ?3, ?4, ?4, ?5, 0)",
        params![text_content, wing, room, valid_from, heat_i32],
    )
    .unwrap();
    conn.last_insert_rowid()
}

#[test]
fn test_normalize_time_score_prefers_center_and_clamps_edges() {
    assert!((normalize_time_score(150, 100, 200) - 1.0).abs() < 0.0001);
    assert_eq!(normalize_time_score(100, 100, 200), 0.0);
    assert_eq!(normalize_time_score(200, 100, 200), 0.0);
    assert_eq!(normalize_time_score(50, 100, 200), 0.0);
}

#[test]
fn test_normalize_heat_score_maps_range() {
    assert_eq!(normalize_heat_score(0), 0.0);
    assert!((normalize_heat_score(5_000) - 0.5).abs() < 0.0001);
    assert_eq!(normalize_heat_score(10_000), 1.0);
    assert_eq!(normalize_heat_score(15_000), 1.0);
}

#[test]
fn test_hybrid_query_default_top_k_and_ranking_config() {
    let query = HybridQuery::new("memory", RecallQuery::by_time_range(100, 200));
    assert_eq!(query.top_k, 100);
    assert_eq!(query.semantic_limit, 200);
    assert_eq!(query.ranking_config, HybridRankingConfig::default());
}

#[test]
fn test_hybrid_search_result_computes_composite_score() {
    let mut result = HybridSearchResult::new(make_record(1, "entry", 150, 8_000));
    let config = HybridRankingConfig::default();
    result.time_score = 0.8;
    result.semantic_score = 0.5;
    result.heat_score = 0.8;
    result.recompute_composite_score(&config);

    assert!((result.composite_score - 0.68).abs() < 0.0001);
}

#[test]
fn test_hybrid_search_merge_deduplicates_and_keeps_all_scores() {
    let query = HybridQuery::new("memory", RecallQuery::by_time_range(100, 200)).with_top_k(10);
    let time_results = vec![
        make_record(1, "time+semantic", 150, 8_000),
        make_record(2, "time-only", 160, 9_000),
    ];
    let semantic_results = vec![
        SearchResult {
            record: make_record(1, "time+semantic", 150, 8_000),
            similarity: 0.95,
            rank: 1,
        },
        SearchResult {
            record: make_record(3, "semantic-only", 90, 7_000),
            similarity: 0.85,
            rank: 2,
        },
    ];

    let results = Searcher::merge_hybrid_results(&query, time_results, semantic_results);
    assert_eq!(results.len(), 3);

    let overlap = results.iter().find(|item| item.record.id == 1).unwrap();
    assert!(overlap.time_score > 0.0);
    assert!(overlap.semantic_score > 0.0);

    let semantic_only = results.iter().find(|item| item.record.id == 3).unwrap();
    assert_eq!(semantic_only.time_score, 0.0);
    assert!(semantic_only.semantic_score > 0.0);
}

#[test]
fn test_hybrid_query_top_k_is_clamped() {
    let query = HybridQuery::new("memory", RecallQuery::by_time_range(100, 200)).with_top_k(5_000);
    assert_eq!(query.top_k, 1_000);
    assert_eq!(query.semantic_limit, 1_000);
}

#[test]
fn test_load_hybrid_ranking_config_from_toml() {
    let dir = tempdir().unwrap();
    std::fs::write(
        dir.path().join("config.toml"),
        "[search.hybrid]\ntime_weight = 0.2\nsemantic_weight = 0.5\nheat_weight = 0.3\n",
    )
    .unwrap();

    let config = load_hybrid_ranking_config(dir.path()).unwrap();
    assert_eq!(
        config,
        HybridRankingConfig {
            time_weight: 0.2,
            semantic_weight: 0.5,
            heat_weight: 0.3,
        }
    );
}

#[tokio::test]
#[serial]
async fn test_searcher_hybrid_search_returns_time_results_when_semantic_unavailable() {
    let dir = tempdir().unwrap();
    let config = MempalaceConfig::new(Some(dir.path().to_path_buf()));
    let db_path = config.config_dir.join("vectors.db");
    let conn = Connection::open(&db_path).unwrap();
    ensure_memory_schema(&conn).unwrap();

    let now = Utc::now().timestamp();
    insert_memory(&conn, "recent memory", "self", "journal", now - 20, 8_000);
    insert_memory(&conn, "older memory", "self", "journal", now - 60, 6_000);
    drop(conn);

    let searcher = Searcher::new(config);
    let results = searcher
        .hybrid_search(HybridQuery::new(
            "semantic query",
            RecallQuery::by_time_range(now - 120, now).with_wing("self"),
        ))
        .await
        .unwrap();

    assert_eq!(results.len(), 2);
    assert!(results[0].record.heat_i32 >= results[1].record.heat_i32);
}
