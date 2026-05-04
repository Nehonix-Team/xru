# Variable Management & Interpolation

The XRU engine provides a robust state management system using dynamic variables and recursive interpolation.

---

## 1. Variable Declaration

### Explicit Declaration: `let`
Variables can be explicitly declared within any scope.
```xru
let project_name = "XyPriss"
let port = 8080
U.LOG: "Configuring {project_name} on port {port}"

# -- Variables support re-assignment
let ROUTE = "/v1"
#IF: {IS_PROD}
    let ROUTE = "/api"
#END
```

### Escaping Support
Values assigned with `let` support standard escape sequences like `\n` (newline) and `\t` (tab). This is particularly useful for injecting formatted code blocks.
```xru
let COMMENT = "/**\n * Multi-line comment\n */"
```
```

### Implicit Capture: `as`
The result or target of a directive can be captured into a variable.
```xru
S.EXEC: "git rev-parse --short HEAD" as COMMIT
#SELECT: "apps/{app_name}" as APP_ROOT
#ARG: "--mode" as MODE
```

---

## 2. Scoping Rules

XRU uses a hierarchical scoping system.

### Block Scopes (`#BEGIN`, `#CREATE`, `#GLOBAL`)
These structural directives create a **New Sub-Scope**.
- Variables defined inside are **local** to the block.
- **Shadowing**: You can redefine a variable from a parent scope; it will be restored after the `#END`.

```xru
let name = "Global"
#BEGIN: "local.txt"
    let name = "Local"
    U.LOG: "{name}" // Prints "Local"
#END
U.LOG: "{name}" // Prints "Global"
```

### Control Scopes (`#IF`, `#ELSE`)
Conditional blocks **Share the Current Scope**.
- Variables defined inside an `#IF` persist after the `#END`.
- This allows conditional configuration setup.

```xru
#IF: exists("prod.flag")
    let env = "production"
#ELSE:
    let env = "development"
#END
U.LOG: "Environment is {env}" // 'env' is accessible here
```

### Loop Scopes (`#FOR`)
Iteration blocks create a **Fresh Sub-Scope** for every iteration.
- Variables defined in one iteration do **not** leak into the next.
- The iteration variable (`S` in the example below) is local to the current loop step.

```xru
#FOR: S in ["a", "b"]
    let TEMP = "{S}_temp"
#END
// TEMP is NOT accessible here
```
```

### Usage Tracking
Every variable must be used via `{VAR}` interpolation. Unused variables trigger a warning to help maintain clean scripts.

---

## 3. String Interpolation `{}`
Variable values are injected using the `{IDENTIFIER}` syntax. Interpolation is recursive, meaning a variable's value can contain other variables.

```xru
let base_url = "https://api.example.com"
let endpoint = "{base_url}/v1/auth"
U.LOG: "Connecting to {endpoint}"
```

### Undefined Variables
Accessing an undefined variable triggers a fatal error injection: `[ERROR: UNDEFINED_VAR]`. This ensures that no invalid configuration is silently generated.

