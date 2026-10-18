# Command Line Interface (CLI)

The `xru` binary provides an operational toolset for script execution, static type checking, rule packaging, and language server hosting.

---

## 1. Script Execution: `xru run`

Executes a target `.xru` file within its mapped module context:

```bash
xru run <file.xru> [flags...]
```

### 1.1 Dynamic CLI Arguments (`arg`)
When a script declares arguments with `arg`:

```xru
arg env: string = "development"
arg port: number = 8080
arg appName: string // Required
```

Values are supplied via `--flag=value` or `--flag value` syntax:

```bash
xru run deploy.xru --appName=MyService --port=3000 --env=production
```

### 1.2 Fail-Fast Argument Validation
Arguments are verified before runtime startup:
- **Missing Required Argument (`ERR_MISSING_ARG`)**: If a required argument lacks input, execution halts immediately:
  ```text
  [RUNTIME ERROR] Missing required argument '--appName'.
  ```
- **Type Coercion Failure (`ERR_ARG_COERCION`)**: Supplying an incompatible value (e.g., `--port text` for a `number`) generates an immediate error.

---

## 2. Static Analysis: Dry-Run Mode (`--check`)

Verifies syntax, type safety, and `_mod.xru` configuration without modifying files on disk:

```bash
xru run script.xru --check
```

Validation failures output structured diagnostics:
```text
[VALIDATION ERROR] in src/main.xru:
  • src/main.xru:14:1: [ERR_UNKNOWN_METHOD] Method 'llog' not found on unit 'std'
  • src/main.xru:18:5: [ERR_TYPE_MISMATCH] Cannot assign type 'string' to variable 'port' of type 'number'
```

---

## 3. Rule Packaging: `xru build`

Packages rules into a standalone bundle:

```bash
# Compile and bundle rules
xru build script.xru --bundle

# Export distribution bundle
xru build script.xru --out=./dist/bundle.xpack
```

---

## 4. Language Server Protocol: `xru lsp`

Launches the native LSP daemon over standard I/O (stdin/stdout). Compatible with editors and IDEs (VS Code, Neovim, Zed) to provide:
- Real-time diagnostics.
- Contextual code completion for symbols, units, methods, and `@map` schemas.
- Hover documentation showing type signatures and documentation.
- Automated dependency imports.

```bash
xru lsp
```

---

## 5. Global Flags

| Flag | Description |
| :--- | :--- |
| **`--version`**, **`-v`** | Prints binary version, operating system, and architecture |
| **`--help`**, **`-h`** | Prints command syntax and flag options |
| **`--verbose`** | Enables verbose logging output during evaluation |
