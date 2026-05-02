# XRU Directives

Directives are top-level instructions starting with `#`. They are categorized into two main families: **Scoping** and **Utility**.

---

## 1. Scoping Directives
These directives define the target of your transformations.

### `#SELECT:<path>`
Defines a base directory (sandbox) for all subsequent rules.
- **`<path>`**: Path relative to the initial target directory.
- Use `#SELECT: .` to return to the root.
- **Capture**: `#SELECT: path as VAR` stores the absolute path in `VAR`.

### `#BEGIN:<path>` / `#END`
Opens a transformation block for an existing file.
- **`<path>`**: Relative path to the file.
- If the file is missing, the block is skipped.

### `#CREATE:<path>` / `#END`
Creates a new file with the provided static content.
- Everything between the tags is treated as raw file content.

### `#GLOBAL`
Applies subsequent actions to **all files** in the current sandbox (recursively).
- Automatically ignores `.git`, `node_modules`, etc.

---

## 2. Utility Directives
These directives handle feedback, validation, and system execution.

### `#LOG:<message>`
Prints a message to the console. Supports [Log Colorization](./syntax.md#log-colorization).

### `#ASSERT:<condition>`
Validates a requirement (e.g., `#ASSERT: exists("file.txt")`). Execution stops on failure.

### `#EXEC:<command>`
Executes a shell command in the current sandbox.
- **Capture**: `#EXEC: "cmd" as OUT` stores the output in `OUT`.

### `#INCLUDE:<path>`
Recursively includes another `.xru` rule file.

### `#BREAK` or `#EXIT:<code>`
Terminates the program immediately.
