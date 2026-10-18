# Standard Library Reference (Stdlib)

The XRU standard library provides built-in system modules implemented as static singletons within the Go runtime.

> [!NOTE]
> Standard library modules cannot be instantiated via `@`. Their methods are called directly via the module identifier (e.g. `Crypto.sha256(...)`, `Fs.read(...)`).

---

## 1. `std` (Standard I/O & Diagnostics)

The `std` module replaces legacy utility directives with structured terminal output functions:

| Method | Signature | Description |
| :--- | :--- | :--- |
| **`std.println(msg)`** | `(msg: any) -> void` | Prints message with newline |
| **`std.print(msg)`** | `(msg: any) -> void` | Prints message without newline |
| **`std.info(msg)`** | `(msg: string) -> void` | Prints formatted informational message (cyan) |
| **`std.success(msg)`** | `(msg: string) -> void` | Prints formatted success message (green) |
| **`std.warn(msg)`** | `(msg: string) -> void` | Prints formatted warning message (yellow) |
| **`std.error(msg)`** | `(msg: string) -> void` | Prints formatted error message (red) |
| **`std.getRandomBytes(n)`** | `(n: number) -> string` | Generates pseudo-random byte string of length `n` |

---

## 2. `fs` (Filesystem Operations)

Provides standard filesystem interaction:

| Method | Signature | Description |
| :--- | :--- | :--- |
| **`fs.read(path)`** | `(path: string) -> string` | Reads and returns full file content |
| **`fs.write(path, text)`** | `(path: string, text: string) -> bool` | Writes or overwrites file content |
| **`fs.append(path, text)`** | `(path: string, text: string) -> bool` | Appends text to target file |
| **`fs.exists(path)`** | `(path: string) -> bool` | Checks if target path exists on disk |
| **`fs.isFile(path)`** | `(path: string) -> bool` | Checks if target path is a regular file |
| **`fs.isDir(path)`** | `(path: string) -> bool` | Checks if target path is a directory |
| **`fs.size(path)`** | `(path: string) -> number` | Returns file size in bytes |
| **`fs.ensureDir(dirPath)`** | `(dirPath: string) -> bool` | Recursively creates parent directories if missing |
| **`fs.remove(path)`** | `(path: string) -> bool` | Deletes a file or directory |
| **`fs.move(src, dst)`** | `(src: string, dst: string) -> bool` | Moves or renames path |
| **`fs.copy(src, dst)`** | `(src: string, dst: string) -> bool` | Copies file or directory recursively |

---

## 3. `crypto` (Cryptographic Primitives)

| Method | Signature | Description |
| :--- | :--- | :--- |
| **`crypto.sha256(text)`** | `(text: string) -> string` | Computes hex-encoded SHA-256 hash |
| **`crypto.md5(text)`** | `(text: string) -> string` | Computes MD5 hash |
| **`crypto.uuid()`** | `() -> string` | Generates a standard UUID v4 string |
| **`crypto.getRandomBytes(n)`** | `(n: number) -> string` | Generates cryptographically secure random bytes |
| **`crypto.randomHex(n)`** | `(n: number) -> string` | Generates random hex-encoded string of length `n` |
| **`crypto.randomBase64(n)`** | `(n: number) -> string` | Generates base64-encoded string of `n` bytes |

---

## 4. `sys` (Operating System Environment)

| Method / Property | Signature | Description |
| :--- | :--- | :--- |
| **`sys.getEnv(key)`** | `(key: string) -> string` | Returns value of environment variable |
| **`sys.setEnv(key, val)`** | `(key: string, val: string) -> bool` | Sets environment variable for current process |
| **`sys.env()`** | `() -> map` | Returns dictionary of all environment variables |
| **`sys.os()`** | `() -> string` | Returns OS name (`linux`, `darwin`, `windows`) |
| **`sys.arch()`** | `() -> string` | Returns CPU architecture (`amd64`, `arm64`) |
| **`sys.sleep(dur)`** | `(dur: duration) -> void` | Suspends execution for specified duration |
| **`sys.exit(code)`** | `(code: number) -> void` | Terminates process with exit code |

---

## 5. `date` (Structured Temporal Data)

Functions return records conforming to the `DateInfo @map` interface:

| Method | Signature | Description |
| :--- | :--- | :--- |
| **`date.now()`** | `() -> DateInfo` | Returns current system timestamp |
| **`date.parse(str, fmt)`** | `(str: string, fmt: string) -> DateInfo` | Parses formatted date string |
| **`date.format(date, fmt)`** | `(date: DateInfo, fmt: string) -> string` | Formats date object to string |
| **`date.add(date, dur)`** | `(date: DateInfo, dur: duration) -> DateInfo` | Adds duration to date |
| **`date.diff(d1, d2)`** | `(d1: DateInfo, d2: DateInfo) -> duration` | Computes time delta between two dates |

---

## 6. Utility Singletons (`Json`, `Math`, `Path`)

### 6.1 `Json`
- **`Json.Parse(raw: string) -> any`**: Parses JSON string into native XRU structures.
- **`Json.Stringify(value: any) -> string`**: Serializes XRU structures to formatted JSON.

### 6.2 `Math`
- **`Math.Round(n)`**, **`Math.Floor(n)`**, **`Math.Ceil(n)`**: Rounding operations.
- **`Math.Abs(n)`**: Absolute value calculation.
- **`Math.Min(a, b)`**, **`Math.Max(a, b)`**: Value bounds comparison.
- **`Math.Random()`**: Generates pseudo-random float between `0.0` and `1.0`.

### 6.3 `Path`
- **`Path.Join(...parts)`**: Joins path components using host OS separator.
- **`Path.Dir(path)`**: Returns directory name of path.
- **`Path.Base(path)`**: Returns base filename with extension.
- **`Path.Ext(path)`**: Returns file extension (e.g., `.ts`, `.json`).

---

## 7. Native Methods on `string` and `list`

### 7.1 Strings (`string`)
Direct receiver method chaining:
```xru
let s = "  Hello World  "
s.trim().toLower().replace("world", "xru") // "hello xru"
```
- `.len() -> number`
- `.trim()`, `.trimLeft()`, `.trimRight() -> string`
- `.toUpper()`, `.toLower() -> string`
- `.contains(sub: string) -> bool`
- `.startsWith(prefix: string) -> bool`
- `.endsWith(suffix: string) -> bool`
- `.split(sep: string) -> list[string]`
- `.replace(old: string, new: string) -> string`
- `.slice(start: number, end?: number) -> string`

### 7.2 Lists (`list`)
```xru
let items = ["c", "b", "a"]
items.push("z")
let count = items.len()
let joined = items.join(", ")
```
- `.len() -> number`
- `.join(sep: string) -> string`
- `.contains(item: any) -> bool`
- `.push(item: any) -> void`
- `.pop() -> any`
- `.reverse() -> list`
- `.slice(start: number, end?: number) -> list`
- `.get(index: number) -> any` / `.at(index: number) -> any`
