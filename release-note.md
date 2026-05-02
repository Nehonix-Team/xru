# XRU (XyPriss Rule Unit) Release Notes

All notable changes to this project will be documented in this file.

## [0.1.1] - 2026-05-02
update compiled binaries for darwin and linux architectures

## [0.1.0] - 2026-05-02

### Added
- **Initial Standalone Release**: Extracted XRU engine from XFPM core.
- **Universal Injection**: Support for `@*INJECT` (e.g., `@GOINJECT`, `@RUSTINJECT`).
- **Structured Operations**: Added `&append:` and `&regex:`.
- **CLI Interface**: Dedicated `xru` binary for standalone text patching.
- **Cross-Platform Support**: Build pipeline for Linux, Darwin, and Windows (amd64/arm64).
