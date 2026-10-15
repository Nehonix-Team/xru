package lsp

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
