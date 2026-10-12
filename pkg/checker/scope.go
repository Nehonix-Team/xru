package checker

import (
	"fmt"

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
	Name        string
	Kind        SymbolKind
	Type        string // Pour les variables/constantes: type. Pour les fonctions: return type.
	Pos         lexer.Position
	IsGlobal    bool               // Déclaré avec 'global'
	ValueStr    string             // Valeur initiale / par défaut pour inspection
	Directives  map[string]bool    // Pour les #UNIT : directives déclarées
	Params      []*ast.Parameter   // Pour les fonctions : paramètres
	Methods     map[string]*Symbol // Pour les #UNIT : méthodes exposées
	Used        bool               // Indique si le symbole a été utilisé
	MinArgs     int                // Nombre minimum d'arguments obligatoires
	HasOptional bool               // Les paramètres au-delà de MinArgs sont optionnels (non variadique)
	IsVariadic  bool               // Vrai si accepte des arguments supplémentaires
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
	formatPos := func(pos lexer.Position) string {
		if pos.Line > 0 {
			return fmt.Sprintf(" at %s", pos)
		}
		return ""
	}

	// 1. Collision in current scope
	if existing, exists := s.symbols[sym.Name]; exists {
		if existing.Kind == SymUnit && sym.Kind == SymUnit {
			return fmt.Errorf("Module alias '%s' is already imported%s", sym.Name, formatPos(existing.Pos))
		} else if existing.Kind == SymUnit {
			return fmt.Errorf("Variable '%s' collides with imported module '%s'%s", sym.Name, existing.Name, formatPos(existing.Pos))
		} else if sym.Kind == SymUnit {
			return fmt.Errorf("Imported module '%s' collides with variable '%s'%s", sym.Name, existing.Name, formatPos(existing.Pos))
		}
		return fmt.Errorf("Identifier '%s' is already declared%s", sym.Name, formatPos(existing.Pos))
	}

	// 2. Collision with outer scope (e.g. imported module in parent scope)
	// Do not check parent collision if this is an isolated unit scope (properties accessed via &)
	if !s.isUnit && s.parent != nil {
		if existing, exists := s.parent.Resolve(sym.Name); exists {
			if existing.Kind == SymUnit && sym.Kind == SymUnit {
				return fmt.Errorf("Module alias '%s' is already imported%s", sym.Name, formatPos(existing.Pos))
			} else if existing.Kind == SymUnit {
				return fmt.Errorf("Variable '%s' collides with imported module '%s'%s", sym.Name, existing.Name, formatPos(existing.Pos))
			} else if sym.Kind == SymUnit {
				return fmt.Errorf("Imported module '%s' collides with existing identifier '%s'%s", sym.Name, existing.Name, formatPos(existing.Pos))
			}
		}
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
