package navigation

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/Nehonix-Team/xru/pkg/modules"
)

// resolveQualifiedDefinition resolves explicit "module::member" qualified expressions.
func (n *Navigator) resolveQualifiedDefinition(modCfg *modules.ModConfig, fileImports []FileImport, fullExpr, fileDir string) *Location {
	parts := strings.Split(fullExpr, "::")
	if len(parts) != 2 {
		return nil
	}
	targetMod := parts[0]
	targetWord := parts[1]

	var targetNs string
	for _, imp := range fileImports {
		if imp.Module == targetMod || imp.Alias == targetMod {
			targetNs = imp.Source
			if imp.Module != "" {
				targetMod = imp.Module
			}
			break
		}
	}

	modFile := n.ResolveModuleFilePath(modCfg, targetNs, targetMod, fileDir)
	if modFile != "" {
		lIdx, charPos, found := FindDeclarationInFile(modFile, targetWord)
		if found {
			return &Location{
				URI: pathToURI(modFile),
				Range: Range{
					Start: Position{Line: lIdx, Character: charPos},
					End:   Position{Line: lIdx, Character: charPos + len(targetWord)},
				},
			}
		}
	}
	if stdFile := n.findStdFile(fileDir, strings.ToLower(targetMod)); stdFile != "" {
		lIdx, charPos, found := FindDeclarationInFile(stdFile, targetWord)
		if found {
			return &Location{
				URI: pathToURI(stdFile),
				Range: Range{
					Start: Position{Line: lIdx, Character: charPos},
					End:   Position{Line: lIdx, Character: charPos + len(targetWord)},
				},
			}
		}
	}
	return nil
}

// resolveNamespaceDefinition resolves definitions for namespace identifiers declared in _mod.xru.
func resolveNamespaceDefinition(modCfg *modules.ModConfig, modFilePath, word string) *Location {
	if modCfg == nil || modFilePath == "" {
		return nil
	}
	if _, ok := modCfg.Mappings[word]; !ok {
		return nil
	}
	if modContent, err := os.ReadFile(modFilePath); err == nil {
		mLines := strings.Split(string(modContent), "\n")
		for lIdx, mLine := range mLines {
			if strings.Contains(mLine, word) {
				re := regexp.MustCompile(fmt.Sprintf(`\b(%s)\b`, regexp.QuoteMeta(word)))
				if loc := re.FindStringIndex(mLine); loc != nil {
					return &Location{
						URI: pathToURI(modFilePath),
						Range: Range{
							Start: Position{Line: lIdx, Character: loc[0]},
							End:   Position{Line: lIdx, Character: loc[1]},
						},
					}
				}
			}
		}
	}
	return &Location{
		URI: pathToURI(modFilePath),
		Range: Range{
			Start: Position{Line: 0, Character: 0},
			End:   Position{Line: 0, Character: 0},
		},
	}
}

// resolveImportLineDefinition resolves symbols when the cursor is directly on a use/from import statement.
func (n *Navigator) resolveImportLineDefinition(modCfg *modules.ModConfig, fileImports []FileImport, line int, word, fileDir string) *Location {
	for _, imp := range fileImports {
		if imp.Line != line {
			continue
		}
		isMember := false
		if imp.Member == word {
			isMember = true
		} else {
			for _, m := range imp.Members {
				if m == word {
					isMember = true
					break
				}
			}
		}
		if isMember {
			modFile := n.ResolveModuleFilePath(modCfg, imp.Source, imp.Module, fileDir)
			if modFile != "" {
				lIdx, charPos, found := FindDeclarationInFile(modFile, word)
				if found {
					return &Location{
						URI: pathToURI(modFile),
						Range: Range{
							Start: Position{Line: lIdx, Character: charPos},
							End:   Position{Line: lIdx, Character: charPos + len(word)},
						},
					}
				}
				return &Location{
					URI: pathToURI(modFile),
					Range: Range{
						Start: Position{Line: 0, Character: 0},
						End:   Position{Line: 0, Character: 0},
					},
				}
			}
		}

		if imp.Module == word {
			modFile := n.ResolveModuleFilePath(modCfg, imp.Source, imp.Module, fileDir)
			if modFile != "" {
				lIdx, charPos, _ := FindDeclarationInFile(modFile, imp.Module)
				return &Location{
					URI: pathToURI(modFile),
					Range: Range{
						Start: Position{Line: lIdx, Character: charPos},
						End:   Position{Line: lIdx, Character: charPos + len(imp.Module)},
					},
				}
			}
		}
	}
	return nil
}

// resolveImportedSymbolDefinition resolves definitions for symbols that were imported into the current file.
func (n *Navigator) resolveImportedSymbolDefinition(modCfg *modules.ModConfig, fileImports []FileImport, lines []string, word, uri, fileDir string) *Location {
	for _, imp := range fileImports {
		isMember := false
		if imp.Member == word {
			isMember = true
		} else {
			for _, m := range imp.Members {
				if m == word {
					isMember = true
					break
				}
			}
		}

		if isMember {
			modFile := n.ResolveModuleFilePath(modCfg, imp.Source, imp.Module, fileDir)
			if modFile != "" {
				lIdx, charPos, found := FindDeclarationInFile(modFile, word)
				if found {
					return &Location{
						URI: pathToURI(modFile),
						Range: Range{
							Start: Position{Line: lIdx, Character: charPos},
							End:   Position{Line: lIdx, Character: charPos + len(word)},
						},
					}
				}
				return &Location{
					URI: pathToURI(modFile),
					Range: Range{
						Start: Position{Line: 0, Character: 0},
						End:   Position{Line: 0, Character: 0},
					},
				}
			}
			// Point to import declaration line if module file is virtual or external
			reWordInLine := regexp.MustCompile(fmt.Sprintf(`\b(%s)\b`, regexp.QuoteMeta(word)))
			if loc := reWordInLine.FindStringIndex(lines[imp.Line]); loc != nil {
				return &Location{
					URI: uri,
					Range: Range{
						Start: Position{Line: imp.Line, Character: loc[0]},
						End:   Position{Line: imp.Line, Character: loc[1]},
					},
				}
			}
		}

		if imp.Alias != "" && imp.Alias == word {
			return &Location{
				URI: uri,
				Range: Range{
					Start: Position{Line: imp.Line, Character: 0},
					End:   Position{Line: imp.Line, Character: len(lines[imp.Line])},
				},
			}
		}

		if imp.Module == word {
			modFile := n.ResolveModuleFilePath(modCfg, imp.Source, imp.Module, fileDir)
			if modFile != "" {
				lIdx, charPos, _ := FindDeclarationInFile(modFile, imp.Module)
				return &Location{
					URI: pathToURI(modFile),
					Range: Range{
						Start: Position{Line: lIdx, Character: charPos},
						End:   Position{Line: lIdx, Character: charPos + len(imp.Module)},
					},
				}
			}
		}
	}
	return nil
}
