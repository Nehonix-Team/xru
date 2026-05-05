package engine

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/Nehonix-Team/xru/internal/engine/ast"
	"github.com/Nehonix-Team/xru/internal/engine/util" 
)

// executeModuleAction exécute un appel de module (utils, sys, fs, etc.).
func executeModuleAction(scope *Scope, cb, rulePath, mod, method, target, as string, line int, r *Runner) error {
	moduleName := resolveModuleName(scope, mod)
	target = util.Interpolate(target, scope)

	switch strings.ToLower(moduleName) {
	case "utils", "u":
		return execUtilsModule(scope, cb, method, target, as, line, r)
	case "sys", "s":
		return execSysModule(scope, cb, method, target, as, line, r)
	case "fs":
		return execFsModule(scope, cb, rulePath, method, target, as, line, r)
	default:
		return fmt.Errorf("%s:%d: unknown module '%s'", r.CurrentFile, line, moduleName)
	}
}

func resolveModuleName(scope *Scope, mod string) string {
	if scope.Modules != nil {
		if val, ok := scope.Modules[mod]; ok {
			return val
		}
	}
	return mod
}

// --- Module utils ---

func execUtilsModule(scope *Scope, cb, method, target, as string, line int, r *Runner) error {
	switch strings.ToUpper(method) {
	case "LOG":
		msg := colorify(unescape(target))
		if scope.Capture != nil {
			scope.Capture.WriteString(msg + "\n")
		} else {
			fmt.Printf("%s\n", msg)
		}
		if as != "" {
			scope.Set(as, ast.Literal(target), line)
		}
		return nil
	case "BREAK", "EXIT":
		code := 0
		if target != "" {
			if c, err := strconv.Atoi(target); err == nil {
				code = c
			}
		}
		os.Exit(code) // On garde os.Exit pour EXIT/BREAK car c'est l'intention
		return nil
	case "ARG", "ARGS":
		val := getTerminalArg(target, r)
		if as != "" {
			scope.Set(as, ast.Literal(val), line)
		}
		return nil
	default:
		return fmt.Errorf("%s:%d: unknown method '%s' for module 'utils'", r.CurrentFile, line, method)
	}
}

// --- Module sys ---

func execSysModule(scope *Scope, cb, method, target, as string, line int, r *Runner) error {
	switch strings.ToUpper(method) {
	case "EXEC":
		var cmd *exec.Cmd
		if runtime.GOOS == "windows" {
			cmd = exec.Command("cmd", "/C", target)
		} else {
			cmd = exec.Command("sh", "-c", target)
		}
		cmd.Dir = cb
		if as != "" {
			out, _ := cmd.Output()
			scope.Set(as, ast.Literal(strings.TrimSpace(string(out))), line)
		} else {
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			cmd.Run()
		}
		return nil
	case "ARG":
		val := getTerminalArg(target, r)
		if as != "" {
			scope.Set(as, ast.Literal(val), line)
		}
		return nil
	case "GET":
		var val string
		switch strings.ToUpper(target) {
		case "OS":
			val = runtime.GOOS
		case "ARCH":
			val = runtime.GOARCH
		case "USER":
			val = os.Getenv("USER")
			if val == "" {
				val = os.Getenv("USERNAME")
			}
		case "CWD":
			val, _ = os.Getwd()
		default:
			return fmt.Errorf("%s:%d: unknown system property '%s'", r.CurrentFile, line, target)
		}
		if as != "" {
			scope.Set(as, ast.Literal(val), line)
		}
		return nil
	default:
		return fmt.Errorf("%s:%d: unknown method '%s' for module 'sys'", r.CurrentFile, line, method)
	}
}

// --- Module fs ---

func execFsModule(scope *Scope, cb, rulePath, method, target, as string, line int, r *Runner) error {
	switch strings.ToUpper(method) {
	case "MKDIR":
		os.MkdirAll(filepath.Join(cb, target), 0755)
		return nil
	case "RM":
		os.RemoveAll(filepath.Join(cb, target))
		return nil
	case "TOUCH":
		os.WriteFile(filepath.Join(cb, target), []byte(""), 0644)
		return nil
	case "COPY", "MOVE":
		return execFsCopyMove(cb, method, target, line, r)
	case "READ_JSON":
		return execFsReadJSON(scope, cb, rulePath, target, as, line, r)
	default:
		return fmt.Errorf("%s:%d: unknown method '%s' for module 'fs'", r.CurrentFile, line, method)
	}
}

func execFsCopyMove(cb, method, target string, line int, r *Runner) error {
	parts := strings.SplitN(target, "->", 2)
	if len(parts) < 2 {
		return fmt.Errorf("%s:%d: %s requires 'src -> dst' syntax", r.CurrentFile, line, method)
	}
	src := filepath.Join(cb, strings.TrimSpace(parts[0]))
	dst := filepath.Join(cb, strings.TrimSpace(parts[1]))
	if strings.ToUpper(method) == "MOVE" {
		os.Rename(src, dst)
	} else {
		input, err := os.ReadFile(src)
		if err != nil {
			return fmt.Errorf("%s:%d: could not read src file '%s': %v", r.CurrentFile, line, src, err)
		}
		os.WriteFile(dst, input, 0644)
	}
	return nil
}

func execFsReadJSON(scope *Scope, cb, rulePath, target, as string, line int, r *Runner) error {
	fullPath := filepath.Join(cb, target)
	data, err := os.ReadFile(fullPath)
	if err != nil {
		fullPath = filepath.Join(filepath.Dir(rulePath), target)
		data, err = os.ReadFile(fullPath)
	}
	if err != nil {
		return fmt.Errorf("%s:%d: could not read file '%s' (tried in %s and %s)",
			r.CurrentFile, line, target, cb, filepath.Dir(rulePath))
	}
	var parsed interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		return fmt.Errorf("%s:%d: could not parse JSON in '%s': %v",
			r.CurrentFile, line, target, err)
	}
	if as != "" {
		scope.Set(as, parsed, line)
	}
	return nil
}

