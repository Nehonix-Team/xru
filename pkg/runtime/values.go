package runtime

import (
	"fmt"
	"strings"
	"time"
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

// UnitInstance représente une instance vivante d'une #UNIT
type UnitInstance struct {
	UnitName   string
	Properties map[string]Value
	Methods    map[string]NativeMethod
	Directives map[string]NativeDirective
}

func (u *UnitInstance) Type() ValueType { return ValUnit }
func (u *UnitInstance) String() string  { return fmt.Sprintf("@%s()", u.UnitName) }

type NativeMethod func(args []Value) (Value, error)
type NativeDirective func(key Value, val Value) error

// FunctionValue represents a callable function or file operation
type FunctionValue struct {
	Name string
	Fn   NativeMethod
}

func (f FunctionValue) Type() ValueType { return "function" }
func (f FunctionValue) String() string  { return fmt.Sprintf("<func %s>", f.Name) }
