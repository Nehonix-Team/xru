# XRU Core Components

The XRU language architecture is divided into two distinct functional layers: **Structural Directives** and **Logic Operations**.

---

## 1. Structural Directives
Structural directives define the execution context, file targeting, and control flow. 

> [!NOTE]
> **Indentation support**: Structural directives starting with `#` now support leading whitespace for better visual nesting. The `#` character marks the start of the directive regardless of its position on the line.

### Context & Scoping
#### `#USE:<Module> [as Alias]`
Mounts a module into the execution context.
```xru
#USE: sys as S
#USE: utils as U
S.EXEC: "ls"
```

#### `#SELECT:"<Path>" [as Alias]`
Defines the working directory (sandbox) for subsequent actions.
```xru
#IF: !exists("dist")
    FS.MKDIR: "dist"
#END
#SELECT: "dist"
// All following file operations will be relative to 'dist'
#CREATE: "version.txt"
    1.0.0
#END
```

#### `#ARG:"<Key>" as <VarName>`
Reads an argument from the terminal command line.
```xru
#ARG: 1 as PROJECT_NAME
#ARG: "--env" as ENVIRONMENT
U.LOG: "Initializing {PROJECT_NAME} in {ENVIRONMENT} mode..."
```

#### `#BEGIN:"<Path>" [as Alias]` / `#END`
Defines a transformation block for an existing file.
```xru
#BEGIN: "package.json"
  SET version "2.0.0"
  MERGE dependencies { "xypriss": "latest" }
#END
```

#### `#CREATE:"<Path>" [as Alias]` / `#END`
Generates a new file with the provided content. Supports interpolation and automatic dedenting.
Use `--raw` to preserve exact script indentation in the output.

```xru
#CREATE: "src/index.ts"
    console.log("Hello World");
#END

#CREATE: "raw_file.txt" --raw
    This will keep its
    leading spaces.
#END
```

#### `#GLOBAL`
Applies subsequent actions to all files within the current sandbox recursively.
```xru
#GLOBAL
    ~~ "Copyright 2023" "Copyright 2024"
#END
```

---

## 2. Control Flow
XRU supports native conditional logic that can be nested within structural blocks.

### `#IF: <Condition>` / `#ELSE IF:` / `#ELSE:` / `#END`
Defines a conditional execution block.
- **Conditions**: Barewords are supported (no quotes needed).
- **Functions**: `exists(path)` checks for file existence within the current sandbox.
- **Operators**: `==`, `!=` for variable comparison.

```xru
#IF: !exists("config.json")
    U.LOG: "Missing config, creating default..."
    #CREATE: "config.json"
       { "theme": "dark" }
    #END
#ELSE IF: exists("config.old.json")
    U.LOG: "Migrating old config..."
    FS.MOVE: "config.old.json" -> "config.json"
#ELSE:
    U.LOG: "Config found!"
#END
```

---

## 3. Iteration & Modularity

### `#FOR: <Variable> in <List>` / `#END`
Iterates over an array structure. This is the primary tool for reducing code duplication in multi-server or multi-component architectures.
- **Scope**: Each iteration creates a fresh sub-scope inheriting from the parent.
- **Variables**: The iteration variable is available within the block via `{VAR}`.

```xru
let SERVERS = ["auth", "main", "web"]

#FOR: S in {SERVERS}
    let ROUTE = "/{S}"
    #IF: {S} == "main"
        let ROUTE = "/api"
    #END
    U.LOG: "Configuring {S} on {ROUTE}"
#END
```

### `#INCLUDE: "<Path>"`
Embeds the content of another XRU file at the current position. 
- **Variable inheritance**: The included file has access to all variables defined in the current scope.
- **Recursive usage**: Included files can contain further `#INCLUDE` directives.

```xru
let PROJECT_NAME = "MyProject"
#INCLUDE: "common_setup.xru"
```

---

## 4. Logic Operations
Logic operations handle system interaction, diagnostics, and file manipulation. They are organized into **Modules**.

See [Standard Modules](modules.md) for a complete reference of available modules and actions (Utils, Sys, FS).

