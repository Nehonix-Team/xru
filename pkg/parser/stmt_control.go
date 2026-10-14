package parser

import (
	"github.com/Nehonix-Team/xru/pkg/ast"
	"github.com/Nehonix-Team/xru/pkg/lexer"
)

// with Target (asynchrone)
func (p *Parser) parseWithStatement() ast.Statement {
	pos := p.curToken.Pos
	p.nextToken()

	if p.curTokenIs(lexer.LBRACE) {
		p.addError(pos, "'with' operator does not accept a block: use 'with <expr>' to spawn an async task (see syntax_specification.md)")
		return nil
	}

	target := p.parseExpression()
	if target == nil {
		p.addError(pos, "expected expression after 'with'")
		return nil
	}

	// XRU 2.0: 'with' est un opérateur asynchrone strict qui ne prend plus de bloc magique
	if p.peekTokenIs(lexer.LBRACE) {
		p.addError(p.peekToken.Pos, "'with' operator does not accept a block: use 'with <expr>' to spawn an async task (see syntax_specification.md)")
		return nil
	}

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
		if p.curTokenIs(lexer.SEMICOLON) {
			p.nextToken()
			continue
		}
		bodyStmt := p.parseStatement()
		if bodyStmt != nil {
			stmt.Consequence = append(stmt.Consequence, bodyStmt)
		}
		if p.peekTokenIs(lexer.SEMICOLON) {
			p.nextToken()
		}
		p.nextToken()
	}
	if p.curTokenIs(lexer.EOF) {
		p.addError(stmt.Position, "unclosed block (missing '}')")
		return nil
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
			if p.curTokenIs(lexer.SEMICOLON) {
				p.nextToken()
				continue
			}
			bodyStmt := p.parseStatement()
			if bodyStmt != nil {
				elif.Body = append(elif.Body, bodyStmt)
			}
			if p.peekTokenIs(lexer.SEMICOLON) {
				p.nextToken()
			}
			p.nextToken()
		}
		if p.curTokenIs(lexer.EOF) {
			p.addError(elif.Condition.Pos(), "unclosed block (missing '}')")
			return nil
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
			if p.curTokenIs(lexer.SEMICOLON) {
				p.nextToken()
				continue
			}
			bodyStmt := p.parseStatement()
			if bodyStmt != nil {
				stmt.Alternative = append(stmt.Alternative, bodyStmt)
			}
			if p.peekTokenIs(lexer.SEMICOLON) {
				p.nextToken()
			}
			p.nextToken()
		}
		if p.curTokenIs(lexer.EOF) {
			p.addError(stmt.Position, "unclosed block (missing '}')")
			return nil
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

// while condition { ... }
func (p *Parser) parseWhileStatement() ast.Statement {
	startPos := p.curToken.Pos
	p.nextToken() // saute 'while'

	condition := p.parseExpression()
	if condition == nil {
		p.addError(startPos, "expected condition expression after 'while'")
		return nil
	}

	if !p.expectPeek(lexer.LBRACE) {
		return nil
	}

	p.nextToken() // saute '{'
	stmt := &ast.WhileStatement{
		Position:  startPos,
		Condition: condition,
	}

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
		p.addError(stmt.Position, "unclosed while block (missing '}')")
		return nil
	}

	return stmt
}
