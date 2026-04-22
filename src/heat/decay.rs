use crate::storage::memory::{MAX_HEAT_I32, MIN_HEAT_I32};
use chrono::{DateTime, Utc};

pub fn days_since_access(last_accessed: DateTime<Utc>, now: DateTime<Utc>) -> f64 {
    if now <= last_accessed {
        0.0
    } else {
        (now - last_accessed).num_seconds() as f64 / 86_400.0
    }
}

pub fn calculate_heat(
    base_i32: i32,
    days_since_access: f64,
    access_count: u32,
    decay_rate: f64,
) -> i32 {
    let base = clamp_heat(base_i32) as f64;

    if access_count == 0 {
        return base.round() as i32;
    }

    let safe_days = days_since_access.max(0.0);
    let decay_multiplier = (-decay_rate.max(0.0) * safe_days).exp();
    let access_multiplier = (access_count as f64 + 1.0).ln();
    let heat = base * decay_multiplier * access_multiplier;

    clamp_heat(heat.round() as i32)
}

pub fn clamp_heat(heat: i32) -> i32 {
    heat.clamp(MIN_HEAT_I32, MAX_HEAT_I32)
}
