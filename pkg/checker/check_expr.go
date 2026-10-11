package checker

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/Nehonix-Team/xru/pkg/ast"
)

var (
	durationRegex             = regexp.MustCompile(`^[0-9]+(\.[0-9]+)?(ms|s|m|h)$`)
	reUnitPropInterpolation   = regexp.MustCompile(`\{&([a-zA-Z_][a-zA-Z0-9_]*)\}`)
	reInterpolationIdentifier = regexp.MustCompile(`\{([a-zA-Z_][a-zA-Z0-9_]*)\}`)
)

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

	case *ast.StringLiteral:
		if !c.scope.InUnitScope() {
			matches := reUnitPropInterpolation.FindAllStringSubmatch(e.Value, -1)
			for _, m := range matches {
				if len(m) > 1 {
					varPos := e.Pos()
					if idx := strings.Index(e.Value, "{&"+m[1]+"}"); idx >= 0 {
						varPos.Column += 1 + idx + 1
					}
					c.addError(varPos, "ERR_INVALID_PROP_ACCESS",
						fmt.Sprintf("Property access '&%s' is only allowed inside a #UNIT definition", m[1]))
				}
			}
		} else {
			matches := reUnitPropInterpolation.FindAllStringSubmatch(e.Value, -1)
			for _, m := range matches {
				if len(m) > 1 {
					propName := m[1]
					if _, ok := c.scope.Resolve("&" + propName); !ok {
						if _, ok2 := c.scope.Resolve(propName); !ok2 {
							varPos := e.Pos()
							if idx := strings.Index(e.Value, "{&"+propName+"}"); idx >= 0 {
								varPos.Column += 1 + idx + 1
							}
							c.addError(varPos, "ERR_UNDEFINED_PROPERTY",
								fmt.Sprintf("Property '&%s' is not defined in this unit", propName))
						}
					}
				}
			}
		}
		masked := strings.ReplaceAll(e.Value, "{{", "\x00\x00")
		masked = strings.ReplaceAll(masked, "}}", "\x01\x01")
		for _, m := range reInterpolationIdentifier.FindAllStringSubmatch(masked, -1) {
			if len(m) > 1 {
				varName := m[1]
				if sym, ok := c.scope.Resolve(varName); ok {
					sym.Used = true
				} else {
					varPos := e.Pos()
					if idx := strings.Index(e.Value, "{"+varName+"}"); idx >= 0 {
						varPos.Column += 1 + idx + 1
					}
					c.addError(varPos, "ERR_UNDEFINED_VARIABLE",
						fmt.Sprintf("Undefined variable '%s' in string interpolation", varName))
				}
			}
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
				if sym, ok := c.scope.Resolve(ident.Value); ok {
					sym.Used = true
					if strings.HasPrefix(sym.Type, "task") {
						propLower := strings.ToLower(member.Property)
						if propLower != "wait" && propLower != "await" && propLower != "result" &&
							propLower != "isdone" && propLower != "done" && propLower != "cancel" {
							c.addError(e.Pos(), "ERR_UNKNOWN_METHOD",
								fmt.Sprintf("Method '%s' not found on task", member.Property))
						}
					} else {
						var unitSym *Symbol
						if sym.Kind == SymUnit {
							unitSym = sym
						} else if sym.Type != "" {
							if uSym, ok := c.scope.Resolve(sym.Type); ok && uSym.Kind == SymUnit {
								unitSym = uSym
							}
						}
						if unitSym != nil && unitSym.Methods != nil {
							if targetMethod, methodOk := unitSym.Methods[member.Property]; !methodOk {
								c.addError(e.Pos(), "ERR_UNKNOWN_METHOD",
									fmt.Sprintf("Method '%s' not found on unit '%s'", member.Property, unitSym.Name))
							} else if targetMethod.Kind == SymFunc {
								c.validateCallArgs(e, member.Property, targetMethod.Params)
							}
						}
					}
				}
			}
		} else if scopeAccess, ok := e.Caller.(*ast.ScopeAccessExpression); ok {
			if sym, ok := c.scope.Resolve(scopeAccess.Scope); ok {
				sym.Used = true
				if sym.Kind == SymUnit && len(sym.Methods) > 0 {
					if targetMethod, methodOk := sym.Methods[scopeAccess.Member]; !methodOk {
						c.addError(e.Pos(), "ERR_UNKNOWN_METHOD",
							fmt.Sprintf("Method '%s' not found on module '%s'", scopeAccess.Member, sym.Name))
					} else if targetMethod.Kind == SymFunc {
						c.validateCallArgs(e, scopeAccess.Member, targetMethod.Params)
					}
				}
			} else {
				c.addError(e.Pos(), "ERR_UNDEFINED_VARIABLE",
					fmt.Sprintf("Module or scope '%s' is not defined", scopeAccess.Scope))
			}
		} else if ident, ok := e.Caller.(*ast.Identifier); ok {
			if sym, ok := c.scope.Resolve(ident.Value); ok {
				sym.Used = true
				if sym.Kind == SymFunc {
					c.validateCallArgs(e, ident.Value, sym.Params)
				}
			} else {
				c.addError(e.Pos(), "ERR_UNDEFINED_VARIABLE",
					fmt.Sprintf("Function '%s' is not defined", ident.Value))
			}
		} else {
			c.checkExpression(e.Caller)
		}
		for _, arg := range e.Arguments {
			c.checkExpression(arg)
		}

	case *ast.ScopeAccessExpression:
		if sym, ok := c.scope.Resolve(e.Scope); ok {
			sym.Used = true
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
		if sym, ok := c.scope.Resolve(e.Value); !ok {
			c.addError(e.Pos(), "ERR_UNDEFINED_VARIABLE",
				fmt.Sprintf("Variable '%s' is not defined", e.Value))
		} else {
			sym.Used = true
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

	case *ast.IfExpression:
		c.checkExpression(e.Condition)
		c.checkExpression(e.Consequence)
		c.checkExpression(e.Alternative)

	case *ast.WithExpression:
		c.checkExpression(e.Target)
		switch t := e.Target.(type) {
		case *ast.Identifier:
			if sym, ok := c.scope.Resolve(t.Value); ok && sym.Kind == SymUnit {
				c.addError(e.Pos(), "ERR_INVALID_WITH_TARGET",
					fmt.Sprintf("Cannot use module '%s' directly as with target: expected a method or function call (e.g. with %s.fetch(...))", sym.Name, sym.Name))
			}
		}

	case *ast.RangeExpression:
		c.checkExpression(e.Start)
		c.checkExpression(e.End)

	case *ast.CatchExpression:
		if e.ErrVar != "" {
			errSym := &Symbol{
				Name: e.ErrVar,
				Kind: SymVar,
				Type: "string",
				Pos:  e.Pos(),
			}
			if _, exists := c.scope.symbols[e.ErrVar]; !exists {
				_ = c.scope.Define(errSym)
			}
		}
		c.checkExpression(e.Target)
		if e.Fallback != nil {
			if expr, ok := e.Fallback.(ast.Expression); ok {
				c.checkExpression(expr)
			} else if stmt, ok := e.Fallback.(ast.Statement); ok {
				c.checkStatement(stmt)
			}
		}
	}
}

func (c *Checker) validateCallArgs(call *ast.CallExpression, name string, params []*ast.Parameter) {
	if params == nil {
		return
	}
	expectedCount := len(params)
	actualCount := len(call.Arguments)
	if actualCount != expectedCount {
		c.addError(call.Pos(), "ERR_ARG_COUNT",
			fmt.Sprintf("Function '%s' expects %d arguments, but got %d", name, expectedCount, actualCount))
		return
	}
	for i, param := range params {
		if param.Type != "" && param.Type != "any" {
			argType := c.inferType(call.Arguments[i])
			if !isTypeCompatible(param.Type, argType) {
				c.addError(call.Arguments[i].Pos(), "ERR_TYPE_MISMATCH",
					fmt.Sprintf("Argument %d ('%s') of function '%s' expects type '%s', got '%s'", i+1, param.Name, name, param.Type, argType))
			}
		}
	}
}
