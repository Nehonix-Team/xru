package main

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/Nehonix-Team/xru/internal/compiler"
	"github.com/Nehonix-Team/xru/internal/engine"
)

// runBuild compile un fichier .xru en scripts shell (.sh) et PowerShell (.ps1).
func runBuild(rulePath string) {
	rf, _ := engine.ParseFile(rulePath)
	shScript, _ := compiler.CompileToSH(rf)
	psScript, _ := compiler.CompileToPS1(rf)
	base := strings.TrimSuffix(rulePath, filepath.Ext(rulePath))
	os.WriteFile(base+".sh", []byte(shScript), 0755)
	os.WriteFile(base+".ps1", []byte(psScript), 0644)
}
