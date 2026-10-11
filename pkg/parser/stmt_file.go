package parser

import (
	"github.com/Nehonix-Team/xru/pkg/ast"
	"github.com/Nehonix-Team/xru/pkg/lexer"
)

// dispatch <pathExpr> [, <patternExpr>]
func (p *Parser) parseDispatchTreeStatement() *ast.DispatchTreeStatement {
	stmt := &ast.DispatchTreeStatement{
		Position: p.curToken.Pos,
		Pattern:  ".rules.xru",
		PassArgs: make(map[string]ast.Expression),
	}
	p.nextToken() // avance sur le premier argument
	stmt.PathExpr = p.parseExpression()
	if strLit, ok := stmt.PathExpr.(*ast.StringLiteral); ok {
		stmt.Path = strLit.Value
	}

	if p.peekTokenIs(lexer.COMMA) {
		p.nextToken() // saute ','
		p.nextToken() // avance sur l'argument pattern
		stmt.PatternExpr = p.parseExpression()
		if strLit, ok := stmt.PatternExpr.(*ast.StringLiteral); ok {
			stmt.Pattern = strLit.Value
		}
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
