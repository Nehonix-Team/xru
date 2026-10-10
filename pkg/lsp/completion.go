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
					})
				}
				return &CompletionList{IsIncomplete: true, Items: items}
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
								})
							}
							return &CompletionList{IsIncomplete: true, Items: items}
						}
					}
				}
			}

			// 2.C Strictly return member items or empty list.
			// Do NOT fall through to global snippets or keywords after dot!
			return &CompletionList{IsIncomplete: true, Items: items}
		}
	}

	// 3. Syntax Snippets (Declarations, control flow, templates)
	for _, snip := range SyntaxSnippets {
		items = append(items, CompletionItem{
			Label:            snip.Name,
			Kind:             CompletionItemKindSnippet,
			Detail:           snip.Detail,
			Documentation:    MarkupContent{Kind: "markdown", Value: snip.Doc},
			InsertText:       snip.Snippet,
			InsertTextFormat: InsertTextFormatSnippet,
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

	return &CompletionList{IsIncomplete: true, Items: items}
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
	return alias
}

