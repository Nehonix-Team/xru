# Variables & Interpolation

XRU allows capturing and reusing state via dynamic variables and F-string style interpolation.

---

## 1. Declaration

### Explicit: `let`
Declare variables anywhere.
- `let app = "XyPriss"`

### Implicit: `as`
Capture the result of a directive or action.
- `#EXEC: "git rev-parse HEAD" as COMMIT`

---

## 2. Scoping Rules
XRU uses a hierarchical variable stack:

1. **Global Scope**: Declared at the top level of the `.xru` file.
2. **Local Scope**: Declared inside a block (`#BEGIN`, `#CREATE`).
   - Local variables shadow global ones.
   - They are destroyed once the block ends (`#END`).

---

## 3. Interpolation `{}`
Use `{NAME}` to inject values into strings, paths, or objects.

```xru
let theme = "dark"
#LOG: "Setting theme to {theme}"
SET config.ui.theme "{theme}"
```

### Undefined Variables
Accessing a missing variable results in:
`[ERROR: UNDEFINED_VAR]`
This prevents silent failures in generated configurations.
