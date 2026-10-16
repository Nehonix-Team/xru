package checker

import (
	"fmt"
	"strings"

	"github.com/Nehonix-Team/xru/pkg/ast"
	"github.com/Nehonix-Team/xru/pkg/lexer"
)

func (c *Checker) inferType(expr ast.Expression) string {
	if expr == nil {
		return "any"
	}
	switch e := expr.(type) {
	case *ast.BlockStatement:
		return "void"
	case *ast.StringLiteral:
		return "string"
	case *ast.NumberLiteral:
		return "number"
	case *ast.BoolLiteral:
		return "bool"
	case *ast.DurationLiteral:
		return "duration"
	case *ast.ListLiteral:
		if len(e.Elements) == 0 {
			return "list"
		}
		return "list[" + c.inferType(e.Elements[0]) + "]"
	case *ast.MapLiteral:
		return "map"
	case *ast.LambdaExpression:
		return "function"
	case *ast.InstantiationExpression:
		return e.UnitName
	case *ast.WithExpression:
		targetType := c.inferType(e.Target)
		if targetType != "" && targetType != "any" {
			return "task[" + targetType + "]"
		}
		return "task"
	case *ast.RangeExpression:
		return "range"
	case *ast.Identifier:
		if strings.EqualFold(e.Value, "nil") || strings.EqualFold(e.Value, "null") {
			return "any"
		}
		if sym, ok := c.scope.Resolve(e.Value); ok {
			if sym.Type != "" {
				return sym.Type
			}
			if sym.Kind == SymUnit {
				return sym.Name
			}
		}
		return "any"
	case *ast.PropAccess:
		if sym, ok := c.scope.Resolve("&" + e.Name); ok {
			return sym.Type
		}
		if sym, ok := c.scope.Resolve(e.Name); ok {
			return sym.Type
		}
		unitSym := c.scope.GetEnclosingUnit()
		if unitSym != nil && unitSym.Methods != nil {
			if mSym, ok := unitSym.Methods[e.Name]; ok && mSym.Type != "" {
				return mSym.Type
			}
			for mName, mSym := range unitSym.Methods {
				if strings.EqualFold(mName, e.Name) && mSym.Type != "" {
					return mSym.Type
				}
			}
		}
		return "any"
	case *ast.PrefixExpression:
		if e.Operator == "!" {
			return "bool"
		}
		if e.Operator == "-" {
			return "number"
		}
		return "any"
	case *ast.InfixExpression:
		switch e.Operator {
		case "==", "!=", "<", "<=", ">", ">=", "&&", "||", "or":
			return "bool"
		case "+":
			leftType := c.inferType(e.Left)
			rightType := c.inferType(e.Right)
			if leftType == "string" || rightType == "string" {
				return "string"
			}
			return "number"
		case "-", "*", "/", "%":
			return "number"
		}
		return "any"
	case *ast.CallExpression:
		if member, ok := e.Caller.(*ast.MemberExpression); ok {
			objType := c.inferType(member.Object)
			if objType == "string" {
				switch strings.ToLower(member.Property) {
				case "len":
					return "number"
				case "contains", "startswith", "endswith":
					return "bool"
				case "split":
					return "list[string]"
				case "trim", "trimleft", "trimright", "toupper", "tolower", "replace", "slice":
					return "string"
				}
			} else if strings.HasPrefix(objType, "list") {
				switch strings.ToLower(member.Property) {
				case "len":
					return "number"
				case "join":
					return "string"
				case "contains":
					return "bool"
				case "push":
					return "void"
				case "get", "at", "pop":
					if strings.HasPrefix(objType, "list[") && strings.HasSuffix(objType, "]") {
						return strings.TrimSuffix(strings.TrimPrefix(objType, "list["), "]")
					}
					return "any"
				case "reverse", "slice":
					return objType
				}
			}

			if ident, ok := member.Object.(*ast.Identifier); ok {
				if sym, ok := c.scope.Resolve(ident.Value); ok {
					if strings.HasPrefix(sym.Type, "task") {
						propLower := strings.ToLower(member.Property)
						if propLower == "isdone" || propLower == "done" {
							return "bool"
						}
						if propLower == "cancel" {
							return "void"
						}
						if propLower == "wait" || propLower == "await" || propLower == "result" {
							if strings.HasPrefix(sym.Type, "task[") && strings.HasSuffix(sym.Type, "]") {
								return strings.TrimSuffix(strings.TrimPrefix(sym.Type, "task["), "]")
							}
							return "any"
						}
						return "any"
					}
					var unitSym *Symbol
					if sym.Methods != nil {
						unitSym = sym
					} else if sym.Type != "" {
						if uSym, ok := c.scope.Resolve(sym.Type); ok && uSym.Methods != nil {
							unitSym = uSym
						}
					}
					if unitSym != nil {
						for mName, mSym := range unitSym.Methods {
							if strings.EqualFold(mName, member.Property) {
								if mSym.Type != "" {
									return mSym.Type
								}
							}
						}
					}
				}
			}
		} else if scopeAccess, ok := e.Caller.(*ast.ScopeAccessExpression); ok {
			if scopeAccess.IsUnitSelf {
				unitSym := c.scope.GetEnclosingUnit()
				if unitSym != nil && unitSym.Methods != nil {
					if mSym, ok := unitSym.Methods[scopeAccess.Member]; ok && mSym.Type != "" {
						return mSym.Type
					}
					for mName, mSym := range unitSym.Methods {
						if strings.EqualFold(mName, scopeAccess.Member) && mSym.Type != "" {
							return mSym.Type
						}
					}
				}
			} else if sym, ok := c.scope.Resolve(scopeAccess.Scope); ok {
				if sym.Kind == SymUnit && sym.Methods != nil {
					if mSym, ok := sym.Methods[scopeAccess.Member]; ok && mSym.Type != "" {
						return mSym.Type
					}
					for mName, mSym := range sym.Methods {
						if strings.EqualFold(mName, scopeAccess.Member) && mSym.Type != "" {
							return mSym.Type
						}
					}
				}
			}
		} else if propAccess, ok := e.Caller.(*ast.PropAccess); ok {
			unitSym := c.scope.GetEnclosingUnit()
			if unitSym != nil && unitSym.Methods != nil {
				if mSym, ok := unitSym.Methods[propAccess.Name]; ok && mSym.Type != "" {
					return mSym.Type
				}
				for mName, mSym := range unitSym.Methods {
					if strings.EqualFold(mName, propAccess.Name) && mSym.Type != "" {
						return mSym.Type
					}
				}
			}
		} else if ident, ok := e.Caller.(*ast.Identifier); ok {
			if sym, ok := c.scope.Resolve(ident.Value); ok && sym.Kind == SymFunc {
				if sym.Type != "" {
					return sym.Type
				}
			}
		}
		return "any"
	case *ast.GlobalCallExpression:
		if sym, ok := c.scope.Resolve(e.Function); ok && sym.Kind == SymFunc {
			if sym.Type != "" {
				return sym.Type
			}
		}
		return "any"
	case *ast.GlobalAccessExpression:
		if sym, ok := c.scope.Resolve(e.Name); ok {
			return sym.Type
		}
		return "any"
	case *ast.IfExpression:
		conseqType := c.inferType(e.Consequence)
		altType := c.inferType(e.Alternative)
		if conseqType == altType {
			return conseqType
		}
		if isTypeCompatible(conseqType, altType) {
			return conseqType
		}
		if isTypeCompatible(altType, conseqType) {
			return altType
		}
		return "any"
	case *ast.CatchExpression:
		targetType := c.inferType(e.Target)
		if e.Fallback != nil {
			if fbExpr, ok := e.Fallback.(ast.Expression); ok {
				fbType := c.inferType(fbExpr)
				if targetType == "" || targetType == "any" {
					return fbType
				}
			}
		}
		if targetType != "" {
			return targetType
		}
		return "any"
	case *ast.MemberExpression:
		var targetType string
		if ident, ok := e.Object.(*ast.Identifier); ok {
			if sym, ok := c.scope.Resolve(ident.Value); ok && sym.Type != "" {
				targetType = sym.Type
			}
		}
		if targetType == "" {
			targetType = c.inferType(e.Object)
		}
		targetType = c.resolveType(targetType)

		if c.interfaces != nil {
			if iface, ok := c.interfaces[targetType]; ok {
				for _, f := range iface.Fields {
					if f.Name == e.Property {
						return f.Type
					}
				}
			}
		}
		if sym, ok := c.scope.Resolve(targetType); ok && sym.Methods != nil {
			if m, ok := sym.Methods[e.Property]; ok && m.Type != "" {
				return m.Type
			}
		}
		if strings.HasPrefix(targetType, "@map") && strings.Contains(targetType, "{") {
			fields := parseInlineMapFieldDefs(targetType)
			for _, f := range fields {
				if f.Name == e.Property {
					return f.Type
				}
			}
		}
		return "any"
	default:
		return "any"
	}
}

func (c *Checker) resolveType(typeName string) string {
	typeName = strings.TrimSpace(typeName)
	if typeName == "" {
		return "any"
	}

	// Traitement des types indexés: TExample["c"] ou TExample['c']
	if idx := strings.Index(typeName, "["); idx != -1 && strings.HasSuffix(typeName, "]") {
		baseName := strings.TrimSpace(typeName[:idx])
		key := strings.TrimSuffix(typeName[idx+1:], "]")
		key = strings.Trim(strings.TrimSpace(key), "\"'")

		if c.interfaces != nil {
			if iface, ok := c.interfaces[baseName]; ok {
				for _, f := range iface.Fields {
					if f.Name == key {
						return f.Type
					}
				}
			}
		}
		if sym, ok := c.scope.Resolve(baseName); ok && sym.Methods != nil {
			if m, ok := sym.Methods[key]; ok && m.Type != "" {
				return m.Type
			}
		}
	}
	return typeName
}

func (c *Checker) validateTypeAnnotation(pos lexer.Position, typeName string) {
	typeName = strings.TrimSpace(typeName)
	if typeName == "" || typeName == "any" || typeName == "void" {
		return
	}
	typeName = strings.TrimSuffix(typeName, "?")

	if strings.Contains(typeName, " or ") {
		for _, part := range strings.Split(typeName, " or ") {
			c.validateTypeAnnotation(pos, strings.TrimSpace(part))
		}
		return
	}

	if idx := strings.Index(typeName, "["); idx != -1 && strings.HasSuffix(typeName, "]") {
		baseName := strings.TrimSpace(typeName[:idx])
		key := strings.TrimSuffix(typeName[idx+1:], "]")
		key = strings.Trim(strings.TrimSpace(key), "\"'")

		if strings.EqualFold(baseName, "list") || strings.EqualFold(baseName, "task") || strings.EqualFold(baseName, "map") {
			c.validateTypeAnnotation(pos, key)
			return
		}

		if c.interfaces != nil {
			if iface, ok := c.interfaces[baseName]; ok {
				found := false
				for _, f := range iface.Fields {
					if f.Name == key {
						found = true
						break
					}
				}
				if !found {
					c.addError(pos, "ERR_NO_SUCH_PROPERTY",
						fmt.Sprintf("Property '%s' does not exist on type '%s'", key, baseName))
					return
				}
				return
			}
		}
		if sym, ok := c.scope.Resolve(baseName); ok && sym.Methods != nil {
			if _, ok := sym.Methods[key]; ok {
				return
			}
			c.addError(pos, "ERR_NO_SUCH_PROPERTY",
				fmt.Sprintf("Property '%s' does not exist on type '%s'", key, baseName))
			return
		}
		// Si baseName n'est ni une interface ni un module connu
		if _, isBuiltin := c.availableModules[baseName]; !isBuiltin {
			c.addError(pos, "ERR_UNDEFINED_TYPE",
				fmt.Sprintf("Type interface '%s' is not defined", baseName))
		}
	}
}

func parseInlineMapFieldDefs(typeStr string) []*ast.InterfaceField {
	var fields []*ast.InterfaceField
	braceIdx := strings.Index(typeStr, "{")
	closeIdx := strings.LastIndex(typeStr, "}")
	if braceIdx == -1 || closeIdx == -1 || closeIdx <= braceIdx {
		return fields
	}
	body := strings.TrimSpace(typeStr[braceIdx+1 : closeIdx])
	body = strings.ReplaceAll(body, ";", ",")
	body = strings.ReplaceAll(body, "\n", ",")

	parts := strings.Split(body, ",")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		var colonIndices []int
		for idx, r := range p {
			if r == ':' {
				colonIndices = append(colonIndices, idx)
			}
		}
		if len(colonIndices) <= 1 {
			colonIdx := strings.Index(p, ":")
			if colonIdx == -1 {
				continue
			}
			namePart := strings.TrimSpace(p[:colonIdx])
			typePart := strings.TrimSpace(p[colonIdx+1:])
			optional := false
			if strings.HasSuffix(namePart, "?") {
				optional = true
				namePart = strings.TrimSpace(strings.TrimSuffix(namePart, "?"))
			}
			if strings.HasSuffix(typePart, "?") {
				optional = true
				typePart = strings.TrimSpace(strings.TrimSuffix(typePart, "?"))
			}
			fields = append(fields, &ast.InterfaceField{
				Name:     namePart,
				Type:     typePart,
				Optional: optional,
			})
		} else {
			type rawField struct {
				name  string
				fType string
			}
			var subFields []rawField
			for ci := 0; ci < len(colonIndices); ci++ {
				currColon := colonIndices[ci]
				var name string
				if ci == 0 {
					name = strings.TrimSpace(p[:currColon])
				} else {
					prevColon := colonIndices[ci-1]
					mid := strings.TrimSpace(p[prevColon+1 : currColon])
					words := strings.Fields(mid)
					if len(words) > 0 {
						name = words[len(words)-1]
						subFields[ci-1].fType = strings.TrimSpace(strings.Join(words[:len(words)-1], " "))
					}
				}
				subFields = append(subFields, rawField{name: name})
				if ci == len(colonIndices)-1 {
					subFields[ci].fType = strings.TrimSpace(p[currColon+1:])
				}
			}
			for _, sf := range subFields {
				namePart := sf.name
				typePart := sf.fType
				optional := false
				if strings.HasSuffix(namePart, "?") {
					optional = true
					namePart = strings.TrimSpace(strings.TrimSuffix(namePart, "?"))
				}
				if strings.HasSuffix(typePart, "?") {
					optional = true
					typePart = strings.TrimSpace(strings.TrimSuffix(typePart, "?"))
				}
				fields = append(fields, &ast.InterfaceField{
					Name:     namePart,
					Type:     typePart,
					Optional: optional,
				})
			}
		}
	}
	return fields
}

func (c *Checker) isTypeCompatible(expected, actual string) bool {
	if expected == "" || expected == "any" {
		return true
	}
	if actual == "" || actual == "any" {
		return true
	}
	expected = c.resolveType(expected)
	actual = c.resolveType(actual)

	expectedOpt := strings.HasSuffix(expected, "?")
	cleanExpected := strings.TrimSuffix(expected, "?")
	cleanActual := strings.TrimSuffix(actual, "?")

	if expectedOpt && (actual == "nil" || actual == "null") {
		return true
	}
	if actual == "nil" || actual == "null" {
		return expectedOpt || strings.Contains(expected, "nil") || strings.Contains(expected, "null")
	}

	if strings.EqualFold(cleanExpected, cleanActual) {
		return true
	}
	if (strings.EqualFold(cleanExpected, "bool") || strings.EqualFold(cleanExpected, "boolean")) &&
		(strings.EqualFold(cleanActual, "bool") || strings.EqualFold(cleanActual, "boolean")) {
		return true
	}
	if strings.EqualFold(cleanExpected, "number") && (strings.EqualFold(cleanActual, "int") || strings.EqualFold(cleanActual, "float") || strings.EqualFold(cleanActual, "number")) {
		return true
	}
	if strings.HasPrefix(cleanExpected, "list") && strings.HasPrefix(cleanActual, "list") {
		if cleanExpected == "list" || cleanActual == "list" {
			return true
		}
		tExpected := strings.TrimSuffix(strings.TrimPrefix(cleanExpected, "list["), "]")
		tActual := strings.TrimSuffix(strings.TrimPrefix(cleanActual, "list["), "]")
		return c.isTypeCompatible(tExpected, tActual)
	}
	if strings.Contains(cleanExpected, " or ") {
		for _, part := range strings.Split(cleanExpected, " or ") {
			if c.isTypeCompatible(strings.TrimSpace(part), actual) {
				return true
			}
		}
		return false
	}
	if strings.HasPrefix(cleanExpected, "\"") && strings.HasSuffix(cleanExpected, "\"") {
		if cleanActual == "string" || cleanActual == cleanExpected {
			return true
		}
	}
	if strings.HasPrefix(cleanExpected, "func") || strings.HasPrefix(cleanExpected, "function") {
		if cleanActual == "function" || strings.HasPrefix(cleanActual, "func") {
			return true
		}
	}
	// Interface checking
	if _, isExpectedIface := c.interfaces[cleanExpected]; isExpectedIface {
		return cleanExpected == cleanActual
	}
	if cleanExpected == "map" || cleanExpected == "@map" {
		if cleanActual == "map" || cleanActual == "@map" || strings.HasPrefix(cleanActual, "@map") {
			return true
		}
		if _, isIface := c.interfaces[cleanActual]; isIface {
			return true
		}
	}
	if strings.HasPrefix(cleanExpected, "task") && strings.HasPrefix(cleanActual, "task") {
		if cleanExpected == "task" || cleanActual == "task" {
			return true
		}
		tExpected := strings.TrimSuffix(strings.TrimPrefix(cleanExpected, "task["), "]")
		tActual := strings.TrimSuffix(strings.TrimPrefix(cleanActual, "task["), "]")
		return c.isTypeCompatible(tExpected, tActual)
	}
	return false
}

func isTypeCompatible(expected, actual string) bool {
	dummy := New()
	return dummy.isTypeCompatible(expected, actual)
}
