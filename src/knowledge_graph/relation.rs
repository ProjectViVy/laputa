use rusqlite::Row;
use serde::{Deserialize, Serialize};
use std::str::FromStr;

/// Laputa 支持的主体关系类型。
#[derive(Debug, Clone, Copy, Serialize, Deserialize, PartialEq, Eq, Hash)]
#[serde(rename_all = "snake_case")]
pub enum RelationKind {
    PersonPerson,
    PersonProject,
    PersonSelf,
}

impl RelationKind {
    /// 返回稳定的数据库谓词字符串。
    pub fn as_str(self) -> &'static str {
        match self {
            Self::PersonPerson => "person_person",
            Self::PersonProject => "person_project",
            Self::PersonSelf => "person_self",
        }
    }

    pub(crate) fn all() -> &'static [&'static str] {
        &["person_person", "person_project", "person_self"]
    }
}

impl FromStr for RelationKind {
    type Err = ();

    fn from_str(value: &str) -> Result<Self, Self::Err> {
        match value {
            "person_person" => Ok(Self::PersonPerson),
            "person_project" => Ok(Self::PersonProject),
            "person_self" => Ok(Self::PersonSelf),
            _ => Err(()),
        }
    }
}

/// 结构化的关系读取模型，区分当前关系与历史关系。
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
pub struct RelationRecord {
    pub subject: String,
    pub object: String,
    pub relation_type: RelationKind,
    pub resonance: i32,
    pub valid_from: Option<String>,
    pub valid_to: Option<String>,
    pub source_closet: Option<String>,
    pub source_file: Option<String>,
    pub current: bool,
}

impl RelationRecord {
    pub(crate) fn from_row(row: &Row<'_>) -> rusqlite::Result<Self> {
        let predicate: String = row.get(1)?;
        let valid_to: Option<String> = row.get(5)?;

        Ok(Self {
            subject: row.get(0)?,
            relation_type: predicate.parse::<RelationKind>().ok().ok_or_else(|| {
                rusqlite::Error::InvalidColumnType(
                    1,
                    "predicate".to_string(),
                    rusqlite::types::Type::Text,
                )
            })?,
            object: row.get(2)?,
            resonance: row.get::<_, f64>(3)?.round() as i32,
            valid_from: row.get(4)?,
            valid_to: valid_to.clone(),
            source_closet: row.get(6)?,
            source_file: row.get(7)?,
            current: valid_to.is_none(),
        })
    }
}

/// WakePack 需要的高共振关系模型。
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub struct ResonantRelation {
    pub subject: String,
    pub predicate: String,
    pub object: String,
    pub relation_type: RelationKind,
    pub resonance: i32,
    pub confidence: f64,
    pub valid_from: Option<String>,
    pub source_file: Option<String>,
}

/// 周摘要需要的关系变更模型。
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
pub struct RelationChange {
    pub subject: String,
    pub predicate: String,
    pub object: String,
    pub previous_resonance: Option<i32>,
    pub current_resonance: i32,
    pub delta: i32,
    pub valid_from: Option<String>,
    pub source_file: Option<String>,
}
