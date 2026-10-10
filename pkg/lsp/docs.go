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
			"getrandombytes": {
				Detail: "Crypto.getRandomBytes(length: number = 32) -> string",
				Doc:    "Generates cryptographically secure random bytes formatted as hexadecimal.",
			},
			"randomhex": {
				Detail: "Crypto.randomHex(length: number = 32) -> string",
				Doc:    "Generates a random hexadecimal string of the specified byte length.",
			},
			"randombase64": {
				Detail: "Crypto.randomBase64(length: number = 32) -> string",
				Doc:    "Generates random bytes encoded in standard Base64.",
			},
			"sha256": {
				Detail: "Crypto.sha256(data: string) -> string",
				Doc:    "Computes the SHA-256 cryptographic hash of a given string.",
			},
			"uuid": {
				Detail: "Crypto.uuid() -> string",
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

	// 3. Recherche par méthode d'une unité si mot correspond
	for _, unitDoc := range BuiltinUnitsDocs {
		if methodDoc, ok := unitDoc.Methods[targetWord]; ok {
			return &methodDoc
		}
	}

	// 4. Méthodes chirurgicales
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

