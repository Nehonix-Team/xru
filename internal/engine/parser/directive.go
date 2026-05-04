package parser

import (
	"strings"
	"unicode"

	"github.com/Nehonix-Team/xru/internal/engine/ast"
)

func parseTarget(line string, strictQuotes bool) (string, string, bool) {
	line = strings.TrimSpace(line)
	raw := false
	if strings.HasSuffix(line, " --raw") {
		raw = true
		line = strings.TrimSpace(strings.TrimSuffix(line, " --raw"))
	}
	inQuote := false
	var quoteChar rune
	idx := -1
	for i, r := range line {
		if r == '"' || r == '\'' {
			if !inQuote {
				inQuote = true
				quoteChar = r
			} else if r == quoteChar {
				inQuote = false
			}
		}
		if !inQuote && i <= len(line)-4 && line[i:i+4] == " as " {
			idx = i
			break
		}
	}
	if inQuote {
		return "[SYNTAX_ERROR: UNCLOSED_QUOTE]", "", false
	}
	target := line
	as := ""
	if idx != -1 {
		target = strings.TrimSpace(line[:idx])
		as = strings.TrimSpace(line[idx+4:])
	}
	if target != "" {
		isQuoted := (len(target) >= 2) && ((target[0] == '"' && target[len(target)-1] == '"') ||
			(target[0] == '\'' && target[len(target)-1] == '\''))

		if isQuoted {
			target = target[1 : len(target)-1]
		} else {
			isNumber := true
			for _, r := range target {
				if !unicode.IsDigit(r) {
					isNumber = false
					break
				}
			}
			if !isNumber && strictQuotes {
				return "[SYNTAX_ERROR: MISSING_QUOTES]", "", false
			}
		}
	}
	return target, as, raw
}

func parseVar(line string) (string, string, bool) {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "let ") {
		return "", "", false
	}
	idx := strings.Index(line, "=")
	if idx == -1 {
		return "", "", false
	}
	name := strings.TrimSpace(line[4:idx])
	val := strings.TrimSpace(line[idx+1:])
	
	if strings.HasPrefix(val, "\"") || strings.HasPrefix(val, "'") {
		quote := val[0]
		if len(val) < 2 || val[len(val)-1] != quote {
			return name, "[SYNTAX_ERROR: UNCLOSED_QUOTE]", true
		}
		val = val[1 : len(val)-1]
	}
	return name, val, true
}

func parseActionLine(line string) (ast.PatchOp, string, string, bool) {
	if strings.HasPrefix(line, "++") { return ast.PatchMerge, "", strings.TrimSpace(line[2:]), true }
	if strings.HasPrefix(line, "--") { return ast.PatchRM, "", strings.TrimSpace(line[2:]), true }
	if strings.HasPrefix(line, ">>") { return ast.PatchRPK, "", strings.TrimSpace(line[2:]), true }
	if strings.HasPrefix(line, "<<") { return ast.PatchAppend, "", strings.TrimSpace(line[2:]), true }
	if strings.HasPrefix(line, "~~") { return ast.PatchRegex, "", strings.TrimSpace(line[2:]), true }
	if strings.HasPrefix(line, "&") {
		parts := strings.SplitN(line[1:], ":", 2)
		opStr := strings.ToLower(parts[0])
		initial := ""
		if len(parts) > 1 { initial = parts[1] }
		var op ast.PatchOp
		switch opStr {
		case "rm": op = ast.PatchRM
		case "merge", "add": op = ast.PatchMerge
		case "append": op = ast.PatchAppend
		case "regex": op = ast.PatchRegex
		case "rpk", "rp-k": op = ast.PatchRPK
		case "rpv", "rp-v": op = ast.PatchRPV
		}
		if op != "" { return op, "", strings.TrimSpace(initial), true }
	}
	upper := strings.ToUpper(line)
	keywords := []struct{k string; op ast.PatchOp}{
		{"MERGE ", ast.PatchMerge},
		{"SET ", ast.PatchSet},
		{"REMOVE ", ast.PatchRM},
		{"PUSH ", ast.PatchPush},
	}
	for _, kv := range keywords {
		if strings.HasPrefix(upper, kv.k) {
			rest := strings.TrimSpace(line[len(kv.k):])
			path := ""
			initial := rest
			idx := -1
			for i, r := range rest {
				if r == '{' || r == '[' || r == '"' || r == '\'' || unicode.IsSpace(r) {
					idx = i
					break
				}
			}
			if idx != -1 {
				path = strings.TrimSpace(rest[:idx])
				initial = strings.TrimSpace(rest[idx:])
			} else {
				path = rest
				initial = ""
			}
			return kv.op, path, initial, true
		}
	}
	return "", "", "", false
}
