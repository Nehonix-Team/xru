package parser

import (
	"fmt"

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

// parseFileStatement gère :
// 1. Moderne : file <pathExpr>, (doc, err) -> { ... }
// 2. Moderne sans virgule : file <pathExpr> (doc, err) -> { ... }
// 3. Rétrocompatible : file <pathExpr> { ... }
func (p *Parser) parseFileStatement() *ast.FileStatement {
	stmt := &ast.FileStatement{Position: p.curToken.Pos}

	// Avance sur l'expression de chemin
	p.nextToken()
	stmt.PathExpr = p.parseExpression()
	if strLit, ok := stmt.PathExpr.(*ast.StringLiteral); ok {
		stmt.Path = strLit.Value
	}

	// Cas 1 : virgule suivie du Handler lambda
	if p.peekTokenIs(lexer.COMMA) {
		p.nextToken() // saute ','
		p.nextToken() // premier token de la lambda
		stmt.Handler = p.parseExpression()
		return stmt
	}

	// Cas 2 : Handler lambda direct sans virgule
	if p.peekTokenIs(lexer.LPAREN) {
		p.nextToken() // '('
		stmt.Handler = p.parseParenOrLambdaExpression()
		return stmt
	}

	// Cas 3 : Ancien bloc direct { ... } (rétrocompatibilité)
	if p.peekTokenIs(lexer.LBRACE) {
		p.nextToken() // '{'
		p.nextToken() // premier token du bloc
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

	p.addError(stmt.Position, fmt.Sprintf("expected ',' with handler lambda or '{' block after file path, got '%s'", p.peekToken.Literal))
	return stmt
}
