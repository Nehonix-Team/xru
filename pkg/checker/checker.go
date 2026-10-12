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
	hasModStmt       bool
	declaredVars     []*Symbol
	importedSyms     []*Symbol        // Alias créés par 'use' (pour détecter les imports inutilisés)
	unreachable      []lexer.Position // Instructions situées après un appel 'never' (std.fatal, Sys.exit)
}

func New() *Checker {
	c := &Checker{
		errors:           []string{},
		scope:            NewScope(nil, false),
		availableModules: make(map[string]*Symbol),
		declaredVars:     []*Symbol{},
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

func (c *Checker) AllDeclaredVars() []*Symbol {
	return c.declaredVars
}

func (c *Checker) UnusedSymbols() []*Symbol {
	var unused []*Symbol
	for _, sym := range c.declaredVars {
		if sym.Used {
			continue
		}
		if sym.Name == "" || sym.Name[0] == '_' {
			continue
		}
		// Si déclaré global dans un fichier module, c'est un export public destiné à l'extérieur
		if sym.IsGlobal {
			continue
		}
		unused = append(unused, sym)
	}
	return unused
}

// UnusedImports retourne les modules importés via 'use' qui ne sont jamais référencés.
func (c *Checker) UnusedImports() []*Symbol {
	var unused []*Symbol
	for _, sym := range c.importedSyms {
		if !sym.Used {
			unused = append(unused, sym)
		}
	}
	return unused
}

// UnreachablePositions retourne les positions des instructions jamais exécutées
// car situées après un appel de type 'never' (std.fatal, Sys.exit).
func (c *Checker) UnreachablePositions() []lexer.Position {
	return c.unreachable
}

// checkStatementList vérifie une séquence d'instructions et marque comme
// inatteignables celles qui suivent un appel de type 'never'.
func (c *Checker) checkStatementList(stmts []ast.Statement) {
	terminated := false
	for _, stmt := range stmts {
		if terminated {
			c.unreachable = append(c.unreachable, stmt.Pos())
		}
		c.checkStatement(stmt)
		if es, ok := stmt.(*ast.ExpressionStatement); ok && es.Expression != nil {
			if c.inferType(es.Expression) == "never" {
				terminated = true
			}
		}
	}
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

	c.checkStatementList(prog.Statements)
}
