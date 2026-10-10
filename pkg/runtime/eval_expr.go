package runtime

import (
	"fmt"
	"math"

	"github.com/Nehonix-Team/xru/pkg/ast"
)

func (ev *Evaluator) evalInfix(infix *ast.InfixExpression, env *Environment) (Value, error) {
	left, err := ev.Eval(infix.Left, env)
	if err != nil {
		return nil, err
	}

	// Évaluation en court-circuit pour les opérateurs logiques && et || / or
	switch infix.Operator {
	case "&&":
		if !isTruthy(left) {
			return BoolValue{Val: false}, nil
		}
		right, err := ev.Eval(infix.Right, env)
		if err != nil {
			return nil, err
		}
		return BoolValue{Val: isTruthy(right)}, nil

	case "||", "or":
		if isTruthy(left) {
			return BoolValue{Val: true}, nil
		}
		right, err := ev.Eval(infix.Right, env)
		if err != nil {
			return nil, err
		}
		return BoolValue{Val: isTruthy(right)}, nil
	}

	right, err := ev.Eval(infix.Right, env)
	if err != nil {
		return nil, err
	}

	switch infix.Operator {
	case "+":
		// Concaténation si l'un est chaîne
		if left.Type() == ValString || right.Type() == ValString {
			return StringValue{Val: left.String() + right.String()}, nil
		}
		if leftNum, ok := left.(NumberValue); ok {
			if rightNum, ok := right.(NumberValue); ok {
				return NumberValue{Val: leftNum.Val + rightNum.Val}, nil
			}
		}
		return nil, fmt.Errorf("operator '+' not supported between %s and %s", left.Type(), right.Type())

	case "-":
		if leftNum, ok := left.(NumberValue); ok {
			if rightNum, ok := right.(NumberValue); ok {
				return NumberValue{Val: leftNum.Val - rightNum.Val}, nil
			}
		}
		return nil, fmt.Errorf("operator '-' not supported between %s and %s", left.Type(), right.Type())

	case "*":
		if leftNum, ok := left.(NumberValue); ok {
			if rightNum, ok := right.(NumberValue); ok {
				return NumberValue{Val: leftNum.Val * rightNum.Val}, nil
			}
		}
		return nil, fmt.Errorf("operator '*' not supported between %s and %s", left.Type(), right.Type())

	case "/":
		if leftNum, ok := left.(NumberValue); ok {
			if rightNum, ok := right.(NumberValue); ok {
				if rightNum.Val == 0 {
					return nil, fmt.Errorf("division by zero")
				}
				return NumberValue{Val: leftNum.Val / rightNum.Val}, nil
			}
		}
		return nil, fmt.Errorf("operator '/' not supported between %s and %s", left.Type(), right.Type())

	case "%":
		if leftNum, ok := left.(NumberValue); ok {
			if rightNum, ok := right.(NumberValue); ok {
				if rightNum.Val == 0 {
					return nil, fmt.Errorf("modulo by zero")
				}
				return NumberValue{Val: math.Mod(leftNum.Val, rightNum.Val)}, nil
			}
		}
		return nil, fmt.Errorf("operator '%%' not supported between %s and %s", left.Type(), right.Type())

	case "<":
		if leftNum, ok := left.(NumberValue); ok {
			if rightNum, ok := right.(NumberValue); ok {
				return BoolValue{Val: leftNum.Val < rightNum.Val}, nil
			}
		}
		return BoolValue{Val: left.String() < right.String()}, nil

	case "<=":
		if leftNum, ok := left.(NumberValue); ok {
			if rightNum, ok := right.(NumberValue); ok {
				return BoolValue{Val: leftNum.Val <= rightNum.Val}, nil
			}
		}
		return BoolValue{Val: left.String() <= right.String()}, nil

	case ">":
		if leftNum, ok := left.(NumberValue); ok {
			if rightNum, ok := right.(NumberValue); ok {
				return BoolValue{Val: leftNum.Val > rightNum.Val}, nil
			}
		}
		return BoolValue{Val: left.String() > right.String()}, nil

	case ">=":
		if leftNum, ok := left.(NumberValue); ok {
			if rightNum, ok := right.(NumberValue); ok {
				return BoolValue{Val: leftNum.Val >= rightNum.Val}, nil
			}
		}
		return BoolValue{Val: left.String() >= right.String()}, nil

	case "==":
		if left.Type() == ValNil && right.Type() == ValNil {
			return BoolValue{Val: true}, nil
		}
		if left.Type() == ValNil || right.Type() == ValNil {
			return BoolValue{Val: false}, nil
		}
		if leftNum, ok := left.(NumberValue); ok {
			if rightNum, ok := right.(NumberValue); ok {
				return BoolValue{Val: leftNum.Val == rightNum.Val}, nil
			}
		}
		if leftBool, ok := left.(BoolValue); ok {
			if rightBool, ok := right.(BoolValue); ok {
				return BoolValue{Val: leftBool.Val == rightBool.Val}, nil
			}
		}
		return BoolValue{Val: left.String() == right.String()}, nil

	case "!=":
		if left.Type() == ValNil && right.Type() == ValNil {
			return BoolValue{Val: false}, nil
		}
		if left.Type() == ValNil || right.Type() == ValNil {
			return BoolValue{Val: true}, nil
		}
		if leftNum, ok := left.(NumberValue); ok {
			if rightNum, ok := right.(NumberValue); ok {
				return BoolValue{Val: leftNum.Val != rightNum.Val}, nil
			}
		}
		if leftBool, ok := left.(BoolValue); ok {
			if rightBool, ok := right.(BoolValue); ok {
				return BoolValue{Val: leftBool.Val != rightBool.Val}, nil
			}
		}
		return BoolValue{Val: left.String() != right.String()}, nil
	}
	return BoolValue{Val: false}, nil
}

func (ev *Evaluator) evalPrefix(prefix *ast.PrefixExpression, env *Environment) (Value, error) {
	val, err := ev.Eval(prefix.Right, env)
	if err != nil {
		return nil, err
	}
	switch prefix.Operator {
	case "!":
		return BoolValue{Val: !isTruthy(val)}, nil
	case "-":
		if num, ok := val.(NumberValue); ok {
			return NumberValue{Val: -num.Val}, nil
		}
		return nil, fmt.Errorf("unary '-' not supported on %s", val.Type())
	default:
		return nil, fmt.Errorf("unsupported prefix operator '%s'", prefix.Operator)
	}
}

func isTruthy(val Value) bool {
	if val == nil {
		return false
	}
	switch v := val.(type) {
	case NilValue:
		return false
	case BoolValue:
		return v.Val
	case NumberValue:
		return v.Val != 0
	case StringValue:
		return v.Val != ""
	case ListValue:
		return len(v.Elements) > 0
	case MapValue:
		return len(v.Pairs) > 0
	default:
		return true
	}
}
