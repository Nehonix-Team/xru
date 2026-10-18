# Migration Guide: Legacy Directives to XRU 2.0

> [!NOTE]
> In **XRU 2.0**, legacy hash directives (`#USE`, `#BEGIN`, `#SELECT`, `#IF`, `#VAR`, etc.) have been completely replaced with modern, first-class statements and universal `{ ... }` blocks.

This document provides a mapping table between legacy v1 directives and idiomatic XRU 2.0 syntax.

---

## Equivalency Table: v1 Directives to v2.0 Syntax

| Legacy Directive (v1) | Modern XRU 2.0 Syntax | Documentation Reference |
| :--- | :--- | :--- |
| `#USE: <mod> [as <alias>]` | `use <mod> [as <alias>]` or `from "<ns>" use ...` | [Modular Architecture](./modules.md) |
| `#ARG: "<key>" as <var>` | `arg <var>: <type> [= <default>]` | [Variables & Scoping](./variables.md) |
| `#VAR: <name>` / `#TSVAR` | `let <name> = <<<...>>>` or multi-line strings | [Syntax Specification](./syntax.md) |
| `#IF: <condition>` / `#END` | `if <condition> { ... }` / `elif` / `else` | [Control Flow & Matching](./conditions.md) |
| `#FOR: <item> in <list>` | `for <item> in <collection> { ... }` | [Control Flow & Matching](./conditions.md) |
| `#BEGIN: "<file>"` / `#END` | `file "<path>", (doc, err) -> { ... }` | [Multi-View Transformations](./transformations.md) |
| `#SELECT: "<path>"` | `dispatch "<path>", [<pattern>]` or `Path` | [Hierarchical Orchestration](./orchestration.md) |
| `#LOG: "<color>..."` | `std.info(...)`, `std.success(...)`, `std.error(...)` | [Standard Library](./stdlib.md) |
| `#EXEC: "<cmd>"` | `cmd.exec("<cmd>")` | [Standard Library](./stdlib.md) |
| `#EXIT: <code>` | `sys.exit(<code>)` | [Standard Library](./stdlib.md) |

---

## Comparative Example

### Legacy Syntax (v1):
```xru
#USE: utils as U
#ARG: "--env" as ENV

#IF: {ENV} == "prod"
    #LOG: "<cyan>[INFO]</> Production deployment started"
    #BEGIN: "config.json"
        SET debug false
    #END
#END
```

### Modern Syntax (XRU 2.0):
```xru
use std
arg env: string = "development"

if env == "production" {
    std.info("Production deployment started")
    file "config.json", (doc) -> {
        doc.setJson("debug", false)
    }
}
```

For detailed guides, refer to the **[Documentation Index](./README.md)**.
