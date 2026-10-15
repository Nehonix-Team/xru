package lsp

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Nehonix-Team/xru/pkg/checker"
	"github.com/Nehonix-Team/xru/pkg/lexer"
	"github.com/Nehonix-Team/xru/pkg/modules"
	"github.com/Nehonix-Team/xru/pkg/parser"
)

func (s *Server) handleHover(uri string, pos Position) *Hover {
	text, ok := s.docs[uri]
	if !ok {
		return nil
	}

	fullExpr, word := getFullExpressionAt(text, pos.Line, pos.Character)
	if word == "" {
		return nil
	}

	filename := uriToPath(uri)
	l := lexer.New(filename, text)
	p := parser.New(l)
	prog := p.ParseProgram()
	var c *checker.Checker
	if prog != nil {
		c = checker.New()
		c.SetCurrentFile(filename)
		c.Check(prog)
	}

	lines := strings.Split(text, "\n")
	var curLine string
	if pos.Line >= 0 && pos.Line < len(lines) {
		curLine = lines[pos.Line]
	}

	// 0. Hover on property key inside a typed map literal: e.g. `a: 6` in `const data: TExample = { a: 6 }`
	if curLine != "" && !strings.Contains(fullExpr, ".") {
		reKey := regexp.MustCompile(fmt.Sprintf(`^\s*(%s)\s*(?::|->)`, regexp.QuoteMeta(word)))
		if matchLoc := reKey.FindStringSubmatchIndex(curLine); matchLoc != nil && len(matchLoc) >= 4 {
			if pos.Character >= matchLoc[2] && pos.Character <= matchLoc[3] {
				mapStartLine := -1
				depth := 0
				for l := pos.Line; l >= 0; l-- {
					line := lines[l]
					for _, r := range line {
						if r == '{' {
							depth++
						} else if r == '}' {
							depth--
						}
					}
					if depth > 0 && (strings.Contains(line, "@map") || strings.Contains(line, "=")) {
						mapStartLine = l
						break
					}
				}
				if mapStartLine != -1 {
					expectedType := resolveExpectedMapType(lines, mapStartLine, filename)
					if expectedType != "" {
						if propInfo := findPropertyDeclaration(lines, expectedType, word, uri); propInfo != nil {
							return formatPropertyHover(propInfo)
						}
					}
				}
			}
		}
	}

	// 1. Hover on property or method via dot access (e.g. data.a, validExample.c, opts.silent, $f{params.a})
	if strings.Contains(fullExpr, ".") {
		dotIdx := strings.LastIndex(fullExpr, ".")
		rawObj := strings.TrimSpace(fullExpr[:dotIdx])
		memberName := strings.TrimSpace(fullExpr[dotIdx+1:])
		rawObj = strings.TrimPrefix(rawObj, "$f{")
		rawObj = strings.TrimPrefix(rawObj, "${")
		rawObj = strings.TrimPrefix(rawObj, "{")

		if rawObj != "" && memberName != "" {
			if vInfo := findVariableTargetInfo(lines, rawObj); vInfo != nil {
				targetType := vInfo.interfaceName
				if targetType == "" {
					targetType = vInfo.unitType
				}
				if targetType != "" {
					if propInfo := findPropertyDeclaration(lines, targetType, memberName, uri); propInfo != nil {
						return formatPropertyHover(propInfo)
					}
				}
				if vInfo.isString {
					if doc, ok := StringMethodsDocs[memberName]; ok {
						return &Hover{Contents: MarkupContent{Kind: "markdown", Value: FormatMarkdown(&doc)}}
					}
				}
				if vInfo.isNumber {
					if doc, ok := NumberMethodsDocs[memberName]; ok {
						return &Hover{Contents: MarkupContent{Kind: "markdown", Value: FormatMarkdown(&doc)}}
					}
				}
				if vInfo.isList {
					if doc, ok := ListMethodsDocs[memberName]; ok {
						return &Hover{Contents: MarkupContent{Kind: "markdown", Value: FormatMarkdown(&doc)}}
					}
				}
				if vInfo.isMap {
					if doc, ok := MapMethodsDocs[memberName]; ok {
						return &Hover{Contents: MarkupContent{Kind: "markdown", Value: FormatMarkdown(&doc)}}
					}
				}
			}
		}
	} else if checker.IsBuiltinUnit(word) {
		if c != nil {
			if _, inScope := c.GetScope().Resolve(word); !inScope {
				return nil
			}
		}
	}

	lookupExpr := fullExpr
	if strings.Contains(fullExpr, ".") {
		parts := strings.Split(fullExpr, ".")
		objectName := parts[0]
		methodName := parts[1]
		if vInfo := findVariableTargetInfo(lines, objectName); vInfo != nil && vInfo.unitType != "" {
			lookupExpr = vInfo.unitType + "." + methodName
		} else if c != nil {
			for _, d := range c.AllDeclaredVars() {
				if d.Name == objectName && d.Type != "" {
					lookupExpr = d.Type + "." + methodName
					break
				}
			}
		}
	}

	if docItem := GetDocForWord(lookupExpr, word); docItem != nil {
		return &Hover{
			Contents: MarkupContent{
				Kind:  "markdown",
				Value: FormatMarkdown(docItem),
			},
		}
	}

	// 2. Check user-defined symbols (variables, constants, arguments, functions, modules)
	if prog != nil && c != nil {

		// 2.A: Si l'expression est sous la forme scope::member
		if strings.Contains(fullExpr, "::") {
			parts := strings.Split(fullExpr, "::")
			if len(parts) == 2 {
				scopeName, memberName := parts[0], parts[1]
				if unitSym, exists := c.GetScope().Resolve(scopeName); exists && unitSym.Kind == checker.SymUnit {
					if method, exists := unitSym.Methods[memberName]; exists {
						var md strings.Builder
						var params []string
						for _, param := range method.Params {
							pType := param.Type
							if pType == "" {
								pType = "any"
							}
							params = append(params, fmt.Sprintf("%s: %s", param.Name, pType))
						}
						retType := method.Type
						if retType == "" {
							retType = "void"
						}
						md.WriteString(fmt.Sprintf("```xru\n(method) %s::%s(%s): %s\n```\n", scopeName, memberName, strings.Join(params, ", "), retType))
						fileContent := s.getFileContentForPos(uri, method.Pos.Filename)
						if doc := extractDocComment(fileContent, method.Pos.Line); doc != "" {
							md.WriteString("\n" + doc + "\n")
						}
						return &Hover{
							Contents: MarkupContent{Kind: "markdown", Value: md.String()},
						}
					}
				}
			}
		}

		var sym *checker.Symbol
		if s, ok := c.GetScope().Resolve(word); ok {
			sym = s
		} else {
			for _, s := range c.AllDeclaredVars() {
				if s.Name == word {
					sym = s
					break
				}
			}
		}
		if sym != nil {
			var md strings.Builder
			switch sym.Kind {
			case checker.SymVar:
				typeStr := sym.Type
				if typeStr == "" {
					typeStr = "any"
				}
				prefix := "(variable)"
				if sym.IsGlobal {
					prefix = "(global variable)"
				}
				if sym.ValueStr != "" {
					md.WriteString(fmt.Sprintf("```xru\n%s %s: %s = %s\n```\n", prefix, sym.Name, typeStr, sym.ValueStr))
				} else {
					md.WriteString(fmt.Sprintf("```xru\n%s %s: %s\n```\n", prefix, sym.Name, typeStr))
				}
			case checker.SymConst:
				typeStr := sym.Type
				if typeStr == "" {
					typeStr = "any"
				}
				prefix := "(constant)"
				if sym.IsGlobal {
					prefix = "(global constant)"
				}
				if sym.ValueStr != "" {
					md.WriteString(fmt.Sprintf("```xru\n%s %s: %s = %s\n```\n", prefix, sym.Name, typeStr, sym.ValueStr))
				} else {
					md.WriteString(fmt.Sprintf("```xru\n%s %s: %s\n```\n", prefix, sym.Name, typeStr))
				}
			case checker.SymFunc:
				var params []string
				for _, param := range sym.Params {
					pType := param.Type
					if pType == "" {
						pType = "any"
					}
					params = append(params, fmt.Sprintf("%s: %s", param.Name, pType))
				}
				retType := sym.Type
				if retType == "" {
					retType = "void"
				}
				prefix := "func"
				if sym.IsGlobal {
					prefix = "global func"
				}
				md.WriteString(fmt.Sprintf("```xru\n%s %s(%s): %s\n```\n", prefix, sym.Name, strings.Join(params, ", "), retType))
			case checker.SymUnit:
				md.WriteString(fmt.Sprintf("```xru\n(module) %s\n```\n", sym.Name))
			default:
				md.WriteString(fmt.Sprintf("```xru\n%s: %s\n```\n", sym.Name, sym.Type))
			}
			fileContent := s.getFileContentForPos(uri, sym.Pos.Filename)
			if doc := extractDocComment(fileContent, sym.Pos.Line); doc != "" {
				md.WriteString("\n" + doc + "\n")
			}
			return &Hover{
				Contents: MarkupContent{
					Kind:  "markdown",
					Value: md.String(),
				},
			}
		}
	}

	// 3. Check exported workspace modules via _mod.xru
	fileDir := filepath.Dir(uriToPath(uri))
	if modFilePath := modules.FindModConfigFile(fileDir); modFilePath != "" {
		if modCfg, errs := modules.LoadModConfig(modFilePath); len(errs) == 0 {
			for _, cand := range modCfg.GetAllExportedCandidates() {
				if cand.Kind != "module" && cand.Name == word {
					var md strings.Builder
					md.WriteString(fmt.Sprintf("```xru\n(exported %s) %s::%s: %s\n```\n", cand.Kind, cand.ModuleName, cand.Name, cand.Type))
					md.WriteString(fmt.Sprintf("\n*Exported in namespace `%s`*\n", cand.Namespace))
					md.WriteString(fmt.Sprintf("\n💡 *To import:* `%s`", cand.ImportStatement))
					return &Hover{
						Contents: MarkupContent{Kind: "markdown", Value: md.String()},
					}
				} else if cand.Kind == "module" && cand.Name == word {
					var md strings.Builder
					md.WriteString(fmt.Sprintf("```xru\n(module) %s\n```\n", cand.Name))
					md.WriteString(fmt.Sprintf("\n*File: `%s`*\n", filepath.Base(cand.FilePath)))
					md.WriteString(fmt.Sprintf("\n💡 *To import:* `%s`", cand.ImportStatement))
					return &Hover{
						Contents: MarkupContent{Kind: "markdown", Value: md.String()},
					}
				}
			}
		}
	}

	return nil
}

func (s *Server) handleDocumentSymbols(uri string) []DocumentSymbol {
	text, ok := s.docs[uri]
	if !ok {
		return []DocumentSymbol{}
	}

	var symbols []DocumentSymbol
	lines := strings.Split(text, "\n")
	for idx, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "//") {
			continue
		}

		// Use imports
		if m := regexp.MustCompile(`^\s*use\s+([a-zA-Z0-9_.]+)(?:\s+as\s+([a-zA-Z_][a-zA-Z0-9_]*))?`).FindStringSubmatch(line); len(m) > 1 {
			name := m[1]
			detail := "import"
			if len(m) > 2 && m[2] != "" {
				name = m[2]
				detail = "import as " + m[2]
			}
			symbols = append(symbols, DocumentSymbol{
				Name:   name,
				Detail: detail,
				Kind:   SymbolKindModule,
				Range:  Range{Start: Position{Line: idx, Character: 0}, End: Position{Line: idx, Character: len(line)}},
			})
		}

		// Unit
		if m := regexp.MustCompile(`^\s*unit\s+([a-zA-Z_][a-zA-Z0-9_]*)`).FindStringSubmatch(line); len(m) > 1 {
			symbols = append(symbols, DocumentSymbol{
				Name:   m[1],
				Detail: "unit",
				Kind:   SymbolKindClass,
				Range:  Range{Start: Position{Line: idx, Character: 0}, End: Position{Line: idx, Character: len(line)}},
			})
		}

		// Func
		if m := regexp.MustCompile(`^\s*(?:global\s+)?func\s+([a-zA-Z_][a-zA-Z0-9_]*)`).FindStringSubmatch(line); len(m) > 1 {
			symbols = append(symbols, DocumentSymbol{
				Name:   m[1],
				Detail: "function",
				Kind:   SymbolKindFunction,
				Range:  Range{Start: Position{Line: idx, Character: 0}, End: Position{Line: idx, Character: len(line)}},
			})
		}

		// Arg
		if m := regexp.MustCompile(`^\s*arg\s+([a-zA-Z_][a-zA-Z0-9_]*)`).FindStringSubmatch(line); len(m) > 1 {
			symbols = append(symbols, DocumentSymbol{
				Name:   m[1],
				Detail: "argument",
				Kind:   SymbolKindProperty,
				Range:  Range{Start: Position{Line: idx, Character: 0}, End: Position{Line: idx, Character: len(line)}},
			})
		}

		// Mod
		if m := regexp.MustCompile(`^\s*mod\s+([a-zA-Z_][a-zA-Z0-9_]*)`).FindStringSubmatch(line); len(m) > 1 {
			symbols = append(symbols, DocumentSymbol{
				Name:   m[1],
				Detail: "module",
				Kind:   SymbolKindModule,
				Range:  Range{Start: Position{Line: idx, Character: 0}, End: Position{Line: idx, Character: len(line)}},
			})
		}

		// From ... use
		if m := regexp.MustCompile(`^\s*from\s+"[^"]+"\s+use\s+(?:([a-zA-Z_][a-zA-Z0-9_]*)(?:\s+as\s+([a-zA-Z_][a-zA-Z0-9_]*))?|([a-zA-Z_][a-zA-Z0-9_]*)::([a-zA-Z_][a-zA-Z0-9_]*))`).FindStringSubmatch(line); len(m) > 1 {
			name := m[1]
			detail := "from import"
			if len(m) > 2 && m[2] != "" {
				name = m[2]
				detail = "from import as " + m[2]
			} else if len(m) > 4 && m[4] != "" {
				name = m[4]
				detail = "imported member from " + m[3]
			}
			if name != "" {
				symbols = append(symbols, DocumentSymbol{
					Name:   name,
					Detail: detail,
					Kind:   SymbolKindModule,
					Range:  Range{Start: Position{Line: idx, Character: 0}, End: Position{Line: idx, Character: len(line)}},
				})
			}
		}

		// Let / const / global let / global const
		if m := regexp.MustCompile(`^\s*(?:global\s+)?(?:let|const)\s+([a-zA-Z_][a-zA-Z0-9_]*)`).FindStringSubmatch(line); len(m) > 1 {
			symbols = append(symbols, DocumentSymbol{
				Name:   m[1],
				Detail: "variable",
				Kind:   SymbolKindVariable,
				Range:  Range{Start: Position{Line: idx, Character: 0}, End: Position{Line: idx, Character: len(line)}},
			})
		}
	}

	return symbols
}

func (s *Server) getFileContentForPos(uri string, filename string) string {
	if filename != "" {
		if content, ok := s.docs[pathToURI(filename)]; ok {
			return content
		}
		if data, err := os.ReadFile(filename); err == nil {
			return string(data)
		}
	}
	if content, ok := s.docs[uri]; ok {
		return content
	}
	return ""
}

func extractDocComment(fileContent string, targetLine int) string {
	if targetLine <= 1 || fileContent == "" {
		return ""
	}
	lines := strings.Split(fileContent, "\n")
	return extractDocCommentForLine(lines, targetLine-1)
}

func extractDocCommentForLine(lines []string, declLine0Idx int) string {
	idx := declLine0Idx - 1
	if idx < 0 || idx >= len(lines) {
		return ""
	}

	var commentLines []string
	inBlockComment := false

	for idx >= 0 {
		rawLine := lines[idx]
		line := strings.TrimSpace(rawLine)

		if inBlockComment {
			if strings.HasPrefix(line, "/*") {
				cleaned := strings.TrimPrefix(line, "/*")
				cleaned = strings.TrimPrefix(cleaned, "*")
				cleaned = strings.TrimSpace(cleaned)
				if cleaned != "" {
					commentLines = append([]string{cleaned}, commentLines...)
				}
				break
			}
			cleaned := strings.TrimPrefix(line, "*")
			cleaned = strings.TrimSpace(cleaned)
			commentLines = append([]string{cleaned}, commentLines...)
			idx--
			continue
		}

		if strings.HasSuffix(line, "*/") {
			inBlockComment = true
			cleaned := strings.TrimSuffix(line, "*/")
			cleaned = strings.TrimPrefix(cleaned, "/*")
			cleaned = strings.TrimPrefix(cleaned, "*")
			cleaned = strings.TrimSpace(cleaned)
			if cleaned != "" {
				commentLines = append([]string{cleaned}, commentLines...)
			}
			if strings.HasPrefix(line, "/*") {
				break
			}
			idx--
			continue
		}

		if strings.HasPrefix(line, "//") {
			cleaned := strings.TrimPrefix(line, "//")
			cleaned = strings.TrimSpace(cleaned)
			if cleaned != "" {
				commentLines = append([]string{cleaned}, commentLines...)
			}
			idx--
			continue
		}

		// Non-comment line -> stop scanning
		break
	}

	return strings.Join(commentLines, "\n\n")
}

func formatPropertyHover(propInfo *PropertyInfo) *Hover {
	var md strings.Builder
	if propInfo.IsMethod {
		sig := strings.TrimPrefix(propInfo.PropType, "func")
		sig = strings.TrimSpace(sig)
		md.WriteString(fmt.Sprintf("```xru\n(method) %s.%s%s\n```\n", propInfo.TypeName, propInfo.PropName, sig))
	} else {
		opt := ""
		if propInfo.IsOptional {
			opt = "?"
		}
		md.WriteString(fmt.Sprintf("```xru\n(property) %s.%s%s: %s\n```\n", propInfo.TypeName, propInfo.PropName, opt, propInfo.PropType))
	}
	if propInfo.Doc != "" {
		md.WriteString("\n" + propInfo.Doc + "\n")
	}
	return &Hover{
		Contents: MarkupContent{
			Kind:  "markdown",
			Value: md.String(),
		},
	}
}

