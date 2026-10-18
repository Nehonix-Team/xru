# Control Flow & Pattern Matching

XRU 2.0 provides standard branching constructs, collection iterations, and declarative pattern matching.

---

## 1. Conditional Branching: `if`, `elif`, `else`

Condition expressions must evaluate to a strict `bool` value. Parentheses are optional, but block delimiters `{ ... }` are required.

```xru
use std

let environment = "staging"
let port = 8080

if environment == "production" {
    std.info("Production profile active on port $f{port}")
} elif environment == "staging" {
    std.info("Staging environment active")
} else {
    std.info("Local development profile active")
}
```

### Logical Operators
- `&&` (Logical AND)
- `||` (Logical OR)
- `!` (Logical NOT)

```xru
if isEnabled && (count > 0 || mode == "force") {
    std.success("Pre-conditions satisfied")
}
```

---

## 2. Loops & Iterations: `for ... in`

The `for ... in` statement iterates over collection types and numeric ranges.

### 2.1 Iterating Over Numeric Ranges (`range`)
```xru
for i in 1..5 {
    std.println("Index: $f{i}")
}
```

### 2.2 Iterating Over Collections (`list`)
```xru
let services = ["auth", "database", "gateway", "queue"]

for service in services {
    std.info("Initializing $f{service}...")
}
```

### 2.3 Unrolling Elements and Indices
By specifying two binding identifiers, loops provide both the element value and its 0-based index:

```xru
let packages = ["core", "utils", "router"]

for pkg, idx in packages {
    std.println("Package #$f{idx}: $f{pkg}")
}
```

---

## 3. Pattern Matching: `match` / `case` / `default`

The `match` block provides declarative pattern matching:

```xru
let command = "deploy"
let targetEnv = "prod"

match command {
    case "build" {
        std.info("Executing build pipeline...")
    }
    case "test" {
        std.info("Running test suites...")
    }
    case "deploy" {
        std.success("Deploying to $f{targetEnv}...")
    }
    default {
        std.error("Unknown command: $f{command}")
    }
}
```

### Static Rules:
- The `default` branch is optional.
- Branch pattern types must match the type of the matched expression.
- References to undeclared identifiers within case branches trigger `[ERR_UNDEFINED_VARIABLE]`.
