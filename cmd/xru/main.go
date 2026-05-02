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

	"github.com/Nehonix-Team/xru/internal/engine"
	"github.com/Nehonix-Team/xru/internal/utils"
)

const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorMagenta = "\033[35m"
	colorCyan   = "\033[36m"
	colorGray    = "\033[90m"
	colorWhite   = "\033[97m"
)

var verbose bool

type Scope struct {
	Vars    map[string]string
	Modules map[string]string // Alias -> ModuleName
	Parent  *Scope
}

func (s *Scope) Flatten() map[string]string {
	res := make(map[string]string)
	if s.Parent != nil {
		for k, v := range s.Parent.Flatten() {
			res[k] = v
		}
	}
	for k, v := range s.Vars {
		res[k] = v
	}
	return res
}

func (s *Scope) Set(name, val string) {
	if s.Vars == nil {
		s.Vars = make(map[string]string)
	}
	s.Vars[name] = val
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
	// Support specific closing tags for convenience
	s = strings.ReplaceAll(s, "</red>", colorReset)
	s = strings.ReplaceAll(s, "</green>", colorReset)
	s = strings.ReplaceAll(s, "</yellow>", colorReset)
	s = strings.ReplaceAll(s, "</blue>", colorReset)
	s = strings.ReplaceAll(s, "</magenta>", colorReset)
	s = strings.ReplaceAll(s, "</cyan>", colorReset)
	s = strings.ReplaceAll(s, "</gray>", colorReset)
	s = strings.ReplaceAll(s, "</white>", colorReset)
	return s
}

var rootScope = &Scope{
	Vars:    make(map[string]string),
	Modules: map[string]string{"U": "Utils", "S": "Sys"}, // Default aliases
}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "XRU: XyPriss Rule Unit — Structured Text Patcher (%s)\n\n", utils.BinVersion)
		fmt.Fprintf(os.Stderr, "Usage:\n")
		fmt.Fprintf(os.Stderr, "  xru [options] <rule_file.xru> [target_dir]\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nCommands:\n")
		fmt.Fprintf(os.Stderr, "  upgrade      Update to latest version\n")
		fmt.Fprintf(os.Stderr, "  version      Show version info\n")
	}

	v := flag.Bool("v", false, "Enable verbose output")
	flag.Parse()
	verbose = *v
	args := flag.Args()

	if len(args) < 1 {
		flag.Usage()
		os.Exit(1)
	}

	command := args[0]

	switch command {
	case "version":
		fmt.Printf("XRU %s (%s/%s)\n", utils.BinVersion, runtime.GOOS, runtime.GOARCH)
	case "upgrade":
		handleUpgrade()
	case "run":
		if len(args) < 2 {
			fmt.Printf("%sError: Missing rule file. Usage: xru run <rule_file> [target]%s\n", colorRed, colorReset)
			os.Exit(1)
		}
		target := "."
		if len(args) > 2 {
			target = args[2]
		}
		runPatch(args[1], target)
	default:
		// Fallback to shorthand: xru <file> [target]
		target := "."
		if len(args) > 1 {
			target = args[1]
		}
		runPatch(args[0], target)
	}
}

func runPatch(rulePath, targetDir string) {
	absTarget, err := filepath.Abs(targetDir)
	if err != nil {
		fmt.Printf("%sError resolving target directory: %v%s\n", colorRed, err, colorReset)
		os.Exit(1)
	}

	rf, err := engine.ParseFile(rulePath)
	if err != nil {
		fmt.Printf("%sError parsing rule file: %v%s\n", colorRed, err, colorReset)
		os.Exit(1)
	}

	if verbose {
		fmt.Printf("%s[RUN]%s Applying XRU rules to %s...\n", colorBlue, colorReset, absTarget)
	}

	executeRules(rf.Rules, absTarget, absTarget, rulePath, rootScope)

	if verbose {
		fmt.Printf("%s[SUCCESS]%s Transformation complete.\n", colorGreen, colorReset)
	}
}

func executeRules(rules []engine.Rule, initialTarget, currentBase, rulePath string, scope *Scope) {
	cb := currentBase
	for _, rule := range rules {
		target := engine.Interpolate(rule.Target, scope.Flatten())

		if rule.Type == engine.RuleTypeVar {
			val := engine.Interpolate(rule.Content, scope.Flatten())
			scope.Set(rule.Target, val)
			continue
		}

		if rule.Type == engine.RuleTypeSelect {
			if filepath.IsAbs(target) {
				cb = target
			} else {
				cb = filepath.Join(initialTarget, target)
			}
			if verbose {
				fmt.Printf("%s[SELECT]%s Switching sandbox to: %s\n", colorCyan, colorReset, cb)
			}
			if rule.As != "" {
				scope.Set(rule.As, cb)
			}
			continue
		}

		if rule.Type == engine.RuleTypeUse {
			name := engine.Interpolate(rule.Target, scope.Flatten())
			alias := rule.As
			if alias == "" {
				alias = name
			}
			if scope.Modules == nil {
				scope.Modules = make(map[string]string)
			}
			scope.Modules[alias] = name
			continue
		}

		if rule.Type == engine.RuleTypeModule {
			parts := strings.SplitN(rule.Target, ".", 2)
			executeModuleAction(scope, cb, parts[0], parts[1], rule.Content, rule.As, rulePath, initialTarget)
			continue
		}

		if rule.Type == engine.RuleTypeInclude {
			includePath := target
			if !filepath.IsAbs(includePath) {
				includePath = filepath.Join(filepath.Dir(rulePath), includePath)
			}
			if verbose {
				fmt.Printf("%s[INCLUDE]%s Loading rules from: %s\n", colorBlue, colorReset, includePath)
			}
			irf, err := engine.ParseFile(includePath)
			if err != nil {
				fmt.Printf("%s[INCLUDE ERROR]%s %v\n", colorRed, colorReset, err)
				continue
			}
			executeRules(irf.Rules, initialTarget, cb, includePath, scope)
			continue
		}

		if err := applyRule(cb, rule, scope); err != nil {
			if verbose {
				fmt.Printf("%s[ERROR]%s %v\n", colorRed, colorReset, err)
			}
		}
	}
}

func handleUpgrade() {
	fmt.Printf("%s[UPGRADE]%s Checking for updates...\n", colorCyan, colorReset)
	
	binaryName := "xru"
	osName := runtime.GOOS
	archName := runtime.GOARCH
	
	ext := ""
	if osName == "windows" {
		ext = ".exe"
	}
	
	url := fmt.Sprintf("https://github.com/Nehonix-Team/xru/releases/latest/download/%s-%s-%s%s", binaryName, osName, archName, ext)
	
	fmt.Printf("%s[DOWNLOAD]%s Fetching latest binary from: %s\n", colorBlue, colorReset, url)
	
	executablePath, err := os.Executable()
	if err != nil {
		fmt.Printf("%sError: Could not find executable path: %v%s\n", colorRed, err, colorReset)
		return
	}

	tmpPath := executablePath + ".tmp"
	out, err := os.Create(tmpPath)
	if err != nil {
		fmt.Printf("%sError: Could not create temp file: %v%s\n", colorRed, err, colorReset)
		return
	}
	defer out.Close()

	resp, err := http.Get(url)
	if err != nil {
		fmt.Printf("%sError: Download failed: %v%s\n", colorRed, err, colorReset)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("%sError: Server returned %s. Ensure a release exists for your platform.%s\n", colorRed, resp.Status, colorReset)
		return
	}

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		fmt.Printf("%sError: Failed to save binary: %v%s\n", colorRed, err, colorReset)
		return
	}

	os.Chmod(tmpPath, 0755)

	if runtime.GOOS == "windows" {
		oldPath := executablePath + ".old"
		os.Rename(executablePath, oldPath)
		err = os.Rename(tmpPath, executablePath)
	} else {
		err = os.Rename(tmpPath, executablePath)
	}

	if err != nil {
		fmt.Printf("%sError: Could not replace binary: %v. You might need sudo.%s\n", colorRed, err, colorReset)
		return
	}

	fmt.Printf("%s[SUCCESS]%s Upgrade complete! Run 'xru version' to verify.\n", colorGreen, colorReset)
}

func applyRule(targetDir string, rule engine.Rule, parentScope *Scope) error {
	// Create local scope for this rule
	scope := &Scope{Vars: make(map[string]string), Modules: parentScope.Modules, Parent: parentScope}

	target := engine.Interpolate(rule.Target, scope.Flatten())
	if rule.As != "" {
		scope.Set(rule.As, target)
	}
	vars := scope.Flatten()

	switch rule.Type {
	case engine.RuleTypeCreate:
		fullPath := filepath.Join(targetDir, target)
		if verbose {
			fmt.Printf("  %s+%s Creating %s\n", colorGreen, colorReset, target)
		}
		os.MkdirAll(filepath.Dir(fullPath), 0755)
		content := engine.Interpolate(rule.Content, vars)
		return os.WriteFile(fullPath, []byte(content), 0644)

	case engine.RuleTypeBegin:
		fullPath := filepath.Join(targetDir, target)
		if verbose {
			fmt.Printf("  %s→%s Patching %s\n", colorBlue, colorReset, target)
		}
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			return fmt.Errorf("file %s does not exist", target)
		}
		data, err := os.ReadFile(fullPath)
		if err != nil {
			return err
		}
		content := string(data)
		for _, action := range rule.Actions {
			content = applyAction(content, action, filepath.Ext(fullPath), scope, targetDir)
		}
		return os.WriteFile(fullPath, []byte(content), 0644)

	case engine.RuleTypeGlobal:
		return filepath.Walk(targetDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				name := info.Name()
				if name == ".git" || name == "node_modules" || name == "dist" || name == "vendor" || name == ".gemini" {
					return filepath.SkipDir
				}
				return nil
			}
			ext := filepath.Ext(path)
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			content := string(data)
			original := content
			for _, action := range rule.Actions {
				content = applyAction(content, action, ext, scope, targetDir)
			}
			if content != original {
				if verbose {
					fmt.Printf("  %s*%s Global match: %s\n", colorYellow, colorReset, path)
				}
				return os.WriteFile(path, []byte(content), info.Mode())
			}
			return nil
		})
	}
	return nil
}

func applyAction(content string, action engine.Action, fileExt string, scope *Scope, cb string) string {
	vars := scope.Flatten()
	switch a := action.(type) {
	case engine.VarAction:
		val := engine.Interpolate(a.Value, vars)
		scope.Set(a.Name, val)
		return content
	case engine.ModuleAction:
		executeModuleAction(scope, cb, a.Module, a.Method, a.Target, a.As, "", "")
		return content
	case engine.InjectAction:
		if a.Lang != "" {
			targetExt := "." + strings.ToLower(a.Lang)
			if targetExt != fileExt {
				return content
			}
		}
		code := engine.Interpolate(a.Code, vars)
		return engine.InjectCode(content, a.Key, code)
	case engine.PatchAction:
		path := engine.Interpolate(a.Path, vars)
		val := engine.InterpolateValue(a.Value, vars)
		return engine.ApplyPatch(content, a.Op, path, val)
	}
	return content
}

func executeModuleAction(scope *Scope, cb, mod, method, target, as, rulePath, initialTarget string) {
	// Resolve module name from alias
	moduleName := mod
	if scope.Modules != nil {
		if val, ok := scope.Modules[mod]; ok {
			moduleName = val
		}
	}

	target = engine.Interpolate(target, scope.Flatten())

	switch strings.ToLower(moduleName) {
	case "utils", "u":
		switch strings.ToUpper(method) {
		case "LOG":
			fmt.Printf("%s\n", colorify(unescape(target)))
			if as != "" {
				scope.Set(as, target)
			}
		case "ASSERT":
			cond := target
			if strings.HasPrefix(cond, "exists(") && strings.HasSuffix(cond, ")") {
				path := strings.Trim(cond[7:len(cond)-1], "\"' ")
				absPath := filepath.Join(cb, path)
				if _, err := os.Stat(absPath); os.IsNotExist(err) {
					fmt.Printf("%s[ASSERT FAILED]%s Requirement not met: %s\n", colorRed, colorReset, cond)
					os.Exit(1)
				}
				if verbose {
					fmt.Printf("%s[ASSERT OK]%s Condition met: %s\n", colorGreen, colorReset, cond)
				}
			}
		case "INCLUDE":
			if rulePath == "" {
				fmt.Printf("%s[INCLUDE ERROR]%s Cannot include from non-file context\n", colorRed, colorReset)
				return
			}
			includePath := target
			if !filepath.IsAbs(includePath) {
				includePath = filepath.Join(filepath.Dir(rulePath), includePath)
			}
			irf, err := engine.ParseFile(includePath)
			if err != nil {
				fmt.Printf("%s[INCLUDE ERROR]%s %v\n", colorRed, colorReset, err)
				return
			}
			executeRules(irf.Rules, initialTarget, cb, includePath, scope)
		case "BREAK", "EXIT":
			code := 0
			if target != "" {
				if c, err := strconv.Atoi(target); err == nil {
					code = c
				}
			}
			os.Exit(code)
		}
	case "sys", "s":
		switch strings.ToUpper(method) {
		case "EXEC":
			cmdStr := target
			var cmd *exec.Cmd
			if runtime.GOOS == "windows" {
				cmd = exec.Command("cmd", "/C", cmdStr)
			} else {
				cmd = exec.Command("sh", "-c", cmdStr)
			}
			cmd.Dir = cb
			if as != "" {
				out, err := cmd.Output()
				if err != nil {
					fmt.Printf("%s[EXEC ERROR]%s %v\n", colorRed, colorReset, err)
				}
				scope.Set(as, strings.TrimSpace(string(out)))
			} else {
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				if err := cmd.Run(); err != nil {
					fmt.Printf("%s[EXEC ERROR]%s %v\n", colorRed, colorReset, err)
				}
			}
		}
	}
}
