use chrono::{Duration, TimeZone, Utc};
use laputa::config::MempalaceConfig;
use laputa::export::FullExporter;
use laputa::knowledge_graph::{KnowledgeGraph, RelationKind};
use laputa::rhythm::WeeklyCapsuleGenerator;
use laputa::storage::memory::ensure_memory_schema;
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
            text_content,
            wing,
            room,
            valid_from,
            last_accessed,
            access_count,
            importance_score,
            heat_i32,
            emotion_valence,
            emotion_arousal,
            discard_candidate
         ) VALUES (?1, ?2, ?3, ?4, ?4, 0, 5.0, ?5, 10, 40, 0)",
        params![text_content, wing, room, valid_from, heat_i32],
    )
    .unwrap();
}

fn init_identity(config: &MempalaceConfig, content: &str) {
    std::fs::create_dir_all(&config.config_dir).unwrap();
    std::fs::write(config.config_dir.join("identity.md"), content).unwrap();
}

fn prepare_memory_store(config: &MempalaceConfig) -> Connection {
    let db_path = config.config_dir.join("vectors.db");
    let conn = Connection::open(db_path).unwrap();
    ensure_memory_schema(&conn).unwrap();
    conn
}

fn prepare_knowledge_graph(config: &MempalaceConfig) -> KnowledgeGraph {
    KnowledgeGraph::new(config.config_dir.join("knowledge.db").to_str().unwrap()).unwrap()
}

#[test]
#[serial]
fn test_full_export_creates_documented_bundle_and_prefers_identity_md() {
    let dir = tempdir().unwrap();
    let config = MempalaceConfig::new(Some(dir.path().to_path_buf()));
    config.init().unwrap();
    init_identity(&config, "## L0 - IDENTITY\n\nuser_name: tester\n");
    std::fs::write(
        config.config_dir.join("identity.txt"),
        "wrong legacy identity",
    )
    .unwrap();

    let conn = prepare_memory_store(&config);
    let monday = Utc.with_ymd_and_hms(2026, 4, 6, 0, 0, 0).unwrap();
    for day in 0..7 {
        insert_memory(
            &conn,
            &format!("High heat weekly memory {day} about Rust scheduler and export structure."),
            "self",
            "journal",
            (monday + Duration::days(day)).timestamp(),
            6_000 + day as i32,
        );
    }
    insert_memory(
        &conn,
        "cold memory should stay out of full export",
        "self",
        "journal",
        monday.timestamp(),
        4_900,
    );
    drop(conn);

    let kg = prepare_knowledge_graph(&config);
    kg.upsert_relation(
        "tester",
        "Mira",
        RelationKind::PersonPerson,
        72,
        Some("2026-04-08"),
        None,
        Some("relation.md"),
    )
    .unwrap();

    WeeklyCapsuleGenerator::new(config.clone())
        .generate_for_week(monday + Duration::days(2))
        .unwrap()
        .expect("capsule should be generated");

    let exporter = FullExporter::new(config.clone()).unwrap();
    let result = exporter.export_full(None).unwrap();

    assert!(result.export_dir.exists());
    assert!(result.identity_path.exists());
    assert!(result.relation_path.exists());
    assert_eq!(result.exported_memory_count, 7);
    assert_eq!(result.capsule_count, 1);

    let identity = std::fs::read_to_string(&result.identity_path).unwrap();
    assert!(identity.contains("user_name: tester"));
    assert!(!identity.contains("wrong legacy identity"));

    let relation = std::fs::read_to_string(&result.relation_path).unwrap();
    assert!(relation.contains("tester"));
    assert!(relation.contains("Mira"));

    let manifest: serde_json::Value = serde_json::from_str(
        &std::fs::read_to_string(result.export_dir.join("manifest.json")).unwrap(),
    )
    .unwrap();
    assert_eq!(manifest["memory_count"], 7);
    assert_eq!(manifest["capsule_count"], 1);
    assert_eq!(manifest["capsule_export_status"], "exported");

    let memory_lines = std::fs::read_to_string(
        result
            .export_dir
            .join("memories")
            .join("core-memories.jsonl"),
    )
    .unwrap();
    assert!(memory_lines.contains("High heat weekly memory"));
    assert!(!memory_lines.contains("cold memory"));
}

#[test]
#[serial]
fn test_full_export_writes_capsule_fallback_manifest_and_keeps_primary_data_unchanged() {
    let dir = tempdir().unwrap();
    let config = MempalaceConfig::new(Some(dir.path().to_path_buf()));
    config.init().unwrap();
    init_identity(&config, "## L0 - IDENTITY\n\nuser_name: fallback-user\n");

    let conn = prepare_memory_store(&config);
    insert_memory(&conn, "high heat memory", "self", "journal", 1, 7_200);
    let before_count: i64 = conn
        .query_row("SELECT COUNT(*) FROM memories", [], |row| row.get(0))
        .unwrap();
    drop(conn);

    let exporter = FullExporter::new(config.clone()).unwrap();
    let result = exporter.export_full(None).unwrap();

    let manifest: serde_json::Value = serde_json::from_str(
        &std::fs::read_to_string(result.export_dir.join("manifest.json")).unwrap(),
    )
    .unwrap();
    assert_eq!(manifest["capsule_export_status"], "not_available");
    assert_eq!(manifest["capsule_count"], 0);

    let relation = std::fs::read_to_string(&result.relation_path).unwrap();
    assert!(relation.contains("No active relations exported"));

    let reloaded = MempalaceConfig::new(Some(dir.path().to_path_buf()));
    let full_export_state = reloaded.full_export_state.unwrap();
    assert_eq!(full_export_state.last_export_path, result.export_dir);
    assert_eq!(full_export_state.last_exported_memory_count, 1);

    let conn = Connection::open(config.config_dir.join("vectors.db")).unwrap();
    let after_count: i64 = conn
        .query_row("SELECT COUNT(*) FROM memories", [], |row| row.get(0))
        .unwrap();
    assert_eq!(before_count, after_count);
}

#[test]
#[serial]
fn test_full_export_fails_without_identity_and_does_not_record_fake_state() {
    let dir = tempdir().unwrap();
    let config = MempalaceConfig::new(Some(dir.path().to_path_buf()));
    config.init().unwrap();

    let conn = prepare_memory_store(&config);
    insert_memory(&conn, "high heat memory", "self", "journal", 1, 6_100);
    drop(conn);

    let exporter = FullExporter::new(config).unwrap();
    let error = exporter.export_full(None).unwrap_err();
    assert!(error.to_string().contains("identity.md"));

    let reloaded = MempalaceConfig::new(Some(dir.path().to_path_buf()));
    assert!(reloaded.full_export_state.is_none());
}
