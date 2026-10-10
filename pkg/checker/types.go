package checker

import (
	"strings"

	"github.com/Nehonix-Team/xru/pkg/ast"
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
			if ident, ok := member.Object.(*ast.Identifier); ok {
				if sym, ok := c.scope.Resolve(ident.Value); ok && sym.Methods != nil {
					for mName, mSym := range sym.Methods {
						if strings.EqualFold(mName, member.Property) {
							if mSym.Type != "" {
								return mSym.Type
							}
						}
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
	default:
		return "any"
	}
}

func isTypeCompatible(expected, actual string) bool {
	if expected == "" || expected == "any" {
		return true
	}
	if actual == "" || actual == "any" {
		return true
	}
	if strings.EqualFold(expected, actual) {
		return true
	}
	if (strings.EqualFold(expected, "bool") || strings.EqualFold(expected, "boolean")) &&
		(strings.EqualFold(actual, "bool") || strings.EqualFold(actual, "boolean")) {
		return true
	}
	if strings.EqualFold(expected, "number") && (strings.EqualFold(actual, "int") || strings.EqualFold(actual, "float") || strings.EqualFold(actual, "number")) {
		return true
	}
	if strings.HasPrefix(expected, "list") && strings.HasPrefix(actual, "list") {
		if expected == "list" || actual == "list" {
			return true
		}
		tExpected := strings.TrimSuffix(strings.TrimPrefix(expected, "list["), "]")
		tActual := strings.TrimSuffix(strings.TrimPrefix(actual, "list["), "]")
		return isTypeCompatible(tExpected, tActual)
	}
	if strings.HasPrefix(expected, "map") && strings.HasPrefix(actual, "map") {
		return true
	}
	return false
}
