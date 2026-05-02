/***************************************************************************
 * XFPM — XRU Language Parser
 *
 * Parses .xru source into a RuleFile AST with structured values.
 ***************************************************************************** */

package engine

import (
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

	var currentRule *Rule
	var globalRule *Rule
	
	type pendingAction struct {
		isInject bool
		lang     string
		key      string
		op       PatchOp
		path     string
		buf      []string
	}
	var pending *pendingAction

	commitPending := func() {
		if pending == nil {
			return
		}
		body := strings.Join(pending.buf, "\n")
		var a Action
		if pending.isInject {
			a = InjectAction{Lang: pending.lang, Key: pending.key, Code: body}
		} else {
			a = PatchAction{Op: pending.op, Path: pending.path, Value: ParseValue(body)}
		}

		if currentRule != nil {
			currentRule.Actions = append(currentRule.Actions, a)
		} else {
			if globalRule == nil {
				globalRule = &Rule{Type: RuleTypeGlobal}
			}
			globalRule.Actions = append(globalRule.Actions, a)
		}
		pending = nil
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "//") {
			continue
		}

		if strings.HasPrefix(trimmed, "#BEGIN:") {
			commitPending()
			if currentRule != nil {
				rf.Rules = append(rf.Rules, *currentRule)
			}
			currentRule = &Rule{Type: RuleTypeBegin, Target: strings.TrimSpace(strings.TrimPrefix(trimmed, "#BEGIN:"))}
			continue
		}

		if strings.HasPrefix(trimmed, "#CREATE:") {
			commitPending()
			if currentRule != nil {
				rf.Rules = append(rf.Rules, *currentRule)
			}
			target, as := parseTarget(strings.TrimPrefix(trimmed, "#CREATE:"))
			currentRule = &Rule{Type: RuleTypeCreate, Target: target, As: as}
			continue
		}

		if strings.HasPrefix(trimmed, "#SELECT:") {
			commitPending()
			target, as := parseTarget(strings.TrimPrefix(trimmed, "#SELECT:"))
			rf.Rules = append(rf.Rules, Rule{Type: RuleTypeSelect, Target: target, As: as})
			continue
		}

		if strings.HasPrefix(trimmed, "#BREAK:") || strings.HasPrefix(trimmed, "#EXIT:") {
			commitPending()
			if currentRule != nil {
				rf.Rules = append(rf.Rules, *currentRule)
			}
			val := strings.TrimPrefix(trimmed, "#BREAK:")
			if val == trimmed {
				val = strings.TrimPrefix(trimmed, "#EXIT:")
			}
			currentRule = &Rule{Type: RuleTypeBreak, Target: strings.TrimSpace(val)}
			continue
		}

		if strings.HasPrefix(trimmed, "#LOG:") {
			commitPending()
			target, as := parseTarget(strings.TrimPrefix(trimmed, "#LOG:"))
			rf.Rules = append(rf.Rules, Rule{Type: RuleTypeLog, Target: target, As: as})
			continue
		}

		if strings.HasPrefix(trimmed, "#ASSERT:") {
			commitPending()
			target, as := parseTarget(strings.TrimPrefix(trimmed, "#ASSERT:"))
			rf.Rules = append(rf.Rules, Rule{Type: RuleTypeAssert, Target: target, As: as})
			continue
		}

		if strings.HasPrefix(trimmed, "#INCLUDE:") {
			commitPending()
			target, as := parseTarget(strings.TrimPrefix(trimmed, "#INCLUDE:"))
			rf.Rules = append(rf.Rules, Rule{Type: RuleTypeInclude, Target: target, As: as})
			continue
		}

		if strings.HasPrefix(trimmed, "#EXEC:") {
			commitPending()
			target, as := parseTarget(strings.TrimPrefix(trimmed, "#EXEC:"))
			rf.Rules = append(rf.Rules, Rule{Type: RuleTypeExec, Target: target, As: as})
			continue
		}

		if trimmed == "#END" || (currentRule != nil && strings.HasPrefix(trimmed, "#END:"+currentRule.Target)) {
			commitPending()
			if currentRule != nil {
				rf.Rules = append(rf.Rules, *currentRule)
				currentRule = nil
			}
			continue
		}

		if strings.HasPrefix(trimmed, "@") && strings.Contains(trimmed, "INJECT:") {
			commitPending()
			tag := strings.TrimPrefix(trimmed, "@")
			idx := strings.Index(tag, "INJECT:")
			lang := tag[:idx]
			key := strings.TrimSpace(tag[idx+len("INJECT:"):])
			pending = &pendingAction{isInject: true, lang: lang, key: key}
			continue
		}

		if trimmed == "@END" {
			commitPending()
			continue
		}

		// Actions (Symbols: ++ -- >> << ~~) or Keywords (MERGE SET REMOVE PUSH)
		op, path, initial, ok := parseActionLine(trimmed)
		if ok {
			commitPending()
			pending = &pendingAction{isInject: false, op: op, path: path, buf: []string{initial}}
			continue
		}

		if currentRule != nil && currentRule.Type == RuleTypeCreate {
			currentRule.Content += line + "\n"
			continue
		}

		if pending != nil {
			pending.buf = append(pending.buf, line)
		}
	}

	commitPending()
	if currentRule != nil {
		rf.Rules = append(rf.Rules, *currentRule)
	}
	if globalRule != nil {
		rf.Rules = append(rf.Rules, *globalRule)
	}

	return rf, nil
}

func parseTarget(line string) (string, string) {
	line = strings.TrimSpace(line)
	// Look for " as " but ignore if it's inside quotes
	inQuote := false
	idx := -1
	for i := 0; i < len(line)-4; i++ {
		if line[i] == '"' || line[i] == '\'' { inQuote = !inQuote }
		if !inQuote && line[i:i+4] == " as " {
			idx = i
			break
		}
	}

	if idx != -1 {
		return strings.TrimSpace(line[:idx]), strings.TrimSpace(line[idx+4:])
	}
	return line, ""
}

func parseActionLine(line string) (PatchOp, string, string, bool) {
	// Symbols (Direct root actions)
	if strings.HasPrefix(line, "++") { return PatchMerge, "", strings.TrimSpace(line[2:]), true }
	if strings.HasPrefix(line, "--") { return PatchRM, "", strings.TrimSpace(line[2:]), true }
	if strings.HasPrefix(line, ">>") { return PatchRPK, "", strings.TrimSpace(line[2:]), true }
	if strings.HasPrefix(line, "<<") { return PatchAppend, "", strings.TrimSpace(line[2:]), true }
	if strings.HasPrefix(line, "~~") { return PatchRegex, "", strings.TrimSpace(line[2:]), true }

	// Legacy & Support
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

	// Keywords (Path-based or explicit)
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
			// Path is everything until the first structural character or space
			path := ""
			initial := rest
			
			// Find where the value starts
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
				// No structural character found, might be just a path (for RM)
				path = rest
				initial = ""
			}
			return kv.op, path, initial, true
		}
	}

	return "", "", "", false
}

// ParseValue is a simple recursive descent parser for XRU structures.
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
	if p.pos >= len(p.src) {
		return Literal("")
	}

	switch p.src[p.pos] {
	case '{':
		return p.parseObject()
	case '[':
		return p.parseArray()
	default:
		return p.parseLiteral()
	}
}

func (p *valParser) parseObject() Object {
	obj := make(Object)
	p.pos++ // skip {
	for {
		p.skipWS()
		if p.pos >= len(p.src) || p.src[p.pos] == '}' {
			if p.pos < len(p.src) { p.pos++ }
			break
		}

		key := p.parseKey()
		p.skipWS()
		if p.pos < len(p.src) && p.src[p.pos] == ':' {
			p.pos++
		}
		val := p.parse()
		obj[key] = val

		p.skipWS()
		if p.pos < len(p.src) && p.src[p.pos] == ',' {
			p.pos++
		}
	}
	return obj
}

func (p *valParser) parseArray() Array {
	arr := make(Array, 0)
	p.pos++ // skip [
	for {
		p.skipWS()
		if p.pos >= len(p.src) || p.src[p.pos] == ']' {
			if p.pos < len(p.src) { p.pos++ }
			break
		}

		val := p.parse()
		arr = append(arr, val)

		p.skipWS()
		if p.pos < len(p.src) && p.src[p.pos] == ',' {
			p.pos++
		}
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
			if p.src[p.pos] == quote && p.src[p.pos-1] != '\\' {
				break
			}
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
			if p.src[p.pos] == quote && p.src[p.pos-1] != '\\' {
				break
			}
			p.pos++
		}
		val := string(p.src[start:p.pos])
		if p.pos < len(p.src) { p.pos++ }
		return Literal(unescape(val))
	}

	start := p.pos
	// Parse until next delimiter
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
		// Handle // comments in value
		if p.pos+1 < len(p.src) && p.src[p.pos] == '/' && p.src[p.pos+1] == '/' {
			for p.pos < len(p.src) && p.src[p.pos] != '\n' {
				p.pos++
			}
			continue
		}
		break
	}
}

func ParseFile(path string) (*RuleFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return parseNew(string(data))
}
