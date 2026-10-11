package lsp

import "strings"

func isInsideWithNet(lines []string, currentLine int) bool {
	braceDepth := 0
	for i := currentLine; i >= 0; i-- {
		l := lines[i]
		for j := len(l) - 1; j >= 0; j-- {
			if l[j] == '}' {
				braceDepth++
			} else if l[j] == '{' {
				if braceDepth > 0 {
					braceDepth--
				} else {
					lower := strings.ToLower(l)
					return strings.Contains(lower, "with") && strings.Contains(lower, "net.fetch")
				}
			}
		}
	}
	return false
}

func (s *Server) handleWithNetCompletion(trimmedPrefix string) []CompletionItem {
	var items []CompletionItem

	// A. User already typed "set " or "set"
	if strings.HasPrefix(trimmedPrefix, "set ") || trimmedPrefix == "set" {
		props := []struct {
			name, snippet, detail, doc string
		}{
			{
				name:    "method",
				snippet: "method = \"${1|GET,POST,PUT,DELETE,HEAD|}\"",
				detail:  "set method = <string> (\"GET\" | \"POST\" | \"PUT\" | \"DELETE\")",
				doc:     "HTTP request method. Default is GET.",
			},
			{
				name:    "timeout",
				snippet: "timeout = ${1:30s}",
				detail:  "set timeout = <duration> (e.g. 10s, 30s, 1m)",
				doc:     "Maximum duration before connection timeout.",
			},
			{
				name:    "extractTo",
				snippet: "extractTo = \"${1:./src}\"",
				detail:  "set extractTo = <path: string>",
				doc:     "Target directory path to auto-extract downloaded archive (zip, tar, tar.gz).",
			},
			{
				name:    "dest",
				snippet: "dest = \"${1:./output.bin}\"",
				detail:  "set dest = <path: string>",
				doc:     "Destination file path where the raw downloaded asset should be written.",
			},
		}
		for _, p := range props {
			items = append(items, CompletionItem{
				Label:            p.name,
				Kind:             CompletionItemKindProperty,
				Detail:           p.detail,
				Documentation:    MarkupContent{Kind: "markdown", Value: p.doc},
				InsertText:       p.snippet,
				InsertTextFormat: InsertTextFormatSnippet,
			})
		}
		return items
	}

	// B. User typed "on " or "on"
	if strings.HasPrefix(trimmedPrefix, "on ") || trimmedPrefix == "on" {
		events := []struct {
			name, snippet, detail, doc string
		}{
			{
				name:    "progress",
				snippet: "progress -> (${1:percent}, ${2:speed}) => ${0}",
				detail:  "on progress -> (percent: number, speed: string) => void",
				doc:     "Callback triggered during download streaming reporting percentage and speed.",
			},
			{
				name:    "complete",
				snippet: "complete -> (${1:data}, ${2:err}) => {\n\tif ${2:err} != nil {\n\t\tUtils.log(\"<red>[ERROR]</> {${2:err}}\")\n\t} else {\n\t\tUtils.log(\"<green>[SUCCESS]</> Download completed\")\n\t}\n}",
				detail:  "on complete -> (data: any, err: error) => void",
				doc:     "Callback invoked when the download terminates (successfully or with error).",
			},
		}
		for _, ev := range events {
			items = append(items, CompletionItem{
				Label:            ev.name,
				Kind:             CompletionItemKindEvent,
				Detail:           ev.detail,
				Documentation:    MarkupContent{Kind: "markdown", Value: ev.doc},
				InsertText:       ev.snippet,
				InsertTextFormat: InsertTextFormatSnippet,
			})
		}
		return items
	}

	// C. Autocompletion of all directives supported inside `with Net.fetch`
	for _, d := range NetFetchDirectives {
		items = append(items, CompletionItem{
			Label:            d.Name,
			Kind:             CompletionItemKindKeyword,
			Detail:           d.Detail,
			Documentation:    MarkupContent{Kind: "markdown", Value: d.Doc},
			InsertText:       d.Snippet,
			InsertTextFormat: InsertTextFormatSnippet,
		})
	}
	// Also allow calling Utils inside with callbacks
	for _, uDoc := range BuiltinUnitsDocs {
		unitName := strings.TrimPrefix(uDoc.Detail, "unit ")
		items = append(items, CompletionItem{
			Label:            unitName,
			Kind:             CompletionItemKindClass,
			Detail:           uDoc.Detail,
			Documentation:    MarkupContent{Kind: "markdown", Value: uDoc.Doc},
			InsertText:       unitName,
			InsertTextFormat: InsertTextFormatPlainText,
		})
	}
	return items
}
