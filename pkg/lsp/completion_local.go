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
	reFileStmt := regexp.MustCompile(`^\s*file\s+.*?(?:,\s*)?\((.*?)\)\s*(?:->|=>)`)
	reLambda := regexp.MustCompile(`\((.*?)\)\s*(?:->|=>)\s*\{?$`)
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
					// 1. Check file statement
					if m := reFileStmt.FindStringSubmatch(lines[i]); len(m) > 1 {
						pList := strings.Split(m[1], ",")
						if len(pList) > 0 {
							docParam := strings.TrimSpace(pList[0])
							if docParam != "" {
								params = append(params, paramInfo{name: docParam, paramType: "Document"})
							}
						}
						if len(pList) > 1 {
							errParam := strings.TrimSpace(pList[1])
							if errParam != "" {
								params = append(params, paramInfo{name: errParam, paramType: "string"})
							}
						}
						return params
					}

					// 2. Check func statement
					if m := reFunc.FindStringSubmatch(lines[i]); len(m) > 1 {
						paramStr := strings.TrimSpace(m[1])
						if paramStr != "" {
							for _, p := range splitParamsRespectingBraces(paramStr) {
								p = strings.TrimSpace(p)
								colonIdx := strings.Index(p, ":")
								pName := p
								pType := "any"
								if colonIdx != -1 {
									pName = strings.TrimSpace(p[:colonIdx])
									pType = strings.TrimSpace(p[colonIdx+1:])
								}
								pName = strings.TrimSuffix(pName, "?")
								if pName != "" {
									params = append(params, paramInfo{name: pName, paramType: pType})
								}
							}
						}
						return params
					}

					// 3. Generic lambda (x, y) -> {
					if m := reLambda.FindStringSubmatch(lines[i]); len(m) > 1 {
						paramStr := strings.TrimSpace(m[1])
						if paramStr != "" {
							for _, p := range splitParamsRespectingBraces(paramStr) {
								p = strings.TrimSpace(p)
								colonIdx := strings.Index(p, ":")
								pName := p
								pType := "any"
								if colonIdx != -1 {
									pName = strings.TrimSpace(p[:colonIdx])
									pType = strings.TrimSpace(p[colonIdx+1:])
								}
								pName = strings.TrimSuffix(pName, "?")
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

	// 1. Paramètres de la fonction ou du bloc englobant courant
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
				Documentation:    MarkupContent{Kind: "markdown", Value: fmt.Sprintf("Parameter `%s` of type `%s` in current block", p.name, pType)},
				InsertText:       p.name,
				InsertTextFormat: InsertTextFormatPlainText,
				SortText:         fmt.Sprintf("00_%s", p.name),
			})
		}
	}

	reLetConst := regexp.MustCompile(`^\s*(?:global\s+)?(let|const|arg)\s+([a-zA-Z_][a-zA-Z0-9_]*)(?:\s*:\s*([^=]+?))?(?:\s*=|$)`)
	reFunc := regexp.MustCompile(`^\s*(?:global\s+)?func\s+([a-zA-Z_][a-zA-Z0-9_]*)\s*\((.*?)\)(?:\s*->\s*([a-zA-Z0-9_]+))?`)
	reUnit := regexp.MustCompile(`^\s*unit\s+([a-zA-Z_][a-zA-Z0-9_]*)`)
	reUse := regexp.MustCompile(`^\s*use\s+([a-zA-Z_][a-zA-Z0-9_]*)(?:\s+as\s+([a-zA-Z_][a-zA-Z0-9_]*))?`)
	reFromUse := regexp.MustCompile(`^\s*from\s+"[^"]+"\s+use\s+(.+)$`)
	reForLoop := regexp.MustCompile(`^\s*for\s+([a-zA-Z_][a-zA-Z0-9_]*)(?:\s*,\s*([a-zA-Z_][a-zA-Z0-9_]*))?\s+in\b`)
	reCatchErr := regexp.MustCompile(`\bcatch\s+([a-zA-Z_][a-zA-Z0-9_]*)`)

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
				if eqIdx := strings.Index(line, "="); eqIdx != -1 {
					valPart := strings.TrimSpace(line[eqIdx+1:])
					if strings.HasPrefix(valPart, "\"") || strings.HasPrefix(valPart, "'") {
						typeStr = "string"
					} else if valPart == "true" || valPart == "false" {
						typeStr = "bool"
					} else if strings.HasPrefix(valPart, "[") {
						typeStr = "list"
					} else if strings.HasPrefix(valPart, "@") {
						uName := strings.TrimPrefix(valPart, "@")
						if pIdx := strings.Index(uName, "("); pIdx != -1 {
							uName = uName[:pIdx]
						}
						typeStr = strings.TrimSpace(uName)
					} else if len(valPart) > 0 && (valPart[0] >= '0' && valPart[0] <= '9' || valPart[0] == '-') {
						typeStr = "number"
					} else {
						typeStr = "any"
					}
				} else {
					typeStr = "any"
				}
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

		// 2.5 catch <errVar>
		if m := reCatchErr.FindStringSubmatch(line); len(m) > 1 {
			errName := m[1]
			if errName != "or" && errName != "" && !seen[errName] {
				seen[errName] = true
				items = append(items, CompletionItem{
					Label:            errName,
					Kind:             CompletionItemKindVariable,
					Detail:           fmt.Sprintf("(error) %s: string", errName),
					Documentation:    MarkupContent{Kind: "markdown", Value: fmt.Sprintf("Error variable captured by `catch` at line %d", idx+1)},
					InsertText:       errName,
					InsertTextFormat: InsertTextFormatPlainText,
					SortText:         fmt.Sprintf("01_%s", errName),
				})
			}
		}

		// 3. for <item> [, <idx>] in ...
		if m := reForLoop.FindStringSubmatch(line); len(m) > 1 {
			itemName := m[1]
			if !seen[itemName] {
				seen[itemName] = true
				items = append(items, CompletionItem{
					Label:            itemName,
					Kind:             CompletionItemKindVariable,
					Detail:           fmt.Sprintf("(loop variable) %s", itemName),
					Documentation:    MarkupContent{Kind: "markdown", Value: fmt.Sprintf("For loop iteration variable at line %d", idx+1)},
					InsertText:       itemName,
					InsertTextFormat: InsertTextFormatPlainText,
					SortText:         fmt.Sprintf("01_%s", itemName),
				})
			}
			if len(m) > 2 && m[2] != "" {
				idxName := m[2]
				if !seen[idxName] {
					seen[idxName] = true
					items = append(items, CompletionItem{
						Label:            idxName,
						Kind:             CompletionItemKindVariable,
						Detail:           fmt.Sprintf("(loop index) %s: number", idxName),
						Documentation:    MarkupContent{Kind: "markdown", Value: fmt.Sprintf("For loop index variable at line %d", idx+1)},
						InsertText:       idxName,
						InsertTextFormat: InsertTextFormatPlainText,
						SortText:         fmt.Sprintf("01_%s", idxName),
					})
				}
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
		if pInfo := parseDeclareLine(line); pInfo != nil {
			ampName := "&" + pInfo.name
			if !seen[ampName] {
				seen[ampName] = true
				vis := "public"
				if pInfo.isPrivate {
					vis = "private"
				}
				items = append(items, CompletionItem{
					Label:            ampName,
					Kind:             CompletionItemKindField,
					Detail:           fmt.Sprintf("(&prop, %s) %s: %s", vis, pInfo.name, pInfo.propType),
					Documentation:    MarkupContent{Kind: "markdown", Value: fmt.Sprintf("Internal unit property at line %d", idx+1)},
					InsertText:       ampName,
					InsertTextFormat: InsertTextFormatPlainText,
					SortText:         fmt.Sprintf("01_%s", pInfo.name),
				})
			}
		}

		// 6b. unit definition -> &Unit:: and &self::
		if m := reUnit.FindStringSubmatch(line); len(m) > 1 {
			uName := m[1]
			unitScopeLabel := "&" + uName + "::"
			if !seen[unitScopeLabel] {
				seen[unitScopeLabel] = true
				items = append(items, CompletionItem{
					Label:            unitScopeLabel,
					Kind:             CompletionItemKindModule,
					Detail:           fmt.Sprintf("(&self) %s internal method reference", uName),
					Documentation:    MarkupContent{Kind: "markdown", Value: fmt.Sprintf("Access internal methods of unit `%s`", uName)},
					InsertText:       unitScopeLabel,
					InsertTextFormat: InsertTextFormatPlainText,
					SortText:         "00_self_unit",
				})
			}
			if !seen["&self::"] {
				seen["&self::"] = true
				items = append(items, CompletionItem{
					Label:            "&self::",
					Kind:             CompletionItemKindModule,
					Detail:           "(&self) internal method reference",
					Documentation:    MarkupContent{Kind: "markdown", Value: "Access internal methods of current unit"},
					InsertText:       "&self::",
					InsertTextFormat: InsertTextFormatPlainText,
					SortText:         "00_self_generic",
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
