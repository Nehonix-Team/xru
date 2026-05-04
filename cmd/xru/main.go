package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Nehonix-Team/xru/internal/engine/parser"
	"github.com/Nehonix-Team/xru/internal/utils"
)

func main() {
	v := flag.Bool("v", false, "Enable verbose output")
	flag.Parse()
	verbose = *v
	args := flag.Args()

	if len(args) < 1 {
		os.Exit(1)
	}

	switch args[0] {
	case "version":
		fmt.Printf("XRU %s\n", utils.BinVersion)
	case "upgrade":
		handleUpgrade()
	case "build":
		if len(args) < 2 {
			os.Exit(1)
		}
		runBuild(args[1])
	default:
		target, tArgs := parseRunArgs(args)
		terminalArgs = tArgs
		runPatch(args[0], target)
	}
}

func parseRunArgs(args []string) (target string, tArgs []string) {
	target = "."
	if len(args) > 1 {
		if strings.HasPrefix(args[1], "-") {
			tArgs = args[1:]
		} else {
			target = args[1]
			tArgs = args[2:]
		}
	}
	return
}

func runPatch(rulePath, targetDir string) {
	currentFile = rulePath
	absTarget, _ := filepath.Abs(targetDir)

	rf, err := parser.ParseFile(rulePath)
	if err != nil {
		fmt.Printf("%sError: %v%s\n", colorRed, err, colorReset)
		os.Exit(1)
	}

	rootScope := newRootScope()
	executeRules(rf.Rules, absTarget, absTarget, rulePath, rootScope)
	rootScope.CheckUnused(currentFile)
}
