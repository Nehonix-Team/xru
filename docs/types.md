# Type System & Interfaces

XRU 2.0 features a strict, predictive type system designed to enforce the **Fail-Fast & Zero-Side-Effect on Error** principle. Type assertions are validated during static checking (`checker`) and enforced by the runtime evaluator (`evaluator`).

---

## 1. Primitive Data Types

| Type | Description | Syntax & Examples |
| :--- | :--- | :--- |
| **`string`** | UTF-8 encoded text sequence | `"text"`, `'literal'`, `<<<HEREDOC>>>` |
| **`number`** | Integer and floating-point numeric values | `42`, `3000`, `3.14`, `-10` |
| **`bool`** | Boolean truth values | `true`, `false` |
| **`duration`** | Strongly typed time duration | `250ms`, `30s`, `5m`, `2h`, `7d` |
| **`nil`** / `null` | Absence of value | `nil` |
| **`void`** | Return type for functions yielding no value | `func notify() -> void` |
| **`any`** | Dynamic type bypass for unconstrained variables | `arg payload: any` |

### The `duration` Type
Durations are first-class primitives preventing ambiguity between milliseconds, seconds, and hours:
```xru
let timeout: duration = 30s
let pollInterval: duration = 500ms
let leaseTime: duration = 24h
```
Values assigned to `duration` variables that fail to match the pattern `[0-9]+(ms|s|m|h|d)` trigger an immediate static error (`ERR_INVALID_DURATION`).

---

## 2. Collections & Composite Types

### 2.1 Lists (`list` / `list[T]`)
Lists are ordered dynamic arrays. They can be constrained by element type (`list[string]`, `list[number]`) or unconstrained (`list`):

```xru
let tags: list[string] = ["api", "auth", "v2"]
let numbers: list[number] = [10, 20, 30]

tags.push("security")
let count = tags.len() // 4
let first = tags.get(0) // "api"
```

### 2.2 Associative Maps (`map`)
Maps represent key-value dictionaries with string keys:
```xru
let options = map {
    host: "0.0.0.0",
    port: 8080,
    debug: true
}
```

### 2.3 Numerical Ranges (`range`)
Represents an iterable integer range (`start..end`):
```xru
let steps: range = 1..5
for i in steps {
    std.println("Step $f{i}")
}
```

---

## 3. Asynchronous Task Handles (`task` / `task[T]`)

Expressions evaluated with the `with` operator return a typed task handle `task[T]`:

```xru
let worker: task[string] = with net.fetch("https://api.example.com/status")

// Task receiver methods
let isReady: bool = worker.isDone()   // Non-blocking query
let response: string = worker.wait() // Blocking synchronization
worker.cancel()                      // Task interruption
```

---

## 4. Interfaces & Data Schemas (`interface`)

XRU maintains a formal separation between **method contracts** and **in-memory data schemas**:

### 4.1 Unit Method Contracts (`interface Name { ... }`)
Defines functional method signatures implemented by units (`unit ... includes ...`):
```xru
interface StorageDriver {
    func read(path: string) -> string
    func write(path: string, content: string) -> bool
}
```

### 4.2 In-Memory Data Schemas (`interface Name @map { ... }`)
The `@map` modifier declares an in-memory structural data schema:

```xru
interface DatabaseConfig @map {
    host string
    port number
    user string
    password? string        // The '?' modifier marks an optional field
    timeout duration
}
```

#### Strict Interface Syntax Rules:
1. **Colons `:` are prohibited in field definitions**: Field names and types must be separated exclusively by whitespace (`host string`, `port number`). Including colons (e.g., `host: string`) results in a compile-time syntax error.
2. **Duplicate field collision prevention (`ERR_DUPLICATE_FIELD`)**: Declaring duplicate fields (including case-insensitive collisions such as `host` and `Host`) triggers an immediate static error.
3. **Instantiations require `@map`**: Objects conforming to `@map` interfaces must be initialized using `@map { ... }`.

```xru
const dbConfig: DatabaseConfig = @map {
    host: "10.0.0.1",
    port: 5432,
    user: "postgres",
    timeout: 5s
}
```

### 4.3 Nested Interfaces
Interfaces can be nested to model complex data trees:

```xru
interface SslOptions @map {
    enabled bool
    certPath? string
}

interface WebServerConfig @map {
    domain string
    ssl SslOptions
}

const server: WebServerConfig = @map {
    domain: "api.domain.com",
    ssl: @map {
        enabled: true,
        certPath: "/etc/ssl/cert.pem"
    }
}
```

---

## 5. Indexed Types (`Interface["field"]`)

To decouple signatures from duplicate type declarations, field types can be indexed directly from an interface:

```xru
interface AppMeta @map {
    version string
    buildNumber number
}

// Return type is resolved directly to AppMeta["version"] (string)
func getAppVersion() -> AppMeta["version"] {
    return "2.0.0"
}
```

In language servers and IDEs, typing `AppMeta["` prompts autocomplete proposals for all declared fields.

---

## 6. Union Types & Optionals

XRU supports type unions and literal constraints using the `or` keyword:

```xru
interface DeployTarget @map {
    stage "development" or "staging" or "production"
    concurrency 1 or 2 or 4 or 8
    fallbackUrl string or nil
}

func findUser(id: string) -> string? {
    if id == "" {
        return nil
    }
    return "User $f{id}"
}
```
