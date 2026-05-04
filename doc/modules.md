# XRU Standard Modules

XRU provides a set of built-in modules to handle system operations, file manipulation, and script utilities.

---

## 1. Utils Module (Alias: `U`)
Core utilities for diagnostics and process control.

| Action | Syntax | Description |
| :--- | :--- | :--- |
| `LOG` | `U.LOG: <content>` | Prints a message to stdout. Supports HSL/ANSI color tags. |
| `EXIT` | `U.EXIT: <code>` | Terminates the XRU process immediately with the specified exit code. |

### Examples
```xru
U.LOG: "<green>[SUCCESS]</> Process completed."
#IF: !exists("required.file")
    U.LOG: "<red>[FATAL]</> Missing dependency."
    U.EXIT: 1
#END
```

---

## 2. Sys Module (Alias: `S`)
Low-level system interactions and shell execution.

| Action | Syntax | Description |
| :--- | :--- | :--- |
| `EXEC` | `S.EXEC: <cmd> [as Var]` | Executes a shell command within the current sandbox. Capture output with `as`. |
| `ARG` | `S.ARG: <key> as Var` | Reads a terminal argument. Key can be a flag (e.g. `"--mode"`) or an index (e.g. `1`). |

### Examples
```xru
S.ARG: 1 as PROJECT_NAME
S.EXEC: "git rev-parse --short HEAD" as COMMIT
U.LOG: "Deploying {PROJECT_NAME} at {COMMIT}..."
```

---

## 3. File System Module (Alias: `FS`)
Cross-platform file and directory manipulation. Highly recommended over `S.EXEC` for portability.

| Action | Syntax | Description |
| :--- | :--- | :--- |
| `MKDIR` | `FS.MKDIR: <path>` | Creates a directory and all necessary parents (equivalent to `mkdir -p`). |
| `RM` | `FS.RM: <path>` | Recursively deletes a file or directory. |
| `TOUCH` | `FS.TOUCH: <path>` | Creates an empty file or updates the timestamp of an existing one. |
| `COPY` | `FS.COPY: <src> -> <dst>` | Copies a file from source to destination. |
| `MOVE` | `FS.MOVE: <src> -> <dst>` | Moves/Renames a file or directory. |
| `READ_JSON` | `FS.READ_JSON: <path> as Var` | Reads a JSON file and parses it as an object/list. |

### Examples
```xru
FS.MKDIR: "logs/archive"
FS.TOUCH: "logs/session.log"
FS.COPY: "config.json" -> "config.backup.json"
FS.MOVE: "temp/build.zip" -> "dist/build.zip"
FS.RM: "temp"
```

### Note on Path Separators
For `COPY` and `MOVE`, use the `->` separator to distinguish between source and destination.
Example: `FS.MOVE: old_name.txt -> new_name.txt`

