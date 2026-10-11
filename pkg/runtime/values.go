package runtime

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Nehonix-Team/xru/pkg/ast"
)

type ValueType string

const (
	ValString   ValueType = "string"
	ValNumber   ValueType = "number"
	ValBool     ValueType = "bool"
	ValDuration ValueType = "duration"
	ValList     ValueType = "list"
	ValMap      ValueType = "map"
	ValUnit     ValueType = "unit"
	ValTask     ValueType = "task"
	ValNil      ValueType = "nil"
)

type Value interface {
	Type() ValueType
	String() string
}

type StringValue struct{ Val string }
func (v StringValue) Type() ValueType { return ValString }
func (v StringValue) String() string  { return v.Val }

type NumberValue struct{ Val float64 }
func (v NumberValue) Type() ValueType { return ValNumber }
func (v NumberValue) String() string  { return fmt.Sprintf("%v", v.Val) }

type BoolValue struct{ Val bool }
func (v BoolValue) Type() ValueType { return ValBool }
func (v BoolValue) String() string  { return fmt.Sprintf("%t", v.Val) }

type DurationValue struct{ Val time.Duration }
func (v DurationValue) Type() ValueType { return ValDuration }
func (v DurationValue) String() string  { return v.Val.String() }

type ListValue struct{ Elements []Value }
func (v ListValue) Type() ValueType { return ValList }
func (v ListValue) String() string {
	var s []string
	for _, elem := range v.Elements {
		s = append(s, elem.String())
	}
	return "[" + strings.Join(s, ", ") + "]"
}

type MapValue struct{ Pairs map[string]Value }
func (v MapValue) Type() ValueType { return ValMap }
func (v MapValue) String() string {
	var s []string
	for k, val := range v.Pairs {
		s = append(s, fmt.Sprintf("%s -> %s", k, val.String()))
	}
	return "[" + strings.Join(s, ", ") + "]"
}

type NilValue struct{}
func (v NilValue) Type() ValueType { return ValNil }
func (v NilValue) String() string  { return "nil" }

type RangeValue struct {
	Start int64
	End   int64
}

func (r RangeValue) Type() ValueType { return "range" }
func (r RangeValue) String() string  { return fmt.Sprintf("%d..%d", r.Start, r.End) }

// UnitInstance représente une instance vivante d'une #UNIT
type UnitInstance struct {
	UnitName    string
	Properties  map[string]Value
	Methods     map[string]NativeMethod
	Directives  map[string]NativeDirective
	MethodDefs  []*ast.FuncStatement
	IsPrototype bool
}

func (u *UnitInstance) Type() ValueType { return ValUnit }
func (u *UnitInstance) String() string  { return fmt.Sprintf("@%s()", u.UnitName) }

// TaskValue représente une tâche asynchrone concurrente lancée via 'with <expr>'
type TaskValue struct {
	done   chan struct{}
	result Value
	err    error
	mu     sync.Mutex
	isDone bool
	cancel context.CancelFunc
}

func (t *TaskValue) Type() ValueType { return ValTask }
func (t *TaskValue) String() string  { return "<Task>" }

func (t *TaskValue) Wait() (Value, error) {
	<-t.done
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.err != nil {
		return NilValue{}, t.err
	}
	if t.result == nil {
		return NilValue{}, nil
	}
	return t.result, nil
}

func (t *TaskValue) IsDone() BoolValue {
	t.mu.Lock()
	defer t.mu.Unlock()
	return BoolValue{Val: t.isDone}
}

func (t *TaskValue) Cancel() NilValue {
	if t.cancel != nil {
		t.cancel()
	}
	return NilValue{}
}

type NativeMethod func(args []Value) (Value, error)
type NativeDirective func(key Value, val Value) error

// FunctionValue represents a callable function or file operation
type FunctionValue struct {
	Name string
	Fn   NativeMethod
}

func (f FunctionValue) Type() ValueType { return "function" }
func (f FunctionValue) String() string  { return fmt.Sprintf("<func %s>", f.Name) }

// UserFunction represents a function declared via #FUNC or #GLOBAL #FUNC
type UserFunction struct {
	Stmt *ast.FuncStatement
	Env  *Environment
}

func (u UserFunction) Type() ValueType { return "function" }
func (u UserFunction) String() string  { return fmt.Sprintf("<func %s>", u.Stmt.Name) }

// ReturnValue captures early returns from functions
type ReturnValue struct {
	Val Value
}

func (r ReturnValue) Type() ValueType { return r.Val.Type() }
func (r ReturnValue) String() string  { return r.Val.String() }

// LambdaValue represents an anonymous function or event closure (params) => expr
type LambdaValue struct {
	Params []*ast.Parameter
	Body   ast.Expression
	Env    *Environment
}

func (l LambdaValue) Type() ValueType { return "function" }
func (l LambdaValue) String() string  { return "<lambda>" }

