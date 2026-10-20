package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/Nehonix-Team/xru/internal/utils"
	"github.com/Nehonix-Team/xru/pkg/ast"
	"github.com/Nehonix-Team/xru/pkg/checker"
	"github.com/Nehonix-Team/xru/pkg/dispatcher"
	"github.com/Nehonix-Team/xru/pkg/lexer"
	"github.com/Nehonix-Team/xru/pkg/lsp"
	"github.com/Nehonix-Team/xru/pkg/modules"
	"github.com/Nehonix-Team/xru/pkg/packager"
	"github.com/Nehonix-Team/xru/pkg/parser"
	"github.com/Nehonix-Team/xru/pkg/runtime"
)

func printUsage() {
	fmt.Println(`XRU 2.0 (XyPriss Rule Unit) - CLI & Template Orchestrator

Usage:
  xru run <file.xru> [flags...]     Execute an XRU 2.0 rule file
  xru fmt <file.xru ...>            Format XRU source code cleanly (prettier-like)
  xru lsp                           Start language server (LSP) for editor integration
  xru build [entry.xru] <output.xru> Build project into a single standalone .xru module
  xru pack [output.xpack]           Package project into a structured .xpack bundle
  xru unpack <file.xpack> [dest]    Unpack an .xpack bundle to destination
  xru version                       Show XRU version

Flags (for run):
  --<name>=<val>                    Pass CLI arguments matching #ARG declarations
  -v                                Enable verbose output`)
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(0)
	}

	// Détection du mode verbeux global (-v ou --verbose)
	verbose := false
	var filteredArgs []string
	for _, a := range os.Args[1:] {
		if a == "-v" || a == "--verbose" {
			verbose = true
		} else {
			filteredArgs = append(filteredArgs, a)
		}
	}

	if len(filteredArgs) == 0 {
		if verbose {
			fmt.Printf("XRU %s\n", utils.BinVersion)
			os.Exit(0)
		}
		printUsage()
		os.Exit(0)
	}

	cmd := filteredArgs[0]
	cmdArgs := filteredArgs[1:]

	switch cmd {
	case "version", "--version":
		fmt.Printf("XRU %s \n", utils.BinVersion)

	case "help", "--help", "-h":
		printUsage()

	case "lsp", "language-server":
		srv := lsp.NewServer(os.Stdin, os.Stdout)
		if err := srv.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "LSP server error: %v\n", err)
			os.Exit(1)
		}

	case "build":
		handleBuild(cmdArgs, verbose)

	case "fmt", "format":
		handleFmt(cmdArgs)

	case "pack":
		handlePack(cmdArgs, verbose)

	case "unpack":
		handleUnpack(cmdArgs, verbose)

	case "run":
		if len(cmdArgs) < 1 {
			fmt.Println("Error: 'xru run' requires a rule file. (e.g., xru run main.xru)")
			os.Exit(1)
		}
		handleRun(cmdArgs[0], cmdArgs[1:], verbose)

	default:
		// Si l'utilisateur passe directement un fichier .xru (ex: ./xru main.xru)
		if strings.HasSuffix(cmd, ".xru") {
			handleRun(cmd, cmdArgs, verbose)
		} else {
			fmt.Printf("Unknown command '%s'. Run 'xru --help' for usage.\n", cmd)
			os.Exit(1)
		}
	}
}

func handleRun(ruleFile string, cliArgs []string, verbose bool) {
	runtime.SetGlobalCliArgs(cliArgs)
	content, err := os.ReadFile(ruleFile)
	if err != nil {
		fmt.Printf("\033[31m[ERROR]\033[0m Cannot read file '%s': %v\n", ruleFile, err)
		os.Exit(1)
	}

	if verbose {
		fmt.Printf("\033[36m[DEBUG]\033[0m Lexing & Parsing '%s'...\n", ruleFile)
	}

	// 1. Lexing
	l := lexer.New(ruleFile, string(content))

	// 2. Parsing
	p := parser.New(l)
	prog := p.ParseProgram()
	if len(p.Errors()) > 0 {
		fmt.Printf("\033[31m[SYNTAX ERROR]\033[0m in %s:\n", ruleFile)
		for _, e := range p.Errors() {
			fmt.Printf("  • %s\n", e)
		}
		os.Exit(1)
	}

	if verbose {
		fmt.Printf("\033[36m[DEBUG]\033[0m Validating AST & Checking Types...\n")
	}

	// 3. Type Checking & Strict Validation
	c := checker.New()
	c.SetCurrentFile(ruleFile)
	c.Check(prog)
	if len(c.Errors()) > 0 {
		fmt.Printf("\033[31m[VALIDATION ERROR]\033[0m in %s:\n", ruleFile)
		for _, e := range c.Errors() {
			fmt.Printf("  • %s\n", e)
		}
		os.Exit(1)
	}

	// 4. Runtime & VM
	ruleDir := filepath.Dir(ruleFile)
	if absDir, err := filepath.Abs(ruleDir); err == nil {
		ruleDir = absDir
	}

	eval := runtime.NewEvaluator()
	eval.SetTreeDispatcher(func(dt *ast.DispatchTreeStatement, ev *runtime.Evaluator, treeEnv *runtime.Environment) error {
		disp := dispatcher.New(ruleDir, ev, treeEnv)
		if verbose {
			fmt.Printf("\033[36m[DEBUG]\033[0m Dispatching tree in '%s' (pattern: '%s')...\n", dt.Path, dt.Pattern)
		}
		return disp.DispatchTree(dt.Path, dt.Pattern)
	})
	env := runtime.NewEnvironment(nil)
	env.SetCurrentDir(ruleDir)

	// Extraction des #ARG déclarés dans le fichier racine et les modules importés pour validation stricte
	declaredArgs := make(map[string]*ast.ArgStatement)
	rootArgs := make(map[string]bool)
	var orderedArgs []*ast.ArgStatement
	seenOrdered := make(map[string]bool)

	collectArgsFromProg := func(p *ast.Program) {
		if p == nil {
			return
		}
		for _, stmt := range p.Statements {
			if argStmt, ok := stmt.(*ast.ArgStatement); ok {
				declaredArgs[strings.ToUpper(argStmt.Name)] = argStmt
				if argStmt.Short != "" {
					declaredArgs[strings.ToUpper(argStmt.Short)] = argStmt
				}
				if !seenOrdered[argStmt.Name] {
					seenOrdered[argStmt.Name] = true
					orderedArgs = append(orderedArgs, argStmt)
				}
			}
		}
	}
	collectArgsFromProg(prog)
	for _, stmt := range prog.Statements {
		if argStmt, ok := stmt.(*ast.ArgStatement); ok {
			rootArgs[strings.ToUpper(argStmt.Name)] = true
			if argStmt.Short != "" {
				rootArgs[strings.ToUpper(argStmt.Short)] = true
			}
		}
	}

	// Collecter les arguments dans les modules résolus via _mod.xru ou les imports
	modFilePath := modules.FindModConfigFile(ruleDir)
	var modCfg *modules.ModConfig
	if modFilePath != "" {
		if cfg, errs := modules.LoadModConfig(modFilePath); len(errs) == 0 && cfg != nil {
			modCfg = cfg
		}
	}

	scannedFiles := make(map[string]bool)
	scanFileForArgs := func(fPath string) {
		absPath, err := filepath.Abs(fPath)
		if err != nil || scannedFiles[absPath] {
			return
		}
		scannedFiles[absPath] = true
		content, err := os.ReadFile(absPath)
		if err != nil {
			return
		}
		pParser := parser.New(lexer.New(absPath, string(content)))
		pProg := pParser.ParseProgram()
		collectArgsFromProg(pProg)
	}

	for _, stmt := range prog.Statements {
		if fromStmt, ok := stmt.(*ast.FromUseStatement); ok && modCfg != nil {
			if fromStmt.Module != "" {
				if modDetails, err := modCfg.ResolveModule(fromStmt.Source, fromStmt.Module); err == nil && modDetails != nil {
					scanFileForArgs(modDetails.FilePath)
				}
			}
			for _, m := range fromStmt.Modules {
				if modDetails, err := modCfg.ResolveModule(fromStmt.Source, m); err == nil && modDetails != nil {
					scanFileForArgs(modDetails.FilePath)
				}
			}
		}
	}

	if modCfg != nil {
		for _, targetPattern := range modCfg.Mappings {
			if strings.HasSuffix(targetPattern, ".xru") {
				cleanPath := targetPattern
				if !filepath.IsAbs(cleanPath) {
					cleanPath = filepath.Join(modCfg.Dir, cleanPath)
				}
				scanFileForArgs(cleanPath)
			}
		}
	}

	// Interception de l'aide dynamique (--help ou -h)
	for _, a := range cliArgs {
		if a == "--help" || a == "-h" {
			printScriptHelp(ruleFile, declaredArgs)
			os.Exit(0)
		}
	}

	// Parsing et validation des arguments CLI (--name=val, -m minimal, positionnels, etc.)
	boundArgs := make(map[string]bool)
	posArgIdx := 0

	for i := 0; i < len(cliArgs); i++ {
		arg := cliArgs[i]
		if !strings.HasPrefix(arg, "-") {
			// Argument positionnel : bind au prochain argument déclaré non encore assigné
			for posArgIdx < len(orderedArgs) && boundArgs[orderedArgs[posArgIdx].Name] {
				posArgIdx++
			}
			if posArgIdx >= len(orderedArgs) {
				fmt.Printf("\033[31m[CLI ERROR]\033[0m Unexpected positional argument '%s'\n", arg)
				os.Exit(1)
			}
			declArg := orderedArgs[posArgIdx]
			posArgIdx++
			boundArgs[declArg.Name] = true
			actualKey := declArg.VarName()
			assignedVal := parseCliVal(arg)

			if rootArgs[strings.ToUpper(declArg.Name)] {
				env.Set(declArg.Name, assignedVal, false)
				env.Set(actualKey, assignedVal, false)
			}
			env.SetGlobal(declArg.Name, assignedVal)
			env.SetGlobal(actualKey, assignedVal)
			continue
		}

		trimmed := strings.TrimLeft(arg, "-")
		var key, valStr string
		hasValue := false

		if strings.Contains(trimmed, "=") {
			parts := strings.SplitN(trimmed, "=", 2)
			key = strings.ToUpper(parts[0])
			valStr = parts[1]
			hasValue = true
		} else {
			key = strings.ToUpper(trimmed)
			// Vérifier si le token suivant est la valeur (ex: --xems false, --mode minimal)
			if i+1 < len(cliArgs) && !strings.HasPrefix(cliArgs[i+1], "-") {
				valStr = cliArgs[i+1]
				hasValue = true
				i++ // consommer la valeur
			}
		}

		targetKey := key
		isNegative := false
		if strings.HasPrefix(key, "NO-") {
			targetKey = strings.TrimPrefix(key, "NO-")
			isNegative = true
		}

		// Validation stricte contre les #ARG déclarés
		if _, ok := declaredArgs[targetKey]; !ok {
			var validList []string
			for k := range declaredArgs {
				validList = append(validList, strings.ToLower(k))
			}
			sort.Strings(validList)
			fmt.Printf("\033[31m[ERROR]\033[0m Unknown CLI argument '%s'. Declared arguments: %s\n", arg, strings.Join(validList, ", "))
			os.Exit(1)
		}

		declArg := declaredArgs[targetKey]
		boundArgs[declArg.Name] = true
		actualKey := declArg.VarName()
		var assignedVal runtime.Value
		if isNegative {
			assignedVal = runtime.BoolValue{Val: false}
		} else if hasValue {
			assignedVal = parseCliVal(valStr)
		} else {
			assignedVal = runtime.BoolValue{Val: true}
		}
		// Si l'argument est déclaré dans le fichier racine, on le définit dans son scope local
		if rootArgs[targetKey] {
			env.Set(declArg.Name, assignedVal, false)
			env.Set(actualKey, assignedVal, false)
			env.Set(targetKey, assignedVal, false)
		}
		// Enregistrement global pour les modules importés
		env.SetGlobal(declArg.Name, assignedVal)
		env.SetGlobal(actualKey, assignedVal)
		env.SetGlobal(targetKey, assignedVal)
	}

	if verbose {
		fmt.Printf("\033[36m[DEBUG]\033[0m Executing rules on Runtime VM...\n")
	}

	// Exécution du programme
	if _, err := eval.Eval(prog, env); err != nil {
		if runtime.IsSignalHandling() {
			select {}
		}
		if !eval.RunErrorTraps(err) {
			fmt.Printf("\033[31m[RUNTIME ERROR]\033[0m %v\n", err)
		}
		eval.RunExitTraps(1)
		os.Exit(1)
	}
	eval.RunExitTraps(0)
}

func handlePack(args []string, verbose bool) {
	filename := "project.xpack"
	if len(args) > 0 {
		filename = args[0]
	}

	// Assure que l'extension est bien .xpack si non précisée
	if !strings.HasSuffix(filename, ".xpack") && !strings.Contains(filename, ".") {
		filename += ".xpack"
	}

	rootDir, _ := os.Getwd()
	outDir := filepath.Join(rootDir, "out")
	if err := os.MkdirAll(outDir, 0755); err != nil {
		fmt.Printf("\033[31m[PACK ERROR]\033[0m Failed to create 'out' directory: %v\n", err)
		os.Exit(1)
	}

	// Si l'utilisateur fournit déjà un chemin, on respecte, sinon on place dans out/
	var outputPath string
	if filepath.IsAbs(filename) || strings.Contains(filename, string(filepath.Separator)) {
		outputPath = filename
	} else {
		outputPath = filepath.Join(outDir, filename)
	}
	if !filepath.IsAbs(outputPath) {
		outputPath = filepath.Join(rootDir, outputPath)
	}

	// Initialisation du packager (auto-détecte .xruinclude en priorité sur .xruignore)
	pkg := packager.New(rootDir, "")
	pkg.Verbose = verbose

	// Empaqueter l'ensemble du projet racine en filtrant strictement selon le fichier actif
	entryDir := "."

	relOutput, _ := filepath.Rel(rootDir, outputPath)
	fmt.Printf("\033[36m[PACK]\033[0m Packaging project into '\033[1m%s\033[0m' (using %s)...\n", relOutput, pkg.FilterFile())
	if err := pkg.Pack(outputPath, entryDir); err != nil {
		fmt.Printf("\033[31m[PACK ERROR]\033[0m %v\n", err)
		os.Exit(1)
	}

	// Calcul et affichage de la taille finale
	outStat, statErr := os.Stat(outputPath)
	sizeStr := ""
	if statErr == nil {
		sizeBytes := outStat.Size()
		if sizeBytes < 1024 {
			sizeStr = fmt.Sprintf(" (%d bytes)", sizeBytes)
		} else if sizeBytes < 1024*1024 {
			sizeStr = fmt.Sprintf(" (%.2f KB)", float64(sizeBytes)/1024.0)
		} else {
			sizeStr = fmt.Sprintf(" (%.2f MB)", float64(sizeBytes)/(1024.0*1024.0))
		}
	}
	fmt.Printf("\033[32m[SUCCESS]\033[0m Successfully built package %s%s\n", relOutput, sizeStr)
}

func handleUnpack(args []string, verbose bool) {
	if len(args) < 1 {
		fmt.Println("Error: 'xru unpack' requires package path. (e.g., xru unpack project.xpack [dest])")
		os.Exit(1)
	}
	packFile := args[0]
	dest := "."
	if len(args) > 1 {
		dest = args[1]
	}

	rootDir, _ := os.Getwd()
	pkg := packager.New(rootDir, ".xruignore")
	pkg.Verbose = verbose
	manifest, err := pkg.Unpack(packFile, dest)
	if err != nil {
		fmt.Printf("\033[31m[UNPACK ERROR]\033[0m %v\n", err)
		os.Exit(1)
	}

	version := "2.0.0"
	if manifest != nil {
		version = manifest.Version
	}
	fmt.Printf("\033[32m[SUCCESS]\033[0m Unpacked package (v%s) to %s\n", version, dest)
}

func parseCliVal(val string) runtime.Value {
	lower := strings.ToLower(val)
	if lower == "true" {
		return runtime.BoolValue{Val: true}
	}
	if lower == "false" {
		return runtime.BoolValue{Val: false}
	}
	if lower == "simple" || lower == "minimal" {
		return runtime.StringValue{Val: "minimal"}
	}
	if n, err := strconv.ParseFloat(val, 64); err == nil {
		return runtime.NumberValue{Val: n}
	}
	return runtime.StringValue{Val: val}
}

func printScriptHelp(ruleFile string, declaredArgs map[string]*ast.ArgStatement) {
	baseName := filepath.Base(ruleFile)
	if baseName == "init.xru" {
		fmt.Printf("\nUsage: xfpm init [name] [options]\n\n")
	} else {
		fmt.Printf("\nUsage: xru run %s [options]\n\n", baseName)
	}
	if len(declaredArgs) > 0 {
		fmt.Println("Options:")
		seen := make(map[*ast.ArgStatement]bool)
		var list []*ast.ArgStatement
		for _, arg := range declaredArgs {
			if !seen[arg] {
				seen[arg] = true
				list = append(list, arg)
			}
		}
		sort.Slice(list, func(i, j int) bool {
			return list[i].Name < list[j].Name
		})
		for _, arg := range list {
			flagStr := fmt.Sprintf("--%s", arg.Name)
			if arg.Short != "" {
				flagStr = fmt.Sprintf("--%s, -%s", arg.Name, arg.Short)
			}
			typeStr := arg.Type
			if typeStr == "" {
				typeStr = "any"
			}
			defaultStr := ""
			if arg.DefaultValue != nil {
				defaultStr = fmt.Sprintf(" (default: %s)", arg.DefaultValue.String())
			}
			fmt.Printf("  %-28s <%s>%s\n", flagStr, typeStr, defaultStr)
		}
		fmt.Printf("  %-28s Show this help message\n\n", "--help, -h")
	}
}

