## [0.2.8] - 2026-06-8

feat(engine): implement compact orchestration and hardened execution
Added support for native #JSONVAR blocks and object iteration (#FOR). The engine is now strict, halting on undefined or unused variables.

Added
Compact Orchestration: Single-file orchestration using #JSONVAR and object iteration.
Strict Execution: Fatal errors for undefined/unused variables.
Dot Notation: Support for {OBJ.prop} and recursive interpolation.
Typed Blocks: Added #TSVAR, #JSVAR with automatic dedenting.
Smart FS: FS.READ_JSON with intelligent path resolution.

