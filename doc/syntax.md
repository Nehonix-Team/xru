# XRU Syntax Specification

The XRU (XyPriss Rule Unit) engine is a **Structured Text Patcher** designed to provide precise, formatting-preserving transformations with a rigorous structural layout.

---

## 1. Syntax Foundations

### Structural Layout
- **Indentation Support**: Structural directives starting with `#` now support leading whitespace. This allows for standard indentation practices to visualize nesting.

#### Correct Usage
```xru
#USE: utils
#IF: exists("config.json")
    #BEGIN: "config.json"
        SET theme "dark"
    #END
#END
```

### Strict Quoting Policy
XRU enforces a strict quoting policy for string literals to prevent ambiguity with variables or keywords.
- **Mandatory Quotes**: String literals (log messages, file paths, regex patterns, flag names) **MUST** be enclosed in single (`'`) or double (`"`) quotes.
- **Optional Quotes**: Quotes are optional for the `#USE` directive (module names) and for purely numeric values (e.g. `1`, `8080`).

#### Examples
```xru
#USE: sys as S             // OK: Optional for #USE
U.LOG: "Hello world"       // OK: Mandatory for text
S.ARG: "--mode" as m       // OK: Mandatory for flag names
S.ARG: 1 as first          // OK: Optional for numbers
FS.MKDIR: "sandbox"        // OK: Mandatory for paths
```

### Comments
Use the `//` prefix for single-line comments.
```xru
// This is a comment
let version = "1.0.0" // Also a comment
```

---

## 2. Standardized Log Colorization

When utilizing the `U.LOG` operation, the engine supports XML-like inline tags for terminal colorization.

| Tag | Resulting Color |
| :--- | :--- |
| `<red>` | Red |
| `<green>` | Green |
| `<yellow>` | Yellow |
| `<blue>` | Blue |
| `<magenta>` | Magenta |
| `<cyan>` | Cyan |
| `<gray>` | Gray |
| `<white>` | White |
| `</>` | Reset to Terminal Default |

### Implementation Examples
```xru
U.LOG: "<cyan>[INFO]</> Transformation sequence initiated..."
U.LOG: "<red>[ERROR]</> Directory '<white>{path}</>' not found."
U.LOG: "<green>[SUCCESS]</> Build completed in <yellow>{duration}</>ms."
```

