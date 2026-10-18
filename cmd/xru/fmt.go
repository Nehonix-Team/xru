package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Nehonix-Team/xru/pkg/formatter"
)

func handleFmt(args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: xru fmt <file.xru ...>")
		os.Exit(1)
	}

	for _, arg := range args {
		if arg == "-w" || arg == "--write" {
			continue
		}
		path := arg
		data, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading %s: %v\n", path, err)
			continue
		}
		formatted, err := formatter.Format(string(data))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting %s: %v\n", path, err)
			continue
		}
		if err := os.WriteFile(path, []byte(formatted), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing %s: %v\n", path, err)
			continue
		}
		fmt.Printf("Formatted %s\n", filepath.Base(path))
	}
}
