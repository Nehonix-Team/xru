package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/Nehonix-Team/xru/internal/engine"
)

// Scope représente l'espace de variables d'une règle ou d'un bloc.
type Scope struct {
	Vars     map[string]interface{}
	DefLines map[string]int
	Used     map[string]bool
	Modules  map[string]string // Alias -> ModuleName
	Parent   *Scope
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
		case engine.Object:
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
		case engine.Object:
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

func (s *Scope) RegisterModule(alias, name string, line int) {
	if s.Modules == nil {
		s.Modules = make(map[string]string)
	}
	if existing, ok := s.Modules[alias]; ok {
		if existing == name {
			return
		}
		fmt.Printf("%s:%d: %serror:%s module alias '%s' already defined in this scope (points to '%s')\n",
			currentFile, line, colorRed, colorReset, alias, existing)
		os.Exit(1)
	}
	s.Modules[alias] = name
}

func (s *Scope) CheckUnused(file string) {
	if verbose {
		fmt.Printf("[DEBUG] CheckUnused for scope %p\n", s)
		for name := range s.DefLines {
			fmt.Printf("  - Variable '%s' defined. Used: %v\n", name, s.Used[name])
		}
	}
	for name, line := range s.DefLines {
		if !s.Used[name] {
			fmt.Printf("%s:%d: %serror:%s variable '%s' is defined but never used\n",
				file, line, colorRed, colorReset, name)
			os.Exit(1)
		}
	}
}

func newRootScope() *Scope {
	return &Scope{
		Vars:     make(map[string]interface{}),
		DefLines: make(map[string]int),
		Used:     make(map[string]bool),
		Modules:  make(map[string]string),
	}
}
