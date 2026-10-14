package checker

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/Nehonix-Team/xru/pkg/ast"
	"github.com/Nehonix-Team/xru/pkg/lexer"
	"github.com/Nehonix-Team/xru/pkg/parser"
)

var (
	durationRegex = regexp.MustCompile(`^[0-9]+(\.[0-9]+)?(ms|s|m|h)$`)
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
		spans := ast.ExtractInterpolations(e.Value)
		for _, span := range spans {
			if span.Expr == "" {
				continue
			}
			varPos := e.Pos()
			varPos.Column += 1 + span.StartIdx + 3

			if strings.HasPrefix(span.Expr, "&") {
				propName := strings.TrimPrefix(span.Expr, "&")
				if !c.scope.InUnitScope() {
					c.addError(varPos, "ERR_INVALID_PROP_ACCESS",
						fmt.Sprintf("Property access '&%s' is only allowed inside a #UNIT definition", propName))
				} else {
					if _, ok := c.scope.Resolve("&" + propName); !ok {
						if _, ok2 := c.scope.Resolve(propName); !ok2 {
							c.addError(varPos, "ERR_UNDEFINED_PROPERTY",
								fmt.Sprintf("Property '&%s' is not defined in this unit", propName))
						}
					}
				}
				continue
			}

			// Analyse et validation de l'expression d'interpolation
			subLexer := lexer.New(e.Pos().Filename, span.Expr)
			subParser := parser.New(subLexer)
			parsedExpr := subParser.ParseExpression()
			if len(subParser.Errors()) > 0 {
				c.addError(varPos, "ERR_INVALID_INTERPOLATION",
					fmt.Sprintf("Syntax error in string interpolation '$f{%s}': %s", span.Expr, subParser.Errors()[0]))
				continue
			}

			errCountBefore := len(c.errors)
			c.checkExpression(parsedExpr)
			for i := errCountBefore; i < len(c.errors); i++ {
				errStr := c.errors[i]
				idx := strings.Index(errStr, ": [")
				if idx != -1 {
					codeAndMsg := errStr[idx+2:]
					c.errors[i] = fmt.Sprintf("%s: %s", varPos, codeAndMsg)
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
			objType := c.inferType(member.Object)
			if objType == "string" {
				if spec, ok := GetStringMethodSpec(member.Property); ok {
					c.validateNativeMethodCall(e, "string", member.Property, spec)
				} else {
					c.addError(e.Pos(), "ERR_UNKNOWN_METHOD",
						fmt.Sprintf("Method '%s' not found on string", member.Property))
					for _, arg := range e.Arguments {
						c.checkExpression(arg)
					}
				}
				return
			} else if strings.HasPrefix(objType, "list") {
				if spec, ok := GetListMethodSpec(member.Property); ok {
					c.validateNativeMethodCall(e, "list", member.Property, spec)
				} else {
					c.addError(e.Pos(), "ERR_UNKNOWN_METHOD",
						fmt.Sprintf("Method '%s' not found on list", member.Property))
					for _, arg := range e.Arguments {
						c.checkExpression(arg)
					}
				}
				return
			}

			if ident, ok := member.Object.(*ast.Identifier); ok {
				if sym, ok := c.scope.Resolve(ident.Value); ok {
					sym.Used = true
					if sym.Kind == SymUnit && !IsBuiltinUnit(sym.Name) && !IsBuiltinUnit(sym.Type) {
						c.addError(e.Pos(), "ERR_CANNOT_CALL_STATIC",
							fmt.Sprintf("Cannot call method '%s' directly on unit definition '%s'. You must first instantiate it: 'let instance = @%s()'.", member.Property, sym.Name, sym.Name))
						return
					}
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
						} else if sym.Methods != nil {
							unitSym = sym
						} else if sym.Type != "" {
							if uSym, ok := c.scope.Resolve(sym.Type); ok && uSym.Kind == SymUnit {
								unitSym = uSym
							}
						}
						if unitSym != nil && unitSym.Methods != nil {
							targetMethod, methodOk := unitSym.Methods[member.Property]
							if !methodOk {
								for mName, mSym := range unitSym.Methods {
									if strings.EqualFold(mName, member.Property) {
										targetMethod = mSym
										methodOk = true
										break
									}
								}
							}
							if !methodOk {
								c.addError(e.Pos(), "ERR_UNKNOWN_METHOD",
									fmt.Sprintf("Method '%s' not found on unit '%s'", member.Property, unitSym.Name))
							} else if targetMethod.Kind == SymFunc {
								c.validateCallArgs(e, member.Property, targetMethod)
							}
						}
					}
				}
			}
		} else if scopeAccess, ok := e.Caller.(*ast.ScopeAccessExpression); ok {
			if sym, ok := c.scope.Resolve(scopeAccess.Scope); ok {
				sym.Used = true
				if sym.Kind == SymUnit && len(sym.Methods) > 0 {
					targetMethod, methodOk := sym.Methods[scopeAccess.Member]
					if !methodOk {
						for mName, mSym := range sym.Methods {
							if strings.EqualFold(mName, scopeAccess.Member) {
								targetMethod = mSym
								methodOk = true
								break
							}
						}
					}
					if !methodOk {
						c.addError(e.Pos(), "ERR_UNKNOWN_METHOD",
							fmt.Sprintf("Method '%s' not found on module '%s'", scopeAccess.Member, sym.Name))
					} else if targetMethod.Kind == SymFunc {
						c.validateCallArgs(e, scopeAccess.Member, targetMethod)
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
					c.validateCallArgs(e, ident.Value, sym)
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

	case *ast.GlobalCallExpression:
		for _, arg := range e.Arguments {
			c.checkExpression(arg)
		}
		if sym, ok := c.scope.Resolve(e.Function); ok {
			sym.Used = true
		}

	case *ast.ScopeAccessExpression:
		if sym, ok := c.scope.Resolve(e.Scope); ok {
			sym.Used = true
			if sym.Kind == SymUnit && len(sym.Methods) > 0 {
				if methodSym, memberOk := sym.Methods[e.Member]; !memberOk {
					c.addError(e.Pos(), "ERR_UNKNOWN_MEMBER",
						fmt.Sprintf("Member '%s' not found on module '%s'", e.Member, e.Scope))
				} else {
					methodSym.Used = true
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
				if !IsBuiltinUnit(sym.Name) && !IsBuiltinUnit(sym.Type) {
					c.addError(e.Pos(), "ERR_INVALID_MODULE_ACCESS",
						fmt.Sprintf("Cannot access member '%s' on module '%s' with '.': use '::' syntax instead (e.g. %s::%s)", e.Property, sym.Name, sym.Name, e.Property))
					return
				}
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
		c.checkStatementList(e.Statements)

	case *ast.IfExpression:
		c.checkExpression(e.Condition)
		c.checkExpression(e.Consequence)
		c.checkExpression(e.Alternative)

	case *ast.WithExpression:
		if _, isCatch := e.Target.(*ast.CatchExpression); isCatch {
			c.addError(e.Pos(), "ERR_CATCH_ON_TASK",
				"'catch' on a 'with' expression never captures async errors in the caller: the task is still running. Use 'task.wait() catch err' instead")
		}
		c.checkExpression(e.Target)
		switch t := e.Target.(type) {
		case *ast.Identifier:
			if sym, ok := c.scope.Resolve(t.Value); ok && sym.Kind == SymUnit {
				c.addError(e.Pos(), "ERR_INVALID_WITH_TARGET",
					fmt.Sprintf("Cannot use module '%s' directly as with target: expected a method or function call (e.g. with %s.fetch(...))", sym.Name, sym.Name))
			}
		}

	case *ast.InstantiationExpression:
		if IsBuiltinUnit(e.UnitName) {
			c.addError(e.Pos(), "ERR_CANNOT_INSTANTIATE_BUILTIN",
				fmt.Sprintf("Built-in module '%s' cannot be instantiated with '@'. Use '%s.<method>()' directly.", e.UnitName, e.UnitName))
			return
		}
		if sym, ok := c.scope.Resolve(e.UnitName); ok {
			sym.Used = true
			if sym.Kind != SymUnit {
				c.addError(e.Pos(), "ERR_NOT_A_UNIT",
					fmt.Sprintf("'%s' is not a unit and cannot be instantiated", e.UnitName))
			}
		} else {
			c.addError(e.Pos(), "ERR_UNDEFINED_UNIT",
				fmt.Sprintf("Unit '%s' is not defined", e.UnitName))
		}

	case *ast.RangeExpression:
		c.checkExpression(e.Start)
		c.checkExpression(e.End)

	case *ast.CatchExpression:
		if _, isWith := e.Target.(*ast.WithExpression); isWith {
			c.addError(e.Pos(), "ERR_CATCH_ON_TASK",
				"'catch' on a 'with' expression never captures async errors: the task is still running. Use 'task.wait() catch err' instead")
		}
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

func (c *Checker) validateCallArgs(call *ast.CallExpression, name string, sym *Symbol) {
	if sym == nil || (sym.Params == nil && sym.MinArgs == 0 && !sym.IsVariadic) {
		return
	}
	actualCount := len(call.Arguments)
	expectedCount := len(sym.Params)
	minArgs := expectedCount
	if sym.MinArgs > 0 || (sym.MinArgs == 0 && sym.IsVariadic) {
		minArgs = sym.MinArgs
	}

	if sym.IsVariadic {
		if actualCount < minArgs {
			c.addError(call.Pos(), "ERR_ARG_COUNT",
				fmt.Sprintf("Function '%s' expects at least %d argument(s), but got %d", name, minArgs, actualCount))
			return
		}
	} else if sym.HasOptional {
		if actualCount < sym.MinArgs || actualCount > expectedCount {
			c.addError(call.Pos(), "ERR_ARG_COUNT",
				fmt.Sprintf("Function '%s' expects %d to %d argument(s), but got %d", name, sym.MinArgs, expectedCount, actualCount))
			return
		}
	} else {
		if actualCount != expectedCount {
			c.addError(call.Pos(), "ERR_ARG_COUNT",
				fmt.Sprintf("Function '%s' expects %d argument(s), but got %d", name, expectedCount, actualCount))
			return
		}
	}

	for i, param := range sym.Params {
		if i < actualCount && param.Type != "" && param.Type != "any" {
			argType := c.inferType(call.Arguments[i])
			if argType != "any" && !isTypeCompatible(param.Type, argType) {
				c.addError(call.Arguments[i].Pos(), "ERR_TYPE_MISMATCH",
					fmt.Sprintf("Argument %d ('%s') of function '%s' expects type '%s', got '%s'", i+1, param.Name, name, param.Type, argType))
			}
		}
	}
}

func (c *Checker) validateNativeMethodCall(call *ast.CallExpression, receiverType string, methodName string, spec NativeMethodSpec) {
	for _, arg := range call.Arguments {
		c.checkExpression(arg)
	}

	minArgs := 0
	maxArgs := len(spec.Params)
	for _, p := range spec.Params {
		if !p.Optional {
			minArgs++
		}
	}

	actualCount := len(call.Arguments)
	if actualCount < minArgs || actualCount > maxArgs {
		if minArgs == maxArgs {
			c.addError(call.Pos(), "ERR_ARG_COUNT",
				fmt.Sprintf("Method '%s' on %s expects %d argument(s), but got %d", methodName, receiverType, minArgs, actualCount))
		} else {
			c.addError(call.Pos(), "ERR_ARG_COUNT",
				fmt.Sprintf("Method '%s' on %s expects %d to %d argument(s), but got %d", methodName, receiverType, minArgs, maxArgs, actualCount))
		}
		return
	}

	for i, arg := range call.Arguments {
		param := spec.Params[i]
		if param.Type != "" && param.Type != "any" {
			argType := c.inferType(arg)
			if argType != "any" && !isTypeCompatible(param.Type, argType) {
				c.addError(arg.Pos(), "ERR_TYPE_MISMATCH",
					fmt.Sprintf("Argument %d ('%s') of method '%s' expects type '%s', got '%s'", i+1, param.Name, methodName, param.Type, argType))
			}
		}
	}
}
