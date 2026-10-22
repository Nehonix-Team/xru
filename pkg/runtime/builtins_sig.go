package runtime

func newSigUnit() *UnitInstance {
	return &UnitInstance{
		UnitName: "Sig",
		Properties: map[string]Value{
			"interrupt": StringValue{Val: "interrupt"},
			"terminate": StringValue{Val: "terminate"},
			"hangup":    StringValue{Val: "hangup"},
			"exit":      StringValue{Val: "exit"},
			"error":     StringValue{Val: "error"},
		},
		Methods:    make(map[string]NativeMethod),
		Directives: make(map[string]NativeDirective),
	}
}
