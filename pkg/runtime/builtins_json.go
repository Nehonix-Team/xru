package runtime

import (
	"encoding/json"
	"fmt"
)

func newJsonUnit() *UnitInstance {
	return &UnitInstance{
		UnitName:   "Json",
		Properties: make(map[string]Value),
		Methods: map[string]NativeMethod{
			"Parse": func(args []Value) (Value, error) {
				if len(args) < 1 {
					return NilValue{}, fmt.Errorf("Json.Parse requires raw string")
				}
				raw := args[0].String()
				var parsed any
				if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
					return NilValue{}, fmt.Errorf("Json.Parse failed: %w", err)
				}
				return goValueToXruValue(parsed), nil
			},
			"Stringify": func(args []Value) (Value, error) {
				if len(args) < 1 {
					return NilValue{}, fmt.Errorf("Json.Stringify requires 1 argument")
				}
				goVal := xruValueToGoValue(args[0])
				b, err := json.Marshal(goVal)
				if err != nil {
					return NilValue{}, fmt.Errorf("Json.Stringify failed: %w", err)
				}
				return StringValue{Val: string(b)}, nil
			},
		},
		Directives: make(map[string]NativeDirective),
	}
}
