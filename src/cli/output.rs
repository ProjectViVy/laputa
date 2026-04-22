use crate::vector_storage::MemoryRecord;

pub fn render_init_success(name: &str, db_path: &str) -> String {
    format!("initialized: {name}\ndb_path: {db_path}")
}

pub fn render_diary_write_success(memory_id: i64, tags: &[String]) -> String {
    if tags.is_empty() {
        return format!("memory_id: {memory_id}");
    }

    format!("memory_id: {memory_id}\ntags: {}", tags.join(","))
}

pub fn render_recall_results(time_range: &str, records: &[MemoryRecord]) -> String {
    if records.is_empty() {
        return format!("time_range: {time_range}\nresults: 0");
    }

    let mut lines = vec![
        format!("time_range: {time_range}"),
        format!("results: {}", records.len()),
    ];
    for record in records {
        lines.push(format!(
            "- id={} wing={} room={} heat_i32={} valid_from={}",
            record.id, record.wing, record.room, record.heat_i32, record.valid_from
        ));
        lines.push(record.text_content.trim().to_string());
    }
    lines.join("\n")
}

pub fn render_mark_success(record: &MemoryRecord) -> String {
    let mut lines = vec![
        format!("memory_id: {}", record.id),
        format!("heat_i32: {}", record.heat_i32),
        format!("archive_candidate: {}", record.is_archive_candidate),
    ];

    if record.emotion_valence != 0 || record.emotion_arousal != 0 {
        lines.push(format!("emotion_valence: {}", record.emotion_valence));
        lines.push(format!("emotion_arousal: {}", record.emotion_arousal));
    }

    if let Some(reason) = &record.reason {
        lines.push(format!("reason: {reason}"));
    }

    lines.join("\n")
}
