# Hierarchical Orchestration (`dispatch`)

Complex codebases require traversing directories to execute modular transformation rules across sub-packages and microservices.

XRU 2.0 introduces the **`dispatch`** statement, combining automated traversal, sandboxed execution, and permanent sanitization of descriptor files.

---

## 1. The `dispatch` Statement

The `dispatch` statement takes a directory path expression and an optional matching pattern expression:

```xru
// 1. Direct invocation (implicit default pattern: ".rules.xru")
dispatch "./src"

// 2. Explicit pattern targeting
dispatch "./services/auth", "patch.xru"

// 3. Dynamic target expressions
const microservice = "payment"
dispatch "./services/$f{microservice}", "*.rules.xru"
```

---

## 2. Runtime Execution & Sandboxing

When `dispatch <path>, [<pattern>]` is evaluated:

1. **Path Resolution**: Target paths and patterns are evaluated dynamically at runtime, verifying they yield `string` values.
2. **Directory Traversal**: The engine traverses the target tree, discovering all rule files matching the pattern.
3. **Sandboxed Sub-Instance**: Each matching rule file executes inside its own dedicated runtime instance with isolated scope and independent working directory.
4. **Scope Inheritance & Root Imports**:
   - Sub-descriptors inherit parent global variables without identifier collisions (`[ERR_DUPLICATE_SYMBOL]`).
   - Root-level exports can be imported explicitly via:
     ```xru
     from "root" use all::dbHost, all::envMode
     ```

---

## 3. Sandboxed Primitives

Sub-descriptors executed via `dispatch` receive injected execution primitives:

| Primitive | Signature | Description |
| :--- | :--- | :--- |
| **`Keep(files)`** | `(files: list[string]) -> void` | Preserves specified files and removes all other contents in current directory |
| **`Drop(files)`** | `(files: list[string]) -> void` | Deletes specified files or directories within current directory |
| **`Path`** | `string` | Absolute path of the current sub-descriptor host directory |
| **`Env`** | `map` | Read-only map of process environment variables |

#### Example in `clean.rules.xru`:
```xru
use std

std.info("Purging non-essential assets in $f{Path}...")

Keep([
    "src",
    "package.json",
    "tsconfig.json"
])

Drop([
    "temp.log",
    ".cache"
])
```

---

## 4. Descriptor Sanitization

During aggressive directory pruning (`Keep` or `Drop`), accidental deletion of the active rule file is a common risk in shell scripting.

> [!IMPORTANT]
> **Descriptor Sanitization**:
> The XRU file engine permanently protects:
> - The executing descriptor file.
> - Any file bearing the `.xru` extension.
> 
> Even if a rule executes `Keep(["src"])` and the rule file resides at the root level, the engine rejects the deletion of `.xru` files.
