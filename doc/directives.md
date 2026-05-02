# XRU Core Components

The XRU language architecture is divided into two distinct functional layers: **Structural Directives** and **Logic Operations**.

---

## 1. Structural Directives
Structural directives define the execution context, file targeting, and script organization. They are always prefixed with the `#` symbol.

### `#USE:<Module> [as Alias]`
Mounts an internal or external module into the script execution context.
- **`<Module>`**: The identifier of the module to load.
- **`[as Alias]`**: Optional local identifier.
- **Defaults**: `Utils` (aliased to `U`), `Sys` (aliased to `S`).

### `#SELECT:<Path> [as Alias]`
Defines the working directory (sandbox) for all relative path resolutions.
- **`<Path>`**: Target directory path.
- **`[as Alias]`**: Captures the absolute path into a variable.

### `#BEGIN:<Path> [as Alias]` / `#END`
Defines a transformation block for an existing file.
- Changes made within this block are local to the specified file.

### `#CREATE:<Path> [as Alias]` / `#END`
Defines a block for the creation of a new file with the specified content.

### `#GLOBAL`
Instructs the engine to apply subsequent actions to all files within the current sandbox recursively.

### `#INCLUDE:<Path>`
Statically imports and executes another `.xru` rule file.

---

## 2. Logic Operations
Logic operations handle program flow, system interaction, and diagnostics. They follow the namespaced syntax: `Alias.Action: "Content"`.

### Module: `Utils` (Default Alias: `U`)
Provides core utility functions for script execution.

| Action | Description |
| :--- | :--- |
| `U.LOG` | Transmits a formatted message to the standard output. |
| `U.ASSERT` | Evaluates a condition and halts execution if validation fails. |

### Module: `Sys` (Default Alias: `S`)
Provides low-level system interactions.

| Action | Description |
| :--- | :--- |
| `S.EXEC` | Executes a shell command within the current sandbox. |
| `S.EXIT` | Terminates the XRU process with a specific exit code. |
