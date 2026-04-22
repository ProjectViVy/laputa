use crate::api::LaputaError;
use crate::config::MempalaceConfig;
use crate::rhythm::WeeklyCapsuleGenerator;
use async_trait::async_trait;
use chrono::{DateTime, Datelike, Duration, Timelike, Utc};
use serde::{Deserialize, Serialize};
use std::fs::{self, OpenOptions};
use std::io::Write;
use std::path::{Path, PathBuf};
use std::sync::atomic::{AtomicBool, Ordering};
use std::sync::{Arc, Mutex};
use tokio::sync::Notify;
use tokio::task::JoinHandle;
use tokio::time::{sleep, Duration as TokioDuration};

const DEFAULT_WEEKLY_SCHEDULE: &str = "0 2 * * 1";
const DEFAULT_MAX_RETRIES: u32 = 3;
const DEFAULT_RETRY_DELAY_SECONDS: u64 = 300;
const POLL_INTERVAL_SECONDS: u64 = 30;

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct RhythmConfig {
    pub weekly_schedule: String,
    pub enabled: bool,
    pub max_retries: u32,
    pub retry_delay_seconds: u64,
}

impl Default for RhythmConfig {
    fn default() -> Self {
        Self {
            weekly_schedule: DEFAULT_WEEKLY_SCHEDULE.to_string(),
            enabled: false,
            max_retries: DEFAULT_MAX_RETRIES,
            retry_delay_seconds: DEFAULT_RETRY_DELAY_SECONDS,
        }
    }
}

impl RhythmConfig {
    pub fn load_from_dir(config_dir: &Path) -> Result<Self, LaputaError> {
        let path = config_dir.join("laputa.toml");
        let content = fs::read_to_string(&path).map_err(|error| {
            LaputaError::ConfigError(format!("failed to read {}: {error}", path.display()))
        })?;

        let mut config = Self::default();
        let mut in_rhythm_section = false;

        for raw_line in content.lines() {
            let line = raw_line.split('#').next().unwrap_or("").trim();
            if line.is_empty() {
                continue;
            }

            if line.starts_with('[') && line.ends_with(']') {
                in_rhythm_section = &line[1..line.len() - 1] == "rhythm";
                continue;
            }

            if !in_rhythm_section {
                continue;
            }

            let Some((key, value)) = line.split_once('=') else {
                continue;
            };
            let key = key.trim();
            let value = value.trim();

            match key {
                "weekly_schedule" => {
                    config.weekly_schedule = parse_string_value(value).ok_or_else(|| {
                        LaputaError::ConfigError(format!(
                            "invalid weekly_schedule in {}",
                            path.display()
                        ))
                    })?;
                }
                "enabled" => {
                    config.enabled = value.parse::<bool>().map_err(|error| {
                        LaputaError::ConfigError(format!(
                            "invalid enabled value in {}: {error}",
                            path.display()
                        ))
                    })?;
                }
                "max_retries" => {
                    config.max_retries = value.parse::<u32>().map_err(|error| {
                        LaputaError::ConfigError(format!(
                            "invalid max_retries value in {}: {error}",
                            path.display()
                        ))
                    })?;
                }
                "retry_delay_seconds" => {
                    config.retry_delay_seconds = value.parse::<u64>().map_err(|error| {
                        LaputaError::ConfigError(format!(
                            "invalid retry_delay_seconds value in {}: {error}",
                            path.display()
                        ))
                    })?;
                }
                _ => {}
            }
        }

        Ok(config)
    }
}

fn parse_string_value(value: &str) -> Option<String> {
    let trimmed = value.trim();
    if trimmed.starts_with('"') && trimmed.ends_with('"') && trimmed.len() >= 2 {
        return Some(trimmed[1..trimmed.len() - 1].to_string());
    }

    None
}

#[derive(Debug, Clone, PartialEq, Eq)]
struct WeeklyCronSchedule {
    minute: u32,
    hour: u32,
    day_of_week: u32,
}

impl WeeklyCronSchedule {
    fn parse(expression: &str) -> Result<Self, LaputaError> {
        let parts = expression.split_whitespace().collect::<Vec<_>>();
        if parts.len() != 5 {
            return Err(LaputaError::ConfigError(format!(
                "cron expression must have 5 fields, got {expression}"
            )));
        }

        let minute = parse_cron_number(parts[0], 0, 59, "minute")?;
        let hour = parse_cron_number(parts[1], 0, 23, "hour")?;
        if parts[2] != "*" || parts[3] != "*" {
            return Err(LaputaError::ConfigError(
                "only wildcard day-of-month and month fields are supported".to_string(),
            ));
        }

        let raw_day_of_week = parse_cron_number(parts[4], 0, 7, "day_of_week")?;
        let day_of_week = if raw_day_of_week == 7 {
            0
        } else {
            raw_day_of_week
        };

        Ok(Self {
            minute,
            hour,
            day_of_week,
        })
    }

    fn scheduled_slot_at(&self, now: DateTime<Utc>) -> Option<DateTime<Utc>> {
        let current_day = now.weekday().num_days_from_sunday() as i64;
        let target_day = self.day_of_week as i64;
        let candidate_day = now.date_naive() - Duration::days(current_day - target_day);
        let candidate_naive = candidate_day.and_hms_opt(self.hour, self.minute, 0)?;
        let candidate = DateTime::<Utc>::from_naive_utc_and_offset(candidate_naive, Utc);

        if now < candidate {
            None
        } else {
            Some(candidate)
        }
    }
}

fn parse_cron_number(
    value: &str,
    min: u32,
    max: u32,
    field_name: &str,
) -> Result<u32, LaputaError> {
    let parsed = value.parse::<u32>().map_err(|error| {
        LaputaError::ConfigError(format!("invalid {field_name} value {value}: {error}"))
    })?;

    if parsed < min || parsed > max {
        return Err(LaputaError::ConfigError(format!(
            "{field_name} value {parsed} is outside {min}..={max}"
        )));
    }

    Ok(parsed)
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
pub struct SchedulerExecutionLog {
    pub timestamp: String,
    pub module: String,
    pub scheduled_for: String,
    pub task_status: String,
    pub result: Option<String>,
    pub error: Option<String>,
    pub duration_ms: u64,
    pub attempts: u32,
}

#[async_trait]
pub trait WeeklyTaskRunner: Send + Sync {
    async fn run(&self, scheduled_for: DateTime<Utc>) -> Result<String, LaputaError>;
}

#[derive(Debug)]
struct CapsuleTaskRunner {
    config_dir: PathBuf,
}

#[async_trait]
impl WeeklyTaskRunner for CapsuleTaskRunner {
    async fn run(&self, scheduled_for: DateTime<Utc>) -> Result<String, LaputaError> {
        let generator =
            WeeklyCapsuleGenerator::new(MempalaceConfig::new(Some(self.config_dir.clone())));
        let capsule = generator
            .generate_for_week(scheduled_for)
            .map_err(map_anyhow_error)?;

        match capsule {
            Some(summary) => Ok(summary.week_id),
            None => Ok("no-capsule".to_string()),
        }
    }
}

fn map_anyhow_error(error: anyhow::Error) -> LaputaError {
    LaputaError::StorageError(error.to_string())
}

pub struct RhythmScheduler {
    config_dir: PathBuf,
    config: RhythmConfig,
    schedule: WeeklyCronSchedule,
    runner: Arc<dyn WeeklyTaskRunner>,
    is_running: Arc<AtomicBool>,
    stop_signal: Arc<Notify>,
    handle: Mutex<Option<JoinHandle<()>>>,
}

impl RhythmScheduler {
    pub fn load(config_dir: impl AsRef<Path>) -> Result<Self, LaputaError> {
        let config_dir = config_dir.as_ref().to_path_buf();
        let config = RhythmConfig::load_from_dir(&config_dir)?;
        Self::build(
            config_dir.clone(),
            config,
            Arc::new(CapsuleTaskRunner { config_dir }),
        )
    }

    pub fn with_runner(
        config_dir: impl AsRef<Path>,
        runner: Arc<dyn WeeklyTaskRunner>,
    ) -> Result<Self, LaputaError> {
        let config_dir = config_dir.as_ref().to_path_buf();
        let config = RhythmConfig::load_from_dir(&config_dir)?;
        Self::build(config_dir, config, runner)
    }

    fn build(
        config_dir: PathBuf,
        config: RhythmConfig,
        runner: Arc<dyn WeeklyTaskRunner>,
    ) -> Result<Self, LaputaError> {
        let schedule = WeeklyCronSchedule::parse(&config.weekly_schedule)?;
        Ok(Self {
            config_dir,
            config,
            schedule,
            runner,
            is_running: Arc::new(AtomicBool::new(false)),
            stop_signal: Arc::new(Notify::new()),
            handle: Mutex::new(None),
        })
    }

    pub fn config(&self) -> &RhythmConfig {
        &self.config
    }

    pub fn scheduled_slot_at(&self, now: DateTime<Utc>) -> Option<DateTime<Utc>> {
        self.schedule.scheduled_slot_at(now)
    }

    pub fn is_running(&self) -> bool {
        self.is_running.load(Ordering::SeqCst)
    }

    pub async fn start(&self) -> Result<(), LaputaError> {
        if !self.config.enabled {
            return Ok(());
        }

        if self.is_running.swap(true, Ordering::SeqCst) {
            return Ok(());
        }

        let config_dir = self.config_dir.clone();
        let config = self.config.clone();
        let schedule = self.schedule.clone();
        let runner = self.runner.clone();
        let is_running = self.is_running.clone();
        let stop_signal = self.stop_signal.clone();

        let handle = tokio::spawn(async move {
            loop {
                if !is_running.load(Ordering::SeqCst) {
                    break;
                }

                let _ = Self::run_pending_internal(
                    &config_dir,
                    &config,
                    &schedule,
                    runner.clone(),
                    Utc::now(),
                )
                .await;

                tokio::select! {
                    _ = stop_signal.notified() => break,
                    _ = sleep(TokioDuration::from_secs(POLL_INTERVAL_SECONDS)) => {}
                }
            }

            is_running.store(false, Ordering::SeqCst);
        });

        let mut guard = self.handle.lock().unwrap();
        *guard = Some(handle);
        Ok(())
    }

    pub async fn stop(&self) -> Result<(), LaputaError> {
        if !self.is_running.swap(false, Ordering::SeqCst) {
            return Ok(());
        }

        self.stop_signal.notify_waiters();
        let handle = self.handle.lock().unwrap().take();
        if let Some(handle) = handle {
            let _ = handle.await;
        }
        Ok(())
    }

    pub async fn run_pending_at(
        &self,
        now: DateTime<Utc>,
    ) -> Result<Option<SchedulerExecutionLog>, LaputaError> {
        Self::run_pending_internal(
            &self.config_dir,
            &self.config,
            &self.schedule,
            self.runner.clone(),
            now,
        )
        .await
    }

    async fn run_pending_internal(
        config_dir: &Path,
        config: &RhythmConfig,
        schedule: &WeeklyCronSchedule,
        runner: Arc<dyn WeeklyTaskRunner>,
        now: DateTime<Utc>,
    ) -> Result<Option<SchedulerExecutionLog>, LaputaError> {
        if !config.enabled {
            return Ok(None);
        }

        let Some(scheduled_for) = schedule.scheduled_slot_at(now) else {
            return Ok(None);
        };

        let paths = SlotPaths::new(config_dir, scheduled_for);
        fs::create_dir_all(&paths.rhythm_dir)?;

        if paths.done_path.exists() {
            return Ok(None);
        }

        let mut lock_file = match OpenOptions::new()
            .write(true)
            .create_new(true)
            .open(&paths.lock_path)
        {
            Ok(file) => file,
            Err(error) if error.kind() == std::io::ErrorKind::AlreadyExists => return Ok(None),
            Err(error) => return Err(LaputaError::StorageError(error.to_string())),
        };
        let _ = writeln!(lock_file, "{}", scheduled_for.to_rfc3339());

        let execution = Self::execute_with_retries(config, runner, scheduled_for).await;
        let serialized = serde_json::to_string(&execution)
            .map_err(|error| LaputaError::StorageError(error.to_string()))?;

        {
            let mut log = OpenOptions::new()
                .create(true)
                .append(true)
                .open(&paths.log_path)?;
            writeln!(log, "{serialized}")?;
        }

        fs::write(&paths.done_path, &serialized)?;
        let _ = fs::remove_file(&paths.lock_path);
        Ok(Some(execution))
    }

    async fn execute_with_retries(
        config: &RhythmConfig,
        runner: Arc<dyn WeeklyTaskRunner>,
        scheduled_for: DateTime<Utc>,
    ) -> SchedulerExecutionLog {
        let started_at = std::time::Instant::now();
        let mut attempts = 0;

        loop {
            attempts += 1;

            match runner.run(scheduled_for).await {
                Ok(result) => {
                    let task_status = if result == "no-capsule" {
                        "skipped"
                    } else {
                        "generated"
                    };

                    return SchedulerExecutionLog {
                        timestamp: Utc::now().to_rfc3339(),
                        module: "rhythm::scheduler".to_string(),
                        scheduled_for: scheduled_for.to_rfc3339(),
                        task_status: task_status.to_string(),
                        result: Some(result),
                        error: None,
                        duration_ms: started_at.elapsed().as_millis() as u64,
                        attempts,
                    };
                }
                Err(error) => {
                    let last_error = error.to_string();
                    if attempts > config.max_retries {
                        return SchedulerExecutionLog {
                            timestamp: Utc::now().to_rfc3339(),
                            module: "rhythm::scheduler".to_string(),
                            scheduled_for: scheduled_for.to_rfc3339(),
                            task_status: "failed".to_string(),
                            result: None,
                            error: Some(last_error),
                            duration_ms: started_at.elapsed().as_millis() as u64,
                            attempts,
                        };
                    }

                    if config.retry_delay_seconds > 0 {
                        sleep(TokioDuration::from_secs(config.retry_delay_seconds)).await;
                    }
                }
            }
        }
    }
}

struct SlotPaths {
    rhythm_dir: PathBuf,
    log_path: PathBuf,
    lock_path: PathBuf,
    done_path: PathBuf,
}

impl SlotPaths {
    fn new(config_dir: &Path, scheduled_for: DateTime<Utc>) -> Self {
        let rhythm_dir = config_dir.join("rhythm");
        let slot_id = format!(
            "{:04}{:02}{:02}T{:02}{:02}{:02}",
            scheduled_for.year(),
            scheduled_for.month(),
            scheduled_for.day(),
            scheduled_for.hour(),
            scheduled_for.minute(),
            scheduled_for.second()
        );

        Self {
            log_path: rhythm_dir.join("scheduler.log"),
            lock_path: rhythm_dir.join(format!("scheduler-{slot_id}.lock")),
            done_path: rhythm_dir.join(format!("scheduler-{slot_id}.done.json")),
            rhythm_dir,
        }
    }
}
