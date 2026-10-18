package checker

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Nehonix-Team/xru/pkg/ast"
	"github.com/Nehonix-Team/xru/pkg/lexer"
	"github.com/Nehonix-Team/xru/pkg/modules"
	"github.com/Nehonix-Team/xru/pkg/parser"
)

func (c *Checker) checkUse(s *ast.UseStatement) {
	if s.Module == "_mod" {
		c.addError(s.Pos(), "ERR_RESERVED_MOD_NAME", "'_mod' is a module mapping configuration file and cannot be imported via 'use'")
		return
	}
	alias := s.Alias
	if alias == "" {
		alias = s.Module
	}
	var sourceSym *Symbol
	for name, sym := range c.availableModules {
		if strings.EqualFold(name, s.Module) {
			sourceSym = sym
			break
		}
	}
	if s.CatchVar != "" {
		catchSym := &Symbol{
			Name: s.CatchVar,
			Kind: SymVar,
			Type: "any",
			Pos:  s.Pos(),
		}
		_ = c.scope.Define(catchSym)
		c.declaredVars = append(c.declaredVars, catchSym)
	}

	if sourceSym != nil {
		aliasSym := &Symbol{
			Name:       alias,
			Kind:       SymUnit,
			Type:       sourceSym.Name,
			Pos:        s.Pos(),
			Directives: sourceSym.Directives,
			Methods:    sourceSym.Methods,
		}
		if err := c.scope.Define(aliasSym); err != nil {
			c.addError(s.Pos(), "ERR_DUPLICATE_SYMBOL", err.Error())
		} else {
			c.importedSyms = append(c.importedSyms, aliasSym)
		}
	} else {
		if s.CatchVar == "" {
			c.addError(s.Pos(), "ERR_UNKNOWN_MODULE", fmt.Sprintf("Module '%s' does not exist", s.Module))
		} else {
			aliasSym := &Symbol{
				Name: alias,
				Kind: SymUnit,
				Pos:  s.Pos(),
			}
			_ = c.scope.Define(aliasSym)
			c.importedSyms = append(c.importedSyms, aliasSym)
		}
	}
}

func (c *Checker) checkMap(s *ast.MapStatement) {
	// Map enregistre des correspondances de modules
}

func (c *Checker) checkMod(s *ast.ModStatement) {
	c.hasModStmt = true
	if s.Name == "_mod" {
		c.addError(s.Pos(), "ERR_RESERVED_MOD_NAME", "Module name '_mod' is reserved for project mapping configuration")
	}
}

func (c *Checker) checkFromUse(s *ast.FromUseStatement) {
	fileDir := "."
	if c.currentFile != "" {
		fileDir = filepath.Dir(c.currentFile)
	}
	modFilePath := modules.FindModConfigFile(fileDir)
	var modCfg *modules.ModConfig
	if modFilePath == "" {
		if s.Source == "root" {
			// Implicit fallback: resolve root from project root
			wsRoot := findWorkspaceRootDir(fileDir)
			modCfg = &modules.ModConfig{
				Dir: wsRoot,
				Mappings: map[string]string{
					"root": "./",
				},
			}
		} else {
			c.addError(s.Pos(), "ERR_MISSING_MOD_FILE", "No '_mod.xru' configuration file found in workspace for module resolution")
			return
		}
	} else {
		cfg, modErrs := modules.LoadModConfig(modFilePath)
		if len(modErrs) > 0 {
			for _, errStr := range modErrs {
				c.addError(s.Pos(), "ERR_INVALID_MOD_CONFIG", errStr)
			}
			return
		}
		modCfg = cfg
	}

	if _, ok := modCfg.Mappings[s.Source]; !ok {
		c.addError(s.Pos(), "ERR_UNKNOWN_NAMESPACE", fmt.Sprintf("Module namespace '%s' is not mapped in _mod.xru", s.Source))
		return
	}

	// Cas 1 : multi-modules from "services" use (auth, cdn)
	if len(s.Modules) > 0 {
		for _, modName := range s.Modules {
			modDetails, err := modCfg.ResolveModule(s.Source, modName)
			if err != nil {
				c.addError(s.Pos(), "ERR_MODULE_NOT_FOUND", fmt.Sprintf("Module '%s' not found in mapped namespace '%s': %v", modName, s.Source, err))
				continue
			}
			if c.hasCircularImport(modDetails.FilePath, c.currentFile, modCfg) {
				c.addError(s.Pos(), "ERR_CIRCULAR_IMPORT", fmt.Sprintf("Circular module import detected between '%s' and '%s'", filepath.Base(c.currentFile), filepath.Base(modDetails.FilePath)))
				continue
			}
			methods := make(map[string]*Symbol)
			for expName, exp := range modDetails.Exports {
				symKind := SymVar
				if exp.Kind == "func" {
					symKind = SymFunc
				} else if exp.Kind == "const" {
					symKind = SymConst
				}
				methods[expName] = &Symbol{
					Name:   expName,
					Kind:   symKind,
					Type:   exp.Type,
					Params: exp.Params,
					Pos:    exp.Position,
				}
			}
			symPos := s.Pos()
			if p, ok := s.MemberPositions[modName]; ok {
				symPos = p
			}
			modSym := &Symbol{
				Name:    modName,
				Kind:    SymUnit,
				Pos:     symPos,
				Methods: methods,
			}
			if existing, ok := c.scope.Resolve(modName); ok && existing.Pos.Line == 0 {
				existing.Kind = SymUnit
				existing.Methods = methods
				existing.Pos = symPos
				c.importedSyms = append(c.importedSyms, existing)
				continue
			}
			if err := c.scope.Define(modSym); err != nil {
				c.addError(s.Pos(), "ERR_DUPLICATE_SYMBOL", err.Error())
			} else {
				c.importedSyms = append(c.importedSyms, modSym)
			}
		}
		return
	}

	// Cas 2 : membre individuel from "services" use auth::b
	if s.Member != "" {
		if s.CatchVar != "" {
			catchSym := &Symbol{
				Name: s.CatchVar,
				Kind: SymVar,
				Type: "any",
				Pos:  s.Pos(),
			}
			_ = c.scope.Define(catchSym)
			c.declaredVars = append(c.declaredVars, catchSym)
		}
		modDetails, err := modCfg.ResolveModule(s.Source, s.Module)
		if err != nil {
			if s.CatchVar == "" {
				c.addError(s.Pos(), "ERR_MODULE_NOT_FOUND", fmt.Sprintf("Module '%s' not found in mapped namespace '%s': %v", s.Module, s.Source, err))
			} else {
				targetName := s.Member
				if s.Alias != "" {
					targetName = s.Alias
				}
				memberSym := &Symbol{
					Name: targetName,
					Kind: SymVar,
					Type: "any",
					Pos:  s.Pos(),
				}
				_ = c.scope.Define(memberSym)
				c.importedSyms = append(c.importedSyms, memberSym)
			}
			return
		}
		if c.hasCircularImport(modDetails.FilePath, c.currentFile, modCfg) {
			c.addError(s.Pos(), "ERR_CIRCULAR_IMPORT", fmt.Sprintf("Circular module import detected between '%s' and '%s'", filepath.Base(c.currentFile), filepath.Base(modDetails.FilePath)))
			return
		}
		exp, ok := modDetails.Exports[s.Member]
		if !ok {
			if parentSym, hasParent := c.scope.Resolve(s.Member); hasParent {
				targetName := s.Member
				if s.Alias != "" {
					targetName = s.Alias
				}
				memberSym := &Symbol{
					Name:   targetName,
					Kind:   parentSym.Kind,
					Type:   parentSym.Type,
					Params: parentSym.Params,
					Pos:    s.Pos(),
				}
				_ = c.scope.Define(memberSym)
				c.importedSyms = append(c.importedSyms, memberSym)
				return
			}
			if s.Source == "root" || s.Module == "main" {
				if argExp := findExportedArg(modCfg, s.Member); argExp != nil {
					targetName := s.Member
					if s.Alias != "" {
						targetName = s.Alias
					}
					memberSym := &Symbol{
						Name: targetName,
						Kind: SymVar,
						Type: argExp.Type,
						Pos:  s.Pos(),
					}
					_ = c.scope.Define(memberSym)
					c.importedSyms = append(c.importedSyms, memberSym)
					return
				}
			}
			if s.CatchVar == "" {
				c.addError(s.Pos(), "ERR_MEMBER_NOT_FOUND", fmt.Sprintf("Member '%s' is not exported by module '%s'", s.Member, s.Module))
			} else {
				targetName := s.Member
				if s.Alias != "" {
					targetName = s.Alias
				}
				memberSym := &Symbol{
					Name: targetName,
					Kind: SymVar,
					Type: "any",
					Pos:  s.Pos(),
				}
				_ = c.scope.Define(memberSym)
				c.importedSyms = append(c.importedSyms, memberSym)
			}
			return
		}
		symKind := SymVar
		if exp.Kind == "func" {
			symKind = SymFunc
		} else if exp.Kind == "const" {
			symKind = SymConst
		}
		targetName := s.Member
		if s.Alias != "" {
			targetName = s.Alias
		}
		symPos := s.Pos()
		if p, ok := s.MemberPositions[s.Member]; ok {
			symPos = p
		}
		memberSym := &Symbol{
			Name:   targetName,
			Kind:   symKind,
			Type:   exp.Type,
			Params: exp.Params,
			Pos:    symPos,
		}
		if existing, ok := c.scope.Resolve(targetName); ok && existing.Pos.Line == 0 {
			existing.Kind = symKind
			existing.Type = exp.Type
			existing.Params = exp.Params
			existing.Pos = symPos
			c.importedSyms = append(c.importedSyms, existing)
			return
		}
		if err := c.scope.Define(memberSym); err != nil {
			c.addError(s.Pos(), "ERR_DUPLICATE_SYMBOL", err.Error())
		} else {
			c.importedSyms = append(c.importedSyms, memberSym)
		}
		return
	}

	// Cas 2b : membres groupés from "services" use auth::{b, c}
	if len(s.Members) > 0 {
		modDetails, err := modCfg.ResolveModule(s.Source, s.Module)
		if err != nil {
			c.addError(s.Pos(), "ERR_MODULE_NOT_FOUND", fmt.Sprintf("Module '%s' not found in mapped namespace '%s': %v", s.Module, s.Source, err))
			return
		}
		if c.hasCircularImport(modDetails.FilePath, c.currentFile, modCfg) {
			c.addError(s.Pos(), "ERR_CIRCULAR_IMPORT", fmt.Sprintf("Circular module import detected between '%s' and '%s'", filepath.Base(c.currentFile), filepath.Base(modDetails.FilePath)))
			return
		}
		for _, memName := range s.Members {
			exp, ok := modDetails.Exports[memName]
			if !ok {
				if parentSym, hasParent := c.scope.Resolve(memName); hasParent {
					exp = &modules.ExportedMember{
						Name: memName,
						Kind: "var",
						Type: parentSym.Type,
					}
					ok = true
				} else if s.Source == "root" || s.Module == "main" {
					if argExp := findExportedArg(modCfg, memName); argExp != nil {
						exp = argExp
						ok = true
					}
				}
			}
			if !ok {
				c.addError(s.Pos(), "ERR_MEMBER_NOT_FOUND", fmt.Sprintf("Member '%s' is not exported by module '%s'", memName, s.Module))
				continue
			}
			symKind := SymVar
			if exp.Kind == "func" {
				symKind = SymFunc
			} else if exp.Kind == "const" {
				symKind = SymConst
			}
			symPos := s.Pos()
			if p, ok := s.MemberPositions[memName]; ok {
				symPos = p
			}
			memberSym := &Symbol{
				Name:   memName,
				Kind:   symKind,
				Type:   exp.Type,
				Params: exp.Params,
				Pos:    symPos,
			}
			if existing, ok := c.scope.Resolve(memName); ok && existing.Pos.Line == 0 {
				existing.Kind = symKind
				existing.Type = exp.Type
				existing.Params = exp.Params
				existing.Pos = symPos
				c.importedSyms = append(c.importedSyms, existing)
				continue
			}
			if err := c.scope.Define(memberSym); err != nil {
				c.addError(s.Pos(), "ERR_DUPLICATE_SYMBOL", err.Error())
			} else {
				c.importedSyms = append(c.importedSyms, memberSym)
			}
		}
		return
	}

	// Cas 2c : module complet sous forme de Map (gArgs::@map as args)
	if s.AsMap && len(s.Members) == 0 {
		alias := s.Alias
		if alias == "" {
			alias = s.Module
		}
		modDetails, err := modCfg.ResolveModule(s.Source, s.Module)
		if err != nil {
			c.addError(s.Pos(), "ERR_MODULE_NOT_FOUND", fmt.Sprintf("Module '%s' not found in mapped namespace '%s': %v", s.Module, s.Source, err))
			return
		}
		if c.hasCircularImport(modDetails.FilePath, c.currentFile, modCfg) {
			c.addError(s.Pos(), "ERR_CIRCULAR_IMPORT", fmt.Sprintf("Circular module import detected between '%s' and '%s'", filepath.Base(c.currentFile), filepath.Base(modDetails.FilePath)))
			return
		}
		methods := make(map[string]*Symbol)
		for expName, exp := range modDetails.Exports {
			symKind := SymVar
			if exp.Kind == "func" {
				symKind = SymFunc
			} else if exp.Kind == "const" {
				symKind = SymConst
			}
			methods[expName] = &Symbol{
				Name:   expName,
				Kind:   symKind,
				Type:   exp.Type,
				Params: exp.Params,
				Pos:    exp.Position,
			}
		}
		mapSym := &Symbol{
			Name:    alias,
			Kind:    SymVar,
			Type:    "map",
			Pos:     s.Pos(),
			Methods: methods,
		}
		if existing, ok := c.scope.Resolve(alias); ok && existing.Pos.Line == 0 {
			existing.Kind = SymVar
			existing.Type = "map"
			existing.Methods = methods
			existing.Pos = s.Pos()
			c.importedSyms = append(c.importedSyms, existing)
			return
		}
		if err := c.scope.Define(mapSym); err != nil {
			c.addError(s.Pos(), "ERR_DUPLICATE_SYMBOL", err.Error())
		} else {
			c.importedSyms = append(c.importedSyms, mapSym)
		}
		return
	}

	// Cas 3 : module complet from "services" use auth as a (ou use auth)
	alias := s.Alias
	if alias == "" {
		alias = s.Module
	}
	modDetails, err := modCfg.ResolveModule(s.Source, s.Module)
	if err != nil {
		c.addError(s.Pos(), "ERR_MODULE_NOT_FOUND", fmt.Sprintf("Module '%s' not found in mapped namespace '%s': %v", s.Module, s.Source, err))
		return
	}
	if c.hasCircularImport(modDetails.FilePath, c.currentFile, modCfg) {
		c.addError(s.Pos(), "ERR_CIRCULAR_IMPORT", fmt.Sprintf("Circular module import detected between '%s' and '%s'", filepath.Base(c.currentFile), filepath.Base(modDetails.FilePath)))
		return
	}
	methods := make(map[string]*Symbol)
	for expName, exp := range modDetails.Exports {
		symKind := SymVar
		if exp.Kind == "func" {
			symKind = SymFunc
		} else if exp.Kind == "const" {
			symKind = SymConst
		}
		methods[expName] = &Symbol{
			Name:   expName,
			Kind:   symKind,
			Type:   exp.Type,
			Params: exp.Params,
			Pos:    exp.Position,
		}
	}
	modSym := &Symbol{
		Name:    alias,
		Kind:    SymUnit,
		Pos:     s.Pos(),
		Methods: methods,
	}
	if existing, ok := c.scope.Resolve(alias); ok {
		if existing.Kind == SymUnit {
			if existing.Methods == nil {
				existing.Methods = make(map[string]*Symbol)
			}
			for k, v := range methods {
				existing.Methods[k] = v
			}
			c.importedSyms = append(c.importedSyms, existing)
			return
		}
	}
	if err := c.scope.Define(modSym); err != nil {
		c.addError(s.Pos(), "ERR_DUPLICATE_SYMBOL", err.Error())
	} else {
		c.importedSyms = append(c.importedSyms, modSym)
	}
}

func (c *Checker) hasCircularImport(targetFilePath, currentFilePath string, modCfg *modules.ModConfig) bool {
	if targetFilePath == "" || currentFilePath == "" {
		return false
	}
	absTarget, err1 := filepath.Abs(targetFilePath)
	absCurrent, err2 := filepath.Abs(currentFilePath)
	if err1 != nil || err2 != nil {
		return false
	}
	if absTarget == absCurrent {
		return true
	}

	content, err := os.ReadFile(targetFilePath)
	if err != nil {
		return false
	}
	l := lexer.New(targetFilePath, string(content))
	p := parser.New(l)
	prog := p.ParseProgram()
	for _, stmt := range prog.Statements {
		fromStmt, ok := stmt.(*ast.FromUseStatement)
		if !ok {
			continue
		}
		var modsToCheck []string
		if len(fromStmt.Modules) > 0 {
			modsToCheck = append(modsToCheck, fromStmt.Modules...)
		} else if fromStmt.Module != "" {
			modsToCheck = append(modsToCheck, fromStmt.Module)
		} else if fromStmt.Alias != "" {
			modsToCheck = append(modsToCheck, fromStmt.Alias)
		}
		for _, m := range modsToCheck {
			subDetails, err := modCfg.ResolveModule(fromStmt.Source, m)
			if err == nil && subDetails != nil {
				subAbs, err := filepath.Abs(subDetails.FilePath)
				if err == nil && subAbs == absCurrent {
					return true
				}
			}
		}
	}
	return false
}

func findExportedArg(cfg *modules.ModConfig, argName string) *modules.ExportedMember {
	if cfg == nil {
		return nil
	}
	for ns := range cfg.Mappings {
		mods, err := cfg.GetNamespaceModules(ns)
		if err != nil {
			continue
		}
		for _, m := range mods {
			if exp, ok := m.Exports[argName]; ok && exp.Kind == "arg" {
				return exp
			}
		}
	}
	return nil
}

func findWorkspaceRootDir(startDir string) string {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return startDir
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "_mod.xru")); err == nil {
			return dir
		}
		if _, err := os.Stat(filepath.Join(dir, "main.xru")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return startDir
}
