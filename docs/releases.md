# Release Notes

## [v2.0.0] - 2026-10-17

XRU 2.0.0 represents a major architectural redesign of the language, establishing it as a deterministic, strongly typed multi-view code transformation and orchestration engine.

### Key Changes

#### 1. Predictive Static Type System
- Introduction of static compile-time type verification (`checker`) and runtime type enforcement (`evaluator`).
- Primitive types support (`string`, `number`, `bool`, `duration`, `nil`, `void`, `any`).
- Introduction of the native **`duration`** type (`500ms`, `30s`, `5m`, `2h`, `7d`).
- Generic collection typing via **`list[T]`**, alongside **`map`** and numeric **`range`** (`1..10`).
- Typed asynchronous task handles via **`task[T]`**.

#### 2. Separation of Declaration (`=`) and Mutation (`=:`)
- The `=` operator is reserved strictly for initial variable declarations (`let`, `const`, `declare`, `global`).
- Reassigning mutable variables requires the dedicated **`=:`** operator.
- The **`const`** keyword strictly locks bindings against mutations, verified at compile-time (`ERR_CONST_MUTATION`).

#### 3. Collision-Free String Interpolation (`$f{...}`)
- Deprecation of legacy `{...}` and `${...}` interpolation patterns.
- String literals are inert by default, preventing collisions with foreign code (TypeScript, Bash, JSON).
- The explicit **`$f{...}`** token serves as the sole evaluation vector for dynamic expressions.

#### 4. Multi-View Document Engine (`file`)
- Replacement of legacy file directives with the closure-based `file "path", (doc, err) -> { ... }` construct.
- Synchronized multi-view `Document` instance:
  - **Raw View**: Full-string access (`.content`, `.append`, `.prepend`).
  - **Line View**: Line-indexed inspection and replacement (`.lines`, `.getLine`, `.removeLines`).
  - **JSON View**: Structured dot-notated key mutation (`.getJson`, `.setJson`, `.removeJson`) preserving document formatting.
  - **Semantic View**: Whitespace-tolerant replacements (`.replaceFuzzy`, `.removeFuzzy`) and **Bracket Balancing** (`.removeBalanced`) for syntax-aware code removal.
- Atomic automatic disk persistence upon clean closure exit.

#### 5. User Units (`unit`) vs Native Built-in Modules
- User prototype units with encapsulated state (`declare &prop`, internal scoping `&Unit::method()` / `&self::method()`, and mandatory instantiation via `@UnitName()`).
- Standard library modules (`std`, `sys`, `fs`, `net`, `crypto`, `date`, `math`, `json`, `cmd`, `path`) converted to static singletons called directly without `@`.

#### 6. Interface Contracts & Data Schemas (`@map`)
- `interface Name { ... }` for unit method protocols.
- `interface Name @map { ... }` for in-memory data schemas.
- Strict syntax: Colons `:` prohibited in interface field definitions (whitespace separator required).
- Indexed type access: `Interface["field"]`.

#### 7. Sandboxed Hierarchical Orchestration (`dispatch`)
- Replaced legacy dispatch directives with the first-class `dispatch <path>, [<pattern>]` statement.
- Injected sandbox primitives: `Keep`, `Drop`, `Path`, `Env`.
- Permanent sanitization protecting active `.xru` descriptor files from deletion during pruning operations.

#### 8. Hermetic Module System (`_mod.xru`)
- Namespace-based dependency resolution declared via `_mod.xru`.
- Import side-effect isolation: Only declarations are evaluated; standalone top-level script statements do not execute upon import.
- Automatic export type inference for IDE and LSP tooling.

---

## [v0.1.8] and Prior (Legacy v1)
*(Historical releases operating under the legacy v1 patcher syntax with `#` directives)*
