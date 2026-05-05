package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/Nehonix-Team/xru/internal/utils"
	"github.com/Nehonix-Team/xru/pkg/engine"
)

func main() {
	v := flag.Bool("v", false, "Enable verbose output")
	flag.Parse()
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
		runner := engine.NewRunner()
		runner.Verbose = *v
		runner.TerminalArgs = tArgs
		
		if err := runner.Run(args[0], target); err != nil {
			fmt.Printf("%serror:%s %v\n", engine.ColorRed, engine.ColorReset, err)
			os.Exit(1)
		}
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

