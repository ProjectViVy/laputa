use crate::api::LaputaError;

const DEFAULT_RECALL_LIMIT: usize = 100;
const MAX_RECALL_LIMIT: usize = 1_000;

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct RecallQuery {
    pub start: i64,
    pub end: i64,
    pub wing: Option<String>,
    pub room: Option<String>,
    pub limit: usize,
    pub include_discarded: bool,
}

impl RecallQuery {
    pub fn by_time_range(start: i64, end: i64) -> Self {
        Self {
            start,
            end,
            wing: None,
            room: None,
            limit: DEFAULT_RECALL_LIMIT,
            include_discarded: false,
        }
    }

    pub fn with_wing(mut self, wing: impl Into<String>) -> Self {
        self.wing = Some(wing.into());
        self
    }

    pub fn with_room(mut self, room: impl Into<String>) -> Self {
        self.room = Some(room.into());
        self
    }

    pub fn with_limit(mut self, limit: usize) -> Self {
        self.limit = limit.min(MAX_RECALL_LIMIT);
        self
    }

    pub fn include_discarded(mut self, include_discarded: bool) -> Self {
        self.include_discarded = include_discarded;
        self
    }

    pub fn validated_limit(&self) -> usize {
        self.limit.clamp(1, MAX_RECALL_LIMIT)
    }

    pub fn validate(&self) -> Result<(), LaputaError> {
        if self.start > self.end {
            return Err(LaputaError::ValidationError(format!(
                "start must be <= end, got start={} end={}",
                self.start, self.end
            )));
        }
        Ok(())
    }
}
