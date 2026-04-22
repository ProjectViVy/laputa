use crate::api::LaputaError;
use crate::heat::config::HeatConfig;
use crate::storage::memory::{MAX_HEAT_I32, MIN_HEAT_I32};

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum HeatState {
    Locked,
    Active,
    ArchiveCandidate,
    PackCandidate,
}

impl HeatState {
    pub fn from_heat(heat: i32, config: &HeatConfig) -> Result<Self, LaputaError> {
        if !(MIN_HEAT_I32..=MAX_HEAT_I32).contains(&heat) {
            return Err(LaputaError::HeatThresholdError(heat));
        }

        if heat > config.hot_threshold {
            Ok(Self::Locked)
        } else if heat >= config.warm_threshold {
            Ok(Self::Active)
        } else if heat >= config.cold_threshold {
            Ok(Self::ArchiveCandidate)
        } else {
            Ok(Self::PackCandidate)
        }
    }

    pub fn should_archive(self) -> bool {
        matches!(self, Self::ArchiveCandidate | Self::PackCandidate)
    }
}
