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

### Quoting Policy (Barewords)
XRU follows a "Bareword" policy. Quotes (single or double) around directive targets or logic operation content are **optional**.
- `U.LOG: Hello World` is equivalent to `U.LOG: "Hello World"`
- `#IF: {var} == val` is equivalent to `#IF: "{var}" == "val"`

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
