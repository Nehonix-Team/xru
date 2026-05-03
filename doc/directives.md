# XRU Core Components

The XRU language architecture is divided into two distinct functional layers: **Structural Directives** and **Logic Operations**.

---

## 1. Structural Directives
Structural directives define the execution context, file targeting, and control flow. 

> [!IMPORTANT]
> **Column-0 Rule**: All structural directives starting with `#` must be positioned at the very beginning of the line (Column 0). However, spaces are allowed after the `#` character to visualize nesting (e.g., `#  IF:`).

### Context & Scoping
#### `#USE:<Module> [as Alias]`
Mounts a module into the execution context.
- **Defaults**: `Utils` (alias `U`), `Sys` (alias `S`).

#### `#SELECT:<Path> [as Alias]`
Defines the working directory (sandbox).
- **Security**: XRU validates that the path exists. If not, execution is aborted.

#### `#BEGIN:<Path> [as Alias]` / `#END`
Defines a transformation block for an existing file. Supports nested control flow.

#### `#CREATE:<Path> [as Alias]` / `#END`
Creates a new file with the provided content.

#### `#GLOBAL`
Applies subsequent actions to all files within the current sandbox recursively.

---

## 2. Control Flow
XRU supports native conditional logic that can be nested within structural blocks.

### `#IF: <Condition>` / `#ELSE IF:` / `#ELSE:` / `#END`
Defines a conditional execution block.
- **Conditions**: Barewords are supported (no quotes needed).
- **Functions**: `exists(path)` checks for file existence within the current sandbox.
- **Operators**: `==`, `!=` for variable comparison.

```xru
#IF: exists(config.json)
    U.LOG: "Config found!"
#ELSE:
    U.LOG: "Missing config, creating default..."
    #CREATE: config.json
       { "theme": "dark" }
    #END
#END
```

---

## 3. Logic Operations
Logic operations handle system interaction, diagnostics, and file manipulation. They are organized into **Modules**.

See [Standard Modules](modules.md) for a complete reference of available modules and actions (Utils, Sys, FS).
