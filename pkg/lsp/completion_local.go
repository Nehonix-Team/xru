package lsp

import (
	"fmt"
	"regexp"
	"strings"
)

type paramInfo struct {
	name      string
	paramType string
}

func findCurrentFunctionParams(lines []string, currentLine int) []paramInfo {
	var params []paramInfo
	reFunc := regexp.MustCompile(`^\s*(?:global\s+)?func\s+[a-zA-Z_][a-zA-Z0-9_]*\s*\((.*?)\)`)
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
					if m := reFunc.FindStringSubmatch(lines[i]); len(m) > 1 {
						paramStr := strings.TrimSpace(m[1])
						if paramStr != "" {
							for _, p := range strings.Split(paramStr, ",") {
								p = strings.TrimSpace(p)
								parts := strings.Split(p, ":")
								pName := strings.TrimSpace(parts[0])
								pType := "any"
								if len(parts) > 1 {
									pType = strings.TrimSpace(parts[1])
								}
								if pName != "" {
									params = append(params, paramInfo{name: pName, paramType: pType})
								}
							}
						}
						return params
					}
				}
			}
		}
	}
	return params
}

func collectLocalSymbols(lines []string, currentLine int) []CompletionItem {
	var items []CompletionItem
	seen := make(map[string]bool)

	// 1. Paramètres de la fonction englobante courante
	currentFuncParams := findCurrentFunctionParams(lines, currentLine)
	for _, p := range currentFuncParams {
		if !seen[p.name] {
			seen[p.name] = true
			pType := p.paramType
			if pType == "" {
				pType = "any"
			}
			items = append(items, CompletionItem{
				Label:            p.name,
				Kind:             CompletionItemKindVariable,
				Detail:           fmt.Sprintf("(parameter) %s: %s", p.name, pType),
				Documentation:    MarkupContent{Kind: "markdown", Value: fmt.Sprintf("Parameter `%s` of current function", p.name)},
				InsertText:       p.name,
				InsertTextFormat: InsertTextFormatPlainText,
				SortText:         fmt.Sprintf("00_%s", p.name),
			})
		}
	}

	reLetConst := regexp.MustCompile(`^\s*(?:global\s+)?(let|const|arg)\s+([a-zA-Z_][a-zA-Z0-9_]*)(?:\s*:\s*([^=]+?))?(?:\s*=|$)`)
	reFunc := regexp.MustCompile(`^\s*(?:global\s+)?func\s+([a-zA-Z_][a-zA-Z0-9_]*)\s*\((.*?)\)(?:\s*->\s*([a-zA-Z0-9_]+))?`)
	reUnit := regexp.MustCompile(`^\s*unit\s+([a-zA-Z_][a-zA-Z0-9_]*)`)
	reDeclare := regexp.MustCompile(`^\s*(?:declare|prop)\s+([a-zA-Z_][a-zA-Z0-9_]*)(?:\s*:\s*([^=]+?))?(?:\s*=|$)`)
	reUse := regexp.MustCompile(`^\s*use\s+([a-zA-Z_][a-zA-Z0-9_]*)(?:\s+as\s+([a-zA-Z_][a-zA-Z0-9_]*))?`)
	reFromUse := regexp.MustCompile(`^\s*from\s+"[^"]+"\s+use\s+(.+)$`)
	reForLoop := regexp.MustCompile(`^\s*for\s+([a-zA-Z_][a-zA-Z0-9_]*)\s+in\b`)

	for idx, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*") || strings.HasPrefix(trimmed, "*") {
			continue
		}

		// 2. let / const / arg
		if m := reLetConst.FindStringSubmatch(line); len(m) > 2 {
			kindStr := m[1]
			name := m[2]
			typeStr := strings.TrimSpace(m[3])
			if typeStr == "" {
				typeStr = "any"
			}
			if !seen[name] {
				seen[name] = true
				itemKind := CompletionItemKindVariable
				if kindStr == "const" {
					itemKind = CompletionItemKindConstant
				} else if kindStr == "arg" {
					itemKind = CompletionItemKindProperty
				}
				items = append(items, CompletionItem{
					Label:            name,
					Kind:             itemKind,
					Detail:           fmt.Sprintf("(%s) %s: %s", kindStr, name, typeStr),
					Documentation:    MarkupContent{Kind: "markdown", Value: fmt.Sprintf("Declared at line %d", idx+1)},
					InsertText:       name,
					InsertTextFormat: InsertTextFormatPlainText,
					SortText:         fmt.Sprintf("01_%s", name),
				})
			}
		}

		// 3. for <item> in ...
		if m := reForLoop.FindStringSubmatch(line); len(m) > 1 {
			name := m[1]
			if !seen[name] {
				seen[name] = true
				items = append(items, CompletionItem{
					Label:            name,
					Kind:             CompletionItemKindVariable,
					Detail:           fmt.Sprintf("(loop variable) %s", name),
					Documentation:    MarkupContent{Kind: "markdown", Value: fmt.Sprintf("For loop iteration variable at line %d", idx+1)},
					InsertText:       name,
					InsertTextFormat: InsertTextFormatPlainText,
					SortText:         fmt.Sprintf("01_%s", name),
				})
			}
		}

		// 4. func
		if m := reFunc.FindStringSubmatch(line); len(m) > 1 {
			name := m[1]
			params := strings.TrimSpace(m[2])
			retType := strings.TrimSpace(m[3])
			if retType == "" {
				retType = "void"
			}
			if !seen[name] {
				seen[name] = true
				items = append(items, CompletionItem{
					Label:            name,
					Kind:             CompletionItemKindFunction,
					Detail:           fmt.Sprintf("func %s(%s) -> %s", name, params, retType),
					Documentation:    MarkupContent{Kind: "markdown", Value: fmt.Sprintf("Function defined at line %d", idx+1)},
					InsertText:       name,
					InsertTextFormat: InsertTextFormatPlainText,
					SortText:         fmt.Sprintf("02_%s", name),
				})
			}
		}

		// 5. unit
		if m := reUnit.FindStringSubmatch(line); len(m) > 1 {
			name := m[1]
			if !seen[name] {
				seen[name] = true
				items = append(items, CompletionItem{
					Label:            name,
					Kind:             CompletionItemKindClass,
					Detail:           fmt.Sprintf("unit %s", name),
					Documentation:    MarkupContent{Kind: "markdown", Value: fmt.Sprintf("Unit defined at line %d", idx+1)},
					InsertText:       name,
					InsertTextFormat: InsertTextFormatPlainText,
					SortText:         fmt.Sprintf("03_%s", name),
				})
			}
		}

		// 6. declare &prop inside unit
		if m := reDeclare.FindStringSubmatch(line); len(m) > 1 {
			name := m[1]
			typeStr := strings.TrimSpace(m[2])
			if typeStr == "" {
				typeStr = "any"
			}
			ampName := "&" + name
			if !seen[ampName] {
				seen[ampName] = true
				items = append(items, CompletionItem{
					Label:            ampName,
					Kind:             CompletionItemKindField,
					Detail:           fmt.Sprintf("(&prop) %s: %s", name, typeStr),
					Documentation:    MarkupContent{Kind: "markdown", Value: fmt.Sprintf("Internal unit property at line %d", idx+1)},
					InsertText:       ampName,
					InsertTextFormat: InsertTextFormatPlainText,
					SortText:         fmt.Sprintf("01_%s", name),
				})
			}
		}

		// 7. use <module> as <alias>
		if m := reUse.FindStringSubmatch(line); len(m) > 1 {
			name := m[1]
			if len(m) > 2 && m[2] != "" {
				name = m[2]
			}
			if !seen[name] {
				seen[name] = true
				items = append(items, CompletionItem{
					Label:            name,
					Kind:             CompletionItemKindModule,
					Detail:           fmt.Sprintf("(imported module) %s", name),
					Documentation:    MarkupContent{Kind: "markdown", Value: fmt.Sprintf("Imported module at line %d", idx+1)},
					InsertText:       name,
					InsertTextFormat: InsertTextFormatPlainText,
					SortText:         fmt.Sprintf("03_%s", name),
				})
			}
		}

		// 8. from "..." use ...
		if m := reFromUse.FindStringSubmatch(line); len(m) > 1 {
			usePart := strings.TrimSpace(m[1])
			if strings.HasPrefix(usePart, "(") && strings.HasSuffix(usePart, ")") {
				inner := strings.Trim(usePart, "()")
				for _, piece := range strings.Split(inner, ",") {
					item := strings.TrimSpace(piece)
					if item != "" && !seen[item] {
						seen[item] = true
						items = append(items, CompletionItem{
							Label:            item,
							Kind:             CompletionItemKindModule,
							Detail:           fmt.Sprintf("(imported module) %s", item),
							Documentation:    MarkupContent{Kind: "markdown", Value: fmt.Sprintf("Imported module at line %d", idx+1)},
							InsertText:       item,
							InsertTextFormat: InsertTextFormatPlainText,
							SortText:         fmt.Sprintf("03_%s", item),
						})
					}
				}
			} else if colonIdx := strings.Index(usePart, "::"); colonIdx != -1 {
				member := strings.TrimSpace(usePart[colonIdx+2:])
				if member != "" && !seen[member] {
					seen[member] = true
					items = append(items, CompletionItem{
						Label:            member,
						Kind:             CompletionItemKindFunction,
						Detail:           fmt.Sprintf("(imported member) %s", member),
						Documentation:    MarkupContent{Kind: "markdown", Value: fmt.Sprintf("Imported member from module at line %d", idx+1)},
						InsertText:       member,
						InsertTextFormat: InsertTextFormatPlainText,
						SortText:         fmt.Sprintf("02_%s", member),
					})
				}
			} else {
				parts := strings.Fields(usePart)
				if len(parts) > 0 {
					name := parts[0]
					if len(parts) >= 3 && parts[1] == "as" {
						name = parts[2]
					}
					if name != "" && !seen[name] {
						seen[name] = true
						items = append(items, CompletionItem{
							Label:            name,
							Kind:             CompletionItemKindModule,
							Detail:           fmt.Sprintf("(imported module) %s", name),
							Documentation:    MarkupContent{Kind: "markdown", Value: fmt.Sprintf("Imported module at line %d", idx+1)},
							InsertText:       name,
							InsertTextFormat: InsertTextFormatPlainText,
							SortText:         fmt.Sprintf("03_%s", name),
						})
					}
				}
			}
		}
	}

	return items
}
