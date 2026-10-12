package parser

import (
	"fmt"

	"github.com/Nehonix-Team/xru/pkg/ast"
	"github.com/Nehonix-Team/xru/pkg/lexer"
)

// func Name(params...) -> ReturnType { ... }
func (p *Parser) parseFuncStatement() *ast.FuncStatement {
	stmt := &ast.FuncStatement{Position: p.curToken.Pos}
	if !p.expectPeek(lexer.IDENT) {
		return nil
	}
	stmt.Name = p.curToken.Literal
	stmt.Parameters = p.parseParameterList()

	if p.peekTokenIs(lexer.ARROW) {
		p.nextToken() // saute '->'
		if !p.expectPeek(lexer.IDENT) {
			return nil
		}
		stmt.ReturnType = p.curToken.Literal
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
		p.addError(stmt.Position, fmt.Sprintf("unclosed func '%s' (missing '}')", stmt.Name))
	}
	return stmt
}

func (p *Parser) parseParameterList() []*ast.Parameter {
	var params []*ast.Parameter
	if !p.expectPeek(lexer.LPAREN) {
		return params
	}

	if p.peekTokenIs(lexer.RPAREN) {
		p.nextToken()
		return params
	}

	p.nextToken()
	for {
		if !p.curTokenIs(lexer.IDENT) {
			break
		}
		param := &ast.Parameter{Name: p.curToken.Literal}
		if p.peekTokenIs(lexer.COLON) {
			p.nextToken() // saute ':'
			if p.expectPeek(lexer.IDENT) {
				param.Type = p.curToken.Literal
			}
		}
		params = append(params, param)

		if p.peekTokenIs(lexer.COMMA) {
			p.nextToken()
			p.nextToken()
			continue
		}
		break
	}

	p.expectPeek(lexer.RPAREN)
	return params
}

func (p *Parser) parseReturnStatement() *ast.ReturnStatement {
	stmt := &ast.ReturnStatement{Position: p.curToken.Pos}
	if !p.peekTokenIs(lexer.RBRACE) && !p.peekTokenIs(lexer.EOF) {
		p.nextToken()
		stmt.Value = p.parseExpression()
	}
	return stmt
}

func (p *Parser) parseGlobalStatement() ast.Statement {
	p.nextToken() // saute 'global'
	if p.curTokenIs(lexer.FUNC) {
		fnStmt := p.parseFuncStatement()
		if fnStmt != nil {
			fnStmt.IsGlobal = true
			return fnStmt
		}
		return nil
	}
	if p.curTokenIs(lexer.LET) || p.curTokenIs(lexer.CONST) {
		varStmt := p.parseVarStatement()
		if varStmt != nil {
			varStmt.IsGlobal = true
			return varStmt
		}
		return nil
	}
	if p.curTokenIs(lexer.ARG) {
		argStmt := p.parseArgStatement()
		if argStmt != nil {
			argStmt.IsGlobal = true
			return argStmt
		}
		return nil
	}
	stmt := p.parseStatement()
	return stmt
}
