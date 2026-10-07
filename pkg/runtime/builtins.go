package runtime

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func NewBuiltinUnits() map[string]*UnitInstance {
	units := make(map[string]*UnitInstance)

	// 1. #UNIT Utils
	utilsUnit := &UnitInstance{
		UnitName:   "Utils",
		Properties: make(map[string]Value),
		Methods: map[string]NativeMethod{
			"Log": func(args []Value) (Value, error) {
				if len(args) == 0 {
					return NilValue{}, nil
				}
				msg := formatTerminalColors(args[0].String())
				fmt.Println(msg)
				return NilValue{}, nil
			},
		},
		Directives: make(map[string]NativeDirective),
	}
	units["Utils"] = utilsUnit

	// 2. #UNIT Fs
	fsUnit := &UnitInstance{
		UnitName:   "Fs",
		Properties: make(map[string]Value),
		Methods: map[string]NativeMethod{
			"EnsureDir": func(args []Value) (Value, error) {
				if len(args) < 1 {
					return NilValue{}, fmt.Errorf("Fs.EnsureDir requires directory path")
				}
				err := os.MkdirAll(args[0].String(), 0755)
				return BoolValue{Val: err == nil}, err
			},
			"Remove": func(args []Value) (Value, error) {
				if len(args) < 1 {
					return NilValue{}, fmt.Errorf("Fs.Remove requires target path")
				}
				err := os.RemoveAll(args[0].String())
				return BoolValue{Val: err == nil}, err
			},
			"Exists": func(args []Value) (Value, error) {
				if len(args) < 1 {
					return BoolValue{Val: false}, nil
				}
				_, err := os.Stat(args[0].String())
				return BoolValue{Val: !os.IsNotExist(err)}, nil
			},
		},
		Directives: make(map[string]NativeDirective),
	}
	units["Fs"] = fsUnit

	// 3. #UNIT Net
	netUnit := &UnitInstance{
		UnitName:   "Net",
		Properties: make(map[string]Value),
		Methods: map[string]NativeMethod{
			"Fetch": func(args []Value) (Value, error) {
				if len(args) < 2 {
					return NilValue{}, fmt.Errorf("Net.Fetch requires url and dest path")
				}
				url := args[0].String()
				dest := args[1].String()

				resp, err := http.Get(url)
				if err != nil {
					return BoolValue{Val: false}, err
				}
				defer resp.Body.Close()

				if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
					return BoolValue{Val: false}, err
				}

				out, err := os.Create(dest)
				if err != nil {
					return BoolValue{Val: false}, err
				}
				defer out.Close()

				_, err = io.Copy(out, resp.Body)
				return BoolValue{Val: err == nil}, err
			},
			"Clone": func(args []Value) (Value, error) {
				if len(args) < 2 {
					return NilValue{}, fmt.Errorf("Net.Clone requires repo url and dest")
				}
				repo := args[0].String()
				dest := args[1].String()

				cmd := exec.Command("git", "clone", "--depth", "1", repo, dest)
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				err := cmd.Run()
				return BoolValue{Val: err == nil}, err
			},
		},
		Directives: map[string]NativeDirective{
			"#SET": func(key Value, val Value) error {
				return nil
			},
			"#HEADER": func(key Value, val Value) error {
				return nil
			},
			"#ON": func(key Value, val Value) error {
				return nil
			},
		},
	}
	units["Net"] = netUnit

	// 4. #UNIT Sys
	sysUnit := &UnitInstance{
		UnitName:   "Sys",
		Properties: make(map[string]Value),
		Methods: map[string]NativeMethod{
			"Env": func(args []Value) (Value, error) {
				if len(args) < 1 {
					return StringValue{Val: ""}, nil
				}
				return StringValue{Val: os.Getenv(args[0].String())}, nil
			},
			"Time": func(args []Value) (Value, error) {
				return StringValue{Val: time.Now().Format(time.RFC3339)}, nil
			},
		},
		Directives: make(map[string]NativeDirective),
	}
	units["Sys"] = sysUnit

	return units
}

func formatTerminalColors(s string) string {
	s = strings.ReplaceAll(s, "<green>", "\033[32m")
	s = strings.ReplaceAll(s, "<yellow>", "\033[33m")
	s = strings.ReplaceAll(s, "<cyan>", "\033[36m")
	s = strings.ReplaceAll(s, "<blue>", "\033[34m")
	s = strings.ReplaceAll(s, "<red>", "\033[31m")
	s = strings.ReplaceAll(s, "<bold>", "\033[1m")
	s = strings.ReplaceAll(s, "</>", "\033[0m")
	return s
}
