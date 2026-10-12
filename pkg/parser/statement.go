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
	case lexer.MAP:
		return p.parseMapStatement()
	case lexer.MOD:
		return p.parseModStatement()
	case lexer.FROM:
		return p.parseFromUseStatement()
	case lexer.FOR:
		return p.parseForStatement()
	case lexer.IN:
		return p.parseInStatement()
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
		bodyStmt := p.parseStatement()
		if bodyStmt != nil {
			stmt.Body = append(stmt.Body, bodyStmt)
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
		bodyStmt := p.parseStatement()
		if bodyStmt != nil {
			stmt.Body = append(stmt.Body, bodyStmt)
		}
		p.nextToken()
	}

	if p.curTokenIs(lexer.EOF) {
		p.addError(pos, "unclosed 'in' block (missing '}')")
	}

	return stmt
}

