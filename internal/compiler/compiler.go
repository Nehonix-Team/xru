package compiler

import (
	"github.com/Nehonix-Team/xru/internal/compiler/ps1"
	"github.com/Nehonix-Team/xru/internal/compiler/sh"
	"github.com/Nehonix-Team/xru/internal/engine/ast"
)

// CompileToSH orchestrates the compilation of XRU to a Unix Shell script.
func CompileToSH(rf *ast.RuleFile) (string, error) {
	return sh.Compile(rf)
}

// CompileToPS1 orchestrates the compilation of XRU to a Windows PowerShell script.
func CompileToPS1(rf *ast.RuleFile) (string, error) {
	return ps1.Compile(rf)
}
