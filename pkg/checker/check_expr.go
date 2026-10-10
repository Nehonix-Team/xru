package checker

import (
	"fmt"
	"regexp"

	"github.com/Nehonix-Team/xru/pkg/ast"
)

var durationRegex = regexp.MustCompile(`^[0-9]+(\.[0-9]+)?(ms|s|m|h)$`)

func (c *Checker) checkExpression(expr ast.Expression) {
	if expr == nil {
		return
	}

	switch e := expr.(type) {
	case *ast.LambdaExpression:
		lambdaScope := NewScope(c.scope, false)
		c.scope = lambdaScope
		defer func() { c.scope = c.scope.parent }()

		for _, p := range e.Parameters {
			paramSym := &Symbol{Name: p.Name, Kind: SymVar, Type: p.Type, Pos: e.Pos()}
			if err := c.scope.Define(paramSym); err != nil {
				c.addError(e.Pos(), "ERR_DUPLICATE_SYMBOL", err.Error())
			}
		}
		c.checkExpression(e.Body)

	case *ast.PropAccess:
		if !c.scope.InUnitScope() {
			c.addError(e.Pos(), "ERR_INVALID_PROP_ACCESS",
				fmt.Sprintf("Property access '&%s' is only allowed inside a #UNIT definition", e.Name))
		}

	case *ast.DurationLiteral:
		if !durationRegex.MatchString(e.Value) {
			c.addError(e.Pos(), "ERR_INVALID_DURATION",
				fmt.Sprintf("Invalid duration format '%s' (expected format: 500ms, 30s, 2m, 1h)", e.Value))
		}

	case *ast.CallExpression:
		if member, ok := e.Caller.(*ast.MemberExpression); ok {
			c.checkExpression(member.Object)
			if ident, ok := member.Object.(*ast.Identifier); ok {
				if sym, ok := c.scope.Resolve(ident.Value); ok && sym.Kind == SymUnit {
					if sym.Methods != nil {
						if _, methodOk := sym.Methods[member.Property]; !methodOk {
							c.addError(e.Pos(), "ERR_UNKNOWN_METHOD",
								fmt.Sprintf("Method '%s' not found on unit '%s'", member.Property, sym.Name))
						}
					}
				}
			}
		} else if scopeAccess, ok := e.Caller.(*ast.ScopeAccessExpression); ok {
			if sym, ok := c.scope.Resolve(scopeAccess.Scope); ok {
				if sym.Kind == SymUnit && len(sym.Methods) > 0 {
					if _, methodOk := sym.Methods[scopeAccess.Member]; !methodOk {
						c.addError(e.Pos(), "ERR_UNKNOWN_METHOD",
							fmt.Sprintf("Method '%s' not found on module '%s'", scopeAccess.Member, sym.Name))
					}
				}
			} else {
				c.addError(e.Pos(), "ERR_UNDEFINED_VARIABLE",
					fmt.Sprintf("Module or scope '%s' is not defined", scopeAccess.Scope))
			}
		} else {
			c.checkExpression(e.Caller)
		}
		for _, arg := range e.Arguments {
			c.checkExpression(arg)
		}

	case *ast.ScopeAccessExpression:
		if sym, ok := c.scope.Resolve(e.Scope); ok {
			if sym.Kind == SymUnit && len(sym.Methods) > 0 {
				if _, memberOk := sym.Methods[e.Member]; !memberOk {
					c.addError(e.Pos(), "ERR_UNKNOWN_MEMBER",
						fmt.Sprintf("Member '%s' not found on module '%s'", e.Member, e.Scope))
				}
			}
		} else {
			c.addError(e.Pos(), "ERR_UNDEFINED_VARIABLE",
				fmt.Sprintf("Module or scope '%s' is not defined", e.Scope))
		}

	case *ast.MemberExpression:
		c.checkExpression(e.Object)
		if ident, ok := e.Object.(*ast.Identifier); ok {
			if sym, ok := c.scope.Resolve(ident.Value); ok && sym.Kind == SymUnit {
				if sym.Methods != nil {
					if _, methodOk := sym.Methods[e.Property]; !methodOk {
						if _, dirOk := sym.Directives[e.Property]; !dirOk {
							c.addError(e.Pos(), "ERR_UNKNOWN_METHOD",
								fmt.Sprintf("Method or property '%s' not found on unit '%s'", e.Property, sym.Name))
						}
					}
				}
			}
		}

	case *ast.PrefixExpression:
		c.checkExpression(e.Right)

	case *ast.InfixExpression:
		c.checkExpression(e.Left)
		c.checkExpression(e.Right)

	case *ast.Identifier:
		if e.Value == "nil" || e.Value == "null" || e.Value == "true" || e.Value == "false" {
			return
		}
		if _, ok := c.scope.Resolve(e.Value); !ok {
			c.addError(e.Pos(), "ERR_UNDEFINED_VARIABLE",
				fmt.Sprintf("Variable '%s' is not defined", e.Value))
		}

	case *ast.ListLiteral:
		for _, el := range e.Elements {
			c.checkExpression(el)
		}

	case *ast.MapLiteral:
		for _, pair := range e.Pairs {
			c.checkExpression(pair.Key)
			c.checkExpression(pair.Value)
		}

	case *ast.BlockStatement:
		for _, stmt := range e.Statements {
			c.checkStatement(stmt)
		}
	}
}
