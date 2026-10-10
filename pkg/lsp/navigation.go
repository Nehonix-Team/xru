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
	} else if wordStart > 1 && l[wordStart-1] == '.' {
		prefixEnd := wordStart - 1
		prefixStart := prefixEnd
		for prefixStart > 0 && isWordChar(l[prefixStart-1]) {
			prefixStart--
		}
		if prefixEnd > prefixStart {
			prefix := l[prefixStart:prefixEnd]
			return prefix + "." + targetWord, targetWord
		}
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

func (s *Server) handleDefinition(uri string, pos Position) *Location {
	text, ok := s.docs[uri]
	if !ok {
		return nil
	}

	fullExpr, word := getFullExpressionAt(text, pos.Line, pos.Character)
	if word == "" {
		return nil
	}

	fileDir := filepath.Dir(uriToPath(uri))
	modFilePath := modules.FindModConfigFile(fileDir)
	if modFilePath != "" {
		if modCfg, errs := modules.LoadModConfig(modFilePath); len(errs) == 0 && modCfg != nil {
			// 1. Check if word is a mapped namespace in _mod.xru (e.g. "services")
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

			// 2. Check if word is a module in any mapped namespace (e.g. "auth", "cdn", "db")
			targetWord := word
			targetModule := ""
			if strings.Contains(fullExpr, "::") {
				parts := strings.Split(fullExpr, "::")
				if len(parts) == 2 {
					targetModule = parts[0]
					targetWord = parts[1]
				}
			}

			// 2.A If targetWord is a module name, jump to the module file itself!
			for ns := range modCfg.Mappings {
				mods, err := modCfg.GetNamespaceModules(ns)
				if err != nil {
					continue
				}
				if m, ok := mods[targetWord]; ok {
					line := 0
					char := 0
					if content, err := os.ReadFile(m.FilePath); err == nil {
						cLines := strings.Split(string(content), "\n")
						for lIdx, cLine := range cLines {
							if strings.HasPrefix(strings.TrimSpace(cLine), "mod ") && strings.Contains(cLine, targetWord) {
								line = lIdx
								break
							}
						}
					}
					return &Location{
						URI: pathToURI(m.FilePath),
						Range: Range{
							Start: Position{Line: line, Character: char},
							End:   Position{Line: line, Character: char + len(targetWord)},
						},
					}
				}
			}

			// 2.B If targetWord is an exported member of a module (e.g. auth::login)
			for _, cand := range modCfg.GetAllExportedCandidates() {
				if cand.Kind == "module" {
					continue
				}
				if (targetModule == "" || cand.ModuleName == targetModule) && cand.Name == targetWord {
					if content, err := os.ReadFile(cand.FilePath); err == nil {
						cLines := strings.Split(string(content), "\n")
						for lIdx, cLine := range cLines {
							cTrim := strings.TrimSpace(cLine)
							if strings.HasPrefix(cTrim, "//") {
								continue
							}
							if strings.Contains(cLine, targetWord) {
								cRe := regexp.MustCompile(fmt.Sprintf(`\b(%s)\b`, regexp.QuoteMeta(targetWord)))
								if loc := cRe.FindStringIndex(cLine); loc != nil {
									return &Location{
										URI: pathToURI(cand.FilePath),
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
						URI: pathToURI(cand.FilePath),
						Range: Range{
							Start: Position{Line: 0, Character: 0},
							End:   Position{Line: 0, Character: 0},
						},
					}
				}
			}
		}
	}

	// 3. Search definition in current file (let, const, arg, func, unit, lambda)
	lines := strings.Split(text, "\n")
	for idx, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "//") {
			continue
		}

		// 3.A. let / const / arg / func / unit definition
		re := regexp.MustCompile(fmt.Sprintf(`\b(?:let|const|arg|func|unit|declare)\s+(%s)\b`, regexp.QuoteMeta(word)))
		if loc := re.FindStringSubmatchIndex(line); loc != nil && len(loc) >= 4 {
			return &Location{
				URI: uri,
				Range: Range{
					Start: Position{Line: idx, Character: loc[2]},
					End:   Position{Line: idx, Character: loc[3]},
				},
			}
		}

		// 3.B. from "<ns>" use mod as <alias>
		reFromAs := regexp.MustCompile(fmt.Sprintf(`\buse\s+[a-zA-Z0-9_.]+\s+as\s+(%s)\b`, regexp.QuoteMeta(word)))
		if loc := reFromAs.FindStringSubmatchIndex(line); loc != nil && len(loc) >= 4 {
			return &Location{
				URI: uri,
				Range: Range{
					Start: Position{Line: idx, Character: loc[2]},
					End:   Position{Line: idx, Character: loc[3]},
				},
			}
		}

		// 3.C. lambda parameter e.g. (data, err) =>
		if strings.Contains(line, "=>") {
			parts := strings.Split(line, "=>")
			paramPart := parts[0]
			paramRe := regexp.MustCompile(fmt.Sprintf(`\b(%s)\b`, regexp.QuoteMeta(word)))
			if loc := paramRe.FindStringIndex(paramPart); loc != nil {
				return &Location{
					URI: uri,
					Range: Range{
						Start: Position{Line: idx, Character: loc[0]},
						End:   Position{Line: idx, Character: loc[1]},
					},
				}
			}
		}
	}

	// 4. Fallback for 'std'
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
