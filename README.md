# XRU: XyPriss Rule Unit — Syntax Documentation

XRU is a Domain-Specific Language (DSL) designed for **Structured Text Transformation**. Unlike traditional patchers that rely on strict JSON round-tripping, the XRU engine operates as a **Structured Text Patcher (STP)**, preserving formatting, comments, and non-standard syntaxes while applying complex mutations.

---

## Quick Install (Pre-built Binaries)

### Linux / macOS
```bash
curl -sL https://raw.githubusercontent.com/Nehonix-Team/xru/master/install.sh | bash
```

### Windows (PowerShell)
```powershell
irm https://raw.githubusercontent.com/Nehonix-Team/xru/master/install.ps1 | iex
```

---

## 1. Scoping Directives

Directives that define which file(s) are being targeted.

### `#BEGIN:<path>` / `#END`
Opens a transformation block for an existing file.
- **`<path>`**: Relative path to the file from the project root.
- If the file does not exist, the engine skips the block (silent by default, warning in verbose mode).

### `#CREATE:<path>` / `#END`
Creates a new file with the provided static content.
- Everything between the opening and closing tag is treated as raw file content.

### `#SELECT:<path>`
Defines a base directory (sandbox) for all subsequent scoping rules.
- **`<path>`**: Path relative to the initial target directory (or absolute).
- Useful for monorepos or complex project structures.
- Use `#SELECT: .` to return to the root.

### `#GLOBAL` Rules
Any action placed outside of a scoping block applies to **all matching files** in the target directory (recursively).
- Large directories like `.git`, `node_modules`, `dist` are automatically ignored for performance.

---

## 2. Control & Utility Directives

Advanced directives for execution flow and debugging.

### `#LOG:<message>`
Prints a colored message to the console for feedback during transformation.

### `#ASSERT: <condition>`
Validates a requirement before proceeding. If it fails, execution stops immediately with an error code.
- Example: `#ASSERT: exists("package.json")`

### `#INCLUDE:<path>`
Recursively includes another `.xru` file. This allows modularizing complex rule sets.

### `#EXEC:<command>`
Executes a shell command in the current sandbox directory.

### `#BREAK` or `#EXIT:<code>`
Terminates the program immediately. Optionally provides an exit code.

---

## 3. Action Syntax (inside Blocks)

XRU supports a hybrid syntax for maximum flexibility.

### Symbols (Quick Root Actions)
Best for top-level modifications or whole-object merges.
- `++ { ... }` : **MERGE** (Deep merge an object)
- `-- { ... }` : **REMOVE** (Delete specific keys)
- `>> { ... }` : **RENAME** (Rename object keys)
- `<< { ... }` : **APPEND** (Add item to an array)
- `~~ { ... }` : **REGEX** (Search & Replace via regex)

### Directives (Path-Targeted Actions)
Best for precise deep-patching without repeating the structure.
- `MERGE path { ... }` : Fusion at a specific path (e.g., `MERGE settings.theme { "mode": "dark" }`).
- `SET path value` : Overwrite or create a value (e.g., `SET version "1.0.0"`).
- `REMOVE path` : Delete a specific key or branch (e.g., `REMOVE internal.debug`).
- `PUSH path value` : Append to an array at path (e.g., `PUSH features "auth"`).

### Code Injections
`@*INJECT:<key>` / `@END`
Injects code at a marker (e.g., `// xfpm: key`) in the target file.
- Optional language prefix: `@TSINJECT`, `@GOINJECT`, etc.

---

## 4. Variables & Interpolation

XRU allows capturing execution state and reusing it via dynamic variables.

### Capture with `as`
Append `as NAME` to any directive to store its target or result.
- `#SELECT: path as VAR` : Stores the resolved absolute path.
- `#EXEC: command as OUT` : Stores the command string.
- `#CREATE: file as F` : Stores the file path.

### Explicit Declaration with `let`
You can declare variables anywhere without triggering an action.
- `let version = "1.0.0"`
- `let path = "./dist"`

### Scoping Rules
XRU manages variables using a hierarchical stack:
1. **Global Scope**: Variables declared at the root of the `.xru` file.
2. **Local Scope**: Variables declared inside a block (`#BEGIN`, `#CREATE`). 
   - Scoped variables are only visible within their block.
   - They are destroyed after `#END`.
   - They can **shadow** global variables with the same name.

### F-String Interpolation `{}`
Use `{NAME}` to inject variable values into strings, paths, or patch values.
- `#LOG: "Current path is {VAR}"`
- `#BEGIN: {VAR}/config.json`
- `SET version "{VAR}"`

Interpolation is **recursive**: it works inside nested JSON objects during patch operations.

---

## 5. Language Features

- **Comments**: Use `//` for single-line comments anywhere.
- **Unquoted Keys**: Keys in objects don't require quotes if they are simple strings.
- **XyPriss Variables**: Native support and preservation of `&(var).key` syntax.
- **Formatting**: XRU preserves indentation and existing comments in your source files.

---

## 6. CLI Usage

```bash
xru [options] <rule_file.xru> [target_directory]
```

### Options:
- `-v`, `--verbose`: Show detailed execution logs and internal warnings.
- `version`: Show version and platform info.
- `upgrade`: Automatically update to the latest version.
