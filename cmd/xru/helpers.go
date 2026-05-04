package main

import (
	"fmt"
	"os"
	"strings"
)

var currentFile string
var verbose bool
var terminalArgs []string

func unescape(s string) string {
	s = strings.ReplaceAll(s, "\\n", "\n")
	s = strings.ReplaceAll(s, "\\t", "\t")
	return s
}

func checkSyntaxError(val string, line int) {
	if !strings.HasPrefix(val, "[SYNTAX_ERROR:") && !strings.HasPrefix(val, "[ERROR:") {
		return
	}
	msg := val
	switch val {
	case "[SYNTAX_ERROR: UNCLOSED_QUOTE]":
		msg = "missing terminating '\"' or \"'\" character"
	case "[SYNTAX_ERROR: UNCLOSED_BRACE]":
		msg = "missing terminating '}' for variable interpolation"
	case "[SYNTAX_ERROR: MISSING_QUOTES]":
		msg = "string literals must be enclosed in quotes (e.g. \"text\")"
	}
	fmt.Printf("%s:%d: %serror:%s %s\n", currentFile, line, colorRed, colorReset, msg)
	os.Exit(1)
}
