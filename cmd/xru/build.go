package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Nehonix-Team/xru/pkg/packager"
)

// handleBuild bundles a project into a single standalone, minified .xru file.
// Usage:
//   xru build <output.xru>             (defaults entry to main.xru)
//   xru build <entry.xru> <output.xru>
//   xru build <entry.xru> -o <output.xru>
func handleBuild(args []string, verbose bool) {
	var entryFile string
	var outputFile string

	var cleanArgs []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "-o" || a == "--out" {
			if i+1 < len(args) {
				outputFile = args[i+1]
				i++
				continue
			}
		} else if strings.HasPrefix(a, "-o=") {
			outputFile = strings.TrimPrefix(a, "-o=")
		} else if strings.HasPrefix(a, "--out=") {
			outputFile = strings.TrimPrefix(a, "--out=")
		} else {
			cleanArgs = append(cleanArgs, a)
		}
	}

	if outputFile == "" {
		if len(cleanArgs) >= 2 {
			entryFile = cleanArgs[0]
			outputFile = cleanArgs[1]
		} else if len(cleanArgs) == 1 {
			arg := cleanArgs[0]
			if _, err := os.Stat(arg); os.IsNotExist(err) {
				entryFile = "main.xru"
				outputFile = arg
			} else {
				if filepath.Base(arg) == "main.xru" {
					fmt.Println("Error: 'xru build' requires an output file path.")
					fmt.Println("Usage: xru build [entry.xru] <output.xru>")
					os.Exit(1)
				}
				if _, err := os.Stat("main.xru"); err == nil {
					entryFile = "main.xru"
					outputFile = arg
				} else {
					fmt.Println("Error: 'xru build' requires an output file path.")
					fmt.Println("Usage: xru build [entry.xru] <output.xru>")
					os.Exit(1)
				}
			}
		} else {
			fmt.Println("Error: 'xru build' requires an output file path.")
			fmt.Println("Usage: xru build [entry.xru] <output.xru>")
			os.Exit(1)
		}
	} else {
		if len(cleanArgs) >= 1 {
			entryFile = cleanArgs[0]
		} else {
			entryFile = "main.xru"
		}
	}

	if !strings.HasSuffix(outputFile, ".xru") && !strings.Contains(filepath.Base(outputFile), ".") {
		outputFile += ".xru"
	}

	rootDir, _ := os.Getwd()
	var absOutput string
	if filepath.IsAbs(outputFile) {
		absOutput = outputFile
	} else {
		absOutput = filepath.Join(rootDir, outputFile)
	}

	if err := os.MkdirAll(filepath.Dir(absOutput), 0755); err != nil {
		fmt.Printf("\033[31m[BUILD ERROR]\033[0m Failed to create directory '%s': %v\n", filepath.Dir(absOutput), err)
		os.Exit(1)
	}

	pkg := packager.New(rootDir, ".xruignore")
	pkg.Verbose = verbose

	relEntry := entryFile
	if filepath.IsAbs(entryFile) {
		if re, err := filepath.Rel(rootDir, entryFile); err == nil {
			relEntry = re
		}
	}
	relOutput, _ := filepath.Rel(rootDir, absOutput)
	fmt.Printf("\033[36m[BUILD]\033[0m Bundling '\033[1m%s\033[0m' into '\033[1m%s\033[0m'...\n", relEntry, relOutput)

	bundledContent, err := pkg.BundleAndMinifyWithEntry(entryFile)
	if err != nil {
		fmt.Printf("\033[31m[BUILD ERROR]\033[0m %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(absOutput, []byte(bundledContent), 0644); err != nil {
		fmt.Printf("\033[31m[BUILD ERROR]\033[0m Failed to write output file: %v\n", err)
		os.Exit(1)
	}

	outStat, statErr := os.Stat(absOutput)
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
	fmt.Printf("\033[32m[SUCCESS]\033[0m Successfully built standalone module %s%s\n", relOutput, sizeStr)
}
