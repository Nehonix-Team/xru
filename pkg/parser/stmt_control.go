package parser

import (
	"github.com/Nehonix-Team/xru/pkg/ast"
	"github.com/Nehonix-Team/xru/pkg/lexer"
)

// with Target { ... } ou with Target (asynchrone)
func (p *Parser) parseWithStatement() ast.Statement {
	pos := p.curToken.Pos
	p.nextToken()

	// Cas with { ... } (bloc concurrent sans cible explicite)
	if p.curTokenIs(lexer.LBRACE) {
		stmt := &ast.WithStatement{Position: pos}
		p.nextToken()
		for !p.curTokenIs(lexer.RBRACE) && !p.curTokenIs(lexer.EOF) {
			bodyStmt := p.parseStatement()
			if bodyStmt != nil {
				stmt.Body = append(stmt.Body, bodyStmt)
			}
			p.nextToken()
		}
		if p.curTokenIs(lexer.EOF) {
			p.addError(stmt.Position, "unclosed with block (missing '}')")
		}
		return stmt
	}

	target := p.parseExpression()
	if target == nil {
		p.addError(pos, "expected expression after 'with'")
		return nil
	}

	// Si suivi d'un bloc '{' : bloc with avec cible
	if p.peekTokenIs(lexer.LBRACE) {
		stmt := &ast.WithStatement{Position: pos, Target: target}
		p.nextToken() // saute sur '{'
		p.nextToken()
		for !p.curTokenIs(lexer.RBRACE) && !p.curTokenIs(lexer.EOF) {
			bodyStmt := p.parseStatement()
			if bodyStmt != nil {
				stmt.Body = append(stmt.Body, bodyStmt)
			}
			p.nextToken()
		}
		if p.curTokenIs(lexer.EOF) {
			p.addError(stmt.Position, "unclosed with block (missing '}')")
		}
		return stmt
	}

	// Sinon : expression de tâche asynchrone autonome (ex: with Net.fetch("..."))
	return &ast.ExpressionStatement{
		Position: pos,
		Expression: &ast.WithExpression{
			Position: pos,
			Target:   target,
		},
	}
}

// Directive invocation (ex: set method = "GET", header "Accept" -> "...", on progress -> ...)
func (p *Parser) parseCustomDirective() *ast.CustomDirectiveInvocation {
	stmt := &ast.CustomDirectiveInvocation{
		Position:  p.curToken.Pos,
		Directive: p.curToken.Literal,
	}

	p.nextToken()
	stmt.Key = p.parseExpression()

	if p.peekTokenIs(lexer.ARROW) || p.peekTokenIs(lexer.ASSIGN) {
		p.nextToken() // saute '->' ou '='
		p.nextToken()
		stmt.Value = p.parseExpression()
	}
	return stmt
}

// if condition { ... } elif condition { ... } else { ... }
// ou inline: if condition -> consequence else alternative
func (p *Parser) parseIfStatement() ast.Statement {
	startPos := p.curToken.Pos
	p.nextToken()
	condition := p.parseExpression()
	if condition == nil {
		return nil
	}

	// Si forme inline: if cond -> ... else ...
	if p.peekTokenIs(lexer.ARROW) {
		p.nextToken() // curToken devient '->'
		p.nextToken() // début de consequence
		consequence := p.parseExpression()
		if consequence == nil {
			return nil
		}
		if !p.expectPeek(lexer.ELSE) {
			return nil
		}
		p.nextToken() // saute 'else'
		alternative := p.parseExpression()
		if alternative == nil {
			return nil
		}
		return &ast.ExpressionStatement{
			Position: startPos,
			Expression: &ast.IfExpression{
				Position:    startPos,
				Condition:   condition,
				Consequence: consequence,
				Alternative: alternative,
			},
		}
	}

	stmt := &ast.IfStatement{Position: startPos, Condition: condition}

	if !p.expectPeek(lexer.LBRACE) {
		return nil
	}

	p.nextToken()
	for !p.curTokenIs(lexer.RBRACE) && !p.curTokenIs(lexer.EOF) {
		bodyStmt := p.parseStatement()
		if bodyStmt != nil {
			stmt.Consequence = append(stmt.Consequence, bodyStmt)
		}
		p.nextToken()
	}

	for p.peekTokenIs(lexer.ELIF) {
		p.nextToken() // saute 'elif'
		elif := &ast.ElifStatement{}
		p.nextToken()
		elif.Condition = p.parseExpression()
		if !p.expectPeek(lexer.LBRACE) {
			return nil
		}
		p.nextToken()
		for !p.curTokenIs(lexer.RBRACE) && !p.curTokenIs(lexer.EOF) {
			bodyStmt := p.parseStatement()
			if bodyStmt != nil {
				elif.Body = append(elif.Body, bodyStmt)
			}
			p.nextToken()
		}
		stmt.ElifBlocks = append(stmt.ElifBlocks, elif)
	}

	if p.peekTokenIs(lexer.ELSE) {
		p.nextToken() // saute 'else'
		if !p.expectPeek(lexer.LBRACE) {
			return nil
		}
		p.nextToken()
		for !p.curTokenIs(lexer.RBRACE) && !p.curTokenIs(lexer.EOF) {
			bodyStmt := p.parseStatement()
			if bodyStmt != nil {
				stmt.Alternative = append(stmt.Alternative, bodyStmt)
			}
			p.nextToken()
		}
	}

	return stmt
}

// match target { case pattern { ... } default { ... } }
func (p *Parser) parseMatchStatement() *ast.MatchStatement {
	stmt := &ast.MatchStatement{Position: p.curToken.Pos}
	p.nextToken()
	stmt.Target = p.parseExpression()

	if !p.expectPeek(lexer.LBRACE) {
		return nil
	}

	p.nextToken()
	for !p.curTokenIs(lexer.RBRACE) && !p.curTokenIs(lexer.EOF) {
		if p.curTokenIs(lexer.CASE) {
			kase := &ast.MatchCase{}
			p.nextToken()
			kase.Pattern = p.parseExpression()

			if !p.expectPeek(lexer.LBRACE) {
				return nil
			}

			p.nextToken()
			for !p.curTokenIs(lexer.RBRACE) && !p.curTokenIs(lexer.EOF) {
				bodyStmt := p.parseStatement()
				if bodyStmt != nil {
					kase.Body = append(kase.Body, bodyStmt)
				}
				p.nextToken()
			}
			stmt.Cases = append(stmt.Cases, kase)
			p.nextToken()
			continue
		}

		if p.curTokenIs(lexer.DEFAULT) {
			if !p.expectPeek(lexer.LBRACE) {
				return nil
			}
			p.nextToken()
			for !p.curTokenIs(lexer.RBRACE) && !p.curTokenIs(lexer.EOF) {
				bodyStmt := p.parseStatement()
				if bodyStmt != nil {
					stmt.DefaultBlock = append(stmt.DefaultBlock, bodyStmt)
				}
				p.nextToken()
			}
			p.nextToken()
			break
		}
		p.nextToken()
	}

	if p.curTokenIs(lexer.EOF) {
		p.addError(stmt.Position, "unclosed match block (missing '}')")
	}
	return stmt
}
