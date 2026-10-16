package lsp

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var reInterfaceDef = regexp.MustCompile(`^\s*interface\s+([a-zA-Z_][a-zA-Z0-9_]*)(?:\s+@map)?`)
var reUnitDef = regexp.MustCompile(`^\s*unit\s+([a-zA-Z_][a-zA-Z0-9_]*)`)

// isTypeContext determines if the cursor is at a position where a type annotation is expected.
func isTypeContext(prefixLine string, lines []string, lineIdx int) bool {
	trimmed := strings.TrimSpace(prefixLine)

	// 1. After colon ':' (excluding double colons '::')
	lastColon := -1
	for i := 0; i < len(prefixLine); i++ {
		if prefixLine[i] == ':' {
			if (i > 0 && prefixLine[i-1] == ':') || (i+1 < len(prefixLine) && prefixLine[i+1] == ':') {
				continue
			}
			lastColon = i
		}
	}
	if lastColon != -1 {
		afterColon := prefixLine[lastColon+1:]
		// If there is an '=' after colon, we are past the type (in the value expression)
		if !strings.Contains(afterColon, "=") {
			return true
		}
	}

	// 2. After arrow '->'
	if arrowIdx := strings.LastIndex(prefixLine, "->"); arrowIdx != -1 {
		afterArrow := prefixLine[arrowIdx+2:]
		if !strings.Contains(afterArrow, "{") {
			return true
		}
	}

	// 3. Union type: ends with " or" or " or "
	if strings.HasSuffix(trimmed, " or") || strings.Contains(prefixLine, " or ") {
		return true
	}

	// 4. Inside generic brackets: list[..., task[..., map[...
	if bracketIdx := strings.LastIndex(prefixLine, "["); bracketIdx != -1 {
		beforeBracket := strings.TrimSpace(prefixLine[:bracketIdx])
		if strings.HasSuffix(beforeBracket, "list") || strings.HasSuffix(beforeBracket, "task") || strings.HasSuffix(beforeBracket, "map") {
			afterBracket := prefixLine[bracketIdx+1:]
			if !strings.Contains(afterBracket, "]") {
				return true
			}
		}
	}

	// 5. After 'includes' or 'extends'
	if strings.Contains(prefixLine, "includes ") || strings.Contains(prefixLine, "extends ") {
		return true
	}

	// 6. Inside an interface declaration body
	if isInsideInterfaceBlock(lines, lineIdx) {
		if !strings.HasPrefix(trimmed, "//") && !strings.HasPrefix(trimmed, "func") && !strings.HasPrefix(trimmed, "interface") {
			// Field declaration: "name " or "name? "
			parts := strings.Fields(trimmed)
			if len(parts) >= 1 && !strings.Contains(trimmed, "{") && !strings.Contains(trimmed, "}") {
				return true
			}
		}
	}

	return false
}

// isInsideInterfaceBlock checks if the current line is inside an interface { ... } block
func isInsideInterfaceBlock(lines []string, currentLine int) bool {
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
					trimmed := strings.TrimSpace(l)
					return strings.HasPrefix(trimmed, "interface ") || strings.HasPrefix(trimmed, "interface\t")
				}
			}
		}
	}
	return false
}

// collectTypeCompletions returns autocompletion items for all supported types (builtins, compound, interfaces, units).
func collectTypeCompletions(uri string, lines []string, prefixLine string) []CompletionItem {
	var items []CompletionItem
	seen := make(map[string]bool)

	addItem := func(item CompletionItem) {
		if !seen[item.Label] {
			seen[item.Label] = true
			items = append(items, item)
		}
	}

	// 1. Primitive types
	primitives := []struct {
		name   string
		detail string
		doc    string
	}{
		{"string", "Primitive type: string", "UTF-8 character string"},
		{"number", "Primitive type: number", "Numeric value (integer or floating-point)"},
		{"bool", "Primitive type: bool", "Boolean logic value (true or false)"},
		{"duration", "Primitive type: duration", "Time duration literal (e.g. 500ms, 10s, 5m, 1h)"},
		{"any", "Dynamic type: any", "Arbitrary value type; disables strict static verification"},
		{"void", "Return type: void", "Indicates function returns no value"},
		{"never", "Return type: never", "Indicates function never returns (terminates process immediately)"},
		{"nil", "Type: nil", "Null or absent value"},
	}

	for _, p := range primitives {
		addItem(CompletionItem{
			Label:            p.name,
			Kind:             CompletionItemKindKeyword,
			Detail:           p.detail,
			Documentation:    MarkupContent{Kind: "markdown", Value: p.doc},
			InsertText:       p.name,
			InsertTextFormat: InsertTextFormatPlainText,
			SortText:         "00_" + p.name,
		})
	}

	// 2. Collection & compound types
	addItem(CompletionItem{
		Label:            "list[T]",
		Kind:             CompletionItemKindTypeParameter,
		Detail:           "Generic list collection",
		Documentation:    MarkupContent{Kind: "markdown", Value: "Ordered generic list of elements: `list[T]`"},
		InsertText:       "list[${1:any}]",
		InsertTextFormat: InsertTextFormatSnippet,
		SortText:         "01_list_generic",
	})
	addItem(CompletionItem{
		Label:            "list",
		Kind:             CompletionItemKindTypeParameter,
		Detail:           "List collection",
		Documentation:    MarkupContent{Kind: "markdown", Value: "Ordered collection of elements"},
		InsertText:       "list",
		InsertTextFormat: InsertTextFormatPlainText,
		SortText:         "01_list",
	})
	addItem(CompletionItem{
		Label:            "map",
		Kind:             CompletionItemKindTypeParameter,
		Detail:           "Generic map dictionary",
		Documentation:    MarkupContent{Kind: "markdown", Value: "Key-value dictionary associative collection"},
		InsertText:       "map",
		InsertTextFormat: InsertTextFormatPlainText,
		SortText:         "01_map",
	})
	addItem(CompletionItem{
		Label:            "@map",
		Kind:             CompletionItemKindTypeParameter,
		Detail:           "Structured @map type",
		Documentation:    MarkupContent{Kind: "markdown", Value: "Structured @map typed dictionary"},
		InsertText:       "@map",
		InsertTextFormat: InsertTextFormatPlainText,
		SortText:         "01_map_at",
	})
	addItem(CompletionItem{
		Label:            "@map { ... }",
		Kind:             CompletionItemKindSnippet,
		Detail:           "Inline typed map schema",
		Documentation:    MarkupContent{Kind: "markdown", Value: "Define an inline @map schema contract with typed fields"},
		InsertText:       "@map{\n\t${1:field}: ${2:string}\n}",
		InsertTextFormat: InsertTextFormatSnippet,
		SortText:         "01_map_inline",
	})
	addItem(CompletionItem{
		Label:            "task[T]",
		Kind:             CompletionItemKindTypeParameter,
		Detail:           "Asynchronous task",
		Documentation:    MarkupContent{Kind: "markdown", Value: "Asynchronous background task handle with result type: `task[T]`"},
		InsertText:       "task[${1:any}]",
		InsertTextFormat: InsertTextFormatSnippet,
		SortText:         "01_task_generic",
	})
	addItem(CompletionItem{
		Label:            "task",
		Kind:             CompletionItemKindTypeParameter,
		Detail:           "Asynchronous task",
		Documentation:    MarkupContent{Kind: "markdown", Value: "Asynchronous background task handle"},
		InsertText:       "task",
		InsertTextFormat: InsertTextFormatPlainText,
		SortText:         "01_task",
	})
	addItem(CompletionItem{
		Label:            "range",
		Kind:             CompletionItemKindTypeParameter,
		Detail:           "Sequential range",
		Documentation:    MarkupContent{Kind: "markdown", Value: "Numeric range interval (`start..end`)"},
		InsertText:       "range",
		InsertTextFormat: InsertTextFormatPlainText,
		SortText:         "01_range",
	})
	addItem(CompletionItem{
		Label:            "func(...) -> ...",
		Kind:             CompletionItemKindSnippet,
		Detail:           "Function signature type",
		Documentation:    MarkupContent{Kind: "markdown", Value: "First-class function type signature"},
		InsertText:       "func(${1:arg}: ${2:any}) -> ${3:void}",
		InsertTextFormat: InsertTextFormatSnippet,
		SortText:         "01_func_sig",
	})

	// 3. Document interfaces from the current file
	for _, l := range lines {
		if m := reInterfaceDef.FindStringSubmatch(l); len(m) > 1 {
			ifaceName := m[1]
			addItem(CompletionItem{
				Label:            ifaceName,
				Kind:             CompletionItemKindInterface,
				Detail:           "interface " + ifaceName,
				Documentation:    MarkupContent{Kind: "markdown", Value: fmt.Sprintf("Interface contract `%s` declared in current file", ifaceName)},
				InsertText:       ifaceName,
				InsertTextFormat: InsertTextFormatPlainText,
				SortText:         "00_" + ifaceName,
			})
		}
		if m := reUnitDef.FindStringSubmatch(l); len(m) > 1 {
			unitName := m[1]
			addItem(CompletionItem{
				Label:            unitName,
				Kind:             CompletionItemKindClass,
				Detail:           "unit " + unitName,
				Documentation:    MarkupContent{Kind: "markdown", Value: fmt.Sprintf("Unit `%s` declared in current file", unitName)},
				InsertText:       unitName,
				InsertTextFormat: InsertTextFormatPlainText,
				SortText:         "02_" + unitName,
			})
		}
	}

	// 4. Interfaces and Units from Standard Library (std/*.xru)
	curFileDir := ""
	if uri != "" {
		curFileDir = filepath.Dir(uriToPath(uri))
	}
	if stdDir := FindStdDir(curFileDir); stdDir != "" {
		entries, err := os.ReadDir(stdDir)
		if err == nil {
			for _, entry := range entries {
				if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".xru") {
					continue
				}
				content, err := os.ReadFile(filepath.Join(stdDir, entry.Name()))
				if err != nil {
					continue
				}
				for _, line := range strings.Split(string(content), "\n") {
					if m := reInterfaceDef.FindStringSubmatch(line); len(m) > 1 {
						ifaceName := m[1]
						addItem(CompletionItem{
							Label:            ifaceName,
							Kind:             CompletionItemKindInterface,
							Detail:           fmt.Sprintf("std interface %s (%s)", ifaceName, entry.Name()),
							Documentation:    MarkupContent{Kind: "markdown", Value: fmt.Sprintf("Standard library interface contract `%s` from `%s`", ifaceName, entry.Name())},
							InsertText:       ifaceName,
							InsertTextFormat: InsertTextFormatPlainText,
							SortText:         "02_" + ifaceName,
						})
					}
					if m := reUnitDef.FindStringSubmatch(line); len(m) > 1 {
						unitName := m[1]
						addItem(CompletionItem{
							Label:            unitName,
							Kind:             CompletionItemKindClass,
							Detail:           fmt.Sprintf("std unit %s (%s)", unitName, entry.Name()),
							Documentation:    MarkupContent{Kind: "markdown", Value: fmt.Sprintf("Standard library unit `%s` from `%s`", unitName, entry.Name())},
							InsertText:       unitName,
							InsertTextFormat: InsertTextFormatPlainText,
							SortText:         "03_" + unitName,
						})
					}
				}
			}
		}
	}

	// 5. Standard built-in units
	builtinUnits := []string{"std", "sys", "fs", "net", "crypto", "math", "date", "json", "cmd", "path", "local"}
	for _, bu := range builtinUnits {
		addItem(CompletionItem{
			Label:            bu,
			Kind:             CompletionItemKindClass,
			Detail:           "builtin unit " + bu,
			Documentation:    MarkupContent{Kind: "markdown", Value: fmt.Sprintf("Built-in standard unit `%s`", bu)},
			InsertText:       bu,
			InsertTextFormat: InsertTextFormatPlainText,
			SortText:         "04_" + bu,
		})
	}

	return items
}
