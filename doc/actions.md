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
Injects code at a dynamic marker within a source file. XRU uses a **Universal Marker Detection** system that is language-agnostic.

- **Comment Styles**: Supports `//`, `#`, `--`, `/*`, `<!--`.
- **Triggers**: Detects `-->`, `xru:`, `xfpm:`, or direct keys.
- **Key Format**: Works with or without `{{}}` braces.

#### Example Markers (Target Files)
- `// --> {{imports}}`
- `# xru: configuration`
- `/* xfpm: style */`
- `<!-- --> {{footer}}`

#### Example Usage (XRU)
```xru
#BEGIN: main.ts
  @TSINJECT: imports
    import { Logger } from './logger';
  @END
#END
```
