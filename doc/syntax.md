# XRU Syntax Specification

The XRU (XyPriss Rule Unit) engine is a **Structured Text Patcher** designed to provide precise, formatting-preserving transformations.

---

## 1. Syntax Foundations

### Comments
Use the `//` prefix for single-line comments. These are ignored by the parser and can be placed anywhere in the script.

### Quoting Policy
Outer quotes (single or double) are automatically stripped from structural directives and logic operations. Internal quotes (e.g., within JSON objects or shell commands) are preserved.

### Namespacing
Logic operations utilize a namespaced syntax to ensure modularity and avoid naming collisions: `Namespace.Action: "Content"`.

---

## 2. Standardized Log Colorization

When utilizing the `U.LOG` operation, the engine supports XML-like inline tags for terminal colorization. These tags are natively highlighted in the XRU VS Code extension.

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
