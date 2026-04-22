use chrono::{Duration, TimeZone, Utc};
use laputa::archiver::{ArchiveConfig, ArchiveExporter, ArchiveMarker};
use laputa::config::MempalaceConfig;
use laputa::storage::memory::ensure_memory_schema;
use laputa::vector_storage::VectorStorage;
use rusqlite::{params, Connection};
use serial_test::serial;
use tempfile::tempdir;

fn seed_memory(
    conn: &Connection,
    text_content: &str,
    valid_from: i64,
    last_accessed: i64,
    heat_i32: i32,
    is_archive_candidate: bool,
) -> i64 {
    conn.execute(
        "INSERT INTO memories (
            text_content, wing, room, valid_from, last_accessed, heat_i32, is_archive_candidate, discard_candidate
         ) VALUES (?1, 'self', 'journal', ?2, ?3, ?4, ?5, 0)",
        params![
            text_content,
            valid_from,
            last_accessed,
            heat_i32,
            if is_archive_candidate { 1_i64 } else { 0_i64 }
        ],
    )
    .unwrap();
    conn.last_insert_rowid()
}

#[test]
#[serial]
fn test_mark_archive_candidates_respects_threshold_and_is_idempotent() {
    let dir = tempdir().unwrap();
    let db_path = dir.path().join("vectors.db");
    let index_path = dir.path().join("vectors.usearch");
    let conn = Connection::open(&db_path).unwrap();
    ensure_memory_schema(&conn).unwrap();

    let now = Utc.with_ymd_and_hms(2026, 4, 15, 12, 0, 0).unwrap();
    let low = seed_memory(
        &conn,
        "low heat",
        now.timestamp(),
        (now - Duration::days(10)).timestamp(),
        1_999,
        false,
    );
    let boundary = seed_memory(
        &conn,
        "boundary heat",
        now.timestamp(),
        (now - Duration::days(5)).timestamp(),
        2_000,
        false,
    );
    let hot = seed_memory(
        &conn,
        "hot memory",
        now.timestamp(),
        now.timestamp(),
        8_500,
        false,
    );
    drop(conn);

    let store = VectorStorage::new(&db_path, &index_path).unwrap();

    let first = store
        .mark_low_heat_memories_as_archive_candidates(2_000)
        .unwrap();
    assert_eq!(first, vec![low]);

    let second = store
        .mark_low_heat_memories_as_archive_candidates(2_000)
        .unwrap();
    assert!(second.is_empty());

    assert!(store.get_memory_by_id(low).unwrap().is_archive_candidate);
    assert!(
        !store
            .get_memory_by_id(boundary)
            .unwrap()
            .is_archive_candidate
    );
    assert!(!store.get_memory_by_id(hot).unwrap().is_archive_candidate);
}

#[test]
#[serial]
fn test_list_archive_candidates_returns_only_candidates_in_coldest_first_order() {
    let dir = tempdir().unwrap();
    let db_path = dir.path().join("vectors.db");
    let index_path = dir.path().join("vectors.usearch");
    let conn = Connection::open(&db_path).unwrap();
    ensure_memory_schema(&conn).unwrap();

    let now = Utc.with_ymd_and_hms(2026, 4, 15, 12, 0, 0).unwrap();
    let older = seed_memory(
        &conn,
        "older 1500",
        now.timestamp(),
        (now - Duration::days(30)).timestamp(),
        1_500,
        true,
    );
    let newer_same_heat = seed_memory(
        &conn,
        "newer 1500",
        now.timestamp(),
        (now - Duration::days(2)).timestamp(),
        1_500,
        true,
    );
    let warmer = seed_memory(
        &conn,
        "warmer 1800",
        now.timestamp(),
        (now - Duration::days(40)).timestamp(),
        1_800,
        true,
    );
    let _non_candidate = seed_memory(
        &conn,
        "non candidate",
        now.timestamp(),
        now.timestamp(),
        9_000,
        false,
    );
    drop(conn);

    let store = VectorStorage::new(&db_path, &index_path).unwrap();
    let records = store.list_archive_candidates(10).unwrap();

    let ids: Vec<i64> = records.iter().map(|record| record.id).collect();
    assert_eq!(ids, vec![older, newer_same_heat, warmer]);
    assert!(records.iter().all(|record| record.is_archive_candidate));
}

#[test]
#[serial]
fn test_archive_marker_loads_config_runs_daily_check_and_does_not_delete_records() {
    let dir = tempdir().unwrap();
    std::fs::write(
        dir.path().join("laputa.toml"),
        r#"
[archive]
enabled = true
archive_threshold = 2100
check_interval_days = 3
"#,
    )
    .unwrap();

    let db_path = dir.path().join("vectors.db");
    let index_path = dir.path().join("vectors.usearch");
    let conn = Connection::open(&db_path).unwrap();
    ensure_memory_schema(&conn).unwrap();

    let now = Utc.with_ymd_and_hms(2026, 4, 15, 12, 0, 0).unwrap();
    let low = seed_memory(
        &conn,
        "archive me",
        now.timestamp(),
        (now - Duration::days(7)).timestamp(),
        2_099,
        false,
    );
    let keep = seed_memory(
        &conn,
        "keep me",
        now.timestamp(),
        now.timestamp(),
        2_100,
        false,
    );
    let original_count: i64 = conn
        .query_row("SELECT COUNT(*) FROM memories", [], |row| row.get(0))
        .unwrap();
    drop(conn);

    let store = VectorStorage::new(&db_path, &index_path).unwrap();
    let marker = ArchiveMarker::load_from_dir(&store, dir.path()).unwrap();

    assert_eq!(marker.config().archive_threshold, 2_100);
    assert_eq!(marker.config().check_interval_days, 3);

    let updated = marker.run_daily_check().unwrap();
    assert_eq!(updated, 1);

    let listed = marker.list_candidates(10).unwrap();
    assert_eq!(listed.len(), 1);
    assert_eq!(listed[0].id, low);
    assert_eq!(listed[0].text_content, "archive me");
    assert_eq!(listed[0].heat_i32, 2_099);
    assert_eq!(
        listed[0].last_accessed,
        Utc.timestamp_opt((now - Duration::days(7)).timestamp(), 0)
            .single()
            .unwrap()
    );

    let low_record = store.get_memory_by_id(low).unwrap();
    let keep_record = store.get_memory_by_id(keep).unwrap();
    assert!(low_record.is_archive_candidate);
    assert!(!keep_record.is_archive_candidate);

    let final_count = store.memory_count().unwrap() as i64;
    assert_eq!(final_count, original_count);
}

#[test]
#[serial]
fn test_archive_exporter_exports_only_candidates_and_records_runtime_metadata() {
    let dir = tempdir().unwrap();
    let config = MempalaceConfig::new(Some(dir.path().to_path_buf()));
    config.init().unwrap();

    let db_path = dir.path().join("vectors.db");
    let index_path = dir.path().join("vectors.usearch");
    let conn = Connection::open(&db_path).unwrap();
    ensure_memory_schema(&conn).unwrap();

    let now = Utc.with_ymd_and_hms(2026, 4, 15, 12, 0, 0).unwrap();
    let coldest = seed_memory(
        &conn,
        "coldest candidate",
        now.timestamp(),
        (now - Duration::days(30)).timestamp(),
        1_200,
        true,
    );
    let warmer = seed_memory(
        &conn,
        "warmer candidate",
        now.timestamp(),
        (now - Duration::days(5)).timestamp(),
        1_600,
        true,
    );
    let keep = seed_memory(
        &conn,
        "keep in primary",
        now.timestamp(),
        now.timestamp(),
        8_500,
        false,
    );
    drop(conn);

    let store = VectorStorage::new(&db_path, &index_path).unwrap();
    let exporter = ArchiveExporter::new(&store, config.clone(), ArchiveConfig::default()).unwrap();

    let result = exporter.export_candidates(None).unwrap();
    assert_eq!(result.exported_count, 2);
    assert!(result.export_path.exists());
    assert!(result.exported_at > 0);

    let exported = Connection::open(&result.export_path).unwrap();
    let exported_ids: Vec<i64> = exported
        .prepare("SELECT id FROM memories ORDER BY heat_i32 ASC, last_accessed ASC, id ASC")
        .unwrap()
        .query_map([], |row| row.get(0))
        .unwrap()
        .collect::<rusqlite::Result<Vec<_>>>()
        .unwrap();
    assert_eq!(exported_ids, vec![coldest, warmer]);

    let exported_count: i64 = exported
        .query_row("SELECT COUNT(*) FROM memories", [], |row| row.get(0))
        .unwrap();
    assert_eq!(exported_count, 2);

    let metadata_path: String = exported
        .query_row(
            "SELECT value FROM archive_export_metadata WHERE key = 'source_db_path'",
            [],
            |row| row.get(0),
        )
        .unwrap();
    assert_eq!(metadata_path, db_path.to_string_lossy());

    let updated_config = MempalaceConfig::new(Some(dir.path().to_path_buf()));
    let archive_state = updated_config.archive_state.unwrap();
    assert_eq!(archive_state.last_export_path, result.export_path);
    assert_eq!(archive_state.last_exported_count, 2);

    let source_count: i64 = store
        .db
        .query_row("SELECT COUNT(*) FROM memories", [], |row| row.get(0))
        .unwrap();
    assert_eq!(source_count, 3);
    assert!(
        store
            .get_memory_by_id(coldest)
            .unwrap()
            .is_archive_candidate
    );
    assert!(store.get_memory_by_id(warmer).unwrap().is_archive_candidate);
    assert!(!store.get_memory_by_id(keep).unwrap().is_archive_candidate);
}

#[test]
#[serial]
fn test_archive_exporter_rejects_empty_exports_and_keeps_runtime_metadata_clean() {
    let dir = tempdir().unwrap();
    let config = MempalaceConfig::new(Some(dir.path().to_path_buf()));
    config.init().unwrap();

    let db_path = dir.path().join("vectors.db");
    let index_path = dir.path().join("vectors.usearch");
    let conn = Connection::open(&db_path).unwrap();
    ensure_memory_schema(&conn).unwrap();
    seed_memory(&conn, "not a candidate", 1, 1, 8_000, false);
    drop(conn);

    let store = VectorStorage::new(&db_path, &index_path).unwrap();
    let exporter = ArchiveExporter::new(&store, config, ArchiveConfig::default()).unwrap();
    let error = exporter.export_candidates(None).unwrap_err();
    assert!(error.to_string().contains("no archive candidates"));

    let reloaded = MempalaceConfig::new(Some(dir.path().to_path_buf()));
    assert!(reloaded.archive_state.is_none());

    let exports_dir = dir.path().join("archives");
    assert!(!exports_dir.exists());
}

#[test]
#[serial]
fn test_archive_exporter_allows_repeat_exports_without_mutating_primary_db() {
    let dir = tempdir().unwrap();
    let config = MempalaceConfig::new(Some(dir.path().to_path_buf()));
    config.init().unwrap();

    let db_path = dir.path().join("vectors.db");
    let index_path = dir.path().join("vectors.usearch");
    let conn = Connection::open(&db_path).unwrap();
    ensure_memory_schema(&conn).unwrap();

    let now = Utc.with_ymd_and_hms(2026, 4, 15, 12, 0, 0).unwrap();
    let candidate = seed_memory(
        &conn,
        "candidate",
        now.timestamp(),
        (now - Duration::days(1)).timestamp(),
        1_400,
        true,
    );
    drop(conn);

    let store = VectorStorage::new(&db_path, &index_path).unwrap();
    let exporter = ArchiveExporter::new(&store, config, ArchiveConfig::default()).unwrap();

    let first = exporter.export_candidates(None).unwrap();
    std::thread::sleep(std::time::Duration::from_secs(1));
    let second = exporter.export_candidates(None).unwrap();

    assert_ne!(first.export_path, second.export_path);
    assert!(first.export_path.exists());
    assert!(second.export_path.exists());
    assert!(
        store
            .get_memory_by_id(candidate)
            .unwrap()
            .is_archive_candidate
    );

    let source_count: i64 = store
        .db
        .query_row("SELECT COUNT(*) FROM memories", [], |row| row.get(0))
        .unwrap();
    assert_eq!(source_count, 1);
}

#[test]
fn test_archive_config_rejects_invalid_values() {
    let err = ArchiveConfig::from_toml_str(
        r#"
[archive]
archive_threshold = 12000
"#,
    )
    .unwrap_err();
    assert!(err.to_string().contains("archive_threshold"));

    let err = ArchiveConfig::from_toml_str(
        r#"
[archive]
check_interval_days = 0
"#,
    )
    .unwrap_err();
    assert!(err.to_string().contains("check_interval_days"));
}
