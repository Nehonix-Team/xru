package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/Nehonix-Team/xru/internal/engine"
)

// executeModuleAction exécute un appel de module (utils, sys, fs, etc.).
func executeModuleAction(scope *Scope, cb, rulePath, mod, method, target, as string, line int) {
	moduleName := resolveModuleName(scope, mod)
	target = engine.Interpolate(target, scope)

	switch strings.ToLower(moduleName) {
	case "utils", "u":
		execUtilsModule(scope, cb, method, target, as, line)
	case "sys", "s":
		execSysModule(scope, cb, method, target, as, line)
	case "fs":
		execFsModule(scope, cb, rulePath, method, target, as, line)
	default:
		fmt.Printf("%s:%d: %serror:%s unknown module '%s'\n", currentFile, line, colorRed, colorReset, moduleName)
		os.Exit(1)
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

func execUtilsModule(scope *Scope, cb, method, target, as string, line int) {
	switch strings.ToUpper(method) {
	case "LOG":
		fmt.Printf("%s\n", colorify(unescape(target)))
		if as != "" {
			scope.Set(as, target, line)
		}
	case "BREAK", "EXIT":
		code := 0
		if target != "" {
			if c, err := strconv.Atoi(target); err == nil {
				code = c
			}
		}
		os.Exit(code)
	case "ARG", "ARGS":
		val := getTerminalArg(target)
		if as != "" {
			scope.Set(as, val, line)
		}
	default:
		fmt.Printf("%s:%d: %serror:%s unknown method '%s' for module 'utils'\n", currentFile, line, colorRed, colorReset, method)
		os.Exit(1)
	}
}

// --- Module sys ---

func execSysModule(scope *Scope, cb, method, target, as string, line int) {
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
			scope.Set(as, strings.TrimSpace(string(out)), line)
		} else {
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			cmd.Run()
		}
	case "ARG":
		val := getTerminalArg(target)
		if as != "" {
			scope.Set(as, val, line)
		}
	default:
		fmt.Printf("%s:%d: %serror:%s unknown method '%s' for module 'sys'\n", currentFile, line, colorRed, colorReset, method)
		os.Exit(1)
	}
}

// --- Module fs ---

func execFsModule(scope *Scope, cb, rulePath, method, target, as string, line int) {
	switch strings.ToUpper(method) {
	case "MKDIR":
		os.MkdirAll(filepath.Join(cb, target), 0755)

	case "RM":
		os.RemoveAll(filepath.Join(cb, target))

	case "TOUCH":
		os.WriteFile(filepath.Join(cb, target), []byte(""), 0644)

	case "COPY", "MOVE":
		execFsCopyMove(cb, method, target, line)

	case "READ_JSON":
		execFsReadJSON(scope, cb, rulePath, target, as, line)

	default:
		fmt.Printf("%s:%d: %serror:%s unknown method '%s' for module 'fs'\n", currentFile, line, colorRed, colorReset, method)
		os.Exit(1)
	}
}

func execFsCopyMove(cb, method, target string, line int) {
	parts := strings.SplitN(target, "->", 2)
	if len(parts) < 2 {
		fmt.Printf("%s:%d: %serror:%s %s requires 'src -> dst' syntax\n", currentFile, line, colorRed, colorReset, method)
		os.Exit(1)
	}
	src := filepath.Join(cb, strings.TrimSpace(parts[0]))
	dst := filepath.Join(cb, strings.TrimSpace(parts[1]))
	if strings.ToUpper(method) == "MOVE" {
		os.Rename(src, dst)
	} else {
		input, _ := os.ReadFile(src)
		os.WriteFile(dst, input, 0644)
	}
}

func execFsReadJSON(scope *Scope, cb, rulePath, target, as string, line int) {
	fullPath := filepath.Join(cb, target)
	data, err := os.ReadFile(fullPath)
	if err != nil {
		fullPath = filepath.Join(filepath.Dir(rulePath), target)
		data, err = os.ReadFile(fullPath)
	}
	if err != nil {
		fmt.Printf("%s:%d: %serror:%s could not read file '%s' (tried in %s and %s)\n",
			currentFile, line, colorRed, colorReset, target, cb, filepath.Dir(rulePath))
		os.Exit(1)
	}
	var parsed interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		fmt.Printf("%s:%d: %serror:%s could not parse JSON in '%s': %v\n",
			currentFile, line, colorRed, colorReset, target, err)
		os.Exit(1)
	}
	if as != "" {
		scope.Set(as, parsed, line)
	}
}
