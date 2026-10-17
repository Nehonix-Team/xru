package lsp

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Nehonix-Team/xru/pkg/modules"
)

func (s *Server) handleMemberCompletion(uri string, lines []string, lineIdx int, prefixLine string) (*CompletionList, bool) {
	dotIdx := strings.LastIndex(prefixLine, ".")
	if dotIdx < 0 {
		return nil, false
	}

	// S'assurer que le point correspond bien à un accès membre en cours de saisie,
	// c'est-à-dire que le texte après le point ne contient que des caractères d'identifiant (sans (, ", {, etc.)
	afterDot := prefixLine[dotIdx+1:]
	for i := 0; i < len(afterDot); i++ {
		if !isWordChar(afterDot[i]) {
			return nil, false
		}
	}

	rawBefore := prefixLine[:dotIdx]
	target := extractTargetBeforeDotMultiline(lines, lineIdx, rawBefore)
	if target == "" {
		return nil, false
	}

	filePath := uriToPath(uri)
	fileDir := filepath.Dir(filePath)

	// 1. Resolve expression before dot (literals, chained calls, local variables, properties, etc.)
	targetInfo := resolveExpressionTargetInfo(uri, lines, target)
	if targetInfo != nil {
		if targetInfo.isTask {
			retT := targetInfo.taskReturnType
			if retT == "" {
				retT = "any"
			}
			var items []CompletionItem
			items = append(items, CompletionItem{
				Label:            "wait",
				Kind:             CompletionItemKindMethod,
				Detail:           fmt.Sprintf("wait() -> %s", retT),
				Documentation:    MarkupContent{Kind: "markdown", Value: fmt.Sprintf("Waits synchronously for the background task to complete and returns `%s`.", retT)},
				InsertText:       "wait()",
				InsertTextFormat: InsertTextFormatPlainText,
				SortText:         "00_wait",
			})
			items = append(items, CompletionItem{
				Label:            "isDone",
				Kind:             CompletionItemKindMethod,
				Detail:           "isDone() -> bool",
				Documentation:    MarkupContent{Kind: "markdown", Value: "Checks whether the background task has completed (non-blocking)."},
				InsertText:       "isDone()",
				InsertTextFormat: InsertTextFormatPlainText,
				SortText:         "01_isDone",
			})
			items = append(items, CompletionItem{
				Label:            "cancel",
				Kind:             CompletionItemKindMethod,
				Detail:           "cancel() -> void",
				Documentation:    MarkupContent{Kind: "markdown", Value: "Cancels the background task execution."},
				InsertText:       "cancel()",
				InsertTextFormat: InsertTextFormatPlainText,
				SortText:         "02_cancel",
			})
			items = append(items, CompletionItem{
				Label:            "result",
				Kind:             CompletionItemKindMethod,
				Detail:           fmt.Sprintf("result() -> %s", retT),
				Documentation:    MarkupContent{Kind: "markdown", Value: fmt.Sprintf("Alias for `wait()`. Returns `%s`.", retT)},
				InsertText:       "result()",
				InsertTextFormat: InsertTextFormatPlainText,
				SortText:         "03_result",
			})
			return &CompletionList{IsIncomplete: false, Items: items}, true
		}

		if targetInfo.isString {
			return &CompletionList{IsIncomplete: false, Items: getStringCompletionItems()}, true
		}
		if targetInfo.isNumber {
			return &CompletionList{IsIncomplete: false, Items: getNumberCompletionItems()}, true
		}
		if targetInfo.isList {
			return &CompletionList{IsIncomplete: false, Items: getListCompletionItems()}, true
		}

		// Module imported as map: from "..." use mod::@map as alias
		if targetInfo.fromModuleName != "" {
			if modFilePath := modules.FindModConfigFile(fileDir); modFilePath != "" {
				if modCfg, errs := modules.LoadModConfig(modFilePath); len(errs) == 0 && modCfg != nil {
					mDetails, err := modCfg.ResolveModule(targetInfo.fromModuleSource, targetInfo.fromModuleName)
					if err == nil && mDetails != nil {
						var modItems []CompletionItem
						for _, exp := range mDetails.Exports {
							kind := CompletionItemKindProperty
							insertText := exp.Name
							detail := fmt.Sprintf("%s %s: %s", exp.Kind, exp.Name, exp.Type)
							if exp.Kind == "func" {
								kind = CompletionItemKindMethod
								insertText = exp.Name + "()"
								detail = fmt.Sprintf("func %s(...) %s", exp.Name, exp.Type)
							}
							modItems = append(modItems, CompletionItem{
								Label:            exp.Name,
								Kind:             kind,
								Detail:           detail,
								Documentation:    MarkupContent{Kind: "markdown", Value: fmt.Sprintf("Field from module `%s::@map`", mDetails.Name)},
								InsertText:       insertText,
								InsertTextFormat: InsertTextFormatPlainText,
								SortText:         fmt.Sprintf("00_%s", exp.Name),
							})
						}
						return &CompletionList{IsIncomplete: false, Items: modItems}, true
					}
				}
			}
		}

		if targetInfo.interfaceName != "" {
			ifaceMembers := collectInterfaceMembers(lines, targetInfo.interfaceName, filePath)
			if len(ifaceMembers) > 0 {
				return &CompletionList{IsIncomplete: false, Items: ifaceMembers}, true
			}
		}

		if targetInfo.isMap {
			return &CompletionList{IsIncomplete: false, Items: getMapCompletionItems(targetInfo)}, true
		}

		if targetInfo.unitType != "" {
			unitMembers := collectUnitMembers(lines, targetInfo.unitType)
			if len(unitMembers) > 0 {
				return &CompletionList{IsIncomplete: false, Items: unitMembers}, true
			}
		}
	}

	// 2. Direct unit, interface, or workspace module name check
	resolvedName := resolveUnitAlias(lines, target)
	unitKey := strings.ToLower(resolvedName)

	// 2.A Built-in units (e.g. date., math., std., net., fs., sys., crypto.)
	if unitDoc, ok := BuiltinUnitsDocs[unitKey]; ok {
		var items []CompletionItem
		for mName, mDoc := range unitDoc.Methods {
			items = append(items, CompletionItem{
				Label:            mName,
				Kind:             CompletionItemKindMethod,
				Detail:           mDoc.Detail,
				Documentation:    MarkupContent{Kind: "markdown", Value: mDoc.Doc},
				InsertText:       mName,
				InsertTextFormat: InsertTextFormatPlainText,
				SortText:         fmt.Sprintf("00_%s", mName),
			})
		}
		return &CompletionList{IsIncomplete: false, Items: items}, true
	}

	// 2.B Direct Interface definition access (e.g. TExample.)
	if ifaceMembers := collectInterfaceMembers(lines, target, filePath); len(ifaceMembers) > 0 {
		return &CompletionList{IsIncomplete: false, Items: ifaceMembers}, true
	}

	// 2.C Direct Unit definition access (e.g. TemplateManager.)
	if unitMembers := collectUnitMembers(lines, target); len(unitMembers) > 0 {
		return &CompletionList{IsIncomplete: false, Items: unitMembers}, true
	}

	// 2.D Workspace module from _mod.xru (e.g. cdn., auth.)
	if modFilePath := modules.FindModConfigFile(fileDir); modFilePath != "" {
		if modCfg, errs := modules.LoadModConfig(modFilePath); len(errs) == 0 && modCfg != nil {
			for ns := range modCfg.Mappings {
				mDetails, err := modCfg.ResolveModule(ns, resolvedName)
				if err == nil && mDetails != nil {
					var items []CompletionItem
					for _, exp := range mDetails.Exports {
						kind := CompletionItemKindProperty
						insertText := exp.Name
						detail := fmt.Sprintf("%s %s: %s", exp.Kind, exp.Name, exp.Type)
						if exp.Kind == "func" {
							kind = CompletionItemKindMethod
							insertText = exp.Name + "()"
							detail = fmt.Sprintf("func %s(...) %s", exp.Name, exp.Type)
						}
						items = append(items, CompletionItem{
							Label:            exp.Name,
							Kind:             kind,
							Detail:           detail,
							Documentation:    MarkupContent{Kind: "markdown", Value: fmt.Sprintf("Exported from `%s` in namespace `%s`", mDetails.Name, ns)},
							InsertText:       insertText,
							InsertTextFormat: InsertTextFormatPlainText,
							SortText:         fmt.Sprintf("00_%s", exp.Name),
						})
					}
					return &CompletionList{IsIncomplete: false, Items: items}, true
				}
			}
		}
	}

	// 2.E Strictly return member items or empty list.
	// Do NOT fall through to global snippets or keywords after dot!
	return &CompletionList{IsIncomplete: false, Items: []CompletionItem{}}, true
}

func collectUnitMembers(lines []string, unitName string) []CompletionItem {
	var items []CompletionItem

	// 1. Built-in unit
	if unitDoc, ok := BuiltinUnitsDocs[strings.ToLower(unitName)]; ok {
		for mName, mDoc := range unitDoc.Methods {
			items = append(items, CompletionItem{
				Label:            mName,
				Kind:             CompletionItemKindMethod,
				Detail:           mDoc.Detail,
				Documentation:    MarkupContent{Kind: "markdown", Value: mDoc.Doc},
				InsertText:       mName + "()",
				InsertTextFormat: InsertTextFormatPlainText,
				SortText:         fmt.Sprintf("00_%s", mName),
			})
		}
		return items
	}

	// 2. Document unit definition
	reUnitStart := regexp.MustCompile(fmt.Sprintf(`^\s*unit\s+%s\b`, regexp.QuoteMeta(unitName)))
	reFunc := regexp.MustCompile(`^\s*(?:global\s+)?func\s+([a-zA-Z_][a-zA-Z0-9_]*)\s*\((.*?)\)(?:\s*->\s*([a-zA-Z0-9_\[\]]+))?`)
	reProp := regexp.MustCompile(`^\s*(?:prop|declare|let)\s+([a-zA-Z_][a-zA-Z0-9_]*)(?:\s*:\s*([^=]+?))?(?:\s*=|$)`)

	inUnit := false
	depth := 0
	for _, l := range lines {
		trimmed := strings.TrimSpace(l)
		if strings.HasPrefix(trimmed, "//") {
			continue
		}
		if !inUnit {
			if reUnitStart.MatchString(l) {
				inUnit = true
				depth += strings.Count(l, "{") - strings.Count(l, "}")
			}
			continue
		}

		depth += strings.Count(l, "{") - strings.Count(l, "}")
		if depth <= 0 {
			break
		}

		if m := reFunc.FindStringSubmatch(l); len(m) > 1 {
			fName := m[1]
			params := strings.TrimSpace(m[2])
			ret := strings.TrimSpace(m[3])
			if ret == "" {
				ret = "void"
			}
			items = append(items, CompletionItem{
				Label:            fName,
				Kind:             CompletionItemKindMethod,
				Detail:           fmt.Sprintf("func %s(%s) -> %s", fName, params, ret),
				Documentation:    MarkupContent{Kind: "markdown", Value: fmt.Sprintf("Method of unit `%s`", unitName)},
				InsertText:       fName + "()",
				InsertTextFormat: InsertTextFormatPlainText,
				SortText:         fmt.Sprintf("00_%s", fName),
			})
		}

		if m := reProp.FindStringSubmatch(l); len(m) > 1 {
			pName := m[1]
			pType := strings.TrimSpace(m[2])
			if pType == "" {
				pType = "any"
			}
			items = append(items, CompletionItem{
				Label:            pName,
				Kind:             CompletionItemKindProperty,
				Detail:           fmt.Sprintf("prop %s: %s", pName, pType),
				Documentation:    MarkupContent{Kind: "markdown", Value: fmt.Sprintf("Property of unit `%s`", unitName)},
				InsertText:       pName,
				InsertTextFormat: InsertTextFormatPlainText,
				SortText:         fmt.Sprintf("01_%s", pName),
			})
		}
	}

	// 3. Included interface methods (unit <unitName> includes <IfaceName>)
	reUnitIncludes := regexp.MustCompile(fmt.Sprintf(`^\s*unit\s+%s\s+(?:includes|extends)\s+([a-zA-Z0-9_]+)`, regexp.QuoteMeta(unitName)))
	for _, l := range lines {
		if m := reUnitIncludes.FindStringSubmatch(l); len(m) > 1 {
			ifaceMembers := collectInterfaceMembers(lines, m[1], "")
			items = append(items, ifaceMembers...)
			break
		}
	}

	return items
}

func getStringCompletionItems() []CompletionItem {
	var items []CompletionItem
	for mName, mDoc := range StringMethodsDocs {
		items = append(items, CompletionItem{
			Label:            mName,
			Kind:             CompletionItemKindMethod,
			Detail:           mDoc.Detail,
			Documentation:    MarkupContent{Kind: "markdown", Value: mDoc.Doc},
			InsertText:       mName + "()",
			InsertTextFormat: InsertTextFormatPlainText,
			SortText:         fmt.Sprintf("00_%s", mName),
		})
	}
	return items
}

func getListCompletionItems() []CompletionItem {
	var items []CompletionItem
	for mName, mDoc := range ListMethodsDocs {
		items = append(items, CompletionItem{
			Label:            mName,
			Kind:             CompletionItemKindMethod,
			Detail:           mDoc.Detail,
			Documentation:    MarkupContent{Kind: "markdown", Value: mDoc.Doc},
			InsertText:       mName + "()",
			InsertTextFormat: InsertTextFormatPlainText,
			SortText:         fmt.Sprintf("00_%s", mName),
		})
	}
	return items
}

func getNumberCompletionItems() []CompletionItem {
	var items []CompletionItem
	for mName, mDoc := range NumberMethodsDocs {
		items = append(items, CompletionItem{
			Label:            mName,
			Kind:             CompletionItemKindMethod,
			Detail:           mDoc.Detail,
			Documentation:    MarkupContent{Kind: "markdown", Value: mDoc.Doc},
			InsertText:       mName + "()",
			InsertTextFormat: InsertTextFormatPlainText,
			SortText:         fmt.Sprintf("00_%s", mName),
		})
	}
	return items
}

func getMapCompletionItems(varInfo *varTargetInfo) []CompletionItem {
	var items []CompletionItem
	if varInfo != nil && len(varInfo.mapFields) > 0 {
		items = append(items, varInfo.mapFields...)
	}
	for mName, mDoc := range MapMethodsDocs {
		items = append(items, CompletionItem{
			Label:            mName,
			Kind:             CompletionItemKindMethod,
			Detail:           mDoc.Detail,
			Documentation:    MarkupContent{Kind: "markdown", Value: mDoc.Doc},
			InsertText:       mName + "()",
			InsertTextFormat: InsertTextFormatPlainText,
			SortText:         fmt.Sprintf("10_%s", mName),
		})
	}
	return items
}

func extractTargetBeforeDot(rawBefore string) string {
	return extractTargetCandidate(rawBefore)
}

func extractTargetBeforeDotMultiline(lines []string, lineIdx int, rawBeforeCurLine string) string {
	rawBefore := rawBeforeCurLine
	currLine := lineIdx

	for {
		target := extractTargetCandidate(rawBefore)
		if target != "" || currLine <= 0 {
			if target != "" {
				return target
			}
			return strings.TrimSpace(rawBefore)
		}
		currLine--
		if currLine < 0 || currLine >= len(lines) {
			break
		}
		rawBefore = lines[currLine] + "\n" + rawBefore
		if len(rawBefore) > 10000 {
			break
		}
	}
	return strings.TrimSpace(rawBefore)
}

func extractTargetCandidate(rawBefore string) string {
	rawBefore = strings.TrimSpace(rawBefore)
	if rawBefore == "" {
		return ""
	}

	depthParen := 0
	depthBracket := 0
	depthBrace := 0
	inString := false
	var quoteChar byte

	for i := len(rawBefore) - 1; i >= 0; i-- {
		ch := rawBefore[i]

		if inString {
			if ch == quoteChar && (i == 0 || rawBefore[i-1] != '\\') {
				inString = false
			}
			continue
		}

		switch ch {
		case '"', '\'':
			inString = true
			quoteChar = ch
		case ')':
			depthParen++
		case '(':
			if depthParen > 0 {
				depthParen--
			} else {
				return strings.TrimSpace(rawBefore[i+1:])
			}
		case ']':
			depthBracket++
		case '[':
			if depthBracket > 0 {
				depthBracket--
			} else {
				return strings.TrimSpace(rawBefore[i+1:])
			}
		case '}':
			depthBrace++
		case '{':
			if depthBrace > 0 {
				depthBrace--
			} else {
				return strings.TrimSpace(rawBefore[i+1:])
			}
		default:
			if depthParen == 0 && depthBracket == 0 && depthBrace == 0 {
				// Expression or statement delimiters
				if ch == '=' || ch == ',' || ch == ';' ||
					ch == '+' || ch == '-' || ch == '*' || ch == '/' || ch == '%' ||
					ch == '!' || ch == '|' || ch == '^' ||
					ch == '<' || ch == '>' || ch == '?' || ch == '~' {
					return strings.TrimSpace(rawBefore[i+1:])
				}
				// Whitespace delimiter: if we already have non-empty expression after whitespace
				if ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' {
					after := strings.TrimSpace(rawBefore[i+1:])
					if after != "" {
						// If 'after' starts with '{', check if preceded by @map or map or @unit
						if strings.HasPrefix(after, "{") {
							prefix := strings.TrimSpace(rawBefore[:i+1])
							if strings.HasSuffix(prefix, "@map") || strings.HasSuffix(prefix, "map") || strings.HasSuffix(prefix, "@Map") {
								continue
							}
						}
						// If 'after' starts with '(', check if preceded by function name or method name
						if strings.HasPrefix(after, "(") {
							continue
						}
						return after
					}
				}
			}
		}
	}

	if depthParen > 0 || depthBracket > 0 || depthBrace > 0 {
		return ""
	}

	return strings.TrimSpace(rawBefore)
}


