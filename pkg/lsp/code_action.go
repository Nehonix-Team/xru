package lsp

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Nehonix-Team/xru/pkg/modules"
)

func (s *Server) handleCodeAction(uri string, rng Range, context CodeActionContext) []CodeAction {
	text, ok := s.docs[uri]
	if !ok {
		return []CodeAction{}
	}

	var actions []CodeAction
	seenTitles := make(map[string]bool)

	// 1. Diagnostic-based quick fixes (e.g. Variable 'std' is not defined)
	for _, diag := range context.Diagnostics {
		m := regexp.MustCompile(`Variable '([^']+)' is not defined`).FindStringSubmatch(diag.Message)
		if len(m) > 1 {
			symName := m[1]
			addImportActionsForSymbol(uri, text, symName, diag, &actions, seenTitles)
		}
	}

	// 2. Cursor word based quick fixes
	word := getWordAt(text, rng.Start.Line, rng.Start.Character)
	if word != "" {
		addImportActionsForSymbol(uri, text, word, Diagnostic{Range: rng}, &actions, seenTitles)
	}

	return actions
}

func addImportActionsForSymbol(uri, text, symName string, diag Diagnostic, actions *[]CodeAction, seen map[string]bool) {
	// Built-in units
	builtinsImports := map[string]string{
		"std":    "use std\n",
		"net":    "use net as Net\n",
		"Net":    "use net as Net\n",
		"fs":     "use fs as Fs\n",
		"Fs":     "use fs as Fs\n",
		"sys":    "use sys as Sys\n",
		"Sys":    "use sys as Sys\n",
		"crypto": "use crypto as Crypto\n",
		"Crypto": "use crypto as Crypto\n",
	}

	if importStmt, exists := builtinsImports[symName]; exists {
		if !strings.Contains(text, strings.TrimSpace(importStmt)) {
			title := fmt.Sprintf("Add '%s'", strings.TrimSpace(importStmt))
			if !seen[title] {
				seen[title] = true
				*actions = append(*actions, CodeAction{
					Title:       title,
					Kind:        "quickfix",
					IsPreferred: true,
					Diagnostics: []Diagnostic{diag},
					Edit: &WorkspaceEdit{
						Changes: map[string][]TextEdit{
							uri: {
								{
									Range: Range{
										Start: Position{Line: 0, Character: 0},
										End:   Position{Line: 0, Character: 0},
									},
									NewText: importStmt,
								},
							},
						},
					},
				})
			}
		}
	}

	// Modules and members from _mod.xru
	fileDir := filepath.Dir(uriToPath(uri))
	modFilePath := modules.FindModConfigFile(fileDir)
	if modFilePath != "" {
		if modCfg, errs := modules.LoadModConfig(modFilePath); len(errs) == 0 {
			candidates := modCfg.GetAllExportedCandidates()
			for _, cand := range candidates {
				// Module-level import
				if cand.Kind == "module" && cand.Name == symName {
					importStmt := cand.ImportStatement + "\n"
					if !strings.Contains(text, cand.ImportStatement) {
						title := fmt.Sprintf("Import module '%s' from \"%s\"", cand.Name, cand.Namespace)
						if !seen[title] {
							seen[title] = true
							*actions = append(*actions, CodeAction{
								Title:       title,
								Kind:        "quickfix",
								IsPreferred: true,
								Diagnostics: []Diagnostic{diag},
								Edit: &WorkspaceEdit{
									Changes: map[string][]TextEdit{
										uri: {
											{
												Range: Range{
													Start: Position{Line: 0, Character: 0},
													End:   Position{Line: 0, Character: 0},
												},
												NewText: importStmt,
											},
										},
									},
								},
							})
						}
					}
				}

				// Member-level import
				if cand.Kind != "module" && cand.Name == symName {
					importStmt := cand.ImportStatement + "\n"
					if !strings.Contains(text, cand.ImportStatement) {
						title := fmt.Sprintf("Import '%s' from \"%s\" (%s)", cand.Name, cand.Namespace, cand.ModuleName)
						if !seen[title] {
							seen[title] = true
							*actions = append(*actions, CodeAction{
								Title:       title,
								Kind:        "quickfix",
								IsPreferred: true,
								Diagnostics: []Diagnostic{diag},
								Edit: &WorkspaceEdit{
									Changes: map[string][]TextEdit{
										uri: {
											{
												Range: Range{
													Start: Position{Line: 0, Character: 0},
													End:   Position{Line: 0, Character: 0},
												},
												NewText: importStmt,
											},
										},
									},
								},
							})
						}
					}
				}
			}
		}
	}
}
