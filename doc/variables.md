# Variable Management & Interpolation

The XRU engine provides a robust state management system using dynamic variables and recursive interpolation.

---

## 1. Variable Declaration

### Explicit Declaration: `let`
Variables can be explicitly declared within any scope.
- `let project_name = "XyPriss"`

### Implicit Capture: `as`
The result or target of a directive or operation can be captured using the `as` keyword.
- `S.EXEC: "git rev-parse HEAD" as COMMIT_HASH`
- `#SELECT: "./dist" as SANDBOX_ROOT`

---

## 2. Scoping Architecture
XRU implements a hierarchical scoping model to ensure predictable state transitions:

1.  **Global Scope**: Variables defined at the root of the `.xru` file.
2.  **Local Scope**: Variables defined within a block (`#BEGIN`, `#CREATE`).
    - Local variables shadow global variables with the same identifier.
    - Local state is strictly confined to the block and is deallocated upon `#END`.

---

## 3. String Interpolation `{}`
Variable values are injected into strings, paths, or structured objects using the `{IDENTIFIER}` syntax.

```xru
let theme_mode = "dark"
U.LOG: "Applying {theme_mode} configuration..."
SET ui.theme "{theme_mode}"
```

### Undefined Variable Handling
Accessing an undefined or out-of-scope variable triggers an immediate error injection:
`[ERROR: UNDEFINED_VAR]`
This fails-fast to prevent corrupted project configurations.
