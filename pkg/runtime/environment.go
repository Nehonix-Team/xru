package runtime

import (
	"fmt"
	"strings"
)

type Environment struct {
	parent          *Environment
	store           map[string]Value
	globals         map[string]Value
	constants       map[string]bool
	globalConstants map[string]bool
	unitRef         *UnitInstance // Pointeur vers l'unité courante si dans #UNIT
	currentDir      string
	isBlockScope    bool // Vrai si dans une fonction, un bloc { }, une boucle, etc.
}

func NewEnvironment(parent *Environment) *Environment {
	var g map[string]Value
	var gc map[string]bool
	var uRef *UnitInstance
	if parent != nil {
		g = parent.globals
		gc = parent.globalConstants
		uRef = parent.unitRef
	} else {
		g = make(map[string]Value)
		gc = make(map[string]bool)
	}
	return &Environment{
		parent:          parent,
		store:           make(map[string]Value),
		globals:         g,
		constants:       make(map[string]bool),
		globalConstants: gc,
		unitRef:         uRef,
		isBlockScope:    false,
	}
}

func NewBlockEnvironment(parent *Environment) *Environment {
	env := NewEnvironment(parent)
	env.isBlockScope = true
	return env
}

func (e *Environment) IsBlockScope() bool {
	if e.isBlockScope {
		return true
	}
	if e.parent != nil {
		return e.parent.IsBlockScope()
	}
	return false
}

func (e *Environment) SetGlobal(name string, val Value) {
	e.globals[name] = val
}

func (e *Environment) SetGlobalConst(name string) {
	e.globalConstants[name] = true
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
		return fmt.Errorf("cannot reassign constant '%s': only variables defined with 'let' can be modified", name)
	}
	e.store[name] = val
	if isConst {
		e.constants[name] = true
	}
	return nil
}

func (e *Environment) IsConst(name string) bool {
	if e.constants[name] || e.globalConstants[name] {
		return true
	}
	if e.parent != nil {
		return e.parent.IsConst(name)
	}
	return false
}

func (e *Environment) Reassign(name string, val Value) error {
	if strings.HasPrefix(name, "&") {
		propName := strings.TrimPrefix(name, "&")
		if e.unitRef != nil {
			if _, ok := e.unitRef.Properties[propName]; ok {
				e.unitRef.Properties[propName] = val
				return nil
			}
			return fmt.Errorf("undefined property '&%s' on unit '%s'", propName, e.unitRef.UnitName)
		}
		if e.parent != nil {
			return e.parent.Reassign(name, val)
		}
		return fmt.Errorf("cannot mutate internal property '%s' outside of unit context", name)
	}
	if e.IsConst(name) {
		return fmt.Errorf("cannot reassign constant '%s': only variables defined with 'let' can be modified", name)
	}
	if _, ok := e.store[name]; ok {
		e.store[name] = val
		return nil
	}
	if _, ok := e.globals[name]; ok {
		e.globals[name] = val
		return nil
	}
	if e.parent != nil {
		return e.parent.Reassign(name, val)
	}
	return fmt.Errorf("cannot reassign undefined variable '%s'", name)
}

func (e *Environment) Get(name string) (Value, bool) {
	if val, ok := e.store[name]; ok {
		return val, true
	}
	if val, ok := e.globals[name]; ok {
		return val, true
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
