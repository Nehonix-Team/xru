package lsp

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/Nehonix-Team/xru/pkg/modules"
)

func (s *Server) collectGlobalCompletions(uri string, lines []string, importInsertLine int) []CompletionItem {
	var items []CompletionItem

	// 1. Syntax Snippets (Declarations, control flow, templates)
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

	// 2. Units completion (with auto-import for unimported builtins)
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

	// 3. Surgical file methods
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

	// 4. Keywords
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

	// 5. Auto-import from _mod.xru mapped modules
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

	return items
}
