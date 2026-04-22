//! Heat calculation and state classification utilities.
pub mod config;
pub mod decay;
pub mod service;
pub mod state;

pub use config::HeatConfig;
pub use service::HeatService;
pub use state::HeatState;
