package parser

import (
	"fmt"
	"strings"

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
	if p.peekTokenIs(lexer.SEMICOLON) {
		p.nextToken()
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
		stmt.Type = p.parseTypeAnnotation()
	}

	if p.peekTokenIs(lexer.OR) || p.peekTokenIs(lexer.ASSIGN) {
		p.nextToken() // mange 'or' ou '='
		p.nextToken() // début de l'expression par défaut
		stmt.DefaultValue = p.parseExpression()
	}
	if p.peekTokenIs(lexer.SEMICOLON) {
		p.nextToken()
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
		stmt.Type = p.parseTypeAnnotation()
	}

	if !p.expectPeek(lexer.ASSIGN) {
		return nil
	}

	assignPos := p.curToken.Pos
	if p.peekToken.Pos.Line != assignPos.Line || p.peekTokenIs(lexer.EOF) || p.peekTokenIs(lexer.RBRACE) || p.peekTokenIs(lexer.SEMICOLON) {
		p.addError(assignPos, "expected expression after '=' in variable declaration")
		return stmt
	}

	p.nextToken() // début de la valeur
	stmt.Value = p.parseExpression()
	if stmt.Value == nil {
		p.addError(assignPos, "expected expression after '=' in variable declaration")
		return stmt
	}
	if p.peekTokenIs(lexer.SEMICOLON) {
		p.nextToken()
	} else if p.peekToken.Pos.Line == p.curToken.Pos.Line && !p.peekTokenIs(lexer.EOF) && !p.peekTokenIs(lexer.RBRACE) && !p.peekTokenIs(lexer.COMMENT) {
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
		if p.peekTokenIs(lexer.AT) {
			p.nextToken() // saute '@'
			if !p.peekTokenIs(lexer.MAP) && !(p.peekTokenIs(lexer.IDENT) && strings.ToLower(p.peekToken.Literal) == "map") {
				p.addError(p.curToken.Pos, "expected 'map' after '@'")
				return nil
			}
			p.nextToken() // saute 'map'
			stmt.AsMap = true
			if p.peekTokenIs(lexer.LBRACE) {
				p.nextToken() // saute '{'
				p.nextToken()
				for !p.curTokenIs(lexer.RBRACE) && !p.curTokenIs(lexer.EOF) {
					if p.curTokenIs(lexer.IDENT) {
						stmt.Members = append(stmt.Members, p.curToken.Literal)
					}
					if p.peekTokenIs(lexer.COMMA) || p.peekTokenIs(lexer.SEMICOLON) {
						p.nextToken()
					}
					p.nextToken()
				}
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

// interface Name @map { field: type, ... } ou type Name = @map { ... }
func (p *Parser) parseInterfaceStatement() *ast.InterfaceStatement {
	stmt := &ast.InterfaceStatement{Position: p.curToken.Pos}
	if p.peekTokenIs(lexer.IDENT) || p.peekTokenIs(lexer.MAP) {
		p.nextToken()
		stmt.Name = p.curToken.Literal
	} else {
		p.expectPeek(lexer.IDENT)
		return nil
	}

	// 'type Name = ...' : saute '='
	if p.peekTokenIs(lexer.ASSIGN) {
		p.nextToken()
	}

	// Support de '@map' (ex: type Name = @map { ... } ou interface Name @map { ... })
	if p.peekTokenIs(lexer.AT) {
		p.nextToken() // curToken devient '@'
		if !p.expectPeek(lexer.MAP) && !(p.curTokenIs(lexer.IDENT) && strings.ToLower(p.curToken.Literal) == "map") {
			p.addError(p.curToken.Pos, "type definition must specify '@map'")
			return nil
		}
		stmt.IsMap = true
	}

	if !p.expectPeek(lexer.LBRACE) {
		return nil
	}

	stmt.Fields = p.parseInterfaceFields()
	return stmt
}

func (p *Parser) parseInterfaceFields() []*ast.InterfaceField {
	var fields []*ast.InterfaceField
	p.nextToken() // saute '{'

	for !p.curTokenIs(lexer.RBRACE) && !p.curTokenIs(lexer.EOF) {
		if p.curTokenIs(lexer.SEMICOLON) {
			p.nextToken()
			continue
		}
		if p.curTokenIs(lexer.FUNC) {
			fnPos := p.curToken.Pos
			if p.expectPeek(lexer.IDENT) {
				methodName := p.curToken.Literal
				params := p.parseParameterList()
				retType := "void"
				if p.peekTokenIs(lexer.ARROW) || p.peekTokenIs(lexer.COLON) {
					p.nextToken() // saute '->' ou ':'
					retType = p.parseTypeAnnotation()
				}
				var paramSigs []string
				for _, param := range params {
					sig := param.Name
					if param.Optional {
						sig += "?"
					}
					if param.Type != "" {
						sig += ": " + param.Type
					}
					paramSigs = append(paramSigs, sig)
				}
				fnType := fmt.Sprintf("func(%s) -> %s", strings.Join(paramSigs, ", "), retType)
				fields = append(fields, &ast.InterfaceField{
					Position:   fnPos,
					Name:       methodName,
					Type:       fnType,
					Params:     params,
					ReturnType: retType,
				})
			}
		} else if p.curTokenIs(lexer.IDENT) {
			field := &ast.InterfaceField{
				Position: p.curToken.Pos,
				Name:     p.curToken.Literal,
			}
			if p.peekTokenIs(lexer.QUESTION) {
				field.Optional = true
				p.nextToken() // saute '?'
			}

			// Pas de ':' dans un block d'interface : séparation par espace
			if p.peekTokenIs(lexer.COLON) {
				p.addError(p.peekToken.Pos, "colons ':' are not allowed in interface fields; separate field name and type with whitespace (e.g. 'name type')")
				p.nextToken() // saute ':' pour continuer
			}

			// Type du champ
			field.Type = p.parseTypeAnnotation()
			if strings.HasSuffix(field.Type, "?") {
				field.Optional = true
				field.Type = strings.TrimSuffix(field.Type, "?")
			}
			fields = append(fields, field)
		}
		if p.peekTokenIs(lexer.COMMA) || p.peekTokenIs(lexer.SEMICOLON) {
			p.nextToken()
		}
		p.nextToken()
	}
	return fields
}

