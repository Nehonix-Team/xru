# XRU Actions

Actions are the operations performed **inside** scoping blocks (`#BEGIN`, `#CREATE`) or under `#GLOBAL`. They define how the text should be modified.

> [!TIP]
> While structural directives (`#`) are anchored at Column 0, actions should be **indented** for readability to visualize the block scope.

---

## 1. Quick Symbols
Best for top-level modifications or whole-object merges. Barewords are supported for values.

| Symbol | Action | Description |
| :--- | :--- | :--- |
| `++` | **MERGE** | Deep merge an object into the target. |
| `--` | **REMOVE** | Delete specific keys or array items. |
| `>>` | **RENAME** | Rename object keys. |
| `<<` | **APPEND** | Add an item to an array. |
| `~~` | **REGEX** | Search and Replace via regular expressions. |

---

## 2. Path-Targeted Keywords
Best for precise deep-patching without repeating the entire file structure.

### `SET <path> <value>`
Overwrites or creates a value at a specific path. 
- Example: `SET version 1.2.3` (Bareword)
- Example: `SET ui.theme "dark"`

### `MERGE <path> <object>`
Performs a deep merge at a specific nested path.
- Example: `MERGE settings.theme { mode: dark }`

### `REMOVE <path>`
Deletes a specific key or branch.

### `PUSH <path> <value>`
Appends a value to an array at a specific path.

---

## 3. Code Injections
Used for injecting raw code at specific markers in source files.

### `@*INJECT: <key>` / `@END`
Injects code at a marker (e.g., `// xfpm: key`).
- **Language Filtering**: Use `@TSINJECT`, `@GOINJECT`, etc., to only apply if the file extension matches.
- **Marker Syntax**: Markers in source files should follow the `// --> {{key}}` pattern.

```xru
#BEGIN: main.ts
  @TSINJECT: imports
    import { Logger } from './logger';
  @END
#END
```
