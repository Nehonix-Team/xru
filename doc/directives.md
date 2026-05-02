# XRU Directives & Modules

XRU separates **Structure** (Structural Directives) from **Logic** (Modular Actions).

---

## 1. Structural Directives
These directives start with `#` and define the target files and the script structure.

### `#USE:<module> [as alias]`
Loads a built-in module.
- `#USE: Utils as U`
- Default aliases: `U` for `Utils`, `S` for `Sys`.

### `#SELECT:<path>`
Defines a base directory (sandbox) for subsequent rules.

### `#BEGIN:<path>` / `#END`
Opens a transformation block for an existing file.

### `#CREATE:<path>` / `#END`
Creates a new file with provided content.

### `#GLOBAL`
Applies actions to all files in the sandbox.

### `#INCLUDE:<path>`
Recursively includes another rule file.

---

## 2. Modular Actions
Logic and utilities are accessed via namespaces in the format `Alias.Action:`.

### Module: `Utils` (Alias: `U`)
| Action | Description |
| :--- | :--- |
| `U.LOG` | Prints a colored message. |
| `U.ASSERT` | Validates a condition. |
| `U.INCLUDE` | Dynamic rule inclusion. |

### Module: `Sys` (Alias: `S`)
| Action | Description |
| :--- | :--- |
| `S.EXEC` | Executes a shell command. |
| `S.EXIT` | Terminates the program with a code. |
