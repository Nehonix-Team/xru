package runtime

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	goRuntime "runtime"
	"strconv"
	"strings"
	"syscall"

	"github.com/Nehonix-Team/xru/internal/utils"
	"github.com/Nehonix-Team/xru/pkg/ast"
	"github.com/Nehonix-Team/xru/pkg/modules"
)

// parseSemVer parses semantic version strings like "v2.0.67", "2.0.67", "2.1"
func parseSemVer(v string) (major, minor, patch int, err error) {
	v = strings.TrimSpace(v)
	v = strings.TrimPrefix(v, "v")
	v = strings.TrimPrefix(v, "V")

	// Strip pre-release or build metadata (e.g., "-beta", "+build")
	if idx := strings.IndexAny(v, "-+"); idx != -1 {
		v = v[:idx]
	}

	parts := strings.Split(v, ".")
	if len(parts) >= 1 && parts[0] != "" {
		major, err = strconv.Atoi(parts[0])
		if err != nil {
			return 0, 0, 0, fmt.Errorf("invalid semver major: %s", parts[0])
		}
	}
	if len(parts) >= 2 && parts[1] != "" {
		minor, err = strconv.Atoi(parts[1])
		if err != nil {
			return 0, 0, 0, fmt.Errorf("invalid semver minor: %s", parts[1])
		}
	} 
	if len(parts) >= 3 && parts[2] != "" {
		patch, err = strconv.Atoi(parts[2])
		if err != nil {
			return 0, 0, 0, fmt.Errorf("invalid semver patch: %s", parts[2])
		}
	}
	return major, minor, patch, nil
}

// compareSemVer returns:
// -1 if v1 < v2
//  0 if v1 == v2
//  1 if v1 > v2
func compareSemVer(v1, v2 string) (int, error) {
	maj1, min1, pat1, err1 := parseSemVer(v1)
	if err1 != nil {
		return 0, err1
	}
	maj2, min2, pat2, err2 := parseSemVer(v2)
	if err2 != nil {
		return 0, err2
	}

	if maj1 != maj2 {
		if maj1 < maj2 {
			return -1, nil
		}
		return 1, nil
	}
	if min1 != min2 {
		if min1 < min2 {
			return -1, nil
		}
		return 1, nil
	}
	if pat1 != pat2 {
		if pat1 < pat2 {
			return -1, nil
		}
		return 1, nil
	}
	return 0, nil
}

// checkVersionConstraint evaluates a version constraint against currentVer.
// Supported operators: >=, <=, ==, =, >, <, or plain version (defaults to >=).
func checkVersionConstraint(currentVer, constraint string) (bool, error) {
	constraint = strings.TrimSpace(constraint)
	if constraint == "" || constraint == "*" {
		return true, nil
	}

	op := ">="
	targetVer := constraint

	if strings.HasPrefix(constraint, ">=") {
		op = ">="
		targetVer = strings.TrimSpace(constraint[2:])
	} else if strings.HasPrefix(constraint, "<=") {
		op = "<="
		targetVer = strings.TrimSpace(constraint[2:])
	} else if strings.HasPrefix(constraint, "==") {
		op = "=="
		targetVer = strings.TrimSpace(constraint[2:])
	} else if strings.HasPrefix(constraint, "=") {
		op = "=="
		targetVer = strings.TrimSpace(constraint[1:])
	} else if strings.HasPrefix(constraint, ">") {
		op = ">"
		targetVer = strings.TrimSpace(constraint[1:])
	} else if strings.HasPrefix(constraint, "<") {
		op = "<"
		targetVer = strings.TrimSpace(constraint[1:])
	}

	cmp, err := compareSemVer(currentVer, targetVer)
	if err != nil {
		return false, err
	}

	switch op {
	case ">=":
		return cmp >= 0, nil
	case "<=":
		return cmp <= 0, nil
	case "==", "=":
		return cmp == 0, nil
	case ">":
		return cmp > 0, nil
	case "<":
		return cmp < 0, nil
	default:
		return false, fmt.Errorf("unknown operator '%s'", op)
	}
}

// autoUpdateXRU fetches and executes the native deployment script from dll.nehonix.com
// to update both the binary and the stdlib atomically.
func autoUpdateXRU(silent bool) error {
	isWindows := goRuntime.GOOS == "windows"
	scriptName := "install.sh"
	if isWindows {
		scriptName = "install.ps1"
	}

	installerURL := fmt.Sprintf("https://dll.nehonix.com/repo/xypriss/xfpm/scripts/%s", scriptName)

	if !silent {
		fmt.Printf("\033[36m[XRU AUTO-UPDATE]\033[0m Fetching deployment engine from %s...\n", installerURL)
	}

	resp, err := http.Get(installerURL)
	if err != nil {
		return fmt.Errorf("failed to fetch installer script: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("installer server returned status: %s", resp.Status)
	}

	tmpPattern := "xru_auto_install_*.sh"
	if isWindows {
		tmpPattern = "xru_auto_install_*.ps1"
	}

	tmpFile, err := os.CreateTemp("", tmpPattern)
	if err != nil {
		return fmt.Errorf("failed to create temp installer script: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	_, err = io.Copy(tmpFile, resp.Body)
	tmpFile.Close()
	if err != nil {
		return fmt.Errorf("failed to write installer script: %w", err)
	}

	if !isWindows {
		_ = os.Chmod(tmpPath, 0755)
	}

	if !silent {
		fmt.Printf("\033[36m[XRU AUTO-UPDATE]\033[0m Executing update sequence for XRU and stdlib...\n")
	}

	var cmd *exec.Cmd
	if isWindows {
		cmd = exec.Command("powershell.exe", "-NoProfile", "-ExecutionPolicy", "Bypass", "-File", tmpPath, "-Action", "xru")
	} else {
		cmd = exec.Command("bash", tmpPath, "xru")
	}

	if !silent {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("update sequence execution failed: %w", err)
	}

	binName := "xru"
	if isWindows {
		binName = "xru.exe"
	}

	execPath, err := os.Executable()
	if err != nil {
		execPath = binName
	}

	// Resolve active/updated binary location (in case previous binary was cleaned/relocated by installer)
	targetExec := execPath
	if _, statErr := os.Stat(targetExec); statErr != nil {
		if p, err := exec.LookPath(binName); err == nil {
			targetExec = p
		} else if home, err := os.UserHomeDir(); err == nil {
			candidates := []string{
				filepath.Join(home, ".xfpm", "bin", binName),
				filepath.Join(home, ".local", "bin", binName),
			}
			if isWindows {
				if localAppData := os.Getenv("LOCALAPPDATA"); localAppData != "" {
					candidates = append(candidates, filepath.Join(localAppData, "xfpm", "bin", binName))
				}
			} else {
				candidates = append(candidates, filepath.Join("/usr/local/bin", binName))
			}
			for _, candidate := range candidates {
				if _, err := os.Stat(candidate); err == nil {
					targetExec = candidate
					break
				}
			}
		}
	}
	targetExec, _ = filepath.EvalSymlinks(targetExec)

	if !silent {
		fmt.Printf("\n\033[32m[XRU AUTO-UPDATE]\033[0m Successfully updated XRU runtime and standard library.\n")
		fmt.Printf("\033[36m[XRU AUTO-UPDATE]\033[0m Resuming execution with updated engine...\n\n")
	}

	// Restart process seamlessly
	updateEnv := append(os.Environ(), "XRU_AUTO_UPDATE_ATTEMPT=1")
	if !isWindows {
		return syscall.Exec(targetExec, os.Args, updateEnv)
	}

	restartCmd := exec.Command(targetExec, os.Args[1:]...)
	restartCmd.Env = updateEnv
	restartCmd.Stdin = os.Stdin
	restartCmd.Stdout = os.Stdout
	restartCmd.Stderr = os.Stderr
	if err := restartCmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		os.Exit(1)
	}
	os.Exit(0)
	return nil
}

// sysRequireFn implements Sys.require(constraint, options?)
func sysRequireFn(args []Value) (Value, error) {
	if len(args) < 1 {
		return NilValue{}, fmt.Errorf("Sys.require requires at least 1 argument (version constraint)")
	}

	constraint := args[0].String()
	onMismatch := "fatal"
	mode := "log"

	if len(args) > 1 {
		if m, ok := args[1].(MapValue); ok {
			if val, exists := m.Pairs["onMismatch"]; exists {
				onMismatch = strings.ToLower(val.String())
			}
			if val, exists := m.Pairs["mode"]; exists {
				mode = strings.ToLower(val.String())
			}
		}
	}

	currentVer := utils.BinVersion
	matched, err := checkVersionConstraint(currentVer, constraint)
	if err != nil {
		return NilValue{}, fmt.Errorf("invalid version constraint '%s': %w", constraint, err)
	}

	if matched {
		return BoolValue{Val: true}, nil
	}

	// Mismatch handling
	switch onMismatch {
	case "auto-update", "update":
		if os.Getenv("XRU_AUTO_UPDATE_ATTEMPT") != "" {
			fmt.Fprintf(os.Stderr, "\n\033[1;31m[XRU VERSION ERROR]\033[0m Successfully updated to latest available release (\033[33m%s\033[0m), but required version \033[1m%s\033[0m is still not satisfied.\nThe requested release may not be published on GitHub yet.\n\n", currentVer, constraint)
			os.Exit(1)
		}
		err := autoUpdateXRU(mode == "silent")
		if err != nil {
			fmt.Fprintf(os.Stderr, "\033[31m[XRU AUTO-UPDATE ERROR]\033[0m %v\n", err)
			os.Exit(1)
		}
		return BoolValue{Val: true}, nil

	case "warn":
		if mode != "silent" {
			fmt.Fprintf(os.Stderr, "\033[33m[XRU VERSION WARNING]\033[0m !:XRU %s, currently running %s.\n", constraint, currentVer)
		}
		return BoolValue{Val: false}, nil

	default: // "fatal"
		fmt.Fprintf(os.Stderr, "\n\033[1;31m[XRU VERSION ERROR]\033[0m This script requires XRU \033[1m%s\033[0m (current version: \033[33m%s\033[0m).\n", constraint, currentVer)
		if goRuntime.GOOS == "windows" {
			fmt.Fprintf(os.Stderr, "To update XRU on Windows (PowerShell):\n")
			fmt.Fprintf(os.Stderr, "  \033[36m• powershell -ExecutionPolicy Bypass -Command \"irm https://dll.nehonix.com/repo/xypriss/xfpm/scripts/install.ps1 | iex\" -Action xru\033[0m\n")
			fmt.Fprintf(os.Stderr, "  \033[36m• or install full suite: irm https://dll.nehonix.com/repo/xypriss/xfpm/scripts/install.ps1 | iex\033[0m\n\n")
		} else {
			fmt.Fprintf(os.Stderr, "To update XRU on Linux / macOS:\n")
			fmt.Fprintf(os.Stderr, "  \033[36m• curl -fsSL https://dll.nehonix.com/repo/xypriss/xfpm/scripts/install.sh | bash -s -- xru\033[0m\n")
			fmt.Fprintf(os.Stderr, "  \033[36m• or install full suite: curl -fsSL https://dll.nehonix.com/repo/xypriss/xfpm/scripts/install.sh | bash\033[0m\n\n")
		}
		os.Exit(1)
		return BoolValue{Val: false}, nil
	}
}

// sysVersionFn implements Sys.version()
func sysVersionFn(args []Value) (Value, error) {
	return StringValue{Val: utils.BinVersion}, nil
}

// CheckAndEnforceAstRequire pre-scans the program AST for top-level Sys.require(...) declarations
// and validates or auto-updates the XRU engine BEFORE the strict type checker runs.
func CheckAndEnforceAstRequire(prog *ast.Program) error {
	if prog == nil {
		return nil
	}

	sysAliases := map[string]bool{
		"sys": true,
		"Sys": true,
	}

	// 1. Scan use statements to track aliases (e.g. `use sys as s6`)
	for _, stmt := range prog.Statements {
		if useStmt, ok := stmt.(*ast.UseStatement); ok {
			if strings.EqualFold(useStmt.Module, "sys") {
				if useStmt.Alias != "" {
					sysAliases[useStmt.Alias] = true
				}
				sysAliases[useStmt.Module] = true
			}
		}
	}

	astConstants := make(map[string]string)

	// Resolve local constants declared in the root program
	for _, stmt := range prog.Statements {
		if vs, ok := stmt.(*ast.VarStatement); ok && vs.IsConst {
			if sLit, ok := vs.Value.(*ast.StringLiteral); ok {
				astConstants[vs.Name] = sLit.Value
			}
		}
	}

	// Resolve imported constants via _mod.xru mapping
	fileDir := "."
	if filename := prog.Pos().Filename; filename != "" {
		fileDir = filepath.Dir(filename)
	}
	modFilePath := modules.FindModConfigFile(fileDir)
	var modCfg *modules.ModConfig
	if modFilePath != "" {
		if cfg, errs := modules.LoadModConfig(modFilePath); len(errs) == 0 {
			modCfg = cfg
		}
	}

	if modCfg != nil {
		for _, stmt := range prog.Statements {
			fromStmt, ok := stmt.(*ast.FromUseStatement)
			if !ok {
				continue
			}

			if fromStmt.Module != "" {
				modDetails, err := modCfg.ResolveModule(fromStmt.Source, fromStmt.Module)
				if err == nil && modDetails != nil {
					// 1. Single member: from "source" use module::member [as alias]
					if fromStmt.Member != "" {
						if exp, ok := modDetails.Exports[fromStmt.Member]; ok && exp.Kind == "const" && exp.Value != "" {
							name := fromStmt.Member
							if fromStmt.Alias != "" {
								name = fromStmt.Alias
							}
							astConstants[name] = exp.Value
						}
					}
					// 2. Multiple members: from "source" use module::@map{a, b} or module::(a, b)
					if len(fromStmt.Members) > 0 {
						for _, mem := range fromStmt.Members {
							if exp, ok := modDetails.Exports[mem]; ok && exp.Kind == "const" && exp.Value != "" {
								astConstants[mem] = exp.Value
							}
						}
					}
					// 3. Whole module imported: from "source" use module [as alias]
					if fromStmt.Member == "" && len(fromStmt.Members) == 0 {
						prefix := fromStmt.Module
						if fromStmt.Alias != "" {
							prefix = fromStmt.Alias
						}
						for memName, exp := range modDetails.Exports {
							if exp.Kind == "const" && exp.Value != "" {
								astConstants[prefix+"."+memName] = exp.Value
								astConstants[prefix+"::"+memName] = exp.Value
							}
						}
					}
				}
			}

			// 4. Multiple modules: from "source" use (modA, modB)
			if len(fromStmt.Modules) > 0 {
				for _, mName := range fromStmt.Modules {
					modDetails, err := modCfg.ResolveModule(fromStmt.Source, mName)
					if err == nil && modDetails != nil {
						for memName, exp := range modDetails.Exports {
							if exp.Kind == "const" && exp.Value != "" {
								astConstants[mName+"."+memName] = exp.Value
								astConstants[mName+"::"+memName] = exp.Value
							}
						}
					}
				}
			}
		}
	}

	// 2. Scan statements for Sys.require(...)
	for _, stmt := range prog.Statements {
		exprStmt, ok := stmt.(*ast.ExpressionStatement)
		if !ok {
			continue
		}
		callExpr, ok := exprStmt.Expression.(*ast.CallExpression)
		if !ok {
			continue
		}

		isRequireCall := false
		if memberExpr, ok := callExpr.Caller.(*ast.MemberExpression); ok {
			if strings.EqualFold(memberExpr.Property, "require") {
				if ident, ok := memberExpr.Object.(*ast.Identifier); ok {
					if sysAliases[ident.Value] || strings.EqualFold(ident.Value, "sys") {
						isRequireCall = true
					}
				}
			}
		} else if scopeExpr, ok := callExpr.Caller.(*ast.ScopeAccessExpression); ok {
			if strings.EqualFold(scopeExpr.Member, "require") {
				if sysAliases[scopeExpr.Scope] || strings.EqualFold(scopeExpr.Scope, "sys") {
					isRequireCall = true
				}
			}
		}

		if !isRequireCall {
			continue
		}

		// Found Sys.require(constraint, options?)
		if len(callExpr.Arguments) < 1 {
			continue
		}

		var constraint string
		if strLit, ok := callExpr.Arguments[0].(*ast.StringLiteral); ok {
			constraint = strLit.Value
			spans := ast.ExtractInterpolations(constraint)
			if len(spans) > 0 {
				for _, span := range spans {
					if cVal, ok := astConstants[span.Expr]; ok {
						constraint = strings.Replace(constraint, "$f{"+span.Expr+"}", cVal, 1)
					}
				}
			}
		} else if ident, ok := callExpr.Arguments[0].(*ast.Identifier); ok {
			if cVal, ok := astConstants[ident.Value]; ok {
				constraint = cVal
			} else {
				continue
			}
		} else if memberExpr, ok := callExpr.Arguments[0].(*ast.MemberExpression); ok {
			if objIdent, ok := memberExpr.Object.(*ast.Identifier); ok {
				key := objIdent.Value + "." + memberExpr.Property
				if cVal, ok := astConstants[key]; ok {
					constraint = cVal
				} else {
					continue
				}
			} else {
				continue
			}
		} else if scopeExpr, ok := callExpr.Arguments[0].(*ast.ScopeAccessExpression); ok {
			key := scopeExpr.Scope + "::" + scopeExpr.Member
			if cVal, ok := astConstants[key]; ok {
				constraint = cVal
			} else {
				continue
			}
		} else {
			continue
		}

		// If there are still unresolved interpolations (e.g. non-constant dynamic values),
		// skip pre-AST constraint parsing so we don't fail with a bogus semver error,
		// and allow runtime sysRequireFn to perform the check once evaluated.
		if strings.Contains(constraint, "$f{") {
			continue
		}

		onMismatch := "fatal"
		mode := "log"

		if len(callExpr.Arguments) > 1 {
			if mapLit, ok := callExpr.Arguments[1].(*ast.MapLiteral); ok {
				for _, pair := range mapLit.Pairs {
					keyStr := ""
					if kIdent, ok := pair.Key.(*ast.Identifier); ok {
						keyStr = kIdent.Value
					} else if kStr, ok := pair.Key.(*ast.StringLiteral); ok {
						keyStr = kStr.Value
					}

					valStr := ""
					if vIdent, ok := pair.Value.(*ast.Identifier); ok {
						valStr = vIdent.Value
					} else if vStr, ok := pair.Value.(*ast.StringLiteral); ok {
						valStr = vStr.Value
					}

					if strings.EqualFold(keyStr, "onMismatch") {
						onMismatch = strings.ToLower(valStr)
					}
					if strings.EqualFold(keyStr, "mode") {
						mode = strings.ToLower(valStr)
					}
				}
			}
		}

		// Enforce version requirement immediately!
		currentVer := utils.BinVersion
		matched, err := checkVersionConstraint(currentVer, constraint)
		if err != nil {
			return fmt.Errorf("invalid version constraint '%s': %w", constraint, err)
		}

		if matched {
			return nil
		}

		// Version mismatch handling
		switch onMismatch {
		case "auto-update", "update":
			if os.Getenv("XRU_AUTO_UPDATE_ATTEMPT") != "" {
				fmt.Fprintf(os.Stderr, "\n\033[1;31m[XRU VERSION ERROR]\033[0m Successfully updated to latest available release (\033[33m%s\033[0m), but required version \033[1m%s\033[0m is still not satisfied.\nThe requested release may not be published on GitHub yet.\n\n", currentVer, constraint)
				os.Exit(1)
			}
			err := autoUpdateXRU(mode == "silent")
			if err != nil {
				fmt.Fprintf(os.Stderr, "\033[31m[XRU AUTO-UPDATE ERROR]\033[0m %v\n", err)
				os.Exit(1)
			}
			return nil

		case "warn":
			if mode != "silent" {
				fmt.Fprintf(os.Stderr, "\033[33m[XRU VERSION WARNING]\033[0m !:XRU %s, currently running %s.\n", constraint, currentVer)
			}
			return nil

		default: // "fatal"
			fmt.Fprintf(os.Stderr, "\n\033[1;31m[XRU VERSION ERROR]\033[0m This script requires XRU \033[1m%s\033[0m (current version: \033[33m%s\033[0m).\n", constraint, currentVer)
			if goRuntime.GOOS == "windows" {
				fmt.Fprintf(os.Stderr, "To update XRU on Windows (PowerShell):\n")
				fmt.Fprintf(os.Stderr, "  \033[36m• powershell -ExecutionPolicy Bypass -Command \"irm https://dll.nehonix.com/repo/xypriss/xfpm/scripts/install.ps1 | iex\" -Action xru\033[0m\n")
				fmt.Fprintf(os.Stderr, "  \033[36m• or install full suite: irm https://dll.nehonix.com/repo/xypriss/xfpm/scripts/install.ps1 | iex\033[0m\n\n")
			} else {
				fmt.Fprintf(os.Stderr, "To update XRU on Linux / macOS:\n")
				fmt.Fprintf(os.Stderr, "  \033[36m• curl -fsSL https://dll.nehonix.com/repo/xypriss/xfpm/scripts/install.sh | bash -s -- xru\033[0m\n")
				fmt.Fprintf(os.Stderr, "  \033[36m• or install full suite: curl -fsSL https://dll.nehonix.com/repo/xypriss/xfpm/scripts/install.sh | bash\033[0m\n\n")
			}
			os.Exit(1)
			return fmt.Errorf("XRU version %s does not satisfy %s", currentVer, constraint)
		}
	}

	return nil
}

