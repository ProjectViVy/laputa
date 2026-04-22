use std::fs;
use std::path::{Path, PathBuf};

use anyhow::Error;
use chrono::{Datelike, NaiveDate, TimeZone, Utc};
use tokio::runtime::Builder;
use uuid::Uuid;

use crate::api::LaputaError;
use crate::cli::commands::{
    Cli, Commands, DiaryCommand, DiarySubcommands, InitCommand, MarkCommand, RecallCommand,
    WakeupCommand,
};
use crate::cli::output;
use crate::config::MempalaceConfig;
use crate::diary::{Diary, DiaryWriteRequest};
use crate::identity::IdentityInitializer;
use crate::searcher::{RecallQuery, Searcher};
use crate::vector_storage::{UserIntervention, VectorStorage};

const MIN_RECALL_LIMIT: usize = 1;
const MAX_RECALL_LIMIT: usize = 10_000;
const MIN_ALLOWED_DATE_YEAR: i32 = 1900;
const MAX_ALLOWED_DATE_YEAR: i32 = 2100;
const MAX_TIME_RANGE_DAYS: i64 = 365;

pub fn run(cli: Cli) -> Result<String, LaputaError> {
    let runtime = Builder::new_current_thread()
        .enable_all()
        .build()
        .map_err(|error| LaputaError::ConfigError(error.to_string()))?;
    runtime.block_on(run_async(cli))
}

async fn run_async(cli: Cli) -> Result<String, LaputaError> {
    match cli.command {
        Commands::Init(command) => handle_init(cli.config_dir, command),
        Commands::Diary(command) => handle_diary(cli.config_dir, command),
        Commands::Recall(command) => handle_recall(cli.config_dir, command).await,
        Commands::Wakeup(command) => handle_wakeup(cli.config_dir, command).await,
        Commands::Mark(command) => handle_mark(cli.config_dir, command),
    }
}

fn handle_init(config_dir: Option<PathBuf>, command: InitCommand) -> Result<String, LaputaError> {
    let trimmed_name = command.name.trim();
    if trimmed_name.is_empty() {
        return Err(LaputaError::ValidationError(
            "user_name must not be blank".to_string(),
        ));
    }

    let config = MempalaceConfig::new(config_dir);
    let initializer = IdentityInitializer::new(&config.config_dir);
    let db_path = initializer.initialize(trimmed_name)?;
    Ok(output::render_init_success(&command.name, &db_path))
}

fn handle_diary(config_dir: Option<PathBuf>, command: DiaryCommand) -> Result<String, LaputaError> {
    let config = MempalaceConfig::new(config_dir);
    ensure_initialized(&config.config_dir)?;

    match command.command {
        DiarySubcommands::Write(write) => {
            let agent = load_user_name(&config.config_dir)?;
            let tags = parse_tags(write.tags);
            let diary =
                Diary::new(config.config_dir.join("vectors.db")).map_err(map_anyhow_error)?;
            let request = DiaryWriteRequest {
                agent,
                content: write.content,
                tags: tags.clone(),
                emotion: write.emotion,
                timestamp: None,
                wing: write.wing,
                room: write.room,
            };
            let memory_id = diary.write(request).map_err(map_anyhow_error)?;
            Ok(output::render_diary_write_success(memory_id, &tags))
        }
    }
}

async fn handle_recall(
    config_dir: Option<PathBuf>,
    command: RecallCommand,
) -> Result<String, LaputaError> {
    let config = MempalaceConfig::new(config_dir);
    ensure_initialized(&config.config_dir)?;

    let (start, end) = parse_time_range(&command.time_range)?;
    let query = RecallQuery::by_time_range(start, end)
        .with_limit(normalize_recall_limit(command.limit))
        .include_discarded(command.include_discarded);
    let query = if let Some(wing) = command.wing.clone() {
        query.with_wing(wing)
    } else {
        query
    };
    let query = if let Some(room) = command.room.clone() {
        query.with_room(room)
    } else {
        query
    };

    let searcher = Searcher::new(config);
    let records = searcher
        .recall_by_time_range(query)
        .await
        .map_err(map_anyhow_error)?;
    Ok(output::render_recall_results(
        &command.time_range,
        records.as_slice(),
    ))
}

async fn handle_wakeup(
    config_dir: Option<PathBuf>,
    command: WakeupCommand,
) -> Result<String, LaputaError> {
    let config = MempalaceConfig::new(config_dir);
    ensure_initialized(&config.config_dir)?;
    let searcher = Searcher::new(config);
    searcher
        .wake_up(command.wing)
        .await
        .map_err(map_anyhow_error)
}

fn handle_mark(config_dir: Option<PathBuf>, command: MarkCommand) -> Result<String, LaputaError> {
    let config = MempalaceConfig::new(config_dir);
    ensure_initialized(&config.config_dir)?;
    let memory_id = parse_memory_id(&command.id)?;
    let storage = VectorStorage::new(
        config.config_dir.join("vectors.db"),
        config.config_dir.join("vectors.usearch"),
    )
    .map_err(map_anyhow_error)?;

    let intervention = if command.important {
        UserIntervention::Important {
            reason: command
                .reason
                .unwrap_or_else(|| "marked important via CLI".to_string()),
        }
    } else if command.forget {
        UserIntervention::Forget {
            reason: command
                .reason
                .unwrap_or_else(|| "marked for forget via CLI".to_string()),
        }
    } else {
        UserIntervention::EmotionAnchor {
            valence: command.valence.ok_or_else(|| {
                LaputaError::ValidationError(
                    "--emotion-anchor requires --valence and --arousal".to_string(),
                )
            })?,
            arousal: command.arousal.ok_or_else(|| {
                LaputaError::ValidationError(
                    "--emotion-anchor requires --valence and --arousal".to_string(),
                )
            })?,
            reason: command
                .reason
                .unwrap_or_else(|| "emotion anchored via CLI".to_string()),
        }
    };

    let updated = storage
        .apply_intervention(memory_id, intervention)
        .map_err(map_anyhow_error)?;
    Ok(output::render_mark_success(&updated))
}

fn ensure_initialized(config_dir: &Path) -> Result<(), LaputaError> {
    let identity_path = config_dir.join("identity.md");
    if identity_path.exists() {
        return Ok(());
    }

    Err(LaputaError::ConfigError(format!(
        "Laputa is not initialized in {}. Run `laputa init --name <NAME>` first.",
        config_dir.display()
    )))
}

fn load_user_name(config_dir: &Path) -> Result<String, LaputaError> {
    let identity_path = config_dir.join("identity.md");
    let content = fs::read_to_string(&identity_path).map_err(LaputaError::from)?;

    for line in content.lines() {
        if let Some(value) = line.strip_prefix("user_name:") {
            let user_name = value.trim().to_string();
            if !user_name.is_empty() {
                return Ok(user_name);
            }
        }
    }

    Err(LaputaError::ConfigError(format!(
        "identity.md is missing user_name in {}",
        identity_path.display()
    )))
}

fn parse_tags(raw: Option<String>) -> Vec<String> {
    raw.unwrap_or_default()
        .split(',')
        .map(str::trim)
        .filter(|tag| !tag.is_empty())
        .map(ToOwned::to_owned)
        .collect()
}

fn parse_time_range(raw: &str) -> Result<(i64, i64), LaputaError> {
    let (start_raw, end_raw) = raw.split_once('~').ok_or_else(|| {
        LaputaError::ValidationError(format!(
            "time-range must use `YYYY-MM-DD~YYYY-MM-DD`, got `{raw}`"
        ))
    })?;

    let start_date = NaiveDate::parse_from_str(start_raw.trim(), "%Y-%m-%d").map_err(|_| {
        LaputaError::ValidationError(format!(
            "invalid start date in time-range `{raw}`; expected YYYY-MM-DD"
        ))
    })?;
    let end_date = NaiveDate::parse_from_str(end_raw.trim(), "%Y-%m-%d").map_err(|_| {
        LaputaError::ValidationError(format!(
            "invalid end date in time-range `{raw}`; expected YYYY-MM-DD"
        ))
    })?;

    validate_time_range_dates(start_date, end_date, raw)?;

    let start = Utc
        .from_utc_datetime(
            &start_date
                .and_hms_opt(0, 0, 0)
                .ok_or_else(|| LaputaError::ValidationError("invalid start date".to_string()))?,
        )
        .timestamp();
    let end = Utc
        .from_utc_datetime(
            &end_date
                .and_hms_opt(23, 59, 59)
                .ok_or_else(|| LaputaError::ValidationError("invalid end date".to_string()))?,
        )
        .timestamp();

    if start > end {
        return Err(LaputaError::ValidationError(format!(
            "start date must be before or equal to end date in `{raw}`"
        )));
    }

    Ok((start, end))
}

fn parse_memory_id(raw: &str) -> Result<i64, LaputaError> {
    let trimmed = raw.trim();
    if let Ok(id) = trimmed.parse::<i64>() {
        if id <= 0 {
            return Err(LaputaError::ValidationError(format!(
                "invalid memory id `{trimmed}`; expected positive numeric memory_id"
            )));
        }
        return Ok(id);
    }

    if Uuid::parse_str(trimmed).is_ok() {
        return Err(LaputaError::ValidationError(
            "Phase 1 CLI currently accepts numeric memory_id values; UUID support is not wired yet."
                .to_string(),
        ));
    }

    Err(LaputaError::ValidationError(format!(
        "invalid memory id `{trimmed}`; expected numeric memory_id"
    )))
}

fn normalize_recall_limit(limit: usize) -> usize {
    limit.clamp(MIN_RECALL_LIMIT, MAX_RECALL_LIMIT)
}

fn validate_time_range_dates(
    start_date: NaiveDate,
    end_date: NaiveDate,
    raw: &str,
) -> Result<(), LaputaError> {
    validate_date_year(start_date, "start", raw)?;
    validate_date_year(end_date, "end", raw)?;

    let span_days = end_date.signed_duration_since(start_date).num_days();
    if span_days > MAX_TIME_RANGE_DAYS {
        return Err(LaputaError::ValidationError(format!(
            "time-range `{raw}` exceeds the maximum span of {MAX_TIME_RANGE_DAYS} days"
        )));
    }

    Ok(())
}

fn validate_date_year(date: NaiveDate, label: &str, raw: &str) -> Result<(), LaputaError> {
    if !(MIN_ALLOWED_DATE_YEAR..=MAX_ALLOWED_DATE_YEAR).contains(&date.year()) {
        return Err(LaputaError::ValidationError(format!(
            "{label} date in time-range `{raw}` must stay within {MIN_ALLOWED_DATE_YEAR:04}-01-01 and {MAX_ALLOWED_DATE_YEAR:04}-12-31"
        )));
    }

    Ok(())
}

fn map_anyhow_error(error: Error) -> LaputaError {
    if let Some(laputa_error) = error.downcast_ref::<LaputaError>() {
        return laputa_error.clone();
    }

    if let Some(sql_error) = error.downcast_ref::<rusqlite::Error>() {
        return LaputaError::StorageError(sql_error.to_string());
    }

    if let Some(io_error) = error.downcast_ref::<std::io::Error>() {
        return LaputaError::from(std::io::Error::new(io_error.kind(), io_error.to_string()));
    }

    let message = error.to_string();
    if message.contains("Memory not found") {
        return LaputaError::NotFound(message);
    }
    if message.contains("Cannot open SQLite") || message.contains("Non-UTF8 index path") {
        return LaputaError::ConfigError(message);
    }
    if message.contains("Invalid RFC3339")
        || message.contains("Unknown emotion")
        || message.contains("cannot be empty")
    {
        return LaputaError::ValidationError(message);
    }

    LaputaError::StorageError(message)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_parse_tags_splits_csv() {
        assert_eq!(
            parse_tags(Some("work, focus, ,work".to_string())),
            vec!["work".to_string(), "focus".to_string(), "work".to_string()]
        );
    }

    #[test]
    fn test_parse_time_range_supports_whole_days() {
        let (start, end) = parse_time_range("2026-04-01~2026-04-13").unwrap();
        assert!(end > start);
        assert_eq!(end - start, (13 * 24 * 60 * 60) - 1);
    }

    #[test]
    fn test_parse_time_range_rejects_invalid_format() {
        let error = parse_time_range("2026/04/01").unwrap_err();
        assert!(matches!(error, LaputaError::ValidationError(_)));
    }

    #[test]
    fn test_parse_time_range_rejects_extreme_dates() {
        let error = parse_time_range("0001-01-01~2026-04-01").unwrap_err();
        assert!(matches!(error, LaputaError::ValidationError(_)));
    }

    #[test]
    fn test_parse_time_range_rejects_ranges_over_365_days() {
        let error = parse_time_range("2025-01-01~2026-04-02").unwrap_err();
        assert!(matches!(error, LaputaError::ValidationError(_)));
    }

    #[test]
    fn test_parse_memory_id_rejects_zero_and_negative_values() {
        assert!(matches!(
            parse_memory_id("0").unwrap_err(),
            LaputaError::ValidationError(_)
        ));
        assert!(matches!(
            parse_memory_id("-1").unwrap_err(),
            LaputaError::ValidationError(_)
        ));
    }

    #[test]
    fn test_parse_memory_id_rejects_uuid_for_phase_one() {
        let error = parse_memory_id("550e8400-e29b-41d4-a716-446655440000").unwrap_err();
        assert!(matches!(error, LaputaError::ValidationError(_)));
    }

    #[test]
    fn test_normalize_recall_limit_clamps_to_supported_range() {
        assert_eq!(normalize_recall_limit(0), 1);
        assert_eq!(normalize_recall_limit(42), 42);
        assert_eq!(normalize_recall_limit(20_000), 10_000);
    }

    #[test]
    fn test_handle_init_rejects_blank_name_before_initialize() {
        let error = handle_init(
            None,
            InitCommand {
                name: "   ".to_string(),
            },
        )
        .unwrap_err();
        assert!(matches!(error, LaputaError::ValidationError(_)));
    }
}
