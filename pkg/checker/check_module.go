package checker

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/Nehonix-Team/xru/pkg/ast"
	"github.com/Nehonix-Team/xru/pkg/modules"
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
	if sourceSym != nil {
		aliasSym := &Symbol{
			Name:       alias,
			Kind:       SymUnit,
			Type:       sourceSym.Type,
			Pos:        s.Pos(),
			Directives: sourceSym.Directives,
			Methods:    sourceSym.Methods,
		}
		if err := c.scope.Define(aliasSym); err != nil {
			c.addError(s.Pos(), "ERR_DUPLICATE_SYMBOL", err.Error())
		}
	} else {
		c.addError(s.Pos(), "ERR_UNKNOWN_MODULE", fmt.Sprintf("Module '%s' does not exist", s.Module))
	}
}

func (c *Checker) checkMap(s *ast.MapStatement) {
	// Map enregistre des correspondances de modules
}

func (c *Checker) checkMod(s *ast.ModStatement) {
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
	if modFilePath == "" {
		c.addError(s.Pos(), "ERR_MISSING_MOD_FILE", "No '_mod.xru' configuration file found in workspace for module resolution")
		return
	}

	modCfg, modErrs := modules.LoadModConfig(modFilePath)
	if len(modErrs) > 0 {
		for _, errStr := range modErrs {
			c.addError(s.Pos(), "ERR_INVALID_MOD_CONFIG", errStr)
		}
		return
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
				Name:    modName,
				Kind:    SymUnit,
				Pos:     s.Pos(),
				Methods: methods,
			}
			if err := c.scope.Define(modSym); err != nil {
				c.addError(s.Pos(), "ERR_DUPLICATE_SYMBOL", err.Error())
			}
		}
		return
	}

	// Cas 2 : membre individuel from "services" use auth::b
	if s.Member != "" {
		modDetails, err := modCfg.ResolveModule(s.Source, s.Module)
		if err != nil {
			c.addError(s.Pos(), "ERR_MODULE_NOT_FOUND", fmt.Sprintf("Module '%s' not found in mapped namespace '%s': %v", s.Module, s.Source, err))
			return
		}
		exp, ok := modDetails.Exports[s.Member]
		if !ok {
			c.addError(s.Pos(), "ERR_MEMBER_NOT_FOUND", fmt.Sprintf("Member '%s' is not exported by module '%s'", s.Member, s.Module))
			return
		}
		symKind := SymVar
		if exp.Kind == "func" {
			symKind = SymFunc
		} else if exp.Kind == "const" {
			symKind = SymConst
		}
		memberSym := &Symbol{
			Name:   s.Member,
			Kind:   symKind,
			Type:   exp.Type,
			Params: exp.Params,
			Pos:    s.Pos(),
		}
		if err := c.scope.Define(memberSym); err != nil {
			c.addError(s.Pos(), "ERR_DUPLICATE_SYMBOL", err.Error())
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
	if err := c.scope.Define(modSym); err != nil {
		c.addError(s.Pos(), "ERR_DUPLICATE_SYMBOL", err.Error())
	}
}
