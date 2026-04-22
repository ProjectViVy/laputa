use std::process::Command;

use chrono::Utc;
use serial_test::serial;
use tempfile::tempdir;

fn run_cli(args: &[&str]) -> std::process::Output {
    Command::new(env!("CARGO_BIN_EXE_laputa"))
        .args(args)
        .output()
        .unwrap()
}

fn stdout_string(output: &std::process::Output) -> String {
    String::from_utf8_lossy(&output.stdout).trim().to_string()
}

fn stderr_string(output: &std::process::Output) -> String {
    String::from_utf8_lossy(&output.stderr).trim().to_string()
}

fn extract_memory_id(stdout: &str) -> i64 {
    stdout
        .lines()
        .find_map(|line| line.strip_prefix("memory_id: "))
        .unwrap()
        .parse::<i64>()
        .unwrap()
}

#[test]
#[serial]
fn test_cli_init_diary_recall_wakeup_and_mark_flow() {
    let dir = tempdir().unwrap();
    let config_dir = dir.path().to_string_lossy().to_string();

    let init = run_cli(&["--config-dir", &config_dir, "init", "--name", "tester"]);
    assert!(init.status.success(), "stderr: {}", stderr_string(&init));
    let init_stdout = stdout_string(&init);
    assert!(init_stdout.contains("initialized: tester"));
    assert!(init_stdout.contains("db_path:"));

    let diary = run_cli(&[
        "--config-dir",
        &config_dir,
        "diary",
        "write",
        "--content",
        "CLI diary memory",
        "--tags",
        "work,cli",
    ]);
    assert!(diary.status.success(), "stderr: {}", stderr_string(&diary));
    let diary_stdout = stdout_string(&diary);
    assert!(diary_stdout.contains("memory_id: "));
    assert!(diary_stdout.contains("tags: work,cli"));
    let memory_id = extract_memory_id(&diary_stdout);

    let today = Utc::now().format("%Y-%m-%d").to_string();
    let range = format!("{today}~{today}");
    let recall = run_cli(&[
        "--config-dir",
        &config_dir,
        "recall",
        "--time-range",
        &range,
    ]);
    assert!(
        recall.status.success(),
        "stderr: {}",
        stderr_string(&recall)
    );
    let recall_stdout = stdout_string(&recall);
    assert!(recall_stdout.contains("results: 1"));
    assert!(recall_stdout.contains("CLI diary memory"));

    let wakeup = run_cli(&["--config-dir", &config_dir, "wakeup"]);
    assert!(
        wakeup.status.success(),
        "stderr: {}",
        stderr_string(&wakeup)
    );
    let wakeup_stdout = stdout_string(&wakeup);
    assert!(wakeup_stdout.contains("\"user_name\":\"tester\""));
    assert!(wakeup_stdout.contains("\"recent_state\""));

    let mark = run_cli(&[
        "--config-dir",
        &config_dir,
        "mark",
        "--id",
        &memory_id.to_string(),
        "--important",
    ]);
    assert!(mark.status.success(), "stderr: {}", stderr_string(&mark));
    let mark_stdout = stdout_string(&mark);
    assert!(mark_stdout.contains(&format!("memory_id: {memory_id}")));
    assert!(mark_stdout.contains("heat_i32: 9000"));
}

#[test]
#[serial]
fn test_cli_diary_write_requires_initialization() {
    let dir = tempdir().unwrap();
    let config_dir = dir.path().to_string_lossy().to_string();

    let output = run_cli(&[
        "--config-dir",
        &config_dir,
        "diary",
        "write",
        "--content",
        "should fail",
    ]);

    assert!(!output.status.success());
    assert!(stderr_string(&output).contains("Laputa is not initialized"));
}

#[test]
#[serial]
fn test_cli_recall_rejects_invalid_time_range() {
    let dir = tempdir().unwrap();
    let config_dir = dir.path().to_string_lossy().to_string();

    let init = run_cli(&["--config-dir", &config_dir, "init", "--name", "tester"]);
    assert!(init.status.success(), "stderr: {}", stderr_string(&init));

    let output = run_cli(&[
        "--config-dir",
        &config_dir,
        "recall",
        "--time-range",
        "2026/04/01",
    ]);

    assert!(!output.status.success());
    assert!(stderr_string(&output).contains("time-range must use"));
}

#[test]
#[serial]
fn test_cli_mark_rejects_invalid_id() {
    let dir = tempdir().unwrap();
    let config_dir = dir.path().to_string_lossy().to_string();

    let init = run_cli(&["--config-dir", &config_dir, "init", "--name", "tester"]);
    assert!(init.status.success(), "stderr: {}", stderr_string(&init));

    let output = run_cli(&[
        "--config-dir",
        &config_dir,
        "mark",
        "--id",
        "not-a-number",
        "--important",
    ]);

    assert!(!output.status.success());
    assert!(stderr_string(&output).contains("invalid memory id"));
}

#[test]
#[serial]
fn test_cli_missing_required_argument_returns_clap_error() {
    let output = run_cli(&["init"]);

    assert!(!output.status.success());
    assert!(stderr_string(&output).contains("--name"));
}
