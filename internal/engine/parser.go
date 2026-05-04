/***************************************************************************
 * XFPM — XRU Language Parser
 *
 * Parses .xru source into a RuleFile AST with structured values and nesting.
 * Enforces column-0 for structural directives (#), allows spaces after #.
 ***************************************************************************** */

package engine

import (
	"fmt"
	"os"
	"strings"
	"unicode"
)

// Parse converts .xru source text into a RuleFile AST.
func Parse(src string) (*RuleFile, error) {
	return parseNew(src)
}

func parseNew(src string) (*RuleFile, error) {
	lines := strings.Split(src, "\n")
	rf := &RuleFile{}

	var stack []*Rule
	var globalRule *Rule

	type pendingAction struct {
		isInject bool
		lang     string
		key      string
		op       PatchOp
		path     string
		buf      []string
		line     int
		raw      bool
	}
	var pending *pendingAction

	commitPending := func() {
		if pending == nil {
			return
		}
		body := strings.Join(pending.buf, "\n")
		var a Action
		if pending.isInject {
			if !pending.raw {
				body = Dedent(body)
			}
			a = InjectAction{Lang: pending.lang, Key: pending.key, Code: body, Raw: pending.raw, Line: pending.line}
		} else {
			a = PatchAction{Op: pending.op, Path: pending.path, Value: ParseValue(body), Line: pending.line}
		}

		if len(stack) > 0 {
			parent := stack[len(stack)-1]
			parent.Actions = append(parent.Actions, a)
		} else {
			if globalRule == nil {
				globalRule = &Rule{Type: RuleTypeGlobal, Line: pending.line}
			}
			globalRule.Actions = append(globalRule.Actions, a)
		}
		pending = nil
	}

	for i, line := range lines {
		lineNum := i + 1
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "//") {
			continue
		}

		// Directives starting with # (allows leading whitespace for nesting)
		if strings.HasPrefix(trimmed, "#") {
			// Allow spaces after # for nesting visualization: #  IF
			directiveLine := "#" + strings.TrimSpace(trimmed[1:])
			
			if strings.HasPrefix(directiveLine, "#BEGIN:") {
				commitPending()
				target, as, _ := parseTarget(strings.TrimPrefix(directiveLine, "#BEGIN:"), true)
				rule := &Rule{Type: RuleTypeBegin, Target: target, As: as, Line: lineNum}
				stack = append(stack, rule)
				continue
			}
			if strings.HasPrefix(directiveLine, "#CREATE:") {
				commitPending()
				target, as, raw := parseTarget(strings.TrimPrefix(directiveLine, "#CREATE:"), true)
				rule := &Rule{Type: RuleTypeCreate, Target: target, As: as, Raw: raw, Line: lineNum}
				stack = append(stack, rule)
				continue
			}
			if strings.HasPrefix(directiveLine, "#IF:") {
				commitPending()
				cond := strings.TrimSpace(strings.TrimPrefix(directiveLine, "#IF:"))
				rule := &Rule{Type: RuleTypeIf, Target: cond, Line: lineNum}
				stack = append(stack, rule)
				continue
			}
			if strings.HasPrefix(directiveLine, "#ELSEIF:") {
				commitPending()
				if len(stack) > 0 {
					last := stack[len(stack)-1]
					if last.Type == RuleTypeIf || last.Type == RuleTypeElseIf {
						prev := *stack[len(stack)-1]
						stack = stack[:len(stack)-1]
						if len(stack) > 0 {
							stack[len(stack)-1].SubRules = append(stack[len(stack)-1].SubRules, prev)
						} else {
							rf.Rules = append(rf.Rules, prev)
						}
					}
				}
				cond := strings.TrimSpace(strings.TrimPrefix(directiveLine, "#ELSEIF:"))
				rule := &Rule{Type: RuleTypeElseIf, Target: cond, Line: lineNum}
				stack = append(stack, rule)
				continue
			}
			if strings.HasPrefix(directiveLine, "#ELSE") {
				commitPending()
				if len(stack) > 0 {
					last := stack[len(stack)-1]
					if last.Type == RuleTypeIf || last.Type == RuleTypeElseIf {
						prev := *stack[len(stack)-1]
						stack = stack[:len(stack)-1]
						if len(stack) > 0 {
							stack[len(stack)-1].SubRules = append(stack[len(stack)-1].SubRules, prev)
						} else {
							rf.Rules = append(rf.Rules, prev)
						}
					}
				}
				rule := &Rule{Type: RuleTypeElse, Line: lineNum}
				stack = append(stack, rule)
				continue
			}
			if strings.HasPrefix(directiveLine, "#SELECT:") {
				commitPending()
				target, as, _ := parseTarget(strings.TrimPrefix(directiveLine, "#SELECT:"), true)
				rule := Rule{Type: RuleTypeSelect, Target: target, As: as, Line: lineNum}
				if len(stack) > 0 {
					stack[len(stack)-1].SubRules = append(stack[len(stack)-1].SubRules, rule)
				} else {
					rf.Rules = append(rf.Rules, rule)
				}
				continue
			}
			if strings.HasPrefix(directiveLine, "#USE:") {
				commitPending()
				target, as, _ := parseTarget(strings.TrimPrefix(directiveLine, "#USE:"), false)
				rule := Rule{Type: RuleTypeUse, Target: target, As: as, Line: lineNum}
				if len(stack) > 0 {
					stack[len(stack)-1].SubRules = append(stack[len(stack)-1].SubRules, rule)
				} else {
					rf.Rules = append(rf.Rules, rule)
				}
				continue
			}
			if strings.HasPrefix(directiveLine, "#ARG:") {
				commitPending()
				target, as, _ := parseTarget(strings.TrimPrefix(directiveLine, "#ARG:"), true)
				rule := Rule{Type: RuleTypeArg, Target: target, As: as, Line: lineNum}
				if len(stack) > 0 {
					stack[len(stack)-1].SubRules = append(stack[len(stack)-1].SubRules, rule)
				} else {
					rf.Rules = append(rf.Rules, rule)
				}
				continue
			}
			if strings.HasPrefix(directiveLine, "#FOR:") {
				commitPending()
				line := strings.TrimSpace(strings.TrimPrefix(directiveLine, "#FOR:"))
				rule := &Rule{Type: RuleTypeFor, Target: line, Line: lineNum}
				stack = append(stack, rule)
				continue
			}
			// Handle #VAR: name or #TSVAR: name
			if strings.HasSuffix(strings.Split(directiveLine, ":")[0], "VAR") {
				commitPending()
				cmdParts := strings.Split(directiveLine, ":")
				cmdName := strings.TrimPrefix(strings.TrimSpace(cmdParts[0]), "#")
				parts := strings.SplitN(directiveLine, ":", 2)
				if len(parts) == 2 {
					name := strings.TrimSpace(parts[1])
					rule := &Rule{Type: RuleTypeVarBlock, Command: cmdName, Target: name, Line: lineNum}
					stack = append(stack, rule)
					continue
				}
			}
			if strings.HasPrefix(directiveLine, "#INCLUDE:") {
				commitPending()
				target, as, _ := parseTarget(strings.TrimPrefix(directiveLine, "#INCLUDE:"), true)
				rule := Rule{Type: RuleTypeInclude, Target: target, As: as, Line: lineNum}
				if len(stack) > 0 {
					stack[len(stack)-1].SubRules = append(stack[len(stack)-1].SubRules, rule)
				} else {
					rf.Rules = append(rf.Rules, rule)
				}
				continue
			}
			if strings.HasPrefix(directiveLine, "#CALL:") {
				commitPending()
				target, as, _ := parseTarget(strings.TrimPrefix(directiveLine, "#CALL:"), true)
				rule := &Rule{Type: RuleTypeCall, Target: target, As: as, Line: lineNum}
				stack = append(stack, rule)
				continue
			}
			if strings.HasPrefix(directiveLine, "#GLOBAL:") {
				commitPending()
				target := strings.TrimSpace(strings.TrimPrefix(directiveLine, "#GLOBAL:"))
				rule := Rule{Type: RuleTypeGlobal, Target: target, Line: lineNum}
				if len(stack) > 0 {
					stack[len(stack)-1].SubRules = append(stack[len(stack)-1].SubRules, rule)
				} else {
					rf.Rules = append(rf.Rules, rule)
				}
				continue
			}
			if strings.HasPrefix(directiveLine, "#GLOBAL") {
				commitPending()
				rule := &Rule{Type: RuleTypeGlobal, Line: lineNum}
				stack = append(stack, rule)
				continue
			}
			if strings.HasPrefix(directiveLine, "#END") {
				commitPending()
				if len(stack) > 0 {
					rule := *stack[len(stack)-1]
					stack = stack[:len(stack)-1]
					if rule.Type == RuleTypeCreate && !rule.Raw {
						rule.Content = Dedent(rule.Content)
					}
					if len(stack) > 0 {
						stack[len(stack)-1].SubRules = append(stack[len(stack)-1].SubRules, rule)
					} else {
						rf.Rules = append(rf.Rules, rule)
					}
				}
				continue
			}
		}

		// Modular Action: Alias.Method: Target [as Alias]
		if idx := strings.Index(trimmed, "."); idx != -1 {
			colonIdx := strings.Index(trimmed, ":")
			if colonIdx != -1 && colonIdx > idx {
				callPart := strings.TrimSpace(trimmed[:colonIdx])
				if strings.Contains(callPart, ".") && !strings.ContainsAny(callPart, " /\\\"'{}[],") {
					commitPending()
					parts := strings.SplitN(trimmed, ":", 2)
					call := parts[0]
					rest := ""
					if len(parts) > 1 {
						rest = parts[1]
					}
					callParts := strings.SplitN(call, ".", 2)
					module := strings.TrimSpace(callParts[0])
					method := strings.TrimSpace(callParts[1])
					target, as, _ := parseTarget(rest, true)
					if len(stack) > 0 {
						stack[len(stack)-1].SubRules = append(stack[len(stack)-1].SubRules, Rule{Type: RuleTypeModule, Target: module + "." + method, Content: target, As: as, Line: lineNum})
					} else {
						rf.Rules = append(rf.Rules, Rule{Type: RuleTypeModule, Target: module + "." + method, Content: target, As: as, Line: lineNum})
					}
					continue
				}
			}
		}

		if strings.HasPrefix(trimmed, "@") && strings.Contains(trimmed, "INJECT:") {
			commitPending()
			tag := strings.TrimPrefix(trimmed, "@")
			idx := strings.Index(tag, "INJECT:")
			lang := tag[:idx]
			keyPart := strings.TrimSpace(tag[idx+len("INJECT:"):])
			if (len(keyPart) >= 2 && keyPart[0] == '"' && keyPart[len(keyPart)-1] == '"') ||
				(len(keyPart) >= 2 && keyPart[0] == '\'' && keyPart[len(keyPart)-1] == '\'') {
				keyPart = keyPart[1 : len(keyPart)-1]
			}
			raw := false
			if strings.HasSuffix(keyPart, " --raw") {
				raw = true
				keyPart = strings.TrimSpace(strings.TrimSuffix(keyPart, " --raw"))
			}
			pending = &pendingAction{isInject: true, lang: lang, key: keyPart, raw: raw, line: lineNum}
			continue
		}

		if trimmed == "@END" {
			commitPending()
			pending = nil
			continue
		}

		// Variable declaration: let NAME = VALUE
		if name, val, ok := parseVar(trimmed); ok {
			commitPending()
			if len(stack) > 0 {
				last := stack[len(stack)-1]
				if last.Type == RuleTypeBegin || last.Type == RuleTypeCreate {
					last.Actions = append(last.Actions, VarAction{Name: name, Value: val, Line: lineNum})
				} else {
					last.SubRules = append(last.SubRules, Rule{Type: RuleTypeVar, Target: name, Content: val, Line: lineNum})
				}
			} else {
				rf.Rules = append(rf.Rules, Rule{Type: RuleTypeVar, Target: name, Content: val, Line: lineNum})
			}
			continue
		}

		// Actions
		op, path, initial, ok := parseActionLine(trimmed)
		if ok {
			commitPending()
			pending = &pendingAction{isInject: false, op: op, path: path, buf: []string{initial}, line: lineNum}
			continue
		}

		// Enregistrement des lignes pour les actions multi-lignes
		if pending != nil {
			pending.buf = append(pending.buf, line)
			continue
		}

		if trimmed == "" {
			if len(stack) > 0 && (stack[len(stack)-1].Type == RuleTypeCreate || stack[len(stack)-1].Type == RuleTypeVarBlock) {
				stack[len(stack)-1].Content += line + "\n"
			}
			continue
		}

		if len(stack) > 0 && (stack[len(stack)-1].Type == RuleTypeCreate || stack[len(stack)-1].Type == RuleTypeVarBlock) && !strings.HasPrefix(trimmed, "#END") {
			stack[len(stack)-1].Content += line + "\n"
			continue
		}

		// SYNTAX ERROR: Unknown line that isn't a structural directive or valid action
		return nil, fmt.Errorf("%d: syntax error: unknown directive or action '%s'", lineNum, trimmed)
	}

	commitPending()
	for len(stack) > 0 {
		rule := *stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if len(stack) > 0 {
			stack[len(stack)-1].SubRules = append(stack[len(stack)-1].SubRules, rule)
		} else {
			rf.Rules = append(rf.Rules, rule)
		}
	}

	if globalRule != nil {
		rf.Rules = append(rf.Rules, *globalRule)
	}

	return rf, nil
}

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
	// Validation: Si on finit la ligne avec un guillemet ouvert
	if inQuote {
		// On ne peut pas retourner d'erreur ici car la signature ne le permet pas encore,
		// mais on peut marquer la cible comme invalide pour déclencher une erreur plus tard.
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
			// Check if it's a number
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
	
	// Check for unclosed quotes in the value
	if strings.HasPrefix(val, "\"") || strings.HasPrefix(val, "'") {
		quote := val[0]
		if len(val) < 2 || val[len(val)-1] != quote {
			return name, "[SYNTAX_ERROR: UNCLOSED_QUOTE]", true
		}
		val = val[1 : len(val)-1]
	}
	return name, val, true
}

func parseActionLine(line string) (PatchOp, string, string, bool) {
	if strings.HasPrefix(line, "++") { return PatchMerge, "", strings.TrimSpace(line[2:]), true }
	if strings.HasPrefix(line, "--") { return PatchRM, "", strings.TrimSpace(line[2:]), true }
	if strings.HasPrefix(line, ">>") { return PatchRPK, "", strings.TrimSpace(line[2:]), true }
	if strings.HasPrefix(line, "<<") { return PatchAppend, "", strings.TrimSpace(line[2:]), true }
	if strings.HasPrefix(line, "~~") { return PatchRegex, "", strings.TrimSpace(line[2:]), true }
	if strings.HasPrefix(line, "&") {
		parts := strings.SplitN(line[1:], ":", 2)
		opStr := strings.ToLower(parts[0])
		initial := ""
		if len(parts) > 1 { initial = parts[1] }
		var op PatchOp
		switch opStr {
		case "rm": op = PatchRM
		case "merge", "add": op = PatchMerge
		case "append": op = PatchAppend
		case "regex": op = PatchRegex
		case "rpk", "rp-k": op = PatchRPK
		case "rpv", "rp-v": op = PatchRPV
		}
		if op != "" { return op, "", strings.TrimSpace(initial), true }
	}
	upper := strings.ToUpper(line)
	keywords := []struct{k string; op PatchOp}{
		{"MERGE ", PatchMerge},
		{"SET ", PatchSet},
		{"REMOVE ", PatchRM},
		{"PUSH ", PatchPush},
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

func ParseValue(s string) Value {
	p := &valParser{src: []rune(s)}
	return p.parse()
}

type valParser struct {
	src []rune
	pos int
}

func (p *valParser) parse() Value {
	p.skipWS()
	if p.pos >= len(p.src) { return Literal("") }
	switch p.src[p.pos] {
	case '{': return p.parseObject()
	case '[': return p.parseArray()
	default: return p.parseLiteral()
	}
}

func (p *valParser) parseObject() Object {
	obj := make(Object)
	p.pos++
	for {
		p.skipWS()
		if p.pos >= len(p.src) || p.src[p.pos] == '}' {
			if p.pos < len(p.src) { p.pos++ }
			break
		}
		key := p.parseKey()
		p.skipWS()
		if p.pos < len(p.src) && p.src[p.pos] == ':' { p.pos++ }
		val := p.parse()
		obj[key] = val
		p.skipWS()
		if p.pos < len(p.src) && p.src[p.pos] == ',' { p.pos++ }
	}
	return obj
}

func (p *valParser) parseArray() Array {
	arr := make(Array, 0)
	p.pos++
	for {
		p.skipWS()
		if p.pos >= len(p.src) || p.src[p.pos] == ']' {
			if p.pos < len(p.src) { p.pos++ }
			break
		}
		val := p.parse()
		arr = append(arr, val)
		p.skipWS()
		if p.pos < len(p.src) && p.src[p.pos] == ',' { p.pos++ }
	}
	return arr
}

func (p *valParser) parseKey() string {
	p.skipWS()
	if p.pos < len(p.src) && (p.src[p.pos] == '"' || p.src[p.pos] == '\'') {
		quote := p.src[p.pos]
		p.pos++
		start := p.pos
		for p.pos < len(p.src) {
			if p.src[p.pos] == quote && p.src[p.pos-1] != '\\' { break }
			p.pos++
		}
		key := string(p.src[start:p.pos])
		if p.pos < len(p.src) { p.pos++ }
		return unescape(key)
	}
	start := p.pos
	for p.pos < len(p.src) && !unicode.IsSpace(p.src[p.pos]) && p.src[p.pos] != ':' && p.src[p.pos] != '}' && p.src[p.pos] != ',' {
		p.pos++
	}
	return string(p.src[start:p.pos])
}

func (p *valParser) parseLiteral() Literal {
	p.skipWS()
	if p.pos < len(p.src) && (p.src[p.pos] == '"' || p.src[p.pos] == '\'') {
		quote := p.src[p.pos]
		p.pos++
		start := p.pos
		for p.pos < len(p.src) {
			if p.src[p.pos] == quote && p.src[p.pos-1] != '\\' { break }
			p.pos++
		}
		val := string(p.src[start:p.pos])
		if p.pos < len(p.src) { p.pos++ }
		return Literal(unescape(val))
	}
	start := p.pos
	for p.pos < len(p.src) && p.src[p.pos] != '}' && p.src[p.pos] != ']' && p.src[p.pos] != ',' && p.src[p.pos] != '\n' {
		p.pos++
	}
	return Literal(strings.TrimSpace(string(p.src[start:p.pos])))
}

func unescape(s string) string {
	s = strings.ReplaceAll(s, "\\\"", "\"")
	s = strings.ReplaceAll(s, "\\'", "'")
	s = strings.ReplaceAll(s, "\\\\", "\\")
	return s
}

func (p *valParser) skipWS() {
	for p.pos < len(p.src) {
		if unicode.IsSpace(p.src[p.pos]) {
			p.pos++
			continue
		}
		if p.pos+1 < len(p.src) && p.src[p.pos] == '/' && p.src[p.pos+1] == '/' {
			for p.pos < len(p.src) && p.src[p.pos] != '\n' { p.pos++ }
			continue
		}
		break
	}
}

func ParseFile(path string) (*RuleFile, error) {
	data, err := os.ReadFile(path)
	if err != nil { return nil, err }
	return parseNew(string(data))
}
