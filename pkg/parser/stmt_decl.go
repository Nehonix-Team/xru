package parser

import (
	"fmt"

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

// arg name [as alias]: type = default  /  arg name: type or default
func (p *Parser) parseArgStatement() *ast.ArgStatement {
	stmt := &ast.ArgStatement{Position: p.curToken.Pos}
	if !p.expectPeek(lexer.IDENT) {
		return nil
	}
	stmt.Name = p.curToken.Literal

	if p.peekTokenIs(lexer.AS) {
		p.nextToken() // mange 'as'
		if !p.expectPeek(lexer.IDENT) {
			return nil
		}
		stmt.Alias = p.curToken.Literal
	}

	if p.peekTokenIs(lexer.COLON) {
		p.nextToken() // mange ':'
		if p.peekTokenIs(lexer.IDENT) || p.peekTokenIs(lexer.MAP) {
			p.nextToken()
			stmt.Type = p.curToken.Literal
		} else {
			return nil
		}
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
	stmt.NamePos = p.curToken.Pos

	if p.peekTokenIs(lexer.COLON) {
		p.nextToken() // mange ':'
		if p.peekTokenIs(lexer.IDENT) || p.peekTokenIs(lexer.MAP) {
			p.nextToken()
		} else {
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

	assignPos := p.curToken.Pos
	if p.peekToken.Pos.Line != assignPos.Line || p.peekTokenIs(lexer.EOF) || p.peekTokenIs(lexer.RBRACE) {
		p.addError(assignPos, "expected expression after '=' in variable declaration")
		return stmt
	}

	p.nextToken() // début de la valeur
	stmt.Value = p.parseExpression()
	if stmt.Value == nil {
		p.addError(assignPos, "expected expression after '=' in variable declaration")
		return stmt
	}
	if p.peekToken.Pos.Line == p.curToken.Pos.Line && !p.peekTokenIs(lexer.EOF) && !p.peekTokenIs(lexer.RBRACE) && !p.peekTokenIs(lexer.COMMENT) {
		p.addError(p.peekToken.Pos, fmt.Sprintf("unexpected token '%s' on same line after variable declaration", p.peekToken.Literal))
	}
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
		if p.peekTokenIs(lexer.IDENT) || p.peekTokenIs(lexer.MAP) {
			p.nextToken()
			stmt.Type = p.curToken.Literal
		} else {
			return nil
		}
	}

	if p.peekTokenIs(lexer.ASSIGN) {
		p.nextToken() // saute '='
		assignPos := p.curToken.Pos
		if p.peekToken.Pos.Line != assignPos.Line || p.peekTokenIs(lexer.EOF) || p.peekTokenIs(lexer.RBRACE) {
			p.addError(assignPos, "expected expression after '=' in declaration")
			return stmt
		}
		p.nextToken()
		stmt.Value = p.parseExpression()
		if stmt.Value == nil {
			p.addError(assignPos, "expected expression after '=' in declaration")
			return stmt
		}
	}
	return stmt
}

// map { alias -> path ... }
func (p *Parser) parseMapStatement() *ast.MapStatement {
	stmt := &ast.MapStatement{
		Position: p.curToken.Pos,
		Mappings: make(map[string]string),
	}
	if !p.expectPeek(lexer.LBRACE) {
		return nil
	}
	p.nextToken() // saute '{'
	for !p.curTokenIs(lexer.RBRACE) && !p.curTokenIs(lexer.EOF) {
		if p.curTokenIs(lexer.IDENT) || p.curTokenIs(lexer.STRING) {
			alias := p.curToken.Literal
			if !p.expectPeek(lexer.ARROW) {
				return nil
			}
			if !p.expectPeek(lexer.STRING) {
				return nil
			}
			path := p.curToken.Literal
			stmt.Mappings[alias] = path
		}
		p.nextToken()
	}
	return stmt
}

// mod name
func (p *Parser) parseModStatement() *ast.ModStatement {
	stmt := &ast.ModStatement{Position: p.curToken.Pos}
	if !p.expectPeek(lexer.IDENT) {
		return nil
	}
	stmt.Name = p.curToken.Literal
	return stmt
}

// from "services" use auth as a
// from "services" use "auth"
// from "services" use auth::b
// from "services" use (auth, cdn, db)
func (p *Parser) parseFromUseStatement() *ast.FromUseStatement {
	stmt := &ast.FromUseStatement{Position: p.curToken.Pos}
	if !p.expectPeek(lexer.STRING) && !p.expectPeek(lexer.IDENT) {
		return nil
	}
	stmt.Source = p.curToken.Literal

	if !p.expectPeek(lexer.USE) {
		return nil
	}

	if p.peekTokenIs(lexer.LPAREN) {
		p.nextToken() // saute '('
		p.nextToken()
		for !p.curTokenIs(lexer.RPAREN) && !p.curTokenIs(lexer.EOF) {
			if p.curTokenIs(lexer.IDENT) || p.curTokenIs(lexer.STRING) {
				cleanMod := p.curToken.Literal
				if len(cleanMod) >= 2 && cleanMod[0] == '"' && cleanMod[len(cleanMod)-1] == '"' {
					cleanMod = cleanMod[1 : len(cleanMod)-1]
				}
				stmt.Modules = append(stmt.Modules, cleanMod)
			}
			if p.peekTokenIs(lexer.COMMA) {
				p.nextToken()
			}
			p.nextToken()
		}
		return stmt
	}

	if p.peekTokenIs(lexer.IDENT) || p.peekTokenIs(lexer.STRING) {
		p.nextToken()
		cleanMod := p.curToken.Literal
		if len(cleanMod) >= 2 && cleanMod[0] == '"' && cleanMod[len(cleanMod)-1] == '"' {
			cleanMod = cleanMod[1 : len(cleanMod)-1]
		}
		stmt.Module = cleanMod
	} else {
		p.expectPeek(lexer.IDENT)
		return nil
	}

	if p.peekTokenIs(lexer.DOUBLE_COLON) {
		p.nextToken() // saute '::'
		if p.peekTokenIs(lexer.LPAREN) {
			p.nextToken() // saute '('
			p.nextToken() // premier token dans '('
			for !p.curTokenIs(lexer.RPAREN) && !p.curTokenIs(lexer.EOF) {
				if p.curTokenIs(lexer.IDENT) {
					stmt.Members = append(stmt.Members, p.curToken.Literal)
				}
				if p.peekTokenIs(lexer.COMMA) {
					p.nextToken()
				}
				p.nextToken()
			}
		} else if p.expectPeek(lexer.IDENT) {
			stmt.Member = p.curToken.Literal
		} else {
			return nil
		}
	}

	if p.peekTokenIs(lexer.AS) {
		p.nextToken() // saute 'as'
		if !p.expectPeek(lexer.IDENT) {
			return nil
		}
		stmt.Alias = p.curToken.Literal
	}
	return stmt
}

