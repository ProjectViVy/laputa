use laputa::config::MempalaceConfig;
use laputa::knowledge_graph::{KnowledgeGraph, RelationKind};
use laputa::searcher::Searcher;
use laputa::storage::memory::ensure_memory_schema;
use laputa::vector_storage::VectorStorage;
use laputa::wakeup::WakePackGenerator;
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
) {
    conn.execute(
        "INSERT INTO memories (
            text_content, wing, room, valid_from, last_accessed, heat_i32, discard_candidate
         ) VALUES (?1, ?2, ?3, ?4, ?4, ?5, 0)",
        params![text_content, wing, room, valid_from, heat_i32],
    )
    .unwrap();
}

fn write_identity(config: &MempalaceConfig, user_name: &str) {
    std::fs::write(
        config.config_dir.join("identity.md"),
        format!(
            "## L0 - IDENTITY\n\nuser_name: {user_name}\nuser_type: personal-memory-assistant\ncreated_at: 2026-04-01T08:00:00Z\nfavorite_mode: focused\n"
        ),
    )
    .unwrap();
}

fn write_capsule(config: &MempalaceConfig, content: &str) {
    let rhythm_dir = config.config_dir.join("rhythm");
    std::fs::create_dir_all(&rhythm_dir).unwrap();
    std::fs::write(rhythm_dir.join("latest-weekly-capsule.md"), content).unwrap();
}

#[test]
#[serial]
fn test_wakepack_generator_builds_structured_payload_with_filters() {
    let dir = tempdir().unwrap();
    let config = MempalaceConfig::new(Some(dir.path().to_path_buf()));
    write_identity(&config, "tester");
    write_capsule(
        &config,
        "# Weekly Capsule\n- shipped search improvements\n- keep focus on retrieval quality",
    );

    let db_path = config.config_dir.join("vectors.db");
    let conn = Connection::open(&db_path).unwrap();
    ensure_memory_schema(&conn).unwrap();

    let now = chrono::Utc::now().timestamp();
    insert_memory(
        &conn,
        "High heat within seven days",
        "self",
        "journal",
        now - 60,
        9_800,
    );
    insert_memory(
        &conn,
        "Older than seven days and should be excluded",
        "self",
        "journal",
        now - (9 * 24 * 60 * 60),
        9_999,
    );
    insert_memory(
        &conn,
        "Second hottest recent memory",
        "self",
        "project",
        now - 120,
        8_100,
    );
    drop(conn);

    let kg = KnowledgeGraph::new(config.config_dir.join("knowledge.db").to_str().unwrap()).unwrap();
    kg.upsert_relation(
        "tester",
        "Laputa",
        RelationKind::PersonProject,
        91,
        Some("2026-04-14"),
        None,
        Some("kg.md"),
    )
    .unwrap();
    kg.upsert_relation(
        "tester",
        "LowSignal",
        RelationKind::PersonPerson,
        32,
        Some("2026-04-14"),
        None,
        None,
    )
    .unwrap();

    let generator = WakePackGenerator::new(config.clone());
    let pack = generator.generate(Some("self".to_string())).unwrap();

    assert_eq!(pack.identity.user_name.as_deref(), Some("tester"));
    assert_eq!(pack.recent_state.len(), 2);
    assert_eq!(pack.recent_state[0].heat_i32, 9_800);
    assert!(pack
        .recent_state
        .iter()
        .all(|item| !item.summary.contains("Older than seven days")));
    assert!(pack.weekly_capsule.is_some());
    assert_eq!(pack.key_relations.len(), 1);
    assert_eq!(pack.key_relations[0].resonance, 91);
    assert_eq!(
        pack.key_relations[0].relation_type,
        RelationKind::PersonProject
    );
    assert!(pack.token_count < 1200);

    let store = VectorStorage::new(
        config.config_dir.join("vectors.db"),
        config.config_dir.join("vectors.usearch"),
    )
    .unwrap();
    let touched = store
        .recall_by_time_range(
            &laputa::searcher::RecallQuery::by_time_range(now - (7 * 24 * 60 * 60), now)
                .with_wing("self"),
        )
        .unwrap();
    assert!(touched.iter().all(|record| record.access_count >= 1));
}

#[test]
#[serial]
fn test_wakepack_generator_gracefully_handles_cold_start_and_missing_capsule() {
    let dir = tempdir().unwrap();
    let config = MempalaceConfig::new(Some(dir.path().to_path_buf()));
    write_identity(&config, "cold-start");

    let generator = WakePackGenerator::new(config);
    let pack = generator.generate(None).unwrap();

    assert_eq!(pack.identity.user_name.as_deref(), Some("cold-start"));
    assert!(pack.recent_state.is_empty());
    assert!(pack.weekly_capsule.is_none());
    assert!(pack.key_relations.is_empty());
    assert!(pack.token_count < 1200);
}

#[tokio::test]
#[serial]
async fn test_searcher_wake_up_returns_json_wakepack() {
    let dir = tempdir().unwrap();
    let config = MempalaceConfig::new(Some(dir.path().to_path_buf()));
    write_identity(&config, "searcher-user");
    write_capsule(&config, "recent weekly rhythm summary");

    let db_path = config.config_dir.join("vectors.db");
    let conn = Connection::open(&db_path).unwrap();
    ensure_memory_schema(&conn).unwrap();
    insert_memory(
        &conn,
        "Recent high-priority note for wakeup",
        "self",
        "journal",
        chrono::Utc::now().timestamp() - 10,
        9_000,
    );
    drop(conn);

    let searcher = Searcher::new(config);
    let wake = searcher.wake_up(Some("self".to_string())).await.unwrap();
    let value: serde_json::Value = serde_json::from_str(&wake).unwrap();

    assert_eq!(value["identity"]["user_name"], "searcher-user");
    assert!(value["recent_state"].as_array().unwrap().len() == 1);
    assert!(value["token_count"].as_u64().unwrap() < 1200);
}
