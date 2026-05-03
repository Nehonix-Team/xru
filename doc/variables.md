# Variable Management & Interpolation

The XRU engine provides a robust state management system using dynamic variables and recursive interpolation.

---

## 1. Variable Declaration

### Explicit Declaration: `let`
Variables can be explicitly declared within any scope.
- **Bareword Support**: `let project_name = XyPriss`
- **Quoted**: `let project_name = "XyPriss"`

### Implicit Capture: `as`
The result or target of a directive can be captured into a variable.
- `S.EXEC: "git rev-parse HEAD" as COMMIT_HASH`
- `#SELECT: apps/{app_name} as ROOT`

---

## 2. Scoping Rules

XRU uses a hierarchical scoping system.

### Block Scopes (`#BEGIN`, `#CREATE`, `#GLOBAL`)
These structural directives create a **New Sub-Scope**.
- Variables defined inside are **local** to the block.
- **Shadowing**: You can redefine a variable from a parent scope; it will be restored after the `#END`.

### Control Scopes (`#IF`, `#ELSE`)
Conditional blocks **Share the Current Scope**.
- Variables defined inside an `#IF` persist after the `#END`.
- This allows conditional configuration setup.

### Usage Tracking
Every variable must be used via `{VAR}` interpolation. Unused variables trigger a warning to help maintain clean scripts.

---

## 3. String Interpolation `{}`
Variable values are injected using the `{IDENTIFIER}` syntax.

```xru
let theme = dark
U.LOG: "Applying {theme}..."
#BEGIN: config.json
  SET ui.theme "{theme}"
#END
```

### Undefined Variables
Accessing an undefined variable triggers a fatal error injection: `[ERROR: UNDEFINED_VAR]`. This ensures that no invalid configuration is silently generated.
