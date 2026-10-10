package parser

import (
	"fmt"

	"github.com/Nehonix-Team/xru/pkg/ast"
	"github.com/Nehonix-Team/xru/pkg/lexer"
)

func (p *Parser) parseStatement() ast.Statement {
	switch p.curToken.Type {
	case lexer.LBRACE:
		return p.parseBlockStatement()
	case lexer.USE:
		return p.parseUseStatement()
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
	case lexer.DISPATCH_TREE:
		return p.parseDispatchTreeStatement()
	case lexer.FILE:
		return p.parseFileStatement()
	case lexer.GLOBAL:
		return p.parseGlobalStatement()
	case lexer.SET, lexer.HEADER, lexer.ON:
		return p.parseCustomDirective()
	case lexer.RETURN:
		return p.parseReturnStatement()
	case lexer.MAP:
		return p.parseMapStatement()
	case lexer.MOD:
		return p.parseModStatement()
	case lexer.FROM:
		return p.parseFromUseStatement()
	case lexer.ILLEGAL:
		p.addError(p.curToken.Pos, fmt.Sprintf("illegal character '%s'", p.curToken.Literal))
		return nil
	case lexer.AMP:
		if p.peekTokenIs(lexer.IDENT) {
			pos := p.curToken.Pos
			p.nextToken() // saute '&', curToken est l'identifiant
			propName := p.curToken.Literal
			if p.peekTokenIs(lexer.MUT_ASSIGN) {
				p.nextToken() // saute l'identifiant, curToken est '=:'
				p.nextToken() // saute '=:', curToken est l'expression
				val := p.parseExpression()
				return &ast.ReassignStatement{
					Position: pos,
					Name:     "&" + propName,
					Value:    val,
				}
			}
			return &ast.ExpressionStatement{
				Position: pos,
				Expression: &ast.PropAccess{
					Position: pos,
					Name:     propName,
				},
			}
		}
		return p.parseExpressionStatement()
	case lexer.IDENT:
		if p.peekTokenIs(lexer.MUT_ASSIGN) {
			return p.parseReassignStatement()
		}
		if p.peekTokenIs(lexer.ASSIGN) {
			p.addError(p.peekToken.Pos, fmt.Sprintf("cannot assign with '=': use '=:' to mutate variable '%s' (e.g. %s =: <value>)", p.curToken.Literal, p.curToken.Literal))
			return nil
		}
		if p.peekTokenIs(lexer.IDENT) || p.peekTokenIs(lexer.STRING) {
			return p.parseCustomDirective()
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
	p.nextToken() // saute '=:'
	stmt.Value = p.parseExpression()
	return stmt
}

// { stmt1; stmt2; ... }
func (p *Parser) parseBlockStatement() *ast.BlockStatement {
	block := &ast.BlockStatement{Position: p.curToken.Pos}
	p.nextToken() // saute '{'
	for !p.curTokenIs(lexer.RBRACE) && !p.curTokenIs(lexer.EOF) {
		stmt := p.parseStatement()
		if stmt != nil {
			block.Statements = append(block.Statements, stmt)
		}
		p.nextToken()
	}
	return block
}

func (p *Parser) parseExpressionStatement() *ast.ExpressionStatement {
	stmt := &ast.ExpressionStatement{
		Position:   p.curToken.Pos,
		Expression: p.parseExpression(),
	}
	return stmt
}
