mod fixtures;

use chrono::{Duration, TimeZone, Utc};
use laputa::api::LaputaError;
use laputa::heat::{HeatConfig, HeatService, HeatState};
use laputa::storage::memory::{ensure_memory_schema, LaputaMemoryRecord};
use laputa::vector_storage::VectorStorage;
use rusqlite::{params, Connection};
use serial_test::serial;
use tempfile::tempdir;

fn make_record(
    heat_i32: i32,
    access_count: u32,
    last_accessed: chrono::DateTime<Utc>,
) -> LaputaMemoryRecord {
    let mut record = LaputaMemoryRecord::new(
        7,
        "heat test memory".to_string(),
        "self".to_string(),
        "journal".to_string(),
        Some("heat.md".to_string()),
        last_accessed.timestamp(),
        None,
        0.9,
        5.0,
    );
    record.heat_i32 = heat_i32;
    record.access_count = access_count;
    record.last_accessed = last_accessed;
    record
}

fn expected_heat(base: i32, days: f64, access_count: u32, decay_rate: f64) -> i32 {
    if access_count == 0 {
        return base.clamp(0, 10_000);
    }

    let heat = base as f64 * (-decay_rate * days.max(0.0)).exp() * (access_count as f64 + 1.0).ln();
    (heat.round() as i32).clamp(0, 10_000)
}

fn solve_decay_rate(base: i32, access_count: u32, target: i32) -> f64 {
    let numerator = target as f64;
    let denominator = base as f64 * (access_count as f64 + 1.0).ln();
    -(numerator / denominator).ln()
}

fn insert_heat_memory(
    conn: &Connection,
    text_content: &str,
    valid_from: i64,
    last_accessed: i64,
    access_count: u32,
    heat_i32: i32,
    is_archive_candidate: bool,
) -> i64 {
    conn.execute(
        "INSERT INTO memories (
            text_content, wing, room, valid_from, last_accessed, access_count, heat_i32, is_archive_candidate, discard_candidate
         ) VALUES (?1, 'self', 'journal', ?2, ?3, ?4, ?5, ?6, 0)",
        params![
            text_content,
            valid_from,
            last_accessed,
            i64::from(access_count),
            heat_i32,
            if is_archive_candidate { 1_i64 } else { 0_i64 }
        ],
    )
    .unwrap();
    conn.last_insert_rowid()
}

#[test]
fn test_heat_formula_matches_decay_equation() {
    let now = Utc.with_ymd_and_hms(2026, 4, 15, 0, 0, 0).unwrap();
    let last_accessed = now - Duration::days(3);
    let record = make_record(5_000, 4, last_accessed);
    let service = HeatService::new(HeatConfig::default()).unwrap();

    let calculated = service.calculate_at(&record, now);
    let expected = expected_heat(5_000, 3.0, 4, 0.1);

    assert_eq!(calculated, expected);
}

#[test]
fn test_zero_access_count_preserves_existing_base_heat() {
    let epoch = Utc.timestamp_opt(0, 0).single().unwrap();
    let now = Utc.with_ymd_and_hms(2026, 4, 15, 0, 0, 0).unwrap();
    let record = make_record(5_000, 0, epoch);
    let service = HeatService::new(HeatConfig::default()).unwrap();

    assert_eq!(service.calculate_at(&record, now), 5_000);
}

#[test]
fn test_epoch_access_time_decays_heat_to_lower_bound() {
    let epoch = Utc.timestamp_opt(0, 0).single().unwrap();
    let now = Utc.with_ymd_and_hms(2026, 4, 15, 0, 0, 0).unwrap();
    let record = make_record(5_000, 8, epoch);
    let service = HeatService::new(HeatConfig::default()).unwrap();

    assert_eq!(service.calculate_at(&record, now), 0);
}

#[test]
fn test_heat_is_clamped_between_zero_and_ten_thousand() {
    let now = Utc.with_ymd_and_hms(2026, 4, 15, 0, 0, 0).unwrap();
    let mut time_machine = fixtures::time_machine::TimeMachine::new();
    time_machine.advance_days(1);

    let last_accessed = now - Duration::seconds(time_machine.freeze() as i64);
    let capped_high = make_record(10_000, 100_000, last_accessed);
    let service = HeatService::new(HeatConfig {
        decay_rate: 0.0,
        ..HeatConfig::default()
    })
    .unwrap();

    assert_eq!(service.calculate_at(&capped_high, now), 10_000);
}

#[test]
fn test_heat_state_boundaries_match_story_definition() {
    let service = HeatService::new(HeatConfig::default()).unwrap();

    assert_eq!(service.state_for_heat(8_001).unwrap(), HeatState::Locked);
    assert_eq!(service.state_for_heat(8_000).unwrap(), HeatState::Active);
    assert_eq!(service.state_for_heat(5_000).unwrap(), HeatState::Active);
    assert_eq!(
        service.state_for_heat(4_999).unwrap(),
        HeatState::ArchiveCandidate
    );
    assert_eq!(
        service.state_for_heat(2_000).unwrap(),
        HeatState::ArchiveCandidate
    );
    assert_eq!(
        service.state_for_heat(1_999).unwrap(),
        HeatState::PackCandidate
    );

    assert!(!service.should_archive(8_000).unwrap());
    assert!(service.should_archive(4_999).unwrap());
    assert!(service.should_archive(1_999).unwrap());
}

#[test]
fn test_sm01_locked_heat_decays_into_active_state() {
    let now = Utc.with_ymd_and_hms(2026, 4, 15, 0, 0, 0).unwrap();
    let record = make_record(9_000, 4, now - Duration::days(1));
    let service = HeatService::new(HeatConfig {
        decay_rate: solve_decay_rate(9_000, 4, 8_000),
        ..HeatConfig::default()
    })
    .unwrap();

    let heat = service.calculate_at(&record, now);
    assert_eq!(heat, 8_000);
    assert_eq!(service.state_for_heat(heat).unwrap(), HeatState::Active);
}

#[test]
fn test_sm02_sm03_sm04_threshold_crossings_are_stable() {
    let now = Utc.with_ymd_and_hms(2026, 4, 15, 0, 0, 0).unwrap();

    for (base, target, expected_state) in [
        (8_000, 7_999, HeatState::Active),
        (5_000, 4_999, HeatState::ArchiveCandidate),
        (2_000, 1_999, HeatState::PackCandidate),
    ] {
        let service = HeatService::new(HeatConfig {
            decay_rate: solve_decay_rate(base, 4, target),
            ..HeatConfig::default()
        })
        .unwrap();
        let record = make_record(base, 4, now - Duration::days(1));
        let heat = service.calculate_at(&record, now);

        assert_eq!(heat, target);
        assert_eq!(service.state_for_heat(heat).unwrap(), expected_state);
    }
}

#[test]
fn test_batch_calculation_matches_single_record_results() {
    let now = Utc.with_ymd_and_hms(2026, 4, 15, 0, 0, 0).unwrap();
    let records = [
        make_record(5_000, 0, now),
        make_record(8_000, 4, now - Duration::days(1)),
        make_record(2_500, 2, now - Duration::days(8)),
    ];
    let service = HeatService::new(HeatConfig::default()).unwrap();

    let batch = service.calculate_batch_at(records.iter(), now);
    let single: Vec<i32> = records
        .iter()
        .map(|record| service.calculate_at(record, now))
        .collect();

    assert_eq!(batch, single);
}

#[test]
fn test_heat_config_loads_from_laputa_toml() {
    let dir = tempdir().unwrap();
    std::fs::write(
        dir.path().join("laputa.toml"),
        r#"
[heat]
enabled = true
hot_threshold = 8100
warm_threshold = 5100
cold_threshold = 2100
decay_rate = 0.25
update_interval_hours = 2
"#,
    )
    .unwrap();

    let service = HeatService::load_from_dir(dir.path()).unwrap();
    assert_eq!(service.config().hot_threshold, 8_100);
    assert_eq!(service.config().warm_threshold, 5_100);
    assert_eq!(service.config().cold_threshold, 2_100);
    assert_eq!(service.config().decay_rate, 0.25);
    assert_eq!(service.config().update_interval_hours, 2);
}

#[test]
fn test_invalid_heat_config_returns_structured_errors() {
    let threshold_error = HeatConfig::from_toml_str(
        r#"
[heat]
hot_threshold = 15000
"#,
    )
    .unwrap_err();
    assert!(matches!(
        threshold_error,
        LaputaError::HeatThresholdError(15_000)
    ));

    let order_error = HeatConfig::from_toml_str(
        r#"
[heat]
hot_threshold = 5000
warm_threshold = 5000
cold_threshold = 2000
"#,
    )
    .unwrap_err();
    assert!(matches!(order_error, LaputaError::ConfigError(_)));
}

#[test]
#[serial]
fn test_heat_decay_pass_updates_only_due_candidates_and_marks_archive_state() {
    let dir = tempdir().unwrap();
    let db_path = dir.path().join("vectors.db");
    let index_path = dir.path().join("vectors.usearch");
    let conn = Connection::open(&db_path).unwrap();
    ensure_memory_schema(&conn).unwrap();

    let now = Utc.with_ymd_and_hms(2026, 4, 15, 12, 0, 0).unwrap();
    let due_last_accessed = (now - Duration::hours(3)).timestamp();
    let recent_last_accessed = (now - Duration::minutes(30)).timestamp();

    let due_id = insert_heat_memory(
        &conn,
        "due decay",
        due_last_accessed,
        due_last_accessed,
        1,
        1_500,
        false,
    );
    let recent_id = insert_heat_memory(
        &conn,
        "recent touch",
        recent_last_accessed,
        recent_last_accessed,
        5,
        9_000,
        false,
    );
    drop(conn);

    let store = VectorStorage::new(&db_path, &index_path).unwrap();
    let service = HeatService::new(HeatConfig::default()).unwrap();

    let updated = store.run_heat_decay_pass_at(&service, now).unwrap();
    assert_eq!(updated, 1);

    let due = store.get_memory_by_id(due_id).unwrap();
    assert!(due.heat_i32 < 2_000);
    assert!(due.is_archive_candidate);

    let recent = store.get_memory_by_id(recent_id).unwrap();
    assert_eq!(recent.heat_i32, 9_000);
    assert!(!recent.is_archive_candidate);
}

#[test]
#[serial]
fn test_compare_and_update_heat_preserves_concurrent_touch() {
    let dir = tempdir().unwrap();
    let db_path = dir.path().join("vectors.db");
    let index_path = dir.path().join("vectors.usearch");
    let conn = Connection::open(&db_path).unwrap();
    ensure_memory_schema(&conn).unwrap();

    let now = Utc.with_ymd_and_hms(2026, 4, 15, 12, 0, 0).unwrap();
    let stale_last_accessed = (now - Duration::hours(4)).timestamp();
    let memory_id = insert_heat_memory(
        &conn,
        "concurrent touch",
        stale_last_accessed,
        stale_last_accessed,
        2,
        5_000,
        false,
    );
    drop(conn);

    let store = VectorStorage::new(&db_path, &index_path).unwrap();
    let candidates = store
        .list_decay_candidates((now - Duration::hours(1)).timestamp(), 10)
        .unwrap();
    assert_eq!(candidates.len(), 1);
    let candidate = &candidates[0];

    store.touch_memory(memory_id).unwrap();

    let updated = store
        .update_heat_fields_if_unchanged(
            memory_id,
            candidate.last_accessed.timestamp(),
            candidate.access_count,
            1_234,
            true,
        )
        .unwrap();
    assert!(!updated);

    let record = store.get_memory_by_id(memory_id).unwrap();
    assert_eq!(record.access_count, 3);
    assert_ne!(
        record.last_accessed.timestamp(),
        candidate.last_accessed.timestamp()
    );
    assert_eq!(record.heat_i32, 5_000);
    assert!(!record.is_archive_candidate);
}

#[test]
#[serial]
fn test_update_heat_fields_if_unchanged_rejects_out_of_range_heat() {
    let dir = tempdir().unwrap();
    let db_path = dir.path().join("vectors.db");
    let index_path = dir.path().join("vectors.usearch");
    let conn = Connection::open(&db_path).unwrap();
    ensure_memory_schema(&conn).unwrap();

    let now = Utc.with_ymd_and_hms(2026, 4, 15, 12, 0, 0).unwrap();
    let timestamp = (now - Duration::hours(4)).timestamp();
    let memory_id = insert_heat_memory(&conn, "unchanged", timestamp, timestamp, 2, 5_000, false);
    drop(conn);

    let store = VectorStorage::new(&db_path, &index_path).unwrap();
    let error = store
        .update_heat_fields_if_unchanged(memory_id, timestamp, 2, -1, true)
        .expect_err("out-of-range heat should be rejected");

    let laputa_error = error
        .downcast_ref::<laputa::api::LaputaError>()
        .expect("error should remain LaputaError");
    assert!(matches!(
        laputa_error,
        laputa::api::LaputaError::ValidationError(_)
    ));

    let record = store.get_memory_by_id(memory_id).unwrap();
    assert_eq!(record.heat_i32, 5_000);
    assert!(!record.is_archive_candidate);
}
