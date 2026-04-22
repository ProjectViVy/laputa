use crate::api::LaputaError;
use serde::{Deserialize, Serialize};

/// 共振度值对象，统一约束在 -100..=100。
#[derive(Debug, Clone, Copy, Serialize, Deserialize, PartialEq, Eq, PartialOrd, Ord)]
pub struct Resonance(i32);

impl Resonance {
    pub fn new(value: i32) -> Result<Self, LaputaError> {
        if !(-100..=100).contains(&value) {
            return Err(LaputaError::ValidationError(format!(
                "resonance must be within -100..=100, got {value}"
            )));
        }

        Ok(Self(value))
    }

    pub fn value(self) -> i32 {
        self.0
    }

    pub fn as_confidence(self) -> f64 {
        self.0 as f64
    }
}
