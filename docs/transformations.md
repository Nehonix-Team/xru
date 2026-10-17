# Multi-View File Transformations (`file`)

The core capability of XRU is its multi-view surgical document manipulation engine.

Unlike naive text replacement tools, XRU operates with a synchronized in-memory **`Document`** entity that simultaneously reflects four integrated perspectives: a raw string view, a linewise indexed view, a structured JSON tree view, and semantic anchor algorithms designed for syntax-aware code manipulation.

---

## 1. The `file` Statement

The `file` statement accepts a target file path (literal, variable, or interpolated expression `$f{...}`) and a lambda expression receiving a `doc` (`Document`) handle:

```xru
file "package.json", (doc, err) -> {
    if err != nil {
        std.error("Failed to access file: $f{err}")
        return
    }

    doc.setJson("version", "2.0.0")
}
```

### Supported Signatures
- **Resilient Mode (`doc, err`)**: If the target file is missing or unreadable, the failure reason is supplied in `err`, allowing conditional recovery.
- **Strict Mode (`doc`)**: When `err` is omitted, reading errors immediately halt execution under the Fail-Fast policy.

### Atomic Auto-Persistence
Upon successful completion of the closure:
- If the document underwent modifications (`doc.isModified == true`), the runtime commits the updated content to disk atomically.
- Manual sync can also be forced at any point using `doc.save()`.

---

## 2. Integrated Document Views

```
                  ┌─────────────────────────────────┐
                  │          doc (Document)         │
                  └────────────────┬────────────────┘
         ┌─────────────────┬───────┴─────────┬──────────────────┐
         ▼                 ▼                 ▼                  ▼
   [ Raw View ]      [ Line View ]     [ JSON View ]      [ Semantic View ]
  .content, .size   .lines, .getLine  .getJson, .setJson  .replaceFuzzy
  .append, .prepend .setLine, .remove .removeJson         .removeBalanced
```

---

## 3. View 1: Raw String

Ideal for global replacements or appending/prepending content.

| Property / Method | Signature | Description |
| :--- | :--- | :--- |
| **`doc.content`** | `string` | Full textual representation of the document |
| **`doc.path`** | `string` | Absolute path of the targeted file |
| **`doc.getContent()`** | `() -> string` | Retrieves the complete string payload |
| **`doc.setContent(text)`** | `(text: string) -> bool` | Overwrites the entire file content |
| **`doc.append(text)`** | `(text: string) -> bool` | Appends text to the end of the file |
| **`doc.prepend(text)`** | `(text: string) -> bool` | Prepends text to the beginning of the file |
| **`doc.contains(str)`** | `(str: string) -> bool` | Verifies presence of a substring |
| **`doc.size()`** | `() -> number` | Current character count of the document |

#### Example:
```xru
file "src/main.ts", (doc) -> {
    if !doc.contains("import 'reflect-metadata'") {
        doc.prepend("import 'reflect-metadata';\n")
    }
}
```

---

## 4. View 2: Linewise Indexed

Provides direct inspection and mutation based on line numbers (1-indexed).

| Method | Signature | Description |
| :--- | :--- | :--- |
| **`doc.lines`** | `list[string]` | Array representation of all lines |
| **`doc.getLines()`** | `() -> list[string]` | Retrieves the line array |
| **`doc.getLine(num)`** | `(lineNum: number) -> string` | Reads line at 1-based index `num` |
| **`doc.setLine(num, text)`** | `(lineNum: number, text: string) -> bool` | Overwrites content at line `num` |
| **`doc.removeLine(num)`** | `(lineNum: number) -> bool` | Removes line `num` |
| **`doc.removeLines(start, end)`** | `(start: number, end: number) -> bool` | Removes inclusive range of lines |

#### Example:
```xru
file ".env", (doc) -> {
    doc.setLine(3, "DATABASE_URL=postgres://user:pass@localhost:5432/db")
}
```

---

## 5. View 3: Structured JSON

Inspects and updates JSON manifests (e.g., `package.json`, `tsconfig.json`) while preserving formatting and indentation.

| Method | Signature | Description |
| :--- | :--- | :--- |
| **`doc.getJson(keyPath)`** | `(keyPath: string) -> any` | Extracts value via dot-notated key path |
| **`doc.setJson(keyPath, val)`** | `(keyPath: string, val: any) -> bool` | Inserts or updates key at path |
| **`doc.removeJson(keyPath)`** | `(keyPath: string) -> bool` | Deletes property at path |

#### Example:
```xru
file "package.json", (doc) -> {
    doc.setJson("name", "core-service")
    doc.setJson("scripts.dev", "xfpm run dev")
    doc.removeJson("dependencies.lodash")
}
```

---

## 6. View 4: Semantic Algorithms (Fuzzy & Bracket Balancing)

Source code modifications often fail when target files differ in whitespace, tabs, or newlines introduced by code formatters (Prettier, rustfmt, gofmt). Semantic algorithms provide robust structural matching.

| Method | Signature | Description |
| :--- | :--- | :--- |
| **`doc.replaceFuzzy(old, new)`** | `(old: string, new: string) -> bool` | Replaces target code ignoring whitespace and newline variances |
| **`doc.removeFuzzy(target)`** | `(target: string) -> bool` | Deletes code matching target with whitespace tolerance |
| **`doc.insertAfter(anchor, text)`** | `(anchor: string, text: string) -> bool` | Inserts text immediately following anchor occurrence |
| **`doc.insertBefore(anchor, text)`** | `(anchor: string, text: string) -> bool` | Inserts text immediately preceding anchor occurrence |
| **`doc.removeBetween(start, end, inc?)`** | `(start: string, end: string, inc?: bool) -> bool` | Deletes all content bounded by two markers |
| **`doc.removeBalanced(anchor)`** | `(anchor: string, open?: string, close?: string) -> bool` | **Bracket Balancing**: locates anchor, tracks matching `{` and `}` while ignoring strings, and removes the balanced block |

### Bracket Balancing (`removeBalanced`) in Practice

Removing a function or class using regex frequently corrupts code when nested blocks are encountered.

`doc.removeBalanced` inspects the balanced token hierarchy:

```typescript
// Target file: src/services.ts
export class ObsoleteService {
    public run() {
        if (true) {
            console.log("Nested block");
        }
    }
}

export class ActiveService { ... }
```

In XRU:
```xru
file "src/services.ts", (doc) -> {
    doc.removeBalanced("export class ObsoleteService")
}
```
The engine resolves opening and closing delimiter depth, ignores braces enclosed in strings or comments, and purges the balanced block completely.
