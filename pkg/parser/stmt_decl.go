package parser

import (
	"github.com/Nehonix-Team/xru/pkg/ast"
	"github.com/Nehonix-Team/xru/pkg/lexer"
)

// use module as Alias
func (p *Parser) parseUseStatement() *ast.UseStatement {
	stmt := &ast.UseStatement{Position: p.curToken.Pos}
	if !p.expectPeek(lexer.IDENT) {
		return nil
	}
	stmt.Module = p.curToken.Literal

	if p.peekTokenIs(lexer.AS) {
		p.nextToken() // mange 'as'
		if !p.expectPeek(lexer.IDENT) {
			return nil
		}
		stmt.Alias = p.curToken.Literal
	}
	return stmt
}

// arg name: type = default  /  arg name: type or default
func (p *Parser) parseArgStatement() *ast.ArgStatement {
	stmt := &ast.ArgStatement{Position: p.curToken.Pos}
	if !p.expectPeek(lexer.IDENT) {
		return nil
	}
	stmt.Name = p.curToken.Literal

	if p.peekTokenIs(lexer.COLON) {
		p.nextToken() // mange ':'
		if !p.expectPeek(lexer.IDENT) {
			return nil
		}
		stmt.Type = p.curToken.Literal
	}

	if p.peekTokenIs(lexer.OR) || p.peekTokenIs(lexer.ASSIGN) {
		p.nextToken() // mange 'or' ou '='
		p.nextToken() // début de l'expression par défaut
		stmt.DefaultValue = p.parseExpression()
	}
	return stmt
}

// let/const name: type = value
func (p *Parser) parseVarStatement() *ast.VarStatement {
	stmt := &ast.VarStatement{
		Position: p.curToken.Pos,
		IsConst:  p.curToken.Type == lexer.CONST,
	}

	if !p.expectPeek(lexer.IDENT) {
		return nil
	}
	stmt.Name = p.curToken.Literal

	if p.peekTokenIs(lexer.COLON) {
		p.nextToken() // mange ':'
		if !p.expectPeek(lexer.IDENT) {
			return nil
		}
		typeStr := p.curToken.Literal
		if p.peekTokenIs(lexer.LBRACKET) {
			typeStr += "["
			p.nextToken() // saute '['
			for !p.curTokenIs(lexer.RBRACKET) && !p.curTokenIs(lexer.EOF) {
				p.nextToken()
				if p.curTokenIs(lexer.RBRACKET) {
					break
				}
				typeStr += p.curToken.Literal
			}
			typeStr += "]"
		}
		stmt.Type = typeStr
	}

	if !p.expectPeek(lexer.ASSIGN) {
		return nil
	}

	p.nextToken() // début de la valeur
	stmt.Value = p.parseExpression()
	return stmt
}

// declare prop: type = value
func (p *Parser) parseDeclareStatement() *ast.DeclareStatement {
	stmt := &ast.DeclareStatement{Position: p.curToken.Pos}
	if !p.expectPeek(lexer.IDENT) {
		return nil
	}
	stmt.Name = p.curToken.Literal

	if p.peekTokenIs(lexer.COLON) {
		p.nextToken() // saute ':'
		if !p.expectPeek(lexer.IDENT) {
			return nil
		}
		stmt.Type = p.curToken.Literal
	}

	if p.peekTokenIs(lexer.ASSIGN) {
		p.nextToken() // saute '='
		p.nextToken()
		stmt.Value = p.parseExpression()
	}
	return stmt
}
