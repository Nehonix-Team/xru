# XRU 2.0 Technical Documentation

This repository houses the formal language specification and reference documentation for **XRU (XyPriss Rule Unit)**, developed by the Nehonix Organization for deterministic code transformation, static orchestration, and project maintenance.

---

## Documentation Structure

The documentation suite is structured into dedicated operational categories:

### 1. Language Fundamentals & Type System
- **[Syntax Specification](./syntax.md)**: Universal `{ ... }` block structure, lexical declarations (`let`, `const`, `global`, `arg`), the `=:` mutation operator, unambiguous `$f{...}` formatting interpolation, comments, and grammar rules.
- **[Type System & Interfaces](./types.md)**: Primitive data types, temporal units (`duration`), composite collections (`list[T]`, `map`, `range`), asynchronous handles (`task[T]`), in-memory data schemas (`interface ... @map`), and indexed types.
- **[Control Flow & Pattern Matching](./conditions.md)**: Conditional branching via `if` / `elif` / `else`, iterations with `for ... in` (collections, index unrolling, numerical ranges), and declarative matching via `match` / `case` / `default`.
- **[Units & Built-in Modules](./units.md)**: Formal differentiation between static system modules (singletons invoked directly without `@`) and user prototype definitions (`unit`, instantiated with `@`, encapsulating state with `declare &prop`, and resolving internal scopes with `&self::method()`).

### 2. File Engine & Concurrency
- **[Multi-View File Transformations (`file`)](./transformations.md)**: The `Document` object model exposed within the `file` block. Details the four synchronized operational views: Raw string, Linewise indexed, Structured JSON, and Semantic Anchor view (Fuzzy matching and Bracket Balancing algorithms).
- **[Concurrency & Error Handling](./async_and_errors.md)**: Asynchronous task execution with `with`, lifecycle of `task[T]` handles (`.wait()`, `.isDone()`, `.cancel()`), and inline error interception with `expr catch err or fallback`.
- **[Hierarchical Orchestration (`dispatch`)](./orchestration.md)**: Recursive directory dispatching, sandboxed sub-rule execution, injected environment primitives (`Keep`, `Drop`, `Path`, `Env`), and permanent descriptor sanitization.

### 3. Modular Architecture & Tooling
- **[Modular Architecture (`_mod.xru`)](./modules.md)**: Hermetic project mapping configuration, namespace resolution, module imports (`use`, `from "<ns>" use ...`), import side-effect isolation, and export type inference.
- **[Standard Library Reference](./stdlib.md)**: Full API specifications for integrated runtime modules (`std`, `sys`, `fs`, `net`, `crypto`, `date`, `math`, `json`, `cmd`, `path`) and built-in primitive receiver methods.
- **[Command Line Interface (CLI)](./cli.md)**: Execution with `xru run`, dry-run static checking with `--check`, packaging with `xru build`, command line argument parsing (`arg`), and LSP server daemon operation (`xru lsp`).
- **[Implementation Walkthrough](./tutorial.md)**: Comprehensive A-to-Z practical tutorial detailing project setup, typed configuration, and multi-file surgical patching.
- **[Release Notes](./releases.md)**: Historical changelog and technical release notes for version 2.0.0.

---

## Core Engineering Tenets

The XRU runtime operates under four architectural requirements:

### 1. Fail-Fast & Zero-Side-Effect on Error
Neither files nor project structures are partially mutated. If the compiler, static type checker, or evaluator encounters any syntax error, type violation, undefined reference, or unresolved dependency, execution halts immediately with structured diagnostics, leaving the host filesystem untouched.

### 2. Strict Decoupling of Declaration and Mutation
The `=` symbol is reserved strictly for initial variable bindings. Any subsequent reassignment requires the explicit mutation operator `=:`. Attempts to reassign constants declared via `const` are rejected during static analysis.

### 3. Syntax Invariance of String Literals
Because XRU operates directly on third-party codebases (TypeScript, JavaScript, Bash scripts, JSON/YAML configs), strings do not evaluate curly braces `{ ... }` or `${...}` sequences. Expressions are evaluated strictly through the explicit `$f{...}` construct, eliminating foreign syntax corruption.

### 4. Atomic Multi-View Persistence
The `file` primitive synchronizes multiple views of target files in memory. Disk persistence is executed atomically upon clean exit of the mutation block if and only if modifications were committed to the document instance.
