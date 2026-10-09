package checker

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/Nehonix-Team/xru/pkg/ast"
	"github.com/Nehonix-Team/xru/pkg/lexer"
)

type SymbolKind string

const (
	SymVar       SymbolKind = "variable"
	SymConst     SymbolKind = "constant"
	SymUnit      SymbolKind = "unit"
	SymFunc      SymbolKind = "function"
	SymDirective SymbolKind = "directive"
)

type Symbol struct {
	Name       string
	Kind       SymbolKind
	Type       string // Pour les variables/constantes: type. Pour les fonctions: return type.
	Pos        lexer.Position
	Directives map[string]bool    // Pour les #UNIT : directives déclarées
	Params     []*ast.Parameter   // Pour les fonctions : paramètres
	Methods    map[string]*Symbol // Pour les #UNIT : méthodes exposées
}

type Scope struct {
	parent  *Scope
	symbols map[string]*Symbol
	isUnit  bool
}

func NewScope(parent *Scope, isUnit bool) *Scope {
	return &Scope{
		parent:  parent,
		symbols: make(map[string]*Symbol),
		isUnit:  isUnit,
	}
}

func (s *Scope) Define(sym *Symbol) error {
	if existing, exists := s.symbols[sym.Name]; exists {
		return fmt.Errorf("[ERR_DUPLICATE_SYMBOL] Identifier '%s' is already declared at %s", sym.Name, existing.Pos)
	}
	s.symbols[sym.Name] = sym
	return nil
}

func (s *Scope) Resolve(name string) (*Symbol, bool) {
	if sym, exists := s.symbols[name]; exists {
		return sym, true
	}
	if s.parent != nil {
		return s.parent.Resolve(name)
	}
	return nil, false
}

func (s *Scope) InUnitScope() bool {
	if s.isUnit {
		return true
	}
	if s.parent != nil {
		return s.parent.InUnitScope()
	}
	return false
}

type Checker struct {
	errors      []string
	scope       *Scope
	currentFunc *ast.FuncStatement
}

func New() *Checker {
	c := &Checker{
		errors: []string{},
		scope:  NewScope(nil, false),
	}
	c.initBuiltinUnits()
	return c
}

func (c *Checker) initBuiltinUnits() {
	netUnit := &Symbol{
		Name:       "Net",
		Kind:       SymUnit,
		Directives: map[string]bool{"set": true, "header": true, "on": true},
		Methods: map[string]*Symbol{
			"Fetch": {Name: "Fetch", Kind: SymFunc, Type: "any"},
			"Clone": {Name: "Clone", Kind: SymFunc, Type: "bool"},
			"Get":   {Name: "Get", Kind: SymFunc, Type: "string"},
		},
	}
	fsUnit := &Symbol{
		Name:       "Fs",
		Kind:       SymUnit,
		Directives: map[string]bool{"set": true},
		Methods: map[string]*Symbol{
			"Exists":    {Name: "Exists", Kind: SymFunc, Type: "bool"},
			"IsDir":     {Name: "IsDir", Kind: SymFunc, Type: "bool"},
			"IsFile":    {Name: "IsFile", Kind: SymFunc, Type: "bool"},
			"Read":      {Name: "Read", Kind: SymFunc, Type: "string"},
			"Size":      {Name: "Size", Kind: SymFunc, Type: "number"},
			"Remove":    {Name: "Remove", Kind: SymFunc, Type: "void"},
			"Move":      {Name: "Move", Kind: SymFunc, Type: "void"},
			"Copy":      {Name: "Copy", Kind: SymFunc, Type: "void"},
			"EnsureDir": {Name: "EnsureDir", Kind: SymFunc, Type: "void"},
		},
	}
	utilsUnit := &Symbol{
		Name: "Utils",
		Kind: SymUnit,
		Methods: map[string]*Symbol{
			"Log": {Name: "Log", Kind: SymFunc, Type: "void"},
		},
	}
	sysUnit := &Symbol{
		Name: "Sys",
		Kind: SymUnit,
		Methods: map[string]*Symbol{
			"GetEnv": {Name: "GetEnv", Kind: SymFunc, Type: "string"},
			"Os":     {Name: "Os", Kind: SymFunc, Type: "string"},
			"Arch":   {Name: "Arch", Kind: SymFunc, Type: "string"},
		},
	}
	cmdUnit := &Symbol{
		Name: "Cmd",
		Kind: SymUnit,
		Methods: map[string]*Symbol{
			"Exec": {Name: "Exec", Kind: SymFunc, Type: "string"},
		},
	}
	cryptoUnit := &Symbol{
		Name: "Crypto",
		Kind: SymUnit,
		Methods: map[string]*Symbol{
			"GetRandomBytes": {Name: "GetRandomBytes", Kind: SymFunc, Type: "string"},
			"RandomBytes":    {Name: "RandomBytes", Kind: SymFunc, Type: "string"},
			"RandomHex":      {Name: "RandomHex", Kind: SymFunc, Type: "string"},
			"RandomBase64":   {Name: "RandomBase64", Kind: SymFunc, Type: "string"},
			"Sha256":         {Name: "Sha256", Kind: SymFunc, Type: "string"},
			"Uuid":           {Name: "Uuid", Kind: SymFunc, Type: "string"},
		},
	}

	c.scope.Define(netUnit)
	c.scope.Define(fsUnit)
	c.scope.Define(utilsUnit)
	c.scope.Define(sysUnit)
	c.scope.Define(cmdUnit)
	c.scope.Define(cryptoUnit)
}

func (c *Checker) Errors() []string {
	return c.errors
}

func (c *Checker) addError(pos lexer.Position, code string, msg string) {
	c.errors = append(c.errors, fmt.Sprintf("%s: [%s] %s", pos, code, msg))
}

func (c *Checker) Check(prog *ast.Program) {
	for _, stmt := range prog.Statements {
		c.checkStatement(stmt)
	}
}

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

func (c *Checker) checkUse(s *ast.UseStatement) {
	alias := s.Alias
	if alias == "" {
		alias = s.Module
	}
	var sourceSym *Symbol
	for name, sym := range c.scope.symbols {
		if strings.EqualFold(name, s.Module) {
			sourceSym = sym
			break
		}
	}
	if sourceSym != nil {
		aliasSym := &Symbol{
			Name:       alias,
			Kind:       SymUnit,
			Type:       sourceSym.Type,
			Pos:        s.Pos(),
			Directives: sourceSym.Directives,
			Methods:    sourceSym.Methods,
		}
		c.scope.Define(aliasSym)
	} else {
		c.addError(s.Pos(), "ERR_UNKNOWN_MODULE", fmt.Sprintf("Module '%s' does not exist", s.Module))
	}
}

func (c *Checker) checkArg(s *ast.ArgStatement) {
	kind := SymVar
	sym := &Symbol{Name: s.Name, Kind: kind, Type: s.Type, Pos: s.Pos()}
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

	sym := &Symbol{Name: s.Name, Kind: kind, Type: s.Type, Pos: s.Pos()}
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
		Name:   s.Name,
		Kind:   SymFunc,
		Type:   s.ReturnType,
		Pos:    s.Pos(),
		Params: s.Parameters,
	}
	c.scope.Define(fnSym)

	oldFunc := c.currentFunc
	c.currentFunc = s
	defer func() { c.currentFunc = oldFunc }()

	fnScope := NewScope(c.scope, false)
	c.scope = fnScope
	defer func() { c.scope = c.scope.parent }()

	for _, p := range s.Parameters {
		paramSym := &Symbol{Name: p.Name, Kind: SymVar, Type: p.Type}
		c.scope.Define(paramSym)
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
			c.scope.Define(&Symbol{Name: p.Name, Kind: SymVar, Type: p.Type})
		}
		c.checkExpression(e.Body)
	case *ast.PropAccess:
		if !c.scope.InUnitScope() {
			c.addError(e.Pos(), "ERR_INVALID_PROP_ACCESS",
				fmt.Sprintf("Property access '&%s' is only allowed inside a #UNIT definition", e.Name))
		}
	case *ast.DurationLiteral:
		re := regexp.MustCompile(`^[0-9]+(\.[0-9]+)?(ms|s|m|h)$`)
		if !re.MatchString(e.Value) {
			c.addError(e.Pos(), "ERR_INVALID_DURATION",
				fmt.Sprintf("Invalid duration format '%s' (expected format: 500ms, 30s, 2m, 1h)", e.Value))
		}
	case *ast.CallExpression:
		c.checkExpression(e.Caller)
		for _, arg := range e.Arguments {
			c.checkExpression(arg)
		}
	case *ast.MemberExpression:
		c.checkExpression(e.Object)
	case *ast.InfixExpression:
		c.checkExpression(e.Left)
		c.checkExpression(e.Right)
	case *ast.BlockStatement:
		for _, stmt := range e.Statements {
			c.checkStatement(stmt)
		}
	}
}

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
	case *ast.InfixExpression:
		switch e.Operator {
		case "==", "!=", "<", "<=", ">", ">=", "&&", "||":
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
