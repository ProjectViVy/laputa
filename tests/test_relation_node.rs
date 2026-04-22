use laputa::api::LaputaError;
use laputa::knowledge_graph::{KnowledgeGraph, RelationKind};
use serial_test::serial;
use tempfile::tempdir;

fn build_graph() -> KnowledgeGraph {
    let dir = tempdir().unwrap();
    KnowledgeGraph::new(dir.path().join("knowledge.db").to_str().unwrap()).unwrap()
}

#[test]
#[serial]
fn test_upsert_relation_creates_structured_current_relation() {
    let kg = build_graph();

    let relation = kg
        .upsert_relation(
            "Alice",
            "Project Atlas",
            RelationKind::PersonProject,
            67,
            Some("2026-04-15"),
            Some("projects"),
            Some("relations.md"),
        )
        .unwrap();

    assert_eq!(relation.relation_type, RelationKind::PersonProject);
    assert_eq!(relation.resonance, 67);
    assert!(relation.current);

    let current = kg.get_current_relations("Alice").unwrap();
    assert_eq!(current.len(), 1);
    assert_eq!(current[0].subject, "Alice");
    assert_eq!(current[0].object, "Project Atlas");
    assert_eq!(current[0].relation_type, RelationKind::PersonProject);
    assert_eq!(current[0].resonance, 67);

    let inverse = kg.get_current_relations("Project Atlas").unwrap();
    assert_eq!(inverse.len(), 1);
    assert_eq!(inverse[0].subject, "Alice");
}

#[test]
#[serial]
fn test_upsert_relation_updates_timeline_and_preserves_history() {
    let kg = build_graph();

    kg.upsert_relation(
        "Alice",
        "Mira",
        RelationKind::PersonPerson,
        40,
        Some("2026-04-10"),
        None,
        Some("before.md"),
    )
    .unwrap();
    kg.upsert_relation(
        "Alice",
        "Mira",
        RelationKind::PersonPerson,
        75,
        Some("2026-04-12"),
        None,
        Some("after.md"),
    )
    .unwrap();

    let current = kg.get_current_relations("Alice").unwrap();
    assert_eq!(current.len(), 1);
    assert_eq!(current[0].resonance, 75);
    assert_eq!(current[0].valid_from.as_deref(), Some("2026-04-12"));
    assert!(current[0].valid_to.is_none());

    let timeline = kg.get_relation_timeline("Alice").unwrap();
    assert_eq!(timeline.len(), 2);
    assert_eq!(timeline[0].resonance, 40);
    assert_eq!(timeline[0].valid_to.as_deref(), Some("2026-04-12"));
    assert_eq!(timeline[1].resonance, 75);
    assert!(timeline[1].current);

    let changes = kg
        .relation_changes_between("2026-04-01", "2026-04-30", 10, 10)
        .unwrap();
    assert!(changes.iter().any(|change| {
        change.subject == "Alice"
            && change.object == "Mira"
            && change.previous_resonance == Some(40)
            && change.current_resonance == 75
            && change.delta == 35
    }));
}

#[test]
#[serial]
fn test_upsert_relation_handles_relation_type_change_without_duplicate_current_rows() {
    let kg = build_graph();

    kg.upsert_relation(
        "Alice",
        "InnerSelf",
        RelationKind::PersonProject,
        55,
        Some("2026-04-10"),
        None,
        None,
    )
    .unwrap();
    kg.upsert_relation(
        "Alice",
        "InnerSelf",
        RelationKind::PersonSelf,
        55,
        Some("2026-04-11"),
        None,
        None,
    )
    .unwrap();
    kg.upsert_relation(
        "Alice",
        "InnerSelf",
        RelationKind::PersonSelf,
        55,
        Some("2026-04-11"),
        None,
        None,
    )
    .unwrap();

    let current = kg.get_current_relations("Alice").unwrap();
    assert_eq!(current.len(), 1);
    assert_eq!(current[0].relation_type, RelationKind::PersonSelf);
    assert_eq!(current[0].resonance, 55);

    let timeline = kg.get_relation_timeline("Alice").unwrap();
    assert_eq!(timeline.len(), 2);
    assert_eq!(timeline[0].relation_type, RelationKind::PersonProject);
    assert_eq!(timeline[1].relation_type, RelationKind::PersonSelf);
}

#[test]
#[serial]
fn test_upsert_relation_validates_resonance_range_and_accepts_boundaries() {
    let kg = build_graph();

    kg.upsert_relation(
        "Alice",
        "Mira",
        RelationKind::PersonPerson,
        -100,
        Some("2026-04-10"),
        None,
        None,
    )
    .unwrap();
    kg.upsert_relation(
        "Alice",
        "Project Atlas",
        RelationKind::PersonProject,
        100,
        Some("2026-04-11"),
        None,
        None,
    )
    .unwrap();

    let error = kg
        .upsert_relation(
            "Alice",
            "Broken",
            RelationKind::PersonProject,
            101,
            Some("2026-04-12"),
            None,
            None,
        )
        .unwrap_err();

    assert!(matches!(error, LaputaError::ValidationError(_)));
}
