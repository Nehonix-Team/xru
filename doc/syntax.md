# XRU Syntax Overview

XRU (XyPriss Rule Unit) is a **Structured Text Patcher** designed to preserve formatting and comments.

---

## General Rules
- **Comments**: Use `//` for single-line comments.
- **Quoting**: Outer quotes are trimmed from directives and modular actions.
- **Namespacing**: Utilities are called using `Namespace.Action: "content"`.

---

## Log Colorization
When using `U.LOG`, you can use XML-like tags to colorize output. These tags are highlighted in VS Code.

| Tag | Color |
| :--- | :--- |
| `<red>` | Red |
| `<green>` | Green |
| `<yellow>` | Yellow |
| `<blue>` | Blue |
| `<magenta>` | Magenta |
| `<cyan>` | Cyan |
| `<gray>` | Gray |
| `<white>` | White |
| `</>` | Reset to default |

**Example:**
```xru
U.LOG: "<cyan>[INFO]</> Starting transformation..."
```
