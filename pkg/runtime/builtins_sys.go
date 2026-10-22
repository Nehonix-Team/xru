package runtime

import (
	"os"
	"path/filepath"
	goRuntime "runtime"
	"strings"
	"time"
)

func cwdFn(args []Value) (Value, error) {
	wd, err := os.Getwd()
	if err != nil {
		return StringValue{Val: ""}, err
	}
	if abs, err := filepath.Abs(wd); err == nil {
		return StringValue{Val: abs}, nil
	}
	return StringValue{Val: wd}, nil
}

func newSysUnit() *UnitInstance {
	return &UnitInstance{
		UnitName:   "Sys",
		Properties: make(map[string]Value),
		Methods: map[string]NativeMethod{
			"Env": func(args []Value) (Value, error) {
				if len(args) < 1 {
					return StringValue{Val: ""}, nil
				}
				return StringValue{Val: os.Getenv(args[0].String())}, nil
			},
			"env": func(args []Value) (Value, error) {
				if len(args) < 1 {
					return StringValue{Val: ""}, nil
				}
				return StringValue{Val: os.Getenv(args[0].String())}, nil
			},
			"getEnv": func(args []Value) (Value, error) {
				if len(args) == 0 {
					return StringValue{Val: ""}, nil
				}
				val := os.Getenv(args[0].String())
				if val == "" && len(args) > 1 {
					val = args[1].String()
				}
				return StringValue{Val: val}, nil
			},
			"GetEnv": func(args []Value) (Value, error) {
				if len(args) == 0 {
					return StringValue{Val: ""}, nil
				}
				val := os.Getenv(args[0].String())
				if val == "" && len(args) > 1 {
					val = args[1].String()
				}
				return StringValue{Val: val}, nil
			},
			"Time": func(args []Value) (Value, error) {
				return StringValue{Val: time.Now().Format(time.RFC3339)}, nil
			},
			"time": func(args []Value) (Value, error) {
				return StringValue{Val: time.Now().Format(time.RFC3339)}, nil
			},
			"Os": func(args []Value) (Value, error) {
				return StringValue{Val: goRuntime.GOOS}, nil
			},
			"os": func(args []Value) (Value, error) {
				return StringValue{Val: goRuntime.GOOS}, nil
			},
			"Arch": func(args []Value) (Value, error) {
				return StringValue{Val: goRuntime.GOARCH}, nil
			},
			"arch": func(args []Value) (Value, error) {
				return StringValue{Val: goRuntime.GOARCH}, nil
			},
			"Sleep":   sleepFn,
			"sleep":   sleepFn,
			"Exit":    exitFn,
			"exit":    exitFn,
			"require": sysRequireFn,
			"Require": sysRequireFn,
			"version": sysVersionFn,
			"Version": sysVersionFn,
			"cwd":     cwdFn,
			"Cwd":     cwdFn,
			"homeDir": func(args []Value) (Value, error) {
				h, err := os.UserHomeDir()
				if err != nil {
					return StringValue{Val: ""}, err
				}
				return StringValue{Val: h}, nil
			},
			"HomeDir": func(args []Value) (Value, error) {
				h, err := os.UserHomeDir()
				if err != nil {
					return StringValue{Val: ""}, err
				}
				return StringValue{Val: h}, nil
			},
			"args": func(args []Value) (Value, error) {
				raw := GetGlobalCliArgs()
				var elems []Value
				for _, a := range raw {
					elems = append(elems, StringValue{Val: a})
				}
				return ListValue{Elements: elems}, nil
			},
			"Args": func(args []Value) (Value, error) {
				raw := GetGlobalCliArgs()
				var elems []Value
				for _, a := range raw {
					elems = append(elems, StringValue{Val: a})
				}
				return ListValue{Elements: elems}, nil
			},
			"hasArg": func(args []Value) (Value, error) {
				raw := GetGlobalCliArgs()
				if len(args) == 0 {
					return BoolValue{Val: len(raw) > 0}, nil
				}
				for _, argVal := range args {
					target := strings.TrimSpace(argVal.String())
					if target == "" {
						continue
					}
					targetClean := strings.TrimLeft(target, "-")
					targetLower := strings.ToLower(targetClean)

					for _, cliArg := range raw {
						cliArgTrimmed := strings.TrimSpace(cliArg)
						if cliArgTrimmed == target {
							return BoolValue{Val: true}, nil
						}
						if strings.HasPrefix(cliArgTrimmed, target+"=") {
							return BoolValue{Val: true}, nil
						}
						cliClean := strings.TrimLeft(cliArgTrimmed, "-")
						cliKey := cliClean
						if idx := strings.Index(cliClean, "="); idx != -1 {
							cliKey = cliClean[:idx]
						}
						if strings.EqualFold(cliKey, targetLower) {
							return BoolValue{Val: true}, nil
						}
					}
				}
				return BoolValue{Val: false}, nil
			},
			"HasArg": func(args []Value) (Value, error) {
				raw := GetGlobalCliArgs()
				if len(args) == 0 {
					return BoolValue{Val: len(raw) > 0}, nil
				}
				for _, argVal := range args {
					target := strings.TrimSpace(argVal.String())
					if target == "" {
						continue
					}
					targetClean := strings.TrimLeft(target, "-")
					targetLower := strings.ToLower(targetClean)

					for _, cliArg := range raw {
						cliArgTrimmed := strings.TrimSpace(cliArg)
						if cliArgTrimmed == target {
							return BoolValue{Val: true}, nil
						}
						if strings.HasPrefix(cliArgTrimmed, target+"=") {
							return BoolValue{Val: true}, nil
						}
						cliClean := strings.TrimLeft(cliArgTrimmed, "-")
						cliKey := cliClean
						if idx := strings.Index(cliClean, "="); idx != -1 {
							cliKey = cliClean[:idx]
						}
						if strings.EqualFold(cliKey, targetLower) {
							return BoolValue{Val: true}, nil
						}
					}
				}
				return BoolValue{Val: false}, nil
			},
		},
		Directives: make(map[string]NativeDirective),
	}
}
