# Modular Architecture & Mapping (`_mod.xru`)

XRU 2.0 provides an explicit, namespace-based module system that eliminates fragile relative paths and guarantees zero side-effects on import.

---

## 1. Hermetic Project Manifest: `_mod.xru`

The project root is defined by a hermetic configuration manifest named `_mod.xru`.

### Hermetic Rule:
> [!WARNING]
> `_mod.xru` is strictly reserved for structural module mapping.
> It must not contain executable statements, top-level functions, variable assignments, or invocations such as `std.println`.
> Violations raise `[ERR_INVALID_MOD_DECLARATION]`.

### Example `_mod.xru`:
```xru
map meta {
    name: "project-rules",
    version: "2.0.0",
    description: "Template and migration rules"
}

map namespaces {
    services: "./src/services",
    utils: "./src/utils",
    generators: "./templates/generators"
}
```

---

## 2. Module Invocations & Imports

### 2.1 Direct Module Import (`use`)
Used for built-in system modules or local modules within the current directory:

```xru
use std
use crypto as Crypto
use fs
```

### 2.2 Namespace Imports (`from "<ns>" use ...`)
Used for modules mapped under a namespace declared in `_mod.xru`:

```xru
// Import entire module under an alias
from "services" use auth as AuthService

// Selectively import exported symbols
from "utils" use string_helpers::capitalize, string_helpers::slugify

// Import all exported symbols
from "generators" use all::ScaffoldUnit
```

---

## 3. Zero Side-Effect Guarantee on Import

To maintain deterministic environments:

1. **Only Declarations Evaluated**: During `use` or `from "<ns>" use`, the engine processes only type definitions, interfaces, units (`unit`), functions (`func`), and module variables (`global let` / `global const`).
2. **Exclusion of Standalone Scripts**: Executable statements (such as top-level expressions, loops, and condition blocks) present in a module file are **never executed** upon import.
3. **Dedicated Entrypoints**: Standalone statements execute only when the file is run directly via `xru run <file.xru>`.

---

## 4. Automatic Export Type Inference

When modules declare exported global variables without explicit type annotations:

```xru
// In src/services/api.xru:
global let downloadTask = with Net.fetch("https://api.example.com")
global let timeoutSeconds = 60
```

The type checker infers their types (`task[string]`, `number`). Downstream importing files and the Language Server Protocol (LSP) receive full typed completion without falling back to `any`.
