# XRU (XyPriss Rule Unit) Release Notes

All notable changes to this project will be documented in this file.

## [0.1.4] - 2026-05-03
feat(engine): add terminal argument support and strict quoting policy
Added support for reading terminal arguments via `#ARG:` and `S.ARG:`. Implemented a strict quoting policy for string literals to improve engine robustness.

### Added
- **Terminal Arguments**: New `#ARG:` directive and `S.ARG` module method to read flags and positional arguments.
- **Smart CLI**: Improved target directory detection when passing flags.

### Fixed
- **Syntax Integrity**: Enforced mandatory quotes for string literals (paths, log messages, etc.) while keeping them optional for module names and numbers.

## [0.1.3] - 2026-05-03
style(vscode): reformat syntax grammar and cleanup test files
Reformat the VS Code syntax definition for improved readability and 
consistency. Additionally, remove the deprecated test.xru file and 
clean up a typo in apps/index.ts.

### Added
- **Extension Market Ready**: Finalized build script and package for VS Code Marketplace.
- **Documentation**: Created comprehensive `syntax.md`, `actions.md`, `modules.md` guides.

## [0.1.2] - 2026-05-02
Add build script for VS Code extension

## [0.1.1] - 2026-05-02
update compiled binaries for darwin and linux architectures

## [0.1.0] - 2026-05-02

### Added
- **Initial Standalone Release**: Extracted XRU engine from XFPM core.
- **Universal Injection**: Support for `@*INJECT` (e.g., `@GOINJECT`, `@RUSTINJECT`).
- **Structured Operations**: Added `&append:` and `&regex:`.
- **CLI Interface**: Dedicated `xru` binary for standalone text patching.
- **Cross-Platform Support**: Build pipeline for Linux, Darwin, and Windows (amd64/arm64).
