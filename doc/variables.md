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

## 2. The Golden Rules of Scoping

To ensure script reliability and maintainability, XRU enforces the following rules:

### No Duplicate Definitions
It is forbidden to define the same identifier twice within the same scope. This applies to both `let` and `as` captures.
- **Error**: `let x = 1; let x = 2` (Immediate termination).

### Shadowing Support
A variable defined in a parent scope can be redefined within a sub-block (`#BEGIN`, `#CREATE`).
- The local definition will shadow the global one until the block ends.
- Once the block reaches `#END`, the original global value is restored.

### Usage Tracking
Every variable defined must be utilized at least once within its scope (via interpolation `{VAR}`).
- **Warning**: The engine will emit a warning (tracked in IDEs) for any unused variables.

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
