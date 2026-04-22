# Laputa Status

## Repository Mode

- Current repository mode: standalone
- Runtime prerequisite: this repository only
- Build/test/init entrypoint:
  - `cargo build`
  - `cargo test`
  - `cargo run -- init`

## Upstream Lineage Tracking

This section records historical source lineage, not present-day runtime requirements.

- Historical upstream project: `mempalace-rs`
- Tracking mode: manual follow
- Tracked commit: `c96cae9a121530897906baa2e9d2f4cc3ebd1af5`
- Tracked short SHA: `c96cae9`
- Upstream describe: `v0.4.2-1-gc96cae9`
- Upstream commit date: `2026-04-12`
- Upstream commit subject: `docs: autonomously update 2026 benchmarks [skip ci]`

## Notes

- Laputa evolved from `mempalace-rs`, but the upstream repository is not required to build, test, or run Laputa.
- Update this file when Laputa manually syncs, rebases, or selectively ports changes from upstream lineage.
- Add divergence notes here when Laputa introduces behavior or architecture that intentionally departs from its historical sources.
