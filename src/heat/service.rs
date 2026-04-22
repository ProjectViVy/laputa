use crate::api::LaputaError;
use crate::heat::config::HeatConfig;
use crate::heat::decay::{calculate_heat, days_since_access};
use crate::heat::state::HeatState;
use crate::storage::memory::LaputaMemoryRecord;
use chrono::{DateTime, Utc};
use std::path::Path;

#[derive(Debug, Clone)]
pub struct HeatService {
    config: HeatConfig,
}

impl HeatService {
    pub fn new(config: HeatConfig) -> Result<Self, LaputaError> {
        config.validate()?;
        Ok(Self { config })
    }

    pub fn load_from_dir(config_dir: &Path) -> Result<Self, LaputaError> {
        Self::new(HeatConfig::load_from_dir(config_dir)?)
    }

    pub fn config(&self) -> &HeatConfig {
        &self.config
    }

    pub fn calculate(&self, record: &LaputaMemoryRecord) -> i32 {
        self.calculate_at(record, Utc::now())
    }

    pub fn calculate_at(&self, record: &LaputaMemoryRecord, now: DateTime<Utc>) -> i32 {
        if !self.config.enabled {
            return record.heat_i32.clamp(0, 10_000);
        }

        let days = days_since_access(record.last_accessed, now);
        calculate_heat(
            record.heat_i32,
            days,
            record.access_count,
            self.config.decay_rate,
        )
    }

    pub fn calculate_batch<'a, I>(&self, records: I) -> Vec<i32>
    where
        I: IntoIterator<Item = &'a LaputaMemoryRecord>,
    {
        let now = Utc::now();
        self.calculate_batch_at(records, now)
    }

    pub fn calculate_batch_at<'a, I>(&self, records: I, now: DateTime<Utc>) -> Vec<i32>
    where
        I: IntoIterator<Item = &'a LaputaMemoryRecord>,
    {
        records
            .into_iter()
            .map(|record| self.calculate_at(record, now))
            .collect()
    }

    pub fn state_for_heat(&self, heat: i32) -> Result<HeatState, LaputaError> {
        HeatState::from_heat(heat, &self.config)
    }

    pub fn state_for_record(&self, record: &LaputaMemoryRecord) -> Result<HeatState, LaputaError> {
        self.state_for_heat(self.calculate(record))
    }

    pub fn state_for_record_at(
        &self,
        record: &LaputaMemoryRecord,
        now: DateTime<Utc>,
    ) -> Result<HeatState, LaputaError> {
        self.state_for_heat(self.calculate_at(record, now))
    }

    pub fn should_archive(&self, heat: i32) -> Result<bool, LaputaError> {
        Ok(self.state_for_heat(heat)?.should_archive())
    }
}
