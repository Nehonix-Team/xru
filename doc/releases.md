# XRU Release Notes

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
