use clap::Parser;
use laputa::cli::Cli;

#[test]
fn test_cli_mark_accepts_important() {
    let cli = Cli::try_parse_from(["laputa", "mark", "--id", "7", "--important"]).unwrap();

    let debug = format!("{cli:?}");
    assert!(debug.contains("Mark"));
    assert!(debug.contains("important: true"));
}

#[test]
fn test_cli_mark_rejects_multiple_flags() {
    let err = Cli::try_parse_from(["laputa", "mark", "--id", "7", "--important", "--forget"])
        .unwrap_err();

    let rendered = err.to_string();
    assert!(rendered.contains("--important"));
    assert!(rendered.contains("--forget"));
}

#[test]
fn test_cli_mark_requires_one_action_flag() {
    let err = Cli::try_parse_from(["laputa", "mark", "--id", "7"]).unwrap_err();

    let rendered = err.to_string();
    assert!(rendered.contains("required"));
}

#[test]
fn test_cli_mark_requires_valence_and_arousal_for_emotion_anchor() {
    let err = Cli::try_parse_from(["laputa", "mark", "--id", "7", "--emotion-anchor"]).unwrap_err();

    let rendered = err.to_string();
    assert!(rendered.contains("--valence"));
    assert!(rendered.contains("--arousal"));
}

#[test]
fn test_cli_mark_accepts_string_id_for_phase_one_validation() {
    let cli =
        Cli::try_parse_from(["laputa", "mark", "--id", "not-a-number", "--important"]).unwrap();

    let debug = format!("{cli:?}");
    assert!(debug.contains("id: \"not-a-number\""));
}
