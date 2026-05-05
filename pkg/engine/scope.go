package engine

import (
	"fmt"
	"strings"
	"github.com/Nehonix-Team/xru/internal/engine/ast"
)

// Scope représente l'espace de variables d'une règle ou d'un bloc.
type Scope struct {
	Vars     map[string]interface{}
	DefLines map[string]int
	Used     map[string]bool
	Modules  map[string]string // Alias -> ModuleName
	Parent   *Scope
	Capture  *strings.Builder // Pour capturer les sorties LOG lors de l'inception
	Runner   *Runner
}

func (s *Scope) Get(name string) (interface{}, bool) {
	parts := strings.Split(name, ".")
	rootName := parts[0]

	var current interface{}
	found := false

	if val, ok := s.Vars[rootName]; ok {
		if s.Used == nil {
			s.Used = make(map[string]bool)
		}
		s.Used[rootName] = true
		if rootName == "SERVERS" {
			if s.Runner.Verbose {
				fmt.Printf("[DEBUG] Scope %p: Marked 'SERVERS' as USED\n", s)
			}
		}
		current = val
		found = true
	} else if s.Parent != nil {
		raw, ok := s.Parent.Get(rootName)
		if ok {
			s.Parent.MarkUsed(rootName)
			current = raw
			found = true
		}
	}

	if !found {
		return "", false
	}

	for i := 1; i < len(parts); i++ {
		prop := parts[i]
		switch v := current.(type) {
		case ast.Object:
			if next, ok := v[prop]; ok {
				current = next
			} else {
				return nil, false
			}
		case map[string]interface{}:
			if next, ok := v[prop]; ok {
				current = next
			} else {
				return nil, false
			}
		default:
			return nil, false
		}
	}

	return current, true
}

func (s *Scope) RawGet(name string) (interface{}, bool) {
	parts := strings.Split(name, ".")
	rootName := parts[0]

	var current interface{}
	found := false

	if val, ok := s.Vars[rootName]; ok {
		current = val
		found = true
	} else if s.Parent != nil {
		raw, ok := s.Parent.RawGet(rootName)
		if ok {
			current = raw
			found = true
		}
	}

	if !found {
		return nil, false
	}

	for i := 1; i < len(parts); i++ {
		prop := parts[i]
		switch v := current.(type) {
		case ast.Object:
			if next, ok := v[prop]; ok {
				current = next
			} else {
				return nil, false
			}
		case map[string]interface{}:
			if next, ok := v[prop]; ok {
				current = next
			} else {
				return nil, false
			}
		default:
			return nil, false
		}
	}

	return current, true
}

func (s *Scope) MarkUsed(name string) {
	if s.Used == nil {
		s.Used = make(map[string]bool)
	}
	s.Used[name] = true
	if s.Parent != nil {
		s.Parent.MarkUsed(name)
	}
}

func (s *Scope) Set(name string, val interface{}, line int) {
	// Lexical update: si la variable existe dans un scope parent, on l'y met à jour
	curr := s
	for curr != nil {
		if _, ok := curr.Vars[name]; ok {
			curr.Vars[name] = val
			return
		}
		curr = curr.Parent
	}
	// Sinon, on la définit dans le scope courant
	if s.Vars == nil {
		s.Vars = make(map[string]interface{})
		s.DefLines = make(map[string]int)
	}
	s.Vars[name] = val
	s.DefLines[name] = line
}

func (s *Scope) RegisterModule(alias, name string, line int) error {
	if s.Modules == nil {
		s.Modules = make(map[string]string)
	}
	if existing, ok := s.Modules[alias]; ok {
		if existing == name {
			return nil
		}
		return fmt.Errorf("module alias '%s' already defined in this scope (points to '%s')", alias, existing)
	}
	s.Modules[alias] = name
	return nil
}

func (s *Scope) CheckUnused(file string) error {
	if s.Runner.Verbose {
		fmt.Printf("[DEBUG] CheckUnused for scope %p\n", s)
		for name := range s.DefLines {
			fmt.Printf("  - Variable '%s' defined. Used: %v\n", name, s.Used[name])
		}
	}
	for name, line := range s.DefLines {
		if !s.Used[name] {
			return fmt.Errorf("%s:%d: variable '%s' is defined but never used", file, line, name)
		}
	}
	return nil
}

func NewRootScope(r *Runner) *Scope {
	s := &Scope{
		Vars:     make(map[string]interface{}),
		DefLines: make(map[string]int),
		Used:     make(map[string]bool),
		Modules:  make(map[string]string),
		Runner:   r,
	}

	// Injection des arguments de terminal (--arg NAME=VAL)
	for i := 0; i < len(r.TerminalArgs); i++ {
		if r.TerminalArgs[i] == "--arg" && i+1 < len(r.TerminalArgs) {
			expr := r.TerminalArgs[i+1]
			parts := strings.SplitN(expr, "=", 2)
			if len(parts) == 2 {
				name := strings.TrimSpace(parts[0])
				val := strings.TrimSpace(parts[1])
				s.Vars[name] = ast.Literal(val)
				s.Used[name] = false // Sera marqué utilisé lors du Get
				if r.Verbose {
					fmt.Printf("[DEBUG] Injected terminal arg: %s=%s\n", name, val)
				}
			}
			i++ // Skip the value
		}
	}

	return s
}