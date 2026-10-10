package lsp

import (
	"fmt"
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

	if docItem := GetDocForWord(fullExpr, word); docItem != nil {
		return &Hover{
			Contents: MarkupContent{
				Kind:  "markdown",
				Value: FormatMarkdown(docItem),
			},
		}
	}

	// 2. Check user-defined symbols (variables, constants, arguments, functions, modules)
	filename := uriToPath(uri)
	l := lexer.New(filename, text)
	p := parser.New(l)
	prog := p.ParseProgram()
	if prog != nil {
		c := checker.New()
		c.Check(prog)

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
						if method.Pos.Line > 0 {
							md.WriteString(fmt.Sprintf("\n*Declared in module '%s'*", scopeName))
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
			if sym.Pos.Line > 0 {
				md.WriteString(fmt.Sprintf("\n*Declared at line %d, column %d*", sym.Pos.Line, sym.Pos.Column))
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
