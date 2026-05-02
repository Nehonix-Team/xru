# XRU Actions

Actions are the operations performed **inside** scoping blocks (`#BEGIN`, `#CREATE`) or under `#GLOBAL`. They define how the text should be modified.

---

## 1. Quick Symbols
Best for top-level modifications or whole-object merges.

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
- `SET version "1.2.3"`

### `MERGE <path> { ... }`
Performs a deep merge at a specific nested path.
- `MERGE settings.theme { "mode": "dark" }`

### `REMOVE <path>`
Deletes a specific key or branch.

### `PUSH <path> <value>`
Appends a value to an array at a specific path.

---

## 3. Code Injections
Used for injecting raw code at specific markers in source files.

### `@*INJECT:<key>` / `@END`
Injects code at a marker (e.g., `// xfpm: key`).
- **Language Specific**: Use `@TSINJECT`, `@GOINJECT`, `@RUSTINJECT`, etc., to only apply if the file extension matches the language.
