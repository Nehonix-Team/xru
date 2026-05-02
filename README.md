# XRU: XyPriss Rule Unit — Syntax Documentation

XRU is a Domain-Specific Language (DSL) designed for **Structured Text Transformation**. Unlike traditional patchers that rely on strict JSON round-tripping, the XRU engine operates as a **Structured Text Patcher (STP)**, preserving formatting, comments, and non-standard syntaxes while applying complex mutations.

---

## 1. Scoping Directives

Directives that define which file(s) are being targeted.

### `#BEGIN:<path>` / `#END`
Opens a transformation block for an existing file.
- **`<path>`**: Relative path to the file from the project root.
- If the file does not exist, the engine skips the block with a warning.

### `#CREATE:<path>` / `#END`
Creates a new file with the provided static content.
- Everything between the opening and closing tag is treated as raw file content.

### Global Rules
Any `@TSINJECT` or `&` directive placed outside of a `#BEGIN` block is treated as a **Global Rule**.
- Global rules are applied to all matching source files (typically `.ts`, `.json`, `.jsonc`, `.md`).

---

## 2. Action Directives

### `@TSINJECT:<key>` / `@END`
Injects a block of code into a file at a specific marker.
- **Marker Syntax**: Looks for `// xfpm: {{key}}` or `// xfpm: key` in the target file.
- The marker line is replaced by the provided code block.
- **Multi-line**: Supports any number of lines between `@TSINJECT` and `@END`.

### `&merge:` (or `&add:`)
Performs a deep-merge of a structured object into the target file.
- **Logic**:
  - If a key exists and the value is an object, it recurses into the existing braces `{ ... }`.
  - If a key exists and the value is a literal (string/number), it replaces the existing value.
  - If a key is missing, it is injected before the closing brace of the current scope.
- **Preservation**: It does not re-parse the whole file. It only modifies the specific lines affected by the merge.

### `&rm:`
Removes keys or values from a structured file.
- Can take a list of keys `[ "key1", "key2" ]` or a nested object to remove specific branches.

### `&rp-k:` (or `&rp-0:`)
Renames a key while preserving its value.
- Syntax: `&rp-k: { "old_name": "new_name" }`

### `&rp-v:` (or `&rp-1:`)
Replaces the value of an existing key.

---

## 3. Language Features

The XRU language is designed to be developer-friendly and "relaxed" compared to strict JSON.

### Unquoted Keys
You can write structures without quoting every key, making the rules more readable.
```xru
&merge: {
  $vars: {
    secret: "xyz"
  }
}
```

### XyPriss Variables
XRU natively supports and preserves XyPriss variable syntax. It will never corrupt or try to "resolve" these strings during the patching phase.
- Example: `"__name__": "&(pkg).name"`

### Comments and Formatting
- **In Rules**: You can use `//` comments inside your `&merge` or `&rm` blocks.
- **In Target Files**: The engine respects and preserves all comments, blank lines, and custom indentation in the files it patches.

### Brace Counting
The engine uses intelligent brace-depth tracking. This allows it to find the correct insertion point even in complex nested structures, regardless of whether the file is valid JSON or uses non-standard extensions (like JSONC).

---

## 4. Example Rule File

```xru
#BEGIN:package.json
&merge: {
  scripts: {
    "dev:xms": "xfpm run dev --mode xms"
  }
}
#END

#BEGIN:xypriss.config.ts
@TSINJECT:{{SECURITY_CONFIG}}
  security: {
    terminalOnly: {
      enable: true,
      allowedTools: ["postman", "curl"]
    }
  },
@END
#END
```
