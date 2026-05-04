package parser

import (
	"strings"
	"unicode"

	"github.com/Nehonix-Team/xru/internal/engine/ast"
)

func ParseValue(s string) ast.Value {
	p := &valParser{src: []rune(s)}
	return p.parse()
}

type valParser struct {
	src []rune
	pos int
}

func (p *valParser) parse() ast.Value {
	p.skipWS()
	if p.pos >= len(p.src) { return ast.Literal("") }
	switch p.src[p.pos] {
	case '{': return p.parseObject()
	case '[': return p.parseArray()
	default: return p.parseLiteral()
	}
}

func (p *valParser) parseObject() ast.Object {
	obj := make(ast.Object)
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

func (p *valParser) parseArray() ast.Array {
	arr := make(ast.Array, 0)
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

func (p *valParser) parseLiteral() ast.Literal {
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
		return ast.Literal(unescape(val))
	}
	start := p.pos
	for p.pos < len(p.src) && p.src[p.pos] != '}' && p.src[p.pos] != ']' && p.src[p.pos] != ',' && p.src[p.pos] != '\n' {
		p.pos++
	}
	return ast.Literal(strings.TrimSpace(string(p.src[start:p.pos])))
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
