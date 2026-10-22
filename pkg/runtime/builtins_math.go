package runtime

import (
	"fmt"
	"math"
	mathRand "math/rand"
)

func newMathUnit() *UnitInstance {
	return &UnitInstance{
		UnitName:   "Math",
		Properties: make(map[string]Value),
		Methods: map[string]NativeMethod{
			"Round": func(args []Value) (Value, error) {
				if len(args) < 1 {
					return NilValue{}, fmt.Errorf("Math.Round requires 1 number argument, got 0")
				}
				n, ok := args[0].(NumberValue)
				if !ok {
					return NilValue{}, fmt.Errorf("Math.Round expects number argument, got %s", args[0].Type())
				}
				return NumberValue{Val: math.Round(n.Val)}, nil
			},
			"Floor": func(args []Value) (Value, error) {
				if len(args) < 1 {
					return NilValue{}, fmt.Errorf("Math.Floor requires 1 number argument, got 0")
				}
				n, ok := args[0].(NumberValue)
				if !ok {
					return NilValue{}, fmt.Errorf("Math.Floor expects number argument, got %s", args[0].Type())
				}
				return NumberValue{Val: math.Floor(n.Val)}, nil
			},
			"Ceil": func(args []Value) (Value, error) {
				if len(args) < 1 {
					return NilValue{}, fmt.Errorf("Math.Ceil requires 1 number argument, got 0")
				}
				n, ok := args[0].(NumberValue)
				if !ok {
					return NilValue{}, fmt.Errorf("Math.Ceil expects number argument, got %s", args[0].Type())
				}
				return NumberValue{Val: math.Ceil(n.Val)}, nil
			},
			"Abs": func(args []Value) (Value, error) {
				if len(args) < 1 {
					return NilValue{}, fmt.Errorf("Math.Abs requires 1 number argument, got 0")
				}
				n, ok := args[0].(NumberValue)
				if !ok {
					return NilValue{}, fmt.Errorf("Math.Abs expects number argument, got %s", args[0].Type())
				}
				return NumberValue{Val: math.Abs(n.Val)}, nil
			},
			"Min": func(args []Value) (Value, error) {
				if len(args) < 2 {
					return NilValue{}, fmt.Errorf("Math.Min requires 2 numbers")
				}
				a, ok1 := args[0].(NumberValue)
				b, ok2 := args[1].(NumberValue)
				if !ok1 || !ok2 {
					return NilValue{}, fmt.Errorf("Math.Min expects number arguments")
				}
				return NumberValue{Val: math.Min(a.Val, b.Val)}, nil
			},
			"Max": func(args []Value) (Value, error) {
				if len(args) < 2 {
					return NilValue{}, fmt.Errorf("Math.Max requires 2 numbers")
				}
				a, ok1 := args[0].(NumberValue)
				b, ok2 := args[1].(NumberValue)
				if !ok1 || !ok2 {
					return NilValue{}, fmt.Errorf("Math.Max expects number arguments")
				}
				return NumberValue{Val: math.Max(a.Val, b.Val)}, nil
			},
			"Random": func(args []Value) (Value, error) {
				return NumberValue{Val: mathRand.Float64()}, nil
			},
		},
		Directives: make(map[string]NativeDirective),
	}
}
