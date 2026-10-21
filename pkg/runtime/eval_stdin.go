package runtime

import (
	"fmt"

	"github.com/Nehonix-Team/xru/pkg/ast"
)

func (ev *Evaluator) evalStdInStatement(s *ast.StdInStatement, env *Environment) (Value, error) {
	stdUnit, ok := ev.builtins["std"]
	if !ok {
		return nil, fmt.Errorf("std module not available for std::in block")
	}

	blockEnv := NewBlockEnvironment(env)

	inputFn := stdUnit.Methods["input"]
	selectFn := stdUnit.Methods["select"]
	confirmFn := stdUnit.Methods["confirm"]

	prevInput, hadInput := env.GetGlobal("input")
	prevSelect, hadSelect := env.GetGlobal("select")
	prevConfirm, hadConfirm := env.GetGlobal("confirm")

	if inputFn != nil {
		env.SetGlobal("input", FunctionValue{Fn: inputFn})
	}
	if selectFn != nil {
		env.SetGlobal("select", FunctionValue{Fn: selectFn})
	}
	if confirmFn != nil {
		env.SetGlobal("confirm", FunctionValue{Fn: confirmFn})
	}

	defer func() {
		if hadInput {
			env.SetGlobal("input", prevInput)
		} else {
			delete(env.globals, "input")
		}
		if hadSelect {
			env.SetGlobal("select", prevSelect)
		} else {
			delete(env.globals, "select")
		}
		if hadConfirm {
			env.SetGlobal("confirm", prevConfirm)
		} else {
			delete(env.globals, "confirm")
		}
	}()

	if s.Body == nil {
		return NilValue{}, nil
	}

	return ev.Eval(s.Body, blockEnv)
}
