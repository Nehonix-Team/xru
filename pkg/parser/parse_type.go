package parser

import (
	"strings"

	"github.com/Nehonix-Team/xru/pkg/lexer"
)

// parseTypeAnnotation parses full type signatures:
// - primitives: string, number, bool, any, void
// - custom types: TExample
// - literal types: "ready", 7
// - union types: string or number, "ready" or "busy"
// - generic & indexed types: list[string], @map[string], TExample["c"]
// - inline maps: @map{ silent?: bool, retries: number }
// - func signatures: func(id: string) -> void or number
func (p *Parser) parseTypeAnnotation() string {
	parseSingleType := func() string {
		if p.peekTokenIs(lexer.EOF) || p.peekTokenIs(lexer.ASSIGN) || p.peekTokenIs(lexer.LBRACE) || p.peekTokenIs(lexer.COMMA) || p.peekTokenIs(lexer.RPAREN) || p.peekTokenIs(lexer.SEMICOLON) {
			return ""
		}

		p.nextToken()
		var sb strings.Builder

		switch p.curToken.Type {
		case lexer.AT:
			sb.WriteString("@")
			if p.peekTokenIs(lexer.MAP) || (p.peekTokenIs(lexer.IDENT) && strings.ToLower(p.peekToken.Literal) == "map") {
				p.nextToken()
				sb.WriteString(p.curToken.Literal)
				if p.peekTokenIs(lexer.LBRACKET) {
					p.nextToken() // saute '['
					sb.WriteString("[")
					for !p.curTokenIs(lexer.RBRACKET) && !p.curTokenIs(lexer.EOF) {
						p.nextToken()
						if p.curTokenIs(lexer.RBRACKET) {
							break
						}
						sb.WriteString(p.curToken.Literal)
					}
					sb.WriteString("]")
				} else if p.peekTokenIs(lexer.LBRACE) {
					p.nextToken() // saute '{'
					sb.WriteString("{")
					braceCount := 1
					for braceCount > 0 && !p.curTokenIs(lexer.EOF) {
						p.nextToken()
						if p.curTokenIs(lexer.LBRACE) {
							braceCount++
						} else if p.curTokenIs(lexer.RBRACE) {
							braceCount--
						}
						if braceCount > 0 {
							sb.WriteString(" ")
							if p.curTokenIs(lexer.STRING) {
								sb.WriteString(`"` + p.curToken.Literal + `"`)
							} else {
								sb.WriteString(p.curToken.Literal)
							}
						}
					}
					sb.WriteString("}")
				}
			}
			return sb.String()

		case lexer.FUNC:
			sb.WriteString("func")
			if p.peekTokenIs(lexer.LPAREN) {
				p.nextToken()
				sb.WriteString("(")
				for !p.curTokenIs(lexer.RPAREN) && !p.curTokenIs(lexer.EOF) {
					p.nextToken()
					if p.curTokenIs(lexer.RPAREN) {
						break
					}
					sb.WriteString(p.curToken.Literal)
					if p.peekTokenIs(lexer.COLON) || p.peekTokenIs(lexer.COMMA) {
						sb.WriteString(" ")
					}
				}
				sb.WriteString(")")
			}
			if p.peekTokenIs(lexer.ARROW) {
				p.nextToken() // saute '->'
				sb.WriteString(" -> ")
				retType := p.parseTypeAnnotation()
				sb.WriteString(retType)
			}
			return sb.String()

		case lexer.STRING:
			return `"` + p.curToken.Literal + `"`

		case lexer.NUMBER, lexer.BOOL, lexer.IDENT, lexer.MAP:
			sb.WriteString(p.curToken.Literal)

			// Gérer indexation ou générique: TExample["c"] ou list[string]
			if p.peekTokenIs(lexer.LBRACKET) {
				p.nextToken() // saute '['
				sb.WriteString("[")
				for !p.curTokenIs(lexer.RBRACKET) && !p.curTokenIs(lexer.EOF) {
					p.nextToken()
					if p.curTokenIs(lexer.RBRACKET) {
						break
					}
					if p.curTokenIs(lexer.STRING) {
						sb.WriteString(`"` + p.curToken.Literal + `"`)
					} else {
						sb.WriteString(p.curToken.Literal)
					}
				}
				sb.WriteString("]")
			}
			if p.peekTokenIs(lexer.QUESTION) {
				p.nextToken() // saute '?'
				sb.WriteString("?")
			}
			return sb.String()

		default:
			res := p.curToken.Literal
			if p.peekTokenIs(lexer.QUESTION) {
				p.nextToken()
				res += "?"
			}
			return res
		}
	}

	first := parseSingleType()
	if first == "" {
		return ""
	}
	if p.peekTokenIs(lexer.QUESTION) {
		p.nextToken()
		if !strings.HasSuffix(first, "?") {
			first = first + "?"
		}
	}

	parts := []string{first}
	for p.peekTokenIs(lexer.OR) {
		p.nextToken() // saute 'or'
		next := parseSingleType()
		if next != "" {
			if p.peekTokenIs(lexer.QUESTION) {
				p.nextToken()
				if !strings.HasSuffix(next, "?") {
					next = next + "?"
				}
			}
			parts = append(parts, next)
		}
	}
	res := strings.Join(parts, " or ")
	if p.peekTokenIs(lexer.QUESTION) {
		p.nextToken()
		if !strings.HasSuffix(res, "?") {
			res = res + "?"
		}
	}
	return res
}
