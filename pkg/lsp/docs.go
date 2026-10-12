package lsp

import (
	"fmt"
	"strings"
)

type DocItem struct {
	Detail string
	Doc    string
}

// BuiltinUnitsDocs contient la documentation complète des unités et de leurs méthodes
var BuiltinUnitsDocs = map[string]struct {
	Detail  string
	Doc     string
	Methods map[string]DocItem
}{
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
				Doc:    "Checks whether the document contains the specified substring.",
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

// SurgicalMethodsDocs contient la documentation des méthodes chirurgicales sur fichiers
var SurgicalMethodsDocs = map[string]DocItem{
	"replace": {
		Detail: "replace(target: string, replacement: string) -> void",
		Doc:    "Replaces all literal occurrences of `target` with `replacement` in the file.",
	},
	"replaceline": {
		Detail: "replaceLine(pattern: string, replacement: string) -> void",
		Doc:    "Replaces any line matching `pattern` with the specified `replacement` line.",
	},
	"remove": {
		Detail: "remove(target: string) -> void",
		Doc:    "Deletes all occurrences of the `target` text from the file.",
	},
	"removelines": {
		Detail: "removeLines(pattern: string) -> void",
		Doc:    "Deletes any line containing the specified `pattern`.",
	},
	"removeblock": {
		Detail: "removeBlock(startPattern: string, endPattern: string) -> void",
		Doc:    "Deletes the contiguous block of lines starting from `startPattern` through `endPattern`.",
	},
	"removeuntil": {
		Detail: "removeUntil(startPattern: string, untilPattern: string) -> void",
		Doc:    "Deletes lines beginning at `startPattern` up until `untilPattern` is encountered.",
	},
	"commentout": {
		Detail: "commentOut(pattern: string) -> void",
		Doc:    "Comments out lines containing `pattern` by prepending `//` markers.",
	},
	"append": {
		Detail: "append(content: string) -> void",
		Doc:    "Appends the given content string to the end of the file.",
	},
	"prepend": {
		Detail: "prepend(content: string) -> void",
		Doc:    "Prepends the given content string to the beginning of the file.",
	},
	"jsonset": {
		Detail: "jsonSet(keyPath: string, value: any) -> void",
		Doc:    "Sets or modifies a nested property in a JSON file via dot notation (e.g. `\"scripts.build\"`).",
	},
	"jsonremove": {
		Detail: "jsonRemove(keyPath: string) -> void",
		Doc:    "Deletes a property key from a JSON file.",
	},
	"keep": {
		Detail: "Keep(files: list[string]) -> void",
		Doc:    "Keeps only the listed files in the directory and drops the rest.",
	},
	"drop": {
		Detail: "Drop(patterns: list[string]) -> void",
		Doc:    "Drops files or directories matching the glob patterns.",
	},
}

// StringMethodsDocs contient la documentation des méthodes natives sur les chaînes
var StringMethodsDocs = map[string]DocItem{
	"len": {
		Detail: "(string).len() -> number",
		Doc:    "Returns the length (number of characters) of the string.",
	},
	"trim": {
		Detail: "(string).trim() -> string",
		Doc:    "Returns the string with leading and trailing whitespace removed.",
	},
	"trimleft": {
		Detail: "(string).trimLeft() -> string",
		Doc:    "Returns the string with leading whitespace removed.",
	},
	"trimright": {
		Detail: "(string).trimRight() -> string",
		Doc:    "Returns the string with trailing whitespace removed.",
	},
	"toupper": {
		Detail: "(string).toUpper() -> string",
		Doc:    "Returns the string with all characters converted to uppercase.",
	},
	"tolower": {
		Detail: "(string).toLower() -> string",
		Doc:    "Returns the string with all characters converted to lowercase.",
	},
	"contains": {
		Detail: "(string).contains(substr: string) -> bool",
		Doc:    "Reports whether `substr` is within the string.",
	},
	"startswith": {
		Detail: "(string).startsWith(prefix: string) -> bool",
		Doc:    "Tests whether the string begins with `prefix`.",
	},
	"endswith": {
		Detail: "(string).endsWith(suffix: string) -> bool",
		Doc:    "Tests whether the string ends with `suffix`.",
	},
	"split": {
		Detail: "(string).split(separator: string) -> list[string]",
		Doc:    "Slices the string into all substrings separated by `separator` and returns a list of them.",
	},
	"replace": {
		Detail: "(string).replace(old: string, new: string) -> string",
		Doc:    "Returns a copy of the string with all occurrences of `old` replaced by `new`.",
	},
	"slice": {
		Detail: "(string).slice(start: number, end?: number) -> string",
		Doc:    "Returns a substring starting from index `start` up to optional index `end` (or the end of the string).",
	},
}

// ListMethodsDocs contient la documentation des méthodes natives sur les listes
var ListMethodsDocs = map[string]DocItem{
	"len": {
		Detail: "(list).len() -> number",
		Doc:    "Returns the number of elements in the list.",
	},
	"join": {
		Detail: "(list).join(separator: string) -> string",
		Doc:    "Concatenates the elements of the list to create a single string, using `separator` between elements.",
	},
	"contains": {
		Detail: "(list).contains(item: any) -> bool",
		Doc:    "Reports whether `item` is present in the list.",
	},
	"push": {
		Detail: "(list).push(item: any) -> void",
		Doc:    "Appends a new element to the end of the list in place.",
	},
	"pop": {
		Detail: "(list).pop() -> any",
		Doc:    "Removes and returns the last element of the list.",
	},
	"reverse": {
		Detail: "(list).reverse() -> list",
		Doc:    "Returns a new list with elements in reverse order.",
	},
	"slice": {
		Detail: "(list).slice(start: number, end?: number) -> list",
		Doc:    "Returns a new list containing a portion of the list from index `start` up to optional index `end`.",
	},
}

// KeywordsDocs documente les mots-clés de base du langage XRU 2.0
var KeywordsDocs = map[string]DocItem{
	"use": {
		Detail: "use <module> [as <Alias>]",
		Doc:    "Imports a standard or external unit module into current rule scope.",
	},
	"as": {
		Detail: "as <Alias>",
		Doc:    "Defines an identifier alias for an imported module or type.",
	},
	"with": {
		Detail: "with <Unit.method(...)> { ... }",
		Doc:    "Opens a declarative block configuring properties (`set`) and event callbacks (`on`).",
	},
	"set": {
		Detail: "set <property> = <value>",
		Doc:    "Sets a configuration parameter inside a declarative `with` block.",
	},
	"on": {
		Detail: "on <event> -> (params) => { ... }",
		Doc:    "Registers an event callback lambda handler inside a declarative `with` block.",
	},
	"header": {
		Detail: "header <Key> -> <Value>",
		Doc:    "Attaches an HTTP header to a network request block.",
	},
	"let": {
		Detail: "let <name>[: type] = <value>",
		Doc:    "Declares a mutable local variable.",
	},
	"const": {
		Detail: "const <name>[: type] = <value>",
		Doc:    "Declares an immutable constant.",
	},
	"arg": {
		Detail: "arg <name>[: type] [= default]",
		Doc:    "Declares a CLI argument accepted by this rule unit (e.g. `--name=value`).",
	},
	"func": {
		Detail: "[global] func <name>(params) [-> returnType] { ... }",
		Doc:    "Defines a reusable function.",
	},
	"unit": {
		Detail: "unit <Name> { ... }",
		Doc:    "Declares a structured reusable unit containing methods and directives.",
	},
	"for": {
		Detail: "for <item> in <collection | start..end> { ... }",
		Doc:    "Iterates sequentially over a collection or numeric range (e.g. `1..totalItems`).",
	},
	"in": {
		Detail: "for <item> in <collection | start..end>",
		Doc:    "Specifies the collection or range target in a `for` loop.",
	},
	"catch": {
		Detail: "<expr> catch <err> [or <fallback>]",
		Doc:    "Safely traps runtime errors from an expression or action into `<err>` with optional `or <fallback>` value or block.",
	},
	"dispatch": {
		Detail: "dispatch \"<path>\" [, \"<pattern>\"]",
		Doc:    "Recursively searches and executes subdescriptor rule scripts (defaults to `.rules.xru`) under the given path.",
	},
	"match": {
		Detail: "match <expr> { case <pattern> { ... } default { ... } }",
		Doc:    "Pattern-matching control flow structure evaluating cases against an expression.",
	},
	"case": {
		Detail: "case <pattern> { ... }",
		Doc:    "Branch of a `match` statement executed if the expression equals the pattern.",
	},
	"default": {
		Detail: "default { ... }",
		Doc:    "Fallback branch of a `match` statement executed if no previous `case` matched.",
	},
}

// GetDocForWord recherche une documentation complète pour un mot ou une expression membre
func GetDocForWord(fullExpr, targetWord string) *DocItem {
	fullExpr = strings.TrimSpace(fullExpr)
	targetWord = strings.ToLower(strings.TrimSpace(targetWord))

	// 1. Recherche par expression membre complète ex: "Utils.log" ou "Net.fetch"
	if strings.Contains(fullExpr, ".") {
		parts := strings.Split(fullExpr, ".")
		if len(parts) == 2 {
			unitKey := strings.ToLower(parts[0])
			methodKey := strings.ToLower(parts[1])
			if unitDoc, ok := BuiltinUnitsDocs[unitKey]; ok {
				if methodDoc, ok := unitDoc.Methods[methodKey]; ok {
					return &methodDoc
				}
				for mK, mV := range unitDoc.Methods {
					if strings.EqualFold(mK, methodKey) {
						return &mV
					}
				}
			}
			if strDoc, ok := StringMethodsDocs[methodKey]; ok {
				return &strDoc
			}
			if listDoc, ok := ListMethodsDocs[methodKey]; ok {
				return &listDoc
			}
		}
	}

	// 2. Recherche par unité
	if unitDoc, ok := BuiltinUnitsDocs[targetWord]; ok {
		return &DocItem{
			Detail: unitDoc.Detail,
			Doc:    unitDoc.Doc,
		}
	}

	// 3. Méthodes document
	if docUnit, ok := BuiltinUnitsDocs["document"]; ok {
		for mK, mV := range docUnit.Methods {
			if strings.EqualFold(mK, targetWord) {
				return &mV
			}
		}
	}

	// 4. Méthodes natives String et List
	if strDoc, ok := StringMethodsDocs[targetWord]; ok {
		return &strDoc
	}
	if listDoc, ok := ListMethodsDocs[targetWord]; ok {
		return &listDoc
	}

	// 5. Méthodes chirurgicales legacy
	if surgDoc, ok := SurgicalMethodsDocs[targetWord]; ok {
		return &surgDoc
	}

	// 5. Mots-clés
	if kwDoc, ok := KeywordsDocs[targetWord]; ok {
		return &kwDoc
	}

	return nil
}

func FormatMarkdown(item *DocItem) string {
	if item == nil {
		return ""
	}
	var sb strings.Builder
	if item.Detail != "" {
		sb.WriteString(fmt.Sprintf("```xru\n%s\n```\n", item.Detail))
	}
	if item.Doc != "" {
		sb.WriteString(item.Doc)
	}
	return sb.String()
}

type DirectiveDoc struct {
	Name    string
	Snippet string
	Detail  string
	Doc     string
}

// NetFetchDirectives lists all valid directives for `with Net.fetch`
var NetFetchDirectives = []DirectiveDoc{
	{
		Name:    "set method",
		Snippet: "set method = \"${1|GET,POST,PUT,DELETE,HEAD|}\"",
		Detail:  "set method = <string>",
		Doc:     "HTTP request method (GET, POST, PUT, DELETE, HEAD). Default: GET.",
	},
	{
		Name:    "set timeout",
		Snippet: "set timeout = ${1:30s}",
		Detail:  "set timeout = <duration>",
		Doc:     "Maximum duration before connection timeout (e.g. 10s, 30s, 1m).",
	},
	{
		Name:    "set extractTo",
		Snippet: "set extractTo = \"${1:./src}\"",
		Detail:  "set extractTo = <string>",
		Doc:     "Target directory to automatically unpack downloaded archive (zip, tar, tar.gz).",
	},
	{
		Name:    "set dest",
		Snippet: "set dest = \"${1:./output.bin}\"",
		Detail:  "set dest = <string>",
		Doc:     "Target file path to save raw downloaded file content.",
	},
	{
		Name:    "header",
		Snippet: "header \"${1:Accept}\" -> \"${2:application/json}\"",
		Detail:  "header <name> -> <value>",
		Doc:     "Attaches an HTTP header to the request.",
	},
	{
		Name:    "on progress",
		Snippet: "on progress -> (${1:percent}, ${2:speed}) => ${0}",
		Detail:  "on progress -> (percent: number, speed: string) => void",
		Doc:     "Event callback triggered during streaming download reporting progress percentage and network speed.",
	},
	{
		Name:    "on complete",
		Snippet: "on complete -> (${1:data}, ${2:err}) => {\n\tif ${2:err} != nil {\n\t\tUtils.log(\"<red>[ERROR]</> {${2:err}}\")\n\t} else {\n\t\tUtils.log(\"<green>[SUCCESS]</> Download complete\")\n\t}\n}",
		Detail:  "on complete -> (data: any, err: error) => void",
		Doc:     "Event callback triggered when the network request completes or fails.",
	},
}

// SyntaxSnippets contains statement & control flow snippets
var SyntaxSnippets = []DirectiveDoc{
	{
		Name:    "with Net.fetch",
		Snippet: "with Net.fetch(\"${1:url}\") {\n\tset method = \"GET\"\n\tset timeout = 30s\n\n\ton progress -> (percent, speed) => Utils.log(\"<cyan>[PROGRESS]</> {percent}% à {speed}\")\n\ton complete -> (data, err) => {\n\t\tif err != nil {\n\t\t\tUtils.log(\"<red>[ERROR]</> {err}\")\n\t\t} else {\n\t\t\tUtils.log(\"<green>[DONE]</> Download complete\")\n\t\t}\n\t}\n}",
		Detail:  "with Net.fetch(url) { directives... }",
		Doc:     "Declarative remote asset download block with real-time streaming progress.",
	},
	{
		Name:    "use ... as ...",
		Snippet: "use ${1:utils} as ${2:Utils}",
		Detail:  "use <module> as <Alias>",
		Doc:     "Imports a unit module under an alias.",
	},
	{
		Name:    "let",
		Snippet: "let ${1:name}: ${2:string} = ${3:\"value\"}",
		Detail:  "let <name>: <type> = <value>",
		Doc:     "Declares a mutable variable.",
	},
	{
		Name:    "const",
		Snippet: "const ${1:name}: ${2:string} = ${3:\"value\"}",
		Detail:  "const <name>: <type> = <value>",
		Doc:     "Declares an immutable constant.",
	},
	{
		Name:    "arg",
		Snippet: "arg ${1:name}: ${2:string} = ${3:\"default\"}",
		Detail:  "arg <name>: <type> = <default>",
		Doc:     "Declares a CLI argument parameter.",
	},
	{
		Name:    "func",
		Snippet: "func ${1:name}(${2:params}) {\n\t$0\n}",
		Detail:  "func <name>(<params>) { ... }",
		Doc:     "Declares a local function.",
	},
	{
		Name:    "global func",
		Snippet: "global func ${1:name}(${2:params}) {\n\t$0\n}",
		Detail:  "global func <name>(<params>) { ... }",
		Doc:     "Declares a global function accessible cross-unit.",
	},
	{
		Name:    "unit",
		Snippet: "unit ${1:Name} {\n\t$0\n}",
		Detail:  "unit <Name> { ... }",
		Doc:     "Declares a custom unit module.",
	},
	{
		Name:    "if",
		Snippet: "if ${1:condition} {\n\t$0\n}",
		Detail:  "if <condition> { ... }",
		Doc:     "Conditional execution block.",
	},
	{
		Name:    "if else",
		Snippet: "if ${1:condition} {\n\t$2\n} else {\n\t$0\n}",
		Detail:  "if <condition> { ... } else { ... }",
		Doc:     "Conditional execution with fallback branch.",
	},
	{
		Name:    "for in",
		Snippet: "for ${1:item} in ${2:items} {\n\t$0\n}",
		Detail:  "for <item> in <list> { ... }",
		Doc:     "Iterates over a list collection.",
	},
	{
		Name:    "for range",
		Snippet: "for ${1:i} in ${2:1}..${3:10} {\n\t$0\n}",
		Detail:  "for <i> in <start>..<end> { ... }",
		Doc:     "Iterates over an integer numeric range.",
	},
	{
		Name:    "with task",
		Snippet: "let ${1:task} = with ${2:targetOperation()}\nlet ${3:result} = ${1:task}.wait()",
		Detail:  "let <task> = with <expr>",
		Doc:     "Launches an asynchronous background task.",
	},
	{
		Name:    "catch err or",
		Snippet: "catch ${1:err} or ${2:\"default\"}",
		Detail:  "<expr> catch <err> or <fallback>",
		Doc:     "Traps runtime errors and provides a fallback value or handler.",
	},
	{
		Name:    "replace",
		Snippet: "replace(\"${1:target}\", \"${2:replacement}\")",
		Detail:  "replace(target, replacement)",
		Doc:     "Replaces all target string occurrences in the file.",
	},
	{
		Name:    "replaceLine",
		Snippet: "replaceLine(\"${1:pattern}\", \"${2:replacement}\")",
		Detail:  "replaceLine(pattern, replacement)",
		Doc:     "Replaces matching line with replacement.",
	},
	{
		Name:    "removeBlock",
		Snippet: "removeBlock(\"${1:startPattern}\", \"${2:endPattern}\")",
		Detail:  "removeBlock(startPattern, endPattern)",
		Doc:     "Deletes the code block between start and end patterns.",
	},
}
