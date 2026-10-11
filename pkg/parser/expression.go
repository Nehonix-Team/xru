package parser

import (
	"fmt"
	"strings"

	"github.com/Nehonix-Team/xru/pkg/ast"
	"github.com/Nehonix-Team/xru/pkg/lexer"
)

// -----------------------------------------------------------------------------
// Niveaux de précédence pour les opérateurs
// -----------------------------------------------------------------------------

const (
	_ int = iota
	LOWEST
	CATCH_PREC  // catch
	LOGICAL_OR  // || or
	LOGICAL_AND // &&
	EQUALS      // == !=
	LESSGREATER // > < >= <=
	RANGE_PREC  // ..
	SUM         // + -
	PRODUCT     // * / %
	PREFIX      // !X, -X
)

var precedences = map[lexer.TokenType]int{
	lexer.CATCH:      CATCH_PREC,
	lexer.OR:         LOGICAL_OR,
	lexer.OR_LOGICAL: LOGICAL_OR,
	lexer.AND:        LOGICAL_AND,
	lexer.EQ:         EQUALS,
	lexer.NEQ:        EQUALS,
	lexer.LT:         LESSGREATER,
	lexer.LTE:        LESSGREATER,
	lexer.GT:         LESSGREATER,
	lexer.GTE:        LESSGREATER,
	lexer.RANGE:      RANGE_PREC,
	lexer.PLUS:       SUM,
	lexer.MINUS:      SUM,
	lexer.SLASH:      PRODUCT,
	lexer.ASTERISK:   PRODUCT,
	lexer.PERCENT:    PRODUCT,
}

func (p *Parser) peekPrecedence() int {
	if prec, ok := precedences[p.peekToken.Type]; ok {
		return prec
	}
	return LOWEST
}

func (p *Parser) curPrecedence() int {
	if prec, ok := precedences[p.curToken.Type]; ok {
		return prec
	}
	return LOWEST
}

func (p *Parser) ParseExpression() ast.Expression {
	return p.parseExpressionWithPrecedence(LOWEST)
}

func (p *Parser) parseExpression() ast.Expression {
	return p.parseExpressionWithPrecedence(LOWEST)
}

func (p *Parser) parseExpressionWithPrecedence(precedence int) ast.Expression {
	expr := p.parsePrimaryExpression()
	if expr == nil {
		return nil
	}

	// Gestion des membres 'Target.Property' ou 'Target.Method()'
	for p.peekTokenIs(lexer.DOT) {
		p.nextToken() // saute '.'
		if !p.expectPeek(lexer.IDENT) {
			return expr
		}
		propName := p.curToken.Literal
		expr = &ast.MemberExpression{
			Position: expr.Pos(),
			Object:   expr,
			Property: propName,
		}

		// Si suivi d'un appel '(...)'
		if p.peekTokenIs(lexer.LPAREN) {
			p.nextToken()
			args := p.parseExpressionList(lexer.RPAREN)
			expr = &ast.CallExpression{
				Position:  expr.Pos(),
				Caller:    expr,
				Arguments: args,
			}
		}
	}

	// Gestion des opérateurs infixes (arithmétique +, -, *, /, % et comparaisons ==, !=, <, <=, >, >=)
	for precedence < p.peekPrecedence() {
		if p.peekTokenIs(lexer.CATCH) {
			p.nextToken() // saute 'catch', curToken est 'catch'
			catchPos := p.curToken.Pos
			var errVar string
			var fallback ast.Node

			if p.peekTokenIs(lexer.IDENT) {
				p.nextToken()
				errVar = p.curToken.Literal
			}

			if p.peekTokenIs(lexer.OR) {
				p.nextToken() // saute 'or', curToken est 'or'
				if p.peekTokenIs(lexer.RETURN) {
					p.nextToken()
					fallback = p.parseReturnStatement()
				} else if p.peekTokenIs(lexer.LBRACE) {
					p.nextToken()
					fallback = p.parseBlockStatement()
				} else {
					p.nextToken()
					fallback = p.parseExpressionWithPrecedence(CATCH_PREC)
				}
			} else if errVar == "" && !p.peekTokenIs(lexer.EOF) && !p.peekTokenIs(lexer.RPAREN) && !p.peekTokenIs(lexer.RBRACE) && !p.peekTokenIs(lexer.COMMA) {
				// Fallback direct sans variable d'erreur, ex: expr catch "fallback"
				p.nextToken()
				fallback = p.parseExpressionWithPrecedence(CATCH_PREC)
			}

			expr = &ast.CatchExpression{
				Position: catchPos,
				Target:   expr,
				ErrVar:   errVar,
				Fallback: fallback,
			}
			continue
		}

		p.nextToken()
		op := p.curToken.Literal
		opPrec := p.curPrecedence()
		p.nextToken()
		right := p.parseExpressionWithPrecedence(opPrec)
		if right == nil {
			return expr
		}
		if op == ".." {
			expr = &ast.RangeExpression{
				Position: expr.Pos(),
				Start:    expr,
				End:      right,
			}
			continue
		}
		expr = &ast.InfixExpression{
			Position: expr.Pos(),
			Left:     expr,
			Operator: op,
			Right:    right,
		}
	}

	return expr
}

func (p *Parser) parsePrimaryExpression() ast.Expression {
	switch p.curToken.Type {
	case lexer.DOUBLE_COLON:
		// Invocation de fonction globale ::Func(...)
		pos := p.curToken.Pos
		if !p.expectPeek(lexer.IDENT) {
			p.addError(pos, "expected identifier after '::'")
			return nil
		}
		fnName := p.curToken.Literal
		if p.peekTokenIs(lexer.LPAREN) {
			p.nextToken()
			args := p.parseExpressionList(lexer.RPAREN)
			return &ast.GlobalCallExpression{
				Position:  pos,
				Function:  fnName,
				Arguments: args,
			}
		}
		return &ast.GlobalAccessExpression{
			Position: pos,
			Name:     fnName,
		}
	case lexer.GLOBAL:
		if p.peekTokenIs(lexer.DOUBLE_COLON) {
			pos := p.curToken.Pos
			p.nextToken() // saute "global" (curToken devient '::')
			if !p.expectPeek(lexer.IDENT) {
				p.addError(pos, "expected identifier after 'global::'")
				return nil
			}
			fnName := p.curToken.Literal
			if p.peekTokenIs(lexer.LPAREN) {
				p.nextToken()
				args := p.parseExpressionList(lexer.RPAREN)
				return &ast.GlobalCallExpression{
					Position:  pos,
					Function:  fnName,
					Arguments: args,
				}
			}
			return &ast.GlobalAccessExpression{
				Position: pos,
				Name:     fnName,
			}
		}
		return &ast.Identifier{Position: p.curToken.Pos, Value: p.curToken.Literal}
	case lexer.IDENT:
		// Support 'global::Func(...)' ou 'global::Var'
		if strings.EqualFold(p.curToken.Literal, "global") && p.peekTokenIs(lexer.DOUBLE_COLON) {
			pos := p.curToken.Pos
			p.nextToken() // saute "global" (curToken devient '::')
			if !p.expectPeek(lexer.IDENT) {
				p.addError(pos, "expected identifier after 'global::'")
				return nil
			}
			fnName := p.curToken.Literal
			if p.peekTokenIs(lexer.LPAREN) {
				p.nextToken()
				args := p.parseExpressionList(lexer.RPAREN)
				return &ast.GlobalCallExpression{
					Position:  pos,
					Function:  fnName,
					Arguments: args,
				}
			}
			return &ast.GlobalAccessExpression{
				Position: pos,
				Name:     fnName,
			}
		}

		identName := p.curToken.Literal
		pos := p.curToken.Pos

		// Support 'a::b' ou 'a::login(...)'
		if p.peekTokenIs(lexer.DOUBLE_COLON) {
			p.nextToken() // saute ident (curToken devient '::')
			if !p.expectPeek(lexer.IDENT) {
				p.addError(pos, "expected identifier after '::'")
				return nil
			}
			member := p.curToken.Literal
			scopeExpr := &ast.ScopeAccessExpression{
				Position: pos,
				Scope:    identName,
				Member:   member,
			}
			if p.peekTokenIs(lexer.LPAREN) {
				p.nextToken()
				args := p.parseExpressionList(lexer.RPAREN)
				return &ast.CallExpression{
					Position:  pos,
					Caller:    scopeExpr,
					Arguments: args,
				}
			}
			return scopeExpr
		}

		ident := &ast.Identifier{Position: pos, Value: identName}
		// Appel direct : Func(...)
		if p.peekTokenIs(lexer.LPAREN) {
			p.nextToken()
			args := p.parseExpressionList(lexer.RPAREN)
			return &ast.CallExpression{
				Position:  ident.Pos(),
				Caller:    ident,
				Arguments: args,
			}
		}
		return ident
	case lexer.BANG:
		pos := p.curToken.Pos
		p.nextToken()
		right := p.parseExpressionWithPrecedence(PREFIX)
		if right == nil {
			return nil
		}
		return &ast.PrefixExpression{
			Position: pos,
			Operator: "!",
			Right:    right,
		}
	case lexer.MINUS:
		pos := p.curToken.Pos
		p.nextToken()
		right := p.parseExpressionWithPrecedence(PREFIX)
		if right == nil {
			return nil
		}
		return &ast.PrefixExpression{
			Position: pos,
			Operator: "-",
			Right:    right,
		}
	case lexer.AMP:
		// Accès propriété interne &prop
		p.nextToken()
		return &ast.PropAccess{Position: p.curToken.Pos, Name: p.curToken.Literal}
	case lexer.WITH:
		pos := p.curToken.Pos
		p.nextToken()
		target := p.parseExpression()
		if target == nil {
			p.addError(pos, "expected expression after 'with'")
			return nil
		}
		return &ast.WithExpression{
			Position: pos,
			Target:   target,
		}
	case lexer.AT:
		// Instanciation @Unit(...)
		pos := p.curToken.Pos
		p.nextToken()
		unitName := p.curToken.Literal
		p.expectPeek(lexer.LPAREN)
		args := p.parseExpressionList(lexer.RPAREN)
		return &ast.InstantiationExpression{
			Position:  pos,
			UnitName:  unitName,
			Arguments: args,
		}
	case lexer.STRING:
		return &ast.StringLiteral{Position: p.curToken.Pos, Value: p.curToken.Literal}
	case lexer.NUMBER:
		return &ast.NumberLiteral{Position: p.curToken.Pos, Value: p.curToken.Literal}
	case lexer.BOOL:
		return &ast.BoolLiteral{Position: p.curToken.Pos, Value: p.curToken.Literal == "true"}
	case lexer.DURATION:
		return &ast.DurationLiteral{Position: p.curToken.Pos, Value: p.curToken.Literal}
	case lexer.LBRACKET:
		return p.parseCollectionLiteral()
	case lexer.LPAREN:
		return p.parseParenOrLambdaExpression()
	case lexer.LBRACE:
		return p.parseBlockStatement()
	case lexer.IF:
		return p.parseIfExpression()
	case lexer.ILLEGAL:
		p.addError(p.curToken.Pos, fmt.Sprintf("illegal character '%s'", p.curToken.Literal))
		return nil
	default:
		p.addError(p.curToken.Pos, fmt.Sprintf("unexpected token '%s'", p.curToken.Literal))
		return nil
	}
}

func (p *Parser) parseCollectionLiteral() ast.Expression {
	startPos := p.curToken.Pos
	p.nextToken() // saute '['

	// Cas collection vide
	if p.curTokenIs(lexer.RBRACKET) {
		return &ast.ListLiteral{Position: startPos, Elements: []ast.Expression{}}
	}

	firstExpr := p.parseExpression()

	// Si suivi d'une flèche '->', c'est une map : ["key" -> "value"]
	if p.peekTokenIs(lexer.ARROW) {
		p.nextToken() // saute '->'
		p.nextToken()
		valExpr := p.parseExpression()
		mapLit := &ast.MapLiteral{
			Position: startPos,
			Pairs: []*ast.MapPair{
				{Key: firstExpr, Value: valExpr},
			},
		}

		for p.peekTokenIs(lexer.COMMA) {
			p.nextToken() // saute ','
			p.nextToken()
			if p.curTokenIs(lexer.RBRACKET) {
				break
			}
			key := p.parseExpression()
			p.expectPeek(lexer.ARROW)
			p.nextToken()
			val := p.parseExpression()
			mapLit.Pairs = append(mapLit.Pairs, &ast.MapPair{Key: key, Value: val})
		}
		p.expectPeek(lexer.RBRACKET)
		return mapLit
	}

	// Sinon, c'est une liste standard : [item1, item2, ...]
	listLit := &ast.ListLiteral{
		Position: startPos,
		Elements: []ast.Expression{firstExpr},
	}
	for p.peekTokenIs(lexer.COMMA) {
		p.nextToken() // saute ','
		p.nextToken()
		if p.curTokenIs(lexer.RBRACKET) {
			break
		}
		listLit.Elements = append(listLit.Elements, p.parseExpression())
	}
	p.expectPeek(lexer.RBRACKET)
	return listLit
}

func (p *Parser) parseExpressionList(end lexer.TokenType) []ast.Expression {
	var list []ast.Expression

	if p.peekTokenIs(end) {
		p.nextToken()
		return list
	}

	p.nextToken()
	list = append(list, p.parseExpression())

	for p.peekTokenIs(lexer.COMMA) {
		p.nextToken()
		p.nextToken()
		list = append(list, p.parseExpression())
	}

	if !p.expectPeek(end) {
		return nil
	}
	return list
}

func (p *Parser) isAheadLambda() bool {
	savedLexer := *p.l
	savedCur := p.curToken
	savedPeek := p.peekToken

	defer func() {
		*p.l = savedLexer
		p.curToken = savedCur
		p.peekToken = savedPeek
	}()

	p.nextToken() // premier token après '('
	if p.curTokenIs(lexer.RPAREN) {
		return p.peekTokenIs(lexer.FAT_ARROW) || p.peekTokenIs(lexer.ARROW)
	}

	for p.curTokenIs(lexer.IDENT) {
		if p.peekTokenIs(lexer.COLON) {
			p.nextToken() // ':'
			if !p.expectPeek(lexer.IDENT) {
				return false
			}
		}
		if p.peekTokenIs(lexer.COMMA) {
			p.nextToken() // ','
			p.nextToken() // prochain IDENT
			continue
		}
		break
	}

	if p.peekTokenIs(lexer.RPAREN) {
		p.nextToken() // ')'
		return p.peekTokenIs(lexer.FAT_ARROW) || p.peekTokenIs(lexer.ARROW)
	}

	return false
}

func (p *Parser) parseParenOrLambdaExpression() ast.Expression {
	startPos := p.curToken.Pos

	if p.isAheadLambda() {
		p.nextToken() // saute '('
		var params []*ast.Parameter
		if !p.curTokenIs(lexer.RPAREN) {
			for {
				if !p.curTokenIs(lexer.IDENT) {
					break
				}
				param := &ast.Parameter{Name: p.curToken.Literal}
				if p.peekTokenIs(lexer.COLON) {
					p.nextToken() // ':'
					if p.peekTokenIs(lexer.IDENT) {
						p.nextToken()
						param.Type = p.curToken.Literal
					}
				}
				params = append(params, param)
				if p.peekTokenIs(lexer.COMMA) {
					p.nextToken() // ','
					p.nextToken() // prochain ident
					continue
				}
				break
			}
			p.expectPeek(lexer.RPAREN)
		}
		if p.peekTokenIs(lexer.FAT_ARROW) || p.peekTokenIs(lexer.ARROW) {
			p.nextToken()
		} else {
			p.expectPeek(lexer.FAT_ARROW)
		}
		p.nextToken() // premier token du corps
		var body ast.Expression
		if p.curTokenIs(lexer.LBRACE) {
			body = p.parseBlockStatement()
		} else {
			body = p.parseExpression()
		}
		return &ast.LambdaExpression{
			Position:   startPos,
			Parameters: params,
			Body:       body,
		}
	}

	// Expression parenthésée classique (expr)
	p.nextToken() // saute '('
	expr := p.parseExpression()
	p.expectPeek(lexer.RPAREN)
	return expr
}

// if <condition> -> <consequence> else <alternative>
func (p *Parser) parseIfExpression() *ast.IfExpression {
	startPos := p.curToken.Pos
	p.nextToken() // saute 'if'

	condition := p.parseExpression()
	if condition == nil {
		return nil
	}

	if !p.expectPeek(lexer.ARROW) {
		return nil
	}

	p.nextToken() // saute '->'
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

	return &ast.IfExpression{
		Position:    startPos,
		Condition:   condition,
		Consequence: consequence,
		Alternative: alternative,
	}
}

