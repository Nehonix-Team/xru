package runtime

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

var terminalReader = bufio.NewReader(os.Stdin)

// exitCodeArg extrait un code de sortie optionnel à l'index donné.
func exitCodeArg(args []Value, idx int, def int) int {
	if len(args) > idx {
		if n, ok := args[idx].(NumberValue); ok {
			return int(n.Val)
		}
	}
	return def
}

// fatalFn : sortie brutale non rattrapable (équivalent de panic/throw non capturé).
func fatalFn(args []Value) (Value, error) {
	msg := ""
	if len(args) > 0 {
		msg = args[0].String()
	}
	fmt.Fprintln(os.Stderr, formatTerminalColors("<red>[FATAL]</> "+msg))
	code := exitCodeArg(args, 1, 1)
	TriggerExitHooks(code)
	os.Exit(code)
	return NilValue{}, nil
}

// exitFn : arrêt immédiat du processus sans message.
func exitFn(args []Value) (Value, error) {
	code := exitCodeArg(args, 0, 0)
	TriggerExitHooks(code)
	os.Exit(code)
	return NilValue{}, nil
}

func sleepFn(args []Value) (Value, error) {
	if len(args) < 1 {
		return NilValue{}, fmt.Errorf("sleep requires duration")
	}
	if dur, ok := args[0].(DurationValue); ok {
		time.Sleep(dur.Val)
		return BoolValue{Val: true}, nil
	}
	if num, ok := args[0].(NumberValue); ok {
		time.Sleep(time.Duration(num.Val) * time.Millisecond)
		return BoolValue{Val: true}, nil
	}
	if parsed, err := time.ParseDuration(args[0].String()); err == nil {
		time.Sleep(parsed)
		return BoolValue{Val: true}, nil
	}
	return BoolValue{Val: true}, nil
}

func newStdUnit() *UnitInstance {
	printlnFn := func(args []Value) (Value, error) {
		if len(args) == 0 {
			fmt.Println()
			return NilValue{}, nil
		}
		var parts []string
		for _, arg := range args {
			parts = append(parts, formatTerminalColors(arg.String()))
		}
		fmt.Println(strings.Join(parts, " "))
		return NilValue{}, nil
	}

	printFn := func(args []Value) (Value, error) {
		if len(args) == 0 {
			return NilValue{}, nil
		}
		var parts []string
		for _, arg := range args {
			parts = append(parts, formatTerminalColors(arg.String()))
		}
		fmt.Print(strings.Join(parts, " "))
		return NilValue{}, nil
	}

	infoFn := func(args []Value) (Value, error) {
		if len(args) == 0 {
			return NilValue{}, nil
		}
		fmt.Println(formatTerminalColors("<cyan>[INFO]</> " + args[0].String()))
		return NilValue{}, nil
	}

	successFn := func(args []Value) (Value, error) {
		if len(args) == 0 {
			return NilValue{}, nil
		}
		fmt.Println(formatTerminalColors("<green>[DONE]</> " + args[0].String()))
		return NilValue{}, nil
	}

	warnFn := func(args []Value) (Value, error) {
		if len(args) == 0 {
			return NilValue{}, nil
		}
		fmt.Println(formatTerminalColors("<yellow>[WARN]</> " + args[0].String()))
		return NilValue{}, nil
	}

	errorFn := func(args []Value) (Value, error) {
		if len(args) == 0 {
			return NilValue{}, nil
		}
		fmt.Println(formatTerminalColors("<red>[ERROR]</> " + args[0].String()))
		return NilValue{}, nil
	}

	typeFn := func(args []Value) (Value, error) {
		if len(args) == 0 {
			return NilValue{}, fmt.Errorf("std.type requires 1 argument")
		}
		return StringValue{Val: string(args[0].Type())}, nil
	}

	inputFn := func(args []Value) (Value, error) {
		prompt := ""
		defVal := ""
		if len(args) > 0 {
			prompt = args[0].String()
		}
		if len(args) > 1 {
			defVal = args[1].String()
		}

		if prompt != "" {
			fmt.Print(formatTerminalColors(prompt))
		}
		line, err := terminalReader.ReadString('\n')
		if err != nil && len(line) == 0 {
			return StringValue{Val: defVal}, nil
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" && defVal != "" {
			return StringValue{Val: defVal}, nil
		}
		return StringValue{Val: line}, nil
	}

	selectFn := func(args []Value) (Value, error) {
		if len(args) < 2 {
			return NilValue{}, fmt.Errorf("select requires at least (prompt, options)")
		}
		prompt := args[0].String()
		var options []string
		if arr, ok := args[1].(ListValue); ok {
			for _, item := range arr.Elements {
				options = append(options, item.String())
			}
		} else {
			return NilValue{}, fmt.Errorf("select: second argument must be a list of options")
		}

		if len(options) == 0 {
			return StringValue{Val: ""}, nil
		}

		defaultIdx := 0
		if len(args) > 2 {
			if n, ok := args[2].(NumberValue); ok {
				defaultIdx = int(n.Val)
			}
		}

		if prompt != "" {
			fmt.Println(formatTerminalColors(prompt))
		}
		for i, opt := range options {
			displayOpt := opt
			if displayOpt == "" {
				displayOpt = "(empty)"
			}
			marker := "  "
			if i == defaultIdx {
				marker = "* "
			}
			fmt.Println(formatTerminalColors(fmt.Sprintf("  <cyan>%s%d)</cyan> %s", marker, i+1, displayOpt)))
		}

		fmt.Print(formatTerminalColors(fmt.Sprintf("<dim>Select [1-%d] (default: %d): </dim>", len(options), defaultIdx+1)))
		line, err := terminalReader.ReadString('\n')
		if err != nil && len(line) == 0 {
			if defaultIdx >= 0 && defaultIdx < len(options) {
				return StringValue{Val: options[defaultIdx]}, nil
			}
			return StringValue{Val: ""}, nil
		}
		line = strings.TrimSpace(strings.TrimRight(line, "\r\n"))
		if line == "" {
			if defaultIdx >= 0 && defaultIdx < len(options) {
				return StringValue{Val: options[defaultIdx]}, nil
			}
			return StringValue{Val: ""}, nil
		}

		if num, err := strconv.Atoi(line); err == nil {
			if num >= 1 && num <= len(options) {
				return StringValue{Val: options[num-1]}, nil
			}
		}

		for _, opt := range options {
			if strings.EqualFold(opt, line) {
				return StringValue{Val: opt}, nil
			}
		}

		if defaultIdx >= 0 && defaultIdx < len(options) {
			return StringValue{Val: options[defaultIdx]}, nil
		}
		return StringValue{Val: line}, nil
	}

	confirmFn := func(args []Value) (Value, error) {
		prompt := ""
		defVal := true
		if len(args) > 0 {
			prompt = args[0].String()
		}
		if len(args) > 1 {
			if b, ok := args[1].(BoolValue); ok {
				defVal = b.Val
			}
		}

		hint := "(Y/n)"
		if !defVal {
			hint = "(y/N)"
		}

		if prompt != "" {
			fmt.Print(formatTerminalColors(fmt.Sprintf("%s <dim>%s</dim> ", prompt, hint)))
		} else {
			fmt.Print(formatTerminalColors(fmt.Sprintf("<dim>%s</dim> ", hint)))
		}

		line, err := terminalReader.ReadString('\n')
		if err != nil && len(line) == 0 {
			return BoolValue{Val: defVal}, nil
		}
		line = strings.TrimSpace(strings.ToLower(strings.TrimRight(line, "\r\n")))
		if line == "" {
			return BoolValue{Val: defVal}, nil
		}
		if line == "y" || line == "yes" || line == "o" || line == "oui" || line == "1" || line == "true" {
			return BoolValue{Val: true}, nil
		}
		if line == "n" || line == "no" || line == "non" || line == "0" || line == "false" {
			return BoolValue{Val: false}, nil
		}
		return BoolValue{Val: defVal}, nil
	}

	return &UnitInstance{
		UnitName:   "std",
		Properties: make(map[string]Value),
		Methods: map[string]NativeMethod{
			"log":     printlnFn,
			"Log":     printlnFn,
			"println": printlnFn,
			"Println": printlnFn,
			"print":   printFn,
			"Print":   printFn,
			"input":   inputFn,
			"Input":   inputFn,
			"select":  selectFn,
			"Select":  selectFn,
			"confirm": confirmFn,
			"Confirm": confirmFn,
			"info":    infoFn,
			"Info":    infoFn,
			"success": successFn,
			"Success": successFn,
			"warn":    warnFn,
			"Warn":    warnFn,
			"error":   errorFn,
			"Error":   errorFn,
			"sleep":   sleepFn,
			"Sleep":   sleepFn,
			"fatal":   fatalFn,
			"Fatal":   fatalFn,
			"exit":    exitFn,
			"Exit":    exitFn,
			"type":    typeFn,
			"Type":    typeFn,
			"typeof":  typeFn,
			"Typeof":  typeFn,
		},
		Directives: make(map[string]NativeDirective),
	}
}
