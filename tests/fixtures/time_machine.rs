//! 时间模拟工具，用于热度衰减测试。
//! 详见架构文档 ADR-012 (测试架构策略)。

pub struct TimeMachine {
    /// 当前模拟时间（相对偏移秒数）
    offset_seconds: u64,
}

impl TimeMachine {
    pub fn new() -> Self {
        Self { offset_seconds: 0 }
    }

    /// 模拟时间流逝（推进 N 天）
    pub fn advance_days(&mut self, days: u64) {
        self.offset_seconds += days * 86_400;
    }

    /// 固定时间，用于精确边界测试
    pub fn freeze(&self) -> u64 {
        self.offset_seconds
    }
}
