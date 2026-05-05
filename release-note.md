# XRU (XyPriss Rule Unit) Release Notes

All notable changes to this project will be documented in this file.

## [0.2.5] - 2026-05-05
feat(module): improve S.EXEC output capture
Updated `sys.EXEC` to capture both `stdout` and `stderr` when using the `as` keyword. This ensures that error messages are included in the captured variable.

## [0.2.4] - 2026-05-05
feat(module): add sys.GET for system information
Introduced a new method `GET` to the `sys` module to retrieve system properties.

### Added
- **sys.GET**: Retrieve system properties like `OS`, `ARCH`, `USER`, and `CWD`.
  Example: `S.GET: "OS" as MY_OS`

## [0.2.3] - 2026-05-05
feat(extension): improve VSCode syntax highlighting
Enhanced the VSCode extension grammar to provide better colorization for `#ARG` directives and variable assignments.

### Added
- **Syntax Highlighting**: Specific rules for `#ARG` variable names and the `as` keyword.

## [0.2.2] - 2026-05-05
feat(orchestrator): fix unified orchestration and regex patching
This release addresses critical issues in the argument handling and regex-based file patching systems.

### Fixed
- **#ARG Defaults**: Fixed a bug where default values were ignored if terminal arguments were missing.
- **Regex Patching (~~)**: Improved parsing of `~~ /re/ -> repl` syntax and fixed optional colon handling.
- **Type Safety**: Resolved an interface conversion panic when interpolating complex objects in rules.
- **Bare-word Args**: Argument names in `#ARG` no longer require mandatory quotes.

## [0.2.1] - 2026-05-05
refactor(library): transform XRU into a reusable Go library
Refactored the core orchestration engine into a public package (`pkg/engine`), allowing other Go tools (like XFPM) to integrate XRU logic directly.

### Added
- **Public Engine API**: Introduced the `pkg/engine` package with a new `Runner` struct for programmatic execution.
- **Error Handling**: Replaced internal `os.Exit` calls with standard Go errors for better library integration.
- **CLI Decoupling**: Updated `cmd/xru` to be a lightweight client for the new engine library.

### Fixed
- **Scope Isolation**: Improved variable scope management in nested inception blocks.

## [0.2.0] - 2026-05-04
feat(orchestrator): unified orchestration and advanced CLI arguments
A major milestone introducing the "Unified Orchestrator" pattern, allowing single-script management for multiple project modes (Default/XMS). This version also stabilizes CLI argument passing and enhances regex-based cleanup routines.

### Added
- **Unified Orchestrator**: Support for multi-mode orchestration within a single `.xru` script using dynamic arguments and conditional logic.
- **CLI Arguments**: Implementation of `--arg NAME=VAL` support in the binary to pass variables directly to rules.
- **Object Reference resolution**: Added support for `{DATA.{MODE}}` syntax to resolve nested objects directly into variables.
- **Smart Whitespace Control**: Improved Inception tag processing to prevent unwanted blank lines and preserve intentional indentation.
- **Regex Deletion**: Enhanced `@INJECT` with multiline regex support for safe cleanup of environment and configuration files.

### Improved
- **Developer Experience**: Added comprehensive English documentation/comments to standard orchestration templates.
- **Engine Stability**: Better error reporting for undefined/unused variables during complex orchestration.

## [0.1.9] - 2026-05-04
feat(engine): implement Inception mode and enhanced orchestration engine
Huge update introducing "Inception" mode for nested XRU logic execution within template files, recursive action execution, and hardened orchestration logic.

### Added
- **Inception Mode**: Support for `<# ... >` tags within template blocks to execute nested XRU logic (FOR, IF, etc.).
- **Recursive Orchestration**: Sub-actions are now correctly executed for #BEGIN, #CREATE, and #GLOBAL blocks.
- **Capture Propagation**: Captured output (via #LOG) is now correctly propagated through nested scopes.
- **Universal Inject Markers**: Expanded regex support for @TSINJECT, @GOINJECT and other dynamic markers.
- **Hardened Interpolation**: Improved escaping logic to support literal braces in code blocks without breaking variable replacement.
- **VS Code Extension (v0.5.0)**: Dynamic language injection based on file extensions and support for Inception tags.

### Fixed
- **Action Ordering**: Resolved issues where #ELSE actions would overwrite successful #IF actions.
- **Context Leakage**: Fixed scope isolation between nested structural blocks.

## [0.1.8] - 2026-05-04
fix(parser): support for single-line comments using //
The parser now correctly ignores lines starting with //, allowing for better documentation within XRU scripts.

### Added
- **Comment Support**: Skip lines starting with `//` (ignoring leading whitespace).

## [0.1.7] - 2026-05-04
feat(engine): implement compact orchestration and hardened execution
Added support for native #JSONVAR blocks and object iteration (#FOR). The engine is now strict, halting on undefined or unused variables.

### Added
- **Compact Orchestration**: Single-file orchestration using #JSONVAR and object iteration.
- **Strict Execution**: Fatal errors for undefined/unused variables.
- **Dot Notation**: Support for {OBJ.prop} and recursive interpolation.
- **Typed Blocks**: Added #TSVAR, #JSVAR with automatic dedenting.
- **Smart FS**: FS.READ_JSON with intelligent path resolution.

## [0.1.6] - 2026-05-04
feat(engine): support structural nesting and smart indentation
Enabled structural directives (#IF, #FOR, #SELECT) to be indented for better script organization. Implemented unified variable syntax with 'let'.

### Added
- **Structural Nesting**: Indented # directives for improved readability.
- **Unified Variables**: Standardized 'let name = value' syntax.
- **VS Code Extension (v0.1.2)**: Improved highlighting for #FOR and indented directives.

## [0.1.5] - 2026-05-03
feat(engine): implement automatic dedenting and Markdown extension support
Improved developer experience by implementing "Smart Dedent" for cleaner output files and extending VS Code support to Markdown documentation.

### Added
- **Smart Dedent**: Automatic removal of common leading whitespace in `#CREATE` and `@INJECT` blocks.
- **Indentation Preservation**: Added `--raw` flag to bypass dedenting when exact spacing is required.
- **VS Code Markdown Support**: XRU syntax highlighting and snippets are now active within Markdown code blocks.

### Improved
- **Documentation**: Comprehensive tutorial and examples added for all directives and modules.
- **Parser Robustness**: Better precedence for multi-line actions inside structural blocks.


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
