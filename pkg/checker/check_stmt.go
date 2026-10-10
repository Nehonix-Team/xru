package checker

import (
	"fmt"
	"strings"

	"github.com/Nehonix-Team/xru/pkg/ast"
)

func (c *Checker) checkStatement(stmt ast.Statement) {
	switch s := stmt.(type) {
	case *ast.UseStatement:
		c.checkUse(s)
	case *ast.ArgStatement:
		c.checkArg(s)
	case *ast.VarStatement:
		c.checkVar(s)
	case *ast.UnitStatement:
		c.checkUnit(s)
	case *ast.WithStatement:
		c.checkWith(s)
	case *ast.FuncStatement:
		c.checkFunc(s)
	case *ast.IfStatement:
		c.checkIf(s)
	case *ast.MapStatement:
		c.checkMap(s)
	case *ast.ModStatement:
		c.checkMod(s)
	case *ast.FromUseStatement:
		c.checkFromUse(s)
	case *ast.BlockStatement:
		for _, bodyStmt := range s.Statements {
			c.checkStatement(bodyStmt)
		}
	case *ast.ExpressionStatement:
		c.checkExpression(s.Expression)
	case *ast.FileStatement:
		for _, bodyStmt := range s.Body {
			c.checkStatement(bodyStmt)
		}
	case *ast.DispatchTreeStatement:
		// DispatchTree statement is valid
	case *ast.ReturnStatement:
		if s.Value != nil {
			c.checkExpression(s.Value)
			if c.currentFunc != nil && c.currentFunc.ReturnType != "" && c.currentFunc.ReturnType != "void" {
				actualType := c.inferType(s.Value)
				if !isTypeCompatible(c.currentFunc.ReturnType, actualType) {
					c.addError(s.Pos(), "ERR_TYPE_MISMATCH",
						fmt.Sprintf("Cannot return value of type '%s' from function '%s' expecting '%s'", actualType, c.currentFunc.Name, c.currentFunc.ReturnType))
				}
			}
		}
	}
}

func (c *Checker) checkArg(s *ast.ArgStatement) {
	kind := SymVar
	valStr := ""
	if s.DefaultValue != nil {
		valStr = s.DefaultValue.String()
	}
	inferredType := s.Type
	if inferredType == "" && s.DefaultValue != nil {
		inferredType = c.inferType(s.DefaultValue)
	}

	sym := &Symbol{
		Name:     s.Name,
		Kind:     kind,
		Type:     inferredType,
		Pos:      s.Pos(),
		ValueStr: valStr,
	}
	if err := c.scope.Define(sym); err != nil {
		c.addError(s.Pos(), "ERR_DUPLICATE_SYMBOL", err.Error())
	}

	if s.DefaultValue != nil {
		c.checkExpression(s.DefaultValue)
		if s.Type != "" {
			actualType := c.inferType(s.DefaultValue)
			if !isTypeCompatible(s.Type, actualType) {
				c.addError(s.Pos(), "ERR_TYPE_MISMATCH",
					fmt.Sprintf("Default value of type '%s' does not match declared type '%s' for argument '%s'", actualType, s.Type, s.Name))
			}
		}
	}
}

func (c *Checker) checkVar(s *ast.VarStatement) {
	kind := SymVar
	if s.IsConst {
		kind = SymConst
	}

	valStr := ""
	if s.Value != nil {
		valStr = s.Value.String()
	}
	inferredType := s.Type
	if inferredType == "" && s.Value != nil {
		inferredType = c.inferType(s.Value)
	}

	sym := &Symbol{
		Name:     s.Name,
		Kind:     kind,
		Type:     inferredType,
		Pos:      s.Pos(),
		IsGlobal: s.IsGlobal,
		ValueStr: valStr,
	}
	if err := c.scope.Define(sym); err != nil {
		c.addError(s.Pos(), "ERR_DUPLICATE_SYMBOL", err.Error())
	}

	c.checkExpression(s.Value)

	if s.Value != nil && s.Type != "" {
		actualType := c.inferType(s.Value)
		if !isTypeCompatible(s.Type, actualType) {
			c.addError(s.Pos(), "ERR_TYPE_MISMATCH",
				fmt.Sprintf("Cannot assign expression of type '%s' to variable '%s' declared as '%s'", actualType, s.Name, s.Type))
		}
	}
}

func (c *Checker) checkUnit(s *ast.UnitStatement) {
	unitSym := &Symbol{
		Name:       s.Name,
		Kind:       SymUnit,
		Pos:        s.Pos(),
		Directives: make(map[string]bool),
		Methods:    make(map[string]*Symbol),
	}

	for _, dir := range s.Directives {
		dirName := "#" + dir.Name
		unitSym.Directives[dirName] = true
	}

	for _, method := range s.Methods {
		unitSym.Methods[method.Name] = &Symbol{
			Name:   method.Name,
			Kind:   SymFunc,
			Type:   method.ReturnType,
			Pos:    method.Pos(),
			Params: method.Parameters,
		}
	}

	if err := c.scope.Define(unitSym); err != nil {
		c.addError(s.Pos(), "ERR_DUPLICATE_SYMBOL", err.Error())
	}

	unitScope := NewScope(c.scope, true)
	c.scope = unitScope
	defer func() { c.scope = c.scope.parent }()

	for _, prop := range s.Properties {
		propSym := &Symbol{Name: prop.Name, Kind: SymVar, Type: prop.Type, Pos: prop.Pos()}
		if err := c.scope.Define(propSym); err != nil {
			c.addError(prop.Pos(), "ERR_DUPLICATE_SYMBOL", err.Error())
		}
		c.checkExpression(prop.Value)
		if prop.Value != nil && prop.Type != "" {
			actualType := c.inferType(prop.Value)
			if !isTypeCompatible(prop.Type, actualType) {
				c.addError(prop.Pos(), "ERR_TYPE_MISMATCH",
					fmt.Sprintf("Cannot assign expression of type '%s' to property '%s' declared as '%s'", actualType, prop.Name, prop.Type))
			}
		}
	}

	for _, method := range s.Methods {
		c.checkFunc(method)
	}
}

func (c *Checker) checkWith(s *ast.WithStatement) {
	if s == nil {
		return
	}
	c.checkExpression(s.Target)

	var targetUnitSym *Symbol
	switch t := s.Target.(type) {
	case *ast.Identifier:
		if sym, ok := c.scope.Resolve(t.Value); ok {
			targetUnitSym = sym
		}
	case *ast.CallExpression:
		if member, ok := t.Caller.(*ast.MemberExpression); ok {
			if ident, ok := member.Object.(*ast.Identifier); ok {
				if sym, ok := c.scope.Resolve(ident.Value); ok {
					targetUnitSym = sym
				}
			}
		}
	}

	// Validation stricte des directives contextuelles dans le bloc
	for _, bodyStmt := range s.Body {
		if customDir, ok := bodyStmt.(*ast.CustomDirectiveInvocation); ok {
			dirName := strings.ToLower(customDir.Directive)
			if targetUnitSym != nil && len(targetUnitSym.Directives) > 0 {
				found := false
				for dKey := range targetUnitSym.Directives {
					if strings.ToLower(dKey) == dirName {
						found = true
						break
					}
				}
				if !found {
					c.addError(customDir.Pos(), "ERR_ORPHAN_DIRECTIVE",
						fmt.Sprintf("Directive '%s' is not defined for with target '%s'", customDir.Directive, targetUnitSym.Name))
				}
			}
			// Validation des propriétés et événements spécifiques pour Net ou Net.fetch
			if targetUnitSym != nil && (strings.EqualFold(targetUnitSym.Name, "Net") || strings.EqualFold(targetUnitSym.Name, "Net.Fetch")) {
				if dirName == "set" && customDir.Key != nil {
					keyName := strings.ToLower(customDir.Key.String())
					if keyName != "method" && keyName != "timeout" && keyName != "extractto" && keyName != "dest" {
						c.addError(customDir.Pos(), "ERR_INVALID_PROPERTY",
							fmt.Sprintf("Property '%s' does not exist on Net.fetch (supported: method, timeout, extractTo, dest)", customDir.Key.String()))
					}
				} else if dirName == "on" && customDir.Key != nil {
					keyName := strings.ToLower(customDir.Key.String())
					if keyName != "progress" && keyName != "complete" {
						c.addError(customDir.Pos(), "ERR_INVALID_EVENT",
							fmt.Sprintf("Event '%s' does not exist on Net.fetch (supported: progress, complete)", customDir.Key.String()))
					}
				}
			}
		}
		c.checkStatement(bodyStmt)
	}
}

func (c *Checker) checkFunc(s *ast.FuncStatement) {
	fnSym := &Symbol{
		Name:     s.Name,
		Kind:     SymFunc,
		Type:     s.ReturnType,
		Pos:      s.Pos(),
		Params:   s.Parameters,
		IsGlobal: s.IsGlobal,
	}
	if err := c.scope.Define(fnSym); err != nil {
		c.addError(s.Pos(), "ERR_DUPLICATE_SYMBOL", err.Error())
	}

	oldFunc := c.currentFunc
	c.currentFunc = s
	defer func() { c.currentFunc = oldFunc }()

	fnScope := NewScope(c.scope, false)
	c.scope = fnScope
	defer func() { c.scope = c.scope.parent }()

	for _, p := range s.Parameters {
		paramSym := &Symbol{Name: p.Name, Kind: SymVar, Type: p.Type, Pos: s.Pos()}
		if err := c.scope.Define(paramSym); err != nil {
			c.addError(s.Pos(), "ERR_DUPLICATE_SYMBOL", err.Error())
		}
	}

	for _, bodyStmt := range s.Body {
		c.checkStatement(bodyStmt)
	}
}

func (c *Checker) checkIf(s *ast.IfStatement) {
	c.checkExpression(s.Condition)
	for _, stmt := range s.Consequence {
		c.checkStatement(stmt)
	}
	for _, elif := range s.ElifBlocks {
		c.checkExpression(elif.Condition)
		for _, stmt := range elif.Body {
			c.checkStatement(stmt)
		}
	}
	for _, stmt := range s.Alternative {
		c.checkStatement(stmt)
	}
}
