# Migration Guide: Legacy Actions to Document Operations

> [!NOTE]
> In **XRU 2.0**, legacy free-floating text actions (`SET`, `MERGE`, `DELETE`, `APPEND`, `PREPEND`, `INJECT`) have been superseded by the **`Document` (`doc`)** object available within `file` closures.

---

## Equivalency Table: v1 Actions to v2.0 `Document` Methods

| Legacy Action (v1) | Modern XRU 2.0 `doc` Method | Description |
| :--- | :--- | :--- |
| `SET key val` | `doc.setJson("key", val)` | Updates or inserts a structured JSON property |
| `MERGE key { ... }` | `doc.setJson("key.sub", val)` | Updates a nested JSON key path |
| `DELETE key` | `doc.removeJson("key")` | Deletes a property from a JSON document |
| `APPEND "text"` / `>>` | `doc.append("text")` | Appends text to the end of the file |
| `PREPEND "text"` / `++` | `doc.prepend("text")` | Prepends text to the start of the file |
| `REPLACE "a" "b"` | `doc.replaceFuzzy("a", "b")` | Replaces target code with whitespace tolerance |
| `@TSINJECT` / `@INJECT` | `doc.insertAfter(...)` / `insertBefore(...)` | Inserts code relative to a target anchor |
| `REMOVE_BLOCK` | `doc.removeBalanced(anchor)` | Removes code block using bracket balancing |

---

## Comparative Example

### Legacy Syntax (v1):
```xru
#BEGIN: "package.json"
    SET version "2.0.0"
    MERGE scripts {
        "build": "xfpm run build"
    }
#END
```

### Modern Syntax (XRU 2.0):
```xru
file "package.json", (doc) -> {
    doc.setJson("version", "2.0.0")
    doc.setJson("scripts.build", "xfpm run build")
}
```

For full details on file manipulation, refer to **[Multi-View File Transformations (`file`)](./transformations.md)**.
