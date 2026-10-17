# Concurrency & Error Handling

XRU 2.0 includes native language-level concurrency powered by lightweight Go goroutines and declarative inline error recovery mechanics.

---

## 1. Native Concurrency (`with`)

The `with` keyword evaluates an expression asynchronously in a background coroutine without blocking execution:

```xru
use net as Net
use std

// Dispatches network fetch in background
let downloadTask: task[string] = with Net.fetch("https://api.github.com/repos/Nehonix-Team/xru/releases")
```

### 1.1 Task Lifecycles & `task[T]`
Any expression evaluated via `with` returns a `task[T]` descriptor handle, where `T` resolves to the underlying return type.

### 1.2 Methods on `task`

| Method | Signature | Description |
| :--- | :--- | :--- |
| **`.isDone()`** | `() -> bool` | Non-blocking inspection returning `true` when the task completes |
| **`.wait()`** | `() -> T` | Blocking synchronization returning the evaluated typed payload |
| **`.cancel()`** | `() -> bool` | Signals early cancellation of the background task |

### 1.3 Polling Example
```xru
use std
use sys

let worker = with computeHeavyTask()

for _ in 1..10 {
    if worker.isDone() {
        std.success("Task completed early")
        return
    }
    std.print(".")
    sys.sleep(200ms)
}

let result = worker.wait()
std.println("Result: $f{result}")
```

---

## 2. Inline Error Interception: `catch ... or ...`

XRU provides a concise single-line syntax for handling expression evaluations:

```xru
let data = riskyOperation() catch err or fallbackValue
```

### 2.1 Mechanics
1. The runtime evaluates `riskyOperation()`.
2. If the operation succeeds without errors:
   - `data` receives the evaluation result.
   - The error variable `err` is initialized to `nil`.
3. If an error is raised:
   - The error object is bound to `err`.
   - `data` assumes the `fallbackValue`.
   - Program execution proceeds cleanly without an unhandled exception.

### 2.2 Network Fallback Example
```xru
use net as Net
use std

let content = Net.fetch("https://config.service.local/settings.json") catch netErr or "{}"

if netErr != nil {
    std.warn("Network lookup failed ($f{netErr}), using fallback config")
}

std.println("Loaded payload: $f{content}")
```

---

## 3. Error Handling in File Operations

When using `file`, supplying the optional `err` argument in the closure prevents runtime halts on missing files:

```xru
file "optional-settings.json", (doc, err) -> {
    if err != nil {
        std.info("Optional file absent, skipping transformation")
        return
    }

    doc.setJson("lastUpdated", "2026-10-17")
}
```

When `err` is omitted (i.e. signature `(doc) -> { ... }`), missing target files trigger a static Fail-Fast halt.
