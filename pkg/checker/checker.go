package checker

import (
	"fmt"

	"github.com/Nehonix-Team/xru/pkg/ast"
	"github.com/Nehonix-Team/xru/pkg/lexer"
	"github.com/Nehonix-Team/xru/pkg/modules"
)

type Checker struct {
	errors           []string
	scope            *Scope
	currentFunc      *ast.FuncStatement
	currentFile      string
	availableModules map[string]*Symbol
}

func New() *Checker {
	c := &Checker{
		errors:           []string{},
		scope:            NewScope(nil, false),
		availableModules: make(map[string]*Symbol),
	}
	c.initBuiltinUnits()
	return c
}

func (c *Checker) SetCurrentFile(file string) {
	c.currentFile = file
}

func (c *Checker) DefineVar(name string) {
	_ = c.scope.Define(&Symbol{Name: name, Kind: SymVar, Type: "any"})
}

func (c *Checker) Errors() []string {
	return c.errors
}

func (c *Checker) GetScope() *Scope {
	return c.scope
}

func (c *Checker) addError(pos lexer.Position, code string, msg string) {
	c.errors = append(c.errors, fmt.Sprintf("%s: [%s] %s", pos, code, msg))
}

func (c *Checker) Check(prog *ast.Program) {
	if c.currentFile == "" && len(prog.Statements) > 0 {
		c.currentFile = prog.Statements[0].Pos().Filename
	}

	// 1. Règle stricte pour _mod.xru : SEULES les déclarations 'map { ... }' y sont autorisées
	if c.currentFile != "" && modules.IsModConfigFile(c.currentFile) {
		for _, stmt := range prog.Statements {
			if _, ok := stmt.(*ast.MapStatement); !ok {
				c.addError(stmt.Pos(), "ERR_INVALID_MOD_DECLARATION",
					"'_mod.xru' is strictly reserved for module mapping definitions ('map { ... }'). Executable code, statements, variables, and imports are not allowed.")
			}
		}
	}

	for _, stmt := range prog.Statements {
		c.checkStatement(stmt)
	}
}
