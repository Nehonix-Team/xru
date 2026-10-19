# XRU (XyPriss Rule Unit)

[![Specification Version](https://img.shields.io/badge/version-2.0.0-0969da.svg)](./docs/releases.md)
[![License](https://img.shields.io/badge/license-NOSL-gray.svg)](LICENSE)
[![Architecture](https://img.shields.io/badge/architecture-Strict%20Types%20%7C%20Fail--Safe-2da44e.svg)](./docs/types.md)

XRU (XyPriss Rule Unit) is a strongly typed, deterministic domain-specific language designed for surgical text and source code transformation, hierarchical rule orchestration, and automated project scaffolding within the Nehonix ecosystem.

Operating as a synchronized Multi-View Document Engine, XRU executes structured mutations across various source file formats (such as TypeScript, Go, Rust, JSON, and YAML) while strictly preserving indentation, line endings, comments, and project styling standards.

---

## Architectural Principles

- **Predictive Static Typing**: Full compile-time verification via the static type checker and runtime enforcement within the evaluator. The engine enforces the *Fail-Fast & Zero-Side-Effect on Error* guarantee: no disk mutations are applied if parsing or semantic analysis detects an anomaly.
- **Multi-View Surgical Document Engine (`file`)**: Exposes an integrated `Document` instance synchronizing four continuous views: Raw Text view, Linewise indexed view, Structured JSON tree view, and Semantic Anchor view (featuring Fuzzy Matching and bracket-balanced block isolation).
- **Native Concurrency (`with` / `task[T]`)**: Non-blocking delegation of operations returning strongly typed task handles supporting non-blocking status inspection (`.isDone()`), typed awaiting (`.wait()`), and cancellation (`.cancel()`).
- **Controlled Variable Mutation (`=:`)**: Formal distinction between initial declaration (`=`) and reassignment (`=:`). Constants (`const`) reject any mutation attempt during compile-time static analysis.
- **Unambiguous Formatting Interpolation (`$f{...}`)**: String literals are inert by default, preventing collisions with foreign code constructs (such as TypeScript destructuring `{ ... }` or shell variables `${...}`).
- **Sandboxed Hierarchical Orchestration (`dispatch`)**: Recursive directory traversal executing nested descriptor rules within isolated environments, providing injected execution primitives (`Keep`, `Drop`, `Path`, `Env`) alongside descriptor protection (`.xru` sanitization).
- **Hermetic Module Architecture (`_mod.xru`)**: Explicit namespace-based dependency resolution, zero side-effects on module import, and automated export type inference for tooling integration.

---

## Installation

XRU can be installed via the Nehonix XFPM package manager or standalone through the official distribution channels.

### Unix Platforms (Linux / macOS)
```bash
curl -sL https://dll.nehonix.com/xru/install.sh | bash
```

### Windows Platforms (PowerShell)
```powershell
irm https://dll.nehonix.com/xru/install.ps1 | iex
```

### Verification
```bash
xru --version
```

---

## Implementation Example

The following script illustrates a standard orchestration routine:

```xru
use std
use fs
use json

// Typed command-line arguments
arg env: string = "production"
arg port: number = 8080
arg appName: string

// Strict memory data contracts (@map)
interface ServerConfig @map {
    host string
    port number
    mode "development" or "production"
}

const config: ServerConfig = @map {
    host: "127.0.0.1",
    port: port,
    mode: env
}

std.info("Configuring $f{appName} under $f{env} profile...")

// Surgical transformation of a JSON manifest
file "package.json", (doc, err) -> {
    if err != nil {
        std.error("Failed to read package.json: $f{err}")
        return
    }

    doc.setJson("name", appName)
    doc.setJson("config.port", config.port)
    doc.setJson("config.mode", config.mode)
    std.success("package.json updated.")
}

// Formatting-tolerant source code modification
file "src/server.ts", (doc) -> {
    doc.replaceFuzzy("const PORT = 3000", "const PORT = $f{port}")

    if !doc.contains("import { logger }") {
        doc.prepend("import { logger } from './utils/logger'\n")
    }
}

// Asynchronous execution with structured fallback
let healthTask: task[string] = with net.fetch("https://status.nehonix.com/ping")
let status = healthTask.wait() catch err or "offline"

std.println("Service status: $f{status}")
```

Executing the routine:
```bash
xru run patch.xru --appName="core-service" --port=3000 --env=production
```

---

## Documentation Index

The complete technical specification and developer guides are organized under the `docs/` directory:

| Document | Description |
| :--- | :--- |
| **[Documentation Index](./docs/README.md)** | Architectural overview and central table of contents |
| **[Syntax Specification](./docs/syntax.md)** | Grammar, lexical tokens, mutations (`let`, `const`, `=:`), string interpolation |
| **[Type System & Interfaces](./docs/types.md)** | Primitive types, collections, duration types, `@map` interfaces, indexed types |
| **[Units & Built-in Modules](./docs/units.md)** | Prototype units (`unit`) vs native static singletons (`Net`, `Fs`, `Crypto`) |
| **[Multi-View File Transformations](./docs/transformations.md)** | The `Document` model (Raw, Linewise, JSON, Fuzzy and Balanced Bracket matching) |
| **[Concurrency & Error Handling](./docs/async_and_errors.md)** | Non-blocking execution (`with`), `task[T]` handles, and inline `catch ... or` recovery |
| **[Control Flow & Matching](./docs/conditions.md)** | Conditional branches `if` / `elif` / `else`, iterations `for ... in`, and `match` constructs |
| **[Hierarchical Orchestration](./docs/orchestration.md)** | Directory dispatching, sandboxing, and injected primitives (`Keep`, `Drop`) |
| **[Modular Architecture](./docs/modules.md)** | Hermetic project configuration (`_mod.xru`), namespaces, and isolated imports |
| **[Standard Library Reference](./docs/stdlib.md)** | Specifications for native modules (`std`, `sys`, `fs`, `net`, `crypto`, `date`, etc.) |
| **[Command Line Interface (CLI)](./docs/cli.md)** | Execution commands (`run`, `build`), static check mode, and LSP server options |
| **[End-to-End Implementation Guide](./docs/tutorial.md)** | Step-by-step implementation walkthrough for project scaffolding and patching |
| **[Release Notes](./docs/releases.md)** | Technical changelog and architectural evolution history |

---

## Command Line Interface

```bash
# Execute a script with typed flags
xru run script.xru --appName=Core --port=8080

# Perform static verification without side effects (dry run)
xru run script.xru --check

# Package rules into a redistributable bundle
xru build script.xru --bundle

# Launch the native Language Server Protocol (LSP) daemon
xru lsp
```

---

## Development Environment Support

XRU includes a reference Language Server Protocol implementation (`pkg/lsp`) providing:
- Real-time diagnostics with static error detection.
- Contextual code completion for symbols, units, methods, and `@map` schema properties.
- Automated dependency resolution and top-level import injection.
- Hover documentation detailing type signatures and member specifications.

---

## License

Distributed under the **NOSL** License. Refer to [LICENSE](LICENSE) for terms and conditions.  
Nehonix Organization.
