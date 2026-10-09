package parser

import (
	"fmt"

	"github.com/Nehonix-Team/xru/pkg/ast"
	"github.com/Nehonix-Team/xru/pkg/lexer"
)

// unit Name { ... }
func (p *Parser) parseUnitStatement() *ast.UnitStatement {
	stmt := &ast.UnitStatement{Position: p.curToken.Pos}
	if !p.expectPeek(lexer.IDENT) {
		return nil
	}
	stmt.Name = p.curToken.Literal

	if !p.expectPeek(lexer.LBRACE) {
		return nil
	}

	p.nextToken()
	for !p.curTokenIs(lexer.RBRACE) && !p.curTokenIs(lexer.EOF) {
		switch p.curToken.Type {
		case lexer.DECLARE:
			decl := p.parseDeclareStatement()
			if decl != nil {
				stmt.Properties = append(stmt.Properties, decl)
			}
		case lexer.DIRECTIVE:
			dir := p.parseDirectiveDefStatement()
			if dir != nil {
				stmt.Directives = append(stmt.Directives, dir)
			}
		case lexer.FUNC:
			fn := p.parseFuncStatement()
			if fn != nil {
				stmt.Methods = append(stmt.Methods, fn)
			}
		default:
			p.addError(p.curToken.Pos, fmt.Sprintf("unexpected statement inside unit '%s': %s", stmt.Name, p.curToken.Literal))
		}
		p.nextToken()
	}

	if p.curTokenIs(lexer.EOF) {
		p.addError(stmt.Position, fmt.Sprintf("unclosed unit '%s' (missing '}')", stmt.Name))
	}

	return stmt
}

// directive Name(params...) { ... }
func (p *Parser) parseDirectiveDefStatement() *ast.DirectiveDefStatement {
	stmt := &ast.DirectiveDefStatement{Position: p.curToken.Pos}
	if !p.expectPeek(lexer.IDENT) {
		return nil
	}
	stmt.Name = p.curToken.Literal
	stmt.Parameters = p.parseParameterList()

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
	return stmt
}
