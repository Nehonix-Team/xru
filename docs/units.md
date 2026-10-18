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

### 3.1 Unit Declarations, State & Visibility (`declare`)
Internal state is declared via the `declare` keyword. A second parameter or modifier (`public` / `private`) can be specified to control encapsulation:

- **`public` (default)**: Property is accessible both internally via `&prop` and externally on instances via `instance.prop`.
- **`private`**: Property is strictly confined to the unit block (accessed via `&prop`). Accessing it externally triggers `[ERR_PRIVATE_PROPERTY]`.

#### Supported `declare` syntaxes:
```xru
unit WorkerPool {
    // 1. Prefix modifier
    declare public poolSize: number = 4
    declare private secretToken: string = "k8s-sec-99"

    // 2. Second parameter via comma
    declare isRunning, public: bool = false
    declare internalCounter, private: number = 0

    // 3. Trailing modifier via comma
    declare maxRetries: number = 3, private
    declare defaultHost: string = "localhost" // default: public
}
```

### 3.2 Strict Rules for the `&` Sigil
1. **Exclusively for `declare` variables**: The `&name` syntax is **strictly and exclusively** used to access local variables declared within the unit block via `declare` (e.g. `&poolSize`, `&isRunning =: true`).
2. **Prohibited method calls `&method()`**: Direct invocation of sibling methods using `&methodName(...)` is strictly forbidden and produces `[ERR_INVALID_PROP_ACCESS]`.
3. **Prohibited method property access `&method`**: Treating a method as a property via `&methodName` is forbidden and produces `[ERR_INVALID_PROP_ACCESS]`.
4. **Prohibited bare calls `method()`**: Calling a unit method bare inside a unit produces `[ERR_INVALID_METHOD_CALL]`.
5. **Only authorized method invocation syntaxes**: To call sibling methods inside a unit, you **MUST** use:
   - **`&self::methodName(...)`**
   - **`&UnitName::methodName(...)`**

All other syntaxes raise errors at both compile/check time and runtime.

### 3.3 Mandatory Instantiation via `@`
To utilize a unit, a concrete instance must be initialized using `@`:

```xru
let poolA: WorkerPool = @WorkerPool()
let poolB: WorkerPool = @WorkerPool()

poolA.start()
poolA.recordTask()

std.println("Pool A poolSize: $f{poolA.poolSize}") // OK: public property
// std.println(poolA.secretToken) // ERROR [ERR_PRIVATE_PROPERTY]: property 'secretToken' of unit 'WorkerPool' is private
```

---

## 4. Internal Invocations: `&Unit::method()` and `&self::method()`

To call sibling methods within the same unit block, explicit scoping is strictly mandatory:

- **`&self::methodName(...)`**: Contextual reference to the current instance.
- **`&UnitName::methodName(...)`**: Explicit scope reference matching the enclosing unit identifier.

```xru
unit Calculator {
    declare private precision: number = 2

    func add(a: number, b: number) -> number {
        return a + b
    }

    func computeFormula(x: number) -> number {
        // Correct invocations:
        let doubled = &self::add(x, x)
        let total = &Calculator::add(doubled, &precision)

        // The following are STRICTLY PROHIBITED and will raise errors:
        // let err1 = &add(x, x) // ERROR [ERR_INVALID_PROP_ACCESS]: Cannot call '&add()' directly.
        // let err2 = add(x, x)  // ERROR [ERR_INVALID_METHOD_CALL]: Cannot call unit method 'add' directly.

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
