package lsp

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Nehonix-Team/xru/pkg/modules"
)

func (s *Server) handleCompletion(uri string, pos Position) *CompletionList {
	text, ok := s.docs[uri]
	if !ok {
		return &CompletionList{Items: []CompletionItem{}}
	}

	lines := strings.Split(text, "\n")
	if pos.Line < 0 || pos.Line >= len(lines) {
		return &CompletionList{Items: []CompletionItem{}}
	}

	line := lines[pos.Line]
	col := pos.Character
	if col > len(line) {
		col = len(line)
	}
	prefixLine := line[:col]
	trimmedPrefix := strings.TrimSpace(prefixLine)

	var items []CompletionItem

	// 0. Handle `from` statement autocompletion (modules and namespaces from _mod.xru)
	if strings.HasPrefix(trimmedPrefix, "from") {
		fromItems := s.handleFromCompletion(uri, prefixLine, trimmedPrefix)
		if len(fromItems) > 0 {
			return &CompletionList{Items: fromItems}
		}
	}

	// 0.1 Handle double colon `::` member completion (e.g. auth::login)
	if colonIdx := strings.LastIndex(prefixLine, "::"); colonIdx != -1 {
		memberItems := s.handleScopeMemberCompletion(uri, prefixLine[:colonIdx])
		return &CompletionList{Items: memberItems}
	}

	// 0.2 Handle `use ` statement for builtin units
	if strings.HasPrefix(trimmedPrefix, "use ") || trimmedPrefix == "use" {
		builtins := []struct {
			name, insert, doc string
		}{
			{"std", "std", "Standard IO & formatting unit (std.println, std.log, etc.)"},
			{"net", "net as Net", "Network & HTTP client unit"},
			{"fs", "fs as Fs", "Filesystem operations unit"},
			{"sys", "sys as Sys", "Operating system & environment unit"},
			{"crypto", "crypto as Crypto", "Cryptographic hashes & encryption unit"},
		}
		for _, b := range builtins {
			items = append(items, CompletionItem{
				Label:            b.name,
				Kind:             CompletionItemKindModule,
				Detail:           fmt.Sprintf("use %s", b.insert),
				Documentation:    MarkupContent{Kind: "markdown", Value: b.doc},
				InsertText:       b.insert,
				InsertTextFormat: InsertTextFormatPlainText,
			})
		}
		return &CompletionList{Items: items}
	}

	// 1. Context Check: Inside a `with Net.fetch(...) {` block
	if isInsideWithNet(lines, pos.Line) {
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
			return &CompletionList{Items: items}
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
			return &CompletionList{Items: items}
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
		return &CompletionList{Items: items}
	}

	// 2. Member completion: e.g. "Utils." or "Net." or "Crypto." or "Fs." or "cdn."
	dotIdx := strings.LastIndex(prefixLine, ".")
	if dotIdx >= 0 {
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

		if unitName != "" {
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
				return &CompletionList{IsIncomplete: false, Items: items}
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
							return &CompletionList{IsIncomplete: false, Items: items}
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
					return &CompletionList{IsIncomplete: false, Items: items}
				}

				if varInfo.unitType != "" {
					unitMembers := collectUnitMembers(lines, varInfo.unitType)
					if len(unitMembers) > 0 {
						return &CompletionList{IsIncomplete: false, Items: unitMembers}
					}
				}
			}

			// 2.D Local unit definition direct access (e.g. TemplateManager.)
			unitMembers := collectUnitMembers(lines, unitName)
			if len(unitMembers) > 0 {
				return &CompletionList{IsIncomplete: false, Items: unitMembers}
			}

			// 2.E Strictly return member items or empty list.
			// Do NOT fall through to global snippets or keywords after dot!
			return &CompletionList{IsIncomplete: false, Items: items}
		}
	}

	// 2.5 Local Symbols from current document (variables, constants, functions, params, local imports)
	localItems := collectLocalSymbols(lines, pos.Line)
	items = append(items, localItems...)

	// 3. Syntax Snippets (Declarations, control flow, templates)
	for _, snip := range SyntaxSnippets {
		items = append(items, CompletionItem{
			Label:            snip.Name,
			Kind:             CompletionItemKindSnippet,
			Detail:           snip.Detail,
			Documentation:    MarkupContent{Kind: "markdown", Value: snip.Doc},
			InsertText:       snip.Snippet,
			InsertTextFormat: InsertTextFormatSnippet,
			SortText:         fmt.Sprintf("30_%s", snip.Name),
		})
	}

	// Déterminer la position d'insertion pour l'import (en haut du fichier)
	importInsertLine := 0
	for idx, l := range lines {
		trimmed := strings.TrimSpace(l)
		if strings.HasPrefix(trimmed, "use ") || strings.HasPrefix(trimmed, "from ") || strings.HasPrefix(trimmed, "mod ") {
			importInsertLine = idx + 1
		}
	}

	// 4. Units completion (with auto-import for unimported builtins)
	for _, uDoc := range BuiltinUnitsDocs {
		unitName := strings.TrimPrefix(uDoc.Detail, "unit ")
		var edits []TextEdit
		var importStatement string
		switch strings.ToLower(unitName) {
		case "std":
			importStatement = "use std"
		case "net":
			importStatement = "use net as Net"
		case "fs":
			importStatement = "use fs as Fs"
		case "sys":
			importStatement = "use sys as Sys"
		case "crypto":
			importStatement = "use crypto as Crypto"
		}

		alreadyImported := false
		if importStatement != "" {
			for _, l := range lines {
				if strings.Contains(l, importStatement) {
					alreadyImported = true
					break
				}
			}
			if !alreadyImported {
				edits = append(edits, TextEdit{
					Range: Range{
						Start: Position{Line: importInsertLine, Character: 0},
						End:   Position{Line: importInsertLine, Character: 0},
					},
					NewText: importStatement + "\n",
				})
			}
		}

		detail := uDoc.Detail
		if !alreadyImported && importStatement != "" {
			detail = fmt.Sprintf("%s (auto-import: %s)", uDoc.Detail, importStatement)
		}

		items = append(items, CompletionItem{
			Label:               unitName,
			Kind:                CompletionItemKindClass,
			Detail:              detail,
			Documentation:       MarkupContent{Kind: "markdown", Value: uDoc.Doc},
			InsertText:          unitName,
			InsertTextFormat:    InsertTextFormatPlainText,
			SortText:            fmt.Sprintf("10_%s", unitName),
			AdditionalTextEdits: edits,
		})
	}

	// 5. Surgical file methods
	for sName, sDoc := range SurgicalMethodsDocs {
		items = append(items, CompletionItem{
			Label:            sName,
			Kind:             CompletionItemKindFunction,
			Detail:           sDoc.Detail,
			Documentation:    MarkupContent{Kind: "markdown", Value: sDoc.Doc},
			InsertText:       sName,
			InsertTextFormat: InsertTextFormatPlainText,
			SortText:         fmt.Sprintf("25_%s", sName),
		})
	}

	// 6. Keywords
	for kwName, kwDoc := range KeywordsDocs {
		items = append(items, CompletionItem{
			Label:            kwName,
			Kind:             CompletionItemKindKeyword,
			Detail:           kwDoc.Detail,
			Documentation:    MarkupContent{Kind: "markdown", Value: kwDoc.Doc},
			InsertText:       kwName,
			InsertTextFormat: InsertTextFormatPlainText,
			SortText:         fmt.Sprintf("20_%s", kwName),
		})
	}

	// 7. Auto-import from _mod.xru mapped modules
	filePath := uriToPath(uri)
	fileDir := filepath.Dir(filePath)
	modFilePath := modules.FindModConfigFile(fileDir)
	if modFilePath != "" {
		if modCfg, errs := modules.LoadModConfig(modFilePath); len(errs) == 0 && modCfg != nil {
			candidates := modCfg.GetAllExportedCandidates()

			for _, cand := range candidates {
				// Vérifier si le symbole n'est pas déjà importé ou présent dans le fichier
				alreadyImported := false
				for _, l := range lines {
					if strings.Contains(l, cand.ImportStatement) {
						alreadyImported = true
						break
					}
				}
				if alreadyImported {
					continue
				}

				kind := CompletionItemKindFunction
				if cand.Kind == "var" || cand.Kind == "const" {
					kind = CompletionItemKindVariable
				} else if cand.Kind == "module" {
					kind = CompletionItemKindModule
				}

				item := CompletionItem{
					Label:            cand.Name,
					Kind:             kind,
					Detail:           fmt.Sprintf("(auto import) %s", cand.ImportStatement),
					Documentation:    MarkupContent{Kind: "markdown", Value: fmt.Sprintf("Auto-imports `%s` from mapped namespace **%s** (%s)", cand.Name, cand.Namespace, cand.ModuleName)},
					InsertText:       cand.Name,
					InsertTextFormat: InsertTextFormatPlainText,
					SortText:         fmt.Sprintf("40_%s", cand.Name),
					AdditionalTextEdits: []TextEdit{
						{
							Range: Range{
								Start: Position{Line: importInsertLine, Character: 0},
								End:   Position{Line: importInsertLine, Character: 0},
							},
							NewText: cand.ImportStatement + "\n",
						},
					},
				}
				items = append(items, item)
			}
		}
	}

	return &CompletionList{IsIncomplete: false, Items: items}
}

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

func (s *Server) handleFromCompletion(uri, prefixLine, trimmedPrefix string) []CompletionItem {
	var items []CompletionItem

	filePath := uriToPath(uri)
	fileDir := filepath.Dir(filePath)
	modFilePath := modules.FindModConfigFile(fileDir)
	if modFilePath == "" {
		return items
	}

	modCfg, errs := modules.LoadModConfig(modFilePath)
	if len(errs) > 0 || modCfg == nil {
		return items
	}

	// Case A: contains "use" -> completing module or module member
	useIdx := strings.Index(trimmedPrefix, " use")
	if useIdx != -1 {
		beforeUse := strings.TrimSpace(trimmedPrefix[:useIdx])
		ns := strings.TrimPrefix(beforeUse, "from")
		ns = strings.TrimSpace(ns)
		ns = strings.Trim(ns, "\"")

		afterUse := strings.TrimSpace(trimmedPrefix[useIdx+4:])

		// Check if there is "::"
		if colonIdx := strings.Index(afterUse, "::"); colonIdx != -1 {
			modName := strings.TrimSpace(afterUse[:colonIdx])
			modDetails, err := modCfg.ResolveModule(ns, modName)
			if err == nil && modDetails != nil {
				for _, exp := range modDetails.Exports {
					kind := CompletionItemKindFunction
					if exp.Kind == "var" || exp.Kind == "const" {
						kind = CompletionItemKindVariable
					}
					items = append(items, CompletionItem{
						Label:            exp.Name,
						Kind:             kind,
						Detail:           fmt.Sprintf("%s %s (%s)", exp.Kind, exp.Name, exp.Type),
						Documentation:    MarkupContent{Kind: "markdown", Value: fmt.Sprintf("Exported from `%s` in namespace `%s`", modName, ns)},
						InsertText:       exp.Name,
						InsertTextFormat: InsertTextFormatPlainText,
					})
				}
			}
			return items
		}

		// User is completing module name in namespace
		mods, err := modCfg.GetNamespaceModules(ns)
		if err == nil {
			for _, m := range mods {
				items = append(items, CompletionItem{
					Label:            m.Name,
					Kind:             CompletionItemKindModule,
					Detail:           fmt.Sprintf("module in namespace \"%s\"", ns),
					Documentation:    MarkupContent{Kind: "markdown", Value: fmt.Sprintf("Defined at `%s`", m.FilePath)},
					InsertText:       m.Name,
					InsertTextFormat: InsertTextFormatPlainText,
				})
			}
		}
		return items
	}

	// Case B: No "use" yet -> completing namespace from _mod.xru
	quoteCount := strings.Count(prefixLine, "\"")

	for ns, targetPath := range modCfg.Mappings {
		doc := fmt.Sprintf("Namespace mapped to `%s` in `_mod.xru`", targetPath)

		if quoteCount%2 == 1 {
			// Opened quote: `from "` or `from "ser`
			items = append(items, CompletionItem{
				Label:            ns,
				Kind:             CompletionItemKindFolder,
				Detail:           fmt.Sprintf("namespace (%s)", targetPath),
				Documentation:    MarkupContent{Kind: "markdown", Value: doc},
				InsertText:       fmt.Sprintf("%s\" use ", ns),
				FilterText:       ns,
				InsertTextFormat: InsertTextFormatPlainText,
			})
		} else if strings.HasSuffix(strings.TrimSpace(prefixLine), "\"") {
			// Quote already closed: e.g. `from "services"`
			items = append(items, CompletionItem{
				Label:            "use",
				Kind:             CompletionItemKindKeyword,
				Detail:           "use <module>",
				Documentation:    MarkupContent{Kind: "markdown", Value: "Specify module to import"},
				InsertText:       " use ",
				InsertTextFormat: InsertTextFormatPlainText,
			})
			return items
		} else {
			// No quotes: e.g. `from` or `from `
			items = append(items, CompletionItem{
				Label:            fmt.Sprintf("\"%s\"", ns),
				Kind:             CompletionItemKindFolder,
				Detail:           fmt.Sprintf("namespace (%s)", targetPath),
				Documentation:    MarkupContent{Kind: "markdown", Value: doc},
				InsertText:       fmt.Sprintf("\"%s\" use ", ns),
				FilterText:       ns,
				InsertTextFormat: InsertTextFormatPlainText,
			})
		}
	}

	return items
}

func (s *Server) handleScopeMemberCompletion(uri, beforeColon string) []CompletionItem {
	var items []CompletionItem
	trimmed := strings.TrimSpace(beforeColon)
	var modIdent string
	for i := len(trimmed) - 1; i >= 0; i-- {
		if !isWordChar(trimmed[i]) {
			modIdent = trimmed[i+1:]
			break
		}
		if i == 0 {
			modIdent = trimmed
		}
	}
	if modIdent == "" {
		return items
	}

	filePath := uriToPath(uri)
	fileDir := filepath.Dir(filePath)
	modFilePath := modules.FindModConfigFile(fileDir)
	if modFilePath == "" {
		return items
	}

	modCfg, errs := modules.LoadModConfig(modFilePath)
	if len(errs) > 0 || modCfg == nil {
		return items
	}

	for ns := range modCfg.Mappings {
		mDetails, err := modCfg.ResolveModule(ns, modIdent)
		if err == nil && mDetails != nil {
			for _, exp := range mDetails.Exports {
				kind := CompletionItemKindFunction
				if exp.Kind == "var" || exp.Kind == "const" {
					kind = CompletionItemKindVariable
				}
				items = append(items, CompletionItem{
					Label:            exp.Name,
					Kind:             kind,
					Detail:           fmt.Sprintf("%s %s (%s)", exp.Kind, exp.Name, exp.Type),
					Documentation:    MarkupContent{Kind: "markdown", Value: fmt.Sprintf("Exported from `%s` in namespace `%s`", modIdent, ns)},
					InsertText:       exp.Name,
					InsertTextFormat: InsertTextFormatPlainText,
				})
			}
			return items
		}
	}
	return items
}

func resolveUnitAlias(lines []string, alias string) string {
	re := regexp.MustCompile(fmt.Sprintf(`\buse\s+([a-zA-Z0-9_]+)\s+as\s+%s\b`, regexp.QuoteMeta(alias)))
	for _, l := range lines {
		if m := re.FindStringSubmatch(l); len(m) > 1 {
			return m[1]
		}
	}
	reFrom := regexp.MustCompile(fmt.Sprintf(`\bfrom\s+"[^"]+"\s+use\s+([a-zA-Z0-9_]+)\s+as\s+%s\b`, regexp.QuoteMeta(alias)))
	for _, l := range lines {
		if m := reFrom.FindStringSubmatch(l); len(m) > 1 {
			return m[1]
		}
	}
	return alias
}

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

		// 3. func
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

		// 4. unit
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

		// 5. declare &prop inside unit
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

		// 6. use <module> as <alias>
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

		// 7. from "..." use ...
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

type varTargetInfo struct {
	isTask         bool
	taskReturnType string
	unitType       string
}

func findVariableTargetInfo(lines []string, varName string) *varTargetInfo {
	reVar := regexp.MustCompile(fmt.Sprintf(`^\s*(?:global\s+)?(?:let|const)\s+%s(?:\s*:\s*([^=]+?))?(?:\s*=\s*(.+))?$`, regexp.QuoteMeta(varName)))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*") {
			continue
		}
		if m := reVar.FindStringSubmatch(line); len(m) > 1 {
			typeStr := strings.TrimSpace(m[1])
			valStr := strings.TrimSpace(m[2])

			// 1. Explicit task type
			if strings.HasPrefix(typeStr, "task[") && strings.HasSuffix(typeStr, "]") {
				inner := strings.TrimSuffix(strings.TrimPrefix(typeStr, "task["), "]")
				return &varTargetInfo{isTask: true, taskReturnType: inner}
			} else if typeStr == "task" {
				return &varTargetInfo{isTask: true, taskReturnType: "any"}
			}

			// 2. Assigned with `with <expr>`
			if strings.HasPrefix(valStr, "with ") {
				withTarget := strings.TrimSpace(strings.TrimPrefix(valStr, "with "))
				retType := "any"
				if strings.Contains(strings.ToLower(withTarget), "net.fetch") {
					retType = "string"
				} else if parenIdx := strings.Index(withTarget, "("); parenIdx != -1 {
					memberPart := withTarget[:parenIdx]
					if dot := strings.LastIndex(memberPart, "."); dot != -1 {
						callObj := strings.TrimSpace(memberPart[:dot])
						callMethod := strings.TrimSpace(memberPart[dot+1:])
						retType = inferUnitMethodReturnType(lines, callObj, callMethod)
					}
				}
				if typeStr != "" && typeStr != "any" {
					retType = typeStr
				}
				return &varTargetInfo{isTask: true, taskReturnType: retType}
			}

			// 3. Instantiated via @UnitName()
			if strings.HasPrefix(valStr, "@") {
				unitName := strings.TrimPrefix(valStr, "@")
				if parenIdx := strings.Index(unitName, "("); parenIdx != -1 {
					unitName = unitName[:parenIdx]
				}
				return &varTargetInfo{unitType: strings.TrimSpace(unitName)}
			}

			// 4. Custom unit type in annotation
			if typeStr != "" && typeStr != "string" && typeStr != "number" && typeStr != "bool" && typeStr != "duration" && typeStr != "any" {
				return &varTargetInfo{unitType: typeStr}
			}
		}
	}
	return nil
}

func inferUnitMethodReturnType(lines []string, objName, methodName string) string {
	unitName := objName
	if varInfo := findVariableTargetInfo(lines, objName); varInfo != nil && varInfo.unitType != "" {
		unitName = varInfo.unitType
	}

	// 1. Built-in units
	if unitDoc, ok := BuiltinUnitsDocs[strings.ToLower(unitName)]; ok {
		if mDoc, ok := unitDoc.Methods[strings.ToLower(methodName)]; ok {
			if arrowIdx := strings.Index(mDoc.Detail, "->"); arrowIdx != -1 {
				return strings.TrimSpace(mDoc.Detail[arrowIdx+2:])
			}
		}
	}

	// 2. Local unit declaration in document
	reUnitStart := regexp.MustCompile(fmt.Sprintf(`^\s*unit\s+%s\b`, regexp.QuoteMeta(unitName)))
	reFunc := regexp.MustCompile(fmt.Sprintf(`^\s*(?:global\s+)?func\s+%s\s*\(.*?\)(?:\s*->\s*([a-zA-Z0-9_\[\]]+))?`, regexp.QuoteMeta(methodName)))

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

		if m := reFunc.FindStringSubmatch(l); len(m) > 0 {
			if len(m) > 1 && m[1] != "" {
				return strings.TrimSpace(m[1])
			}
			return "void"
		}
	}

	return "any"
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


