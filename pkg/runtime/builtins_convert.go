package runtime

import (
	"fmt"
)

func goValueToXruValue(v any) Value {
	if v == nil {
		return NilValue{}
	}
	switch val := v.(type) {
	case bool:
		return BoolValue{Val: val}
	case float64:
		return NumberValue{Val: val}
	case int:
		return NumberValue{Val: float64(val)}
	case string:
		return StringValue{Val: val}
	case []any:
		var elems []Value
		for _, item := range val {
			elems = append(elems, goValueToXruValue(item))
		}
		return ListValue{Elements: elems}
	case map[string]any:
		pairs := make(map[string]Value)
		for k, item := range val {
			pairs[k] = goValueToXruValue(item)
		}
		return MapValue{Pairs: pairs}
	default:
		return StringValue{Val: fmt.Sprintf("%v", val)}
	}
}

func xruValueToGoValue(v Value) any {
	if v == nil {
		return nil
	}
	switch val := v.(type) {
	case NilValue:
		return nil
	case BoolValue:
		return val.Val
	case NumberValue:
		return val.Val
	case StringValue:
		return val.Val
	case DurationValue:
		return val.Val.String()
	case ListValue:
		var list []any
		for _, elem := range val.Elements {
			list = append(list, xruValueToGoValue(elem))
		}
		return list
	case MapValue:
		m := make(map[string]any)
		for k, elem := range val.Pairs {
			m[k] = xruValueToGoValue(elem)
		}
		return m
	default:
		return val.String()
	}
}
