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
			if p.Type != "" {
				c.validateTypeAnnotation(e.Pos(), p.Type)
			}
			paramSym := &Symbol{Name: p.Name, Kind: SymVar, Type: p.Type, Pos: e.Pos()}
			if err := c.scope.Define(paramSym); err != nil {
				c.addError(e.Pos(), "ERR_DUPLICATE_SYMBOL", err.Error())
			}
		}
		c.checkExpression(e.Body)

	case *ast.PropAccess:
		if !c.scope.InUnitScope() {
			c.addError(e.Pos(), "ERR_INVALID_PROP_ACCESS",
				fmt.Sprintf("Property or method access '&%s' is only allowed inside a unit definition", e.Name))
		} else {
			if _, ok := c.scope.Resolve("&" + e.Name); !ok {
				if _, ok2 := c.scope.Resolve(e.Name); !ok2 {
					unitSym := c.scope.GetEnclosingUnit()
					var found bool
					if unitSym != nil && unitSym.Methods != nil {
						if _, ok3 := unitSym.Methods[e.Name]; ok3 {
							found = true
						} else {
							for mName := range unitSym.Methods {
								if strings.EqualFold(mName, e.Name) {
									found = true
									break
								}
							}
						}
					}
					if !found {
						c.addError(e.Pos(), "ERR_UNDEFINED_PROPERTY",
							fmt.Sprintf("Property or method '&%s' is not defined in this unit", e.Name))
					}
				}
			}
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
						} else if iface, ok := c.interfaces[sym.Type]; ok {
							found := false
							for _, f := range iface.Fields {
								if f.Name == member.Property {
									found = true
									break
								}
							}
							if !found {
								c.addError(e.Pos(), "ERR_NO_SUCH_PROPERTY",
									fmt.Sprintf("Property or method '%s' does not exist on type '%s'", member.Property, sym.Type))
							}
						}
					}
				}
			}
		} else if scopeAccess, ok := e.Caller.(*ast.ScopeAccessExpression); ok {
			if scopeAccess.IsUnitSelf {
				if !c.scope.InUnitScope() {
					c.addError(e.Pos(), "ERR_INVALID_PROP_ACCESS",
						fmt.Sprintf("Internal unit reference '&%s::%s' is only allowed inside a unit definition", scopeAccess.Scope, scopeAccess.Member))
				} else {
					unitSym := c.scope.GetEnclosingUnit()
					if unitSym != nil && scopeAccess.Scope != "self" && !strings.EqualFold(scopeAccess.Scope, unitSym.Name) {
						c.addError(e.Pos(), "ERR_INVALID_UNIT_SELF_REF",
							fmt.Sprintf("Cannot reference '&%s::%s' inside unit '%s'; use '&%s::%s' or '&self::%s'", scopeAccess.Scope, scopeAccess.Member, unitSym.Name, unitSym.Name, scopeAccess.Member, scopeAccess.Member))
					} else if unitSym != nil && unitSym.Methods != nil {
						targetMethod, methodOk := unitSym.Methods[scopeAccess.Member]
						if !methodOk {
							for mName, mSym := range unitSym.Methods {
								if strings.EqualFold(mName, scopeAccess.Member) {
									targetMethod = mSym
									methodOk = true
									break
								}
							}
						}
						if !methodOk {
							c.addError(e.Pos(), "ERR_UNKNOWN_METHOD",
								fmt.Sprintf("Method '%s' not found on unit '%s'", scopeAccess.Member, unitSym.Name))
						} else if targetMethod.Kind == SymFunc {
							c.validateCallArgs(e, scopeAccess.Member, targetMethod)
						}
					}
				}
			} else if sym, ok := c.scope.Resolve(scopeAccess.Scope); ok {
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
		} else if propAccess, ok := e.Caller.(*ast.PropAccess); ok {
			if !c.scope.InUnitScope() {
				c.addError(e.Pos(), "ERR_INVALID_PROP_ACCESS",
					fmt.Sprintf("Method access '&%s' is only allowed inside a unit definition", propAccess.Name))
			} else {
				unitSym := c.scope.GetEnclosingUnit()
				var targetMethod *Symbol
				if unitSym != nil && unitSym.Methods != nil {
					targetMethod = unitSym.Methods[propAccess.Name]
					if targetMethod == nil {
						for mName, mSym := range unitSym.Methods {
							if strings.EqualFold(mName, propAccess.Name) {
								targetMethod = mSym
								break
							}
						}
					}
				}
				if targetMethod == nil {
					c.addError(e.Pos(), "ERR_UNKNOWN_METHOD",
						fmt.Sprintf("Method '%s' not found on current unit", propAccess.Name))
				} else if targetMethod.Kind == SymFunc {
					c.validateCallArgs(e, propAccess.Name, targetMethod)
				}
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
		if e.IsUnitSelf {
			if !c.scope.InUnitScope() {
				c.addError(e.Pos(), "ERR_INVALID_PROP_ACCESS",
					fmt.Sprintf("Internal unit reference '&%s::%s' is only allowed inside a unit definition", e.Scope, e.Member))
			} else {
				unitSym := c.scope.GetEnclosingUnit()
				if unitSym != nil && e.Scope != "self" && !strings.EqualFold(e.Scope, unitSym.Name) {
					c.addError(e.Pos(), "ERR_INVALID_UNIT_SELF_REF",
						fmt.Sprintf("Cannot reference '&%s::%s' inside unit '%s'; use '&%s::%s' or '&self::%s'", e.Scope, e.Member, unitSym.Name, unitSym.Name, e.Member, e.Member))
				}
			}
		} else if sym, ok := c.scope.Resolve(e.Scope); ok {
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
		var objType string
		if ident, ok := e.Object.(*ast.Identifier); ok {
			if sym, ok := c.scope.Resolve(ident.Value); ok {
				sym.Used = true
				if sym.Kind == SymUnit {
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
					return
				}
				if sym.Type != "" && sym.Type != "any" {
					objType = sym.Type
				}
			}
		}
		if objType == "" {
			objType = c.inferType(e.Object)
		}
		objType = c.resolveType(objType)
		if objType != "" && objType != "any" && objType != "map" && !strings.HasPrefix(objType, "list") && objType != "string" {
			c.validatePropertyAccessOnType(e, objType)
		}

	case *ast.PrefixExpression:
		c.checkExpression(e.Right)

	case *ast.InfixExpression:
		c.checkExpression(e.Left)
		c.checkExpression(e.Right)

		if e.Operator == "==" || e.Operator == "!=" {
			leftType := c.inferType(e.Left)
			rightType := c.inferType(e.Right)

			if leftType != "any" && rightType != "any" && leftType != "nil" && rightType != "null" && leftType != "null" && rightType != "nil" {
				if !c.isTypeCompatible(leftType, rightType) && !c.isTypeCompatible(rightType, leftType) {
					alwaysVal := "false"
					if e.Operator == "!=" {
						alwaysVal = "true"
					}
					c.addError(e.Pos(), "ERR_TYPE_MISMATCH",
						fmt.Sprintf("Comparison condition will always be %s: types '%s' and '%s' have no overlap",
							alwaysVal, leftType, rightType))
				}
			}
		}

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
	minArgs := sym.MinArgs
	hasOptional := sym.HasOptional

	if !sym.IsVariadic && !hasOptional && expectedCount > 0 {
		calcMin := 0
		hasOpt := false
		for _, p := range sym.Params {
			if !p.Optional && !strings.HasSuffix(p.Type, "?") {
				calcMin++
			} else {
				hasOpt = true
			}
		}
		if hasOpt {
			hasOptional = true
			minArgs = calcMin
		}
	}
	if !hasOptional && !sym.IsVariadic && sym.MinArgs == 0 {
		minArgs = expectedCount
	}

	if sym.IsVariadic {
		if actualCount < minArgs {
			c.addError(call.Pos(), "ERR_ARG_COUNT",
				fmt.Sprintf("Function '%s' expects at least %d argument(s), but got %d", name, minArgs, actualCount))
			return
		}
	} else if hasOptional {
		if actualCount < minArgs || actualCount > expectedCount {
			c.addError(call.Pos(), "ERR_ARG_COUNT",
				fmt.Sprintf("Function '%s' expects %d to %d argument(s), but got %d", name, minArgs, expectedCount, actualCount))
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
		if i < actualCount {
			arg := call.Arguments[i]
			paramType := param.Type
			isOptional := param.Optional || strings.HasSuffix(paramType, "?")
			if isOptional && !strings.HasSuffix(paramType, "?") {
				paramType += "?"
			}
			resolvedParamType := c.resolveType(paramType)
			cleanParamType := strings.TrimSuffix(resolvedParamType, "?")

			// Si argument est nil/null et paramètre optionnel
			if isOptional {
				if ident, ok := arg.(*ast.Identifier); ok && (ident.Value == "nil" || ident.Value == "null") {
					continue
				}
			}

			// 1. Paramètre typé par une interface
			if iface, ok := c.interfaces[cleanParamType]; ok {
				c.validateArgAgainstInterface(arg, iface, param.Name, name, i+1, isOptional)
				continue
			}

			// 2. Paramètre typé par une map inline @map{ ... }
			if strings.HasPrefix(cleanParamType, "@map") && strings.Contains(cleanParamType, "{") {
				c.validateArgAgainstInlineMap(arg, cleanParamType, param.Name, name, i+1, isOptional)
				continue
			}

			// 3. Typage conventionnel
			if resolvedParamType != "" && resolvedParamType != "any" {
				argType := c.inferType(arg)
				if !c.isTypeCompatible(resolvedParamType, argType) {
					c.addError(arg.Pos(), "ERR_TYPE_MISMATCH",
						fmt.Sprintf("Argument %d ('%s') of function '%s' expects type '%s', got '%s'", i+1, param.Name, name, param.Type, argType))
				}
			}
		}
	}
}

func (c *Checker) validatePropertyAccessOnType(e *ast.MemberExpression, typeName string) {
	typeName = strings.TrimSpace(typeName)
	typeName = strings.TrimSuffix(typeName, "?")

	// 1. Interface
	if iface, ok := c.interfaces[typeName]; ok {
		for _, f := range iface.Fields {
			if f.Name == e.Property {
				return
			}
		}
		c.addError(e.Pos(), "ERR_NO_SUCH_PROPERTY",
			fmt.Sprintf("Property '%s' does not exist on type '%s'", e.Property, typeName))
		return
	}

	// 2. Inline map type: @map{ silent?: bool, retries: number }
	if strings.HasPrefix(typeName, "@map") && strings.Contains(typeName, "{") {
		fields := parseInlineMapFieldDefs(typeName)
		for _, f := range fields {
			if f.Name == e.Property {
				return
			}
		}
		c.addError(e.Pos(), "ERR_NO_SUCH_PROPERTY",
			fmt.Sprintf("Property '%s' does not exist on inline map type '%s'", e.Property, typeName))
		return
	}
}

func (c *Checker) validateArgAgainstInterface(arg ast.Expression, iface *ast.InterfaceStatement, paramName, funcName string, argIdx int, isOptional bool) {
	if isOptional {
		if ident, ok := arg.(*ast.Identifier); ok && (ident.Value == "nil" || ident.Value == "null") {
			return
		}
	}

	mapLit, isLit := arg.(*ast.MapLiteral)
	if isLit {
		keys := make(map[string]ast.Expression)
		for _, pair := range mapLit.Pairs {
			if strLit, ok := pair.Key.(*ast.StringLiteral); ok {
				keys[strLit.Value] = pair.Value
			} else if ident, ok := pair.Key.(*ast.Identifier); ok {
				keys[ident.Value] = pair.Value
			}
		}

		// Check for unknown properties not defined on the interface
		if len(iface.Fields) > 0 {
			for key, valExpr := range keys {
				found := false
				for _, f := range iface.Fields {
					if f.Name == key {
						found = true
						break
					}
				}
				if !found {
					c.addError(valExpr.Pos(), "ERR_UNKNOWN_PROPERTY",
						fmt.Sprintf("Property '%s' does not exist on type '%s'", key, iface.Name))
				}
			}
		}

		for _, f := range iface.Fields {
			valExpr, exists := keys[f.Name]
			if !f.Optional && !strings.HasSuffix(f.Type, "?") {
				if !exists {
					c.addError(arg.Pos(), "ERR_TYPE_MISMATCH",
						fmt.Sprintf("Argument %d ('%s') of function '%s' expects interface '%s', but map is missing required property '%s'",
							argIdx, paramName, funcName, iface.Name, f.Name))
					continue
				}
			}
			if exists {
				expectedFieldType := c.resolveType(f.Type)

				// 1. Si f.Type est une map inline ou interface et valExpr est un MapLiteral
				if mapLit, isMap := valExpr.(*ast.MapLiteral); isMap {
					if strings.HasPrefix(expectedFieldType, "@map") && strings.Contains(expectedFieldType, "{") {
						c.validateArgAgainstInlineMap(mapLit, expectedFieldType, f.Name, funcName, argIdx, f.Optional)
						continue
					}
					if c.interfaces != nil {
						if subIface, ok := c.interfaces[expectedFieldType]; ok {
							c.validateArgAgainstInterface(mapLit, subIface, f.Name, funcName, argIdx, f.Optional)
							continue
						}
					}
				}

				// 2. Si f.Type est une signature de fonction: func(...) -> ReturnType
				if lambda, isLambda := valExpr.(*ast.LambdaExpression); isLambda {
					if arrowIdx := strings.Index(f.Type, "->"); arrowIdx != -1 {
						expectedRet := strings.TrimSpace(f.Type[arrowIdx+2:])
						actualRet := "void"
						if lambda.Body != nil {
							if block, isBlock := lambda.Body.(*ast.BlockStatement); isBlock {
								for _, st := range block.Statements {
									if retSt, ok := st.(*ast.ReturnStatement); ok && retSt.Value != nil {
										actualRet = c.inferType(retSt.Value)
										break
									}
								}
							} else {
								actualRet = c.inferType(lambda.Body)
							}
						}
						if !c.isTypeCompatible(expectedRet, actualRet) {
							c.addError(valExpr.Pos(), "ERR_TYPE_MISMATCH",
								fmt.Sprintf("Function in property '%s' returns type '%s', expected '%s'", f.Name, actualRet, expectedRet))
							continue
						}
					}
				}

				valType := c.inferType(valExpr)
				if strings.Contains(expectedFieldType, " or ") || strings.HasPrefix(expectedFieldType, "\"") {
					if strLit, ok := valExpr.(*ast.StringLiteral); ok {
						valType = fmt.Sprintf("%q", strLit.Value)
					} else if numLit, ok := valExpr.(*ast.NumberLiteral); ok {
						valType = fmt.Sprintf("%v", numLit.Value)
					}
				}
				if !c.isTypeCompatible(expectedFieldType, valType) {
					c.addError(valExpr.Pos(), "ERR_TYPE_MISMATCH",
						fmt.Sprintf("Property '%s' in argument %d ('%s') expects type '%s', got '%s'",
							f.Name, argIdx, paramName, f.Type, valType))
				}
			}
		}
		return
	}

	if ident, ok := arg.(*ast.Identifier); ok {
		if sym, ok := c.scope.Resolve(ident.Value); ok {
			sym.Used = true
			if sym.Type != iface.Name {
				c.addError(arg.Pos(), "ERR_TYPE_MISMATCH",
					fmt.Sprintf("Argument %d ('%s') of function '%s' expects interface '%s', got '%s' (type: '%s')",
						argIdx, paramName, funcName, iface.Name, ident.Value, sym.Type))
			}
			return
		}
	}

	argType := c.inferType(arg)
	if !c.isTypeCompatible(iface.Name, argType) {
		c.addError(arg.Pos(), "ERR_TYPE_MISMATCH",
			fmt.Sprintf("Argument %d ('%s') of function '%s' expects interface '%s', got '%s'",
				argIdx, paramName, funcName, iface.Name, argType))
	}
}

func (c *Checker) validateArgAgainstInlineMap(arg ast.Expression, typeStr, paramName, funcName string, argIdx int, isOptional bool) {
	if isOptional {
		if ident, ok := arg.(*ast.Identifier); ok && (ident.Value == "nil" || ident.Value == "null") {
			return
		}
	}

	fields := parseInlineMapFieldDefs(typeStr)
	mapLit, isLit := arg.(*ast.MapLiteral)
	if isLit {
		keys := make(map[string]ast.Expression)
		for _, pair := range mapLit.Pairs {
			if strLit, ok := pair.Key.(*ast.StringLiteral); ok {
				keys[strLit.Value] = pair.Value
			} else if ident, ok := pair.Key.(*ast.Identifier); ok {
				keys[ident.Value] = pair.Value
			}
		}

		// Check for unknown properties not defined in the inline map
		if len(fields) > 0 {
			for key, valExpr := range keys {
				found := false
				for _, f := range fields {
					if f.Name == key {
						found = true
						break
					}
				}
				if !found {
					c.addError(valExpr.Pos(), "ERR_UNKNOWN_PROPERTY",
						fmt.Sprintf("Property '%s' does not exist on inline map type '%s'", key, typeStr))
				}
			}
		}

		for _, f := range fields {
			valExpr, exists := keys[f.Name]
			if !f.Optional && !strings.HasSuffix(f.Type, "?") {
				if !exists {
					c.addError(arg.Pos(), "ERR_TYPE_MISMATCH",
						fmt.Sprintf("Argument %d ('%s') of function '%s' expects inline map, but map is missing required property '%s'",
							argIdx, paramName, funcName, f.Name))
					continue
				}
			}
			if exists {
				expectedFieldType := c.resolveType(f.Type)

				if mapLit, isMap := valExpr.(*ast.MapLiteral); isMap {
					if strings.HasPrefix(expectedFieldType, "@map") && strings.Contains(expectedFieldType, "{") {
						c.validateArgAgainstInlineMap(mapLit, expectedFieldType, f.Name, funcName, argIdx, f.Optional)
						continue
					}
					if c.interfaces != nil {
						if subIface, ok := c.interfaces[expectedFieldType]; ok {
							c.validateArgAgainstInterface(mapLit, subIface, f.Name, funcName, argIdx, f.Optional)
							continue
						}
					}
				}

				if lambda, isLambda := valExpr.(*ast.LambdaExpression); isLambda {
					if arrowIdx := strings.Index(f.Type, "->"); arrowIdx != -1 {
						expectedRet := strings.TrimSpace(f.Type[arrowIdx+2:])
						actualRet := "void"
						if lambda.Body != nil {
							if block, isBlock := lambda.Body.(*ast.BlockStatement); isBlock {
								for _, st := range block.Statements {
									if retSt, ok := st.(*ast.ReturnStatement); ok && retSt.Value != nil {
										actualRet = c.inferType(retSt.Value)
										break
									}
								}
							} else {
								actualRet = c.inferType(lambda.Body)
							}
						}
						if !c.isTypeCompatible(expectedRet, actualRet) {
							c.addError(valExpr.Pos(), "ERR_TYPE_MISMATCH",
								fmt.Sprintf("Function in property '%s' returns type '%s', expected '%s'", f.Name, actualRet, expectedRet))
							continue
						}
					}
				}

				valType := c.inferType(valExpr)
				if strings.Contains(expectedFieldType, " or ") || strings.HasPrefix(expectedFieldType, "\"") {
					if strLit, ok := valExpr.(*ast.StringLiteral); ok {
						valType = fmt.Sprintf("%q", strLit.Value)
					} else if numLit, ok := valExpr.(*ast.NumberLiteral); ok {
						valType = fmt.Sprintf("%v", numLit.Value)
					}
				}
				if !c.isTypeCompatible(expectedFieldType, valType) {
					c.addError(valExpr.Pos(), "ERR_TYPE_MISMATCH",
						fmt.Sprintf("Property '%s' in argument %d ('%s') expects type '%s', got '%s'",
							f.Name, argIdx, paramName, f.Type, valType))
				}
			}
		}
		return
	}

	argType := c.inferType(arg)
	if !c.isTypeCompatible(typeStr, argType) {
		c.addError(arg.Pos(), "ERR_TYPE_MISMATCH",
			fmt.Sprintf("Argument %d ('%s') of function '%s' expects inline map '%s', got '%s'",
				argIdx, paramName, funcName, typeStr, argType))
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
			if argType != "any" && !c.isTypeCompatible(param.Type, argType) {
				c.addError(arg.Pos(), "ERR_TYPE_MISMATCH",
					fmt.Sprintf("Argument %d ('%s') of method '%s' expects type '%s', got '%s'", i+1, param.Name, methodName, param.Type, argType))
			}
		}
	}
}
