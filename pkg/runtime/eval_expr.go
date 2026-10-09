package runtime

import (
	"github.com/Nehonix-Team/xru/pkg/ast"
)

func (ev *Evaluator) evalInfix(infix *ast.InfixExpression, env *Environment) (Value, error) {
	left, err := ev.Eval(infix.Left, env)
	if err != nil {
		return nil, err
	}
	right, err := ev.Eval(infix.Right, env)
	if err != nil {
		return nil, err
	}

	switch infix.Operator {
	case "==":
		if left.Type() == ValNil && right.Type() == ValNil {
			return BoolValue{Val: true}, nil
		}
		if left.Type() == ValNil || right.Type() == ValNil {
			return BoolValue{Val: false}, nil
		}
		return BoolValue{Val: left.String() == right.String()}, nil
	case "!=":
		if left.Type() == ValNil && right.Type() == ValNil {
			return BoolValue{Val: false}, nil
		}
		if left.Type() == ValNil || right.Type() == ValNil {
			return BoolValue{Val: true}, nil
		}
		return BoolValue{Val: left.String() != right.String()}, nil
	}
	return BoolValue{Val: false}, nil
}
