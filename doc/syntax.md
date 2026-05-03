# XRU Syntax Specification

The XRU (XyPriss Rule Unit) engine is a **Structured Text Patcher** designed to provide precise, formatting-preserving transformations with a rigorous structural layout.

---

## 1. Syntax Foundations

### Structural Layout
- **Column-0 Rule**: Every directive starting with `#` must have its anchor (`#`) at the very first character of the line.
- **Nesting Visualization**: To improve readability in nested blocks, spaces are allowed **after** the `#` character.

```xru
#BEGIN: main.ts
#  IF: exists(config.json)
     U.LOG: "Nested logic"
#  END
#END
```

### Strict Quoting Policy
XRU enforces a strict quoting policy for string literals to prevent ambiguity with variables or keywords.
- **Mandatory Quotes**: String literals (log messages, file paths, regex patterns, flag names) **MUST** be enclosed in single (`'`) or double (`"`) quotes.
- **Optional Quotes**: Quotes are optional for the `#USE` directive (module names) and for purely numeric values (e.g. `1`, `8080`).

```xru
#USE: sys as S             // OK: Optional for #USE
U.LOG: "Hello world"       // OK: Mandatory for text
S.ARG: "--mode" as m       // OK: Mandatory for flag names
S.ARG: 1 as first          // OK: Optional for numbers
```

### Comments
Use the `//` prefix for single-line comments.

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

### Implementation Example
```xru
U.LOG: "<cyan>[INFO]</> Transformation sequence initiated..."
```
