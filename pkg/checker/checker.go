package checker

import (
	"fmt"
	"regexp"

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
	Type       string
	Pos        lexer.Position
	Directives map[string]bool // Pour les #UNIT : directives déclarées
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
	errors []string
	scope  *Scope
}

func New() *Checker {
	c := &Checker{
		errors: []string{},
		scope:  NewScope(nil, false),
	}
	// Initialisation des #UNIT natives dans le scope racine
	c.initBuiltinUnits()
	return c
}

func (c *Checker) initBuiltinUnits() {
	netUnit := &Symbol{
		Name:       "Net",
		Kind:       SymUnit,
		Directives: map[string]bool{"#SET": true, "#HEADER": true, "#ON": true},
	}
	fsUnit := &Symbol{
		Name:       "Fs",
		Kind:       SymUnit,
		Directives: map[string]bool{"#SET": true},
	}
	utilsUnit := &Symbol{
		Name: "Utils",
		Kind: SymUnit,
	}
	sysUnit := &Symbol{
		Name: "Sys",
		Kind: SymUnit,
	}
	cmdUnit := &Symbol{
		Name: "Cmd",
		Kind: SymUnit,
	}
	c.scope.Define(netUnit)
	c.scope.Define(fsUnit)
	c.scope.Define(utilsUnit)
	c.scope.Define(sysUnit)
	c.scope.Define(cmdUnit)
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
	case *ast.ExpressionStatement:
		c.checkExpression(s.Expression)
	case *ast.FileStatement:
		for _, bodyStmt := range s.Body {
			c.checkStatement(bodyStmt)
		}
	case *ast.DispatchTreeStatement:
		// DispatchTree statement is valid
	}
}

func (c *Checker) checkArg(s *ast.ArgStatement) {
	kind := SymVar
	sym := &Symbol{Name: s.Name, Kind: kind, Type: s.Type, Pos: s.Pos()}
	if err := c.scope.Define(sym); err != nil {
		c.addError(s.Pos(), "ERR_DUPLICATE_SYMBOL", err.Error())
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
}

func (c *Checker) checkUnit(s *ast.UnitStatement) {
	sym := &Symbol{
		Name:       s.Name,
		Kind:       SymUnit,
		Pos:        s.Pos(),
		Directives: make(map[string]bool),
	}

	// Enregistre les directives autorisées pour cette unité
	for _, dir := range s.Directives {
		dirName := "#" + dir.Name
		sym.Directives[dirName] = true
	}

	if err := c.scope.Define(sym); err != nil {
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
	}

	for _, method := range s.Methods {
		c.checkFunc(method)
	}
}

func (c *Checker) checkWith(s *ast.WithStatement) {
	c.checkExpression(s.Target)

	// Identifier l'unité ciblée par le bloc #WITH
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
			if targetUnitSym != nil && len(targetUnitSym.Directives) > 0 {
				if !targetUnitSym.Directives[customDir.Directive] {
					c.addError(customDir.Pos(), "ERR_ORPHAN_DIRECTIVE",
						fmt.Sprintf("Directive '%s' is not defined for #UNIT '%s'", customDir.Directive, targetUnitSym.Name))
				}
			}
		}
	}
}

func (c *Checker) checkFunc(s *ast.FuncStatement) {
	fnSym := &Symbol{Name: s.Name, Kind: SymFunc, Pos: s.Pos()}
	c.scope.Define(fnSym)

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
	case *ast.PropAccess:
		// Règle stricte : &prop n'est autorisé que DANS une #UNIT
		if !c.scope.InUnitScope() {
			c.addError(e.Pos(), "ERR_INVALID_PROP_ACCESS",
				fmt.Sprintf("Property access '&%s' is only allowed inside a #UNIT definition", e.Name))
		}
	case *ast.DurationLiteral:
		// Validation stricte de format duration
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
	}
}
