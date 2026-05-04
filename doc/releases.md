# XRU Release Notes

## [v0.1.7] - 2026-05-04

### Added
- **Compact Orchestration**: Support for native `#JSONVAR` blocks that parse content into structured objects.
- **Object Iteration**: `#FOR` now supports iterating over Map keys with automatic alphabetical sorting.
- **Dot Notation**: Support for `{OBJ.prop.sub}` syntax in variable interpolation.
- **Recursive Interpolation**: Up to 10 levels of nested variable resolution.
- **Multi-line Blocks**: Added `#VAR` and language-specific variants (`#TSVAR`, `#JSVAR`) with automatic dedenting.
- **Strict Execution**: The engine now halts immediately on undefined variables or unused variable declarations.
- **Smart FS**: `FS.READ_JSON` with automatic path resolution (lookup in sandbox and script directory).

### Changed
- **VS Code Extension (v0.1.3)**: 
    - Full syntax highlighting for typed `#VAR` blocks (TS, JS, JSON).
    - Fixed property access highlighting (dot notation).

---

## [v0.1.6] - 2026-05-04

### Added
- **Loop Directive (`#FOR`)**: Support for iterating over list structures to reduce code duplication.
- **Dynamic Scoping**: Enhanced variable scoping for loops and conditional blocks.
- **Smart Unescape**: Automatic conversion of `\n` and `\t` in variable declarations to support multi-line code generation.

### Changed
- **Relaxed Indentation**: Structural directives (`#IF`, `#FOR`, `#SELECT`, etc.) now support leading whitespace for better script organization.
- **Unified Variable Syntax**: Replaced redundant `#VAR` with unified `let name = value` syntax.
- **VS Code Extension (v0.1.2)**: 
    - Added syntax highlighting for the `#FOR` loop.
    - Fixed highlighting for indented directives.

---

## [v0.1.5] - 2026-05-03
- Initial support for orchestration directives.
- VS Code extension packaging.
