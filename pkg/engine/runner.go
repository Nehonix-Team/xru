package engine

import (
	"fmt"
	"path/filepath"
	"github.com/Nehonix-Team/xru/internal/engine/parser"
)

type Runner struct {
	Verbose      bool
	TerminalArgs []string
	CurrentFile  string
}

func NewRunner() *Runner {
	return &Runner{}
}

// Run applies an .xru rule file to a target directory.
func (r *Runner) Run(rulePath, targetDir string) error {
	r.CurrentFile = rulePath
	absTarget, err := filepath.Abs(targetDir)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for target: %v", err)
	}

	rf, err := parser.ParseFile(rulePath)
	if err != nil {
		return err
	}

	rootScope := NewRootScope(r)
	if r.Verbose {
		fmt.Printf("[DEBUG] Starting execution of %d top-level rules\n", len(rf.Rules))
	}

	if err := executeRules(rf.Rules, absTarget, absTarget, rulePath, rootScope, nil, "", r); err != nil {
		return err
	}

	if err := rootScope.CheckUnused(rulePath); err != nil {
		return err
	}

	return nil
}
