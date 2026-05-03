package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/Nehonix-Team/xru/internal/compiler"
	"github.com/Nehonix-Team/xru/internal/engine"
	"github.com/Nehonix-Team/xru/internal/utils"
)

const (
	colorReset   = "\033[0m"
	colorRed     = "\033[31m"
	colorGreen   = "\033[32m"
	colorYellow  = "\033[33m"
	colorBlue    = "\033[34m"
	colorMagenta = "\033[35m"
	colorCyan    = "\033[36m"
	colorGray    = "\033[90m"
	colorWhite   = "\033[97m"
)

var verbose bool
var currentFile string

type Scope struct {
	Vars     map[string]string
	DefLines map[string]int
	Used     map[string]bool
	Modules  map[string]string // Alias -> ModuleName
	Parent   *Scope
}

func (s *Scope) Get(name string) (string, bool) {
	if val, ok := s.Vars[name]; ok {
		if s.Used == nil {
			s.Used = make(map[string]bool)
		}
		s.Used[name] = true
		return val, true
	}
	if s.Parent != nil {
		return s.Parent.Get(name)
	}
	return "", false
}

func (s *Scope) Set(name, val string, line int) {
	if s.Vars == nil {
		s.Vars = make(map[string]string)
		s.DefLines = make(map[string]int)
	}
	if _, ok := s.Vars[name]; ok {
		fmt.Printf("%s:%d: %serror:%s variable '%s' already defined in this scope\n", currentFile, line, colorRed, colorReset, name)
		os.Exit(1)
	}
	s.Vars[name] = val
	s.DefLines[name] = line
}

func (s *Scope) RegisterModule(alias, name string, line int) {
	if s.Modules == nil {
		s.Modules = make(map[string]string)
	}
	if existing, ok := s.Modules[alias]; ok {
		if existing == name {
			return
		}
		fmt.Printf("%s:%d: %serror:%s module alias '%s' already defined in this scope (points to '%s')\n", currentFile, line, colorRed, colorReset, alias, existing)
		os.Exit(1)
	}
	s.Modules[alias] = name
}

func (s *Scope) CheckUnused() {
	if s.Vars == nil {
		return
	}
	for name := range s.Vars {
		if s.Used == nil || !s.Used[name] {
			line := s.DefLines[name]
			fmt.Printf("%s:%d: %swarning:%s variable '%s' is defined but never used\n", currentFile, line, colorYellow, colorReset, name)
		}
	}
}

func unescape(s string) string {
	s = strings.ReplaceAll(s, "\\n", "\n")
	s = strings.ReplaceAll(s, "\\t", "\t")
	return s
}

func colorify(s string) string {
	s = strings.ReplaceAll(s, "<red>", colorRed)
	s = strings.ReplaceAll(s, "<green>", colorGreen)
	s = strings.ReplaceAll(s, "<yellow>", colorYellow)
	s = strings.ReplaceAll(s, "<blue>", colorBlue)
	s = strings.ReplaceAll(s, "<magenta>", colorMagenta)
	s = strings.ReplaceAll(s, "<cyan>", colorCyan)
	s = strings.ReplaceAll(s, "<gray>", colorGray)
	s = strings.ReplaceAll(s, "<white>", colorWhite)
	s = strings.ReplaceAll(s, "</>", colorReset)
	return s
}

func checkSyntaxError(val string, line int) {
	if val == "[SYNTAX_ERROR: UNCLOSED_QUOTE]" {
		fmt.Printf("%s:%d: %ssyntax error:%s missing terminating '\"' or \"'\" character\n", currentFile, line, colorRed, colorReset)
		os.Exit(1)
	}
	if val == "[SYNTAX_ERROR: UNCLOSED_BRACE]" {
		fmt.Printf("%s:%d: %ssyntax error:%s missing terminating '}' for variable interpolation\n", currentFile, line, colorRed, colorReset)
		os.Exit(1)
	}
}

var rootScope = &Scope{
	Vars:     make(map[string]string),
	DefLines: make(map[string]int),
	Used:     make(map[string]bool),
	Modules:  make(map[string]string),
}

func main() {
	v := flag.Bool("v", false, "Enable verbose output")
	flag.Parse()
	verbose = *v
	args := flag.Args()

	if len(args) < 1 {
		os.Exit(1)
	}

	command := args[0]
	switch command {
	case "version":
		fmt.Printf("XRU %s\n", utils.BinVersion)
	case "upgrade":
		handleUpgrade()
	case "build":
		if len(args) < 2 { os.Exit(1) }
		runBuild(args[1])
	default:
		target := "."
		if len(args) > 1 {
			target = args[1]
		}
		runPatch(args[0], target)
	}
}

func runPatch(rulePath, targetDir string) {
	currentFile = rulePath
	absTarget, _ := filepath.Abs(targetDir)
	rf, err := engine.ParseFile(rulePath)
	if err != nil {
		fmt.Printf("%sError: %v%s\n", colorRed, err, colorReset)
		os.Exit(1)
	}
	executeRules(rf.Rules, absTarget, absTarget, rulePath, rootScope)
	rootScope.CheckUnused()
}

func executeRules(rules []engine.Rule, initialTarget, currentBase, rulePath string, scope *Scope) {
	cb := currentBase
	skipElse := false

	for _, rule := range rules {
		target := engine.Interpolate(rule.Target, scope)
		checkSyntaxError(target, rule.Line)

		switch rule.Type {
		case engine.RuleTypeVar:
			val := engine.Interpolate(rule.Content, scope)
			checkSyntaxError(val, rule.Line)
			scope.Set(rule.Target, val, rule.Line)
			skipElse = false

		case engine.RuleTypeSelect:
			checkSyntaxError(target, rule.Line)
			if filepath.IsAbs(target) {
				cb = target
			} else {
				cb = filepath.Join(initialTarget, target)
			}
			info, err := os.Stat(cb)
			if os.IsNotExist(err) {
				fmt.Printf("%s:%d: %serror:%s directory '%s' does not exist\n", currentFile, rule.Line, colorRed, colorReset, cb)
				os.Exit(1)
			}
			if !info.IsDir() {
				fmt.Printf("%s:%d: %serror:%s path '%s' is a file, but SELECT requires a directory\n", currentFile, rule.Line, colorRed, colorReset, cb)
				os.Exit(1)
			}
			if rule.As != "" {
				scope.Set(rule.As, cb, rule.Line)
			}
			skipElse = false

		case engine.RuleTypeIf:
			if evalCondition(rule.Target, scope, cb) {
				executeRules(rule.SubRules, initialTarget, cb, rulePath, scope)
				skipElse = true
			} else {
				skipElse = false
			}

		case engine.RuleTypeElseIf:
			if !skipElse && evalCondition(rule.Target, scope, cb) {
				executeRules(rule.SubRules, initialTarget, cb, rulePath, scope)
				skipElse = true
			}

		case engine.RuleTypeElse:
			if !skipElse {
				executeRules(rule.SubRules, initialTarget, cb, rulePath, scope)
			}
			skipElse = false

		case engine.RuleTypeUse:
			name := engine.Interpolate(rule.Target, scope)
			alias := rule.As
			if alias == "" {
				alias = name
			}
			scope.RegisterModule(alias, name, rule.Line)
			skipElse = false

		case engine.RuleTypeModule:
			parts := strings.SplitN(rule.Target, ".", 2)
			content := engine.Interpolate(rule.Content, scope)
			checkSyntaxError(content, rule.Line)
			executeModuleAction(scope, cb, parts[0], parts[1], content, rule.As, rule.Line)
			skipElse = false

		case engine.RuleTypeInclude:
			includePath := target
			if !filepath.IsAbs(includePath) {
				includePath = filepath.Join(filepath.Dir(rulePath), includePath)
			}
			irf, err := engine.ParseFile(includePath)
			if err == nil {
				executeRules(irf.Rules, initialTarget, cb, includePath, scope)
			}
			skipElse = false

		case engine.RuleTypeBegin, engine.RuleTypeCreate, engine.RuleTypeGlobal:
			applyRule(initialTarget, cb, rule, scope)
			skipElse = false
		}
	}
}

func evalCondition(cond string, scope *Scope, cb string) bool {
	cond = engine.Interpolate(cond, scope)
	cond = strings.Trim(cond, "\"' ")

	negate := false
	if strings.HasPrefix(cond, "!") {
		negate = true
		cond = strings.TrimSpace(cond[1:])
	}

	result := false
	if strings.HasPrefix(cond, "exists(") && strings.HasSuffix(cond, ")") {
		path := strings.Trim(cond[7:len(cond)-1], "\"' ")
		absPath := filepath.Join(cb, path)
		_, err := os.Stat(absPath)
		result = err == nil
	} else if strings.Contains(cond, "==") {
		parts := strings.SplitN(cond, "==", 2)
		result = strings.Trim(parts[0], "\"' ") == strings.Trim(parts[1], "\"' ")
	} else if strings.Contains(cond, "!=") {
		parts := strings.SplitN(cond, "!=", 2)
		result = strings.Trim(parts[0], "\"' ") != strings.Trim(parts[1], "\"' ")
	} else {
		result = cond == "true"
	}

	if negate {
		return !result
	}
	return result
}

func applyRule(initialTarget, currentBase string, rule engine.Rule, parentScope *Scope) {
	scope := &Scope{
		Vars:     make(map[string]string),
		DefLines: make(map[string]int),
		Used:     make(map[string]bool),
		Modules:  parentScope.Modules,
		Parent:   parentScope,
	}

	target := engine.Interpolate(rule.Target, scope)
	if rule.As != "" {
		scope.Set(rule.As, target, rule.Line)
	}

	switch rule.Type {
	case engine.RuleTypeCreate:
		fullPath := filepath.Join(currentBase, target)
		os.MkdirAll(filepath.Dir(fullPath), 0755)
		content := engine.Interpolate(rule.Content, scope)
		os.WriteFile(fullPath, []byte(content), 0644)
		executeRules(rule.SubRules, initialTarget, currentBase, currentFile, scope)

	case engine.RuleTypeBegin:
		fullPath := filepath.Join(currentBase, target)
		data, err := os.ReadFile(fullPath)
		if err != nil {
			return
		}
		content := string(data)
		for _, action := range rule.Actions {
			content = applyAction(content, action, filepath.Ext(fullPath), scope, currentBase)
		}
		os.WriteFile(fullPath, []byte(content), 0644)
		executeRules(rule.SubRules, initialTarget, currentBase, currentFile, scope)

	case engine.RuleTypeGlobal:
		filepath.Walk(currentBase, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}
			data, _ := os.ReadFile(path)
			content := string(data)
			original := content
			for _, action := range rule.Actions {
				content = applyAction(content, action, filepath.Ext(path), scope, currentBase)
			}
			if content != original {
				os.WriteFile(path, []byte(content), info.Mode())
			}
			return nil
		})
		executeRules(rule.SubRules, initialTarget, currentBase, currentFile, scope)
	}
}

func applyAction(content string, action engine.Action, fileExt string, scope *Scope, cb string) string {
	switch a := action.(type) {
	case engine.VarAction:
		val := engine.Interpolate(a.Value, scope)
		scope.Set(a.Name, val, a.Line)
		return content
	case engine.ModuleAction:
		executeModuleAction(scope, cb, a.Module, a.Method, a.Target, a.As, a.Line)
		return content
	case engine.InjectAction:
		if a.Lang != "" {
			if "."+strings.ToLower(a.Lang) != fileExt {
				return content
			}
		}
		code := engine.Interpolate(a.Code, scope)
		return engine.InjectCode(content, a.Key, code)
	case engine.PatchAction:
		path := engine.Interpolate(a.Path, scope)
		val := engine.InterpolateValue(a.Value, scope)
		return engine.ApplyPatch(content, a.Op, path, val)
	}
	return content
}

func executeModuleAction(scope *Scope, cb, mod, method, target, as string, line int) {
	moduleName := mod
	if scope.Modules != nil {
		if val, ok := scope.Modules[mod]; ok {
			moduleName = val
		}
	}
	target = engine.Interpolate(target, scope)
	switch strings.ToLower(moduleName) {
	case "utils", "u":
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
		default:
			fmt.Printf("%s:%d: %serror:%s unknown method '%s' for module '%s'\n", currentFile, line, colorRed, colorReset, method, moduleName)
			os.Exit(1)
		}
	case "sys", "s":
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
		default:
			fmt.Printf("%s:%d: %serror:%s unknown method '%s' for module '%s'\n", currentFile, line, colorRed, colorReset, method, moduleName)
			os.Exit(1)
		}
	case "fs":
		switch strings.ToUpper(method) {
		case "MKDIR":
			os.MkdirAll(filepath.Join(cb, target), 0755)
		case "RM":
			os.RemoveAll(filepath.Join(cb, target))
		case "TOUCH":
			os.WriteFile(filepath.Join(cb, target), []byte(""), 0644)
		case "COPY", "MOVE":
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
				// Simple copy (file only for now)
				input, _ := os.ReadFile(src)
				os.WriteFile(dst, input, 0644)
			}
		default:
			fmt.Printf("%s:%d: %serror:%s unknown method '%s' for module '%s'\n", currentFile, line, colorRed, colorReset, method, moduleName)
			os.Exit(1)
		}
	default:
		fmt.Printf("%s:%d: %serror:%s unknown module '%s'\n", currentFile, line, colorRed, colorReset, moduleName)
		os.Exit(1)
	}
}

func handleUpgrade() {
	binaryName := "xru"
	osName := runtime.GOOS
	archName := runtime.GOARCH
	ext := ""
	if osName == "windows" { ext = ".exe" }
	url := fmt.Sprintf("https://github.com/Nehonix-Team/xru/releases/latest/download/%s-%s-%s%s", binaryName, osName, archName, ext)
	executablePath, _ := os.Executable()
	tmpPath := executablePath + ".tmp"
	out, _ := os.Create(tmpPath)
	defer out.Close()
	resp, _ := http.Get(url)
	defer resp.Body.Close()
	io.Copy(out, resp.Body)
	os.Chmod(tmpPath, 0755)
	os.Rename(tmpPath, executablePath)
}

func runBuild(rulePath string) {
	rf, _ := engine.ParseFile(rulePath)
	shScript, _ := compiler.CompileToSH(rf)
	psScript, _ := compiler.CompileToPS1(rf)
	base := strings.TrimSuffix(rulePath, filepath.Ext(rulePath))
	os.WriteFile(base+".sh", []byte(shScript), 0755)
	os.WriteFile(base+".ps1", []byte(psScript), 0644)
}
