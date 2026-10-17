# Variables, Scoping & Mutation

This document outlines variable declarations, scope lifecycles, and mutation rules in XRU 2.0.

---

## 1. Declaration Forms

| Keyword | Scope Level | Mutability | Syntax |
| :--- | :--- | :--- | :--- |
| **`let`** | Local block `{ ... }` | Mutable (via `=:`) | `let name: type = value` |
| **`const`** | Local block `{ ... }` | Immutable | `const name: type = value` |
| **`arg`** | Script top-level (CLI) | Immutable | `arg name: type [= default]` |
| **`declare`** | Enclosing `unit` body | Mutable (via `=:`) | `declare &name: type = value` |
| **`global let`** | Module-level | Mutable (via `=:`) | `global let name: type = value` |
| **`global const`**| Module-level | Immutable | `global const name: type = value` |

---

## 2. Safe Mutation Operator (`=:`)

In XRU 2.0, the standard equals sign `=` is reserved strictly for initialization during declaration.
Subsequent variable modification requires the explicit mutation operator **`=:`**:

```xru
let score = 100

// Compilation error: '=' cannot be used for reassignment
// score = 150

// Valid mutation:
score =: 150
score =: score + 10
```

Attempting to mutate an identifier declared with `const` triggers an immediate static checker error:
```xru
const maxWorkers = 50
// maxWorkers =: 60 // [ERR_CONST_MUTATION] Cannot reassign constant 'maxWorkers'
```

---

## 3. Internal Unit Properties (`declare &prop`)

Internal unit properties are defined using `declare` and must be referenced and mutated using the `&` sigil:

```xru
unit SessionManager {
    declare activeToken: string = ""
    declare isConnected: bool = false

    func connect(token: string) -> bool {
        &activeToken =: token
        &isConnected =: true
        return true
    }

    func disconnect() -> void {
        &activeToken =: ""
        &isConnected =: false
    }
}
```

Referencing `&prop` outside of a `unit` definition raises `[ERR_INVALID_PROP_ACCESS]`.

---

## 4. Lexical Scoping & Shadowing

- **Lexical Boundaries**: Variables declared inside a block `{ ... }` are confined to that block and nested descendant blocks.
- **Duplicate Declaration Prevention**: Re-declaring a symbol within the same scope raises `[ERR_DUPLICATE_SYMBOL]`.
- **Closure Inheritance**: Function and lambda closures (`(doc) -> { ... }`) inherit variables declared in parent scopes.
