# XRU Actions

Actions are the operations performed **inside** scoping blocks (`#BEGIN`, `#CREATE`) or under `#GLOBAL`. They define how the text should be modified.

> [!TIP]
> While structural directives (`#`) are anchored at Column 0, actions should be **indented** for readability to visualize the block scope.

---

## 1. Quick Symbols
Best for top-level modifications or whole-object merges. Barewords are supported for values.

| Symbol | Action | Description |
| :--- | :--- | :--- |
| `++` | **MERGE** | Deep merge an object into the target. |
| `--` | **REMOVE** | Delete specific keys or array items. |
| `>>` | **RENAME** | Rename object keys. |
| `<<` | **APPEND** | Add an item to an array. |
| `~~` | **REGEX** | Search and Replace via regular expressions. |

### Examples
```xru
#BEGIN: "settings.json"
  ++ { "theme": "dark", "zoom": 1.2 }
  -- "experimental_features"
  >> "old_key" "new_key"
  << "plugins" "git-integration"
  ~~ "http://localhost:3000" "https://api.myapp.com"
#END
```

---

## 2. Path-Targeted Keywords
Best for precise deep-patching without repeating the entire file structure.

### `SET <path> <value>`
Overwrites or creates a value at a specific path. 
```xru
#BEGIN: "config.json"
  SET version "1.2.3"
  SET ui.colors.primary "#ff0000"
#END
```

### `MERGE <path> <object>`
Performs a deep merge at a specific nested path.
```xru
#BEGIN: "package.json"
  MERGE scripts { "start": "node index.js", "test": "jest" }
#END
```

### `REMOVE <path>`
Deletes a specific key or branch.
```xru
#BEGIN: "config.yaml"
  REMOVE metadata.labels.obsolete
#END
```

### `PUSH <path> <value>`
Appends a value to an array at a specific path.
```xru
#BEGIN: "tsconfig.json"
  PUSH compilerOptions.lib "esnext"
#END
```

---

## 3. Code Injections
Used for injecting raw code at specific markers in source files. XRU uses a **Universal Marker Detection** system.

### `@*INJECT: <key>` / `@END`
Injects code at a dynamic marker. The prefix (`TS`, `JS`, `GO`, `HTML`, `CSS`, `JSON`, etc.) is optional but enables **Dynamic Language Injection** for syntax highlighting in supported editors.

#### Injection Logic
1. The engine searches for a marker in the target file.
2. The marker key can be prefixed with `xru:`, `xfpm:`, `@TSINJECT:`, or even just a keyword like `INJECT:`.
3. If the marker is found, the content between the action and `@END` is injected.
4. **Recursive Orchestration**: Actions inside `#BEGIN` or `#CREATE` are now executed recursively, allowing for nested injections.

#### Marker Detection Examples (Target Files)
```typescript
// xru: ROUTERS_IMPORT
import { base } from './base';

// @TSINJECT: ROUTERS_USE
const app = express();
```

#### Inception Integration
You can use **Inception Tags** (`<# ... >`) inside an injection block to generate dynamic code:

```xru
#BEGIN: "router/index.ts"
  @TSINJECT: ROUTERS_USE
    <#FOR: S in {SERVERS}>
      router.use("/{S}", {S}Router);
    <#END>
  @END
#END
```

---

## 4. Output Capturing
### `#LOG: <message>`
Writes a message to the capture buffer. When used inside an **Inception Tag** `<# ... >`, the logged message is injected directly into the template output instead of being printed to the terminal.

