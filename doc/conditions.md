# XRU Conditional Expressions

XRU supports native conditional logic via the `#IF`, `#ELSE IF`, and `#ELSE` directives. Conditions are evaluated at runtime within the current sandbox context.

---

## 1. Supported Operators & Functions

| Syntax | Description | Example |
| :--- | :--- | :--- |
| `exists(path)` | Returns `true` if the path exists in the current sandbox. | `exists("config.json")` |
| `==` | Equality comparison between variables or literals. | `{env} == "prod"` |
| `!=` | Inequality comparison. | `{version} != "1.0.0"` |
| `!` | **NOT Operator**: Negates the following condition. | `!exists("tmp/")` |
| `true` | Literal boolean true. | `#IF: true` |

### Complex Examples
```xru
// Check if a directory is missing AND a flag is set
#IF: !exists("src") == {auto_init}
    U.LOG: "Initializing source directory..."
    FS.MKDIR: "src"
#END

// Nested conditions
#IF: exists("package.json")
    #IF: !exists("node_modules")
        U.LOG: "<yellow>[WARN]</> node_modules missing, please run install."
    #END
#END
```

---

## 2. Condition Composition

Conditions are "Barewords" by default, meaning quotes are optional unless the value itself contains spaces or special characters that conflict with the parser.

### Examples

#### File Validation & Termination
```xru
#IF: !exists("src/main.ts")
    U.LOG: "<red>[ERROR]</> Main entry point missing!"
    U.EXIT: 1
#END
```

#### Environment Branching
```xru
let mode = "dev"

#IF: {mode} == "prod"
    U.LOG: "Setting up production environment..."
#ELSE IF: {mode} == "dev"
    U.LOG: "Setting up development environment..."
#ELSE:
    U.LOG: "Unknown mode: {mode}"
#END
```

---

## 3. Evaluation Logic

1.  **Interpolation**: Variables (`{VAR}`) are resolved first.
2.  **Negation**: The `!` prefix is processed.
3.  **Function/Operator Call**: The specific logic (`exists`, `==`, etc.) is executed.
4.  **Fallback**: If no pattern matches, the engine checks if the result is exactly the string `"true"`.

