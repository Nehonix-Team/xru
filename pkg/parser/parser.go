package parser

import (
	"fmt"
	"strings"

	"github.com/Nehonix-Team/xru/pkg/ast"
	"github.com/Nehonix-Team/xru/pkg/lexer"
)

type Parser struct {
	l         *lexer.Lexer
	curToken  lexer.Token
	peekToken lexer.Token
	errors    []string
}

func New(l *lexer.Lexer) *Parser {
	p := &Parser{l: l}
	// Lit deux tokens pour initialiser curToken et peekToken
	p.nextToken()
	p.nextToken()
	return p
}

func (p *Parser) nextToken() {
	p.curToken = p.peekToken
	p.peekToken = p.l.NextToken()
}

func (p *Parser) Errors() []string {
	return p.errors
}

func (p *Parser) addError(pos lexer.Position, msg string) {
	p.errors = append(p.errors, fmt.Sprintf("%s: %s", pos, msg))
}

func (p *Parser) curTokenIs(t lexer.TokenType) bool {
	return p.curToken.Type == t
}

func (p *Parser) peekTokenIs(t lexer.TokenType) bool {
	return p.peekToken.Type == t
}

func (p *Parser) expectPeek(t lexer.TokenType) bool {
	if p.peekTokenIs(t) {
		p.nextToken()
		return true
	}
	p.addError(p.peekToken.Pos, fmt.Sprintf("expected token %s, got %s ('%s')", t, p.peekToken.Type, p.peekToken.Literal))
	return false
}

func (p *Parser) ParseProgram() *ast.Program {
	program := &ast.Program{
		Statements: []ast.Statement{},
	}

	for !p.curTokenIs(lexer.EOF) {
		stmt := p.parseStatement()
		if stmt != nil {
			program.Statements = append(program.Statements, stmt)
		}
		p.nextToken()
	}

	return program
}

func (p *Parser) parseStatement() ast.Statement {
	switch p.curToken.Type {
	case lexer.USE_DIR:
		return p.parseUseStatement()
	case lexer.ARG_DIR:
		return p.parseArgStatement()
	case lexer.LET, lexer.CONST:
		return p.parseVarStatement()
	case lexer.UNIT_DIR:
		return p.parseUnitStatement()
	case lexer.FUNC_DIR:
		return p.parseFuncStatement()
	case lexer.WITH_DIR:
		return p.parseWithStatement()
	case lexer.IF_DIR:
		return p.parseIfStatement()
	case lexer.MATCH_DIR:
		return p.parseMatchStatement()
	case lexer.DISPATCH_TREE:
		return p.parseDispatchTreeStatement()
	case lexer.FILE_DIR:
		return p.parseFileStatement()
	case lexer.GLOBAL_DIR:
		return p.parseGlobalStatement()
	case lexer.CUSTOM_DIR:
		return p.parseCustomDirective()
	case lexer.RETURN_DIR:
		return p.parseReturnStatement()
	default:
		// Expression libre (ex: appel de fonction ou méthode)
		return p.parseExpressionStatement()
	}
}

// #USE module as Alias
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

// #ARG NAME: type or default
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

	if p.peekTokenIs(lexer.OR) {
		p.nextToken() // mange 'or'
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

// #UNIT Name ... #END
func (p *Parser) parseUnitStatement() *ast.UnitStatement {
	stmt := &ast.UnitStatement{Position: p.curToken.Pos}
	if !p.expectPeek(lexer.IDENT) {
		return nil
	}
	stmt.Name = p.curToken.Literal

	p.nextToken()
	for !p.curTokenIs(lexer.END_DIR) && !p.curTokenIs(lexer.EOF) {
		switch p.curToken.Type {
		case lexer.DECLARE_DIR:
			decl := p.parseDeclareStatement()
			if decl != nil {
				stmt.Properties = append(stmt.Properties, decl)
			}
		case lexer.DIRECTIVE_DIR:
			dir := p.parseDirectiveDefStatement()
			if dir != nil {
				stmt.Directives = append(stmt.Directives, dir)
			}
		case lexer.FUNC_DIR:
			fn := p.parseFuncStatement()
			if fn != nil {
				stmt.Methods = append(stmt.Methods, fn)
			}
		default:
			p.addError(p.curToken.Pos, fmt.Sprintf("unexpected statement inside #UNIT: %s", p.curToken.Literal))
		}
		p.nextToken()
	}

	if p.curTokenIs(lexer.EOF) {
		p.addError(stmt.Position, fmt.Sprintf("unclosed #UNIT '%s' (missing #END)", stmt.Name))
	}

	return stmt
}

// #DECLARE prop: type = value
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

// #DIRECTIVE Name(params...) ... #END
func (p *Parser) parseDirectiveDefStatement() *ast.DirectiveDefStatement {
	stmt := &ast.DirectiveDefStatement{Position: p.curToken.Pos}
	if !p.expectPeek(lexer.IDENT) {
		return nil
	}
	stmt.Name = p.curToken.Literal
	stmt.Parameters = p.parseParameterList()

	p.nextToken()
	for !p.curTokenIs(lexer.END_DIR) && !p.curTokenIs(lexer.EOF) {
		bodyStmt := p.parseStatement()
		if bodyStmt != nil {
			stmt.Body = append(stmt.Body, bodyStmt)
		}
		p.nextToken()
	}
	return stmt
}

// #FUNC Name(params...) -> ReturnType ... #END
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

	p.nextToken()
	for !p.curTokenIs(lexer.END_DIR) && !p.curTokenIs(lexer.EOF) {
		bodyStmt := p.parseStatement()
		if bodyStmt != nil {
			stmt.Body = append(stmt.Body, bodyStmt)
		}
		p.nextToken()
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

// #WITH Target ... #END
func (p *Parser) parseWithStatement() *ast.WithStatement {
	stmt := &ast.WithStatement{Position: p.curToken.Pos}
	p.nextToken()
	stmt.Target = p.parseExpression()

	p.nextToken()
	for !p.curTokenIs(lexer.END_DIR) && !p.curTokenIs(lexer.EOF) {
		bodyStmt := p.parseStatement()
		if bodyStmt != nil {
			stmt.Body = append(stmt.Body, bodyStmt)
		}
		p.nextToken()
	}

	if p.curTokenIs(lexer.EOF) {
		p.addError(stmt.Position, "unclosed #WITH block (missing #END)")
	}
	return stmt
}

// #CUSTOM_DIR (ex: #SET Method = "GET", #HEADER "Accept" -> "...")
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

func (p *Parser) parseReturnStatement() *ast.ReturnStatement {
	stmt := &ast.ReturnStatement{Position: p.curToken.Pos}
	if !p.peekTokenIs(lexer.END_DIR) && !p.peekTokenIs(lexer.EOF) {
		p.nextToken()
		stmt.Value = p.parseExpression()
	}
	return stmt
}

// #IF condition ... #ELIF ... #ELSE ... #END
func (p *Parser) parseIfStatement() *ast.IfStatement {
	stmt := &ast.IfStatement{Position: p.curToken.Pos}
	p.nextToken()
	stmt.Condition = p.parseExpression()

	p.nextToken()
	for !p.curTokenIs(lexer.END_DIR) && !p.curTokenIs(lexer.ELIF_DIR) && !p.curTokenIs(lexer.ELSE_DIR) && !p.curTokenIs(lexer.EOF) {
		bodyStmt := p.parseStatement()
		if bodyStmt != nil {
			stmt.Consequence = append(stmt.Consequence, bodyStmt)
		}
		p.nextToken()
	}

	for p.curTokenIs(lexer.ELIF_DIR) {
		elif := &ast.ElifStatement{}
		p.nextToken()
		elif.Condition = p.parseExpression()
		p.nextToken()
		for !p.curTokenIs(lexer.END_DIR) && !p.curTokenIs(lexer.ELIF_DIR) && !p.curTokenIs(lexer.ELSE_DIR) && !p.curTokenIs(lexer.EOF) {
			bodyStmt := p.parseStatement()
			if bodyStmt != nil {
				elif.Body = append(elif.Body, bodyStmt)
			}
			p.nextToken()
		}
		stmt.ElifBlocks = append(stmt.ElifBlocks, elif)
	}

	if p.curTokenIs(lexer.ELSE_DIR) {
		p.nextToken()
		for !p.curTokenIs(lexer.END_DIR) && !p.curTokenIs(lexer.EOF) {
			bodyStmt := p.parseStatement()
			if bodyStmt != nil {
				stmt.Alternative = append(stmt.Alternative, bodyStmt)
			}
			p.nextToken()
		}
	}

	return stmt
}

// #MATCH target ... #CASE ... #DEFAULT ... #END
func (p *Parser) parseMatchStatement() *ast.MatchStatement {
	stmt := &ast.MatchStatement{Position: p.curToken.Pos}
	p.nextToken()
	stmt.Target = p.parseExpression()

	p.nextToken()
	for !p.curTokenIs(lexer.END_DIR) && !p.curTokenIs(lexer.EOF) {
		if p.curTokenIs(lexer.CASE_DIR) {
			kase := &ast.MatchCase{}
			p.nextToken()
			kase.Pattern = p.parseExpression()
			p.nextToken()
			for !p.curTokenIs(lexer.CASE_DIR) && !p.curTokenIs(lexer.DEFAULT_DIR) && !p.curTokenIs(lexer.END_DIR) && !p.curTokenIs(lexer.EOF) {
				bodyStmt := p.parseStatement()
				if bodyStmt != nil {
					kase.Body = append(kase.Body, bodyStmt)
				}
				p.nextToken()
			}
			stmt.Cases = append(stmt.Cases, kase)
			continue
		}

		if p.curTokenIs(lexer.DEFAULT_DIR) {
			p.nextToken()
			for !p.curTokenIs(lexer.END_DIR) && !p.curTokenIs(lexer.EOF) {
				bodyStmt := p.parseStatement()
				if bodyStmt != nil {
					stmt.DefaultBlock = append(stmt.DefaultBlock, bodyStmt)
				}
				p.nextToken()
			}
			break
		}
		p.nextToken()
	}
	return stmt
}

func (p *Parser) parseGlobalStatement() ast.Statement {
	p.nextToken() // saute #GLOBAL
	// Peut être suivi de #FUNC ou IDENT ("FUNC" / "func")
	if p.curTokenIs(lexer.FUNC_DIR) || (p.curTokenIs(lexer.IDENT) && strings.EqualFold(p.curToken.Literal, "func")) {
		fnStmt := p.parseFuncStatement()
		if fnStmt != nil {
			fnStmt.IsGlobal = true
		}
		return fnStmt
	}
	stmt := p.parseStatement()
	return stmt
}

func (p *Parser) parseDispatchTreeStatement() *ast.DispatchTreeStatement {
	stmt := &ast.DispatchTreeStatement{
		Position: p.curToken.Pos,
		Pattern:  ".rules.xru",
		PassArgs: make(map[string]ast.Expression),
	}
	if p.expectPeek(lexer.STRING) {
		stmt.Path = p.curToken.Literal
	}
	if p.peekTokenIs(lexer.COMMA) {
		p.nextToken() // saute ','
		if p.expectPeek(lexer.STRING) {
			stmt.Pattern = p.curToken.Literal
		}
	}

	// Support de la syntaxe en bloc (#PASS / #SET)
	for p.peekTokenIs(lexer.PASS_DIR) || p.peekToken.Literal == "#PASS" || p.peekToken.Literal == "#SET" {
		p.nextToken() // avance vers #PASS ou #SET
		if p.expectPeek(lexer.IDENT) {
			key := p.curToken.Literal
			if p.expectPeek(lexer.ASSIGN) {
				p.nextToken() // saute '='
				valExpr := p.parseExpression()
				stmt.PassArgs[key] = valExpr
			}
		}
	}

	if p.peekTokenIs(lexer.END_DIR) {
		p.nextToken() // saute #END
	}

	return stmt
}

func (p *Parser) parseFileStatement() *ast.FileStatement {
	stmt := &ast.FileStatement{Position: p.curToken.Pos}
	if p.expectPeek(lexer.STRING) {
		stmt.Path = p.curToken.Literal
	}
	p.nextToken()
	for !p.curTokenIs(lexer.END_DIR) && !p.curTokenIs(lexer.EOF) {
		bodyStmt := p.parseStatement()
		if bodyStmt != nil {
			stmt.Body = append(stmt.Body, bodyStmt)
		}
		p.nextToken()
	}
	return stmt
}

func (p *Parser) parseExpressionStatement() *ast.ExpressionStatement {
	stmt := &ast.ExpressionStatement{
		Position:   p.curToken.Pos,
		Expression: p.parseExpression(),
	}
	return stmt
}

// -----------------------------------------------------------------------------
// Analyseur d'Expressions
// -----------------------------------------------------------------------------

func (p *Parser) parseExpression() ast.Expression {
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

	// Gestion des opérateurs de comparaison (==, !=, <, <=, >, >=)
	if p.peekTokenIs(lexer.EQ) || p.peekTokenIs(lexer.NEQ) || p.peekTokenIs(lexer.LT) ||
		p.peekTokenIs(lexer.LTE) || p.peekTokenIs(lexer.GT) || p.peekTokenIs(lexer.GTE) {
		p.nextToken()
		op := p.curToken.Literal
		p.nextToken()
		right := p.parseExpression()
		if right == nil {
			return expr
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
	case lexer.LBRACE:
		// Interpolation variable {IDENT}
		pos := p.curToken.Pos
		p.nextToken()
		varName := p.curToken.Literal
		p.expectPeek(lexer.RBRACE)
		return &ast.Identifier{Position: pos, Value: varName}
	case lexer.IDENT:
		ident := &ast.Identifier{Position: p.curToken.Pos, Value: p.curToken.Literal}
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
	case lexer.AMP:
		// Accès propriété interne &prop
		p.nextToken()
		return &ast.PropAccess{Position: p.curToken.Pos, Name: p.curToken.Literal}
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
	default:
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
			k := p.parseExpression()
			p.expectPeek(lexer.ARROW)
			p.nextToken()
			v := p.parseExpression()
			mapLit.Pairs = append(mapLit.Pairs, &ast.MapPair{Key: k, Value: v})
		}
		p.expectPeek(lexer.RBRACKET)
		return mapLit
	}

	// Sinon, c'est une liste standard
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
		elem := p.parseExpression()
		listLit.Elements = append(listLit.Elements, elem)
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
