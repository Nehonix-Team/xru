# Units & Built-in Modules

XRU 2.0 enforces a strict architectural boundary between **native system modules** (static singletons integrated within the Go runtime) and **user prototype units** (structures encapsulating mutable internal state).

---

## 1. Architectural Comparison

| Characteristic | Built-in Native Modules (`Net`, `Crypto`, `Fs`...) | User Prototype Units (`unit Name { ... }`) |
| :--- | :--- | :--- |
| **Nature** | Static global singletons without instance state | Prototypes holding reassignable internal state |
| **Instantiation via `@`** | Prohibited (`ERR_CANNOT_INSTANTIATE_BUILTIN`) | Mandatory (`let obj = @UnitName()`) |
| **Direct Static Method Invocation** | Allowed (`Crypto.sha256(...)`, `Fs.read(...)`) | Prohibited (`ERR_CANNOT_CALL_STATIC`) |
| **Internal State** | None exposed to userland mutation | Defined via `declare &name = ...` |
| **Internal Invocation** | Native Go runtime calls | Explicit scopes `&Unit::method()` or `&self::method()` |

---

## 2. Built-in Native Modules (Static Singletons)

Modules provided directly by the runtime (`std`, `sys`, `fs`, `net`, `crypto`, `date`, `math`, `json`, `cmd`, `path`) are static singletons.

### 2.1 Direct Invocation
Methods are invoked directly via the module identifier (or an alias defined via `use`):

```xru
use crypto as Crypto
use fs as Fs

let hash = Crypto.sha256("secret")
let content = Fs.read("config.json")
```

### 2.2 Prohibited Instantiation with `@`
Attempting to instantiate a built-in module with `@` triggers an immediate static analysis error:

```xru
// let c = @Crypto()
// [ERR_CANNOT_INSTANTIATE_BUILTIN]: Built-in module 'Crypto' cannot be instantiated with '@'. Use 'Crypto.<method>()' directly.
```

---

## 3. User Units (`unit`)

Units serve as lightweight prototype classes featuring encapsulated internal state.

### 3.1 Unit Declarations & Internal Properties (`declare`)
Internal state is declared via the `declare` keyword. All references to internal properties must be prefixed with the `&` sigil:

```xru
unit WorkerPool {
    declare poolSize: number = 4
    declare isRunning: bool = false
    declare tasksProcessed: number = 0

    func start() -> bool {
        if &isRunning == true {
            std.warn("WorkerPool is already running")
            return false
        }

        &isRunning =: true
        std.info("WorkerPool started with $f{&poolSize} workers")
        return true
    }

    func recordTask() -> void {
        &tasksProcessed =: &tasksProcessed + 1
    }

    func getStats() -> number {
        return &tasksProcessed
    }
}
```

### 3.2 Encapsulation Rules
- **Reserved `&` sigil**: Accessing `&property` outside of a `unit` definition raises `[ERR_INVALID_PROP_ACCESS]`.
- **Static invocation prohibited**: Calling instance methods directly on the unit declaration without an instance (e.g., `WorkerPool.start()`) raises `[ERR_CANNOT_CALL_STATIC]`.

### 3.3 Mandatory Instantiation via `@`
To utilize a unit, a concrete instance must be initialized using `@`:

```xru
let poolA: WorkerPool = @WorkerPool()
let poolB: WorkerPool = @WorkerPool()

poolA.start()
poolA.recordTask()

std.println("Pool A Tasks: $f{poolA.getStats()}") // 1
std.println("Pool B Tasks: $f{poolB.getStats()}") // 0
```

---

## 4. Internal Invocations: `&Unit::method()` and `&self::method()`

To avoid identifier collisions with top-level functions, invocations of sibling methods within the same unit require explicit scoping:

- **`&self::methodName(...)`**: Contextual reference to the current instance.
- **`&UnitName::methodName(...)`**: Explicit scope reference matching the enclosing unit identifier.

```xru
unit Calculator {
    func add(a: number, b: number) -> number {
        return a + b
    }

    func computeFormula(x: number) -> number {
        let doubled = &self::add(x, x)
        let total = &Calculator::add(doubled, 100)
        return total
    }
}
```

Attempting to invoke `&OtherUnit::method()` from a non-matching unit generates `[ERR_INVALID_UNIT_SELF_REF]`.

---

## 5. Interface Implementation (`includes`)

Units implement method contracts using the `includes` clause:

```xru
interface Notifier {
    func send(target: string, message: string) -> bool
}

unit EmailNotifier includes Notifier {
    declare smtpHost: string = "smtp.example.com"

    func send(target: string, message: string) -> bool {
        std.info("Dispatching email to $f{target} via $f{&smtpHost}: $f{message}")
        return true
    }
}

let notifier: Notifier = @EmailNotifier()
notifier.send("dev@nehonix.com", "Pipeline complete")
```
The static compiler verifies that all interface methods are implemented with matching signatures.
