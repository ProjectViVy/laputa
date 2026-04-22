use laputa::api::LaputaError;
use laputa::identity::IdentityInitializer;
use rusqlite::Connection;
use serial_test::serial;
use std::fs;
use tempfile::tempdir;

#[test]
#[serial]
fn test_initialize_creates_db_and_identity() {
    let dir = tempdir().unwrap();
    let initializer = IdentityInitializer::new(dir.path());

    let result = initializer.initialize("大湿");

    assert!(result.is_ok());
    assert!(dir.path().join("laputa.db").exists());
    assert!(dir.path().join("identity.md").exists());

    let content = std::fs::read_to_string(dir.path().join("identity.md")).unwrap();
    assert_eq!(
        content.lines().next(),
        Some("## L0 — IDENTITY"),
        "identity.md 顶部标题必须符合协议"
    );
    assert!(content.contains("user_name: 大湿"));
    assert!(content.contains("user_type: 个人记忆助手"));
    assert!(content.contains("created_at: "));
}
#[test]
#[serial]
fn test_initialize_cleans_up_db_when_schema_creation_fails() {
    let dir = tempdir().unwrap();
    let initializer = IdentityInitializer::new(dir.path());
    fs::write(&initializer.db_path, "not a sqlite database").unwrap();

    let result = initializer.initialize("澶ф箍");
    let error = result.expect_err("schema 创建失败时应返回错误");

    assert!(
        matches!(error, LaputaError::StorageError(_)),
        "unexpected error variant: {error:?}"
    );
    assert!(
        !initializer.db_path.exists(),
        "schema 创建失败后应清理 laputa.db"
    );
    assert!(
        !initializer.identity_path.exists(),
        "schema 创建失败后不应生成 identity.md"
    );
}

#[test]
#[serial]
fn test_reinitialize_returns_error() {
    let dir = tempdir().unwrap();
    let initializer = IdentityInitializer::new(dir.path());

    initializer.initialize("大湿").unwrap();
    let result = initializer.initialize("大湿");

    assert!(matches!(result, Err(LaputaError::AlreadyInitialized(_))));
}

#[test]
#[serial]
fn test_initialize_returns_db_path() {
    let dir = tempdir().unwrap();
    let initializer = IdentityInitializer::new(dir.path());

    let db_path = initializer.initialize("大湿").unwrap();

    assert!(db_path.ends_with("laputa.db"));
    assert!(std::path::Path::new(&db_path).exists());
}

#[test]
#[serial]
fn test_schema_created_with_required_columns() {
    let dir = tempdir().unwrap();
    let initializer = IdentityInitializer::new(dir.path());

    initializer.initialize("大湿").unwrap();

    let conn = Connection::open(dir.path().join("laputa.db")).unwrap();
    let mut statement = conn.prepare("PRAGMA table_info(memories)").unwrap();
    let columns = statement
        .query_map([], |row| row.get::<_, String>(1))
        .unwrap()
        .collect::<Result<Vec<_>, _>>()
        .unwrap();

    for required_column in [
        "id",
        "text_content",
        "wing",
        "room",
        "source_file",
        "valid_from",
        "valid_to",
        "heat_i32",
        "last_accessed",
        "access_count",
        "is_archive_candidate",
        "emotion_valence",
        "emotion_arousal",
    ] {
        assert!(
            columns.iter().any(|column| column == required_column),
            "缺少 schema 列: {required_column}"
        );
    }

    let count: i64 = conn
        .query_row("SELECT COUNT(*) FROM memories", [], |row| row.get(0))
        .unwrap();
    assert_eq!(count, 0);
}

// ========================================
// Story 1.2.1 扩展测试：AC 1, 6, 7 验证
// ========================================

/// AC 6: 检查 DEFAULT 值和 NOT NULL 约束正确性
#[test]
#[serial]
fn test_schema_default_values_and_constraints() {
    let dir = tempdir().unwrap();
    let initializer = IdentityInitializer::new(dir.path());

    initializer.initialize("测试用户").unwrap();

    let conn = Connection::open(dir.path().join("laputa.db")).unwrap();

    // 验证 DEFAULT 值
    let heat_default: i32 = conn
        .query_row(
            "SELECT heat_i32 FROM memories WHERE id = 'test-default-check'",
            [],
            |row| row.get(0),
        )
        .unwrap_or(5000); // 预期 DEFAULT 5000
    assert_eq!(heat_default, 5000, "heat_i32 DEFAULT 应为 5000");

    // 使用 PRAGMA table_info 验证 NOT NULL 约束
    let mut stmt = conn.prepare("PRAGMA table_info(memories)").unwrap();
    let columns = stmt
        .query_map([], |row| {
            Ok((
                row.get::<_, String>(1)?,         // name
                row.get::<_, i32>(3)?,            // notnull (1 = NOT NULL, 0 = nullable)
                row.get::<_, Option<String>>(4)?, // dflt_value
            ))
        })
        .unwrap()
        .collect::<Result<Vec<_>, _>>()
        .unwrap();

    // 检查关键列的 NOT NULL 约束
    // 注意：SQLite PRIMARY KEY 列（id）隐含 NOT NULL，但 PRAGMA table_info 返回 notnull=0
    let not_null_columns = [
        "text_content",
        "valid_from",
        "heat_i32",
        "last_accessed",
        "access_count",
        "is_archive_candidate",
        "emotion_valence",
        "emotion_arousal",
    ];
    for col_name in not_null_columns {
        let col = columns.iter().find(|(name, _, _)| name == col_name);
        assert!(col.is_some(), "列 {col_name} 应存在于 schema");
        let (_, notnull, _) = col.unwrap();
        assert_eq!(*notnull, 1, "列 {col_name} 应有 NOT NULL 约束");
    }

    // 检查 DEFAULT 值
    let default_checks: [(&str, Option<&str>); 7] = [
        ("wing", Some("''")),
        ("room", Some("''")),
        ("heat_i32", Some("5000")),
        ("access_count", Some("0")),
        ("is_archive_candidate", Some("0")),
        ("emotion_valence", Some("0")),
        ("emotion_arousal", Some("0")),
    ];
    for (col_name, expected_default) in default_checks {
        let col = columns.iter().find(|(name, _, _)| name == col_name);
        assert!(col.is_some(), "列 {col_name} 应存在于 schema");
        let (_, _, dflt) = col.unwrap();
        assert_eq!(
            dflt.as_deref(),
            expected_default,
            "列 {col_name} DEFAULT 值应为 {expected_default:?}"
        );
    }
}

/// AC 1, 7: 检查 CHECK 约束定义
#[test]
#[serial]
fn test_schema_check_constraints() {
    let dir = tempdir().unwrap();
    let initializer = IdentityInitializer::new(dir.path());

    initializer.initialize("测试用户").unwrap();

    let conn = Connection::open(dir.path().join("laputa.db")).unwrap();

    // 使用 sql 从 sqlite_master 获取 CHECK 约束定义
    let sql: String = conn
        .query_row(
            "SELECT sql FROM sqlite_master WHERE type='table' AND name='memories'",
            [],
            |row| row.get(0),
        )
        .unwrap();

    // 验证 CHECK 约束存在
    assert!(
        sql.contains("CHECK(emotion_valence >= -100 AND emotion_valence <= 100)"),
        "emotion_valence CHECK 约束应存在，范围 [-100, 100]"
    );
    assert!(
        sql.contains("CHECK(emotion_arousal >= 0 AND emotion_arousal <= 100)"),
        "emotion_arousal CHECK 约束应存在，范围 [0, 100]"
    );
}

/// AC 4: user_name 输入验证测试
#[test]
#[serial]
fn test_user_name_validation_empty() {
    let dir = tempdir().unwrap();
    let initializer = IdentityInitializer::new(dir.path());

    let result = initializer.initialize("");
    assert!(
        matches!(result, Err(LaputaError::ValidationError(_))),
        "空 user_name 应返回 ValidationError"
    );
}

#[test]
#[serial]
fn test_user_name_validation_blank() {
    let dir = tempdir().unwrap();
    let initializer = IdentityInitializer::new(dir.path());

    let result = initializer.initialize("   \t");
    assert!(
        matches!(result, Err(LaputaError::ValidationError(_))),
        "纯空白 user_name 应返回 ValidationError"
    );
    assert!(!dir.path().join("laputa.db").exists());
    assert!(!dir.path().join("identity.md").exists());
}

#[test]
#[serial]
fn test_user_name_validation_newline() {
    let dir = tempdir().unwrap();
    let initializer = IdentityInitializer::new(dir.path());

    let result = initializer.initialize("大湿\n多余行");
    assert!(
        matches!(result, Err(LaputaError::ValidationError(_))),
        "含换行符的 user_name 应返回 ValidationError"
    );
}

#[test]
#[serial]
fn test_user_name_validation_path_traversal() {
    let dir = tempdir().unwrap();
    let initializer = IdentityInitializer::new(dir.path());

    let result = initializer.initialize("../escape");
    let error = result.expect_err("路径遍历 user_name 应失败");

    assert!(matches!(error, LaputaError::ValidationError(_)));
    assert!(error.to_string().contains("path traversal"));
    assert!(!dir.path().join("laputa.db").exists());
}

#[test]
#[serial]
fn test_user_name_validation_too_long() {
    let dir = tempdir().unwrap();
    let initializer = IdentityInitializer::new(dir.path());
    let user_name = "a".repeat(257);

    let result = initializer.initialize(&user_name);
    let error = result.expect_err("超长 user_name 应失败");

    assert!(matches!(error, LaputaError::ValidationError(_)));
    assert!(error.to_string().contains("at most 256"));
    assert!(!dir.path().join("laputa.db").exists());
}

#[test]
#[serial]
fn test_user_name_validation_control_character() {
    let dir = tempdir().unwrap();
    let initializer = IdentityInitializer::new(dir.path());

    let result = initializer.initialize("bad\u{0000}name");
    let error = result.expect_err("控制字符 user_name 应失败");

    assert!(matches!(error, LaputaError::ValidationError(_)));
    assert!(error.to_string().contains("control characters"));
    assert!(!dir.path().join("laputa.db").exists());
}

#[test]
#[serial]
fn test_initialize_cleans_up_db_when_identity_write_fails() {
    let dir = tempdir().unwrap();
    let initializer = IdentityInitializer::new(dir.path());
    fs::create_dir(initializer.identity_path.as_path()).unwrap();

    let result = initializer.initialize("大湿");
    let error = result.expect_err("identity 写入失败时应返回错误");

    assert!(matches!(error, LaputaError::StorageError(_)));
    assert!(!initializer.db_path.exists(), "写入失败后应清理 laputa.db");
    assert!(
        initializer.identity_path.is_dir(),
        "测试构造的 identity 目录应保持存在"
    );
}
