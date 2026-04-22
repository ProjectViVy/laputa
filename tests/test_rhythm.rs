use chrono::{TimeZone, Utc};
use laputa::api::LaputaError;
use laputa::config::MempalaceConfig;
use laputa::knowledge_graph::{KnowledgeGraph, RelationKind};
use laputa::rhythm::{
    load_latest_capsule, RhythmScheduler, SchedulerExecutionLog, WeeklyCapsuleGenerator,
    WeeklyTaskRunner,
};
use laputa::storage::memory::ensure_memory_schema;
use rusqlite::{params, Connection};
use serial_test::serial;
use std::fs;
use std::path::Path;
use std::sync::atomic::{AtomicUsize, Ordering};
use std::sync::Arc;
use tempfile::tempdir;

struct MockWeeklyTaskRunner {
    invocations: Arc<AtomicUsize>,
    failures_before_success: usize,
}

#[async_trait::async_trait]
impl WeeklyTaskRunner for MockWeeklyTaskRunner {
    async fn run(&self, _scheduled_for: chrono::DateTime<Utc>) -> Result<String, LaputaError> {
        let attempt = self.invocations.fetch_add(1, Ordering::SeqCst);
        if attempt < self.failures_before_success {
            return Err(LaputaError::StorageError(format!(
                "mock failure on attempt {}",
                attempt + 1
            )));
        }

        Ok("weekly-capsule-generated".to_string())
    }
}

fn insert_memory(
    conn: &Connection,
    text_content: &str,
    wing: &str,
    room: &str,
    valid_from: i64,
    heat_i32: i32,
    emotion_valence: i32,
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
         ) VALUES (?1, ?2, ?3, ?4, ?4, 0, 5.0, ?5, ?6, 40, 0)",
        params![
            text_content,
            wing,
            room,
            valid_from,
            heat_i32,
            emotion_valence
        ],
    )
    .unwrap();
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

fn write_scheduler_config(config_dir: &Path, body: &str) {
    fs::create_dir_all(config_dir).unwrap();
    fs::write(config_dir.join("laputa.toml"), body).unwrap();
}

#[test]
#[serial]
fn test_weekly_capsule_generator_persists_capsule_with_keywords_relations_and_aaak() {
    let dir = tempdir().unwrap();
    let config = MempalaceConfig::new(Some(dir.path().to_path_buf()));
    let conn = prepare_memory_store(&config);

    let monday = Utc.with_ymd_and_hms(2026, 4, 6, 0, 0, 0).unwrap();
    let weekly_times = [
        monday + chrono::Duration::hours(2),
        monday + chrono::Duration::hours(10),
        monday + chrono::Duration::days(1),
        monday + chrono::Duration::days(2),
        monday + chrono::Duration::days(3),
        monday + chrono::Duration::days(4),
        monday + chrono::Duration::days(5),
        monday + chrono::Duration::days(6),
    ];

    let texts = [
        "Rust rhythm scheduler delivered a stable weekly capsule pipeline with retrieval focus and Rust ownership notes repeated for clarity.",
        "Rust retrieval improvements kept weekly scheduler quality high while capsule generation tracked relation changes and focus recovery.",
        "A strong collaboration with Mira deepened after shipping the capsule summarizer and Rust search integration for weekly review.",
        "Scheduler tuning reduced noise and made the weekly capsule output easier to trust during retrieval debugging and planning.",
        "Rust implementation notes highlighted retrieval quality, capsule summarization, and scheduler verification in one focused review.",
        "A high energy checkpoint documented the weekly capsule launch, Rust reliability work, and follow-up retrieval decisions.",
        "The team reviewed capsule keywords, relation resonance shifts, and Rust scheduler behavior across the weekly memory set.",
        "Final weekly reflection confirmed the capsule flow, scheduler discipline, Rust retrieval gains, and clearer collaboration signals.",
    ];

    for (index, timestamp) in weekly_times.iter().enumerate() {
        insert_memory(
            &conn,
            texts[index],
            "self",
            if index % 2 == 0 { "journal" } else { "project" },
            timestamp.timestamp(),
            5_200 + (index as i32 * 350),
            10 + index as i32,
        );
    }

    insert_memory(
        &conn,
        "Older memory that should not appear in this week's capsule.",
        "self",
        "journal",
        (monday - chrono::Duration::days(10)).timestamp(),
        9_900,
        0,
    );
    drop(conn);

    let kg = prepare_knowledge_graph(&config);
    kg.upsert_relation(
        "tester",
        "Mira",
        RelationKind::PersonPerson,
        55,
        Some("2026-03-28"),
        None,
        Some("before-week.md"),
    )
    .unwrap();
    kg.upsert_relation(
        "tester",
        "Mira",
        RelationKind::PersonPerson,
        78,
        Some("2026-04-08"),
        None,
        Some("during-week.md"),
    )
    .unwrap();

    let generator = WeeklyCapsuleGenerator::new(config.clone());
    let capsule = generator
        .generate_for_week(monday + chrono::Duration::days(2))
        .unwrap();
    let capsule = capsule.expect("capsule should be generated");

    assert_eq!(capsule.week_id, "2026-W15");
    assert!(!capsule.incomplete);
    assert!(capsule
        .keywords
        .iter()
        .any(|keyword| keyword.contains("rust")));
    assert!(capsule
        .keywords
        .iter()
        .any(|keyword| keyword.contains("scheduler")));
    assert!(capsule.hot_events.len() >= 3);
    assert!(capsule
        .hot_events
        .iter()
        .all(|event| event.heat_i32 > 5_000));
    assert_eq!(capsule.relation_changes.len(), 1);
    assert!(capsule.relation_changes[0].delta > 10);
    assert!(capsule.compression_ratio >= 10.0);
    assert!(capsule.compressed_content.starts_with("V:3.2"));

    let stored = load_latest_capsule(&config.config_dir).expect("stored capsule should load");
    assert!(stored.content.contains("2026-W15"));
    assert!(stored.content.contains("rust"));

    let conn = Connection::open(config.config_dir.join("vectors.db")).unwrap();
    let capsule_rows: i64 = conn
        .query_row("SELECT COUNT(*) FROM capsules", [], |row| row.get(0))
        .unwrap();
    assert_eq!(capsule_rows, 1);
}

#[test]
#[serial]
fn test_weekly_capsule_generator_marks_incomplete_and_skips_empty_weeks() {
    let dir = tempdir().unwrap();
    let config = MempalaceConfig::new(Some(dir.path().to_path_buf()));
    let conn = prepare_memory_store(&config);

    let monday = Utc.with_ymd_and_hms(2026, 4, 13, 0, 0, 0).unwrap();
    for day in 0..6 {
        insert_memory(
            &conn,
            &format!(
                "Weekly note {day} about scheduler rhythm, capsule planning, and calm review signals."
            ),
            "self",
            "journal",
            (monday + chrono::Duration::days(day)).timestamp(),
            5_400 + (day as i32 * 50),
            5,
        );
    }
    drop(conn);

    let generator = WeeklyCapsuleGenerator::new(config.clone());
    let incomplete = generator
        .generate_for_week(monday + chrono::Duration::days(1))
        .unwrap();
    let incomplete = incomplete.expect("incomplete capsule should still be generated");
    assert!(incomplete.incomplete);
    assert_eq!(incomplete.source_record_count, 6);

    let empty = generator
        .generate_for_week(monday + chrono::Duration::days(14))
        .unwrap();
    assert!(empty.is_none(), "empty weeks should not create capsules");
}

#[test]
#[serial]
fn test_rhythm_scheduler_loads_cron_config_and_matches_monday_slot() {
    let dir = tempdir().unwrap();
    write_scheduler_config(
        dir.path(),
        r#"
[rhythm]
weekly_schedule = "0 2 * * 1"
enabled = true
max_retries = 2
retry_delay_seconds = 0
"#,
    );

    let scheduler = RhythmScheduler::load(dir.path()).unwrap();
    let due_time = Utc.with_ymd_and_hms(2026, 4, 13, 2, 0, 0).unwrap();
    let before_due = Utc.with_ymd_and_hms(2026, 4, 13, 1, 59, 0).unwrap();

    assert!(scheduler.config().enabled);
    assert_eq!(scheduler.config().weekly_schedule, "0 2 * * 1");
    assert_eq!(
        scheduler.scheduled_slot_at(due_time).unwrap(),
        due_time,
        "the scheduler should align the cron slot to Monday 02:00 UTC"
    );
    assert!(
        scheduler.scheduled_slot_at(before_due).is_none(),
        "the schedule must not fire before the configured weekly point"
    );
}

#[test]
#[serial]
fn test_rhythm_scheduler_runs_once_and_writes_success_log() {
    let runtime = tokio::runtime::Runtime::new().unwrap();
    runtime.block_on(async {
        let dir = tempdir().unwrap();
        write_scheduler_config(
            dir.path(),
            r#"
[rhythm]
weekly_schedule = "0 2 * * 1"
enabled = true
max_retries = 1
retry_delay_seconds = 0
"#,
        );

        let invocations = Arc::new(AtomicUsize::new(0));
        let runner = Arc::new(MockWeeklyTaskRunner {
            invocations: invocations.clone(),
            failures_before_success: 0,
        });
        let scheduler = RhythmScheduler::with_runner(dir.path(), runner).unwrap();
        let due_time = Utc.with_ymd_and_hms(2026, 4, 13, 2, 0, 0).unwrap();

        let execution = scheduler.run_pending_at(due_time).await.unwrap();
        let execution = execution.expect("scheduler should execute at the due slot");

        assert_eq!(invocations.load(Ordering::SeqCst), 1);
        assert_eq!(execution.task_status, "generated");
        assert!(execution.error.is_none());

        let second = scheduler.run_pending_at(due_time).await.unwrap();
        assert!(
            second.is_none(),
            "the same scheduler must not execute the same weekly slot twice"
        );

        let log_contents = fs::read_to_string(dir.path().join("rhythm").join("scheduler.log"))
            .expect("scheduler log should be created");
        assert!(log_contents.contains("\"task_status\":\"generated\""));
        assert!(log_contents.contains("\"error\":null"));
    });
}

#[test]
#[serial]
fn test_rhythm_scheduler_retries_failures_and_records_error_field() {
    let runtime = tokio::runtime::Runtime::new().unwrap();
    runtime.block_on(async {
        let dir = tempdir().unwrap();
        write_scheduler_config(
            dir.path(),
            r#"
[rhythm]
weekly_schedule = "0 2 * * 1"
enabled = true
max_retries = 2
retry_delay_seconds = 0
"#,
        );

        let invocations = Arc::new(AtomicUsize::new(0));
        let runner = Arc::new(MockWeeklyTaskRunner {
            invocations: invocations.clone(),
            failures_before_success: usize::MAX,
        });
        let scheduler = RhythmScheduler::with_runner(dir.path(), runner).unwrap();
        let due_time = Utc.with_ymd_and_hms(2026, 4, 13, 2, 0, 0).unwrap();

        let execution = scheduler.run_pending_at(due_time).await.unwrap();
        let execution = execution.expect("a failed slot should still produce an execution log");

        assert_eq!(invocations.load(Ordering::SeqCst), 3);
        assert_eq!(execution.task_status, "failed");
        assert!(execution.error.as_deref().unwrap().contains("mock failure"));
        assert!(execution.attempts >= 3);

        let persisted: SchedulerExecutionLog = serde_json::from_str(
            fs::read_to_string(dir.path().join("rhythm").join("scheduler.log"))
                .unwrap()
                .lines()
                .last()
                .unwrap(),
        )
        .unwrap();
        assert_eq!(persisted.task_status, "failed");
        assert!(persisted.error.unwrap().contains("mock failure"));
    });
}

#[test]
#[serial]
fn test_rhythm_scheduler_start_stop_and_prevent_duplicate_multi_instance_runs() {
    let runtime = tokio::runtime::Runtime::new().unwrap();
    runtime.block_on(async {
        let dir = tempdir().unwrap();
        write_scheduler_config(
            dir.path(),
            r#"
[rhythm]
weekly_schedule = "0 2 * * 1"
enabled = true
max_retries = 0
retry_delay_seconds = 0
"#,
        );

        let invocations = Arc::new(AtomicUsize::new(0));
        let runner = Arc::new(MockWeeklyTaskRunner {
            invocations: invocations.clone(),
            failures_before_success: 0,
        });
        let first = RhythmScheduler::with_runner(dir.path(), runner.clone()).unwrap();
        let second = RhythmScheduler::with_runner(dir.path(), runner).unwrap();

        first.start().await.unwrap();
        assert!(first.is_running());
        first.stop().await.unwrap();
        assert!(!first.is_running());

        let due_time = Utc.with_ymd_and_hms(2026, 4, 13, 2, 0, 0).unwrap();
        let first_result = first.run_pending_at(due_time).await.unwrap();
        let second_result = second.run_pending_at(due_time).await.unwrap();

        assert_eq!(
            invocations.load(Ordering::SeqCst),
            1,
            "shared slot markers must prevent duplicate execution across instances"
        );
        assert!(
            first_result.is_some() ^ second_result.is_some(),
            "exactly one scheduler instance should claim the weekly slot"
        );
    });
}
