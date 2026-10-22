package runtime

import (
	"fmt"
	"path/filepath"
)

func newPathUnit() *UnitInstance {
	return &UnitInstance{
		UnitName:   "Path",
		Properties: make(map[string]Value),
		Methods: map[string]NativeMethod{
			"Join": func(args []Value) (Value, error) {
				if len(args) < 1 {
					return NilValue{}, fmt.Errorf("Path.Join requires at least 1 path argument, got 0")
				}
				var parts []string
				for _, a := range args {
					parts = append(parts, a.String())
				}
				return StringValue{Val: filepath.Join(parts...)}, nil
			},
			"Dir": func(args []Value) (Value, error) {
				if len(args) < 1 {
					return NilValue{}, fmt.Errorf("Path.Dir requires 1 string argument, got 0")
				}
				return StringValue{Val: filepath.Dir(args[0].String())}, nil
			},
			"Base": func(args []Value) (Value, error) {
				if len(args) < 1 {
					return NilValue{}, fmt.Errorf("Path.Base requires 1 string argument, got 0")
				}
				return StringValue{Val: filepath.Base(args[0].String())}, nil
			},
			"Ext": func(args []Value) (Value, error) {
				if len(args) < 1 {
					return NilValue{}, fmt.Errorf("Path.Ext requires 1 string argument, got 0")
				}
				return StringValue{Val: filepath.Ext(args[0].String())}, nil
			},
		},
		Directives: make(map[string]NativeDirective),
	}
}
