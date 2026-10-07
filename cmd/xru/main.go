package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Nehonix-Team/xru/internal/utils"
	"github.com/Nehonix-Team/xru/pkg/ast"
	"github.com/Nehonix-Team/xru/pkg/checker"
	"github.com/Nehonix-Team/xru/pkg/dispatcher"
	"github.com/Nehonix-Team/xru/pkg/lexer"
	"github.com/Nehonix-Team/xru/pkg/packager"
	"github.com/Nehonix-Team/xru/pkg/parser"
	"github.com/Nehonix-Team/xru/pkg/runtime"
)

func printUsage() {
	fmt.Println(`XRU 2.0 (XyPriss Rule Unit) - CLI & Template Orchestrator

Usage:
  xru run <file.xru> [flags...]     Execute an XRU 2.0 rule file
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
			fmt.Printf("XRU v2.0.0 (%s)\n", utils.BinVersion)
			os.Exit(0)
		}
		printUsage()
		os.Exit(0)
	}

	cmd := filteredArgs[0]
	cmdArgs := filteredArgs[1:]

	switch cmd {
	case "version", "--version":
		fmt.Printf("XRU v2.0.0 (%s)\n", utils.BinVersion)

	case "help", "--help", "-h":
		printUsage()

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
	c.Check(prog)
	if len(c.Errors()) > 0 {
		fmt.Printf("\033[31m[VALIDATION ERROR]\033[0m in %s:\n", ruleFile)
		for _, e := range c.Errors() {
			fmt.Printf("  • %s\n", e)
		}
		os.Exit(1)
	}

	// 4. Runtime & VM
	eval := runtime.NewEvaluator()
	env := runtime.NewEnvironment(nil)

	// Injection des arguments CLI (--name=val ou --plugin)
	for _, arg := range cliArgs {
		trimmed := strings.TrimPrefix(arg, "--")
		trimmed = strings.TrimPrefix(trimmed, "-")
		parts := strings.SplitN(trimmed, "=", 2)
		key := strings.ToUpper(parts[0])
		if len(parts) == 2 {
			val := parts[1]
			if val == "true" {
				env.Set(key, runtime.BoolValue{Val: true}, false)
			} else if val == "false" {
				env.Set(key, runtime.BoolValue{Val: false}, false)
			} else {
				env.Set(key, runtime.StringValue{Val: val}, false)
			}
		} else {
			env.Set(key, runtime.BoolValue{Val: true}, false)
		}
	}

	if verbose {
		fmt.Printf("\033[36m[DEBUG]\033[0m Executing rules on Runtime VM...\n")
	}

	// Exécution du programme
	if _, err := eval.Eval(prog, env); err != nil {
		fmt.Printf("\033[31m[RUNTIME ERROR]\033[0m %v\n", err)
		os.Exit(1)
	}

	// Traitement de #DISPATCH_TREE si présent
	for _, stmt := range prog.Statements {
		if dt, ok := stmt.(*ast.DispatchTreeStatement); ok {
			rootDir, _ := os.Getwd()
			disp := dispatcher.New(rootDir, eval, env)
			if verbose {
				fmt.Printf("\033[36m[DEBUG]\033[0m Dispatching tree in '%s' (pattern: '%s')...\n", dt.Path, dt.Pattern)
			}
			if err := disp.DispatchTree(dt.Path, dt.Pattern); err != nil {
				fmt.Printf("\033[31m[DISPATCH ERROR]\033[0m %v\n", err)
				os.Exit(1)
			}
		}
	}
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

	// Seul et unique fichier d'ignore officiel : .xruignore
	ignoreFile := ".xruignore"
	pkg := packager.New(rootDir, ignoreFile)
	pkg.Verbose = verbose

	entryDir := "src"
	if _, err := os.Stat(filepath.Join(rootDir, "src")); os.IsNotExist(err) {
		entryDir = "."
	}

	relOutput, _ := filepath.Rel(rootDir, outputPath)
	fmt.Printf("\033[36m[PACK]\033[0m Packaging project into '\033[1m%s\033[0m' (using %s)...\n", relOutput, ignoreFile)
	if err := pkg.Pack(outputPath, entryDir); err != nil {
		fmt.Printf("\033[31m[PACK ERROR]\033[0m %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("\033[32m[SUCCESS]\033[0m Successfully built package %s\n", relOutput)
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
