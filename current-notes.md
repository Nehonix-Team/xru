## [0.2.1] - 2026-05-05
refactor(library): transform XRU into a reusable Go library
Refactored the core orchestration engine into a public package (`pkg/engine`), allowing other Go tools (like XFPM) to integrate XRU logic directly.

### Added
- **Public Engine API**: Introduced the `pkg/engine` package with a new `Runner` struct for programmatic execution.
- **Error Handling**: Replaced internal `os.Exit` calls with standard Go errors for better library integration.
- **CLI Decoupling**: Updated `cmd/xru` to be a lightweight client for the new engine library.

### Fixed
- **Scope Isolation**: Improved variable scope management in nested inception blocks.

