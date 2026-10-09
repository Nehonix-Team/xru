package parser

import (
	"github.com/Nehonix-Team/xru/pkg/ast"
	"github.com/Nehonix-Team/xru/pkg/lexer"
)

// dispatch_tree "./src", ".rules.xru" { pass mode = mode }
func (p *Parser) parseDispatchTreeStatement() *ast.DispatchTreeStatement {
	stmt := &ast.DispatchTreeStatement{
		Position: p.curToken.Pos,
		Pattern:  ".rules.xru",
		PassArgs: make(map[string]ast.Expression),
	}
	if p.expectPeek(lexer.STRING) {
		stmt.Path = p.curToken.Literal
	}
	if p.peekTokenIs(lexer.COMMA) {
		p.nextToken() // saute ','
		if p.expectPeek(lexer.STRING) {
			stmt.Pattern = p.curToken.Literal
		}
	}

	if !p.expectPeek(lexer.LBRACE) {
		return nil
	}

	p.nextToken()
	for !p.curTokenIs(lexer.RBRACE) && !p.curTokenIs(lexer.EOF) {
		if p.curTokenIs(lexer.PASS) || p.curTokenIs(lexer.SET) {
			if p.expectPeek(lexer.IDENT) {
				key := p.curToken.Literal
				if p.expectPeek(lexer.ASSIGN) {
					p.nextToken() // saute '='
					valExpr := p.parseExpression()
					stmt.PassArgs[key] = valExpr
				}
			}
		}
		p.nextToken()
	}

	return stmt
}

// file "path" { ... }
func (p *Parser) parseFileStatement() *ast.FileStatement {
	stmt := &ast.FileStatement{Position: p.curToken.Pos}
	if p.expectPeek(lexer.STRING) {
		stmt.Path = p.curToken.Literal
	}

	if !p.expectPeek(lexer.LBRACE) {
		return nil
	}

	p.nextToken()
	for !p.curTokenIs(lexer.RBRACE) && !p.curTokenIs(lexer.EOF) {
		bodyStmt := p.parseStatement()
		if bodyStmt != nil {
			stmt.Body = append(stmt.Body, bodyStmt)
		}
		p.nextToken()
	}
	if p.curTokenIs(lexer.EOF) {
		p.addError(stmt.Position, "unclosed file block (missing '}')")
	}
	return stmt
}
