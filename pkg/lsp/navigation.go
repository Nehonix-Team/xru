package lsp

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Nehonix-Team/xru/pkg/modules"
)

func getWordAt(text string, line, char int) string {
	lines := strings.Split(text, "\n")
	if line < 0 || line >= len(lines) {
		return ""
	}
	l := lines[line]
	if char < 0 || char > len(l) {
		return ""
	}

	start := char
	for start > 0 && isWordChar(l[start-1]) {
		start--
	}
	end := char
	for end < len(l) && isWordChar(l[end]) {
		end++
	}

	if start == end {
		return ""
	}
	return l[start:end]
}

func isWordChar(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') || b == '_'
}

func getFullExpressionAt(text string, line, char int) (string, string) {
	lines := strings.Split(text, "\n")
	if line < 0 || line >= len(lines) {
		return "", ""
	}
	l := lines[line]
	targetWord := getWordAt(text, line, char)
	if targetWord == "" {
		return "", ""
	}

	// Look backwards from word start to see if there is `prefix.` or `prefix::`
	wordStart := char
	for wordStart > 0 && isWordChar(l[wordStart-1]) {
		wordStart--
	}
	if wordStart >= 2 && l[wordStart-2:wordStart] == "::" {
		prefixEnd := wordStart - 2
		prefixStart := prefixEnd
		for prefixStart > 0 && isWordChar(l[prefixStart-1]) {
			prefixStart--
		}
		if prefixEnd > prefixStart {
			prefix := l[prefixStart:prefixEnd]
			return prefix + "::" + targetWord, targetWord
		}
	} else if wordStart > 0 && l[wordStart-1] == '.' {
		prefixEnd := wordStart - 1
		prefixStart := prefixEnd
		for prefixStart > 0 && isWordChar(l[prefixStart-1]) {
			prefixStart--
		}
		if prefixEnd > prefixStart {
			prefix := l[prefixStart:prefixEnd]
			return prefix + "." + targetWord, targetWord
		} else if prefixEnd > 0 && (l[prefixEnd-1] == '"' || l[prefixEnd-1] == '\'') {
			return "string." + targetWord, targetWord
		} else if prefixEnd > 0 && l[prefixEnd-1] == ']' {
			return "list." + targetWord, targetWord
		}
		return "." + targetWord, targetWord
	}

	// Look forwards from word end to see if there is `.suffix` or `::suffix`
	wordEnd := char
	for wordEnd < len(l) && isWordChar(l[wordEnd]) {
		wordEnd++
	}
	if wordEnd+2 <= len(l) && l[wordEnd:wordEnd+2] == "::" {
		suffixStart := wordEnd + 2
		suffixEnd := suffixStart
		for suffixEnd < len(l) && isWordChar(l[suffixEnd]) {
			suffixEnd++
		}
		if suffixEnd > suffixStart {
			suffix := l[suffixStart:suffixEnd]
			return targetWord + "::" + suffix, targetWord
		}
	} else if wordEnd < len(l) && l[wordEnd] == '.' {
		suffixStart := wordEnd + 1
		suffixEnd := suffixStart
		for suffixEnd < len(l) && isWordChar(l[suffixEnd]) {
			suffixEnd++
		}
		if suffixEnd > suffixStart {
			suffix := l[suffixStart:suffixEnd]
			return targetWord + "." + suffix, targetWord
		}
	}

	return targetWord, targetWord
}

type FileImport struct {
	Source  string   // Namespace e.g. "dll"
	Module  string   // Module e.g. "gArgs"
	Member  string   // Member e.g. "author"
	Members []string // Grouped members e.g. ["author", "version", ...]
	Alias   string   // Alias e.g. "Fs"
	Line    int      // 0-indexed line where import statement is written
}

func parseFileImports(text string) []FileImport {
	var imports []FileImport
	lines := strings.Split(text, "\n")

	reFromGroup := regexp.MustCompile(`^\s*from\s+["']([^"']+)["']\s+use\s+([a-zA-Z0-9_]+)::\(([^)]+)\)`)
	reFromMember := regexp.MustCompile(`^\s*from\s+["']([^"']+)["']\s+use\s+([a-zA-Z0-9_]+)::([a-zA-Z0-9_]+)`)
	reFromMods := regexp.MustCompile(`^\s*from\s+["']([^"']+)["']\s+use\s+\(([^)]+)\)`)
	reFromAs := regexp.MustCompile(`^\s*from\s+["']([^"']+)["']\s+use\s+([a-zA-Z0-9_]+)\s+as\s+([a-zA-Z0-9_]+)`)
	reFromMod := regexp.MustCompile(`^\s*from\s+["']([^"']+)["']\s+use\s+([a-zA-Z0-9_]+)`)
	reUseAs := regexp.MustCompile(`^\s*use\s+([a-zA-Z0-9_]+)\s+as\s+([a-zA-Z0-9_]+)`)
	reUse := regexp.MustCompile(`^\s*use\s+([a-zA-Z0-9_]+)`)

	for idx, line := range lines {
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(trim, "//") {
			continue
		}
		if m := reFromGroup.FindStringSubmatch(trim); len(m) == 4 {
			rawMembers := strings.Split(m[3], ",")
			var members []string
			for _, rm := range rawMembers {
				mem := strings.TrimSpace(rm)
				if mem != "" {
					members = append(members, mem)
				}
			}
			imports = append(imports, FileImport{
				Source:  m[1],
				Module:  m[2],
				Members: members,
				Line:    idx,
			})
		} else if m := reFromMember.FindStringSubmatch(trim); len(m) == 4 {
			imports = append(imports, FileImport{
				Source: m[1],
				Module: m[2],
				Member: m[3],
				Line:   idx,
			})
		} else if m := reFromMods.FindStringSubmatch(trim); len(m) == 3 {
			rawMods := strings.Split(m[2], ",")
			for _, rm := range rawMods {
				mod := strings.TrimSpace(rm)
				if mod != "" {
					imports = append(imports, FileImport{
						Source: m[1],
						Module: mod,
						Line:   idx,
					})
				}
			}
		} else if m := reFromAs.FindStringSubmatch(trim); len(m) == 4 {
			imports = append(imports, FileImport{
				Source: m[1],
				Module: m[2],
				Alias:  m[3],
				Line:   idx,
			})
		} else if m := reFromMod.FindStringSubmatch(trim); len(m) == 3 {
			imports = append(imports, FileImport{
				Source: m[1],
				Module: m[2],
				Line:   idx,
			})
		} else if m := reUseAs.FindStringSubmatch(trim); len(m) == 3 {
			imports = append(imports, FileImport{
				Module: m[1],
				Alias:  m[2],
				Line:   idx,
			})
		} else if m := reUse.FindStringSubmatch(trim); len(m) == 2 {
			imports = append(imports, FileImport{
				Module: m[1],
				Line:   idx,
			})
		}
	}
	return imports
}

func findDeclarationInFile(filePath, word string) (line, char int, found bool) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return 0, 0, false
	}
	lines := strings.Split(string(content), "\n")
	reDecl := regexp.MustCompile(fmt.Sprintf(`(?m)^\s*(?:global\s+|export\s+)?(?:let|const|arg|func|unit|declare)\s+(%s)\b`, regexp.QuoteMeta(word)))
	for lIdx, l := range lines {
		lTrim := strings.TrimSpace(l)
		if strings.HasPrefix(lTrim, "//") {
			continue
		}
		if loc := reDecl.FindStringSubmatchIndex(l); loc != nil && len(loc) >= 4 {
			return lIdx, loc[2], true
		}
	}
	reMod := regexp.MustCompile(fmt.Sprintf(`(?m)^\s*mod\s+(%s)\b`, regexp.QuoteMeta(word)))
	for lIdx, l := range lines {
		lTrim := strings.TrimSpace(l)
		if strings.HasPrefix(lTrim, "//") {
			continue
		}
		if loc := reMod.FindStringSubmatchIndex(l); loc != nil && len(loc) >= 4 {
			return lIdx, loc[2], true
		}
	}
	reWord := regexp.MustCompile(fmt.Sprintf(`\b(%s)\b`, regexp.QuoteMeta(word)))
	for lIdx, l := range lines {
		lTrim := strings.TrimSpace(l)
		if strings.HasPrefix(lTrim, "//") {
			continue
		}
		if loc := reWord.FindStringIndex(l); loc != nil {
			return lIdx, loc[0], true
		}
	}
	return 0, 0, false
}

func (s *Server) resolveModuleFilePath(modCfg *modules.ModConfig, sourceNs, modName, curFileDir string) string {
	if modCfg != nil {
		if sourceNs != "" {
			if modDetails, err := modCfg.ResolveModule(sourceNs, modName); err == nil && modDetails != nil {
				return modDetails.FilePath
			}
		} else {
			for ns := range modCfg.Mappings {
				if modDetails, err := modCfg.ResolveModule(ns, modName); err == nil && modDetails != nil {
					return modDetails.FilePath
				}
			}
		}
	}
	localPath := filepath.Join(curFileDir, modName+".xru")
	if info, err := os.Stat(localPath); err == nil && !info.IsDir() {
		return localPath
	}
	return ""
}

func (s *Server) handleDefinition(uri string, pos Position) *Location {
	text, ok := s.docs[uri]
	if !ok {
		return nil
	}

	fullExpr, word := getFullExpressionAt(text, pos.Line, pos.Character)
	if word == "" {
		return nil
	}

	curFilePath := uriToPath(uri)
	fileDir := filepath.Dir(curFilePath)
	modFilePath := modules.FindModConfigFile(fileDir)
	var modCfg *modules.ModConfig
	if modFilePath != "" {
		if cfg, errs := modules.LoadModConfig(modFilePath); len(errs) == 0 && cfg != nil {
			modCfg = cfg
		}
	}

	lines := strings.Split(text, "\n")
	var curLine string
	if pos.Line >= 0 && pos.Line < len(lines) {
		curLine = lines[pos.Line]
	}
	trimmedCurLine := strings.TrimSpace(curLine)
	isCurLineImport := strings.HasPrefix(trimmedCurLine, "from ") || strings.HasPrefix(trimmedCurLine, "use ")

	fileImports := parseFileImports(text)

	// 1. Explicit qualified expression: module::member (e.g. gArgs::author)
	if strings.Contains(fullExpr, "::") {
		parts := strings.Split(fullExpr, "::")
		if len(parts) == 2 {
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

			modFile := s.resolveModuleFilePath(modCfg, targetNs, targetMod, fileDir)
			if modFile != "" {
				lIdx, charPos, found := findDeclarationInFile(modFile, targetWord)
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
			return nil
		}
	}

	// 2. Clicked on a namespace name in _mod.xru
	if modCfg != nil && modFilePath != "" {
		if _, ok := modCfg.Mappings[word]; ok {
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
	}

	// 3. Cursor is directly ON the import line
	if isCurLineImport {
		for _, imp := range fileImports {
			if imp.Line == pos.Line {
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
					modFile := s.resolveModuleFilePath(modCfg, imp.Source, imp.Module, fileDir)
					if modFile != "" {
						lIdx, charPos, found := findDeclarationInFile(modFile, word)
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
					modFile := s.resolveModuleFilePath(modCfg, imp.Source, imp.Module, fileDir)
					if modFile != "" {
						lIdx, charPos, _ := findDeclarationInFile(modFile, imp.Module)
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
		}
	}

	// 4. Search Local Definitions in active file (variables, lambdas, funcs)
	reLocalDecl := regexp.MustCompile(fmt.Sprintf(`(?m)^\s*(?:global\s+|export\s+)?(?:let|const|arg|func|unit|declare)\s+(%s)\b`, regexp.QuoteMeta(word)))
	for idx, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "//") {
			continue
		}
		if loc := reLocalDecl.FindStringSubmatchIndex(line); loc != nil && len(loc) >= 4 {
			return &Location{
				URI: uri,
				Range: Range{
					Start: Position{Line: idx, Character: loc[2]},
					End:   Position{Line: idx, Character: loc[3]},
				},
			}
		}
	}

	reLambda := regexp.MustCompile(`\(([^)]*)\)\s*(?:->|=>)`)
	for idx, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "//") {
			continue
		}
		if locs := reLambda.FindStringSubmatchIndex(line); locs != nil && len(locs) >= 4 {
			paramSub := line[locs[2]:locs[3]]
			paramRe := regexp.MustCompile(fmt.Sprintf(`\b(%s)\b`, regexp.QuoteMeta(word)))
			if pLoc := paramRe.FindStringIndex(paramSub); pLoc != nil {
				startCol := locs[2] + pLoc[0]
				return &Location{
					URI: uri,
					Range: Range{
						Start: Position{Line: idx, Character: startCol},
						End:   Position{Line: idx, Character: startCol + len(word)},
					},
				}
			}
		}
	}

	reFuncParams := regexp.MustCompile(`func\s+[a-zA-Z0-9_]+\s*\(([^)]*)\)`)
	for idx, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "//") {
			continue
		}
		if locs := reFuncParams.FindStringSubmatchIndex(line); locs != nil && len(locs) >= 4 {
			paramSub := line[locs[2]:locs[3]]
			paramRe := regexp.MustCompile(fmt.Sprintf(`\b(%s)\b`, regexp.QuoteMeta(word)))
			if pLoc := paramRe.FindStringIndex(paramSub); pLoc != nil {
				startCol := locs[2] + pLoc[0]
				return &Location{
					URI: uri,
					Range: Range{
						Start: Position{Line: idx, Character: startCol},
						End:   Position{Line: idx, Character: startCol + len(word)},
					},
				}
			}
		}
	}

	reForLoop := regexp.MustCompile(`^\s*for\s+([a-zA-Z0-9_,\s]+)\s+in\b`)
	for idx, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "//") {
			continue
		}
		if locs := reForLoop.FindStringSubmatchIndex(line); locs != nil && len(locs) >= 4 {
			varSub := line[locs[2]:locs[3]]
			varRe := regexp.MustCompile(fmt.Sprintf(`\b(%s)\b`, regexp.QuoteMeta(word)))
			if vLoc := varRe.FindStringIndex(varSub); vLoc != nil {
				startCol := locs[2] + vLoc[0]
				return &Location{
					URI: uri,
					Range: Range{
						Start: Position{Line: idx, Character: startCol},
						End:   Position{Line: idx, Character: startCol + len(word)},
					},
				}
			}
		}
	}

	reMod := regexp.MustCompile(fmt.Sprintf(`^\s*mod\s+(%s)\b`, regexp.QuoteMeta(word)))
	for idx, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "//") {
			continue
		}
		if loc := reMod.FindStringSubmatchIndex(line); loc != nil && len(loc) >= 4 {
			return &Location{
				URI: uri,
				Range: Range{
					Start: Position{Line: idx, Character: loc[2]},
					End:   Position{Line: idx, Character: loc[3]},
				},
			}
		}
	}

	// 5. Check if word is imported into the active file
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
			modFile := s.resolveModuleFilePath(modCfg, imp.Source, imp.Module, fileDir)
			if modFile != "" {
				lIdx, charPos, found := findDeclarationInFile(modFile, word)
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
			if imp.Module == "std" || imp.Module == "fs" || imp.Module == "net" || imp.Module == "crypto" || imp.Module == "cmd" {
				return &Location{
					URI: uri,
					Range: Range{
						Start: Position{Line: imp.Line, Character: 0},
						End:   Position{Line: imp.Line, Character: len(lines[imp.Line])},
					},
				}
			}
			modFile := s.resolveModuleFilePath(modCfg, imp.Source, imp.Module, fileDir)
			if modFile != "" {
				lIdx, charPos, _ := findDeclarationInFile(modFile, imp.Module)
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

	// 6. Builtin 'std' fallback
	if word == "std" {
		return &Location{
			URI: uri,
			Range: Range{
				Start: Position{Line: 0, Character: 0},
				End:   Position{Line: 0, Character: 0},
			},
		}
	}

	return nil
}


func (s *Server) handleReferences(uri string, pos Position) []Location {
	text, ok := s.docs[uri]
	if !ok {
		return []Location{}
	}

	word := getWordAt(text, pos.Line, pos.Character)
	if word == "" {
		return []Location{}
	}

	var locs []Location
	re := regexp.MustCompile(fmt.Sprintf(`\b(%s)\b`, regexp.QuoteMeta(word)))

	s.mu.Lock()
	docsCopy := make(map[string]string)
	for u, t := range s.docs {
		docsCopy[u] = t
	}
	s.mu.Unlock()

	for docURI, docText := range docsCopy {
		lines := strings.Split(docText, "\n")
		for idx, line := range lines {
			matches := re.FindAllStringIndex(line, -1)
			for _, m := range matches {
				locs = append(locs, Location{
					URI: docURI,
					Range: Range{
						Start: Position{Line: idx, Character: m[0]},
						End:   Position{Line: idx, Character: m[1]},
					},
				})
			}
		}
	}
	return locs
}

func (s *Server) handleDocumentHighlight(uri string, pos Position) []DocumentHighlight {
	text, ok := s.docs[uri]
	if !ok {
		return []DocumentHighlight{}
	}

	word := getWordAt(text, pos.Line, pos.Character)
	if word == "" {
		return []DocumentHighlight{}
	}

	var highlights []DocumentHighlight
	lines := strings.Split(text, "\n")
	re := regexp.MustCompile(fmt.Sprintf(`\b(%s)\b`, regexp.QuoteMeta(word)))

	for idx, line := range lines {
		matches := re.FindAllStringIndex(line, -1)
		for _, m := range matches {
			highlights = append(highlights, DocumentHighlight{
				Range: Range{
					Start: Position{Line: idx, Character: m[0]},
					End:   Position{Line: idx, Character: m[1]},
				},
				Kind: HighlightKindRead,
			})
		}
	}
	return highlights
}
