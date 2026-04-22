use crate::api::LaputaError;
use std::fs;
use std::path::Path;

pub const DEFAULT_HOT_THRESHOLD: i32 = 8_000;
pub const DEFAULT_WARM_THRESHOLD: i32 = 5_000;
pub const DEFAULT_COLD_THRESHOLD: i32 = 2_000;
pub const DEFAULT_DECAY_RATE: f64 = 0.1;
pub const DEFAULT_UPDATE_INTERVAL_HOURS: u64 = 1;

#[derive(Debug, Clone, PartialEq)]
pub struct HeatConfig {
    pub enabled: bool,
    pub hot_threshold: i32,
    pub warm_threshold: i32,
    pub cold_threshold: i32,
    pub decay_rate: f64,
    pub update_interval_hours: u64,
}

impl Default for HeatConfig {
    fn default() -> Self {
        Self {
            enabled: true,
            hot_threshold: DEFAULT_HOT_THRESHOLD,
            warm_threshold: DEFAULT_WARM_THRESHOLD,
            cold_threshold: DEFAULT_COLD_THRESHOLD,
            decay_rate: DEFAULT_DECAY_RATE,
            update_interval_hours: DEFAULT_UPDATE_INTERVAL_HOURS,
        }
    }
}

impl HeatConfig {
    pub fn load_from_dir(config_dir: &Path) -> Result<Self, LaputaError> {
        let path = config_dir.join("laputa.toml");
        let content = fs::read_to_string(&path).map_err(|error| {
            LaputaError::ConfigError(format!("failed to read {}: {error}", path.display()))
        })?;

        Self::from_toml_str(&content)
    }

    pub fn from_toml_str(content: &str) -> Result<Self, LaputaError> {
        let mut config = Self::default();
        let mut in_heat_section = false;

        for raw_line in content.lines() {
            let line = raw_line.split('#').next().unwrap_or("").trim();
            if line.is_empty() {
                continue;
            }

            if line.starts_with('[') && line.ends_with(']') {
                in_heat_section = &line[1..line.len() - 1] == "heat";
                continue;
            }

            if !in_heat_section {
                continue;
            }

            let Some((key, value)) = line.split_once('=') else {
                continue;
            };
            let key = key.trim();
            let value = value.trim();

            match key {
                "enabled" => {
                    config.enabled = value.parse::<bool>().map_err(|error| {
                        LaputaError::ConfigError(format!("invalid enabled value {value}: {error}"))
                    })?;
                }
                "hot_threshold" => {
                    config.hot_threshold = parse_i32(value, "hot_threshold")?;
                }
                "warm_threshold" => {
                    config.warm_threshold = parse_i32(value, "warm_threshold")?;
                }
                "cold_threshold" => {
                    config.cold_threshold = parse_i32(value, "cold_threshold")?;
                }
                "decay_rate" => {
                    config.decay_rate = parse_f64(value, "decay_rate")?;
                }
                "update_interval_hours" => {
                    config.update_interval_hours = parse_u64(value, "update_interval_hours")?;
                }
                _ => {}
            }
        }

        config.validate()?;
        Ok(config)
    }

    pub fn validate(&self) -> Result<(), LaputaError> {
        for threshold in [self.cold_threshold, self.warm_threshold, self.hot_threshold] {
            if !(0..=10_000).contains(&threshold) {
                return Err(LaputaError::HeatThresholdError(threshold));
            }
        }

        if self.hot_threshold <= self.warm_threshold {
            return Err(LaputaError::ConfigError(format!(
                "hot_threshold ({}) must be greater than warm_threshold ({})",
                self.hot_threshold, self.warm_threshold
            )));
        }

        if self.warm_threshold <= self.cold_threshold {
            return Err(LaputaError::ConfigError(format!(
                "warm_threshold ({}) must be greater than cold_threshold ({})",
                self.warm_threshold, self.cold_threshold
            )));
        }

        if self.decay_rate < 0.0 || self.decay_rate.is_nan() {
            return Err(LaputaError::ConfigError(format!(
                "decay_rate must be non-negative, got {}",
                self.decay_rate
            )));
        }

        Ok(())
    }
}

fn parse_i32(value: &str, field_name: &str) -> Result<i32, LaputaError> {
    value.parse::<i32>().map_err(|error| {
        LaputaError::ConfigError(format!("invalid {field_name} value {value}: {error}"))
    })
}

fn parse_u64(value: &str, field_name: &str) -> Result<u64, LaputaError> {
    value.parse::<u64>().map_err(|error| {
        LaputaError::ConfigError(format!("invalid {field_name} value {value}: {error}"))
    })
}

fn parse_f64(value: &str, field_name: &str) -> Result<f64, LaputaError> {
    value.parse::<f64>().map_err(|error| {
        LaputaError::ConfigError(format!("invalid {field_name} value {value}: {error}"))
    })
}
