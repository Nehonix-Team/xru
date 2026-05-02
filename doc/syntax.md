# XRU Syntax Overview

XRU (XyPriss Rule Unit) is a **Structured Text Patcher** designed to preserve formatting and comments while applying transformations.

---

## General Rules
- **Comments**: Use `//` for single-line comments.
- **Quoting**: Outer quotes (single or double) are automatically trimmed from directives.
- **Formatting**: Indentation and existing comments in source files are preserved.

---

## Log Colorization
When using the `#LOG` directive, you can use XML-like tags to colorize output:

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
#LOG: "<cyan>[INFO]</> Starting transformation..."
#LOG: "<red>[ERROR]</> Path not found: <gray>{path}</>"
```
