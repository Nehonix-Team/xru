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

	// 1. Si l'expression est qualifiée avec un point (ex: Crypto.GetRandomBytes)
	if strings.Contains(fullExpr, ".") {
		parts := strings.Split(fullExpr, ".")
		objectName := parts[0]
		if objectName != "" && objectName != "string" && objectName != "list" && !strings.HasPrefix(objectName, "\"") && !strings.HasPrefix(objectName, "'") {
			if c != nil {
				if _, inScope := c.GetScope().Resolve(objectName); !inScope {
					// L'objet n'est pas dans le scope (ex: 'Crypto' non importé)
					return nil
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

	if docItem := GetDocForWord(fullExpr, word); docItem != nil {
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

		if sym, ok := c.GetScope().Resolve(word); ok {
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
	idx := targetLine - 2 // 0-indexed line immediately before declaration
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
