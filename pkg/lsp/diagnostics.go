package lsp

import (
	"fmt"
	"net/url"
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

	// 2. Collect checker semantic errors only if syntax was valid
	if len(p.Errors()) == 0 && prog != nil {
		c := checker.New()
		c.SetCurrentFile(filename)
		c.Check(prog)
		for _, errStr := range c.Errors() {
			diag := parseErrorStringToDiagnostic(errStr)
			diagnostics = append(diagnostics, diag)
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
		if err == nil {
			return u.Path
		}
		return strings.TrimPrefix(uri, "file://")
	}
	return uri
}

func pathToURI(path string) string {
	if strings.HasPrefix(path, "file://") {
		return path
	}
	return "file://" + path
}
