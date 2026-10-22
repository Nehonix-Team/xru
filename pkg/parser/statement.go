package parser

import (
	"fmt"
	"strings"

	"github.com/Nehonix-Team/xru/pkg/ast"
	"github.com/Nehonix-Team/xru/pkg/lexer"
)

func (p *Parser) parseStatement() ast.Statement {
	switch p.curToken.Type {
	case lexer.LBRACE:
		return p.parseBlockStatement()
	case lexer.USE:
		return p.parseUseStatement()
	case lexer.TRAP:
		return p.parseTrapStatement()
	case lexer.ARG:
		return p.parseArgStatement()
	case lexer.LET, lexer.CONST:
		return p.parseVarStatement()
	case lexer.UNIT:
		return p.parseUnitStatement()
	case lexer.FUNC:
		return p.parseFuncStatement()
	case lexer.WITH:
		return p.parseWithStatement()
	case lexer.IF:
		return p.parseIfStatement()
	case lexer.MATCH:
		return p.parseMatchStatement()
	case lexer.DISPATCH, lexer.DISPATCH_TREE:
		return p.parseDispatchTreeStatement()
	case lexer.FILE:
		return p.parseFileStatement()
	case lexer.GLOBAL:
		return p.parseGlobalStatement()
	case lexer.SET, lexer.HEADER, lexer.ON:
		p.addError(p.curToken.Pos, fmt.Sprintf("magic directive '%s' is deprecated and unsupported in XRU 2.0: use standard async task syntax", p.curToken.Literal))
		return nil
	case lexer.RETURN:
		return p.parseReturnStatement()
	case lexer.INTERFACE, lexer.TYPE:
		return p.parseInterfaceStatement()
	case lexer.MAP:
		return p.parseMapStatement()
	case lexer.MOD:
		return p.parseModStatement()
	case lexer.FROM:
		return p.parseFromUseStatement()
	case lexer.FOR:
		return p.parseForStatement()
	case lexer.WHILE:
		return p.parseWhileStatement()
	case lexer.BREAK:
		return p.parseBreakStatement()
	case lexer.CONTINUE:
		return p.parseContinueStatement()
	case lexer.IN:
		return p.parseInStatement()
	case lexer.SEMICOLON:
		return nil
	case lexer.ILLEGAL:
		p.addError(p.curToken.Pos, fmt.Sprintf("illegal character '%s'", p.curToken.Literal))
		return nil
	case lexer.AT:
		if p.peekTokenIs(lexer.IDENT) && p.peekToken.Literal == "native" {
			hookName := p.parseNativeAnnotation()
			isGlobal := false
			if p.peekTokenIs(lexer.GLOBAL) {
				isGlobal = true
				p.nextToken()
			}
			if !p.expectPeek(lexer.FUNC) {
				return nil
			}
			p.currentNativeHook = hookName
			fn := p.parseFuncStatement()
			if fn != nil {
				fn.IsGlobal = isGlobal
				return fn
			}
			return nil
		}
		return p.parseExpressionStatement()
	case lexer.AMP:
		if p.peekTokenIs(lexer.IDENT) {
			pos := p.curToken.Pos
			p.nextToken() // saute '&', curToken est l'identifiant
			name := p.curToken.Literal
			if p.peekTokenIs(lexer.DOUBLE_COLON) {
				p.nextToken() // curToken est '::'
				if !p.expectPeek(lexer.IDENT) {
					p.addError(pos, "expected identifier after '::'")
					return nil
				}
				member := p.curToken.Literal
				scopeExpr := &ast.ScopeAccessExpression{
					Position:   pos,
					Scope:      name,
					Member:     member,
					IsUnitSelf: true,
				}
				var expr ast.Expression = scopeExpr
				if p.peekTokenIs(lexer.LPAREN) {
					p.nextToken()
					args := p.parseExpressionList(lexer.RPAREN)
					expr = &ast.CallExpression{
						Position:  pos,
						Caller:    scopeExpr,
						Arguments: args,
					}
				}
				expr = p.parseExpressionWithPrimary(expr, LOWEST)
				stmt := &ast.ExpressionStatement{
					Position:   pos,
					Expression: expr,
				}
				if p.peekTokenIs(lexer.SEMICOLON) {
					p.nextToken()
				}
				return stmt
			}
			if p.peekTokenIs(lexer.MUT_ASSIGN) {
				p.nextToken() // saute l'identifiant, curToken est '=:'
				p.nextToken() // saute '=:', curToken est l'expression
				val := p.parseExpression()
				if p.peekTokenIs(lexer.SEMICOLON) {
					p.nextToken()
				}
				return &ast.ReassignStatement{
					Position: pos,
					Name:     "&" + name,
					Value:    val,
				}
			}
			prop := &ast.PropAccess{
				Position: pos,
				Name:     name,
			}
			var expr ast.Expression = prop
			if p.peekTokenIs(lexer.LPAREN) {
				p.nextToken()
				args := p.parseExpressionList(lexer.RPAREN)
				expr = &ast.CallExpression{
					Position:  pos,
					Caller:    prop,
					Arguments: args,
				}
			}
			expr = p.parseExpressionWithPrimary(expr, LOWEST)
			stmt := &ast.ExpressionStatement{
				Position:   pos,
				Expression: expr,
			}
			if p.peekTokenIs(lexer.SEMICOLON) {
				p.nextToken()
			}
			return stmt
		}
		return p.parseExpressionStatement()
	case lexer.IDENT:
		if p.isStdInStatement() {
			return p.parseStdInStatement()
		}
		if p.peekTokenIs(lexer.MUT_ASSIGN) {
			return p.parseReassignStatement()
		}
		if p.peekTokenIs(lexer.ASSIGN) {
			p.addError(p.peekToken.Pos, fmt.Sprintf("cannot assign with '=': use '=:' to mutate variable '%s' (e.g. %s =: <value>)", p.curToken.Literal, p.curToken.Literal))
			return nil
		}
		return p.parseExpressionStatement()
	default:
		// Expression libre (ex: appel de fonction ou méthode)
		return p.parseExpressionStatement()
	}
}

// ident =: expr
func (p *Parser) parseReassignStatement() *ast.ReassignStatement {
	stmt := &ast.ReassignStatement{
		Position: p.curToken.Pos,
		Name:     p.curToken.Literal,
	}
	if !p.expectPeek(lexer.MUT_ASSIGN) {
		return nil
	}
	assignPos := p.curToken.Pos
	if p.peekToken.Pos.Line != assignPos.Line || p.peekTokenIs(lexer.EOF) || p.peekTokenIs(lexer.RBRACE) {
		p.addError(assignPos, "expected expression after '=:' in reassignment")
		return stmt
	}
	p.nextToken() // saute '=:'
	stmt.Value = p.parseExpression()
	if stmt.Value == nil {
		p.addError(assignPos, "expected expression after '=:' in reassignment")
		return stmt
	}
	if p.peekTokenIs(lexer.SEMICOLON) {
		p.nextToken()
	}
	return stmt
}

// { stmt1; stmt2; ... }
func (p *Parser) parseBlockStatement() *ast.BlockStatement {
	block := &ast.BlockStatement{Position: p.curToken.Pos}
	p.nextToken() // saute '{'
	for !p.curTokenIs(lexer.RBRACE) && !p.curTokenIs(lexer.EOF) {
		if p.curTokenIs(lexer.SEMICOLON) {
			p.nextToken()
			continue
		}
		stmt := p.parseStatement()
		if stmt != nil {
			block.Statements = append(block.Statements, stmt)
		}
		if p.peekTokenIs(lexer.SEMICOLON) {
			p.nextToken()
		}
		p.nextToken()
	}
	if p.curTokenIs(lexer.EOF) {
		p.addError(block.Position, "unclosed block (missing '}')")
	}
	return block
}

func (p *Parser) parseExpressionStatement() *ast.ExpressionStatement {
	stmt := &ast.ExpressionStatement{
		Position:   p.curToken.Pos,
		Expression: p.parseExpression(),
	}
	if p.peekTokenIs(lexer.SEMICOLON) {
		p.nextToken()
	}
	return stmt
}

// for item [, index] in iterable { ... }
func (p *Parser) parseForStatement() *ast.ForStatement {
	pos := p.curToken.Pos
	if !p.expectPeek(lexer.IDENT) {
		return nil
	}
	item := p.curToken.Literal

	index := ""
	if p.peekTokenIs(lexer.COMMA) {
		p.nextToken() // saute ','
		if !p.expectPeek(lexer.IDENT) {
			return nil
		}
		index = p.curToken.Literal
	}

	if !p.expectPeek(lexer.IN) {
		p.addError(p.curToken.Pos, "expected 'in' after variable in for loop")
		return nil
	}

	p.nextToken() // saute 'in'
	iterable := p.parseExpression()
	if iterable == nil {
		return nil
	}

	if !p.expectPeek(lexer.LBRACE) {
		return nil
	}

	stmt := &ast.ForStatement{
		Position: pos,
		Item:     item,
		Index:    index,
		Iterable: iterable,
	}

	p.nextToken() // saute '{'
	for !p.curTokenIs(lexer.RBRACE) && !p.curTokenIs(lexer.EOF) {
		if p.curTokenIs(lexer.SEMICOLON) {
			p.nextToken()
			continue
		}
		bodyStmt := p.parseStatement()
		if bodyStmt != nil {
			stmt.Body = append(stmt.Body, bodyStmt)
		}
		if p.peekTokenIs(lexer.SEMICOLON) {
			p.nextToken()
		}
		p.nextToken()
	}

	if p.curTokenIs(lexer.EOF) {
		p.addError(pos, "unclosed for loop block (missing '}')")
	}

	return stmt
}

// in <path> { ... }
func (p *Parser) parseInStatement() *ast.InStatement {
	pos := p.curToken.Pos

	// saute 'in'
	p.nextToken()

	pathExpr := p.parseExpression()
	if pathExpr == nil {
		p.addError(pos, "expected path expression after 'in'")
		return nil
	}

	if !p.expectPeek(lexer.LBRACE) {
		p.addError(p.curToken.Pos, "expected '{' after in <path>")
		return nil
	}

	stmt := &ast.InStatement{
		Position: pos,
		Path:     pathExpr,
	}

	p.nextToken() // saute '{'
	for !p.curTokenIs(lexer.RBRACE) && !p.curTokenIs(lexer.EOF) {
		if p.curTokenIs(lexer.SEMICOLON) {
			p.nextToken()
			continue
		}
		bodyStmt := p.parseStatement()
		if bodyStmt != nil {
			stmt.Body = append(stmt.Body, bodyStmt)
		}
		if p.peekTokenIs(lexer.SEMICOLON) {
			p.nextToken()
		}
		p.nextToken()
	}

	if p.curTokenIs(lexer.EOF) {
		p.addError(pos, "unclosed 'in' block (missing '}')")
	}

	return stmt
}

func (p *Parser) parseTrapStatement() *ast.TrapStatement {
	stmt := &ast.TrapStatement{Position: p.curToken.Pos}
	p.nextToken() // consomme 'trap'

	// Cas 1: trap sig::@map{...} ou trap sig::interrupt ou trap sig.interrupt
	if p.curTokenIs(lexer.IDENT) && (p.peekTokenIs(lexer.DOUBLE_COLON) || p.peekTokenIs(lexer.DOT)) {
		stmt.Scope = p.curToken.Literal
		p.nextToken() // saute l'identifiant scope (ex: "sig")
		p.nextToken() // saute '::' ou '.'

		// Est-ce un groupement @map{...} ?
		if p.curTokenIs(lexer.AT) {
			p.nextToken() // saute '@'
			if p.curTokenIs(lexer.MAP) || (p.curTokenIs(lexer.IDENT) && strings.ToLower(p.curToken.Literal) == "map") {
				p.nextToken() // saute 'map'
			}
			if p.curTokenIs(lexer.LBRACE) {
				p.nextToken() // saute '{'
				for !p.curTokenIs(lexer.RBRACE) && !p.curTokenIs(lexer.EOF) {
					if p.curTokenIs(lexer.COMMA) || p.curTokenIs(lexer.SEMICOLON) {
						p.nextToken()
						continue
					}
					if p.curTokenIs(lexer.IDENT) {
						stmt.Targets = append(stmt.Targets, p.curToken.Literal)
					}
					p.nextToken()
				}
				if p.curTokenIs(lexer.RBRACE) {
					p.nextToken() // saute '}'
				}
			}
		} else if p.curTokenIs(lexer.IDENT) {
			stmt.Targets = append(stmt.Targets, p.curToken.Literal)
			p.nextToken()
		}
	} else if p.curTokenIs(lexer.LPAREN) || p.curTokenIs(lexer.LBRACKET) {
		closeTok := lexer.RPAREN
		if p.curTokenIs(lexer.LBRACKET) {
			closeTok = lexer.RBRACKET
		}
		p.nextToken() // saute '(' ou '['
		for !p.curTokenIs(closeTok) && !p.curTokenIs(lexer.EOF) {
			if p.curTokenIs(lexer.COMMA) || p.curTokenIs(lexer.SEMICOLON) {
				p.nextToken()
				continue
			}
			if p.curTokenIs(lexer.IDENT) {
				ident := p.curToken.Literal
				if p.peekTokenIs(lexer.DOUBLE_COLON) || p.peekTokenIs(lexer.DOT) {
					p.nextToken() // saute scope
					p.nextToken() // saute '::' ou '.'
					if p.curTokenIs(lexer.IDENT) {
						stmt.Targets = append(stmt.Targets, p.curToken.Literal)
					}
				} else {
					stmt.Targets = append(stmt.Targets, ident)
				}
			}
			p.nextToken()
		}
		if p.curTokenIs(closeTok) {
			p.nextToken() // saute ')' ou ']'
		}
	} else if p.curTokenIs(lexer.IDENT) || (p.curToken.Type != lexer.EOF && p.curToken.Type != lexer.LBRACE && p.curToken.Type != lexer.AS) {
		stmt.Targets = append(stmt.Targets, p.curToken.Literal)
		p.nextToken()
	}

	// Optionnel : as <alias>
	if p.curTokenIs(lexer.AS) {
		p.nextToken() // consomme 'as'
		if p.curTokenIs(lexer.IDENT) {
			stmt.Alias = p.curToken.Literal
			p.nextToken()
		}
	}

	if !p.curTokenIs(lexer.LBRACE) {
		p.addError(p.curToken.Pos, "expected '{' after trap declaration")
		return nil
	}

	stmt.Body = p.parseBlockStatement()
	return stmt
}

func (p *Parser) isStdInStatement() bool {
	if !p.curTokenIs(lexer.IDENT) {
		return false
	}
	if !p.peekTokenIs(lexer.DOUBLE_COLON) && !p.peekTokenIs(lexer.DOT) {
		return false
	}
	lexerClone := p.l.Clone()
	tok1 := lexerClone.NextToken()
	tok2 := lexerClone.NextToken()
	return tok1.Type == lexer.IN && tok2.Type == lexer.LBRACE
}

func (p *Parser) parseStdInStatement() *ast.StdInStatement {
	pos := p.curToken.Pos
	scope := p.curToken.Literal
	p.nextToken() // saute "std" -> curToken est "::" ou "."
	p.nextToken() // saute "::" ou "." -> curToken est "in"
	p.nextToken() // saute "in" -> curToken est "{"
	block := p.parseBlockStatement()
	return &ast.StdInStatement{
		Position: pos,
		Scope:    scope,
		Body:     block,
	}
}

func (p *Parser) parseBreakStatement() *ast.BreakStatement {
	stmt := &ast.BreakStatement{Position: p.curToken.Pos}
	if p.peekTokenIs(lexer.SEMICOLON) {
		p.nextToken()
	}
	return stmt
}

func (p *Parser) parseContinueStatement() *ast.ContinueStatement {
	stmt := &ast.ContinueStatement{Position: p.curToken.Pos}
	if p.peekTokenIs(lexer.SEMICOLON) {
		p.nextToken()
	}
	return stmt
}

