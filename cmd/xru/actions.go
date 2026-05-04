package main

import (
	"strings"

	"github.com/Nehonix-Team/xru/internal/engine/ast"
	"github.com/Nehonix-Team/xru/internal/engine"
	"github.com/Nehonix-Team/xru/internal/engine/patcher"
	"github.com/Nehonix-Team/xru/internal/engine/util" 
)

// applyAction applique une action (inject, patch, var, module) au contenu d'un fichier.
func applyAction(content string, action ast.Action, fileExt string, scope *Scope, cb string, rulePath string) string {
	switch a := action.(type) {
	case ast.VarAction:
		val := util.Interpolate(a.Value, scope)
		scope.Set(a.Name, unescape(val), a.Line)
		return content

	case ast.ModuleAction:
		executeModuleAction(scope, cb, rulePath, a.Module, a.Method, a.Target, a.As, a.Line)
		return content

	case ast.InjectAction:
		if a.Lang != "" && "."+strings.ToLower(a.Lang) != fileExt {
			return content
		}
		code := util.Interpolate(a.Code, scope)
		checkSyntaxError(code, a.Line)
		return engine.InjectCode(content, a.Key, code)

	case ast.PatchAction:
		path := util.Interpolate(a.Path, scope)
		val := util.InterpolateValue(a.Value, scope).(ast.Value)
		return patcher.ApplyPatch(content, a.Op, path, val)
	}
	return content
}
