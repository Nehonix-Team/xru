# Syntax Specification

XRU 2.0 implements a clean, deterministic syntax utilizing universal curly brace blocks `{ ... }`, explicit type annotations, and safe mutation operators.

---

## 1. General Structure & Universal Blocks

All functional and control structures in XRU are delimited by curly braces `{ ... }`. Indentation is free-form and whitespace is ignored outside of string literals.

```xru
use std

if true {
    std.println("Standard block indentation")
}
```

### Comments
Single-line comments are prefixed with `//`. They may appear on an isolated line or following any statement:
```xru
// Configuration parameter
let port = 8080 // Default binding
```

---

## 2. Variable Declarations

XRU enforces strict variable declaration rules to prevent accidental state mutations.

### 2.1 Reassignable Local Variables (`let`)
Declares a block-scoped variable. Type annotations are optional when the type can be inferred from the initialization expression:

```xru
let host: string = "localhost"
let port = 3000 // Inferred as 'number'
```

### 2.2 Immutable Constants (`const`)
Declares an immutable constant that cannot be mutated after evaluation:

```xru
const defaultTimeout: duration = 30s
const version = "2.0.0"

// defaultTimeout =: 10s // Static compiler error: [ERR_CONST_MUTATION]
```

### 2.3 Safe Mutation Operator (`=:`)
To prevent inadvertent reassignments, XRU separates initial variable binding (`=`) from state mutation (`=:`):

```xru
let counter = 0

// counter = 1   // SYNTAX ERROR: '=' is reserved for initial declarations
counter =: 1    // VALID: '=:' performs explicit mutation
counter =: counter + 1
```

> [!IMPORTANT]
> Attempting to reassign a variable using `=` generates an immediate syntax error. Always use `=:` to modify existing `let` and `declare` variables.

### 2.4 Declarative CLI Arguments (`arg`)
Top-level parameters accepted from command-line flags (e.g., `--env=prod`, `--port=8080`) are declared using `arg`:

- **Required**: When no default value is assigned.
- **Optional**: When initialized with a default fallback value.

```xru
arg appName: string             // Required: execution fails if --appName is omitted
arg port: number = 8080         // Optional: defaults to 8080
arg isProd: bool = false        // Optional boolean flag
arg timeout: duration = 30s     // Optional duration parameter
```

### 2.5 Module Global Variables (`global let` / `global const`)
Declares module-level variables accessible across the file or exportable across namespaces:

```xru
global const apiPrefix = "/api/v2"
global let activeConnections: number = 0
```

---

## 3. String Literals & Formatting Interpolation (`$f{...}`)

### 3.1 Literal Delimiters
XRU supports multiple string delimiters depending on formatting requirements:
- **Double quotes (`"..."`)**: Standard string literal.
- **Single quotes (`'...'`)**: Useful for strings containing double quotes.
- **Backticks (`` `...` ``)**: Multi-line strings preserving line feeds.
- **Heredocs (`<<<EOF ... EOF>>>`)**: Verbatim blocks for large code payloads.

```xru
let simple = 'Single quoted text'
let standard = "Double quoted text"
let multi = `
    SELECT id, name
    FROM accounts
    WHERE active = true;
`
```

### 3.2 Unambiguous Interpolation with `$f{...}`
Because XRU operates on source code containing destructuring patterns `{ ... }` and environment variables `${...}`:
- Standard braces `{var}` are treated as literal text to avoid collisions with TypeScript or JSON objects.
- Dollar-brace sequences `${var}` are treated as literal text to avoid collisions with shell scripts.

XRU expressions are evaluated dynamically **only** when prefixed with `$f{...}`:

```xru
let name = "XyPriss"
let port = 8080

// Dynamic expression evaluation
std.println("Binding $f{name} on port $f{port}")

// Foreign code remains inert without escape characters:
let tsCode = "import { useState, useEffect } from 'react'"
let bashScript = "echo ${HOME}/bin"
```

---

## 4. Functions (`func`)

Functions are statically checked. Parameter types and return type annotations (`-> Type`) are mandatory (`-> void` if no value is returned):

```xru
func calculateTotal(quantity: number, unitPrice: number) -> number {
    return quantity * unitPrice
}

func logBanner(title: string) -> void {
    std.info("==============================")
    std.info("=== $f{title} ===")
    std.info("==============================")
}
```

---

## 5. Operators

| Category | Operators | Description |
| :--- | :--- | :--- |
| **Arithmetic** | `+`, `-`, `*`, `/`, `%` | Standard numerical evaluation |
| **Comparison** | `==`, `!=`, `<`, `<=`, `>`, `>=` | Equality and order evaluation |
| **Logical** | `&&`, `\|\|`, `!` | Logical AND, OR, NOT |
| **Concatenation** | `+` (on `string`) | String concatenation |
| **Mutation** | `=:` | Explicit state reassignment |
| **Range** | `..` | Numerical sequence generator (`1..10`) |

---

## 6. Output & Logging via `std`

The integrated `std` module provides standard terminal output primitives:

```xru
use std

std.println("Standard message")
std.info("Informational message")
std.success("Task completed successfully")
std.warn("Warning: Deprecated parameter")
std.error("Critical error encountered")
```
