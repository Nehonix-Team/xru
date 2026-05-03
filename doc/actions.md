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
Injects code at a dynamic marker. The prefix (`TS`, `JS`, `GO`, etc.) is optional but helps with syntax highlighting in some editors.

#### Example Markers (Target Files)
```typescript
// --> {{imports}}
import { base } from './base';

/* xru: logic */
const x = 10;

# xfpm: config
VERSION=1.0
```

#### Example Usage (XRU)
```xru
#BEGIN: "main.ts"
  @TSINJECT: imports
    import { Auth } from './auth';
    import { Store } from './store';
  @END

  @INJECT: logic
    console.log("Injected logic running");
  @END
#END
```

