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

	reFromGroup := regexp.MustCompile(`^\s*from\s+["']?([^"'\s]+)["']?\s+use\s+(?:([a-zA-Z0-9_]+)::)?(?:\{|\()([^})]+)(?:\}|\))`)
	reFromMember := regexp.MustCompile(`^\s*from\s+["']?([^"'\s]+)["']?\s+use\s+([a-zA-Z0-9_]+)::([a-zA-Z0-9_]+)`)
	reFromMods := regexp.MustCompile(`^\s*from\s+["']?([^"'\s]+)["']?\s+use\s+(?:\{|\()([^})]+)(?:\}|\))`)
	reFromAs := regexp.MustCompile(`^\s*from\s+["']?([^"'\s]+)["']?\s+use\s+([a-zA-Z0-9_]+)\s+as\s+([a-zA-Z0-9_]+)`)
	reFromMod := regexp.MustCompile(`^\s*from\s+["']?([^"'\s]+)["']?\s+use\s+["']?([a-zA-Z0-9_]+)["']?`)
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
			src := m[1]
			mod := m[2]
			if mod == "" {
				mod = src
				src = ""
			}
			imports = append(imports, FileImport{
				Source:  src,
				Module:  mod,
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
	reDecl := regexp.MustCompile(fmt.Sprintf(`(?m)^\s*(?:global\s+|export\s+)?(?:let|const|arg|func|unit|declare|interface|type)\s+(%s)\b`, regexp.QuoteMeta(word)))
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

// findStdFile is implemented in std_resolver.go with global XFPM and local project paths resolution.

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
	if stdPath := findStdFile(curFileDir, modName); stdPath != "" {
		return stdPath
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

	// 1.5 Member navigation: obj.member (e.g. validExample.c, opts.silent, params.a, $f{params.a})
	if dotIdx := strings.LastIndex(fullExpr, "."); dotIdx != -1 {
		rawObj := strings.TrimSpace(fullExpr[:dotIdx])
		memberName := strings.TrimSpace(fullExpr[dotIdx+1:])
		rawObj = strings.TrimPrefix(rawObj, "$f{")
		rawObj = strings.TrimPrefix(rawObj, "${")
		rawObj = strings.TrimPrefix(rawObj, "{")
		objName := rawObj

		if objName != "" && memberName != "" {
			if word != objName {
				if varInfo := findVariableTargetInfo(lines, objName); varInfo != nil {
					targetIface := varInfo.interfaceName
					if targetIface == "" {
						targetIface = varInfo.unitType
					}
					if targetIface != "" {
						if loc := findMemberInInterface(lines, targetIface, memberName, uri); loc != nil {
							return loc
						}
					}
					// Builtin primitive methods: string, list, number, map
					if varInfo.isString || strings.EqualFold(targetIface, "string") || objName == "string" {
						if stdBuiltin := findStdFile(fileDir, "builtin"); stdBuiltin != "" {
							if lIdx, charPos, found := findDeclarationInFile(stdBuiltin, memberName); found {
								return &Location{
									URI: pathToURI(stdBuiltin),
									Range: Range{
										Start: Position{Line: lIdx, Character: charPos},
										End:   Position{Line: lIdx, Character: charPos + len(memberName)},
									},
								}
							}
						}
					}
					if varInfo.isList || strings.EqualFold(targetIface, "list") || objName == "list" {
						if stdBuiltin := findStdFile(fileDir, "builtin"); stdBuiltin != "" {
							if lIdx, charPos, found := findDeclarationInFile(stdBuiltin, memberName); found {
								return &Location{
									URI: pathToURI(stdBuiltin),
									Range: Range{
										Start: Position{Line: lIdx, Character: charPos},
										End:   Position{Line: lIdx, Character: charPos + len(memberName)},
									},
								}
							}
						}
					}
					if varInfo.isNumber || strings.EqualFold(targetIface, "number") || objName == "number" {
						if stdBuiltin := findStdFile(fileDir, "builtin"); stdBuiltin != "" {
							if lIdx, charPos, found := findDeclarationInFile(stdBuiltin, memberName); found {
								return &Location{
									URI: pathToURI(stdBuiltin),
									Range: Range{
										Start: Position{Line: lIdx, Character: charPos},
										End:   Position{Line: lIdx, Character: charPos + len(memberName)},
									},
								}
							}
						}
					}
					if varInfo.isMap || strings.EqualFold(targetIface, "map") || objName == "map" {
						if stdBuiltin := findStdFile(fileDir, "builtin"); stdBuiltin != "" {
							if lIdx, charPos, found := findDeclarationInFile(stdBuiltin, memberName); found {
								return &Location{
									URI: pathToURI(stdBuiltin),
									Range: Range{
										Start: Position{Line: lIdx, Character: charPos},
										End:   Position{Line: lIdx, Character: charPos + len(memberName)},
									},
								}
							}
						}
					}
					// Inline map in function or variable
					reFieldDecl := regexp.MustCompile(fmt.Sprintf(`\b(%s)\s*\??\s*(?::|\s+)`, regexp.QuoteMeta(memberName)))
					for idx, l := range lines {
						if strings.Contains(l, "@map") && strings.Contains(l, memberName) {
							if matchLoc := reFieldDecl.FindStringSubmatchIndex(l); matchLoc != nil && len(matchLoc) >= 4 {
								return &Location{
									URI: uri,
									Range: Range{
										Start: Position{Line: idx, Character: matchLoc[2]},
										End:   Position{Line: idx, Character: matchLoc[3]},
									},
								}
							}
						}
					}
				}
			}
		}
	}

	// 1.6 Clicked on a property key inside a typed map literal: e.g. `mode: "log"` in Sys.require("...", @map{ mode: "log" })
	if isKey, keyWord, mapStartLine := isCursorOnMapKey(lines, pos); isKey {
		expectedType := resolveExpectedMapType(lines, mapStartLine, curFilePath)
		if expectedType != "" {
			if loc := findMemberInInterface(lines, expectedType, keyWord, uri); loc != nil {
				return loc
			}
		}
		// In a map key context: NEVER fall through to external variables or modules!
		return nil
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

	// 4. Search Local Definitions in active file (variables, lambdas, funcs, interfaces, types)
	reLocalDecl := regexp.MustCompile(fmt.Sprintf(`(?m)^\s*(?:global\s+|export\s+)?(?:let|const|arg|func|unit|declare|interface|type)\s+(%s)\b`, regexp.QuoteMeta(word)))
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

	// 6. Builtin primitive contracts & types in std/builtin.xru
	lowerWord := strings.ToLower(word)
	if lowerWord == "string" || lowerWord == "list" || lowerWord == "number" || lowerWord == "map" {
		targetIface := strings.ToUpper(lowerWord[:1]) + lowerWord[1:]
		if stdBuiltin := findStdFile(fileDir, "builtin"); stdBuiltin != "" {
			if lIdx, charPos, found := findDeclarationInFile(stdBuiltin, targetIface); found {
				return &Location{
					URI: pathToURI(stdBuiltin),
					Range: Range{
						Start: Position{Line: lIdx, Character: charPos},
						End:   Position{Line: lIdx, Character: charPos + len(targetIface)},
					},
				}
			}
		}
	}
	if word == "DateInfo" {
		if stdDate := findStdFile(fileDir, "date"); stdDate != "" {
			if lIdx, charPos, found := findDeclarationInFile(stdDate, "DateInfo"); found {
				return &Location{
					URI: pathToURI(stdDate),
					Range: Range{
						Start: Position{Line: lIdx, Character: charPos},
						End:   Position{Line: lIdx, Character: charPos + len(word)},
					},
				}
			}
		}
	}

	// 7. Standard library modules: std, path, fs, crypto, date, math, sys, cmd, json, net
	for _, m := range []string{"std", "path", "fs", "crypto", "date", "math", "sys", "cmd", "json", "net"} {
		if word == m {
			if stdFile := findStdFile(fileDir, word); stdFile != "" {
				return &Location{
					URI: pathToURI(stdFile),
					Range: Range{
						Start: Position{Line: 0, Character: 0},
						End:   Position{Line: 0, Character: len(word)},
					},
				}
			}
		}
	}

	return nil
}

type PropertyInfo struct {
	TypeName   string
	PropName   string
	PropType   string
	IsOptional bool
	IsMethod   bool
	Doc        string
	Line       int
	CharStart  int
	CharEnd    int
	URI        string
}

func findPropertyDeclaration(lines []string, targetType, propName string, uri string) *PropertyInfo {
	if targetType == "" || propName == "" {
		return nil
	}
	cleanType := strings.TrimSpace(strings.TrimSuffix(targetType, "?"))

	// 1. If targetType is an inline map type: @map{ a: string, ... }
	if strings.HasPrefix(cleanType, "@map") && strings.Contains(cleanType, "{") {
		reField := regexp.MustCompile(fmt.Sprintf(`\b(%s)(\??)(?:\s*:\s*|\s+)([^,\n}]+)`, regexp.QuoteMeta(propName)))
		if m := reField.FindStringSubmatch(cleanType); len(m) > 3 {
			pType := strings.TrimSpace(m[3])
			return &PropertyInfo{
				TypeName:   "@map",
				PropName:   propName,
				PropType:   pType,
				IsOptional: m[2] == "?",
				IsMethod:   strings.HasPrefix(pType, "func(") || strings.HasPrefix(pType, "func "),
				URI:        uri,
			}
		}
	}

	// 2. Scan active file
	if info := scanLinesForPropertyDeclaration(lines, cleanType, propName, uri); info != nil {
		return info
	}

	// 3. Scan workspace files in current directory
	curFilePath := uriToPath(uri)
	if curFilePath != "" {
		fileDir := filepath.Dir(curFilePath)
		entries, err := os.ReadDir(fileDir)
		if err == nil {
			for _, entry := range entries {
				if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".xru") {
					fp := filepath.Join(fileDir, entry.Name())
					if fp == curFilePath {
						continue
					}
					if content, rErr := os.ReadFile(fp); rErr == nil {
						fLines := strings.Split(string(content), "\n")
						if info := scanLinesForPropertyDeclaration(fLines, cleanType, propName, pathToURI(fp)); info != nil {
							return info
						}
					}
				}
			}
		}

		// 4. Scan stdlib files
		if stdDir := FindStdDir(fileDir); stdDir != "" {
			stdEntries, sErr := os.ReadDir(stdDir)
			if sErr == nil {
				for _, entry := range stdEntries {
					if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".xru") {
						fp := filepath.Join(stdDir, entry.Name())
						if content, rErr := os.ReadFile(fp); rErr == nil {
							fLines := strings.Split(string(content), "\n")
							if info := scanLinesForPropertyDeclaration(fLines, cleanType, propName, pathToURI(fp)); info != nil {
								return info
							}
						}
					}
				}
			}
		}
	}

	return nil
}

func scanLinesForPropertyDeclaration(lines []string, cleanType, propName, uri string) *PropertyInfo {
	reIfaceStart := regexp.MustCompile(fmt.Sprintf(`(?m)^\s*(?:interface|type)\s+%s\b`, regexp.QuoteMeta(cleanType)))
	reField := regexp.MustCompile(fmt.Sprintf(`(?m)^\s*(?:func\s+)?(%s)(\??)(?:\s*:\s*|\s+)([^/\n\r;]+?)(?:;|\s*//(.*))?$`, regexp.QuoteMeta(propName)))

	inTargetIface := false
	braceDepth := 0

	for idx, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*") || strings.HasPrefix(trimmed, "*") {
			continue
		}

		if !inTargetIface {
			if reIfaceStart.MatchString(line) {
				inTargetIface = true
				braceDepth = strings.Count(line, "{") - strings.Count(line, "}")
				if matchLoc := reField.FindStringSubmatchIndex(line); matchLoc != nil && len(matchLoc) >= 4 {
					m := reField.FindStringSubmatch(line)
					propType := strings.TrimSpace(m[3])
					trailingDoc := ""
					if len(m) > 4 {
						trailingDoc = strings.TrimSpace(m[4])
					}
					aboveDoc := extractDocCommentForLine(lines, idx)
					doc := aboveDoc
					if trailingDoc != "" {
						if doc != "" {
							doc += "\n\n" + trailingDoc
						} else {
							doc = trailingDoc
						}
					}
					return &PropertyInfo{
						TypeName:   cleanType,
						PropName:   propName,
						PropType:   propType,
						IsOptional: m[2] == "?",
						IsMethod:   strings.HasPrefix(propType, "func(") || strings.HasPrefix(propType, "func "),
						Doc:        doc,
						Line:       idx,
						CharStart:  matchLoc[2],
						CharEnd:    matchLoc[3],
						URI:        uri,
					}
				}
				if braceDepth == 0 && strings.Contains(line, "{") {
					inTargetIface = false
				}
			}
		} else {
			braceDepth += strings.Count(line, "{") - strings.Count(line, "}")
			if matchLoc := reField.FindStringSubmatchIndex(line); matchLoc != nil && len(matchLoc) >= 4 {
				m := reField.FindStringSubmatch(line)
				propType := strings.TrimSpace(m[3])
				trailingDoc := ""
				if len(m) > 4 {
					trailingDoc = strings.TrimSpace(m[4])
				}
				aboveDoc := extractDocCommentForLine(lines, idx)
				doc := aboveDoc
				if trailingDoc != "" {
					if doc != "" {
						doc += "\n\n" + trailingDoc
					} else {
						doc = trailingDoc
					}
				}
				return &PropertyInfo{
					TypeName:   cleanType,
					PropName:   propName,
					PropType:   propType,
					IsOptional: m[2] == "?",
					IsMethod:   strings.HasPrefix(propType, "func(") || strings.HasPrefix(propType, "func "),
					Doc:        doc,
					Line:       idx,
					CharStart:  matchLoc[2],
					CharEnd:    matchLoc[3],
					URI:        uri,
				}
			}
			if braceDepth <= 0 {
				inTargetIface = false
			}
		}
	}
	return nil
}

// isCursorOnMapKey checks whether pos is on a key identifier inside a @map { ... } or object literal block.
func isCursorOnMapKey(lines []string, pos Position) (bool, string, int) {
	if pos.Line < 0 || pos.Line >= len(lines) {
		return false, "", -1
	}
	curLine := lines[pos.Line]
	trimmed := strings.TrimSpace(curLine)
	if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*") {
		return false, "", -1
	}

	// 1. Must have a key-value colon on this line (excluding ::)
	colonIdx := -1
	for i := 0; i < len(curLine); i++ {
		if curLine[i] == ':' {
			if (i > 0 && curLine[i-1] == ':') || (i+1 < len(curLine) && curLine[i+1] == ':') {
				continue
			}
			colonIdx = i
			break
		}
	}
	if colonIdx == -1 {
		return false, "", -1
	}

	// 2. Cursor MUST be before or on the colon (key position)
	if pos.Character > colonIdx {
		return false, "", -1
	}

	// 3. Extract the key before colon
	reKey := regexp.MustCompile(`^\s*([a-zA-Z_][a-zA-Z0-9_]*|"(?:[^"\\]|\\.)*")\s*:`)
	matchLoc := reKey.FindStringSubmatchIndex(curLine[:colonIdx+1])
	if matchLoc == nil || len(matchLoc) < 4 {
		return false, "", -1
	}
	keyStart := matchLoc[2]
	keyEnd := matchLoc[3]
	if pos.Character < keyStart || pos.Character > keyEnd {
		return false, "", -1
	}
	keyWord := strings.Trim(curLine[keyStart:keyEnd], "\"")

	// 4. Verify we are inside a map literal block @map { ... } or { ... }
	depth := 0
	mapStartLine := -1
	for l := pos.Line; l >= 0; l-- {
		line := lines[l]
		for _, r := range line {
			if r == '{' {
				depth++
			} else if r == '}' {
				depth--
			}
		}
		if depth > 0 && (strings.Contains(line, "@map") || strings.Contains(line, "=") || strings.Contains(line, "(")) {
			trimmedL := strings.TrimSpace(line)
			if strings.HasPrefix(trimmedL, "interface ") || strings.HasPrefix(trimmedL, "type ") || strings.HasPrefix(trimmedL, "unit ") {
				return false, "", -1
			}
			mapStartLine = l
			break
		}
	}

	if mapStartLine == -1 {
		return false, "", -1
	}

	return true, keyWord, mapStartLine
}

func findMemberInInterface(lines []string, ifaceName, memberName string, uri string) *Location {
	if info := findPropertyDeclaration(lines, ifaceName, memberName, uri); info != nil {
		return &Location{
			URI: info.URI,
			Range: Range{
				Start: Position{Line: info.Line, Character: info.CharStart},
				End:   Position{Line: info.Line, Character: info.CharEnd},
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
