# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/).

## [0.2.0] - 2025-09-21

### Added
- **Quick Start** example in README showing how to use `Gen` with default and custom `Config`.
- **Advanced Usage** example demonstrating how to reuse a `Generator` instance in tight loops for performance.
- Support for passing `*uniqid.Config` directly to `Gen()` for convenience.
- Documentation updates with sample outputs for clarity.

### Changed
- `Config` field renamed from `Shard` → **`ShardID`** for better readability and consistency.
- `New` function signature updated to accept `*Config` (optional) for easier configuration.

### Fixed
- Minor code cleanups and improved inline documentation.
- Example code in README now compiles and runs without modification.

## [v0.1.0] - 2025-09-21
### Added
- Initial public release of **uniqid**
- 11-character, URL-safe unique ID generator
- Time-sortable and monotonic
- Shard-aware (up to 1024 nodes)
- Thread-safe implementation
- 100% test coverage
- Benchmark comparison vs UUID, ULID, KSUID
- GitHub Actions workflow (CI, coverage, benchmarks)
- MIT License
