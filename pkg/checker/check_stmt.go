package checker

import (
	"fmt"
	"strings"

	"github.com/Nehonix-Team/xru/pkg/ast"
)

func (c *Checker) checkStatement(stmt ast.Statement) {
	if stmt == nil {
		return
	}
	switch s := stmt.(type) {
	case *ast.UseStatement:
		if s == nil { return }
		c.checkUse(s)
	case *ast.ArgStatement:
		if s == nil { return }
		c.checkArg(s)
	case *ast.VarStatement:
		if s == nil { return }
		c.checkVar(s)
	case *ast.UnitStatement:
		if s == nil { return }
		c.checkUnit(s)
	case *ast.InterfaceStatement:
		if s == nil { return }
		c.checkInterface(s)
	case *ast.WithStatement:
		if s == nil { return }
		c.checkWith(s)
	case *ast.FuncStatement:
		if s == nil { return }
		c.checkFunc(s)
	case *ast.IfStatement:
		if s == nil { return }
		c.checkIf(s)
	case *ast.MapStatement:
		if s == nil { return }
		c.checkMap(s)
	case *ast.ModStatement:
		if s == nil { return }
		c.checkMod(s)
	case *ast.FromUseStatement:
		if s == nil { return }
		c.checkFromUse(s)
	case *ast.ForStatement:
		if s == nil { return }
		c.checkFor(s)
	case *ast.WhileStatement:
		if s == nil { return }
		c.checkWhile(s)
	case *ast.BlockStatement:
		if s == nil { return }
		c.scope = NewScope(c.scope, false)
		c.checkStatementList(s.Statements)
		c.scope = c.scope.parent
	case *ast.ExpressionStatement:
		if s == nil || s.Expression == nil { return }
		c.checkExpression(s.Expression)
	case *ast.FileStatement:
		if s.PathExpr != nil {
			c.checkExpression(s.PathExpr)
			pType := c.inferType(s.PathExpr)
			if pType != "string" && pType != "any" {
				c.addError(s.PathExpr.Pos(), "ERR_TYPE_MISMATCH",
					fmt.Sprintf("File path expression expects string, got '%s'", pType))
			}
		}

		if s.Handler != nil {
			if lambda, ok := s.Handler.(*ast.LambdaExpression); ok {
				childScope := NewScope(c.scope, false)
				oldScope := c.scope
				c.scope = childScope

				isErrFirst := false
				if len(lambda.Parameters) >= 2 {
					p0Name := strings.ToLower(lambda.Parameters[0].Name)
					p1Name := strings.ToLower(lambda.Parameters[1].Name)
					p1Type := lambda.Parameters[1].Type

					if p1Type == "Document" || ((p0Name == "e" || p0Name == "err" || p0Name == "error") && (p1Name == "data" || p1Name == "d" || p1Name == "doc" || p1Name == "content" || p1Type == "Document")) {
						isErrFirst = true
					}
				}

				createDocOrCustomSymbol := func(param *ast.Parameter) *Symbol {
					docType := param.Type
					if docType == "" {
						docType = "Document"
					}
					if iface, ok := c.interfaces[docType]; ok && (len(iface.Fields) > 0 || iface.Pos().Filename != "") {
						return &Symbol{
							Name: param.Name,
							Kind: SymVar,
							Type: docType,
							Pos:  lambda.Pos(),
						}
					}
					docSym := CreateDocumentSymbol(param.Name)
					docSym.Pos = lambda.Pos()
					return docSym
				}

				if len(lambda.Parameters) == 1 {
					p0Name := strings.ToLower(lambda.Parameters[0].Name)
					if p0Name == "e" || p0Name == "err" || p0Name == "error" {
						errType := lambda.Parameters[0].Type
						if errType == "" {
							errType = "any"
						}
						errSym := &Symbol{Name: lambda.Parameters[0].Name, Kind: SymVar, Type: errType, Pos: lambda.Pos()}
						_ = c.scope.Define(errSym)
						c.declaredVars = append(c.declaredVars, errSym)
					} else {
						docSym := createDocOrCustomSymbol(lambda.Parameters[0])
						_ = c.scope.Define(docSym)
						c.declaredVars = append(c.declaredVars, docSym)
					}
				} else if len(lambda.Parameters) >= 2 {
					if isErrFirst {
						errType := lambda.Parameters[0].Type
						if errType == "" {
							errType = "any"
						}
						errSym := &Symbol{
							Name: lambda.Parameters[0].Name,
							Kind: SymVar,
							Type: errType,
							Pos:  lambda.Pos(),
						}
						_ = c.scope.Define(errSym)
						c.declaredVars = append(c.declaredVars, errSym)

						docSym := createDocOrCustomSymbol(lambda.Parameters[1])
						_ = c.scope.Define(docSym)
						c.declaredVars = append(c.declaredVars, docSym)
					} else {
						docSym := createDocOrCustomSymbol(lambda.Parameters[0])
						_ = c.scope.Define(docSym)
						c.declaredVars = append(c.declaredVars, docSym)

						errType := lambda.Parameters[1].Type
						if errType == "" {
							errType = "any"
						}
						errSym := &Symbol{
							Name: lambda.Parameters[1].Name,
							Kind: SymVar,
							Type: errType,
							Pos:  lambda.Pos(),
						}
						_ = c.scope.Define(errSym)
						c.declaredVars = append(c.declaredVars, errSym)
					}
				}

				if lambda.Body != nil {
					c.checkExpression(lambda.Body)
				}

				c.scope = oldScope
			} else {
				c.checkExpression(s.Handler)
			}
		}

		for _, bodyStmt := range s.Body {
			c.checkStatement(bodyStmt)
		}
	case *ast.MatchStatement:
		c.checkMatch(s)
	case *ast.DispatchTreeStatement:
		if s.PathExpr != nil {
			c.checkExpression(s.PathExpr)
		}
		if s.PatternExpr != nil {
			c.checkExpression(s.PatternExpr)
		}
	case *ast.InStatement:
		c.checkIn(s)
	case *ast.ReassignStatement:
		c.checkReassign(s)
	case *ast.ReturnStatement:
		if c.currentFunc != nil && c.currentFunc.ReturnType != "" {
			expectedType := c.currentFunc.ReturnType
			if expectedType == "void" {
				if s.Value != nil {
					c.checkExpression(s.Value)
					actualType := c.inferType(s.Value)
					if actualType != "void" && actualType != "nil" {
						c.addError(s.Pos(), "ERR_TYPE_MISMATCH",
							fmt.Sprintf("Function '%s' has return type 'void', but returns value of type '%s'", c.currentFunc.Name, actualType))
					}
				}
			} else {
				if s.Value == nil {
					c.addError(s.Pos(), "ERR_TYPE_MISMATCH",
						fmt.Sprintf("Function '%s' expects return type '%s', but return statement has no value", c.currentFunc.Name, expectedType))
				} else {
					c.checkExpression(s.Value)
					actualType := c.inferType(s.Value)
					if !c.isTypeCompatible(expectedType, actualType) {
						c.addError(s.Pos(), "ERR_TYPE_MISMATCH",
							fmt.Sprintf("Cannot return value of type '%s' from function '%s' expecting '%s'", actualType, c.currentFunc.Name, expectedType))
					}
				}
			}
		} else {
			if s.Value != nil {
				c.checkExpression(s.Value)
			}
		}
	}
}

func (c *Checker) checkReassign(s *ast.ReassignStatement) {
	lookupName := s.Name
	lookupName = strings.TrimPrefix(lookupName, "&")

	sym, found := c.scope.Resolve(lookupName)
	if !found {
		c.addError(s.Pos(), "ERR_UNDEFINED_VARIABLE", fmt.Sprintf("Variable '%s' is not defined", s.Name))
		c.checkExpression(s.Value)
		return
	}

	if sym.Kind == SymConst {
		c.addError(s.Pos(), "ERR_CONST_REASSIGNMENT",
			fmt.Sprintf("Cannot reassign constant '%s': only variables defined with 'let' can be modified", s.Name))
	} else if sym.Kind != SymVar {
		c.addError(s.Pos(), "ERR_INVALID_ASSIGNMENT",
			fmt.Sprintf("Cannot reassign '%s' which is a %s", s.Name, sym.Kind))
	}

	c.checkExpression(s.Value)

	if s.Value != nil && sym.Type != "" && sym.Type != "any" {
		if mapLit, isMap := s.Value.(*ast.MapLiteral); isMap {
			if c.interfaces != nil {
				if iface, ok := c.interfaces[sym.Type]; ok {
					c.validateArgAgainstInterface(mapLit, iface, s.Name, "assignment", 0, false)
					return
				}
			}
			if strings.HasPrefix(sym.Type, "@map") && strings.Contains(sym.Type, "{") {
				c.validateArgAgainstInlineMap(mapLit, sym.Type, s.Name, "assignment", 0, false)
				return
			}
		}
		actualType := c.inferType(s.Value)
		if !c.isTypeCompatible(sym.Type, actualType) {
			c.addError(s.Pos(), "ERR_TYPE_MISMATCH",
				fmt.Sprintf("Cannot assign expression of type '%s' to variable '%s' declared as '%s'", actualType, s.Name, sym.Type))
		}
	}
}

func (c *Checker) checkArg(s *ast.ArgStatement) {
	kind := SymVar
	if s.IsGlobal && c.scope.parent != nil {
		c.addError(s.Pos(), "ERR_GLOBAL_IN_BLOCK", "Cannot declare 'global arg' inside a block, function, or loop. Global declarations are only permitted at the module root level.")
	}

	valStr := ""
	if s.DefaultValue != nil {
		valStr = s.DefaultValue.String()
	}
	inferredType := s.Type
	if inferredType == "" && s.DefaultValue != nil {
		inferredType = c.inferType(s.DefaultValue)
	}

	varName := s.VarName()
	if existing, ok := c.scope.Resolve(varName); ok && existing.Pos.Line == 0 {
		existing.Type = inferredType
		existing.ValueStr = valStr
		existing.Pos = s.Pos()
	} else {
		sym := &Symbol{
			Name:     varName,
			Kind:     kind,
			Type:     inferredType,
			Pos:      s.Pos(),
			ValueStr: valStr,
			IsGlobal: s.IsGlobal,
		}
		if err := c.scope.Define(sym); err != nil {
			c.addError(s.Pos(), "ERR_DUPLICATE_SYMBOL", err.Error())
		}
	}

	if s.Type != "" {
		pos := s.Pos()
		if s.TypePos.Line > 0 {
			pos = s.TypePos
		}
		c.validateTypeAnnotation(pos, s.Type)
	}

	if s.DefaultValue != nil {
		c.checkExpression(s.DefaultValue)
		if s.Type != "" {
			actualType := c.inferType(s.DefaultValue)
			resolvedType := c.resolveType(s.Type)
			if strings.Contains(resolvedType, " or ") || strings.Contains(resolvedType, "|") || strings.HasPrefix(resolvedType, "\"") {
				if strLit, ok := s.DefaultValue.(*ast.StringLiteral); ok {
					actualType = fmt.Sprintf("%q", strLit.Value)
				}
			}
			if !c.isTypeCompatible(s.Type, actualType) {
				c.addError(s.Pos(), "ERR_TYPE_MISMATCH",
					fmt.Sprintf("Default value of type '%s' does not match declared type '%s' for argument '%s'", actualType, s.Type, s.Name))
			}
		}
	}
}

func (c *Checker) checkVar(s *ast.VarStatement) {
	if s.IsGlobal && c.scope.parent != nil {
		c.addError(s.Pos(), "ERR_GLOBAL_IN_BLOCK", "Cannot declare 'global/export' variable or constant inside a block, function, or loop. Global declarations are only permitted at the module root level.")
	}
	if s.Value == nil {
		c.addError(s.Pos(), "ERR_MISSING_VALUE", fmt.Sprintf("Variable '%s' is declared without a value", s.Name))
	}

	kind := SymVar
	if s.IsConst {
		kind = SymConst
	}

	valStr := ""
	if s.Value != nil {
		valStr = s.Value.String()
	}
	inferredType := s.Type
	if s.Type != "" {
		pos := s.Pos()
		if s.TypePos.Line > 0 {
			pos = s.TypePos
		}
		c.validateTypeAnnotation(pos, s.Type)
	}
	if inferredType == "" && s.Value != nil {
		inferredType = c.inferType(s.Value)
	}

	namePos := s.NamePos
	if namePos.Line == 0 {
		namePos = s.Pos()
	}

	sym := &Symbol{
		Name:     s.Name,
		Kind:     kind,
		Type:     inferredType,
		Pos:      namePos,
		IsGlobal: s.IsGlobal,
		ValueStr: valStr,
	}
	if err := c.scope.Define(sym); err != nil {
		c.addError(s.Pos(), "ERR_DUPLICATE_SYMBOL", err.Error())
	}
	c.declaredVars = append(c.declaredVars, sym)

	c.checkExpression(s.Value)

	if s.Value != nil {
		actualValType := c.inferType(s.Value)
		if actualValType == "void" {
			c.addError(s.Value.Pos(), "ERR_TYPE_MISMATCH",
				fmt.Sprintf("Cannot assign expression of type 'void' to variable '%s'", s.Name))
		}
	}

	if s.Value != nil && s.Type != "" {
		if mapLit, isMap := s.Value.(*ast.MapLiteral); isMap {
			if c.interfaces != nil {
				if iface, ok := c.interfaces[s.Type]; ok {
					c.validateArgAgainstInterface(mapLit, iface, s.Name, "assignment", 0, false)
					return
				}
			}
			if strings.HasPrefix(s.Type, "@map") && strings.Contains(s.Type, "{") {
				c.validateArgAgainstInlineMap(mapLit, s.Type, s.Name, "assignment", 0, false)
				return
			}
		}
		actualType := c.inferType(s.Value)
		if !c.isTypeCompatible(s.Type, actualType) {
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

	if s.Includes != "" {
		c.validateTypeAnnotation(s.Pos(), s.Includes)
		if iface, ok := c.interfaces[s.Includes]; ok {
			for _, f := range iface.Fields {
				if f.Params != nil || strings.HasPrefix(f.Type, "func(") {
					retType := f.ReturnType
					if retType == "" && strings.Contains(f.Type, "->") {
						parts := strings.Split(f.Type, "->")
						retType = strings.TrimSpace(parts[len(parts)-1])
					}
					unitSym.Methods[f.Name] = &Symbol{
						Name:   f.Name,
						Kind:   SymFunc,
						Type:   retType,
						Pos:    f.Position,
						Params: f.Params,
					}
				}
			}
		}
	}

	for _, method := range s.Methods {
		if method.ReturnType != "" {
			c.validateTypeAnnotation(method.Pos(), method.ReturnType)
		}
		for _, p := range method.Parameters {
			if p.Type != "" {
				c.validateTypeAnnotation(method.Pos(), p.Type)
			}
		}
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
	unitScope.unitSym = unitSym
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
			if !c.isTypeCompatible(prop.Type, actualType) {
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
	if s.IsGlobal && c.scope.parent != nil {
		c.addError(s.Pos(), "ERR_GLOBAL_IN_BLOCK", "Cannot declare 'global/export func' inside a block, function, or loop. Global declarations are only permitted at the module root level.")
	}

	if s.ReturnType != "" {
		c.validateTypeAnnotation(s.Pos(), s.ReturnType)
	}
	minArgs := 0
	hasOptional := false
	isVariadic := false
	for _, p := range s.Parameters {
		if p.Type != "" {
			c.validateTypeAnnotation(s.Pos(), p.Type)
		}
		if p.IsVariadic {
			isVariadic = true
		} else if !p.Optional && !strings.HasSuffix(p.Type, "?") {
			minArgs++
		} else {
			hasOptional = true
		}
	}

	fnSym := &Symbol{
		Name:        s.Name,
		Kind:        SymFunc,
		Type:        s.ReturnType,
		Pos:         s.Pos(),
		Params:      s.Parameters,
		IsGlobal:    s.IsGlobal,
		MinArgs:     minArgs,
		HasOptional: hasOptional,
		IsVariadic:  isVariadic,
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
		paramType := p.Type
		if p.IsVariadic {
			if paramType != "" && paramType != "any" {
				paramType = "list[" + paramType + "]"
			} else {
				paramType = "list"
			}
		}
		paramSym := &Symbol{
			Name:     p.Name,
			Kind:     SymVar,
			Type:     paramType,
			Pos:      s.Pos(),
			Optional: p.Optional || strings.HasSuffix(paramType, "?") || p.IsVariadic,
		}
		if err := c.scope.Define(paramSym); err != nil {
			c.addError(s.Pos(), "ERR_DUPLICATE_SYMBOL", err.Error())
		}
	}

	for _, bodyStmt := range s.Body {
		c.checkStatement(bodyStmt)
	}

	cleanRet := strings.TrimSpace(s.ReturnType)
	if cleanRet != "" && !strings.EqualFold(cleanRet, "void") && !strings.EqualFold(cleanRet, "never") && cleanRet != "any" {
		if s.NativeHook == "" && s.Body != nil && !hasReturnStatement(s.Body) {
			c.addError(s.Pos(), "ERR_MISSING_RETURN",
				fmt.Sprintf("Function '%s' declares return type '%s' but has no return statement", s.Name, s.ReturnType))
		}
	}
}

func (c *Checker) checkIf(s *ast.IfStatement) {
	c.checkExpression(s.Condition)

	c.scope = NewScope(c.scope, false)
	for _, stmt := range s.Consequence {
		c.checkStatement(stmt)
	}
	c.scope = c.scope.parent

	for _, elif := range s.ElifBlocks {
		c.checkExpression(elif.Condition)
		c.scope = NewScope(c.scope, false)
		for _, stmt := range elif.Body {
			c.checkStatement(stmt)
		}
		c.scope = c.scope.parent
	}
	if len(s.Alternative) > 0 {
		c.scope = NewScope(c.scope, false)
		for _, stmt := range s.Alternative {
			c.checkStatement(stmt)
		}
		c.scope = c.scope.parent
	}
}

func (c *Checker) checkFor(s *ast.ForStatement) {
	c.checkExpression(s.Iterable)

	loopScope := NewScope(c.scope, false)
	c.scope = loopScope
	defer func() { c.scope = c.scope.parent }()

	itemType := "any"
	iterType := c.inferType(s.Iterable)
	if strings.HasPrefix(iterType, "list[") && strings.HasSuffix(iterType, "]") {
		itemType = strings.TrimSuffix(strings.TrimPrefix(iterType, "list["), "]")
	} else if iterType == "range" {
		itemType = "number"
	}

	itemSym := &Symbol{
		Name: s.Item,
		Kind: SymVar,
		Type: itemType,
		Pos:  s.Pos(),
	}
	_ = c.scope.Define(itemSym)

	if s.Index != "" {
		idxSym := &Symbol{
			Name: s.Index,
			Kind: SymVar,
			Type: "number",
			Pos:  s.Pos(),
		}
		_ = c.scope.Define(idxSym)
	}

	for _, stmt := range s.Body {
		c.checkStatement(stmt)
	}
}

func (c *Checker) checkWhile(s *ast.WhileStatement) {
	if s.Condition != nil {
		c.checkExpression(s.Condition)
	}

	loopScope := NewScope(c.scope, false)
	c.scope = loopScope
	defer func() { c.scope = c.scope.parent }()

	for _, stmt := range s.Body {
		c.checkStatement(stmt)
	}
}

func (c *Checker) checkMatch(s *ast.MatchStatement) {
	if s.Target != nil {
		c.checkExpression(s.Target)
	}
	for _, kase := range s.Cases {
		if kase.Pattern != nil {
			c.checkExpression(kase.Pattern)
		}
		c.scope = NewScope(c.scope, false)
		for _, stmt := range kase.Body {
			c.checkStatement(stmt)
		}
		c.scope = c.scope.parent
	}
	if len(s.DefaultBlock) > 0 {
		c.scope = NewScope(c.scope, false)
		for _, stmt := range s.DefaultBlock {
			c.checkStatement(stmt)
		}
		c.scope = c.scope.parent
	}
}

func (c *Checker) checkIn(stmt *ast.InStatement) {
	if stmt.Path != nil {
		c.checkExpression(stmt.Path)
		pType := c.inferType(stmt.Path)
		if pType != "string" && pType != "any" {
			c.addError(stmt.Path.Pos(), "ERR_TYPE_MISMATCH",
				fmt.Sprintf("In path expression expects string, got '%s'", pType))
		}
	}

	childScope := NewScope(c.scope, false)
	// Primitives de sandboxing locales
	childScope.Define(&Symbol{Name: "keep", Kind: SymFunc, Type: "any", Pos: stmt.Pos()})
	childScope.Define(&Symbol{Name: "Keep", Kind: SymFunc, Type: "any", Pos: stmt.Pos()})
	childScope.Define(&Symbol{Name: "drop", Kind: SymFunc, Type: "any", Pos: stmt.Pos()})
	childScope.Define(&Symbol{Name: "Drop", Kind: SymFunc, Type: "any", Pos: stmt.Pos()})
	childScope.Define(&Symbol{Name: "UseVariant", Kind: SymFunc, Type: "any", Pos: stmt.Pos()})
	childScope.Define(&Symbol{Name: "useVariant", Kind: SymFunc, Type: "any", Pos: stmt.Pos()})
	childScope.Define(&Symbol{Name: "Local", Kind: SymUnit, Type: "Unit", Pos: stmt.Pos()})

	oldScope := c.scope
	c.scope = childScope
	for _, bodyStmt := range stmt.Body {
		c.checkStatement(bodyStmt)
	}
	c.scope = oldScope
}

func (c *Checker) checkInterface(s *ast.InterfaceStatement) {
	if c.interfaces == nil {
		c.interfaces = make(map[string]*ast.InterfaceStatement)
	}
	c.interfaces[s.Name] = s
	fields := make(map[string]*Symbol)
	seenLower := make(map[string]string)
	for _, f := range s.Fields {
		lower := strings.ToLower(f.Name)
		if prev, exists := seenLower[lower]; exists {
			c.addError(f.Position, "ERR_DUPLICATE_FIELD",
				fmt.Sprintf("duplicate or case-colliding field '%s' (conflicts with '%s') in interface '%s'", f.Name, prev, s.Name))
			continue
		}
		seenLower[lower] = f.Name

		if s.IsMap {
			if f.Params != nil || f.Type == "function" || strings.HasPrefix(f.Type, "func(") || strings.HasPrefix(f.Type, "fn(") || strings.HasPrefix(f.Type, "(") {
				c.addError(f.Position, "ERR_MAP_METHOD_FORBIDDEN",
					fmt.Sprintf("Map interface '%s' cannot contain methods or functions (field '%s'); declare a unit instead.", s.Name, f.Name))
			}
		}

		if f.Params != nil {
			if f.ReturnType != "" {
				c.validateTypeAnnotation(f.Position, f.ReturnType)
			}
			for _, p := range f.Params {
				if p.Type != "" {
					c.validateTypeAnnotation(f.Position, p.Type)
				}
			}
		} else if f.Type != "" {
			c.validateTypeAnnotation(f.Position, f.Type)
		}

		fields[f.Name] = &Symbol{
			Name:     f.Name,
			Kind:     SymVar,
			Type:     f.Type,
			Pos:      f.Position,
			Optional: f.Optional,
		}
	}
	sym := &Symbol{
		Name:     s.Name,
		Kind:     SymInterface,
		Type:     s.Name,
		Pos:      s.Pos(),
		Methods:  fields,
		IfaceDef: s,
	}
	if err := c.scope.Define(sym); err != nil {
		c.addError(s.Pos(), "ERR_DUPLICATE_SYMBOL", err.Error())
	}
}

func hasReturnStatement(stmts []ast.Statement) bool {
	for _, st := range stmts {
		switch s := st.(type) {
		case *ast.ReturnStatement:
			return true
		case *ast.BlockStatement:
			if hasReturnStatement(s.Statements) {
				return true
			}
		case *ast.IfStatement:
			if hasReturnStatement(s.Consequence) || hasReturnStatement(s.Alternative) {
				return true
			}
			for _, el := range s.ElifBlocks {
				if hasReturnStatement(el.Body) {
					return true
				}
			}
		case *ast.ForStatement:
			if hasReturnStatement(s.Body) {
				return true
			}
		case *ast.WhileStatement:
			if hasReturnStatement(s.Body) {
				return true
			}
		case *ast.MatchStatement:
			if len(s.DefaultBlock) > 0 && hasReturnStatement(s.DefaultBlock) {
				return true
			}
			for _, mc := range s.Cases {
				if hasReturnStatement(mc.Body) {
					return true
				}
			}
		}
	}
	return false
}

