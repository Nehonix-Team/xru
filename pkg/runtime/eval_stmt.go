package runtime

import (
	"strings"

	"github.com/Nehonix-Team/xru/pkg/ast"
)

func (ev *Evaluator) evalBlockStatement(block *ast.BlockStatement, env *Environment) (Value, error) {
	var result Value = NilValue{}
	for _, stmt := range block.Statements {
		res, err := ev.Eval(stmt, env)
		if err != nil {
			return nil, err
		}
		if ret, ok := res.(ReturnValue); ok {
			return ret, nil
		}
		result = res
	}
	return result, nil
}

func (ev *Evaluator) evalIf(ifStmt *ast.IfStatement, env *Environment) (Value, error) {
	condVal, err := ev.Eval(ifStmt.Condition, env)
	if err != nil {
		return nil, err
	}
	bVal, ok := condVal.(BoolValue)
	if ok && bVal.Val {
		for _, s := range ifStmt.Consequence {
			if _, err := ev.Eval(s, env); err != nil {
				return nil, err
			}
		}
		return NilValue{}, nil
	}

	for _, elif := range ifStmt.ElifBlocks {
		eVal, err := ev.Eval(elif.Condition, env)
		if err != nil {
			return nil, err
		}
		ebVal, ok := eVal.(BoolValue)
		if ok && ebVal.Val {
			for _, s := range elif.Body {
				if _, err := ev.Eval(s, env); err != nil {
					return nil, err
				}
			}
			return NilValue{}, nil
		}
	}

	for _, s := range ifStmt.Alternative {
		if _, err := ev.Eval(s, env); err != nil {
			return nil, err
		}
	}
	return NilValue{}, nil
}

func (ev *Evaluator) evalWith(with *ast.WithStatement, env *Environment) (Value, error) {
	targetVal, err := ev.Eval(with.Target, env)
	if err != nil {
		return nil, err
	}

	withEnv := NewEnvironment(env)
	if unitInst, ok := targetVal.(*UnitInstance); ok {
		withEnv.unitRef = unitInst
	}

	for _, stmt := range with.Body {
		_, err := ev.Eval(stmt, withEnv)
		if err != nil {
			return nil, err
		}
	}

	if withEnv.unitRef != nil && withEnv.unitRef.Methods != nil {
		if completeFn, ok := withEnv.unitRef.Methods["__complete"]; ok {
			return completeFn(nil)
		}
	}

	return NilValue{}, nil
}

func (ev *Evaluator) evalCustomDirective(dir *ast.CustomDirectiveInvocation, env *Environment) (Value, error) {
	var keyVal Value = NilValue{}
	var err error
	if dir.Key != nil {
		if ident, ok := dir.Key.(*ast.Identifier); ok {
			keyVal = StringValue{Val: ident.Value}
		} else {
			keyVal, err = ev.Eval(dir.Key, env)
			if err != nil {
				return nil, err
			}
		}
	}
	var valVal Value = NilValue{}
	if dir.Value != nil {
		valVal, err = ev.Eval(dir.Value, env)
		if err != nil {
			return nil, err
		}
	}
	if env.unitRef != nil && env.unitRef.Directives != nil {
		dirName := dir.Directive
		for dKey, handler := range env.unitRef.Directives {
			if strings.EqualFold(dKey, dirName) {
				return NilValue{}, handler(keyVal, valVal)
			}
		}
	}
	return NilValue{}, nil
}

func (ev *Evaluator) evalDispatchTree(dt *ast.DispatchTreeStatement, env *Environment) (Value, error) {
	treeEnv := NewEnvironment(env)
	for k, expr := range dt.PassArgs {
		val, err := ev.Eval(expr, env)
		if err != nil {
			return NilValue{}, err
		}
		treeEnv.Set(k, val, false)
	}

	if ev.treeDispatcher != nil {
		if err := ev.treeDispatcher(dt, ev, treeEnv); err != nil {
			return NilValue{}, err
		}
	}
	return BoolValue{Val: true}, nil
}
