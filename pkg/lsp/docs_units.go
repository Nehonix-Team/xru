package lsp

// BuiltinUnitsDocs contient la documentation complète des unités et de leurs méthodes
var BuiltinUnitsDocs = map[string]UnitDoc{
	"std": {
		Detail: "unit std",
		Doc:    "Modern standard I/O and utility library providing console printing, formatted diagnostics, and security helpers.",
		Methods: map[string]DocItem{
			"println": {
				Detail: "std.println(message: any) -> void",
				Doc:    "Prints a line to standard output with rich ANSI markup (e.g. `<green>...</>`, `<red>...</>`, `<cyan>...</>`).",
			},
			"print": {
				Detail: "std.print(message: any) -> void",
				Doc:    "Prints text to standard output without adding a trailing newline.",
			},
			"info": {
				Detail: "std.info(message: any) -> void",
				Doc:    "Prints an informational message with a blue [INFO] tag.",
			},
			"success": {
				Detail: "std.success(message: any) -> void",
				Doc:    "Prints a success confirmation message with a green [DONE] tag.",
			},
			"warn": {
				Detail: "std.warn(message: any) -> void",
				Doc:    "Prints a warning notification with a yellow [WARN] tag.",
			},
			"error": {
				Detail: "std.error(message: any) -> void",
				Doc:    "Prints an error message to standard error with a red [ERROR] tag.",
			},
			"sleep": {
				Detail: "std.sleep(duration: duration) -> void",
				Doc:    "Pauses the current execution for the given duration (e.g. `std.sleep(120ms)`).",
			},
			"fatal": {
				Detail: "std.fatal(message: any, code?: number = 1) -> never",
				Doc:    "Prints a red [FATAL] message to stderr and terminates the process immediately with `code`. Cannot be caught. Code after it is unreachable.",
			},
		},
	},
	"net": {
		Detail: "unit Net",
		Doc:    "HTTP networking client and remote package download engine.",
		Methods: map[string]DocItem{
			"fetch": {
				Detail: "Net.fetch(url: string) -> FetchContext",
				Doc:    "Downloads an asset or release from a remote URL with streaming progress, speed computation, and optional extraction (`set extractTo`).",
			},
			"get": {
				Detail: "Net.get(url: string) -> string",
				Doc:    "Performs a simple HTTP GET request and returns the response body as a string.",
			},
			"clone": {
				Detail: "Net.clone(repoUrl: string, dest: string) -> bool",
				Doc:    "Clones a remote Git repository to the target destination folder.",
			},
		},
	},
	"crypto": {
		Detail: "unit Crypto",
		Doc:    "Cryptographic primitives unit for hashing, UUID, and secure randomness.",
		Methods: map[string]DocItem{
			"GetRandomBytes": {
				Detail: "Crypto.GetRandomBytes(length: number = 32) -> string",
				Doc:    "Generates cryptographically secure random bytes formatted as hexadecimal.",
			},
			"RandomHex": {
				Detail: "Crypto.RandomHex(length: number = 32) -> string",
				Doc:    "Generates a random hexadecimal string of the specified byte length.",
			},
			"RandomBase64": {
				Detail: "Crypto.RandomBase64(length: number = 32) -> string",
				Doc:    "Generates random bytes encoded in standard Base64.",
			},
			"Sha256": {
				Detail: "Crypto.Sha256(data: string) -> string",
				Doc:    "Computes the SHA-256 cryptographic hash of a given string.",
			},
			"Uuid": {
				Detail: "Crypto.Uuid() -> string",
				Doc:    "Generates a random UUID v4 identifier string.",
			},
		},
	},
	"fs": {
		Detail: "unit Fs",
		Doc:    "File system manipulation and disk operations unit.",
		Methods: map[string]DocItem{
			"exists": {
				Detail: "Fs.exists(path: string) -> bool",
				Doc:    "Checks whether the specified file or directory path exists.",
			},
			"isdir": {
				Detail: "Fs.isDir(path: string) -> bool",
				Doc:    "Returns true if the given path is an existing directory.",
			},
			"isfile": {
				Detail: "Fs.isFile(path: string) -> bool",
				Doc:    "Returns true if the given path is an existing regular file.",
			},
			"read": {
				Detail: "Fs.read(path: string) -> string",
				Doc:    "Reads the entire contents of a text file into a string.",
			},
			"write": {
				Detail: "Fs.write(path: string, content: string) -> bool",
				Doc:    "Writes content to the destination file, creating parent directories if necessary.",
			},
			"remove": {
				Detail: "Fs.remove(path: string) -> void",
				Doc:    "Deletes a file or directory recursively.",
			},
			"move": {
				Detail: "Fs.move(source: string, destination: string) -> void",
				Doc:    "Moves or renames a file or folder to a new location.",
			},
			"copy": {
				Detail: "Fs.copy(source: string, destination: string) -> void",
				Doc:    "Copies a file or folder tree to the target destination.",
			},
			"ensuredir": {
				Detail: "Fs.ensureDir(path: string) -> void",
				Doc:    "Ensures that a directory exists, creating all intermediate folders if needed.",
			},
			"size": {
				Detail: "Fs.size(path: string) -> number",
				Doc:    "Returns the file size in bytes.",
			},
		},
	},
	"local": {
		Detail: "unit Local",
		Doc:    "Local sandboxing unit available inside subdescriptor rules (`.rules.xru`).",
		Methods: map[string]DocItem{
			"keep": {
				Detail: "Local.Keep(files: list[string]) -> void",
				Doc:    "Keeps only the specified files in current directory and drops all others.",
			},
			"drop": {
				Detail: "Local.Drop(patterns: list[string]) -> void",
				Doc:    "Drops files and directories matching the specified glob patterns in the current folder.",
			},
			"usevariant": {
				Detail: "Local.UseVariant(variantName: string) -> void",
				Doc:    "Selects and applies an alternative variant directory template.",
			},
		},
	},
	"sys": {
		Detail: "unit Sys",
		Doc:    "Operating system and host environment inspection unit.",
		Methods: map[string]DocItem{
			"exit": {
				Detail: "Sys.exit(code?: number = 0) -> never",
				Doc:    "Terminates the process immediately with the given exit code, without printing anything. Code after it is unreachable.",
			},
			"getenv": {
				Detail: "Sys.getEnv(name: string, fallback?: string) -> string",
				Doc:    "Retrieves the value of an OS environment variable with optional fallback.",
			},
			"os": {
				Detail: "Sys.os() -> string",
				Doc:    "Returns the current operating system name (e.g. `linux`, `windows`, `darwin`).",
			},
			"arch": {
				Detail: "Sys.arch() -> string",
				Doc:    "Returns the current machine processor architecture (e.g. `amd64`, `arm64`).",
			},
		},
	},
	"cmd": {
		Detail: "unit Cmd",
		Doc:    "Process execution and command runner unit.",
		Methods: map[string]DocItem{
			"exec": {
				Detail: "Cmd.exec(command: string) -> string",
				Doc:    "Executes a system shell command and returns its standard output.",
			},
		},
	},
	"path": {
		Detail: "unit Path",
		Doc:    "File path manipulation unit (joining, directory extraction, base names, and extensions).",
		Methods: map[string]DocItem{
			"join": {
				Detail: "Path.join(...parts: string) -> string",
				Doc:    "Joins file path elements into a single path, separating them with an OS specific Separator.",
			},
			"dir": {
				Detail: "Path.dir(path: string) -> string",
				Doc:    "Returns all but the last element of path, typically the path's directory.",
			},
			"base": {
				Detail: "Path.base(path: string) -> string",
				Doc:    "Returns the last element of path (trailing path separators are removed before extracting the last element).",
			},
			"ext": {
				Detail: "Path.ext(path: string) -> string",
				Doc:    "Returns the file name extension used by path (e.g. `.xru`, `.json`).",
			},
		},
	},
	"json": {
		Detail: "unit Json",
		Doc:    "JSON encoding and decoding unit.",
		Methods: map[string]DocItem{
			"parse": {
				Detail: "Json.parse(text: string) -> any",
				Doc:    "Parses a valid JSON string into native XRU maps, lists, or primitive values.",
			},
			"stringify": {
				Detail: "Json.stringify(val: any, indent?: string) -> string",
				Doc:    "Converts an XRU value (map, list, primitive) into a formatted JSON string.",
			},
		},
	},
	"math": {
		Detail: "unit Math",
		Doc:    "Mathematical functions and numerical calculation unit.",
		Methods: map[string]DocItem{
			"round": {
				Detail: "Math.round(x: number) -> number",
				Doc:    "Returns the nearest integer, rounding half away from zero.",
			},
			"floor": {
				Detail: "Math.floor(x: number) -> number",
				Doc:    "Returns the greatest integer value less than or equal to x.",
			},
			"ceil": {
				Detail: "Math.ceil(x: number) -> number",
				Doc:    "Returns the least integer value greater than or equal to x.",
			},
			"abs": {
				Detail: "Math.abs(x: number) -> number",
				Doc:    "Returns the absolute value of x.",
			},
			"min": {
				Detail: "Math.min(a: number, b: number) -> number",
				Doc:    "Returns the smaller of two numbers.",
			},
			"max": {
				Detail: "Math.max(a: number, b: number) -> number",
				Doc:    "Returns the larger of two numbers.",
			},
			"random": {
				Detail: "Math.random() -> number",
				Doc:    "Returns a pseudo-random floating number between 0.0 (inclusive) and 1.0 (exclusive).",
			},
		},
	},
	"date": {
		Detail: "unit date",
		Doc:    "Comprehensive Date & Time manipulation unit (formatting, parsing, ISO-8601, arithmetic, diffs).",
		Methods: map[string]DocItem{
			"now": {
				Detail: "date.now() -> @map{ iso: string, timestamp: number, year: number, month: number, day: number, hour: number, minute: number, second: number, weekday: string }",
				Doc:    "Returns the current date and time as a structured Map with `iso`, `timestamp`, `year`, `month`, `day`, `hour`, `minute`, `second`, `weekday`.",
			},
			"iso": {
				Detail: "date.iso(date?: any) -> string",
				Doc:    "Returns an ISO-8601 formatted date string (e.g. `\"2026-10-15T09:50:36Z\"`).",
			},
			"today": {
				Detail: "date.today() -> string",
				Doc:    "Returns today's date in `\"YYYY-MM-DD\"` format.",
			},
			"timestamp": {
				Detail: "date.timestamp(date?: any) -> number",
				Doc:    "Returns current or given date timestamp in Unix milliseconds.",
			},
			"format": {
				Detail: "date.format(dateOrFormat: any, format?: string) -> string",
				Doc:    "Formats date using modern template tokens (e.g. `date.format(\"YYYY-MM-DD HH:mm:ss\")` or `date.format(d, \"DD/MM/YYYY\")`).",
			},
			"parse": {
				Detail: "date.parse(dateStr: string, format?: string) -> @map{ iso: string, timestamp: number, year: number, month: number, day: number, hour: number, minute: number, second: number, weekday: string }",
				Doc:    "Parses a date string (ISO-8601, common formats, or custom format) into a structured Date Map.",
			},
			"add": {
				Detail: "date.add(date: any, amount: any, unit?: string) -> @map{ iso: string, timestamp: number, year: number, month: number, day: number, hour: number, minute: number, second: number, weekday: string }",
				Doc:    "Adds time to date (e.g. `date.add(d, 2, \"days\")`, `date.add(d, 30, \"minutes\")`, `date.add(d, \"2h\")`).",
			},
			"diff": {
				Detail: "date.diff(dateA: any, dateB: any, unit?: string) -> number",
				Doc:    "Calculates difference between two dates in `\"seconds\"`, `\"minutes\"`, `\"hours\"`, or `\"days\"`.",
			},
			"year": {
				Detail: "date.year(date?: any) -> number",
				Doc:    "Returns full year (e.g. `2026`).",
			},
			"month": {
				Detail: "date.month(date?: any) -> number",
				Doc:    "Returns month (1 to 12).",
			},
			"day": {
				Detail: "date.day(date?: any) -> number",
				Doc:    "Returns day of month (1 to 31).",
			},
		},
	},
	"document": {
		Detail: "unit Document",
		Doc:    "Multi-view surgical file manipulation unit providing raw, linewise, JSON, and fuzzy editing capabilities.",
		Methods: map[string]DocItem{
			"getContent": {
				Detail: "doc.getContent() -> string",
				Doc:    "Returns the entire raw content of the document as a string.",
			},
			"setContent": {
				Detail: "doc.setContent(newContent: string) -> bool",
				Doc:    "Replaces the entire content of the document with the provided string.",
			},
			"append": {
				Detail: "doc.append(content: string) -> bool",
				Doc:    "Appends the specified text to the end of the document.",
			},
			"prepend": {
				Detail: "doc.prepend(content: string) -> bool",
				Doc:    "Prepends the specified text to the beginning of the document.",
			},
			"contains": {
				Detail: "doc.contains(pattern: string) -> bool",
				Doc:    "Checks whether the document contains the specified pattern.",
			},
			"size": {
				Detail: "doc.size() -> number",
				Doc:    "Returns the total number of lines in the document.",
			},
			"getLines": {
				Detail: "doc.getLines() -> list[string]",
				Doc:    "Returns all lines of the document as a list of strings.",
			},
			"setLines": {
				Detail: "doc.setLines(lines: list[string]) -> bool",
				Doc:    "Replaces all lines of the document with the provided list of strings.",
			},
			"getLine": {
				Detail: "doc.getLine(lineNum: number) -> string",
				Doc:    "Returns the content of the line at 1-based index `lineNum`.",
			},
			"setLine": {
				Detail: "doc.setLine(lineNum: number, text: string) -> bool",
				Doc:    "Updates the content of the line at 1-based index `lineNum`.",
			},
			"removeLine": {
				Detail: "doc.removeLine(lineNumOrPattern: number | string) -> bool",
				Doc:    "Deletes the line at 1-based index `lineNum`, or deletes the first line containing the given pattern.",
			},
			"removeLines": {
				Detail: "doc.removeLines(pattern: string | startLine, endLine: number) -> bool",
				Doc:    "Deletes all lines containing `pattern`, or deletes lines in the 1-based range [`startLine`, `endLine`].",
			},
			"getJson": {
				Detail: "doc.getJson(keyPath: string) -> any",
				Doc:    "Parses the document as JSON and retrieves the value at dot-separated `keyPath` (e.g. `'scripts.build'`).",
			},
			"setJson": {
				Detail: "doc.setJson(keyPath: string, value: any) -> bool",
				Doc:    "Sets or updates a field at dot-separated `keyPath` with formatted JSON indentation.",
			},
			"removeJson": {
				Detail: "doc.removeJson(keyPath: string) -> bool",
				Doc:    "Removes a field at dot-separated `keyPath` from the JSON structure.",
			},
			"replaceFuzzy": {
				Detail: "doc.replaceFuzzy(target: string, replacement: string) -> bool",
				Doc:    "Replaces target with whitespace-insensitive fuzzy token matching.",
			},
			"removeFuzzy": {
				Detail: "doc.removeFuzzy(target: string) -> bool",
				Doc:    "Deletes target using whitespace-insensitive fuzzy token matching.",
			},
			"removeBetween": {
				Detail: "doc.removeBetween(start: string, end: string, includeBounds?: bool) -> bool",
				Doc:    "Removes content between `start` and `end` delimiters.",
			},
			"removeBalanced": {
				Detail: "doc.removeBalanced(anchor: string, openChar?: string, closeChar?: string) -> bool",
				Doc:    "Surgically removes a block starting at `anchor` by counting matched bracket depth, safely ignoring strings and comments.",
			},
			"save": {
				Detail: "doc.save() -> bool",
				Doc:    "Manually flushes document modifications to disk immediately.",
			},
		},
	},
}
