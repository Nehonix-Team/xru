package lsp

import (
	"fmt"
	"net/url"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/Nehonix-Team/xru/pkg/checker"
	"github.com/Nehonix-Team/xru/pkg/lexer"
	"github.com/Nehonix-Team/xru/pkg/parser"
)

var errRegex = regexp.MustCompile(`^(?:.*?:)?(\d+):(\d+):\s*(?:\[([^\]]+)\]\s*)?(.*)$`)
var quoteRegex = regexp.MustCompile(`'([^']+)'`)

func (s *Server) validateAndPublish(uri, text string) {
	filename := uriToPath(uri)
	l := lexer.New(filename, text)
	p := parser.New(l)
	prog := p.ParseProgram()

	diagnostics := make([]Diagnostic, 0)

	// 1. Collect parser syntax errors
	for _, errStr := range p.Errors() {
		diag := parseErrorStringToDiagnostic(errStr)
		diagnostics = append(diagnostics, diag)
	}

	// 2. Collect checker semantic errors
	if prog != nil {
		c := checker.New()
		c.SetCurrentFile(filename)
		c.Check(prog)
		for _, errStr := range c.Errors() {
			diag := parseErrorStringToDiagnostic(errStr)
			diagnostics = append(diagnostics, diag)
		}

		// 3. Collect unused variable diagnostics (dimmed/flouté in IDE via DiagnosticTagUnnecessary)
		for _, sym := range c.UnusedSymbols() {
			line := sym.Pos.Line - 1
			col := sym.Pos.Column - 1
			if line < 0 {
				line = 0
			}
			if col < 0 {
				col = 0
			}
			length := len(sym.Name)
			if length <= 0 {
				length = 1
			}

			msg := fmt.Sprintf("Variable '%s' is declared but never used", sym.Name)
			if sym.Kind == checker.SymConst {
				msg = fmt.Sprintf("Constant '%s' is declared but never used", sym.Name)
			}

			diagnostics = append(diagnostics, Diagnostic{
				Range: Range{
					Start: Position{Line: line, Character: col},
					End:   Position{Line: line, Character: col + length},
				},
				Severity: SeverityHint,
				Source:   "xru",
				Message:  msg,
				Tags:     []DiagnosticTag{DiagnosticTagUnnecessary},
			})
		}

		lines := strings.Split(text, "\n")
		fullLine := func(line int) Range {
			end := 0
			if line >= 0 && line < len(lines) {
				end = len(strings.TrimRight(lines[line], "\r"))
			}
			return Range{Start: Position{Line: line, Character: 0}, End: Position{Line: line, Character: end}}
		}

		// 4. Imports 'use' / 'from ... use' non utilisés
		for _, sym := range c.UnusedImports() {
			line := sym.Pos.Line - 1
			if line < 0 {
				line = 0
			}
			msg := fmt.Sprintf("Module '%s' is imported but never used", sym.Name)
			if sym.Kind == checker.SymFunc {
				msg = fmt.Sprintf("Imported function '%s' is never used", sym.Name)
			} else if sym.Kind == checker.SymConst || sym.Kind == checker.SymVar {
				msg = fmt.Sprintf("Imported symbol '%s' is never used", sym.Name)
			}

			rng := fullLine(line)
			if line >= 0 && line < len(lines) {
				lineText := lines[line]
				// Si l'import est destructuré ou groupé ({, (, ::) ou membre individuel,
				// on cible précisément le nom du symbole pour ne pas griser toute la ligne si d'autres symboles sont utilisés
				isDestructuredOrMember := strings.Contains(lineText, "{") || strings.Contains(lineText, "(") || strings.Contains(lineText, "::") || sym.Kind != checker.SymUnit
				if isDestructuredOrMember {
					if sym.Pos.Column > 0 {
						col := sym.Pos.Column - 1
						rng = Range{
							Start: Position{Line: line, Character: col},
							End:   Position{Line: line, Character: col + len(sym.Name)},
						}
					} else {
						reWord := regexp.MustCompile(`\b` + regexp.QuoteMeta(sym.Name) + `\b`)
						locs := reWord.FindAllStringIndex(lineText, -1)
						if len(locs) > 0 {
							loc := locs[len(locs)-1]
							rng = Range{
								Start: Position{Line: line, Character: loc[0]},
								End:   Position{Line: line, Character: loc[1]},
							}
						}
					}
				}
			}

			diagnostics = append(diagnostics, Diagnostic{
				Range:    rng,
				Severity: SeverityHint,
				Source:   "xru",
				Message:  msg,
				Tags:     []DiagnosticTag{DiagnosticTagUnnecessary},
			})
		}

		// 5. Code inatteignable après un appel 'never' (std.fatal, Sys.exit)
		for _, pos := range c.UnreachablePositions() {
			line := pos.Line - 1
			if line < 0 {
				line = 0
			}
			diagnostics = append(diagnostics, Diagnostic{
				Range:    fullLine(line),
				Severity: SeverityHint,
				Source:   "xru",
				Message:  "Unreachable code",
				Tags:     []DiagnosticTag{DiagnosticTagUnnecessary},
			})
		}
	}

	_ = s.writeNotification("textDocument/publishDiagnostics", PublishDiagnosticsParams{
		URI:         uri,
		Diagnostics: diagnostics,
	})
}

func parseErrorStringToDiagnostic(errStr string) Diagnostic {
	matches := errRegex.FindStringSubmatch(errStr)
	line := 0
	col := 0
	msg := errStr
	length := 1

	if len(matches) >= 5 {
		if l, err := strconv.Atoi(matches[1]); err == nil && l > 0 {
			line = l - 1
		}
		if c, err := strconv.Atoi(matches[2]); err == nil && c > 0 {
			col = c - 1
		}
		code := matches[3]
		errMsg := matches[4]
		if code != "" {
			msg = fmt.Sprintf("[%s] %s", code, errMsg)
		} else {
			msg = errMsg
		}

		if qm := quoteRegex.FindStringSubmatch(errMsg); len(qm) > 1 {
			length = len(qm[1])
		}
	}

	if length <= 0 {
		length = 1
	}

	return Diagnostic{
		Range: Range{
			Start: Position{Line: line, Character: col},
			End:   Position{Line: line, Character: col + length},
		},
		Severity: SeverityError,
		Source:   "xru",
		Message:  msg,
	}
}

func uriToPath(uri string) string {
	if strings.HasPrefix(uri, "file://") {
		u, err := url.Parse(uri)
		var p string
		if err == nil {
			p, _ = url.PathUnescape(u.Path)
		} else {
			p = strings.TrimPrefix(uri, "file://")
		}
		// Windows drive letter handling: "/C:/path" -> "C:/path" or "/c:/path" -> "C:/path"
		if len(p) >= 3 && p[0] == '/' && p[2] == ':' {
			p = p[1:]
		}
		return filepath.Clean(filepath.FromSlash(p))
	}
	return filepath.Clean(filepath.FromSlash(uri))
}

func pathToURI(path string) string {
	p := filepath.ToSlash(filepath.Clean(path))
	if strings.HasPrefix(p, "file://") {
		return p
	}
	// Windows drive letter: "C:/foo" -> "file:///C:/foo"
	if len(p) >= 2 && p[1] == ':' {
		return "file:///" + p
	}
	if !strings.HasPrefix(p, "/") {
		return "file:///" + p
	}
	return "file://" + p
}
