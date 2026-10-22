package parser

import (
	"fmt"
	"strings"

	"github.com/Nehonix-Team/xru/pkg/ast"
	"github.com/Nehonix-Team/xru/pkg/lexer"
)

// func Name(params...) -> ReturnType { ... }
func (p *Parser) parseFuncStatement() *ast.FuncStatement {
	stmt := &ast.FuncStatement{Position: p.curToken.Pos, NativeHook: p.currentNativeHook}
	p.currentNativeHook = ""
	if !p.expectPeek(lexer.IDENT) {
		return nil
	}
	stmt.Name = p.curToken.Literal
	stmt.Parameters = p.parseParameterList()

	if p.peekTokenIs(lexer.ARROW) || p.peekTokenIs(lexer.COLON) {
		p.nextToken() // saute '->' ou ':'
		stmt.ReturnType = p.parseTypeAnnotation()
	}

	if stmt.NativeHook != "" && !p.peekTokenIs(lexer.LBRACE) {
		if p.peekTokenIs(lexer.SEMICOLON) {
			p.nextToken()
		}
		return stmt
	}

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
			stmt.Body = append(stmt.Body, bodyStmt)
		}
		if p.peekTokenIs(lexer.SEMICOLON) {
			p.nextToken()
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
		isVariadic := false
		if p.curTokenIs(lexer.ELLIPSIS) {
			isVariadic = true
			p.nextToken()
		}
		if !p.curTokenIs(lexer.IDENT) && !p.curTokenIs(lexer.DEFAULT) && !p.curTokenIs(lexer.SET) {
			break
		}
		param := &ast.Parameter{Name: p.curToken.Literal, IsVariadic: isVariadic}
		if p.peekTokenIs(lexer.QUESTION) {
			p.nextToken() // saute '?'
			param.Optional = true
		}
		if p.peekTokenIs(lexer.COLON) {
			p.nextToken() // saute ':'
			if p.peekTokenIs(lexer.ELLIPSIS) {
				param.IsVariadic = true
				p.nextToken()
			}
			param.Type = p.parseTypeAnnotation()
			if strings.HasPrefix(param.Type, "...") {
				param.IsVariadic = true
				param.Type = strings.TrimPrefix(param.Type, "...")
			}
			if strings.HasSuffix(param.Type, "?") {
				param.Optional = true
			}
		}
		if p.peekTokenIs(lexer.ASSIGN) {
			p.nextToken() // saute '='
			param.Optional = true
			p.nextToken() // curToken becomes first token of default value
			param.DefaultValue = p.parseExpression()
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
	if !p.peekTokenIs(lexer.RBRACE) && !p.peekTokenIs(lexer.EOF) && !p.peekTokenIs(lexer.SEMICOLON) {
		p.nextToken()
		stmt.Value = p.parseExpression()
	}
	if p.peekTokenIs(lexer.SEMICOLON) {
		p.nextToken()
	}
	return stmt
}

func (p *Parser) parseGlobalStatement() ast.Statement {
	p.nextToken() // saute 'global'
	if p.curTokenIs(lexer.AT) && p.peekTokenIs(lexer.IDENT) && p.peekToken.Literal == "native" {
		nativeHook := p.parseNativeAnnotation()
		if nativeHook != "" {
			if !p.expectPeek(lexer.FUNC) {
				return nil
			}
			p.currentNativeHook = nativeHook
			fnStmt := p.parseFuncStatement()
			if fnStmt != nil {
				fnStmt.IsGlobal = true
				return fnStmt
			}
			return nil
		}
	}
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
	if p.curTokenIs(lexer.INTERFACE) || p.curTokenIs(lexer.TYPE) {
		ifaceStmt := p.parseInterfaceStatement()
		if ifaceStmt != nil {
			ifaceStmt.IsGlobal = true
			return ifaceStmt
		}
		return nil
	}
	stmt := p.parseStatement()
	return stmt
}

// parseNativeAnnotation parses `@native("hook_name")` when curToken is '@'.
// It returns the hook name and leaves curToken at ')'
func (p *Parser) parseNativeAnnotation() string {
	if !p.curTokenIs(lexer.AT) {
		return ""
	}
	if !p.expectPeek(lexer.IDENT) || p.curToken.Literal != "native" {
		return ""
	}
	if !p.expectPeek(lexer.LPAREN) {
		return ""
	}
	if !p.expectPeek(lexer.STRING) {
		return ""
	}
	hookName := p.curToken.Literal
	if !p.expectPeek(lexer.RPAREN) {
		return ""
	}
	return hookName
}
