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
			a = PatchAction{Op: pending.op, Value: ParseValue(body)}
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
			currentRule = &Rule{Type: RuleTypeCreate, Target: strings.TrimSpace(strings.TrimPrefix(trimmed, "#CREATE:"))}
			continue
		}
		if strings.HasPrefix(trimmed, "#SELECT:") {
			commitPending()
			if currentRule != nil {
				rf.Rules = append(rf.Rules, *currentRule)
			}
			currentRule = &Rule{Type: RuleTypeSelect, Target: strings.TrimSpace(strings.TrimPrefix(trimmed, "#SELECT:"))}
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
			if currentRule != nil {
				rf.Rules = append(rf.Rules, *currentRule)
			}
			currentRule = &Rule{Type: RuleTypeLog, Target: strings.TrimSpace(strings.TrimPrefix(trimmed, "#LOG:"))}
			continue
		}

		if strings.HasPrefix(trimmed, "#ASSERT:") {
			commitPending()
			if currentRule != nil {
				rf.Rules = append(rf.Rules, *currentRule)
			}
			currentRule = &Rule{Type: RuleTypeAssert, Target: strings.TrimSpace(strings.TrimPrefix(trimmed, "#ASSERT:"))}
			continue
		}

		if strings.HasPrefix(trimmed, "#INCLUDE:") {
			commitPending()
			if currentRule != nil {
				rf.Rules = append(rf.Rules, *currentRule)
			}
			currentRule = &Rule{Type: RuleTypeInclude, Target: strings.TrimSpace(strings.TrimPrefix(trimmed, "#INCLUDE:"))}
			continue
		}

		if strings.HasPrefix(trimmed, "#EXEC:") {
			commitPending()
			if currentRule != nil {
				rf.Rules = append(rf.Rules, *currentRule)
			}
			currentRule = &Rule{Type: RuleTypeExec, Target: strings.TrimSpace(strings.TrimPrefix(trimmed, "#EXEC:"))}
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

		// Patch ops
		var op PatchOp
		foundOp := false
		if strings.HasPrefix(trimmed, "&rm:") {
			op = PatchRM; foundOp = true
		} else if strings.HasPrefix(trimmed, "&rp-k:") || strings.HasPrefix(trimmed, "&rp-0:") {
			op = PatchRPK; foundOp = true
		} else if strings.HasPrefix(trimmed, "&rp-v:") || strings.HasPrefix(trimmed, "&rp-1:") {
			op = PatchRPV; foundOp = true
		} else if strings.HasPrefix(trimmed, "&merge:") || strings.HasPrefix(trimmed, "&add:") {
			op = PatchMerge; foundOp = true
		} else if strings.HasPrefix(trimmed, "&append:") {
			op = PatchAppend; foundOp = true
		} else if strings.HasPrefix(trimmed, "&regex:") {
			op = PatchRegex; foundOp = true
		}

		if foundOp {
			commitPending()
			// Extract initial data if on same line
			var initial string
			if op == PatchRM { initial = strings.TrimPrefix(trimmed, "&rm:") }
			if op == PatchRPK { 
				initial = strings.TrimPrefix(trimmed, "&rp-k:") 
				if initial == trimmed { initial = strings.TrimPrefix(trimmed, "&rp-0:") }
			}
			if op == PatchRPV { 
				initial = strings.TrimPrefix(trimmed, "&rp-v:") 
				if initial == trimmed { initial = strings.TrimPrefix(trimmed, "&rp-1:") }
			}
			if op == PatchMerge { 
				initial = strings.TrimPrefix(trimmed, "&merge:") 
				if initial == trimmed { initial = strings.TrimPrefix(trimmed, "&add:") }
			}
			if op == PatchAppend { initial = strings.TrimPrefix(trimmed, "&append:") }
			if op == PatchRegex { initial = strings.TrimPrefix(trimmed, "&regex:") }
			
			pending = &pendingAction{isInject: false, op: op, buf: []string{initial}}
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
