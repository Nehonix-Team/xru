package lsp

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
