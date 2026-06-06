package engine

import (
	"fmt"
	"strings"
)

func unescape(s string) string {
	s = strings.ReplaceAll(s, "\\n", "\n")
	s = strings.ReplaceAll(s, "\\t", "\t")
	return s
}

func checkSyntaxError(val string, line int, r *Runner) error {
	if !strings.HasPrefix(val, "[SYNTAX_ERROR:") && !strings.HasPrefix(val, "[ERROR:") {
		return nil
	}
	msg := val
	switch val {
	case "[SYNTAX_ERROR: UNCLOSED_QUOTE]":
		msg = "missing terminating '\"' or \"'\" character"
	case "[SYNTAX_ERROR: UNCLOSED_BRACE]":
		msg = "missing terminating '}' for variable interpolation"
	case "[SYNTAX_ERROR: MISSING_QUOTES]":
		msg = "string literals must be enclosed in quotes (e.g. \"text\")"
	case "[SYNTAX_ERROR: INVALID_IDENTIFIER]":
		msg = "invalid variable identifier (must follow JS/TS naming rules: no spaces, dashes, etc)"
	}
	return fmt.Errorf("%s:%d: %s", r.CurrentFile, line, msg)
}

