package lsp

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Nehonix-Team/xru/pkg/modules"
)

func (s *Server) handleMemberCompletion(uri string, lines []string, prefixLine string) (*CompletionList, bool) {
	dotIdx := strings.LastIndex(prefixLine, ".")
	if dotIdx < 0 {
		return nil, false
	}

	beforeDot := strings.TrimSpace(prefixLine[:dotIdx])
	var unitName string
	for i := len(beforeDot) - 1; i >= 0; i-- {
		if !isWordChar(beforeDot[i]) {
			unitName = beforeDot[i+1:]
			break
		}
		if i == 0 {
			unitName = beforeDot
		}
	}

	if unitName == "" {
		return nil, false
	}

	var items []CompletionItem

	// Resolve alias if imported via `use <unit> as <alias>`
	resolvedName := resolveUnitAlias(lines, unitName)
	unitKey := strings.ToLower(resolvedName)

	// 2.A Built-in units
	if unitDoc, ok := BuiltinUnitsDocs[unitKey]; ok {
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

	// 2.B Workspace module from _mod.xru (e.g. cdn., auth.)
	filePath := uriToPath(uri)
	fileDir := filepath.Dir(filePath)
	if modFilePath := modules.FindModConfigFile(fileDir); modFilePath != "" {
		if modCfg, errs := modules.LoadModConfig(modFilePath); len(errs) == 0 && modCfg != nil {
			for ns := range modCfg.Mappings {
				mDetails, err := modCfg.ResolveModule(ns, resolvedName)
				if err == nil && mDetails != nil {
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

	// 2.C Local variable check (Task or Unit Instance)
	if varInfo := findVariableTargetInfo(lines, unitName); varInfo != nil {
		if varInfo.isTask {
			retT := varInfo.taskReturnType
			if retT == "" {
				retT = "any"
			}
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

		if varInfo.unitType != "" {
			unitMembers := collectUnitMembers(lines, varInfo.unitType)
			if len(unitMembers) > 0 {
				return &CompletionList{IsIncomplete: false, Items: unitMembers}, true
			}
		}
	}

	// 2.D Local unit definition direct access (e.g. TemplateManager.)
	unitMembers := collectUnitMembers(lines, unitName)
	if len(unitMembers) > 0 {
		return &CompletionList{IsIncomplete: false, Items: unitMembers}, true
	}

	// 2.E Strictly return member items or empty list.
	// Do NOT fall through to global snippets or keywords after dot!
	return &CompletionList{IsIncomplete: false, Items: items}, true
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

	return items
}
