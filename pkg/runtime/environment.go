package runtime

import (
	"fmt"
	"strings"
)

type Environment struct {
	parent     *Environment
	store      map[string]Value
	globals    map[string]Value
	constants  map[string]bool
	unitRef    *UnitInstance // Pointeur vers l'unité courante si dans #UNIT
	currentDir string
}

func NewEnvironment(parent *Environment) *Environment {
	var g map[string]Value
	if parent != nil {
		g = parent.globals
	} else {
		g = make(map[string]Value)
	}
	return &Environment{
		parent:    parent,
		store:     make(map[string]Value),
		globals:   g,
		constants: make(map[string]bool),
	}
}

func (e *Environment) SetGlobal(name string, val Value) {
	e.globals[name] = val
}

func (e *Environment) SetCurrentDir(dir string) {
	e.currentDir = dir
}

func (e *Environment) CurrentDir() string {
	if e.currentDir != "" {
		return e.currentDir
	}
	if e.parent != nil {
		return e.parent.CurrentDir()
	}
	return ""
}

func (e *Environment) All() map[string]Value {
	res := make(map[string]Value)
	for k, v := range e.globals {
		res[k] = v
	}
	if e.parent != nil {
		for k, v := range e.parent.All() {
			res[k] = v
		}
	}
	for k, v := range e.store {
		res[k] = v
	}
	return res
}

func (e *Environment) Set(name string, val Value, isConst bool) error {
	if e.constants[name] {
		return fmt.Errorf("cannot reassign constant '%s'", name)
	}
	e.store[name] = val
	if isConst {
		e.constants[name] = true
	}
	return nil
}

func (e *Environment) Get(name string) (Value, bool) {
	if val, ok := e.store[name]; ok {
		return val, true
	}
	if val, ok := e.globals[name]; ok {
		return val, true
	}
	for k, v := range e.store {
		if strings.EqualFold(k, name) {
			return v, true
		}
	}
	for k, v := range e.globals {
		if strings.EqualFold(k, name) {
			return v, true
		}
	}
	if e.parent != nil {
		return e.parent.Get(name)
	}
	return nil, false
}

func (e *Environment) GetLocal(name string) (Value, bool) {
	if val, ok := e.store[name]; ok {
		return val, true
	}
	if e.parent != nil {
		return e.parent.GetLocal(name)
	}
	return nil, false
}

func (e *Environment) GetGlobal(name string) (Value, bool) {
	if val, ok := e.globals[name]; ok {
		return val, true
	}
	if e.parent != nil {
		return e.parent.GetGlobal(name)
	}
	return nil, false
}
