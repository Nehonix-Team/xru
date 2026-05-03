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


