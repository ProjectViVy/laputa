use std::path::PathBuf;

use clap::{ArgGroup, Args, Parser, Subcommand};

use crate::api::LaputaError;
use crate::cli::handlers;

#[derive(Debug, Parser)]
#[command(name = "laputa", version, about = "Laputa CLI")]
pub struct Cli {
    #[arg(long)]
    pub config_dir: Option<PathBuf>,

    #[command(subcommand)]
    pub command: Commands,
}

#[derive(Debug, Subcommand)]
pub enum Commands {
    Init(InitCommand),
    Diary(DiaryCommand),
    Recall(RecallCommand),
    Wakeup(WakeupCommand),
    Mark(MarkCommand),
}

#[derive(Debug, Args)]
pub struct InitCommand {
    #[arg(long)]
    pub name: String,
}

#[derive(Debug, Args)]
pub struct RecallCommand {
    #[arg(long = "time-range")]
    pub time_range: String,

    #[arg(long)]
    pub wing: Option<String>,

    #[arg(long)]
    pub room: Option<String>,

    #[arg(long, default_value_t = 100)]
    pub limit: usize,

    #[arg(long, default_value_t = false)]
    pub include_discarded: bool,
}

#[derive(Debug, Args)]
pub struct WakeupCommand {
    #[arg(long)]
    pub wing: Option<String>,
}

#[derive(Debug, Args)]
pub struct DiaryCommand {
    #[command(subcommand)]
    pub command: DiarySubcommands,
}

#[derive(Debug, Subcommand)]
pub enum DiarySubcommands {
    Write(DiaryWriteCommand),
}

#[derive(Debug, Args)]
pub struct DiaryWriteCommand {
    #[arg(long)]
    pub content: String,

    #[arg(long)]
    pub tags: Option<String>,

    #[arg(long)]
    pub emotion: Option<String>,

    #[arg(long)]
    pub wing: Option<String>,

    #[arg(long)]
    pub room: Option<String>,
}

#[derive(Debug, Args)]
#[command(group(
    ArgGroup::new("intervention")
        .args(["important", "forget", "emotion_anchor"])
        .required(true)
        .multiple(false)
))]
pub struct MarkCommand {
    #[arg(long)]
    pub id: String,

    #[arg(long)]
    pub important: bool,

    #[arg(long)]
    pub forget: bool,

    #[arg(long = "emotion-anchor", requires_all = ["valence", "arousal"])]
    pub emotion_anchor: bool,

    #[arg(long)]
    pub valence: Option<i32>,

    #[arg(long)]
    pub arousal: Option<u32>,

    #[arg(long)]
    pub reason: Option<String>,
}

impl Cli {
    pub fn run(self) -> Result<String, LaputaError> {
        handlers::run(self)
    }
}
